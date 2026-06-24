// Package continueasnew flags a workflow.NewContinueAsNewError result that is
// discarded (bare statement or _ =) rather than returned, so the workflow ends
// instead of continuing as new.
package continueasnew

import (
	"go/ast"
	"go/token"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"golang.org/x/tools/go/analysis"
)

const tagContinueAsNew = "continue-as-new"

// Settings configures the continueasnew analyzer.
type Settings struct {
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
				// _ = workflow.NewContinueAsNewError(...): single blank assignment discards the result.
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
