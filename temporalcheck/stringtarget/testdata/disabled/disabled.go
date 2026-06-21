// Package disabled holds string-named targets with no expectation marker: with
// Enabled false, the analyzer must stay silent. Enabled is the master switch, so
// it stays silent even with StrictTests on -- the mock setup below would be
// flagged under StrictTests if Enabled gated nothing, so its silence proves the
// gate.
package disabled

import (
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func Workflow(ctx workflow.Context) error {
	// Would be flagged if the production check were enabled.
	_ = workflow.ExecuteActivity(ctx, "Greet", "world")

	return nil
}

func setup(env *testsuite.TestWorkflowEnvironment) {
	// Would be flagged by StrictTests, but Enabled (the master switch) is off.
	env.OnActivity("Greet", nil, nil)
}
