package disabled

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

func chanParam(ctx context.Context, ch chan int) error { return nil }

func caller(ctx workflow.Context) {
	// With the analyzer disabled, even a clearly unserializable chan parameter is
	// not reported.
	workflow.ExecuteActivity(ctx, chanParam, nil)
}
