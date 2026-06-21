// Package client is a minimal stand-in for go.temporal.io/sdk/client. The real
// SDK declares the Client interface directly in this package (with ExecuteWorkflow
// on it), so its method's receiver is client.Client -- not internal.Client. The
// stub mirrors that exactly so the analyzer's receiver match is exercised the way
// it resolves against the real SDK. ExecuteWorkflow takes a standard
// context.Context, options, the workflow target as interface{}, and variadic
// interface{} args -- the type erasure the analyzer compensates for.
package client

import "context"

type StartWorkflowOptions struct{}

type WorkflowRun interface{ GetID() string }

type Client interface {
	ExecuteWorkflow(ctx context.Context, options StartWorkflowOptions, workflow any, args ...any) (WorkflowRun, error)
}
