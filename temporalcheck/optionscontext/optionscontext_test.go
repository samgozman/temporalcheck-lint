package optionscontext_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/optionscontext"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths.

// TestOptionsContext: every seen options/call-kind contradiction is reported --
// the classic child-options/activity-call mistake and its vice versa, the
// local-activity mixup, a sibling wrong-variable swap, a conflict carried into a
// branch, an Execute* on the right-hand side of an assignment, and an aliased
// import.
func TestOptionsContext(t *testing.T) {
	a := optionscontext.NewAnalyzer(optionscontext.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "optionscontextfixtures/flagged")
}

// TestOptionsContext_Benign: correct usage, an unconfigured context, a matching
// helper present in the chain, and every "bail to unknown" case (opaque reset,
// branch-dependent kinds, closure capture, a struct-field context) produce no
// diagnostics.
func TestOptionsContext_Benign(t *testing.T) {
	a := optionscontext.NewAnalyzer(optionscontext.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "optionscontextfixtures/benign")
}

// TestOptionsContext_Disabled: with Disabled set, the analyzer reports nothing
// even on a fixture with a clear contradiction.
func TestOptionsContext_Disabled(t *testing.T) {
	a := optionscontext.NewAnalyzer(optionscontext.Settings{Disabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "optionscontextfixtures/disabled")
}

// TestOptionsContext_Nolint: a //nolint directive naming temporalcheck (or all,
// or bare) on the call's line suppresses its diagnostic; a directive naming only
// other linters, or the analyzer name, does not.
func TestOptionsContext_Nolint(t *testing.T) {
	a := optionscontext.NewAnalyzer(optionscontext.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "optionscontextfixtures/nolint")
}

// TestOptionsContext_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestOptionsContext_NolintFile(t *testing.T) {
	a := optionscontext.NewAnalyzer(optionscontext.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "optionscontextfixtures/nolintfile")
}
