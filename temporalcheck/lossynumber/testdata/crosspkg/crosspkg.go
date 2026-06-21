package crosspkg

import (
	"go.temporal.io/sdk/workflow"

	"lossynumberfixtures/crosspkg/activities"
)

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, activities.AnyParam, 1) // want `activity "AnyParam" parameter 1 has dynamic type any;.*\(lossy-types\)`
}
