package workflowlogger_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/workflowlogger"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths.

// TestWorkflowLogger: every stdlib/zerolog logging call in workflow code is
// reported -- log/slog/fmt functions, *log.Logger and *slog.Logger methods,
// fmt.Fprint* to os.Stdout/os.Stderr, zerolog chains (once each), and calls inside
// a workflow.Go closure, a method workflow and a nested workflow literal.
func TestWorkflowLogger(t *testing.T) {
	a := workflowlogger.NewAnalyzer(workflowlogger.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "workflowloggerfixtures/flagged")
}

// TestWorkflowLogger_Benign: logging from a non-workflow function or an activity,
// the SDK's workflow.GetLogger, fmt.Sprintf/Errorf and fmt.Fprintf to a buffer
// produce no diagnostics.
func TestWorkflowLogger_Benign(t *testing.T) {
	a := workflowlogger.NewAnalyzer(workflowlogger.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "workflowloggerfixtures/benign")
}

// TestWorkflowLogger_Disabled: with Enabled unset (the default), the analyzer
// reports nothing even on a fixture that logs from workflow code.
func TestWorkflowLogger_Disabled(t *testing.T) {
	a := workflowlogger.NewAnalyzer(workflowlogger.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "workflowloggerfixtures/disabled")
}

// TestWorkflowLogger_Nolint: a //nolint directive naming temporalcheck (or all, or
// bare) on the call's line suppresses its diagnostic; a directive naming only
// other linters, or the analyzer name, does not.
func TestWorkflowLogger_Nolint(t *testing.T) {
	a := workflowlogger.NewAnalyzer(workflowlogger.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "workflowloggerfixtures/nolint")
}

// TestWorkflowLogger_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestWorkflowLogger_NolintFile(t *testing.T) {
	a := workflowlogger.NewAnalyzer(workflowlogger.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "workflowloggerfixtures/nolintfile")
}
