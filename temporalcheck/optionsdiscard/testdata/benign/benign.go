// Package benign exercises the shapes the analyzer must leave alone: the correct
// `ctx = ...` form, a `:=` to a fresh variable, the result passed on or used, a
// non-selector call, a selector into another package, and a discarded call to a
// workflow.* function that is not a With*Options entry point. None of these
// produce diagnostics.
package benign

import (
	"strings"

	"go.temporal.io/sdk/workflow"
)

func helper() {}

func consume(ctx workflow.Context) {}

// holder carries a function-typed field, so a call through it is a selector
// whose resolved object is a variable rather than a function.
type holder struct{ run func() }

func Workflow(ctx workflow.Context) error {
	var ao workflow.ActivityOptions

	// The correct form: the returned context is assigned back.
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Assigned to a fresh variable with `:=`; the analyzer does not track later
	// use, so a kept result is never flagged.
	nctx := workflow.WithActivityOptions(ctx, ao)
	consume(nctx)

	// Result passed straight into another call: used, not discarded.
	consume(workflow.WithActivityOptions(ctx, ao))

	// Non-selector call: call.Fun is a bare identifier, not a selector.
	helper()

	// Selector whose callee is a func-typed field: the resolved object is a
	// variable, not a function, so the analyzer can't treat it as a known callee.
	var h holder
	h.run()

	// Selector into a non-Temporal package: the resolved callee is out of scope.
	_ = strings.ToUpper("x")

	// A discarded call to a workflow.* function that is not a With*Options entry
	// point: resolved to the workflow package but ignored.
	workflow.GetLogger(ctx)

	return nil
}
