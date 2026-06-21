package disabled

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

func anyParam(ctx context.Context, v any) error { return nil }

func caller(ctx workflow.Context) {
	// With the analyzer disabled, even a clearly lossy `any` parameter is not
	// reported.
	workflow.ExecuteActivity(ctx, anyParam, 1)
}
