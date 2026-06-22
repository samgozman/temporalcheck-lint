// Package disabled holds discarded continue-as-new errors with no expectation
// markers: run with Settings.Disabled true, the analyzer must stay silent, so
// analysistest sees no diagnostics here.
package disabled

import "go.temporal.io/sdk/workflow"

func Workflow(ctx workflow.Context) error {
	// Would be flagged if the analyzer were enabled.
	workflow.NewContinueAsNewError(ctx, Workflow)
	_ = workflow.NewContinueAsNewError(ctx, Workflow)
	return nil
}
