package nolint

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

func anyParam(ctx context.Context, v any) error { return nil }

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, anyParam, 1) //nolint:temporalcheck // names the plugin: suppressed
	workflow.ExecuteActivity(ctx, anyParam, 1) //nolint
	workflow.ExecuteActivity(ctx, anyParam, 1) //nolint:all
	workflow.ExecuteActivity(ctx, anyParam, 1) //nolint:otherlinter // want `activity "anyParam" parameter 1 has dynamic type any;.*\(lossy-types\)`
	workflow.ExecuteActivity(ctx, anyParam, 1) //nolint:lossynumber // analyzer name, not the plugin: not suppressed // want `activity "anyParam" parameter 1 has dynamic type any;.*\(lossy-types\)`
}
