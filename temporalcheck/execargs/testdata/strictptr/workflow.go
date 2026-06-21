package strictptr

import "go.temporal.io/sdk/workflow"

// With StrictPointers enabled, a value passed where a pointer is expected (and
// vice versa), including the slice forms, is flagged even though Temporal's
// DataConverter would serialize them identically.
func Workflow(ctx workflow.Context) error {
	var a *Activities
	p := Payload{}

	// Exact matches: still fine under StrictPointers.
	_ = workflow.ExecuteActivity(ctx, a.SaveValue, Payload{})
	_ = workflow.ExecuteActivity(ctx, a.SavePointer, &p)
	_ = workflow.ExecuteActivity(ctx, a.SaveValues, []Payload{})
	_ = workflow.ExecuteActivity(ctx, a.SavePointers, []*Payload{})

	// Struct expected, pointer given.
	_ = workflow.ExecuteActivity(ctx, a.SaveValue, &p) // want `ExecuteActivity: arg 1 of "SaveValue" has type \*strictptr.Payload, want strictptr.Payload`

	// Pointer expected, struct value given.
	_ = workflow.ExecuteActivity(ctx, a.SavePointer, p) // want `ExecuteActivity: arg 1 of "SavePointer" has type strictptr.Payload, want \*strictptr.Payload`

	// []struct expected, []pointer given.
	_ = workflow.ExecuteActivity(ctx, a.SaveValues, []*Payload{}) // want `ExecuteActivity: arg 1 of "SaveValues" has type \[\]\*strictptr.Payload, want \[\]strictptr.Payload`

	// []pointer expected, []struct given.
	_ = workflow.ExecuteActivity(ctx, a.SavePointers, []Payload{}) // want `ExecuteActivity: arg 1 of "SavePointers" has type \[\]strictptr.Payload, want \[\]\*strictptr.Payload`

	return nil
}
