// Package disabled holds a string-named target with no expectation marker: run
// with Settings.Enabled false (the default), the analyzer must stay silent, so
// analysistest sees no diagnostics here.
package disabled

import "go.temporal.io/sdk/workflow"

func Workflow(ctx workflow.Context) error {
	// Would be flagged if the check were enabled.
	_ = workflow.ExecuteActivity(ctx, "Greet", "world")

	return nil
}
