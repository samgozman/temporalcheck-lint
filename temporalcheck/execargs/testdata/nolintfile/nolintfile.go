//nolint:temporalcheck // whole-file suppression: a directive before the package clause

// Package nolintfile exercises file-level //nolint suppression: a directive
// placed before the package clause silences every execargs diagnostic in the
// file, so the arity violation below reports nothing.
package nolintfile

import "go.temporal.io/sdk/workflow"

type Activities struct{}

func (a *Activities) Greet(name string) error { return nil }

func Workflow(ctx workflow.Context) error {
	var a *Activities

	// Would be an arity error, but the whole file is suppressed.
	_ = workflow.ExecuteActivity(ctx, a.Greet)

	return nil
}
