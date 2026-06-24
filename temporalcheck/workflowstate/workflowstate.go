// Package workflowstate flags mutation of package-level variables from Temporal
// workflow code. Such mutations break replay determinism and race across concurrent
// workflow executions in the same worker. Captured locals are not flagged: mutating
// a captured local via workflow.Go/Selector callbacks is the SDK's own idiom.
package workflowstate

import (
	"go/ast"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/workflowscope"
	"golang.org/x/tools/go/analysis"
)

const (
	// tagGlobalMutation suffixes the diagnostic so it is clear which check
	// produced it.
	tagGlobalMutation = "global-mutation"
)

// Settings configures the workflowstate analyzer.
type Settings struct {
	Disabled bool
}

// NewAnalyzer builds the workflowstate analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{disabled: settings.Disabled}
	return &analysis.Analyzer{
		Name: "workflowstate",
		Doc:  "flag mutation of a package-level variable from Temporal workflow code, which breaks replay determinism and races across concurrent workflow executions",
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
		workflowscope.Walk(pass, file, func(body *ast.BlockStmt) {
			c.reportMutations(pass, nolint, body)
		})
	}
	return nil, nil
}
