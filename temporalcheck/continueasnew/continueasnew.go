// Package continueasnew implements a static check for the Temporal Go SDK.
//
// A workflow continues as new by *returning* the error built by:
//
//	func NewContinueAsNewError(ctx Context, wfn interface{}, args ...interface{}) error
//
// The returned error is the signal: returning it ends the current run and
// starts a fresh one. Constructing it without returning it -- as a bare call
// statement or a `_ =` assignment -- silently drops that signal, so the workflow
// just falls through and *ends* instead of continuing:
//
//	workflow.NewContinueAsNewError(ctx, MyWorkflow) // built, never returned
//	return nil                                      // workflow ends here
//
// Only the unambiguous discards are flagged (a bare statement and an explicit
// blank assignment). A value assigned to a named variable is left alone: a
// `return err` may follow, and proving it does not would need flow analysis.
// That keeps the check pure AST + types with near-zero false positives, so it is
// on by default.
package continueasnew

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

// tagContinueAsNew suffixes the diagnostic so it is clear which check produced it.
const tagContinueAsNew = "continue-as-new"

// Settings configures the continueasnew analyzer.
type Settings struct {
	// Disabled turns the analyzer off entirely; it reports nothing. The check is
	// on by default: discarding a NewContinueAsNewError result silently aborts the
	// continue-as-new and ends the workflow instead, which is always a bug, so
	// there is nothing to opt into.
	Disabled bool
}

// NewAnalyzer builds the continueasnew analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{disabled: settings.Disabled}
	return &analysis.Analyzer{
		Name: "continueasnew",
		Doc:  "flag a workflow.NewContinueAsNewError result that is discarded rather than returned, so the workflow silently ends instead of continuing as new",
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
		nolint := collectNolint(pass.Fset, file)
		ast.Inspect(file, func(n ast.Node) bool {
			switch s := n.(type) {
			case *ast.ExprStmt:
				// A bare call statement throws the result away entirely -- the
				// classic NewContinueAsNewError(ctx, WF) written where a
				// `return NewContinueAsNewError(...)` was meant.
				if call, ok := s.X.(*ast.CallExpr); ok {
					c.checkDiscarded(pass, nolint, call)
				}
			case *ast.AssignStmt:
				// `_ = workflow.NewContinueAsNewError(...)` discards the error
				// explicitly. The call is single-valued, so the discard is exactly
				// this one-to-one blank assignment; a named target may still be
				// returned later, so anything else is left alone.
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

// isBlank reports whether e is the blank identifier `_`.
func isBlank(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "_"
}
