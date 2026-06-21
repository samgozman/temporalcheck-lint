// Package testsoff is exercised with StrictTests off (and Enabled off): a
// string-named testsuite mock setup stays silent. It proves the test-mock check
// is gated by its own flag, independent of the production Execute* check.
package testsoff

import "go.temporal.io/sdk/testsuite"

func setup(env *testsuite.TestWorkflowEnvironment) {
	// Named by string, but StrictTests is off, so nothing is reported.
	env.OnActivity("Greet", nil, nil)
}
