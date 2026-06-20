// Package conformance is a compile-time contract test against the real Temporal
// SDK. It is a separate module so the SDK's dependency tree stays out of the main
// module and the offline test stub. CI builds it and Dependabot bumps the SDK,
// so a breaking signature change fails here — the cue to update the stub.
package conformance

import "go.temporal.io/sdk/workflow"

// execargs reads each Execute* call as (ctx, target, args...). These assignments
// stop compiling if the real SDK changes that shape; keep them in sync with the
// stub at testdata/temporalsdk.
var (
	_ func(workflow.Context, interface{}, ...interface{}) workflow.Future              = workflow.ExecuteActivity
	_ func(workflow.Context, interface{}, ...interface{}) workflow.Future              = workflow.ExecuteLocalActivity
	_ func(workflow.Context, interface{}, ...interface{}) workflow.ChildWorkflowFuture = workflow.ExecuteChildWorkflow
)
