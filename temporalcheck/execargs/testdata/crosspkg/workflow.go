package crosspkg

import (
	"go.temporal.io/sdk/workflow"

	"temporalcheckfixtures/crosspkg/activities/billing"
)

func Workflow(ctx workflow.Context) error {
	var a *billing.Activities

	// Correct call resolved across the package boundary.
	_ = workflow.ExecuteActivity(ctx, a.Charge, "user-1", 500)

	// Wrong arity must still be caught — proof the cross-package signature was
	// resolved, not skipped.
	_ = workflow.ExecuteActivity(ctx, a.Charge, "user-1") // want `ExecuteActivity: activity "Charge" expects 2 arguments, got 1 \(arity\)`

	// Wrong type on the cross-package activity.
	_ = workflow.ExecuteActivity(ctx, a.Charge, "user-1", "free") // want `ExecuteActivity: arg 2 of "Charge" has type (untyped )?string, want int \(strict-types\)`

	return nil
}
