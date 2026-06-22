// Package disabled holds a worker-panic-shape literal with no expectation marker:
// run with Settings.Disabled true, the analyzer must stay silent, so analysistest
// sees no diagnostics here.
package disabled

import "go.temporal.io/sdk/worker"

func Boot() {
	// Would be flagged if the analyzer were enabled.
	_ = worker.Options{MaxConcurrentWorkflowTaskPollers: 1}
}
