// Package disabled holds a discarded .Get error with no expectation marker: run
// with Settings.Disabled true, the analyzer must stay silent, so analysistest
// sees no diagnostics here.
package disabled

import "go.temporal.io/sdk/workflow"

func Workflow(ctx workflow.Context) error {
	var result string

	// Would be flagged if the analyzer were enabled.
	f := workflow.ExecuteActivity(ctx, "Activity")
	f.Get(ctx, &result)

	return nil
}
