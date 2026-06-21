// Package nolint exercises per-call //nolint suppression. A call carrying a
// directive that names the plugin (temporalcheck), names all, or is bare must
// report nothing; a directive naming only other linters, or the analyzer name,
// must not suppress.
package nolint

import "go.temporal.io/sdk/workflow"

func Workflow(ctx workflow.Context) error {
	// Bare //nolint suppresses every linter, including this one.
	_ = workflow.ExecuteActivity(ctx, "Greet", "world") //nolint

	// Names the plugin explicitly, with a trailing explanation.
	_ = workflow.ExecuteActivity(ctx, "Greet", "world") //nolint:temporalcheck // by name on purpose

	// A directive on any line the call spans suppresses it, even across lines.
	_ = workflow.ExecuteActivity( //nolint:temporalcheck
		ctx,
		"Greet",
		"world",
	)

	// Only other linters named: the diagnostic still fires.
	_ = workflow.ExecuteActivity(ctx, "Greet", "world") //nolint:gocritic // want `ExecuteActivity: target "Greet" is named by string; pass the function reference instead so its arguments can be checked statically \(string-target\)`

	// Names the analyzer rather than the plugin: golangci-lint knows this linter
	// only as "temporalcheck", so this does not suppress.
	_ = workflow.ExecuteActivity(ctx, "Greet", "world") //nolint:stringtarget // want `ExecuteActivity: target "Greet" is named by string; pass the function reference instead so its arguments can be checked statically \(string-target\)`

	return nil
}
