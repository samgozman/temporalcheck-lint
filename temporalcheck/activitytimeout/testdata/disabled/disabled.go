// Package disabled holds a flagged-shape literal with no expectation marker: run
// with Settings.Disabled true, the analyzer must stay silent, so analysistest
// sees no diagnostics here.
package disabled

import "go.temporal.io/sdk/workflow"

func Workflow() {
	// Would be flagged if the analyzer were enabled.
	_ = workflow.ActivityOptions{TaskQueue: "q"}
}
