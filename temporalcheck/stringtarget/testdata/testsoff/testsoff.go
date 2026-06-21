// Package testsoff is exercised with Enabled on but StrictTests off: a
// string-named testsuite mock setup stays silent. It proves the test-mock check
// has its own flag and the production Execute* check does not bleed into mocks.
package testsoff

import "go.temporal.io/sdk/testsuite"

func setup(env *testsuite.TestWorkflowEnvironment) {
	// Named by string, but StrictTests is off, so nothing is reported.
	env.OnActivity("Greet", nil, nil)
}
