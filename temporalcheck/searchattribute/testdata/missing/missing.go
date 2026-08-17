package missing

import (
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// A workflow whose params struct carries a configured field must upsert the
// mapped search attribute somewhere in its body. These workflows never do, so
// each is flagged at its function name.

type Params struct {
	UserID  string
	OrderID string
}

// ProcessOrder takes a UserID but upserts nothing.
func ProcessOrder(ctx workflow.Context, p Params) error { // want `workflow "ProcessOrder" takes field "UserID" but never upserts the "user_id" search attribute \(searchattribute\)`
	return nil
}

// UpsertsWrongAttribute upserts a different attribute, not user_id, so it is
// still flagged: the required attribute is absent.
func UpsertsWrongAttribute(ctx workflow.Context, p Params) error { // want `workflow "UpsertsWrongAttribute" takes field "UserID" but never upserts the "user_id" search attribute \(searchattribute\)`
	_ = workflow.UpsertTypedSearchAttributes(ctx, temporal.NewSearchAttributeKeyKeyword("order_id").ValueSet(p.OrderID))
	return nil
}

// PointerParam takes the params by pointer; one level of pointer is dereferenced
// to reach the struct, so the field is still found and the missing upsert flagged.
func PointerParam(ctx workflow.Context, p *Params) error { // want `workflow "PointerParam" takes field "UserID" but never upserts the "user_id" search attribute \(searchattribute\)`
	return nil
}
