// Package billing holds an activity in a nested package, separate from the
// workflow that calls it, to exercise cross-package signature resolution.
package billing

import "context"

type Activities struct{}

func (a *Activities) Charge(ctx context.Context, userID string, amountCents int) error {
	return nil
}
