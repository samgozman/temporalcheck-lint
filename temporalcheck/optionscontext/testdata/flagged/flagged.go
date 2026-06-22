// Package flagged exercises the seen-contradiction shapes the analyzer reports:
// a context configured with one kind of options helper but fed to an Execute*
// call that reads a different kind, with no matching helper in the visible chain.
package flagged

import (
	"go.temporal.io/sdk/workflow"

	wf "go.temporal.io/sdk/workflow"
)

func greet(ctx workflow.Context) error  { return nil }
func childWf(ctx workflow.Context) error { return nil }

func Workflow(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions

	// The classic bug: child options applied, activity call -- the activity options
	// never apply.
	ctx = workflow.WithChildOptions(ctx, cwo)
	workflow.ExecuteActivity(ctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`

	return nil
}

// ViceVersa: activity options applied, child-workflow call.
func ViceVersa(ctx workflow.Context) error {
	var ao workflow.ActivityOptions
	ctx = workflow.WithActivityOptions(ctx, ao)
	workflow.ExecuteChildWorkflow(ctx, childWf) // want `ExecuteChildWorkflow: this ctx is configured with WithActivityOptions, not WithChildOptions, so the child workflow options never apply; derive it with ctx = workflow.WithChildOptions\(ctx, opts\) \(options-context\)`
	return nil
}

// LocalMixup: activity options applied, local-activity call -- a distinct context
// key, so still a conflict.
func LocalMixup(ctx workflow.Context) error {
	var ao workflow.ActivityOptions
	ctx = workflow.WithActivityOptions(ctx, ao)
	workflow.ExecuteLocalActivity(ctx, greet) // want `ExecuteLocalActivity: this ctx is configured with WithActivityOptions, not WithLocalActivityOptions, so the local activity options never apply; derive it with ctx = workflow.WithLocalActivityOptions\(ctx, opts\) \(options-context\)`
	return nil
}

// WrongVariable: the activity context is configured correctly, but a sibling
// child context is passed to the activity call by mistake.
func WrongVariable(ctx workflow.Context) error {
	var (
		ao  workflow.ActivityOptions
		cwo workflow.ChildWorkflowOptions
	)
	actx := workflow.WithActivityOptions(ctx, ao)
	cctx := workflow.WithChildOptions(ctx, cwo)
	workflow.ExecuteActivity(cctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`
	_ = actx
	return nil
}

// InBranch: the conflicting helper is applied before a branch, so a call inside
// the branch still sees it.
func InBranch(ctx workflow.Context, cond bool) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	if cond {
		workflow.ExecuteActivity(ctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`
	}
	return nil
}

// AsRHS: the Execute* call is the right-hand side of an assignment; it reads the
// pre-assignment context, which carries the conflicting options.
func AsRHS(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	f := workflow.ExecuteActivity(ctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`
	_ = f
	return nil
}

// Aliased: an aliased import of the workflow package resolves to the same
// functions, so the conflict is flagged just the same.
func Aliased(ctx workflow.Context) error {
	var cwo wf.ChildWorkflowOptions
	ctx = wf.WithChildOptions(ctx, cwo)
	wf.ExecuteActivity(ctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`
	return nil
}

// The constructs below all carry the conflicting context into a control-flow
// body: the entry-state options are still in force when the call runs.

func InIfWithInit(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	if x := 1; x > 0 {
		workflow.ExecuteActivity(ctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`
	}
	return nil
}

func InSwitchWithInit(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	switch x := 1; x {
	case 1:
		workflow.ExecuteActivity(ctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`
	}
	return nil
}

func InForLoop(ctx workflow.Context, n int) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	for i := 0; i < n; i++ {
		workflow.ExecuteActivity(ctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`
	}
	return nil
}

func InRangeLoop(ctx workflow.Context, items []int) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	for range items {
		workflow.ExecuteActivity(ctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`
	}
	return nil
}

func InSwitch(ctx workflow.Context, x int) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	switch x {
	case 1:
		workflow.ExecuteActivity(ctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`
	}
	return nil
}

func InTypeSwitch(ctx workflow.Context, v any) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	switch n := 0; v.(type) {
	case int:
		_ = n
		workflow.ExecuteActivity(ctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`
	}
	return nil
}

func InSelect(ctx workflow.Context, ch chan int) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
	select {
	case <-ch:
		workflow.ExecuteActivity(ctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`
	}
	return nil
}

// InLabeledBlock: a labeled statement and a bare nested block are transparent to
// the walk, so the carried conflict still fires.
func InLabeledBlock(ctx workflow.Context) error {
	var cwo workflow.ChildWorkflowOptions
	ctx = workflow.WithChildOptions(ctx, cwo)
loop:
	for {
		{
			workflow.ExecuteActivity(ctx, greet) // want `ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-context\)`
			break loop
		}
	}
	return nil
}
