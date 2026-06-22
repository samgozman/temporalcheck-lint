package flagged

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// Activities and workflows with parameters named like secrets. Each is referenced
// by an Execute* call below, where the analyzer anchors its diagnostic. The number
// in the message is 1-based over the user parameters, after any injected context.

func chargeCard(ctx context.Context, cardNumber string, cvv string) error { return nil }

func login(ctx context.Context, username, password string) error { return nil }

func storeToken(ctx context.Context, token string) error { return nil }

// noCtxSecret omits the optional leading context.Context, so its first parameter
// is the sensitive one.
func noCtxSecret(secret string) error { return nil }

func resetPassword(ctx workflow.Context, ssn string) error { return nil }

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, chargeCard, "n", "c")        // want `activity "chargeCard" parameter 1 "cardNumber" matches the sensitive-data pattern;.*\(sensitive\)` `activity "chargeCard" parameter 2 "cvv" matches the sensitive-data pattern;.*\(sensitive\)`
	workflow.ExecuteActivity(ctx, login, "u", "p")             // want `activity "login" parameter 2 "password" matches the sensitive-data pattern;.*\(sensitive\)`
	workflow.ExecuteLocalActivity(ctx, storeToken, "t")        // want `activity "storeToken" parameter 1 "token" matches the sensitive-data pattern;.*\(sensitive\)`
	workflow.ExecuteActivity(ctx, noCtxSecret, "s")            // want `activity "noCtxSecret" parameter 1 "secret" matches the sensitive-data pattern;.*\(sensitive\)`
	workflow.ExecuteChildWorkflow(ctx, resetPassword, "s")     // want `child workflow "resetPassword" parameter 1 "ssn" matches the sensitive-data pattern;.*\(sensitive\)`
	_ = workflow.NewContinueAsNewError(ctx, resetPassword, "s") // want `workflow "resetPassword" parameter 1 "ssn" matches the sensitive-data pattern;.*\(sensitive\)`
}
