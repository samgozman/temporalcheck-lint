// Package clientstart is exercised with the check enabled: the client.Client
// methods that name a workflow target -- ExecuteWorkflow and
// SignalWithStartWorkflow -- are flagged when that target is a string, exactly
// like the workflow.Execute* calls, with the target resolved at its per-method
// index. A function-reference target is left alone.
package clientstart

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func myWorkflow(ctx workflow.Context) error { return nil }

func start(ctx context.Context, c client.Client) {
	// ExecuteWorkflow names its target third: ExecuteWorkflow(ctx, options, target, args...).
	_, _ = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, "MyWorkflow", 1) // want `ExecuteWorkflow: target "MyWorkflow" is named by string; pass the function reference instead so its arguments can be checked statically \(string-target\)`

	// SignalWithStartWorkflow names its target sixth, after the signal fields and options.
	_, _ = c.SignalWithStartWorkflow(ctx, "wf-id", "sig", nil, client.StartWorkflowOptions{}, "MyWorkflow", 1) // want `SignalWithStartWorkflow: target "MyWorkflow" is named by string; pass the function reference instead so its arguments can be checked statically \(string-target\)`

	// The recommended form: a function reference. Never flagged.
	_, _ = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, myWorkflow, 1)
}
