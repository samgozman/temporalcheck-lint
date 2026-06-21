// Package benign exercises the literals the analyzer must leave alone: a required
// timeout set in either form, an empty literal (populated afterwards), a
// positional literal, a different workflow type, and non-workflow composite
// literals. None produce diagnostics.
package benign

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

type point struct{ x, y int }

func Workflow(ctx workflow.Context) {
	// A required timeout is set -> never flagged, in either accepted form, alone
	// or alongside other fields.
	_ = workflow.ActivityOptions{StartToCloseTimeout: time.Minute}
	_ = workflow.ActivityOptions{ScheduleToCloseTimeout: time.Hour}
	_ = workflow.ActivityOptions{TaskQueue: "q", StartToCloseTimeout: time.Minute}
	_ = workflow.LocalActivityOptions{StartToCloseTimeout: time.Second}

	// Empty literal: typically populated field-by-field afterwards, which this
	// literal-only check can't see, so it is deliberately skipped.
	ao := workflow.ActivityOptions{}
	ao.StartToCloseTimeout = time.Minute
	ctx = workflow.WithActivityOptions(ctx, ao)
	_ = ctx
	_ = &workflow.ActivityOptions{}

	// Positional literal: no field names to test without the struct layout, so it
	// is skipped even though it sets no required timeout (`go vet` already flags
	// unkeyed imported-struct literals).
	_ = workflow.ActivityOptions{"q", 0, 0, 0, 0, nil}

	// A different workflow type that is not an option struct: out of scope.
	_ = workflow.RetryPolicy{MaximumAttempts: 3}

	// Non-workflow composite literals: a local struct, a slice, and a map are all
	// visited but resolve to non-option types, so they are skipped.
	_ = point{1, 2}
	_ = []int{1, 2, 3}
	_ = map[string]int{"a": 1}
}
