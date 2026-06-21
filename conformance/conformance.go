// Package conformance is a compile-time contract test against the real Temporal
// SDK. It is a separate module so the SDK's dependency tree stays out of the main
// module and the offline test stub. CI builds it and Dependabot bumps the SDK,
// so a breaking signature change fails here — the cue to update the stub.
package conformance

import (
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// execargs reads each Execute* call as (ctx, target, args...). These assignments
// stop compiling if the real SDK changes that shape; keep them in sync with the
// stub at testdata/temporalsdk.
var (
	_ func(workflow.Context, interface{}, ...interface{}) workflow.Future              = workflow.ExecuteActivity
	_ func(workflow.Context, interface{}, ...interface{}) workflow.Future              = workflow.ExecuteLocalActivity
	_ func(workflow.Context, interface{}, ...interface{}) workflow.ChildWorkflowFuture = workflow.ExecuteChildWorkflow
)

// The strict-tests checks read each TestWorkflowEnvironment mock setup as
// (target, matchers...). These assignments stop compiling if the real SDK
// changes that shape or moves the type out of testsuite; keep them in sync with
// the testsuite stub at testdata/temporalsdk.
var testEnv *testsuite.TestWorkflowEnvironment

var (
	_ func(interface{}, ...interface{}) *testsuite.MockCallWrapper = testEnv.OnActivity
	_ func(interface{}, ...interface{}) *testsuite.MockCallWrapper = testEnv.OnWorkflow
)

// optionsdiscard reads each With*Options call as (ctx, options) returning a new
// Context. These assignments stop compiling if the real SDK changes that shape
// — e.g. if a call started mutating ctx in place — which is the cue to revisit
// the check; keep the option types in sync with the stub at testdata/temporalsdk.
var (
	_ func(workflow.Context, workflow.ActivityOptions) workflow.Context      = workflow.WithActivityOptions
	_ func(workflow.Context, workflow.LocalActivityOptions) workflow.Context = workflow.WithLocalActivityOptions
	_ func(workflow.Context, workflow.ChildWorkflowOptions) workflow.Context = workflow.WithChildOptions
)
