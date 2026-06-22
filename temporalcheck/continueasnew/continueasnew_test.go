package continueasnew_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/continueasnew"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths.

// TestContinueAsNew: every discarded NewContinueAsNewError result is reported --
// a bare call statement and an explicit blank assignment, including an aliased
// import of the workflow package.
func TestContinueAsNew(t *testing.T) {
	a := continueasnew.NewAnalyzer(continueasnew.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "continueasnewfixtures/flagged")
}

// TestContinueAsNew_Benign: a result returned directly, a result kept in a named
// variable and returned (possibly conditionally), the same-named function from
// another package, and unrelated bare calls produce no diagnostics.
func TestContinueAsNew_Benign(t *testing.T) {
	a := continueasnew.NewAnalyzer(continueasnew.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "continueasnewfixtures/benign")
}

// TestContinueAsNew_Disabled: with Disabled set, the analyzer reports nothing
// even on a fixture that discards a NewContinueAsNewError result.
func TestContinueAsNew_Disabled(t *testing.T) {
	a := continueasnew.NewAnalyzer(continueasnew.Settings{Disabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "continueasnewfixtures/disabled")
}

// TestContinueAsNew_Nolint: a //nolint directive naming temporalcheck (or all, or
// bare) on the call's line suppresses its diagnostic; a directive naming only
// other linters, or the analyzer name, does not.
func TestContinueAsNew_Nolint(t *testing.T) {
	a := continueasnew.NewAnalyzer(continueasnew.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "continueasnewfixtures/nolint")
}

// TestContinueAsNew_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestContinueAsNew_NolintFile(t *testing.T) {
	a := continueasnew.NewAnalyzer(continueasnew.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "continueasnewfixtures/nolintfile")
}
