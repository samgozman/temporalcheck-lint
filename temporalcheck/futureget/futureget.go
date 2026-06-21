// Package futureget implements a static check for the Temporal Go SDK.
//
// A workflow.Future / workflow.ChildWorkflowFuture / converter.EncodedValue
// surfaces an activity, child-workflow or decoded-value result through:
//
//	func (Future) Get(ctx Context, valuePtr interface{}) error
//
// The returned error reports a failed activity, a failed child workflow, or a
// decode error. Dropping it -- as a bare call statement or a `_ =` assignment --
// silently ignores that failure:
//
//	_ = future.Get(ctx, nil) // activity error swallowed
//
// This is errcheck scoped to Temporal's result types. By construction it cannot
// fire on fire-and-forget (that path never calls .Get), and it is pure AST +
// types with near-zero false positives, so it is on by default.
package futureget

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

// tagFutureGet suffixes the diagnostic so it is clear which check produced it.
const tagFutureGet = "future-get"

// Settings configures the futureget analyzer.
type Settings struct {
	// Disabled turns the analyzer off entirely; it reports nothing. The check is
	// on by default: discarding a .Get error swallows an activity/child-workflow
	// failure, which is always a bug, so there is nothing to opt into.
	Disabled bool
}

// NewAnalyzer builds the futureget analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{disabled: settings.Disabled}
	return &analysis.Analyzer{
		Name: "futureget",
		Doc:  "flag a Temporal Future/ChildWorkflowFuture/EncodedValue .Get call whose returned error is discarded, so an activity, child-workflow or decode failure is silently ignored",
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
				// classic dropped `future.Get(ctx, &x)`.
				if call, ok := s.X.(*ast.CallExpr); ok {
					c.checkDiscarded(pass, nolint, call)
				}
			case *ast.AssignStmt:
				// `_ = future.Get(...)` discards the error explicitly. Get is
				// single-valued, so the discard is exactly this one-to-one blank
				// assignment; anything else keeps the error.
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
