// Package internal is a stand-in for go.temporal.io/sdk/internal. The real SDK
// declares Future and ChildWorkflowFuture here and re-publishes them from the
// workflow package as aliases (type Future = internal.Future), mirroring
// workflow.Context. The fixtures reproduce that shape so the analyzer is
// exercised against the SDK's real type identities: a workflow.Future value
// resolves to internal.Future, not a fresh type declared in the workflow stub.
package internal

type Context interface{ isWorkflowContext() }

// Future mirrors the SDK interface. Only Get returns an error; IsReady is here so
// the interface has the shape that makes "match the Get method on this receiver
// type" meaningful rather than trivially the whole interface.
type Future interface {
	Get(ctx Context, valuePtr interface{}) error
	IsReady() bool
}

// ChildWorkflowFuture mirrors the SDK interface: it embeds Future (so Get is
// promoted) and adds child-only methods that return a Future, not an error.
type ChildWorkflowFuture interface {
	Future
	GetChildWorkflowExecution() Future
	SignalChildWorkflow(ctx Context, signalName string, data interface{}) Future
}
