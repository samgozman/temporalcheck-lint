package othercalls

import "go.temporal.io/sdk/workflow"

// Calls in a workflow body that are not search-attribute upserts -- a bare local
// function, a type conversion, a call through a function-typed field -- are
// ignored, so a workflow that makes them but never upserts is still flagged.

type Params struct {
	UserID string
	Fn     func()
}

func helper() {}

func WithOtherCalls(ctx workflow.Context, p Params) error { // want `workflow "WithOtherCalls" takes field "UserID" but never upserts the "user_id" search attribute \(searchattribute\)`
	helper()        // non-selector call
	_ = string("x") // conversion: the callee is an identifier, not a selector
	p.Fn()          // selector call, but Fn is a field, not a function
	return nil
}
