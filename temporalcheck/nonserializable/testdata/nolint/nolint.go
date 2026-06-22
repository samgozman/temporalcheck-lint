package nolint

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

func chanParam(ctx context.Context, ch chan int) error { return nil }

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, chanParam, nil) //nolint:temporalcheck // names the plugin: suppressed
	workflow.ExecuteActivity(ctx, chanParam, nil) //nolint
	workflow.ExecuteActivity(ctx, chanParam, nil) //nolint:all
	workflow.ExecuteActivity(ctx, chanParam, nil) //nolint:otherlinter // want `activity "chanParam" parameter 1 has type chan int;.*\(unencodable\)`
	workflow.ExecuteActivity(ctx, chanParam, nil) //nolint:nonserializable // analyzer name, not the plugin: not suppressed // want `activity "chanParam" parameter 1 has type chan int;.*\(unencodable\)`
}
