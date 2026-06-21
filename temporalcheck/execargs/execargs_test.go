package execargs_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/execargs"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths.

// TestExecArgs checks the fixture packages: "good" must report nothing; "bad"
// and "crosspkg" carry a // want next to each expected diagnostic. "crosspkg"
// calls an activity defined in a separate nested package.
func TestExecArgs(t *testing.T) {
	a := execargs.NewAnalyzer(execargs.Settings{StrictTypes: true})
	analysistest.Run(t, analysistest.TestData(), a,
		"temporalcheckfixtures/good", "temporalcheckfixtures/bad", "temporalcheckfixtures/crosspkg")
}

// TestExecArgs_StrictTypesDisabled: with StrictTypes off (the default), type
// mismatches are silent but arity is still checked.
func TestExecArgs_StrictTypesDisabled(t *testing.T) {
	a := execargs.NewAnalyzer(execargs.Settings{StrictTypes: false})
	analysistest.Run(t, analysistest.TestData(), a, "temporalcheckfixtures/notypes")
}

// TestExecArgs_StrictPointers: with StrictPointers on, value-vs-pointer and
// []T-vs-[]*T mismatches (silently accepted by default) are reported.
func TestExecArgs_StrictPointers(t *testing.T) {
	a := execargs.NewAnalyzer(execargs.Settings{StrictTypes: true, StrictPointers: true})
	analysistest.Run(t, analysistest.TestData(), a, "temporalcheckfixtures/strictptr")
}
