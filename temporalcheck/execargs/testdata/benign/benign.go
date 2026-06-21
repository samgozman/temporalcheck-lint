// Package benign exercises the call shapes the analyzer must ignore without
// reporting: a non-selector call, a selector whose callee is a func-typed field
// (not a function), a selector into a non-Temporal package, a workflow-package
// function that isn't an Execute* entry point, and a zero-parameter activity
// target. None of these produce diagnostics.
package benign

import (
	"strings"

	"go.temporal.io/sdk/workflow"
)

// Ping is a zero-parameter activity, so the arity check computes want == 0.
func Ping() error { return nil }

func helper() {}

// holder carries a function-typed field, so a call through it is a selector
// whose resolved object is a variable rather than a function.
type holder struct{ run func() }

func Workflow(ctx workflow.Context) error {
	// Non-selector call: call.Fun is a bare identifier, not a selector.
	helper()

	// Selector whose callee is a func-typed field: the resolved object is a
	// variable, not a function, so the analyzer can't treat it as a known callee.
	var h holder
	h.run()

	// Selector into a non-Temporal package: the resolved callee is not in the
	// workflow package, so it is out of scope.
	_ = strings.ToUpper("x")

	// Zero-parameter activity target: a valid call with no trailing arguments.
	_ = workflow.ExecuteActivity(ctx, Ping)

	// A workflow-package function that is not an Execute* entry point: the
	// Future's Get method is resolved in the workflow package but ignored.
	fut := workflow.ExecuteActivity(ctx, Ping)
	_ = fut.Get(ctx, nil)

	return nil
}
