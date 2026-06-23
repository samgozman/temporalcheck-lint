package workflowstate_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/workflowstate"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths.

// TestWorkflowState: every mutation rooted at a package-level variable is
// reported -- scalar assignment, ++/--, compound assignment, a field of a global
// struct, an index into a global map/slice, reassignment via append, a
// cross-package qualified selector, and mutations inside a workflow.Go closure,
// an Await callback and a method workflow.
func TestWorkflowState(t *testing.T) {
	a := workflowstate.NewAnalyzer(workflowstate.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "workflowstatefixtures/flagged")
}

// TestWorkflowState_Benign: the idiomatic non-flagged patterns -- a captured
// local mutated from a workflow.Go/Await/Selector callback, a mutated parameter
// or local, a read of a global, a `:=` that declares a new local, and a global
// mutated from a non-workflow function -- produce no diagnostics.
func TestWorkflowState_Benign(t *testing.T) {
	a := workflowstate.NewAnalyzer(workflowstate.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "workflowstatefixtures/benign")
}

// TestWorkflowState_Disabled: with Disabled set, the analyzer reports nothing
// even on a fixture that mutates a global from workflow code.
func TestWorkflowState_Disabled(t *testing.T) {
	a := workflowstate.NewAnalyzer(workflowstate.Settings{Disabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "workflowstatefixtures/disabled")
}

// TestWorkflowState_Nolint: a //nolint directive naming temporalcheck (or all,
// or bare) on the mutation's line suppresses its diagnostic; a directive naming
// only other linters, or the analyzer name, does not.
func TestWorkflowState_Nolint(t *testing.T) {
	a := workflowstate.NewAnalyzer(workflowstate.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "workflowstatefixtures/nolint")
}

// TestWorkflowState_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestWorkflowState_NolintFile(t *testing.T) {
	a := workflowstate.NewAnalyzer(workflowstate.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "workflowstatefixtures/nolintfile")
}
