package structfields

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// A struct passed as an argument carries its fields into history. The analyzer
// flags each *exported* field whose name matches the pattern; unexported fields
// are never serialized, so they are left alone (see benign).

type PaymentRequest struct {
	OrderID    string
	CardNumber string
	CVV        string
	amount     int // unexported: never serialized, never flagged
}

type LoginRequest struct {
	Username string
	Password string
}

func charge(ctx context.Context, req PaymentRequest) error { return nil }

// chargePtr takes the request by pointer; the analyzer dereferences one pointer
// level to reach the struct, and the diagnostic still names the parameter's
// declared (pointer) type.
func chargePtr(ctx context.Context, req *PaymentRequest) error { return nil }

func login(ctx context.Context, req LoginRequest) error { return nil }

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, charge, PaymentRequest{})     // want `activity "charge" parameter 1 \(type structfields.PaymentRequest\) field "CardNumber" matches the sensitive-data pattern;.*\(sensitive\)` `activity "charge" parameter 1 \(type structfields.PaymentRequest\) field "CVV" matches the sensitive-data pattern;.*\(sensitive\)`
	workflow.ExecuteActivity(ctx, chargePtr, &PaymentRequest{}) // want `activity "chargePtr" parameter 1 \(type \*structfields.PaymentRequest\) field "CardNumber" matches the sensitive-data pattern;.*\(sensitive\)` `activity "chargePtr" parameter 1 \(type \*structfields.PaymentRequest\) field "CVV" matches the sensitive-data pattern;.*\(sensitive\)`
	workflow.ExecuteActivity(ctx, login, LoginRequest{})        // want `activity "login" parameter 1 \(type structfields.LoginRequest\) field "Password" matches the sensitive-data pattern;.*\(sensitive\)`
}
