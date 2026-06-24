// Package optionsdiscard flags workflow.WithActivityOptions / WithLocalActivityOptions
// / WithChildOptions calls whose returned context is discarded (bare statement or
// _ =). Each With* function returns a new context; forgetting `ctx =` means the
// options silently never apply.
package optionsdiscard

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
	"golang.org/x/tools/go/analysis"
)

const (
	// tagOptionsDiscard suffixes the diagnostic so it is clear which check
	// produced it.
	tagOptionsDiscard = "options-discard"
)

// entryPoints are the workflow.* functions whose returned context must be kept.
// Each returns a derived context carrying the options; discarding it is the bug.
// Supporting another such function is a single row.
var entryPoints = map[string]bool{
	"WithActivityOptions":      true,
	"WithLocalActivityOptions": true,
	// WithChildOptions is the public name; the SDK declares the underlying setter
	// as internal.WithChildWorkflowOptions but exports it from workflow as this.
	"WithChildOptions": true,
}

// Settings configures the optionsdiscard analyzer.
type Settings struct {
	Disabled bool
}

// NewAnalyzer builds the optionsdiscard analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{disabled: settings.Disabled}
	return &analysis.Analyzer{
		Name: "optionsdiscard",
		Doc:  "flag Temporal workflow.WithActivityOptions/WithLocalActivityOptions/WithChildOptions calls whose returned context is discarded, so the options silently never apply",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk.
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
			switch s := n.(type) {
			case *ast.ExprStmt:
				if call, ok := s.X.(*ast.CallExpr); ok {
					c.checkDiscarded(pass, nolint, call)
				}
			case *ast.AssignStmt:
				if s.Tok == token.ASSIGN && len(s.Lhs) == 1 && len(s.Rhs) == 1 && isBlank(s.Lhs[0]) {
					if call, ok := s.Rhs[0].(*ast.CallExpr); ok {
						c.checkDiscarded(pass, nolint, call)
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

// checkDiscarded reports call when it is a With*Options entry point whose result
// is being thrown away, after honoring //nolint.
func (c *checker) checkDiscarded(pass *analysis.Pass, nolint nolint.Info, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Resolve via Uses so aliased imports still match.
	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return
	}
	if fn.Pkg().Path() != temporalsdk.WorkflowPkg || !entryPoints[fn.Name()] {
		return
	}

	// Honor //nolint after confirming this is a call we flag.
	if nolint.Suppresses(pass.Fset, call) {
		return
	}

	pass.Reportf(call.Pos(),
		"%s: the returned context is discarded, so the options never apply; assign it back with ctx = workflow.%s(ctx, opts) (%s)",
		fn.Name(), fn.Name(), tagOptionsDiscard)
}

// isBlank reports whether e is the blank identifier `_`.
func isBlank(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "_"
}
