//nolint:temporalcheck // file-wide suppression: this file discards With*Options results on purpose

// Package nolintfile carries a //nolint directive before the package clause, so
// every diagnostic in the file is suppressed.
package nolintfile

import "go.temporal.io/sdk/workflow"

func Workflow(ctx workflow.Context) error {
	var ao workflow.ActivityOptions

	// Would be flagged, but the file-wide directive suppresses it.
	workflow.WithActivityOptions(ctx, ao)

	return nil
}
