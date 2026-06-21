// Package testsuite is a minimal stand-in for go.temporal.io/sdk/testsuite. The
// real SDK declares TestWorkflowEnvironment and MockCallWrapper in the internal
// package and re-publishes them here as aliases, so the fixtures mirror that
// shape to exercise the analyzer against the SDK's real type identities.
package testsuite

import "go.temporal.io/sdk/internal"

type TestWorkflowEnvironment = internal.TestWorkflowEnvironment

type MockCallWrapper = internal.MockCallWrapper
