// Package requirestarttoclose exercises the opt-in require-start-to-close
// sub-rule: an ActivityOptions/LocalActivityOptions literal that sets
// ScheduleToCloseTimeout but not StartToCloseTimeout is flagged (the single
// attempt is left unbounded), while a literal that sets StartToCloseTimeout -- in
// either combination -- is left alone. The always-on required-timeout check still
// fires for a literal that sets neither.
package requirestarttoclose

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

func Workflow(ctx workflow.Context) {
	// ScheduleToClose only: accepted at run time, but the attempt is unbounded --
	// the opt-in sub-rule flags it.
	ao := workflow.ActivityOptions{ScheduleToCloseTimeout: time.Hour} // want `ActivityOptions sets ScheduleToCloseTimeout but not StartToCloseTimeout: bound each attempt with StartToCloseTimeout \(require-start-to-close\)`
	ctx = workflow.WithActivityOptions(ctx, ao)
	_ = ctx

	// Alongside other fields, still missing StartToClose: still flagged.
	_ = workflow.ActivityOptions{TaskQueue: "q", ScheduleToCloseTimeout: time.Hour} // want `ActivityOptions sets ScheduleToCloseTimeout but not StartToCloseTimeout: bound each attempt with StartToCloseTimeout \(require-start-to-close\)`

	// LocalActivityOptions is checked the same way.
	_ = workflow.LocalActivityOptions{ScheduleToCloseTimeout: time.Hour} // want `LocalActivityOptions sets ScheduleToCloseTimeout but not StartToCloseTimeout: bound each attempt with StartToCloseTimeout \(require-start-to-close\)`

	// StartToClose present -> the attempt is bounded, so never flagged, whether or
	// not ScheduleToClose is also set.
	_ = workflow.ActivityOptions{StartToCloseTimeout: time.Minute}
	_ = workflow.ActivityOptions{StartToCloseTimeout: time.Minute, ScheduleToCloseTimeout: time.Hour}

	// Neither timeout: the always-on required-timeout check still fires (and the
	// sub-rule does not, since ScheduleToClose is absent).
	_ = workflow.ActivityOptions{TaskQueue: "q"} // want `ActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time \(required-timeout\)`

	// //nolint naming the plugin suppresses the sub-rule diagnostic too.
	_ = workflow.ActivityOptions{ScheduleToCloseTimeout: time.Hour} //nolint:temporalcheck // bounded elsewhere
}
