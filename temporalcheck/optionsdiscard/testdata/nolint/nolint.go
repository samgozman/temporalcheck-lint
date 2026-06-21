// Package nolint exercises per-call //nolint suppression. A call carrying a
// directive that names the plugin (temporalcheck), names all, or is bare must
// report nothing; a directive naming only other linters, or the analyzer name,
// must not suppress.
package nolint

import "go.temporal.io/sdk/workflow"

func Workflow(ctx workflow.Context) error {
	var ao workflow.ActivityOptions

	// Bare //nolint suppresses every linter, including this one.
	workflow.WithActivityOptions(ctx, ao) //nolint

	// Names the plugin explicitly, with a trailing explanation.
	workflow.WithActivityOptions(ctx, ao) //nolint:temporalcheck // discarded on purpose

	// A directive on any line the call spans suppresses it, even across lines.
	workflow.WithActivityOptions( //nolint:temporalcheck
		ctx,
		ao,
	)

	// Only other linters named: the diagnostic still fires.
	workflow.WithActivityOptions(ctx, ao) //nolint:gocritic // want `WithActivityOptions: the returned context is discarded, so the options never apply; assign it back with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-discard\)`

	// Names the analyzer rather than the plugin: golangci-lint knows this linter
	// only as "temporalcheck", so this does not suppress.
	workflow.WithActivityOptions(ctx, ao) //nolint:optionsdiscard // want `WithActivityOptions: the returned context is discarded, so the options never apply; assign it back with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-discard\)`

	return nil
}
