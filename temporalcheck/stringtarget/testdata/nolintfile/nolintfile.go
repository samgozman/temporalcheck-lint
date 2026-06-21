//nolint:temporalcheck // file-wide suppression: this file names targets by string on purpose

// Package nolintfile carries a //nolint directive before the package clause, so
// every diagnostic in the file is suppressed.
package nolintfile

import "go.temporal.io/sdk/workflow"

func Workflow(ctx workflow.Context) error {
	// Would be flagged, but the file-wide directive suppresses it.
	_ = workflow.ExecuteActivity(ctx, "Greet", "world")

	return nil
}
