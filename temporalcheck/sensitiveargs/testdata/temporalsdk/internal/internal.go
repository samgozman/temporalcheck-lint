// Package internal is a stand-in for go.temporal.io/sdk/internal. The real SDK
// declares workflow.Context here and re-publishes it from the workflow package as
// an alias (type Context = internal.Context), so the fixtures mirror that shape to
// exercise the analyzer against the SDK's real type identity.
package internal

// Context is the workflow context, published from workflow as an alias.
type Context interface{ isWorkflowContext() }
