// Package workflow is a minimal stand-in for go.temporal.io/sdk/workflow. It
// exists only so the analyzer's testdata type-checks without vendoring the real
// Temporal SDK. Like the real SDK, the option structs are declared in the
// internal package and re-exported here as aliases; both ActivityOptions and
// LocalActivityOptions carry StartToCloseTimeout and ScheduleToCloseTimeout, at
// least one of which Temporal requires -- which is what the analyzer enforces.
package workflow

import "go.temporal.io/sdk/internal"

// Context mirrors the real SDK, which publishes workflow.Context as an alias to
// an internal type rather than declaring it directly in this package.
type Context = internal.Context

// The option types are aliases to the internal definitions, exactly as the real
// SDK re-exports them. The analyzer must resolve these aliases to the internal
// types, so the fixtures reproduce that identity rather than declaring fresh
// structs in this package.
type (
	ActivityOptions      = internal.ActivityOptions
	LocalActivityOptions = internal.LocalActivityOptions
	RetryPolicy          = internal.RetryPolicy
)

func WithActivityOptions(ctx Context, options ActivityOptions) Context { return ctx }

func WithLocalActivityOptions(ctx Context, options LocalActivityOptions) Context { return ctx }
