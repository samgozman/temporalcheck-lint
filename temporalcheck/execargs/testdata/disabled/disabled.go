// Package disabled holds a genuine arity violation with no expectation marker:
// run with Settings.Disabled the analyzer must stay silent, so analysistest
// sees no diagnostics here.
package disabled

import "go.temporal.io/sdk/workflow"

type Activities struct{}

func (a *Activities) Greet(name string) error { return nil }

func Workflow(ctx workflow.Context) error {
	var a *Activities

	// Too few arguments -- would be an arity error if the analyzer were enabled.
	_ = workflow.ExecuteActivity(ctx, a.Greet)

	return nil
}
