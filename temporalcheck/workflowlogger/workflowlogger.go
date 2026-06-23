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

	"golang.org/x/tools/go/analysis"
)

const (
	workflowPkg = "go.temporal.io/sdk/workflow"
	// workflowInternalPkg is where the SDK actually declares Context; the public
	// workflow.Context is an alias to it, so a parameter's type may surface in
	// either package depending on gotypesalias mode.
	workflowInternalPkg = "go.temporal.io/sdk/internal"
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
		nolint := collectNolint(pass.Fset, file)
		// Walk the file looking for workflow definitions. The first one found on a
		// given path owns its whole subtree (including nested closures), so once we
		// report on its body we stop descending -- a nested workflow closure's
		// logging is already covered, and re-entering it would double-report.
		ast.Inspect(file, func(n ast.Node) bool {
			body, ft := funcBody(n)
			if body == nil {
				return true
			}
			if isWorkflowFunc(pass, ft) {
				c.reportLogging(pass, nolint, body)
				return false
			}
			return true
		})
	}
	return nil, nil
}

// funcBody returns the body and type of a function declaration or literal, or
// (nil, nil) for any other node.
func funcBody(n ast.Node) (*ast.BlockStmt, *ast.FuncType) {
	switch fn := n.(type) {
	case *ast.FuncDecl:
		return fn.Body, fn.Type
	case *ast.FuncLit:
		return fn.Body, fn.Type
	default:
		return nil, nil
	}
}
