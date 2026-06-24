// Package nonserializable flags chan/func types (and, opt-in, structs with no
// exported fields) as Temporal activity/workflow parameter or return types where
// Temporal's DataConverter cannot serialize them. Only the top-level
// parameter/return type is checked to avoid false positives.
package nonserializable

import (
	"go/ast"
	"go/types"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"golang.org/x/tools/go/analysis"
)

// Settings configures the nonserializable analyzer.
type Settings struct {
	Disabled    bool
	EmptyStruct bool // also flag structs with no exported fields (opt-in; json.Marshaler exclusion makes this less clear-cut)
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

// checker threads the analyzer settings through the AST walk.
type checker struct {
	disabled    bool
	emptyStruct bool
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
