// Package clientexec exercises the client.Client entry points: ExecuteWorkflow
// (target third) and SignalWithStartWorkflow (target sixth). Both are matched by
// receiver, and both carry a leading workflow.Context on the target, so arity and
// strict-type checks apply just as they do to workflow.ExecuteChildWorkflow --
// only the target's argument index differs.
package clientexec

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// OrderWorkflow takes the injected workflow.Context plus one real argument.
func OrderWorkflow(ctx workflow.Context, orderID string) error { return nil }

func start(ctx context.Context, c client.Client) {
	// Correct arity and type.
	_, _ = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, OrderWorkflow, "order-1")

	// Too few arguments.
	_, _ = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, OrderWorkflow) // want `ExecuteWorkflow: workflow "OrderWorkflow" expects 1 argument, got 0 \(arity\)`

	// Wrong type: int where a string is expected.
	_, _ = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, OrderWorkflow, 99) // want `ExecuteWorkflow: arg 1 of "OrderWorkflow" has type (untyped )?int, want string \(strict-types\)`

	// Correct signal-with-start.
	_, _ = c.SignalWithStartWorkflow(ctx, "wf-id", "sig", nil, client.StartWorkflowOptions{}, OrderWorkflow, "order-1")

	// Too many arguments past the sixth target slot.
	_, _ = c.SignalWithStartWorkflow(ctx, "wf-id", "sig", nil, client.StartWorkflowOptions{}, OrderWorkflow, "a", "b") // want `SignalWithStartWorkflow: workflow "OrderWorkflow" expects 1 argument, got 2 \(arity\)`

	// Wrong type for the target's argument.
	_, _ = c.SignalWithStartWorkflow(ctx, "wf-id", "sig", nil, client.StartWorkflowOptions{}, OrderWorkflow, 99) // want `SignalWithStartWorkflow: arg 1 of "OrderWorkflow" has type (untyped )?int, want string \(strict-types\)`

	// String-registered target: cannot be resolved to a signature, so skipped.
	_, _ = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, "OrderWorkflow", "order-1")
}
