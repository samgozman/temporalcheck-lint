package disabled

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

func charge(ctx context.Context, cvv string) error { return nil }

func caller(ctx workflow.Context) {
	// With Enabled off the analyzer reports nothing, even on an obvious match.
	workflow.ExecuteActivity(ctx, charge, "c")
}
