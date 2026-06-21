// Package structshapeoff runs with StrictTypes on but StructShape off, proving
// the behavior shift: a wire-compatible-but-distinct struct is now silent (it
// moved out of strict-types into the opt-in struct-shape check), while the harder
// struct cases — incompatible shared field, or no fields in common — remain
// strict-types errors.
package structshapeoff

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

type Activities struct{}

type Send struct {
	ID    string
	Extra bool
}

type Want struct {
	ID    string
	Other string
}

type Conflict struct {
	ID int
}

type Unrelated struct {
	Foo string
}

func (a *Activities) NeedWant(ctx context.Context, p *Want) error { return nil }

func Workflow(ctx workflow.Context) error {
	var a *Activities

	// Wire-compatible but distinct: silent because struct-shape is off.
	_ = workflow.ExecuteActivity(ctx, a.NeedWant, &Send{})

	// Incompatible shared field: still a strict-types error.
	_ = workflow.ExecuteActivity(ctx, a.NeedWant, &Conflict{}) // want `ExecuteActivity: arg 1 of "NeedWant" sends \*structshapeoff.Conflict, target wants \*structshapeoff.Want — field "ID" is incompatible \(int vs string\) \(strict-types\)`

	// No fields in common: still a strict-types error.
	_ = workflow.ExecuteActivity(ctx, a.NeedWant, &Unrelated{}) // want `ExecuteActivity: arg 1 of "NeedWant" sends \*structshapeoff.Unrelated, target wants \*structshapeoff.Want — no fields in common \(strict-types\)`

	return nil
}
