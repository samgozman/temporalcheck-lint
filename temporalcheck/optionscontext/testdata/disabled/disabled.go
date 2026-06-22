// Package disabled proves that with Disabled set, the analyzer reports nothing
// even on a context with a clear options/call-kind contradiction.
package disabled

import "go.temporal.io/sdk/workflow"

func greet(ctx workflow.Context) error { return nil }

func Workflow(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	workflow.ExecuteActivity(ctx, greet)
	return nil
}
