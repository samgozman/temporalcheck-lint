// Package futureget flags a Future/ChildWorkflowFuture/EncodedValue .Get call
// whose returned error is discarded, silently ignoring a failed activity,
// child workflow, or decode error.
package futureget

import (
	"go/ast"
	"go/token"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"golang.org/x/tools/go/analysis"
)

const tagFutureGet = "future-get"

// Settings configures the futureget analyzer.
type Settings struct {
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
				// _ = future.Get(...): single blank assignment discards the error.
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
