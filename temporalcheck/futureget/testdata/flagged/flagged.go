// Package flagged exercises the discarded-error shapes the analyzer reports: a
// bare .Get statement and an explicit blank assignment, across all three
// receiver types (workflow.Future, workflow.ChildWorkflowFuture,
// converter.EncodedValue), including a chained receiver and an aliased import of
// the workflow package.
package flagged

import (
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"

	wf "go.temporal.io/sdk/workflow"
)

func Workflow(ctx workflow.Context, ev converter.EncodedValue) error {
	var result string

	// Bare statement on a workflow.Future: the error is thrown away entirely.
	f := workflow.ExecuteActivity(ctx, "Activity")
	f.Get(ctx, &result) // want `Get: the returned error from Future\.Get is discarded; check it or assign it to a variable you inspect \(future-get\)`

	// Explicit blank assignment is also a discard -- the classic
	// `_ = future.Get(ctx, nil)` that drops the activity error.
	_ = f.Get(ctx, nil) // want `Get: the returned error from Future\.Get is discarded; check it or assign it to a variable you inspect \(future-get\)`

	// workflow.ChildWorkflowFuture: Get is promoted from the embedded Future.
	cf := workflow.ExecuteChildWorkflow(ctx, "Child")
	cf.Get(ctx, &result) // want `Get: the returned error from ChildWorkflowFuture\.Get is discarded; check it or assign it to a variable you inspect \(future-get\)`

	// Chained receiver: GetChildWorkflowExecution() returns a Future, whose
	// discarded .Get is flagged just like a named one.
	cf.GetChildWorkflowExecution().Get(ctx, &result) // want `Get: the returned error from Future\.Get is discarded; check it or assign it to a variable you inspect \(future-get\)`

	// converter.EncodedValue: its Get takes no context but returns the same
	// must-check error.
	ev.Get(&result)     // want `Get: the returned error from EncodedValue\.Get is discarded; check it or assign it to a variable you inspect \(future-get\)`
	_ = ev.Get(&result) // want `Get: the returned error from EncodedValue\.Get is discarded; check it or assign it to a variable you inspect \(future-get\)`

	// An aliased import of the workflow package resolves to the same internal
	// Future type, so it is flagged the same.
	af := wf.ExecuteActivity(ctx, "Activity")
	af.Get(ctx, &result) // want `Get: the returned error from Future\.Get is discarded; check it or assign it to a variable you inspect \(future-get\)`

	return nil
}
