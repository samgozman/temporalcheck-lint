package clientstart

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// payWorkflow is started through the client, not from inside a workflow; its
// leading workflow.Context is the injected one and is skipped.
func payWorkflow(ctx workflow.Context, cardNumber string) error { return nil }

func caller(ctx context.Context, c client.Client) {
	c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, payWorkflow, "n")                          // want `workflow "payWorkflow" parameter 1 "cardNumber" matches the sensitive-data pattern;.*\(sensitive\)`
	c.SignalWithStartWorkflow(ctx, "id", "sig", nil, client.StartWorkflowOptions{}, payWorkflow, "n") // want `workflow "payWorkflow" parameter 1 "cardNumber" matches the sensitive-data pattern;.*\(sensitive\)`
}
