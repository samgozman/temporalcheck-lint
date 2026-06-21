// Package notypes is exercised with StrictTypes disabled (the default): type
// mismatches must be silent, while the arity check must still fire.
package notypes

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

type Activities struct{}

func (a *Activities) Greet(ctx context.Context, name string) (string, error) {
	return "", nil
}

func (a *Activities) Tag(ctx context.Context, ids ...string) error {
	return nil
}

func Workflow(ctx workflow.Context) error {
	var a *Activities

	// Correct arity, wrong type: silenced because strict-types is off.
	_ = workflow.ExecuteActivity(ctx, a.Greet, 42)

	// Variadic, wrong element types: also silenced.
	_ = workflow.ExecuteActivity(ctx, a.Tag, 1, 2)

	// Arity is always checked, regardless of strict-types.
	_ = workflow.ExecuteActivity(ctx, a.Greet) // want `ExecuteActivity: activity "Greet" expects 1 argument, got 0 \(arity\)`

	return nil
}
