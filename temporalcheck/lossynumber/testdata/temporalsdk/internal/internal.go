// Package internal is a stand-in for go.temporal.io/sdk/internal. The real SDK
// declares workflow.Context here and re-publishes it from the workflow package as
// an alias (type Context = internal.Context), so the fixtures mirror that shape
// to exercise the analyzer against the SDK's real type identity.
//
// Note that client.Client is NOT modeled here: unlike Context, the real SDK
// declares the Client interface directly in the client package (internal.Client
// is a separate interface that client.Client merely implements), so the stub
// declares Client in the stub client package. Mirroring that is what makes the
// analyzer's receiver match meaningful -- an alias here would test green yet
// match nothing against the real SDK.
package internal

// Context is the workflow context, published from workflow as an alias.
type Context interface{ isWorkflowContext() }
