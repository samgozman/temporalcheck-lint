package satisfied

import (
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// A workflow that upserts the mapped attribute -- via either the typed or the
// legacy API, with the attribute name appearing as a string literal -- is not
// flagged.

type Params struct {
	UserID string
}

// Typed uses the modern typed API; the attribute name is the literal passed to
// the key constructor.
func Typed(ctx workflow.Context, p Params) error {
	return workflow.UpsertTypedSearchAttributes(ctx, temporal.NewSearchAttributeKeyKeyword("user_id").ValueSet(p.UserID))
}

// Legacy uses the deprecated untyped API; the attribute name is a map key literal.
func Legacy(ctx workflow.Context, p Params) error {
	return workflow.UpsertSearchAttributes(ctx, map[string]interface{}{"user_id": p.UserID})
}

// Nested upserts inside a nested block; the whole body is searched, so it counts.
func Nested(ctx workflow.Context, p Params) error {
	if p.UserID != "" {
		_ = workflow.UpsertTypedSearchAttributes(ctx, temporal.NewSearchAttributeKeyString("user_id").ValueSet(p.UserID))
	}
	return nil
}
