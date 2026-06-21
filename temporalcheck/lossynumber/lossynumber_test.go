package lossynumber_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/lossynumber"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths.

// TestLossyNumber: a top-level lossy dynamic parameter or return -- interface{}/
// any, map[string]any, []any, a named empty interface, or a variadic ...any --
// on an activity, child workflow or client-started workflow is reported at the
// Execute* target reference.
func TestLossyNumber(t *testing.T) {
	a := lossynumber.NewAnalyzer(lossynumber.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "lossynumberfixtures/flagged")
}

// TestLossyNumber_Benign: concrete types, a struct that merely contains an `any`
// field, non-empty interfaces (io.Reader, error, a custom interface), concrete
// collections, an error-only return, and a string-registered (unresolvable)
// target all produce no diagnostics.
func TestLossyNumber_Benign(t *testing.T) {
	a := lossynumber.NewAnalyzer(lossynumber.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "lossynumberfixtures/benign")
}

// TestLossyNumber_Client: client.ExecuteWorkflow -- a method on the SDK's
// client.Client interface, not a package function -- is matched by receiver, and
// its workflow target's lossy parameter/return is reported (with the leading
// workflow.Context skipped).
func TestLossyNumber_Client(t *testing.T) {
	a := lossynumber.NewAnalyzer(lossynumber.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "lossynumberfixtures/clientstart")
}

// TestLossyNumber_CrossPkg: an activity defined in a different package is
// resolved to its signature and its lossy parameter reported at the call site.
func TestLossyNumber_CrossPkg(t *testing.T) {
	a := lossynumber.NewAnalyzer(lossynumber.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "lossynumberfixtures/crosspkg")
}

// TestLossyNumber_Disabled: with Disabled set, the analyzer reports nothing even
// on a clearly lossy `any` parameter.
func TestLossyNumber_Disabled(t *testing.T) {
	a := lossynumber.NewAnalyzer(lossynumber.Settings{Disabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "lossynumberfixtures/disabled")
}

// TestLossyNumber_Nolint: a //nolint directive naming temporalcheck (or all, or
// bare) on the call's line suppresses its diagnostic; a directive naming only
// other linters, or the analyzer name, does not.
func TestLossyNumber_Nolint(t *testing.T) {
	a := lossynumber.NewAnalyzer(lossynumber.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "lossynumberfixtures/nolint")
}

// TestLossyNumber_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestLossyNumber_NolintFile(t *testing.T) {
	a := lossynumber.NewAnalyzer(lossynumber.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "lossynumberfixtures/nolintfile")
}
