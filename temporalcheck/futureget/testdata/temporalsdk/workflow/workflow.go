// Package workflow is a minimal stand-in for go.temporal.io/sdk/workflow. It
// exists only so the analyzer's testdata type-checks without vendoring the real
// Temporal SDK. Like the real SDK, Future and ChildWorkflowFuture are declared in
// the internal package and re-exported here as aliases; the analyzer must resolve
// those aliases to the internal types to match a discarded .Get on them.
package workflow

import "go.temporal.io/sdk/internal"

// Context mirrors the real SDK, which publishes workflow.Context as an alias to
// an internal type rather than declaring it directly here.
type Context = internal.Context

// Future and ChildWorkflowFuture are aliases to the internal definitions, exactly
// as the real SDK re-exports them (workflow.Future = internal.Future).
type (
	Future              = internal.Future
	ChildWorkflowFuture = internal.ChildWorkflowFuture
)

// ExecuteActivity returns a Future, the receiver whose discarded .Get the
// analyzer flags. The body is irrelevant; fixtures only need the static type.
func ExecuteActivity(ctx Context, activity interface{}, args ...interface{}) Future { return nil }

// ExecuteChildWorkflow returns a ChildWorkflowFuture.
func ExecuteChildWorkflow(ctx Context, childWorkflow interface{}, args ...interface{}) ChildWorkflowFuture {
	return nil
}
