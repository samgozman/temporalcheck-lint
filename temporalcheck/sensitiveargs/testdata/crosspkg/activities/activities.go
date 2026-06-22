package activities

import "context"

// ChargeCard is an activity defined in a different package from the workflow that
// executes it. Its parameter names survive in the exported signature, so the
// sensitive one is caught at the call site.
func ChargeCard(ctx context.Context, cvv string) error { return nil }
