//nolint
package nolintfile

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

func anyParam(ctx context.Context, v any) error { return nil }

func caller(ctx workflow.Context) {
	// A //nolint before the package clause suppresses every diagnostic in the file.
	workflow.ExecuteActivity(ctx, anyParam, 1)
}
