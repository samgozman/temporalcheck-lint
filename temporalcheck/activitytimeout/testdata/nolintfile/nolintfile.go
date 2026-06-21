//nolint:temporalcheck // file-wide suppression: these literals omit a required timeout on purpose

// Package nolintfile carries a //nolint directive before the package clause, so
// every diagnostic in the file is suppressed.
package nolintfile

import "go.temporal.io/sdk/workflow"

func Workflow() {
	// Would be flagged, but the file-wide directive suppresses it.
	_ = workflow.ActivityOptions{TaskQueue: "q"}
}
