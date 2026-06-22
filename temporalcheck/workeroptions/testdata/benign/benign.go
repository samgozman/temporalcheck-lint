// Package benign exercises the literals the analyzer must leave alone under
// default settings (worker-panic on, require-options off): a workflow-task field
// set to 0 or a value >= 2, a non-constant value, the activity counterparts set to
// 1 (no such restriction), an empty literal passed to worker.New (require-options
// is off), a positional literal, and non-worker composite literals.
package benign

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func Boot(c client.Client) {
	// 0 means "use the default" -- never flagged.
	_ = worker.Options{MaxConcurrentWorkflowTaskPollers: 0}

	// A value >= 2 is fine.
	_ = worker.Options{MaxConcurrentWorkflowTaskExecutionSize: 2}

	// A non-constant value can't be evaluated statically, so it is skipped.
	_ = worker.Options{MaxConcurrentWorkflowTaskPollers: pollers(c)}

	// The activity counterparts carry no 1-is-illegal restriction.
	_ = worker.Options{MaxConcurrentActivityTaskPollers: 1}
	_ = worker.Options{MaxConcurrentActivityExecutionSize: 1}
	_ = worker.Options{MaxConcurrentLocalActivityExecutionSize: 1}

	// require-options is off by default: an empty literal passed to New is fine.
	_ = worker.New(c, "q", worker.Options{})

	// A positional literal can't be mapped to fields without the struct layout, so
	// it is skipped (go vet already flags unkeyed imported-struct literals).
	_ = worker.Options{1, 1, 1, 1, 1, 0, ""}

	// Non-worker composite literals are out of scope.
	_ = []int{1, 2, 3}
}

func pollers(c client.Client) int { return 2 }
