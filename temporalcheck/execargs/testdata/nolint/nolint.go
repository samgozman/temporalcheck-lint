// Package nolint exercises per-call //nolint suppression. A call carrying a
// directive that names the plugin (temporalcheck), names all, or is bare must
// report nothing; a directive naming only other linters must not suppress.
package nolint

import "go.temporal.io/sdk/workflow"

type Activities struct{}

func (a *Activities) Greet(name string) error { return nil }

func Workflow(ctx workflow.Context) error {
	var a *Activities

	// Bare //nolint suppresses every linter, including this one.
	_ = workflow.ExecuteActivity(ctx, a.Greet) //nolint

	// Names the plugin explicitly.
	_ = workflow.ExecuteActivity(ctx, a.Greet) //nolint:temporalcheck

	// In a list, with a trailing explanation.
	_ = workflow.ExecuteActivity(ctx, a.Greet, "x", "y") //nolint:gocritic,temporalcheck // intentional

	// //nolint:all suppresses too.
	_ = workflow.ExecuteActivity(ctx, a.Greet, "x", "y") //nolint:all

	// A directive on any line the call spans suppresses it, even across lines.
	_ = workflow.ExecuteActivity( //nolint:temporalcheck
		ctx,
		a.Greet,
	)

	// Only other linters named: the execargs diagnostic still fires.
	_ = workflow.ExecuteActivity(ctx, a.Greet) //nolint:gocritic // want `ExecuteActivity: activity "Greet" expects 1 argument, got 0 \(arity\)`

	// Names the analyzer rather than the plugin: golangci-lint knows this linter
	// only as "temporalcheck", so this does not suppress.
	_ = workflow.ExecuteActivity(ctx, a.Greet) //nolint:execargs // want `ExecuteActivity: activity "Greet" expects 1 argument, got 0 \(arity\)`

	return nil
}
