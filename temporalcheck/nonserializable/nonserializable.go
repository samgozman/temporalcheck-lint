// Package nonserializable implements a static check for the Temporal Go SDK.
//
// Temporal serializes activity and workflow arguments and results through its
// DataConverter, whose default is JSON. Some Go types can never round-trip
// through it at all:
//
//   - A chan or func value has no JSON representation -- encoding/json returns an
//     "unsupported type" error for both -- so a parameter or result of that type
//     can never be encoded. This is impossible to get right, never a false
//     positive, and is checked by default.
//   - A struct with fields but no *exported* fields encodes to "{}": JSON only
//     marshals exported fields, so all of its data is silently dropped on the
//     wire. The exception is a type that implements json.Marshaler and so controls
//     its own encoding (json.RawMessage, time.Time, and the like). Because of that
//     exclusion this case is less clear-cut, so it is opt-in.
//
// This is the same type-predicate shape as the sibling lossynumber analyzer
// (which flags types that decode *lossily*); nonserializable flags types that
// can't encode at all. Like lossynumber it looks only at the top level -- a
// struct that merely contains a chan field is not flagged -- to stay
// false-positive-free.
package nonserializable

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// Settings configures the nonserializable analyzer.
type Settings struct {
	// Disabled turns the analyzer off entirely; it reports nothing. The chan/func
	// check is on by default: those types can never be serialized, which is always
	// a bug, so there is nothing to opt into.
	Disabled bool

	// EmptyStruct opts into also flagging a struct that has fields but no exported
	// ones (and does not implement json.Marshaler), which JSON encodes to "{}",
	// silently dropping its data. Off by default: the json.Marshaler exclusion
	// makes this less clear-cut than the always-on chan/func check.
	EmptyStruct bool
}

// NewAnalyzer builds the nonserializable analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{disabled: settings.Disabled, emptyStruct: settings.EmptyStruct}
	return &analysis.Analyzer{
		Name: "nonserializable",
		Doc:  "flag chan and func types (and, opt-in, structs with no exported fields) as Temporal activity/workflow parameter or return types, where Temporal's DataConverter cannot serialize them",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk so the analyzer
// stays free of package-level mutable state.
type checker struct {
	disabled    bool
	emptyStruct bool
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	if c.disabled {
		return nil, nil
	}
	for _, file := range pass.Files {
		nolint := collectNolint(pass.Fset, file)
		ast.Inspect(file, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				c.checkCall(pass, nolint, call)
			}
			return true
		})
	}
	return nil, nil
}

func (c *checker) checkCall(pass *analysis.Pass, nolint nolintInfo, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Resolve via Uses (not the source text), so aliased imports of the
	// workflow/client packages still match.
	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return
	}

	e, ok := entryFor(fn)
	if !ok {
		return
	}
	c.checkTarget(pass, nolint, call, e)
}
