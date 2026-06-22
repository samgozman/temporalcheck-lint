// Package benign exercises the shapes the analyzer must leave alone. Because it
// only ever fires on a SEEN contradiction, every form below -- correct usage, an
// unconfigured context, a matching helper present in the chain, and every "bail
// to unknown" case -- produces no diagnostics.
package benign

import (
	"strings"

	"go.temporal.io/sdk/workflow"
)

func greet(ctx workflow.Context) error   { return nil }
func childWf(ctx workflow.Context) error { return nil }

// twoValueCtx returns a context opaquely, used to exercise a multi-value
// assignment to a tracked context variable.
func twoValueCtx(ctx workflow.Context) (workflow.Context, error) { return ctx, nil }

type holder struct {
	ctx workflow.Context
	run func()
}

// Correct: matching helper applied, then the matching Execute* call.
func Correct(ctx workflow.Context) error {
	var ao workflow.ActivityOptions
	ctx = workflow.WithActivityOptions(ctx, ao)
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

// Unconfigured: a bare parameter, no With*Options applied in sight -- absence,
// never a contradiction. The caller may have configured it.
func Unconfigured(ctx workflow.Context) error {
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

// BothApplied: a conflicting helper AND the matching one are applied (each sets a
// distinct key), so the activity options DO apply -- no conflict.
func BothApplied(ctx workflow.Context) error {
	var (
		ao  workflow.ActivityOptions
		cwo workflow.ChildWorkflowOptions
	)
	ctx = workflow.WithChildOptions(ctx, cwo)
	ctx = workflow.WithActivityOptions(ctx, ao)
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

// OpaqueReset: an opaque helper returning a Context makes the value unknown, so a
// previously-seen conflict no longer fires.
func OpaqueReset(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	ctx = workflow.WithValue(ctx, "k", "v")
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

// BranchedKinds: different kinds assigned in different branches collapse to
// unknown after the if, so the later call does not fire.
func BranchedKinds(ctx workflow.Context, cond bool) error {
	var (
		ao  workflow.ActivityOptions
		cwo workflow.ChildWorkflowOptions
	)
	if cond {
		ctx = workflow.WithActivityOptions(ctx, ao)
	} else {
		ctx = workflow.WithChildOptions(ctx, cwo)
	}
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

// ClosureCapture: a context assigned inside a closure is poisoned -- the closure
// may reconfigure it at an unknown time -- so it is never fired on.
func ClosureCapture(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	workflow.Go(ctx, func(c workflow.Context) { ctx = c })
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

// NonEntry: a call to a workflow.* function that is not an Execute* entry point
// is ignored, even on a configured context.
func NonEntry(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	workflow.GetLogger(ctx)
	return nil
}

// FieldContext: a context held in a struct field is not a plain variable, so it
// is not tracked -- neither the With* derivation nor the Execute* call.
func FieldContext(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	var h holder
	h.ctx = workflow.WithChildOptions(ctx, cwo)
	workflow.ExecuteActivity(h.ctx, greet)
	return nil
}

// ChildCorrect: a sibling child context configured and used correctly.
func ChildCorrect(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	cctx := workflow.WithChildOptions(ctx, cwo)
	workflow.ExecuteChildWorkflow(cctx, childWf)
	return nil
}

// MultiAssign: a multi-value assignment to the context is opaque, so a prior
// conflict no longer fires.
func MultiAssign(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	var err error
	ctx = workflow.WithChildOptions(ctx, cwo)
	ctx, err = twoValueCtx(ctx)
	_ = err
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

// AssignFromVar: assigning a context from another variable (not a With* call) is
// opaque, so the target's options become unknown.
func AssignFromVar(ctx workflow.Context, other workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	ctx = other
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

// ResetAfterLoop: a kind assigned only inside a loop body does not escape it; the
// call after the loop sees an unknown context.
func ResetAfterLoop(ctx workflow.Context, n int) error {
	var cwo workflow.ChildWorkflowOptions
	for i := 0; i < n; i++ {
		ctx = workflow.WithChildOptions(ctx, cwo)
	}
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

// ResetAfterRange / ResetAfterSwitch / ResetAfterSelect mirror ResetAfterLoop for
// the remaining control-flow joins.
func ResetAfterRange(ctx workflow.Context, items []int) error {
	var cwo workflow.ChildWorkflowOptions
	for k, v := range items {
		_, _ = k, v
		ctx = workflow.WithChildOptions(ctx, cwo)
	}
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

func ResetAfterSwitch(ctx workflow.Context, x int) error {
	var cwo workflow.ChildWorkflowOptions
	switch x {
	case 1:
		ctx = workflow.WithChildOptions(ctx, cwo)
	}
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

func ResetAfterSelect(ctx workflow.Context, ch chan int) error {
	var cwo workflow.ChildWorkflowOptions
	select {
	case <-ch:
		ctx = workflow.WithChildOptions(ctx, cwo)
	default:
	}
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

// ExecuteInClosure: an Execute* call inside a closure is out of scope (the walk
// does not descend into function literals), so even a conflicting captured
// context is not flagged there.
func ExecuteInClosure(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	workflow.Go(ctx, func(c workflow.Context) {
		workflow.ExecuteActivity(ctx, greet)
	})
	return nil
}

// ClosureFieldAssign: a closure that assigns to a struct field (not a plain
// variable) does not poison any tracked context, and the field write itself is
// not trackable; the later activity call sees an unconfigured parameter.
func ClosureFieldAssign(ctx workflow.Context) error {
	var h holder
	workflow.Go(ctx, func(c workflow.Context) { h.ctx = c })
	workflow.ExecuteActivity(ctx, greet)
	return nil
}

// NotASelector: a call whose callee is a bare identifier (a builtin) or a
// func-typed field is not a workflow function, so it is ignored.
func NotASelector(ctx workflow.Context) error {
	var h holder
	_ = len("x")
	h.run()
	_ = strings.ToUpper("y")
	return nil
}
