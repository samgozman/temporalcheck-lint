// Package workerpanic exercises the literals the worker-panic rule reports: a
// worker.Options composite literal that sets MaxConcurrentWorkflowTaskExecutionSize
// or MaxConcurrentWorkflowTaskPollers to a constant 1, which panics the worker on
// start (pollers alternate between sticky and non-sticky queues, so 1 is illegal).
// The diagnostic anchors on the offending value expression, not the literal.
package workerpanic

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const one = 1

func Boot(c client.Client) {
	// A literal 1 in either workflow-task field is a guaranteed boot panic.
	_ = worker.Options{MaxConcurrentWorkflowTaskPollers: 1} // want `worker.Options: MaxConcurrentWorkflowTaskPollers must not be 1 — the worker panics on start; use 0 for the default or a value >= 2 \(worker-panic\)`

	_ = worker.Options{MaxConcurrentWorkflowTaskExecutionSize: 1} // want `worker.Options: MaxConcurrentWorkflowTaskExecutionSize must not be 1 — the worker panics on start; use 0 for the default or a value >= 2 \(worker-panic\)`

	// Assigned to a variable, then passed to New: the literal is still inspected.
	opts := worker.Options{MaxConcurrentWorkflowTaskPollers: 1} // want `worker.Options: MaxConcurrentWorkflowTaskPollers must not be 1 — the worker panics on start; use 0 for the default or a value >= 2 \(worker-panic\)`
	_ = worker.New(c, "q", opts)

	// Inline in worker.New: worker-panic fires regardless of require-options.
	_ = worker.New(c, "q", worker.Options{MaxConcurrentWorkflowTaskPollers: 1}) // want `worker.Options: MaxConcurrentWorkflowTaskPollers must not be 1 — the worker panics on start; use 0 for the default or a value >= 2 \(worker-panic\)`

	// A named constant equal to 1 is still a constant 1.
	_ = worker.Options{MaxConcurrentWorkflowTaskPollers: one} // want `worker.Options: MaxConcurrentWorkflowTaskPollers must not be 1 — the worker panics on start; use 0 for the default or a value >= 2 \(worker-panic\)`

	// Pointer literal: the inner worker.Options literal is still inspected.
	_ = &worker.Options{MaxConcurrentWorkflowTaskExecutionSize: 1} // want `worker.Options: MaxConcurrentWorkflowTaskExecutionSize must not be 1 — the worker panics on start; use 0 for the default or a value >= 2 \(worker-panic\)`

	// Both workflow-task fields set to 1: each is reported on its own line.
	_ = worker.Options{
		MaxConcurrentWorkflowTaskExecutionSize: 1, // want `worker.Options: MaxConcurrentWorkflowTaskExecutionSize must not be 1 — the worker panics on start; use 0 for the default or a value >= 2 \(worker-panic\)`
		MaxConcurrentWorkflowTaskPollers:       1, // want `worker.Options: MaxConcurrentWorkflowTaskPollers must not be 1 — the worker panics on start; use 0 for the default or a value >= 2 \(worker-panic\)`
	}
}
