// Package internal is a stand-in for go.temporal.io/sdk/internal. The real SDK
// declares the core types here and re-publishes them from the workflow package
// as aliases (type Context = internal.Context), so the fixtures mirror that
// shape to exercise the analyzer against the SDK's real type identities.
package internal

type Context interface{ isWorkflowContext() }
