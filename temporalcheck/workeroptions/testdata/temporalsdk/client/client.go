// Package client is a minimal stand-in for go.temporal.io/sdk/client. worker.New
// takes a client.Client as its first argument, so the stub declares just enough
// of the interface for the fixtures to construct a worker.New call.
package client

type Client interface{ Close() }
