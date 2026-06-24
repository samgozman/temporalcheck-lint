// Package workflowlogger implements a static check for the Temporal Go SDK.
//
// Temporal workflows replay: a worker re-executes workflow code against recorded
// history to rebuild state, and that re-execution runs the same statements again.
// Logging from workflow code with the standard library (log, log/slog, fmt.Print*)
// or a third-party logger (zerolog) therefore emits the same line on every replay,
// and those loggers are not replay-aware -- they have no notion of "this is a
// replay, suppress it". The SDK's own workflow.GetLogger(ctx) is: it is wired into
// the replay machinery and skips duplicate output during replay. The Temporal docs
// are explicit that workflow code should log only through it.
//
// This analyzer flags direct stdlib/zerolog logging calls inside any workflow
// definition (a function whose first parameter is workflow.Context) -- including
// the closures lexically nested in it, such as workflow.Go goroutines and Selector
// callbacks -- and points at workflow.GetLogger(ctx) instead. Activities (whose
// first parameter is the standard context.Context) are deliberately untouched:
// they do not replay, so logging there is not a determinism hazard.
//
// It is opt-in (Enabled, default off). Configuring an SDK logger or routing output
// through other means is a legitimate choice for some teams, so the analyzer stays
// silent until a project asks for it -- like the stringtarget and sensitiveargs
// checks.
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
	// Enabled is the master switch (default false). Logging through a stdlib or
	// third-party logger from workflow code double-logs on replay, but some teams
	// deliberately wire their own logging, so the check is opt-in; with Enabled off
	// the analyzer reports nothing.
	Enabled bool
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

// checker threads the analyzer settings through the AST walk so the analyzer
// stays free of package-level mutable state.
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
