package crosspkg

import (
	"go.temporal.io/sdk/workflow"

	"sensitiveargsfixtures/crosspkg/activities"
)

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, activities.ChargeCard, "c") // want `activity "ChargeCard" parameter 1 "cvv" matches the sensitive-data pattern;.*\(sensitive\)`
}
