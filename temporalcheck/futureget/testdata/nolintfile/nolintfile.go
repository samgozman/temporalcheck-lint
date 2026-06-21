//nolint:temporalcheck // file-wide suppression: this file discards .Get errors on purpose

// Package nolintfile carries a //nolint directive before the package clause, so
// every diagnostic in the file is suppressed.
package nolintfile

import "go.temporal.io/sdk/workflow"

func Workflow(ctx workflow.Context) error {
	var result string

	// Would be flagged, but the file-wide directive suppresses it.
	f := workflow.ExecuteActivity(ctx, "Activity")
	f.Get(ctx, &result)

	return nil
}
