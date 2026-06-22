package crosspkg

import (
	"go.temporal.io/sdk/workflow"

	"nonserializablefixtures/crosspkg/activities"
)

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, activities.ChanParam, nil) // want `activity "ChanParam" parameter 1 has type chan int;.*\(unencodable\)`
}
