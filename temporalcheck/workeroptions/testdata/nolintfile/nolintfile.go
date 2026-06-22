//nolint:temporalcheck // file-wide suppression: a single-poller worker is intentional in this fixture

// Package nolintfile carries a //nolint directive before the package clause, so
// every diagnostic in the file is suppressed.
package nolintfile

import "go.temporal.io/sdk/worker"

func Boot() {
	// Would be flagged, but the file-wide directive suppresses it.
	_ = worker.Options{MaxConcurrentWorkflowTaskPollers: 1}
}
