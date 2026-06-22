// Package nolint exercises per-call //nolint suppression. A call carrying a
// directive that names the plugin (temporalcheck), names all, or is bare must
// report nothing; a directive naming only other linters, or the analyzer name,
// must not suppress.
package nolint

import "go.temporal.io/sdk/workflow"

func greet(ctx workflow.Context) error { return nil }

func Workflow(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions

	// Bare //nolint suppresses every linter, including this one.
	ctx = workflow.WithChildOptions(ctx, cwo)
	workflow.ExecuteActivity(ctx, greet) //nolint

	// Names the plugin explicitly, with a trailing explanation.
	workflow.ExecuteActivity(ctx, greet) //nolint:temporalcheck // crossed on purpose

	// A directive on any line the call spans suppresses it, even across lines.
	workflow.ExecuteActivity( //nolint:temporalcheck
		ctx,
		greet,
	)

	// Only other linters named: the diagnostic still fires.
	workflow.ExecuteActivity(ctx, greet) //nolint:gocritic // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`

	// Names the analyzer rather than the plugin: golangci-lint knows this linter
	// only as "temporalcheck", so this does not suppress.
	workflow.ExecuteActivity(ctx, greet) //nolint:optionscontext // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`

	return nil
}
