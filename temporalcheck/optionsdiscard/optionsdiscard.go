// Package optionsdiscard implements a static check for the Temporal Go SDK.
//
// workflow.WithActivityOptions / WithLocalActivityOptions / WithChildOptions
// do not mutate the context they receive; each returns a *new* context that
// carries the options:
//
//	func WithActivityOptions(ctx Context, options ActivityOptions) Context
//
// The classic mistake is to forget the `ctx =`:
//
//	workflow.WithActivityOptions(ctx, ao) // result thrown away
//	workflow.ExecuteActivity(ctx, a.Greet) // runs with the OLD ctx -- no options
//
// The options silently never apply, and the activity blows up at run time with a
// missing-timeout error. This analyzer flags any such call whose returned context
// is discarded -- used as a bare expression statement, or assigned to the blank
// identifier -- so the bug is caught at lint time instead. It is errcheck-style:
// pure AST + types, and near-zero false positives, so it is on by default.
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
	// Disabled turns the analyzer off entirely; it reports nothing. The check is
	// on by default: discarding a With*Options result is always a bug, never a
	// legitimate pattern, so there is nothing to opt into.
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
			switch s := n.(type) {
			case *ast.ExprStmt:
				// A bare call statement throws the result away entirely -- the
				// classic "forgot the ctx =" shape.
				if call, ok := s.X.(*ast.CallExpr); ok {
					c.checkDiscarded(pass, nolint, call)
				}
			case *ast.AssignStmt:
				// `_ = workflow.With...Options(...)` discards the result explicitly.
				// With*Options is single-valued, so the discard is exactly this
				// one-to-one blank assignment; anything else keeps the context.
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

	// Resolve via Uses (not the source text), so aliased imports of the workflow
	// package still match.
	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return
	}
	if fn.Pkg().Path() != temporalsdk.WorkflowPkg || !entryPoints[fn.Name()] {
		return
	}

	// Honor //nolint directives ourselves so suppression works the same way in
	// standalone/analysistest runs, not only under golangci-lint. Checked after
	// confirming this is a call we flag, so unrelated calls cost nothing.
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
