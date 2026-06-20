package good

import "go.temporal.io/sdk/workflow"

// ShipmentWorkflow is a child workflow: the injected workflow.Context plus one
// real argument.
func ShipmentWorkflow(ctx workflow.Context, orderID string) error {
	return nil
}

func Workflow(ctx workflow.Context) error {
	var a *Activities

	// Correct arity and types for each entry point.
	_ = workflow.ExecuteActivity(ctx, a.Greet, "world")
	_ = workflow.ExecuteActivity(ctx, a.ProcessOrder, "order-1", 500)
	_ = workflow.ExecuteLocalActivity(ctx, a.Cleanup, "job-1")
	_ = workflow.ExecuteActivity(ctx, ArchiveAll, "bucket-1")

	// Variadic activity: zero, one, and several trailing args are all fine.
	_ = workflow.ExecuteActivity(ctx, a.Notify, "user-1")
	_ = workflow.ExecuteActivity(ctx, a.Notify, "user-1", "tag-a")
	_ = workflow.ExecuteActivity(ctx, a.Notify, "user-1", "tag-a", "tag-b")

	// Correct child workflow.
	_ = workflow.ExecuteChildWorkflow(ctx, ShipmentWorkflow, "order-1")

	// String-registered target: cannot be resolved to a signature, so skipped.
	_ = workflow.ExecuteActivity(ctx, "ProcessOrder", "order-1", 500)

	// Spread call: cannot be matched positionally, so skipped.
	extra := []any{"user-1", "tag-a"}
	_ = workflow.ExecuteActivity(ctx, a.Notify, extra...)

	return nil
}
