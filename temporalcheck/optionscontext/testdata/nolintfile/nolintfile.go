//nolint:temporalcheck // whole-file suppression: directive before the package clause

// Package nolintfile proves a //nolint directive before the package clause
// suppresses every diagnostic in the file.
package nolintfile

import "go.temporal.io/sdk/workflow"

func greet(ctx workflow.Context) error { return nil }

func Workflow(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	workflow.ExecuteActivity(ctx, greet)
	return nil
}
