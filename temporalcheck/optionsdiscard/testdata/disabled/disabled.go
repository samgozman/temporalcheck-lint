// Package disabled holds a discarded With*Options result with no expectation
// marker: run with Settings.Disabled true, the analyzer must stay silent, so
// analysistest sees no diagnostics here.
package disabled

import "go.temporal.io/sdk/workflow"

func Workflow(ctx workflow.Context) error {
	var ao workflow.ActivityOptions

	// Would be flagged if the analyzer were enabled.
	workflow.WithActivityOptions(ctx, ao)

	return nil
}
