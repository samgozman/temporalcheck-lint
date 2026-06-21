package stringtarget_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/stringtarget"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths.

// TestStringTarget: with the check enabled, every string-named target is
// reported -- a literal (by name), a string variable, and a named string type --
// across all three Execute* entry points.
func TestStringTarget(t *testing.T) {
	a := stringtarget.NewAnalyzer(stringtarget.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "stringtargetfixtures/flagged")
}

// TestStringTarget_Benign: call shapes the analyzer must ignore even when
// enabled -- a non-selector call, a selector into another package, a
// function-reference target, and a non-entry-point workflow function -- produce
// no diagnostics.
func TestStringTarget_Benign(t *testing.T) {
	a := stringtarget.NewAnalyzer(stringtarget.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "stringtargetfixtures/benign")
}

// TestStringTarget_Disabled: off by default, the analyzer reports nothing even
// on a fixture that names its target by string.
func TestStringTarget_Disabled(t *testing.T) {
	a := stringtarget.NewAnalyzer(stringtarget.Settings{Enabled: false})
	analysistest.Run(t, analysistest.TestData(), a, "stringtargetfixtures/disabled")
}

// TestStringTarget_DisabledGatesStrictTests: Enabled is the master switch, so
// with Enabled off the analyzer stays silent even when StrictTests is on -- the
// disabled fixture's string-named mock setup, which StrictTests would otherwise
// flag, must report nothing.
func TestStringTarget_DisabledGatesStrictTests(t *testing.T) {
	a := stringtarget.NewAnalyzer(stringtarget.Settings{Enabled: false, StrictTests: true})
	analysistest.Run(t, analysistest.TestData(), a, "stringtargetfixtures/disabled")
}

// TestStringTarget_StrictTests: with Enabled on, StrictTests adds the test-mock
// check on top of the production one, so a string-named OnActivity/OnWorkflow
// mock setup is reported, while a function-reference target is left alone.
func TestStringTarget_StrictTests(t *testing.T) {
	a := stringtarget.NewAnalyzer(stringtarget.Settings{Enabled: true, StrictTests: true})
	analysistest.Run(t, analysistest.TestData(), a, "stringtargetfixtures/tests")
}

// TestStringTarget_StrictTestsDisabled: with StrictTests off, a string-named
// mock setup is silent even when Enabled is on -- the production check does not
// bleed into test mocks.
func TestStringTarget_StrictTestsDisabled(t *testing.T) {
	a := stringtarget.NewAnalyzer(stringtarget.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "stringtargetfixtures/testsoff")
}

// TestStringTarget_Nolint: a //nolint directive naming temporalcheck (or all, or
// bare) on the call's line suppresses its diagnostic; a directive naming only
// other linters, or the analyzer name stringtarget, does not.
func TestStringTarget_Nolint(t *testing.T) {
	a := stringtarget.NewAnalyzer(stringtarget.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "stringtargetfixtures/nolint")
}

// TestStringTarget_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestStringTarget_NolintFile(t *testing.T) {
	a := stringtarget.NewAnalyzer(stringtarget.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "stringtargetfixtures/nolintfile")
}
