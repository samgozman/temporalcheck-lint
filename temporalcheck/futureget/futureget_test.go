package futureget_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/futureget"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths.

// TestFutureGet: every discarded .Get error is reported -- a bare call statement
// and an explicit blank assignment, across workflow.Future,
// workflow.ChildWorkflowFuture and converter.EncodedValue, including a chained
// receiver and an aliased import.
func TestFutureGet(t *testing.T) {
	a := futureget.NewAnalyzer(futureget.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "futuregetfixtures/flagged")
}

// TestFutureGet_Benign: a checked error, an error kept in a variable, a non-Get
// method, a Get on a non-Temporal type, a Get on a type that merely embeds
// Future, and a selector into another package produce no diagnostics.
func TestFutureGet_Benign(t *testing.T) {
	a := futureget.NewAnalyzer(futureget.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "futuregetfixtures/benign")
}

// TestFutureGet_Disabled: with Disabled set, the analyzer reports nothing even on
// a fixture that discards a .Get error.
func TestFutureGet_Disabled(t *testing.T) {
	a := futureget.NewAnalyzer(futureget.Settings{Disabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "futuregetfixtures/disabled")
}

// TestFutureGet_Nolint: a //nolint directive naming temporalcheck (or all, or
// bare) on the call's line suppresses its diagnostic; a directive naming only
// other linters, or the analyzer name, does not.
func TestFutureGet_Nolint(t *testing.T) {
	a := futureget.NewAnalyzer(futureget.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "futuregetfixtures/nolint")
}

// TestFutureGet_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestFutureGet_NolintFile(t *testing.T) {
	a := futureget.NewAnalyzer(futureget.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "futuregetfixtures/nolintfile")
}
