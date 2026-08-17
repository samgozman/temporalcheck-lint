package helpers

import "go.temporal.io/sdk/workflow"

// Unexported functions whose first parameter is workflow.Context are workflow
// *helpers* -- written to split up a workflow's logic and called directly by name
// -- not registered entrypoints run by the Temporal runtime. They carry the same
// params as their caller, so treating them as entrypoints flags every one. Only
// exported functions are considered entrypoints, so none of these are flagged.

type Params struct {
	UserID string
}

func processStep(ctx workflow.Context, p Params) error { return nil }

func handleFailure(ctx workflow.Context, p *Params) error { return nil }

// publishEvent takes two workflow.Context parameters (a common helper shape) and
// is still a helper, not an entrypoint.
func publishEvent(workflowCtx, activityCtx workflow.Context, p Params) error { return nil }
