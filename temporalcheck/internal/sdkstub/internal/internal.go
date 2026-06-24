// Package internal is a stand-in for go.temporal.io/sdk/internal. The real SDK
// declares its core types here and re-publishes them from the public packages as
// aliases (type Context = internal.Context, type ActivityOptions =
// internal.ActivityOptions, ...). The fixtures mirror that shape so the analyzers
// resolve to the SDK's real type identities: an option literal or a workflow.Future
// value surfaces as an internal type, not a fresh type declared in the public
// stub package. A type-matching analyzer that tested against a public-package
// declaration would pass here yet match nothing against the real SDK.
package internal

import "time"

// Context is the workflow context, the first parameter that marks a function as a
// workflow definition. The unexported method keeps it un-implementable from
// fixtures, which only ever receive a Context, never construct one.
type Context interface{ isWorkflowContext() }

// RetryPolicy is a stand-in for the SDK's retry policy; fixtures only need a
// non-timeout field to set.
type RetryPolicy struct{ MaximumAttempts int32 }

// ActivityOptions mirrors the SDK struct. Field order matters: some fixtures
// construct a positional literal, so keep the order stable.
type ActivityOptions struct {
	TaskQueue              string
	ScheduleToCloseTimeout time.Duration
	ScheduleToStartTimeout time.Duration
	StartToCloseTimeout    time.Duration
	HeartbeatTimeout       time.Duration
	RetryPolicy            *RetryPolicy
}

// LocalActivityOptions mirrors the SDK struct; like ActivityOptions it requires
// one of the two timeouts.
type LocalActivityOptions struct {
	ScheduleToCloseTimeout time.Duration
	StartToCloseTimeout    time.Duration
	RetryPolicy            *RetryPolicy
}

// WorkerOptions mirrors the SDK struct, re-exported from the worker package as
// `type Options = internal.WorkerOptions`. The workeroptions analyzer reads the
// five concurrency-limit fields and the two workflow-task fields whose value may
// not be 1. Field order matters: a fixture constructs a positional literal, so
// keep the order stable.
type WorkerOptions struct {
	MaxConcurrentActivityExecutionSize      int
	MaxConcurrentWorkflowTaskExecutionSize  int
	MaxConcurrentActivityTaskPollers        int
	MaxConcurrentWorkflowTaskPollers        int
	MaxConcurrentLocalActivityExecutionSize int
	WorkerStopTimeout                       time.Duration
	Identity                                string
}

// Future mirrors the SDK interface, re-exported as workflow.Future. Get returns
// an error the futureget analyzer flags when discarded; IsReady gives the
// interface a second method so a receiver match is meaningful.
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

// Selector mirrors workflow.Selector: AddFuture registers a callback the
// deterministic runner invokes, the canonical place a captured local is mutated.
type Selector interface {
	AddFuture(future Future, f func(f Future)) Selector
	Select(ctx Context)
}

// MockCallWrapper stands in for the testsuite type of the same name, re-published
// from testsuite as an alias. It carries just enough to let fixtures chain
// .Return(...).Once() after a mock setup.
type MockCallWrapper struct{}

func (c *MockCallWrapper) Return(returnArguments ...any) *MockCallWrapper { return c }

func (c *MockCallWrapper) Once() *MockCallWrapper { return c }

// TestWorkflowEnvironment stands in for the testsuite type of the same name,
// re-published from testsuite as an alias. OnActivity/OnWorkflow take the target
// as interface{} and the matchers as variadic interface{} -- the type erasure the
// execargs/stringtarget strict-tests checks inspect.
type TestWorkflowEnvironment struct{}

func (e *TestWorkflowEnvironment) OnActivity(activity any, args ...any) *MockCallWrapper {
	return &MockCallWrapper{}
}

func (e *TestWorkflowEnvironment) OnWorkflow(workflow any, args ...any) *MockCallWrapper {
	return &MockCallWrapper{}
}
