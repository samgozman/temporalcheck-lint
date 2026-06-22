package activities

import "context"

// ChanParam is an activity defined in a different package from the workflow that
// executes it -- the common project layout. Its signature still resolves through
// go/types, so the unserializable parameter is caught at the call site.
func ChanParam(ctx context.Context, ch chan int) error { return nil }
