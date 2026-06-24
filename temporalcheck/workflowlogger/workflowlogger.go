// Package workflowlogger flags stdlib (log, log/slog, fmt.Print*) and zerolog
// logging calls from Temporal workflow code. Workflows replay, so non-replay-aware
// loggers double-log on every replay. Use workflow.GetLogger(ctx) instead. The
// check is opt-in (off by default).
package workflowlogger

import (
	"go/ast"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/workflowscope"
	"golang.org/x/tools/go/analysis"
)

const (
	// tagWorkflowLogger suffixes the diagnostic so it is clear which check
	// produced it.
	tagWorkflowLogger = "workflow-logger"
)

// Settings configures the workflowlogger analyzer.
type Settings struct {
	Enabled bool // master switch (default false)
}

// NewAnalyzer builds the workflowlogger analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{enabled: settings.Enabled}
	return &analysis.Analyzer{
		Name: "workflowlogger",
		Doc:  "flag standard-library (log, log/slog, fmt.Print*) and zerolog logging calls from Temporal workflow code, which double-log on replay and are not replay-aware; use workflow.GetLogger(ctx)",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk.
type checker struct {
	enabled bool
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	if !c.enabled {
		return nil, nil
	}
	for _, file := range pass.Files {
		nolint := nolint.Collect(pass.Fset, file)
		workflowscope.Walk(pass, file, func(body *ast.BlockStmt) {
			c.reportLogging(pass, nolint, body)
		})
	}
	return nil, nil
}
