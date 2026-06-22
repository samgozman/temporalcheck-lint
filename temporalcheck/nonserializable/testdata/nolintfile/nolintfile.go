//nolint
package nolintfile

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

func chanParam(ctx context.Context, ch chan int) error { return nil }

func caller(ctx workflow.Context) {
	// A //nolint before the package clause suppresses every diagnostic in the file.
	workflow.ExecuteActivity(ctx, chanParam, nil)
}
