package benign

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// None of these should be flagged.

type Params struct {
	UserID string
}

type NoMatch struct {
	OrderID string
	Total   int
}

type unexportedOnly struct {
	userID string // unexported: never serialized, never matched
}

// plainFunc is not a workflow (no workflow.Context), so its UserID field is
// irrelevant to search attributes.
func plainFunc(ctx context.Context, p Params) error { return nil }

// NoMatchingField is a workflow, but its params carry no configured field.
func NoMatchingField(ctx workflow.Context, p NoMatch) error { return nil }

// NoParams is a workflow with only the context.
func NoParams(ctx workflow.Context) error { return nil }

// Unexported has a userID field, but unexported fields never serialize, so no
// attribute is expected.
func Unexported(ctx workflow.Context, p unexportedOnly) error { return nil }

// NonStructParam takes a scalar, not a struct: nothing to inspect.
func NonStructParam(ctx workflow.Context, id string) error { return nil }
