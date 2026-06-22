package custompattern

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// call has one parameter matching the custom pattern (apiKey) and one matching
// only the default pattern (password). The test runs with Pattern set to
// "(?i)apikey", so only apiKey is flagged -- proof the custom pattern replaces the
// default rather than adding to it.
func call(ctx context.Context, apiKey string, password string) error { return nil }

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, call, "k", "p") // want `activity "call" parameter 1 "apiKey" matches the sensitive-data pattern;.*\(sensitive\)`
}
