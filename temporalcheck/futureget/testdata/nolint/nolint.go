// Package nolint exercises per-call //nolint suppression. A call carrying a
// directive that names the plugin (temporalcheck), names all, or is bare must
// report nothing; a directive naming only other linters, or the analyzer name,
// must not suppress.
package nolint

import "go.temporal.io/sdk/workflow"

func Workflow(ctx workflow.Context) error {
	var result string

	f := workflow.ExecuteActivity(ctx, "Activity")

	// Bare //nolint suppresses every linter, including this one.
	f.Get(ctx, &result) //nolint

	// Names the plugin explicitly, with a trailing explanation.
	f.Get(ctx, &result) //nolint:temporalcheck // discarded on purpose

	// A directive on any line the call spans suppresses it, even across lines.
	f.Get( //nolint:temporalcheck
		ctx,
		&result,
	)

	// Only other linters named: the diagnostic still fires.
	f.Get(ctx, &result) //nolint:gocritic // want `Get: the returned error from Future\.Get is discarded; check it or assign it to a variable you inspect \(future-get\)`

	// Names the analyzer rather than the plugin: golangci-lint knows this linter
	// only as "temporalcheck", so this does not suppress.
	f.Get(ctx, &result) //nolint:futureget // want `Get: the returned error from Future\.Get is discarded; check it or assign it to a variable you inspect \(future-get\)`

	return nil
}
