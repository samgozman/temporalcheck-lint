package clientstart

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func myWorkflow(ctx workflow.Context, ch chan int) error { return nil }

func goodWorkflow(ctx workflow.Context, v int64) error { return nil }

func funcReturnWorkflow(ctx workflow.Context) (func(), error) { return nil, nil }

func start(ctx context.Context, c client.Client) {
	c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, myWorkflow, nil)    // want `workflow "myWorkflow" parameter 1 has type chan int;.*\(unencodable\)`
	c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, goodWorkflow, 1)    // concrete int64 parameter: no diagnostic
	c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, funcReturnWorkflow) // want `workflow "funcReturnWorkflow" return 1 has type func\(\);.*\(unencodable\)`

	// SignalWithStartWorkflow names its workflow target as the 6th argument (after
	// the signal fields and options), so the analyzer must resolve it by index.
	c.SignalWithStartWorkflow(ctx, "wf-id", "sig", nil, client.StartWorkflowOptions{}, myWorkflow, nil) // want `workflow "myWorkflow" parameter 1 has type chan int;.*\(unencodable\)`
	c.SignalWithStartWorkflow(ctx, "wf-id", "sig", nil, client.StartWorkflowOptions{}, goodWorkflow, 1) // concrete int64 parameter: no diagnostic
}
