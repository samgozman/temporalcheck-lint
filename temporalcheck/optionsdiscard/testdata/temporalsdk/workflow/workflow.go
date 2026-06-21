// Package workflow is a minimal stand-in for go.temporal.io/sdk/workflow.
// It exists only so the analyzer's testdata type-checks without vendoring the
// real Temporal SDK. The signatures mirror the SDK's: each With*Options call
// takes a context and an options value and returns a NEW context carrying those
// options -- it does not mutate the one passed in, which is what the analyzer
// exists to enforce.
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

// GetLogger is a non-entry-point workflow function, used by fixtures to prove a
// discarded call to some other workflow.* function is left alone.
func GetLogger(ctx Context) any { return nil }
