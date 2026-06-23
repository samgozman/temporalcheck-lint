// Package client is a stand-in for go.temporal.io/sdk/client. The real SDK
// declares the Client interface directly in this package (with ExecuteWorkflow and
// SignalWithStartWorkflow on it), so a method's receiver is client.Client -- not
// internal.Client. The stub mirrors that exactly so the analyzers' receiver match
// is exercised the way it resolves against the real SDK. The workflow target and
// its arguments are interface{}, the type erasure the analyzers resolve by
// inspecting the target's real signature.
package client

import "context"

type StartWorkflowOptions struct{}

type WorkflowRun interface{ GetID() string }

type Client interface {
	ExecuteWorkflow(ctx context.Context, options StartWorkflowOptions, workflow any, args ...any) (WorkflowRun, error)
	// SignalWithStartWorkflow names its workflow target as the 6th argument (after
	// the signal fields and options), so the analyzer resolves it by a per-entry
	// target index rather than a fixed position.
	SignalWithStartWorkflow(ctx context.Context, workflowID string, signalName string, signalArg any, options StartWorkflowOptions, workflow any, args ...any) (WorkflowRun, error)
	// Close releases the client's resources; the workeroptions fixtures call it on
	// the client.Client passed to worker.New.
	Close()
}
