// Package nolint exercises per-literal //nolint suppression of the worker-panic
// diagnostic. A directive that names the plugin (temporalcheck), names all, or is
// bare must suppress; a directive naming only other linters, or the analyzer name,
// must not.
package nolint

import "go.temporal.io/sdk/worker"

func Boot() {
	// Bare //nolint suppresses every linter, including this one.
	_ = worker.Options{MaxConcurrentWorkflowTaskPollers: 1} //nolint

	// Names the plugin explicitly, with a trailing explanation.
	_ = worker.Options{MaxConcurrentWorkflowTaskExecutionSize: 1} //nolint:temporalcheck // single-poller worker is intentional here

	// A directive on the line the offending value sits on suppresses it, even when
	// the literal spans several lines.
	_ = worker.Options{
		MaxConcurrentWorkflowTaskPollers: 1, //nolint:temporalcheck
	}

	// Only other linters named: the diagnostic still fires.
	_ = worker.Options{MaxConcurrentWorkflowTaskPollers: 1} //nolint:gocritic // want `worker.Options: MaxConcurrentWorkflowTaskPollers must not be 1 — the worker panics on start; use 0 for the default or a value >= 2 \(worker-panic\)`

	// Names the analyzer rather than the plugin: golangci-lint knows this linter
	// only as "temporalcheck", so this does not suppress.
	_ = worker.Options{MaxConcurrentWorkflowTaskPollers: 1} //nolint:workeroptions // want `worker.Options: MaxConcurrentWorkflowTaskPollers must not be 1 — the worker panics on start; use 0 for the default or a value >= 2 \(worker-panic\)`
}
