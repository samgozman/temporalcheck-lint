package optionsdiscard_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/optionsdiscard"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths.

// TestOptionsDiscard: every discarded With*Options result is reported -- a bare
// call statement and an explicit blank assignment, across all three entry
// points, including through an aliased import.
func TestOptionsDiscard(t *testing.T) {
	a := optionsdiscard.NewAnalyzer(optionsdiscard.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "optionsdiscardfixtures/flagged")
}

// TestOptionsDiscard_Benign: the shapes that keep the result -- `ctx = ...`,
// `:=` to a fresh variable, the result passed on -- plus calls the analyzer does
// not own, produce no diagnostics.
func TestOptionsDiscard_Benign(t *testing.T) {
	a := optionsdiscard.NewAnalyzer(optionsdiscard.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "optionsdiscardfixtures/benign")
}

// TestOptionsDiscard_Disabled: with Disabled set, the analyzer reports nothing
// even on a fixture that discards a With*Options result.
func TestOptionsDiscard_Disabled(t *testing.T) {
	a := optionsdiscard.NewAnalyzer(optionsdiscard.Settings{Disabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "optionsdiscardfixtures/disabled")
}

// TestOptionsDiscard_Nolint: a //nolint directive naming temporalcheck (or all,
// or bare) on the call's line suppresses its diagnostic; a directive naming only
// other linters, or the analyzer name, does not.
func TestOptionsDiscard_Nolint(t *testing.T) {
	a := optionsdiscard.NewAnalyzer(optionsdiscard.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "optionsdiscardfixtures/nolint")
}

// TestOptionsDiscard_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestOptionsDiscard_NolintFile(t *testing.T) {
	a := optionsdiscard.NewAnalyzer(optionsdiscard.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "optionsdiscardfixtures/nolintfile")
}
