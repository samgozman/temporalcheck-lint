// Package flagged exercises the literals the analyzer reports: an
// ActivityOptions or LocalActivityOptions composite literal that sets fields but
// neither required timeout (StartToCloseTimeout / ScheduleToCloseTimeout),
// including a pointer literal, an aliased import, and an elided element literal.
package flagged

import (
	"time"

	"go.temporal.io/sdk/workflow"

	wf "go.temporal.io/sdk/workflow"
)

func Workflow(ctx workflow.Context) {
	// The idiom: options built, fed to WithActivityOptions, but no timeout set --
	// the activity is rejected at run time.
	ao := workflow.ActivityOptions{TaskQueue: "greetings"} // want `ActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time \(required-timeout\)`
	ctx = workflow.WithActivityOptions(ctx, ao)
	_ = ctx

	// Only a non-timeout field (RetryPolicy) set: still missing a required timeout.
	_ = workflow.ActivityOptions{RetryPolicy: &workflow.RetryPolicy{MaximumAttempts: 3}} // want `ActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time \(required-timeout\)`

	// A timeout field that is not one of the two required ones does not satisfy it.
	_ = workflow.ActivityOptions{ScheduleToStartTimeout: time.Second} // want `ActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time \(required-timeout\)`

	// LocalActivityOptions is checked the same way.
	_ = workflow.LocalActivityOptions{RetryPolicy: &workflow.RetryPolicy{}} // want `LocalActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time \(required-timeout\)`

	// Pointer literal: the inner ActivityOptions literal is still inspected.
	_ = &workflow.ActivityOptions{TaskQueue: "q"} // want `ActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time \(required-timeout\)`

	// Aliased import resolves to the same workflow type, so it is flagged too.
	_ = wf.ActivityOptions{TaskQueue: "q"} // want `ActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time \(required-timeout\)`

	// Nested in a slice with an elided element type: the inner literal is inspected.
	_ = []workflow.ActivityOptions{
		{TaskQueue: "q"}, // want `ActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time \(required-timeout\)`
	}

	// Deeply nested inside ExecuteActivity(WithActivityOptions(...)).Get(...) -- the
	// shape real workflows use. The whole-tree walk still reaches the literal, even
	// when its only field is a RetryPolicy.
	_ = executeActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{ // want `ActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time \(required-timeout\)`
			RetryPolicy: &workflow.RetryPolicy{MaximumAttempts: 5},
		}),
		nil,
	).Get(ctx, nil)
}

// future and executeActivity are local stand-ins for the workflow.Future and
// workflow.ExecuteActivity shapes, just enough to reproduce the real call nesting
// in a fixture (the stub's workflow package doesn't ship Execute*).
type future struct{}

func (future) Get(ctx workflow.Context, out any) error { return nil }

func executeActivity(ctx workflow.Context, target any) future { return future{} }
