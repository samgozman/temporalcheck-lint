// Package workflow is a minimal stand-in for go.temporal.io/sdk/workflow. It
// exists only so the analyzer's testdata type-checks without vendoring the real
// Temporal SDK. NewContinueAsNewError is a plain package function (not a type),
// so unlike workflow.Context it is declared directly here rather than aliased
// from internal; the analyzer matches it by package path + name, which a
// function-shaped stub reproduces faithfully (conformance pins its signature).
package workflow

// Context mirrors the real SDK's workflow context type. A bare interface is
// enough for fixtures, which only thread a Context value through.
type Context interface{ isWorkflowContext() }

// NewContinueAsNewError returns the error a workflow must *return* to continue as
// new. The body is irrelevant; fixtures only need the static signature.
func NewContinueAsNewError(ctx Context, wfn interface{}, args ...interface{}) error {
	return nil
}
