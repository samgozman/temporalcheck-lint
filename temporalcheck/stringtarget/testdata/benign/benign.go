// Package benign exercises the call shapes the analyzer must ignore even when
// enabled: a non-selector call, a selector whose callee is a func-typed field
// (not a function), a selector into a non-Temporal package, a function-reference
// target (the form we want), and a workflow-package function that isn't an
// Execute* entry point. None of these produce diagnostics.
package benign

import (
	"context"
	"strings"

	"go.temporal.io/sdk/workflow"
)

type Activities struct{}

func (a *Activities) Greet(ctx context.Context, name string) error { return nil }

func helper() {}

// holder carries a function-typed field, so a call through it is a selector
// whose resolved object is a variable rather than a function.
type holder struct{ run func() }

func Workflow(ctx workflow.Context) error {
	var a *Activities

	// Non-selector call: call.Fun is a bare identifier, not a selector.
	helper()

	// Selector whose callee is a func-typed field: the resolved object is a
	// variable, not a function, so the analyzer can't treat it as a known callee.
	var h holder
	h.run()

	// Selector into a non-Temporal package: the resolved callee is out of scope.
	_ = strings.ToUpper("x")

	// Function-reference target: the value we want callers to pass; not flagged.
	_ = workflow.ExecuteActivity(ctx, a.Greet, "world")

	// A workflow-package function that is not an Execute* entry point: the
	// Future's Get method is resolved in the workflow package but ignored.
	fut := workflow.ExecuteActivity(ctx, a.Greet, "world")
	_ = fut.Get(ctx, nil)

	return nil
}
