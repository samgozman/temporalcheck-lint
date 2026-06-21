package activities

import "context"

// AnyParam is an activity defined in a different package from the workflow that
// executes it -- the common project layout. Its signature still resolves through
// go/types, so the lossy parameter is caught at the call site.
func AnyParam(ctx context.Context, v any) error { return nil }
