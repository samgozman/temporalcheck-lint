// Package tests is exercised with StrictTests on: the matcher arity of every
// resolvable OnActivity/OnWorkflow mock setup is checked against the target's
// real signature. Unlike Execute*, the matchers must cover every declared
// parameter -- including the injected context -- so the expected count is the
// parameter count, with no context to skip.
package tests

import (
	"context"

	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type Activities struct{}

// Greet takes the injected context plus one real argument: two parameters, so a
// correct mock supplies two matchers.
func (a *Activities) Greet(ctx context.Context, name string) (string, error) {
	return "", nil
}

// Cleanup omits the optional leading context.Context: one parameter, one matcher.
func (a *Activities) Cleanup(jobID string) error { return nil }

// Notify is variadic: the matcher count is unknowable statically, so it is
// skipped rather than risk a false positive.
func (a *Activities) Notify(ctx context.Context, userID string, tags ...string) error { return nil }

// ChildWorkflow takes the injected workflow.Context plus one real argument.
func ChildWorkflow(ctx workflow.Context, orderID string) error { return nil }

func setup(env *testsuite.TestWorkflowEnvironment) {
	var a *Activities

	// Correct arity: one matcher per parameter, including the context.
	env.OnActivity(a.Greet, nil, nil).Return("", nil).Once()
	env.OnActivity(a.Cleanup, nil).Return(nil)
	env.OnWorkflow(ChildWorkflow, nil, nil).Return(nil)

	// Wrong arity: the context matcher is the one most often forgotten.
	env.OnActivity(a.Greet, nil)                 // want `OnActivity: mock for activity "Greet" expects 2 arguments \(one per parameter\), got 1 \(strict-tests\)`
	env.OnWorkflow(ChildWorkflow, nil, nil, nil) // want `OnWorkflow: mock for workflow "ChildWorkflow" expects 2 arguments \(one per parameter\), got 3 \(strict-tests\)`

	// A //nolint directive naming the plugin suppresses the same wrong-arity report.
	env.OnActivity(a.Greet, nil) //nolint:temporalcheck

	// Skipped: a string-named target (stringtarget's job), a spread call, and a
	// variadic target are all out of scope, so none of these is flagged.
	env.OnActivity("Greet", nil, nil)
	matchers := []any{nil, nil}
	env.OnActivity(a.Greet, matchers...)
	env.OnActivity(a.Notify, nil, nil, nil, nil)
}
