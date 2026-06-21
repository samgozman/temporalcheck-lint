// Package flagged is exercised with the check enabled: every Execute* call whose
// target is a string -- a literal, a string variable, or a named string type --
// is reported, while a call that passes the function reference is left alone.
package flagged

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

type Activities struct{}

func (a *Activities) Greet(ctx context.Context, name string) error { return nil }

// ActivityName is a named string type; its underlying type is string, so a value
// of it used as the target is still "named by string".
type ActivityName string

func Workflow(ctx workflow.Context) error {
	var a *Activities

	// String literal targets: unresolvable to a signature, so flagged by name.
	_ = workflow.ExecuteActivity(ctx, "Greet", "world")      // want `ExecuteActivity: target "Greet" is named by string; pass the function reference instead so its arguments can be checked statically \(string-target\)`
	_ = workflow.ExecuteLocalActivity(ctx, "Greet", "world") // want `ExecuteLocalActivity: target "Greet" is named by string; pass the function reference instead so its arguments can be checked statically \(string-target\)`
	_ = workflow.ExecuteChildWorkflow(ctx, "Child", 1)       // want `ExecuteChildWorkflow: target "Child" is named by string; pass the function reference instead so its arguments can be checked statically \(string-target\)`

	// A string held in a variable: still named by string, but no literal to quote.
	name := "Greet"
	_ = workflow.ExecuteActivity(ctx, name, "world") // want `ExecuteActivity: the target is named by string; pass the function reference instead so its arguments can be checked statically \(string-target\)`

	// A named string type resolves through its underlying string.
	var an ActivityName = "Greet"
	_ = workflow.ExecuteActivity(ctx, an, "world") // want `ExecuteActivity: the target is named by string; pass the function reference instead so its arguments can be checked statically \(string-target\)`

	// The recommended form: a function reference. Never flagged -- this is exactly
	// what lets execargs check the arguments.
	_ = workflow.ExecuteActivity(ctx, a.Greet, "world")

	return nil
}
