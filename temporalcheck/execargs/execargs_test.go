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

// TestExecArgs_StructShape: with only StructShape on (proving independence),
// passing one struct where a different struct is wanted is reported, with the
// drift detail; incompatible/no-overlap structs surface as strict-types errors.
func TestExecArgs_StructShape(t *testing.T) {
	a := execargs.NewAnalyzer(execargs.Settings{StructShape: true})
	analysistest.Run(t, analysistest.TestData(), a, "temporalcheckfixtures/structshape")
}

// TestExecArgs_StructShapeOff: with StrictTypes on but StructShape off, the
// wire-compatible-but-distinct struct is silent (moved out of strict-types),
// while incompatible and no-overlap structs remain strict-types errors.
func TestExecArgs_StructShapeOff(t *testing.T) {
	a := execargs.NewAnalyzer(execargs.Settings{StrictTypes: true})
	analysistest.Run(t, analysistest.TestData(), a, "temporalcheckfixtures/structshapeoff")
}

// TestExecArgs_Nolint: a //nolint directive naming temporalcheck (or all, or
// bare) on the call's line suppresses its diagnostic; a directive naming only
// other linters, or the analyzer name execargs, does not.
func TestExecArgs_Nolint(t *testing.T) {
	a := execargs.NewAnalyzer(execargs.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "temporalcheckfixtures/nolint")
}

// TestExecArgs_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestExecArgs_NolintFile(t *testing.T) {
	a := execargs.NewAnalyzer(execargs.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "temporalcheckfixtures/nolintfile")
}

// TestExecArgs_Disabled: with Disabled set, the analyzer reports nothing even on
// a fixture that carries a real arity violation.
func TestExecArgs_Disabled(t *testing.T) {
	a := execargs.NewAnalyzer(execargs.Settings{Disabled: true, StrictTypes: true})
	analysistest.Run(t, analysistest.TestData(), a, "temporalcheckfixtures/disabled")
}
