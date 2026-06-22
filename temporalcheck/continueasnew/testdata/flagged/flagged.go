// Package flagged exercises the discarded shapes the analyzer reports: a bare
// NewContinueAsNewError call statement and an explicit blank assignment,
// including an aliased import of the workflow package. Each builds a
// continue-as-new error that is never returned, so the workflow silently ends.
package flagged

import (
	"go.temporal.io/sdk/workflow"

	wf "go.temporal.io/sdk/workflow"
)

func Workflow(ctx workflow.Context) error {
	// Bare statement: the continue-as-new error is built and thrown away, where
	// `return workflow.NewContinueAsNewError(...)` was meant.
	workflow.NewContinueAsNewError(ctx, Workflow) // want `NewContinueAsNewError: the continue-as-new error is discarded; return it so the workflow continues as new, otherwise the workflow silently ends instead \(continue-as-new\)`

	// Explicit blank assignment is also a discard.
	_ = workflow.NewContinueAsNewError(ctx, Workflow) // want `NewContinueAsNewError: the continue-as-new error is discarded; return it so the workflow continues as new, otherwise the workflow silently ends instead \(continue-as-new\)`

	// An aliased import of the workflow package resolves to the same function, so
	// it is flagged the same.
	wf.NewContinueAsNewError(ctx, Workflow) // want `NewContinueAsNewError: the continue-as-new error is discarded; return it so the workflow continues as new, otherwise the workflow silently ends instead \(continue-as-new\)`

	return nil
}
