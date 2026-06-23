// Package lossynumber implements a static check for the Temporal Go SDK.
//
// Temporal serializes activity and workflow arguments and results through its
// DataConverter, whose default is JSON. encoding/json decodes every JSON number
// into a float64 whenever the destination Go type is interface{}, and a float64
// cannot represent integers past 2^53 exactly. So an int64 carried through a
// dynamically-typed parameter or return -- interface{}/any, map[string]any,
// []any -- round-trips as a float64 and silently loses precision:
//
//	var n int64 = 9007199254740993 // 2^53 + 1
//	// marshaled and decoded into an `any` parameter: becomes 9007199254740992.
//
// This analyzer resolves the function referenced by each Execute* call to its
// real signature and flags any top-level parameter or (non-error) return whose
// type is one of those lossy dynamic types. It deliberately looks only at the
// top level -- a struct that merely contains an `any` field is not flagged --
// to stay false-positive-free. The fix is to use a concrete type.
package lossynumber

import (
	"go/ast"
	"go/types"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"golang.org/x/tools/go/analysis"
)

// Settings configures the lossynumber analyzer.
type Settings struct {
	// Disabled turns the analyzer off entirely; it reports nothing. The check is
	// on by default: a dynamically-typed number silently corrupts past 2^53, which
	// is a latent data-loss bug, so there is nothing to opt into.
	Disabled bool
}

// NewAnalyzer builds the lossynumber analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{disabled: settings.Disabled}
	return &analysis.Analyzer{
		Name: "lossynumber",
		Doc:  "flag interface{}/any, map[string]any and []any as Temporal activity/workflow parameter or return types, where numbers decode as float64 and silently lose precision past 2^53",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk so the analyzer
// stays free of package-level mutable state.
type checker struct {
	disabled bool
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	if c.disabled {
		return nil, nil
	}
	for _, file := range pass.Files {
		nolint := nolint.Collect(pass.Fset, file)
		ast.Inspect(file, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				c.checkCall(pass, nolint, call)
			}
			return true
		})
	}
	return nil, nil
}

func (c *checker) checkCall(pass *analysis.Pass, nolint nolint.Info, call *ast.CallExpr) {
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
