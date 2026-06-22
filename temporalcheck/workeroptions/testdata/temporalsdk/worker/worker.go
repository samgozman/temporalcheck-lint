// Package worker is a minimal stand-in for go.temporal.io/sdk/worker. Like the
// real SDK it re-exports the options struct as an alias to the internal
// definition (type Options = internal.WorkerOptions), so the analyzer must
// resolve that alias -- the literal's type surfaces as internal.WorkerOptions,
// not a type declared here. New takes the options as its third argument, which is
// what the require-options check inspects.
package worker

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/internal"
)

// Options is an alias to the internal struct, exactly as the real SDK
// re-exports it. The analyzer resolves the alias to internal.WorkerOptions.
type Options = internal.WorkerOptions

// Worker is a minimal stand-in for the SDK's worker.Worker interface.
type Worker interface {
	Run(stopCh <-chan interface{}) error
}

// New mirrors the real worker.New(client, taskQueue, options) shape; the
// options literal is the third argument the require-options check inspects.
func New(c client.Client, taskQueue string, options Options) Worker { return nil }
