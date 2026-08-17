package opaque

import (
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// When a workflow upserts search attributes through a source we cannot read
// statically -- a prebuilt map, a spread slice, or a key held in a variable --
// the attribute names are unknown, so we cannot prove the required one is
// missing. Rather than risk a false positive, the whole workflow is left alone.

type Params struct {
	UserID string
}

// PrebuiltMap passes a map built elsewhere: no key literals in the call.
func PrebuiltMap(ctx workflow.Context, p Params) error {
	attrs := map[string]interface{}{"user_id": p.UserID}
	return workflow.UpsertSearchAttributes(ctx, attrs)
}

// Spread passes a slice of updates variadically: no literals in the call.
func Spread(ctx workflow.Context, p Params, updates []temporal.SearchAttributeUpdate) error {
	return workflow.UpsertTypedSearchAttributes(ctx, updates...)
}

// KeyInVar builds the key in a variable, so the attribute name literal is not
// inside the upsert call.
func KeyInVar(ctx workflow.Context, p Params) error {
	key := temporal.NewSearchAttributeKeyKeyword("user_id")
	return workflow.UpsertTypedSearchAttributes(ctx, key.ValueSet(p.UserID))
}
