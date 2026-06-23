// Package workflow is a minimal stand-in for go.temporal.io/sdk/workflow. It
// exists only so the analyzer's testdata type-checks without vendoring the real
// Temporal SDK. Like the real SDK, Context/Future/Selector are declared in the
// internal package and re-exported here as aliases; the analyzer resolves those
// aliases when deciding whether a function is a workflow definition.
package workflow

import "go.temporal.io/sdk/internal"

// Context mirrors the real SDK, which publishes workflow.Context as an alias to
// an internal type rather than declaring it directly here.
type Context = internal.Context

// Future and Selector are aliases to the internal definitions, exactly as the
// real SDK re-exports them.
type (
	Future   = internal.Future
	Selector = internal.Selector
)

// Go runs f as a deterministic workflow coroutine. Its closure captures locals
// from the enclosing workflow function; mutating those is idiomatic and must not
// be flagged. The body is irrelevant; fixtures only need the static signature.
func Go(ctx Context, f func(ctx Context)) {}

// Await blocks until condition returns true. The condition closure reads a
// captured local, the documented pattern for awaiting state.
func Await(ctx Context, condition func() bool) error { return nil }

// NewSelector returns a Selector whose AddFuture callbacks write captured locals.
func NewSelector(ctx Context) Selector { return nil }

// ExecuteActivity returns a Future, used by the Selector idiom fixtures.
func ExecuteActivity(ctx Context, activity interface{}, args ...interface{}) Future { return nil }
