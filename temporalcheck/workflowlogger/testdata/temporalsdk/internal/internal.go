// Package internal is a stand-in for go.temporal.io/sdk/internal. The real SDK
// declares Context here and re-publishes it from the workflow package as an alias
// (type Context = internal.Context). The fixtures reproduce that shape so the
// analyzer is exercised against the SDK's real type identity: a workflow.Context
// parameter resolves to internal.Context, not a fresh type declared in the
// workflow stub. The workflow-scope detection must resolve that alias to match a
// workflow definition.
package internal

// Context mirrors the SDK's workflow context, the first parameter that marks a
// function as a workflow definition.
type Context interface{ isWorkflowContext() }
