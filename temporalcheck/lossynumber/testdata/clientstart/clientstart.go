package clientstart

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func myWorkflow(ctx workflow.Context, v any) error { return nil }

func goodWorkflow(ctx workflow.Context, v int64) error { return nil }

func anyReturnWorkflow(ctx workflow.Context) (map[string]any, error) { return nil, nil }

func start(ctx context.Context, c client.Client) {
	c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, myWorkflow, 1)     // want `workflow "myWorkflow" parameter 1 has dynamic type any;.*\(lossy-types\)`
	c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, goodWorkflow, 1)   // concrete int64 parameter: no diagnostic
	c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, anyReturnWorkflow) // want `workflow "anyReturnWorkflow" return 1 has dynamic type map\[string\]any;.*\(lossy-types\)`
}
