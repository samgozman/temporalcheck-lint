package nonserializable_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/nonserializable"
)

// Fixtures live in testdata/, a self-contained module (see testdata/go.mod), so
// the patterns below are module-relative package paths.

// TestNonSerializable: a top-level chan or func parameter or return -- a bare
// chan/func, a named channel/function type, or a variadic ...chan -- on an
// activity, child workflow or client-started workflow is reported at the Execute*
// target reference.
func TestNonSerializable(t *testing.T) {
	a := nonserializable.NewAnalyzer(nonserializable.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "nonserializablefixtures/flagged")
}

// TestNonSerializable_Benign: concrete types, a struct that merely contains a
// chan/func field, a slice of channels (deeper than top level), a struct with no
// exported fields (the opt-in EmptyStruct check is off), an error-only return, and
// a string-registered (unresolvable) target all produce no diagnostics.
func TestNonSerializable_Benign(t *testing.T) {
	a := nonserializable.NewAnalyzer(nonserializable.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "nonserializablefixtures/benign")
}

// TestNonSerializable_EmptyStruct: with EmptyStruct enabled, a struct that has
// fields but none exported is flagged -- unless it implements json.Marshaler -- and
// the always-on chan check still fires. A fieldless struct{} and a struct with an
// exported field are not flagged.
func TestNonSerializable_EmptyStruct(t *testing.T) {
	a := nonserializable.NewAnalyzer(nonserializable.Settings{EmptyStruct: true})
	analysistest.Run(t, analysistest.TestData(), a, "nonserializablefixtures/emptystruct")
}

// TestNonSerializable_Client: client.ExecuteWorkflow -- a method on the SDK's
// client.Client interface, not a package function -- is matched by receiver, and
// its workflow target's chan/func parameter/return is reported (with the leading
// workflow.Context skipped).
func TestNonSerializable_Client(t *testing.T) {
	a := nonserializable.NewAnalyzer(nonserializable.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "nonserializablefixtures/clientstart")
}

// TestNonSerializable_CrossPkg: an activity defined in a different package is
// resolved to its signature and its chan parameter reported at the call site.
func TestNonSerializable_CrossPkg(t *testing.T) {
	a := nonserializable.NewAnalyzer(nonserializable.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "nonserializablefixtures/crosspkg")
}

// TestNonSerializable_Disabled: with Disabled set, the analyzer reports nothing
// even on a clearly unserializable chan parameter.
func TestNonSerializable_Disabled(t *testing.T) {
	a := nonserializable.NewAnalyzer(nonserializable.Settings{Disabled: true})
	analysistest.Run(t, analysistest.TestData(), a, "nonserializablefixtures/disabled")
}

// TestNonSerializable_Nolint: a //nolint directive naming temporalcheck (or all,
// or bare) on the call's line suppresses its diagnostic; a directive naming only
// other linters, or the analyzer name, does not.
func TestNonSerializable_Nolint(t *testing.T) {
	a := nonserializable.NewAnalyzer(nonserializable.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "nonserializablefixtures/nolint")
}

// TestNonSerializable_NolintFile: a //nolint directive before the package clause
// suppresses every diagnostic in the file.
func TestNonSerializable_NolintFile(t *testing.T) {
	a := nonserializable.NewAnalyzer(nonserializable.Settings{})
	analysistest.Run(t, analysistest.TestData(), a, "nonserializablefixtures/nolintfile")
}
