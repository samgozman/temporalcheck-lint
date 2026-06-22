// Package benign exercises the shapes the analyzer must leave alone: a result
// returned directly, a result kept in a named variable and returned (possibly
// only conditionally), the same-named function from another package, and
// unrelated bare calls. None produce diagnostics.
package benign

import (
	"strings"

	"continueasnewfixtures/other"

	"go.temporal.io/sdk/workflow"
)

func ReturnedDirectly(ctx workflow.Context) error {
	// The correct pattern: returned straight out.
	return workflow.NewContinueAsNewError(ctx, ReturnedDirectly)
}

func ReturnedViaVar(ctx workflow.Context) error {
	// Kept in a named variable, then returned: a `return` follows, so it is left
	// alone. Proving a named target is never returned would need flow analysis;
	// only the unambiguous discards (bare statement, blank assignment) are flagged.
	err := workflow.NewContinueAsNewError(ctx, ReturnedViaVar)
	return err
}

func ReturnedConditionally(ctx workflow.Context, cond bool) error {
	err := workflow.NewContinueAsNewError(ctx, ReturnedConditionally)
	if cond {
		return err
	}
	return nil
}

func NotTheWorkflowFunc(ctx workflow.Context) error {
	// Same function name, different package: matched by package path, so neither a
	// bare call nor a blank assignment here is flagged.
	other.NewContinueAsNewError(ctx, nil)
	_ = other.NewContinueAsNewError(ctx, nil)
	return nil
}

func Unrelated() {
	// An unrelated bare call statement and a non-Temporal selector are not flagged.
	helper()
	_ = strings.ToUpper("x")
}

func helper() {}
