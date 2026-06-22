// Package nolint exercises per-call //nolint suppression. A call carrying a
// directive that names the plugin (temporalcheck), names all, or is bare must
// report nothing; a directive naming only other linters, or the analyzer name,
// must not suppress.
package nolint

import "go.temporal.io/sdk/workflow"

func Workflow(ctx workflow.Context) error {
	// Bare //nolint suppresses every linter, including this one.
	workflow.NewContinueAsNewError(ctx, Workflow) //nolint

	// Names the plugin explicitly, with a trailing explanation.
	workflow.NewContinueAsNewError(ctx, Workflow) //nolint:temporalcheck // intentional

	// Names all linters.
	workflow.NewContinueAsNewError(ctx, Workflow) //nolint:all

	// A directive on any line the call spans suppresses it, even across lines.
	workflow.NewContinueAsNewError( //nolint:temporalcheck
		ctx,
		Workflow,
	)

	// Only other linters named: the diagnostic still fires.
	workflow.NewContinueAsNewError(ctx, Workflow) //nolint:gocritic // want `NewContinueAsNewError: the continue-as-new error is discarded; return it so the workflow continues as new, otherwise the workflow silently ends instead \(continue-as-new\)`

	// Names the analyzer rather than the plugin: golangci-lint knows this linter
	// only as "temporalcheck", so this does not suppress.
	workflow.NewContinueAsNewError(ctx, Workflow) //nolint:continueasnew // want `NewContinueAsNewError: the continue-as-new error is discarded; return it so the workflow continues as new, otherwise the workflow silently ends instead \(continue-as-new\)`

	return nil
}
