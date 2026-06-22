//nolint:temporalcheck // file-wide suppression: this file discards continue-as-new errors on purpose

// Package nolintfile carries a //nolint directive before the package clause, so
// every diagnostic in the file is suppressed.
package nolintfile

import "go.temporal.io/sdk/workflow"

func Workflow(ctx workflow.Context) error {
	// Would be flagged, but the file-wide directive suppresses it.
	workflow.NewContinueAsNewError(ctx, Workflow)
	return nil
}
