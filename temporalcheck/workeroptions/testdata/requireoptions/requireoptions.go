// Package requireoptions exercises the opt-in require-options rule: a
// worker.New(c, q, worker.Options{...}) whose options literal sets none of the
// five concurrency-limit fields runs on the SDK defaults that can overload a
// self-hosted cluster. The rule fires only on the literal passed to worker.New,
// and is satisfied by any one of the five limits regardless of value. Run with
// RequireOptions enabled (worker-panic stays on too, so none of these set 1).
package requireoptions

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func Boot(c client.Client) {
	// Empty options: no concurrency limits set, runs on SDK defaults.
	_ = worker.New(c, "q", worker.Options{}) // want `worker.New: worker.Options sets no concurrency limits, so the worker runs on the SDK defaults \(1k executions, 100k/s\) that can overload a self-hosted cluster; set MaxConcurrent\* limits \(require-options\)`

	// Sets only a non-concurrency field: still no limits.
	_ = worker.New(c, "q", worker.Options{Identity: "w1"}) // want `worker.New: worker.Options sets no concurrency limits, so the worker runs on the SDK defaults \(1k executions, 100k/s\) that can overload a self-hosted cluster; set MaxConcurrent\* limits \(require-options\)`

	// Sets one of the five limits: satisfied, not flagged. Value is irrelevant.
	_ = worker.New(c, "q", worker.Options{MaxConcurrentActivityExecutionSize: 100})
	_ = worker.New(c, "q", worker.Options{MaxConcurrentLocalActivityExecutionSize: 50})

	// Options is a variable rather than a literal: skipped (its fields aren't
	// visible at the call site).
	opts := worker.Options{}
	_ = worker.New(c, "q", opts)

	// A bare worker.Options{} not passed to worker.New: require-options targets the
	// New call site only, so this is left alone.
	_ = worker.Options{}

	// A //nolint on the call line suppresses the require-options diagnostic.
	_ = worker.New(c, "q", worker.Options{}) //nolint:temporalcheck // SDK defaults are fine for this worker

	// A selector call that is not worker.New, and a non-selector (builtin) call:
	// both ignored by the require-options dispatch.
	c.Close()
	_ = len("x")
}
