package nolint

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

func charge(ctx context.Context, cvv string) error { return nil }

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, charge, "c") //nolint:temporalcheck // names the plugin: suppressed
	workflow.ExecuteActivity(ctx, charge, "c") //nolint
	workflow.ExecuteActivity(ctx, charge, "c") //nolint:all
	workflow.ExecuteActivity(ctx, charge, "c") //nolint:otherlinter // want `activity "charge" parameter 1 "cvv" matches the sensitive-data pattern;.*\(sensitive\)`
	workflow.ExecuteActivity(ctx, charge, "c") //nolint:sensitiveargs // analyzer name, not the plugin: not suppressed // want `activity "charge" parameter 1 "cvv" matches the sensitive-data pattern;.*\(sensitive\)`
}
