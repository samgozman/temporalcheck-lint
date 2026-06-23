package activitytimeout_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/activitytimeout"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths.

// TestActivityTimeout: every ActivityOptions/LocalActivityOptions literal that
// sets fields but no required timeout is reported -- including a pointer literal,
// an aliased import, and an elided element literal.
func TestActivityTimeout(t *testing.T) {
	a := activitytimeout.NewAnalyzer(activitytimeout.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "activitytimeoutfixtures/flagged")
}

// TestActivityTimeout_Benign: literals with a required timeout set, empty and
// positional literals, other workflow types, and non-workflow literals produce no
// diagnostics.
func TestActivityTimeout_Benign(t *testing.T) {
	a := activitytimeout.NewAnalyzer(activitytimeout.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "activitytimeoutfixtures/benign")
}

// TestActivityTimeout_RequireStartToClose: with RequireStartToClose set, a literal
// that sets ScheduleToCloseTimeout but not StartToCloseTimeout is flagged by the
// opt-in sub-rule, while the always-on required-timeout check still fires for a
// literal that sets neither.
func TestActivityTimeout_RequireStartToClose(t *testing.T) {
	a := activitytimeout.NewAnalyzer(activitytimeout.Settings{RequireStartToClose: true})
	analysistest.Run(t, analysistest.TestData(), a, "activitytimeoutfixtures/requirestarttoclose")
}

// TestActivityTimeout_Disabled: with Disabled set, the analyzer reports nothing
// even on a fixture whose literal omits a required timeout.
func TestActivityTimeout_Disabled(t *testing.T) {
	a := activitytimeout.NewAnalyzer(activitytimeout.Settings{Disabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "activitytimeoutfixtures/disabled")
}

// TestActivityTimeout_Nolint: a //nolint directive naming temporalcheck (or all,
// or bare) on the literal's line suppresses its diagnostic; a directive naming
// only other linters, or the analyzer name, does not.
func TestActivityTimeout_Nolint(t *testing.T) {
	a := activitytimeout.NewAnalyzer(activitytimeout.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "activitytimeoutfixtures/nolint")
}

// TestActivityTimeout_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestActivityTimeout_NolintFile(t *testing.T) {
	a := activitytimeout.NewAnalyzer(activitytimeout.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "activitytimeoutfixtures/nolintfile")
}
