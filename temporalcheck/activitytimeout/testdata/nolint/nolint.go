// Package nolint exercises per-literal //nolint suppression. A literal carrying a
// directive that names the plugin (temporalcheck), names all, or is bare must
// report nothing; a directive naming only other linters, or the analyzer name,
// must not suppress.
package nolint

import "go.temporal.io/sdk/workflow"

func Workflow() {
	// Bare //nolint suppresses every linter, including this one.
	_ = workflow.ActivityOptions{TaskQueue: "q"} //nolint

	// Names the plugin explicitly, with a trailing explanation.
	_ = workflow.ActivityOptions{TaskQueue: "q"} //nolint:temporalcheck // timeout set elsewhere

	// A directive on any line the literal spans suppresses it, even across lines.
	_ = workflow.ActivityOptions{ //nolint:temporalcheck
		TaskQueue: "q",
	}

	// Only other linters named: the diagnostic still fires.
	_ = workflow.ActivityOptions{TaskQueue: "q"} //nolint:gocritic // want `ActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time \(required-timeout\)`

	// Names the analyzer rather than the plugin: golangci-lint knows this linter
	// only as "temporalcheck", so this does not suppress.
	_ = workflow.ActivityOptions{TaskQueue: "q"} //nolint:activitytimeout // want `ActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time \(required-timeout\)`
}
