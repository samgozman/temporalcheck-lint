package benign

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// order has only benign exported fields; its unexported password is never
// serialized, so a sensitive-looking unexported name is not flagged.
type order struct {
	OrderID  string
	Amount   int
	password string
}

// card has an exported sensitive field, but it is only reached via a slice
// parameter below; the analyzer does not descend into slices, so it is not
// flagged -- proof the field check stays at the top level.
type card struct {
	CardNumber string
}

func process(ctx context.Context, orderID string, amount int) error { return nil }

func withStruct(ctx context.Context, o order) error { return nil }

func withCardSlice(ctx context.Context, cards []card) error { return nil }

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, process, "id", 1)
	workflow.ExecuteActivity(ctx, withStruct, order{})
	workflow.ExecuteActivity(ctx, withCardSlice, nil)
	// A string-registered target can't be resolved to a signature, so it is
	// skipped rather than risk a false positive.
	workflow.ExecuteActivity(ctx, "ChargeCardActivity", "x")
}
