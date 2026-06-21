// Package testsoff is exercised with StrictTests off (the default): a mock setup
// with the wrong matcher arity stays silent, the same way the strict type layers
// stay silent when disabled. The always-on Execute* arity check is unaffected.
package testsoff

import (
	"context"

	"go.temporal.io/sdk/testsuite"
)

type Activities struct{}

func (a *Activities) Greet(ctx context.Context, name string) (string, error) { return "", nil }

func setup(env *testsuite.TestWorkflowEnvironment) {
	var a *Activities

	// Wrong arity, but StrictTests is off, so nothing is reported.
	env.OnActivity(a.Greet, nil)
}
