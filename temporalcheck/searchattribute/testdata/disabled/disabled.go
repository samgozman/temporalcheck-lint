package disabled

import "go.temporal.io/sdk/workflow"

// With the analyzer disabled (the default), even an obvious missing upsert is not
// reported.

type Params struct {
	UserID string
}

func ProcessOrder(ctx workflow.Context, p Params) error {
	return nil
}
