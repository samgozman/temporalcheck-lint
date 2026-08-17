package searchattribute_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/searchattribute"
)

// attrs is the field-alias -> attribute-name map the fixtures are written against:
// UserID, ClientID and XUserID all require the "user_id" search attribute.
var attrs = map[string]string{
	"user_id":   "user_id",
	"client_id": "user_id",
	"x-user-id": "user_id",
}

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths. Every test enables the
// analyzer, since it reports nothing by default.

// TestMissing: a workflow whose params struct carries a configured field but whose
// body never upserts the mapped attribute -- including one that upserts a
// different attribute, and one taking the params by pointer -- is flagged at its
// function name.
func TestMissing(t *testing.T) {
	a := searchattribute.NewAnalyzer(searchattribute.Settings{Enabled: true, Attributes: attrs})
	analysistest.Run(t, analysistest.TestData(), a, "searchattributefixtures/missing")
}

// TestSatisfied: a workflow that upserts the mapped attribute -- via the typed or
// the legacy API, at the top level or nested -- with the attribute name as a
// string literal is not flagged.
func TestSatisfied(t *testing.T) {
	a := searchattribute.NewAnalyzer(searchattribute.Settings{Enabled: true, Attributes: attrs})
	analysistest.Run(t, analysistest.TestData(), a, "searchattributefixtures/satisfied")
}

// TestOpaque: a workflow that upserts through a source we cannot read statically
// (a prebuilt map, a spread slice, or a key held in a variable) is left alone
// rather than risk a false positive.
func TestOpaque(t *testing.T) {
	a := searchattribute.NewAnalyzer(searchattribute.Settings{Enabled: true, Attributes: attrs})
	analysistest.Run(t, analysistest.TestData(), a, "searchattributefixtures/opaque")
}

// TestOtherCalls: calls in a workflow body that are not upserts -- a bare local
// function, a conversion, a call through a function-typed field -- are ignored, so
// a workflow making them but never upserting is still flagged.
func TestOtherCalls(t *testing.T) {
	a := searchattribute.NewAnalyzer(searchattribute.Settings{Enabled: true, Attributes: attrs})
	analysistest.Run(t, analysistest.TestData(), a, "searchattributefixtures/othercalls")
}

// TestHelpers: unexported functions whose first parameter is workflow.Context are
// workflow helpers, not registered entrypoints, so none are flagged even when they
// carry a configured field.
func TestHelpers(t *testing.T) {
	a := searchattribute.NewAnalyzer(searchattribute.Settings{Enabled: true, Attributes: attrs})
	analysistest.Run(t, analysistest.TestData(), a, "searchattributefixtures/helpers")
}

// TestTestFiles: an exported workflow in a _test.go file is skipped (test-harness
// workflows aren't production entrypoints), while one in a regular file in the same
// package is still flagged.
func TestTestFiles(t *testing.T) {
	a := searchattribute.NewAnalyzer(searchattribute.Settings{Enabled: true, Attributes: attrs})
	analysistest.Run(t, analysistest.TestData(), a, "searchattributefixtures/testfiles")
}

// TestBenign: a non-workflow function, a workflow with no configured field, one
// with no params, an unexported field, and a scalar param all produce nothing.
func TestBenign(t *testing.T) {
	a := searchattribute.NewAnalyzer(searchattribute.Settings{Enabled: true, Attributes: attrs})
	analysistest.Run(t, analysistest.TestData(), a, "searchattributefixtures/benign")
}

// TestAliases: field-name matching is case- and separator-insensitive, and several
// aliases map to one attribute, so ClientID and XUserID both require user_id.
func TestAliases(t *testing.T) {
	a := searchattribute.NewAnalyzer(searchattribute.Settings{Enabled: true, Attributes: attrs})
	analysistest.Run(t, analysistest.TestData(), a, "searchattributefixtures/aliases")
}

// TestNolint: a //nolint directive naming temporalcheck (or all, or bare) on the
// workflow's signature line suppresses its diagnostic; a directive naming only
// another linter, or the analyzer name, does not.
func TestNolint(t *testing.T) {
	a := searchattribute.NewAnalyzer(searchattribute.Settings{Enabled: true, Attributes: attrs})
	analysistest.Run(t, analysistest.TestData(), a, "searchattributefixtures/nolint")
}

// TestDisabled: with Enabled off (the default), the analyzer reports nothing even
// on an obvious missing upsert.
func TestDisabled(t *testing.T) {
	a := searchattribute.NewAnalyzer(searchattribute.Settings{Attributes: attrs})
	analysistest.Run(t, analysistest.TestData(), a, "searchattributefixtures/disabled")
}
