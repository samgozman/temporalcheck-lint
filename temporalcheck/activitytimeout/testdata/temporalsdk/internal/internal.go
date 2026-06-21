// Package internal is a stand-in for go.temporal.io/sdk/internal. The real SDK
// declares the core types here and re-publishes them from the workflow package
// as aliases (type Context = internal.Context, type ActivityOptions =
// internal.ActivityOptions), so the fixtures mirror that shape to exercise the
// analyzer against the SDK's real type identities -- the option literals resolve
// to internal types, not workflow ones.
package internal

import "time"

type Context interface{ isWorkflowContext() }

// RetryPolicy is a stand-in for the SDK's retry policy; fixtures only need a
// non-timeout field to set, to prove a literal with other fields but no required
// timeout is still flagged.
type RetryPolicy struct{ MaximumAttempts int32 }

// ActivityOptions mirrors the SDK struct. Field order matters: fixtures construct
// a positional literal to prove the analyzer skips that shape, so keep the order
// stable.
type ActivityOptions struct {
	TaskQueue              string
	ScheduleToCloseTimeout time.Duration
	ScheduleToStartTimeout time.Duration
	StartToCloseTimeout    time.Duration
	HeartbeatTimeout       time.Duration
	RetryPolicy            *RetryPolicy
}

// LocalActivityOptions mirrors the SDK struct; like ActivityOptions it requires
// one of the two timeouts.
type LocalActivityOptions struct {
	ScheduleToCloseTimeout time.Duration
	StartToCloseTimeout    time.Duration
	RetryPolicy            *RetryPolicy
}
