// Package tests is exercised with StrictTests on: a testsuite mock setup --
// OnActivity/OnWorkflow -- whose target is named by string is reported, while a
// call that passes the function reference is left alone. StrictTests is
// independent of Enabled, so this runs with Enabled off.
package tests

import (
	"context"

	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type Activities struct{}

func (a *Activities) Greet(ctx context.Context, name string) error { return nil }

func ChildWorkflow(ctx workflow.Context, orderID string) error { return nil }

// ActivityName is a named string type; its underlying type is string, so a value
// of it used as the target is still "named by string".
type ActivityName string

func setup(env *testsuite.TestWorkflowEnvironment) {
	var a *Activities

	// String literal targets: unresolvable to a signature, so flagged by name.
	env.OnActivity("Greet", nil, nil).Return(nil).Once() // want `OnActivity: target "Greet" is named by string; pass the function reference instead so its arguments can be checked statically \(strict-tests\)`
	env.OnWorkflow("Child", nil, nil).Return(nil)        // want `OnWorkflow: target "Child" is named by string; pass the function reference instead so its arguments can be checked statically \(strict-tests\)`

	// A string held in a variable: still named by string, but no literal to quote.
	name := "Greet"
	env.OnActivity(name, nil, nil) // want `OnActivity: the target is named by string; pass the function reference instead so its arguments can be checked statically \(strict-tests\)`

	// A named string type resolves through its underlying string.
	var an ActivityName = "Greet"
	env.OnActivity(an, nil, nil) // want `OnActivity: the target is named by string; pass the function reference instead so its arguments can be checked statically \(strict-tests\)`

	// The recommended form: a function reference. Never flagged.
	env.OnActivity(a.Greet, nil, nil)
	env.OnWorkflow(ChildWorkflow, nil, nil)
}
