// Package internal is a stand-in for go.temporal.io/sdk/internal. The real SDK
// declares the worker options struct here as WorkerOptions and re-publishes it
// from the worker package as `type Options = internal.WorkerOptions` -- the same
// alias shape as workflow.Context. The fixtures mirror that so the analyzer is
// exercised against the SDK's real type identity: a worker.Options literal
// resolves to internal.WorkerOptions, not a type declared in the worker package.
package internal

import "time"

// WorkerOptions mirrors the SDK struct. The analyzer reads the five
// concurrency-limit fields (require-options) and the two workflow-task fields
// whose value may not be 1 (worker-panic). The non-concurrency fields exist so a
// fixture can set fields without satisfying require-options. Field order matters:
// a fixture constructs a positional literal to prove the analyzer skips that
// shape, so keep the order stable.
type WorkerOptions struct {
	MaxConcurrentActivityExecutionSize      int
	MaxConcurrentWorkflowTaskExecutionSize  int
	MaxConcurrentActivityTaskPollers        int
	MaxConcurrentWorkflowTaskPollers        int
	MaxConcurrentLocalActivityExecutionSize int
	WorkerStopTimeout                       time.Duration
	Identity                                string
}
