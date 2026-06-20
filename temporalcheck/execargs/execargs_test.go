package execargs_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/execargs"
)

// testdata/ is a self-contained Go module (see testdata/go.mod): the Temporal
// SDK import is satisfied by a local stub via a replace directive, so the
// fixtures resolve both for analysistest (offline) and for IDEs. Package
// patterns are therefore module-relative paths.

// TestExecArgs runs the analyzer over the fixture packages. Each carries its
// expectations inline: "good" must produce zero diagnostics, "bad" carries a
// // want comment next to every expected report.
func TestExecArgs(t *testing.T) {
	a := execargs.NewAnalyzer(execargs.Settings{CheckTypes: true})
	analysistest.Run(t, analysistest.TestData(), a, "temporalcheckfixtures/good", "temporalcheckfixtures/bad")
}

// TestExecArgs_CheckTypesDisabled verifies that type mismatches are silenced
// when CheckTypes is off, while the arity check still fires.
func TestExecArgs_CheckTypesDisabled(t *testing.T) {
	a := execargs.NewAnalyzer(execargs.Settings{CheckTypes: false})
	analysistest.Run(t, analysistest.TestData(), a, "temporalcheckfixtures/notypes")
}
