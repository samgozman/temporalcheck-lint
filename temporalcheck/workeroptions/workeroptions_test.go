package workeroptions_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/workeroptions"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths.

// TestWorkerOptions_Panic: every worker.Options literal that sets
// MaxConcurrentWorkflowTaskExecutionSize or MaxConcurrentWorkflowTaskPollers to a
// constant 1 is reported on the offending value, default-on.
func TestWorkerOptions_Panic(t *testing.T) {
	a := workeroptions.NewAnalyzer(workeroptions.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "workeroptionsfixtures/workerpanic")
}

// TestWorkerOptions_RequireOptions: with RequireOptions set, a worker.New whose
// worker.Options literal sets none of the five concurrency limits is flagged;
// setting any one, a variable argument, and a bare literal are not.
func TestWorkerOptions_RequireOptions(t *testing.T) {
	a := workeroptions.NewAnalyzer(workeroptions.Settings{RequireOptions: true})
	analysistest.Run(t, analysistest.TestData(), a, "workeroptionsfixtures/requireoptions")
}

// TestWorkerOptions_Benign: a workflow-task field set to 0 / >= 2 / a non-constant,
// the activity counterparts set to 1, an empty literal passed to New (require-options
// off), a positional literal, and non-worker literals produce no diagnostics.
func TestWorkerOptions_Benign(t *testing.T) {
	a := workeroptions.NewAnalyzer(workeroptions.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "workeroptionsfixtures/benign")
}

// TestWorkerOptions_Disabled: with Disabled set, the analyzer reports nothing even
// on a fixture whose literal would panic the worker.
func TestWorkerOptions_Disabled(t *testing.T) {
	a := workeroptions.NewAnalyzer(workeroptions.Settings{Disabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "workeroptionsfixtures/disabled")
}

// TestWorkerOptions_Nolint: a //nolint directive naming temporalcheck (or all, or
// bare) on the offending value's line suppresses its diagnostic; a directive
// naming only other linters, or the analyzer name, does not.
func TestWorkerOptions_Nolint(t *testing.T) {
	a := workeroptions.NewAnalyzer(workeroptions.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "workeroptionsfixtures/nolint")
}

// TestWorkerOptions_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestWorkerOptions_NolintFile(t *testing.T) {
	a := workeroptions.NewAnalyzer(workeroptions.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "workeroptionsfixtures/nolintfile")
}
