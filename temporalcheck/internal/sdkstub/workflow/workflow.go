// Package workflow is a stand-in for go.temporal.io/sdk/workflow. It exists only
// so the analyzers' fixtures type-check without vendoring the real SDK. The
// Context/Future/options types are aliases to the internal definitions, exactly
// as the real SDK re-exports them, so a type-matching analyzer resolves the alias
// to the internal type. The Execute*/With*Options signatures mirror the SDK's: the
// target and arguments are interface{}, the type erasure the analyzers compensate
// for by inspecting the target's real signature.
package workflow

import "go.temporal.io/sdk/internal"

// Context and the option/future types are aliases to the internal definitions,
// exactly as the real SDK re-exports them (workflow.Context = internal.Context).
type (
	Context              = internal.Context
	Future               = internal.Future
	ChildWorkflowFuture  = internal.ChildWorkflowFuture
	Selector             = internal.Selector
	ActivityOptions      = internal.ActivityOptions
	LocalActivityOptions = internal.LocalActivityOptions
	RetryPolicy          = internal.RetryPolicy
)

// ChildWorkflowOptions is the child-workflow options struct. No analyzer matches
// it by type identity (only the WithChildOptions function is matched), so the stub
// declares it directly; fixtures only need a type to pass.
type ChildWorkflowOptions struct{ TaskQueue string }

// Logger is the replay-aware logger the SDK hands back from GetLogger. Its methods
// live in this package (not log/slog), so calling them is the correct,
// non-flagged way to log from a workflow.
type Logger interface {
	Debug(msg string, keyvals ...interface{})
	Info(msg string, keyvals ...interface{})
	Warn(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})
}

func WithActivityOptions(ctx Context, options ActivityOptions) Context { return ctx }

func WithLocalActivityOptions(ctx Context, options LocalActivityOptions) Context { return ctx }

// WithChildOptions is the public name the SDK exports for the child-workflow
// options setter (the underlying internal function is WithChildWorkflowOptions).
func WithChildOptions(ctx Context, cwo ChildWorkflowOptions) Context { return ctx }

// WithValue is a non-entry-point helper returning a Context, used by fixtures to
// prove that an opaque reassignment makes the context's options unknown.
func WithValue(ctx Context, key any, val any) Context { return ctx }

func ExecuteActivity(ctx Context, activity any, args ...any) Future { return nil }

func ExecuteLocalActivity(ctx Context, activity any, args ...any) Future { return nil }

func ExecuteChildWorkflow(ctx Context, childWorkflow any, args ...any) ChildWorkflowFuture {
	return nil
}

// NewContinueAsNewError returns the error a workflow must return to continue as
// new. The workflow target is interface{} and its arguments are variadic
// interface{}, the same type erasure as Execute*.
func NewContinueAsNewError(ctx Context, wfn any, args ...any) error { return nil }

// Go runs f as a deterministic workflow coroutine. Its closure is still workflow
// code (logging there is flagged) and may capture locals from the enclosing
// workflow (mutating those is idiomatic and must not be flagged).
func Go(ctx Context, f func(ctx Context)) {}

// Await blocks until condition returns true. The condition closure reads a
// captured local, the documented pattern for awaiting state.
func Await(ctx Context, condition func() bool) error { return nil }

// NewSelector returns a Selector whose AddFuture callbacks write captured locals.
func NewSelector(ctx Context) Selector { return nil }

// GetLogger returns the replay-aware workflow logger; the correct alternative to
// stdlib logging from workflow code.
func GetLogger(ctx Context) Logger { return nil }
