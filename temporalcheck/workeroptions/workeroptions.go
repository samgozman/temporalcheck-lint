// Package workeroptions flags worker.Options literals that set
// MaxConcurrentWorkflowTaskExecutionSize or MaxConcurrentWorkflowTaskPollers to 1
// (a guaranteed worker-boot panic), and optionally worker.New calls whose
// worker.Options sets no concurrency limits.
package workeroptions

import (
	"go/ast"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"golang.org/x/tools/go/analysis"
)

// Settings configures the workeroptions analyzer.
type Settings struct {
	Disabled       bool
	RequireOptions bool // also flag worker.New with no concurrency limits set (opt-in)
}

// NewAnalyzer builds the workeroptions analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{
		disabled:       settings.Disabled,
		requireOptions: settings.RequireOptions,
	}
	return &analysis.Analyzer{
		Name: "workeroptions",
		Doc:  "flag Temporal worker.Options literals that set MaxConcurrentWorkflowTask{ExecutionSize,Pollers} to 1 (a worker-boot panic), and optionally worker.New calls whose worker.Options sets no concurrency limits",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk.
type checker struct {
	disabled       bool
	requireOptions bool
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	if c.disabled {
		return nil, nil
	}
	for _, file := range pass.Files {
		nolint := nolint.Collect(pass.Fset, file)
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.CompositeLit:
				// worker-panic: any worker.Options literal, anywhere.
				c.checkPanic(pass, nolint, node)
			case *ast.CallExpr:
				// require-options: the worker.Options literal passed to worker.New.
				if c.requireOptions {
					c.checkRequireOptions(pass, nolint, node)
				}
			}
			return true
		})
	}
	return nil, nil
}
