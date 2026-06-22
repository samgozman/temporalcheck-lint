package bad

import "go.temporal.io/sdk/workflow"

// ShipmentWorkflow takes the injected workflow.Context plus one real argument.
func ShipmentWorkflow(ctx workflow.Context, orderID string) error {
	return nil
}

func Workflow(ctx workflow.Context) error {
	var a *Activities

	// Too few arguments.
	_ = workflow.ExecuteActivity(ctx, a.Greet) // want `ExecuteActivity: activity "Greet" expects 1 argument, got 0 \(arity\)`

	// Too many arguments.
	_ = workflow.ExecuteActivity(ctx, a.Greet, "world", "extra") // want `ExecuteActivity: activity "Greet" expects 1 argument, got 2 \(arity\)`

	// Wrong type: int where a string is expected.
	_ = workflow.ExecuteActivity(ctx, a.Greet, 42) // want `ExecuteActivity: arg 1 of "Greet" has type (untyped )?int, want string \(strict-types\)`

	// Wrong type on the second argument: string where an int is expected.
	_ = workflow.ExecuteActivity(ctx, a.ProcessOrder, "order-1", "oops") // want `ExecuteActivity: arg 2 of "ProcessOrder" has type (untyped )?string, want int \(strict-types\)`

	// Activity without a leading context: one argument too many.
	_ = workflow.ExecuteLocalActivity(ctx, a.Cleanup, "job-1", "extra") // want `ExecuteLocalActivity: activity "Cleanup" expects 1 argument, got 2 \(arity\)`

	// Variadic activity: missing the fixed userID argument.
	_ = workflow.ExecuteActivity(ctx, a.Notify) // want `ExecuteActivity: activity "Notify" expects at least 1 argument, got 0 \(arity\)`

	// Variadic activity: wrong type for the fixed argument.
	_ = workflow.ExecuteActivity(ctx, a.Notify, 42) // want `ExecuteActivity: arg 1 of "Notify" has type (untyped )?int, want string \(strict-types\)`

	// Variadic activity: wrong element type among the trailing args.
	_ = workflow.ExecuteActivity(ctx, a.Notify, "user-1", 7) // want `ExecuteActivity: arg 2 of "Notify" has type (untyped )?int, want string \(strict-types\)`

	// Child workflow missing its orderID argument.
	_ = workflow.ExecuteChildWorkflow(ctx, ShipmentWorkflow) // want `ExecuteChildWorkflow: child workflow "ShipmentWorkflow" expects 1 argument, got 0 \(arity\)`

	// Child workflow with a wrong-typed argument.
	_ = workflow.ExecuteChildWorkflow(ctx, ShipmentWorkflow, 99) // want `ExecuteChildWorkflow: arg 1 of "ShipmentWorkflow" has type (untyped )?int, want string \(strict-types\)`

	// Wrong type rendered with a package qualifier: workflow.Context, not string.
	_ = workflow.ExecuteActivity(ctx, a.Greet, ctx) // want `ExecuteActivity: arg 1 of "Greet" has type workflow.Context, want string \(strict-types\)`

	// Continue-as-new restarts a workflow; its target carries the same leading
	// workflow.Context, so arity and types are checked like a child workflow.
	_ = workflow.NewContinueAsNewError(ctx, ShipmentWorkflow)     // want `NewContinueAsNewError: workflow "ShipmentWorkflow" expects 1 argument, got 0 \(arity\)`
	_ = workflow.NewContinueAsNewError(ctx, ShipmentWorkflow, 99) // want `NewContinueAsNewError: arg 1 of "ShipmentWorkflow" has type (untyped )?int, want string \(strict-types\)`

	return nil
}
