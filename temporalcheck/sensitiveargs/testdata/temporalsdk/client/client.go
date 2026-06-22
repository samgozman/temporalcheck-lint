// Package client is a minimal stand-in for go.temporal.io/sdk/client. The real
// SDK declares the Client interface directly in this package (with ExecuteWorkflow
// on it), so its method's receiver is client.Client -- not internal.Client. The
// stub mirrors that exactly so the analyzer's receiver match is exercised the way
// it resolves against the real SDK.
package client

import "context"

type StartWorkflowOptions struct{}

type WorkflowRun interface{ GetID() string }

type Client interface {
	ExecuteWorkflow(ctx context.Context, options StartWorkflowOptions, workflow any, args ...any) (WorkflowRun, error)
	// SignalWithStartWorkflow mirrors the SDK: the workflow target sits after the
	// signal fields and options (the 6th argument), so the analyzer resolves it by
	// a per-entry target index rather than a fixed position.
	SignalWithStartWorkflow(ctx context.Context, workflowID string, signalName string, signalArg any, options StartWorkflowOptions, workflow any, args ...any) (WorkflowRun, error)
}
