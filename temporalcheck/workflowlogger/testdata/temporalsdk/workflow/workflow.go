// Package workflow is a minimal stand-in for go.temporal.io/sdk/workflow. It
// exists only so the analyzer's testdata type-checks without vendoring the real
// Temporal SDK. Like the real SDK, Context is declared in the internal package and
// re-exported here as an alias; the analyzer resolves that alias when deciding
// whether a function is a workflow definition.
package workflow

import "go.temporal.io/sdk/internal"

// Context mirrors the real SDK, which publishes workflow.Context as an alias to
// an internal type rather than declaring it directly here.
type Context = internal.Context

// Go runs f as a deterministic workflow coroutine. Its closure is still workflow
// code, so logging inside it is flagged. The body is irrelevant; fixtures only
// need the static signature.
func Go(ctx Context, f func(ctx Context)) {}

// Logger is the replay-aware logger the SDK hands back from GetLogger. Its methods
// live in this package (not log/slog), so calling them is the correct,
// non-flagged way to log from a workflow.
type Logger interface {
	Debug(msg string, keyvals ...interface{})
	Info(msg string, keyvals ...interface{})
	Warn(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})
}

// GetLogger returns the replay-aware workflow logger; the benign fixture uses it
// as the correct alternative to stdlib logging.
func GetLogger(ctx Context) Logger { return nil }
