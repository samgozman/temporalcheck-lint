// Package conformance is a compile-time contract test against the real Temporal
// SDK. It is a separate module so the SDK's dependency tree stays out of the main
// module and the offline test stub. CI builds it and Dependabot bumps the SDK,
// so a breaking signature change fails here — the cue to update the stub.
package conformance

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
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

// activitytimeout reads the two required-timeout fields off each options struct.
// These references stop compiling if the real SDK renames or drops either field —
// the cue to update the stub and the check's requiredTimeouts list. The values are
// discarded; this is a compile-time field-existence assertion only.
var (
	_ = workflow.ActivityOptions{}.StartToCloseTimeout
	_ = workflow.ActivityOptions{}.ScheduleToCloseTimeout
	_ = workflow.LocalActivityOptions{}.StartToCloseTimeout
	_ = workflow.LocalActivityOptions{}.ScheduleToCloseTimeout
)

// futureget reads the Get method off each result type whose returned error must
// not be discarded. These interface method expressions stop compiling if the
// real SDK renames Get, drops it from one of these types, or changes its
// signature (Future/ChildWorkflowFuture take a ctx; EncodedValue does not) — the
// cue to update the stub and the check's receiverTypes map.
var (
	_ func(workflow.Future, workflow.Context, interface{}) error              = workflow.Future.Get
	_ func(workflow.ChildWorkflowFuture, workflow.Context, interface{}) error = workflow.ChildWorkflowFuture.Get
	_ func(converter.EncodedValue, interface{}) error                         = converter.EncodedValue.Get
)

// lossynumber resolves the workflow target of client.ExecuteWorkflow, read as
// (ctx, options, target, args...). This method expression stops compiling if the
// real SDK changes that shape or moves ExecuteWorkflow off client.Client — the
// cue to update the stub and the check's client entry; keep the client stub at
// testdata/temporalsdk in sync.
var temporalClient client.Client

var _ func(context.Context, client.StartWorkflowOptions, interface{}, ...interface{}) (client.WorkflowRun, error) = temporalClient.ExecuteWorkflow
