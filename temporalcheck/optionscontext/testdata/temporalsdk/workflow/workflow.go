// Package workflow is a minimal stand-in for go.temporal.io/sdk/workflow.
// It exists only so the analyzer's testdata type-checks without vendoring the
// real Temporal SDK. The signatures mirror the SDK's: each With*Options call
// returns a NEW context carrying the matching options, and each Execute* call
// reads those options back out -- so crossing an activity context with a child
// call (and vice versa) is what the analyzer exists to catch.
package workflow

import "go.temporal.io/sdk/internal"

// Context mirrors the real SDK, which publishes workflow.Context as an alias to
// an internal type rather than declaring it directly in this package.
type Context = internal.Context

// The options types are stand-ins for the real SDK structs; the fixtures only
// need a type to pass, not its fields.
type (
	ActivityOptions      struct{ TaskQueue string }
	LocalActivityOptions struct{ TaskQueue string }
	ChildWorkflowOptions struct{ TaskQueue string }
)

func WithActivityOptions(ctx Context, options ActivityOptions) Context { return ctx }

func WithLocalActivityOptions(ctx Context, options LocalActivityOptions) Context { return ctx }

// WithChildOptions is the public name the SDK exports for the child-workflow
// options setter (the underlying internal function is WithChildWorkflowOptions).
func WithChildOptions(ctx Context, cwo ChildWorkflowOptions) Context { return ctx }

type Future interface {
	Get(ctx Context, valuePtr any) error
}

type ChildWorkflowFuture interface{ Future }

func ExecuteActivity(ctx Context, activity any, args ...any) Future { return nil }

func ExecuteLocalActivity(ctx Context, activity any, args ...any) Future { return nil }

func ExecuteChildWorkflow(ctx Context, childWorkflow any, args ...any) ChildWorkflowFuture {
	return nil
}

// WithValue is a non-entry-point helper returning a Context, used by fixtures to
// prove that an opaque reassignment makes the context's options unknown.
func WithValue(ctx Context, key any, val any) Context { return ctx }

// Go runs f in a coroutine, used by fixtures to exercise the closure-capture bail.
func Go(ctx Context, f func(ctx Context)) {}

// GetLogger is a non-entry-point workflow function, used by fixtures to prove a
// call to some other workflow.* function is left alone.
func GetLogger(ctx Context) any { return nil }
