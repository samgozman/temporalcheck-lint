package sensitiveargs_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/sensitiveargs"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths. Every test enables the
// analyzer, since it reports nothing by default.

// TestSensitiveArgs: a parameter whose name matches the default pattern -- on an
// activity, local activity, child workflow or continue-as-new target -- is
// reported at the Execute* call, with the injected context skipped and the
// parameter numbered 1-based over the user parameters.
func TestSensitiveArgs(t *testing.T) {
	a := sensitiveargs.NewAnalyzer(sensitiveargs.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "sensitiveargsfixtures/flagged")
}

// TestSensitiveArgs_StructFields: an exported field of a struct (or
// pointer-to-struct) parameter whose name matches is reported; unexported fields,
// which never serialize, are not.
func TestSensitiveArgs_StructFields(t *testing.T) {
	a := sensitiveargs.NewAnalyzer(sensitiveargs.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "sensitiveargsfixtures/structfields")
}

// TestSensitiveArgs_Benign: benign parameter and field names, an unexported
// sensitive field, a sensitive field reached only through a slice (not descended),
// and a string-registered (unresolvable) target all produce no diagnostics.
func TestSensitiveArgs_Benign(t *testing.T) {
	a := sensitiveargs.NewAnalyzer(sensitiveargs.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "sensitiveargsfixtures/benign")
}

// TestSensitiveArgs_Client: client.ExecuteWorkflow and SignalWithStartWorkflow --
// methods on the SDK's client.Client interface -- are matched by receiver and
// their workflow target's sensitive parameter reported.
func TestSensitiveArgs_Client(t *testing.T) {
	a := sensitiveargs.NewAnalyzer(sensitiveargs.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "sensitiveargsfixtures/clientstart")
}

// TestSensitiveArgs_CrossPkg: an activity defined in another package keeps its
// parameter names in the exported signature, so its sensitive parameter is caught
// at the call site.
func TestSensitiveArgs_CrossPkg(t *testing.T) {
	a := sensitiveargs.NewAnalyzer(sensitiveargs.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "sensitiveargsfixtures/crosspkg")
}

// TestSensitiveArgs_CustomPattern: a custom Pattern replaces the default, so only
// names matching it are flagged.
func TestSensitiveArgs_CustomPattern(t *testing.T) {
	a := sensitiveargs.NewAnalyzer(sensitiveargs.Settings{Enabled: true, Pattern: `(?i)apikey`})
	analysistest.Run(t, analysistest.TestData(), a, "sensitiveargsfixtures/custompattern")
}

// TestSensitiveArgs_Disabled: with Enabled off (the default), the analyzer reports
// nothing even on an obvious match.
func TestSensitiveArgs_Disabled(t *testing.T) {
	a := sensitiveargs.NewAnalyzer(sensitiveargs.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "sensitiveargsfixtures/disabled")
}

// TestSensitiveArgs_Nolint: a //nolint directive naming temporalcheck (or all, or
// bare) on the call's line suppresses its diagnostic; a directive naming only
// other linters, or the analyzer name, does not.
func TestSensitiveArgs_Nolint(t *testing.T) {
	a := sensitiveargs.NewAnalyzer(sensitiveargs.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "sensitiveargsfixtures/nolint")
}

// TestSensitiveArgs_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestSensitiveArgs_NolintFile(t *testing.T) {
	a := sensitiveargs.NewAnalyzer(sensitiveargs.Settings{Enabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "sensitiveargsfixtures/nolintfile")
}
