// Package benign collects the patterns the analyzer must NOT flag: the
// SDK-documented capture-and-mutate-a-local idiom (workflow.Go, Await, Selector
// callbacks), mutation of a parameter or local, a read of a global, a `:=` that
// declares a fresh local, and a global mutated from a non-workflow function.
package benign

import "go.temporal.io/sdk/workflow"

var counter int

type input struct{ N int }

func Workflow(ctx workflow.Context, in *input) error {
	// Capturing a local and writing it from a workflow.Go coroutine is the
	// documented way to move data between deterministic coroutines -- not flagged.
	total := 0
	workflow.Go(ctx, func(ctx workflow.Context) {
		total += 1
		total++
	})

	// Await reads a captured local in its condition.
	_ = workflow.Await(ctx, func() bool { return total == 5 })

	// A Selector callback writes a captured local result -- the canonical idiom.
	var result string
	sel := workflow.NewSelector(ctx)
	sel.AddFuture(workflow.ExecuteActivity(ctx, "A"), func(f workflow.Future) {
		result = "done"
	})
	sel.Select(ctx)
	_ = result

	// Mutating a parameter (caller-owned memory) and its fields is fine.
	in.N = 7
	in = nil

	// Mutating a local struct/map/slice is fine.
	local := input{}
	local.N = 1
	m := map[string]int{}
	m["k"] = 2

	// A target rooted at a call result cannot be resolved to a named variable, so
	// it is skipped rather than guessed at.
	provide().N = 3

	// Reading a global is not a mutation.
	_ = counter

	// `:=` declares a new local that shadows nothing global.
	counter := 0
	counter++
	_ = counter

	return nil
}

// provide returns a fresh value; a mutation of its result is not a global.
func provide() *input { return &input{} }

// helper is not a workflow definition (no workflow.Context parameter), so a
// global mutation here is out of scope and not flagged.
func helper() {
	counter++
}

// notWorkflow has a leading parameter, but a non-context one, so it is not a
// workflow definition and its global mutation is out of scope. It also exercises
// the first-parameter type check against a basic type and a non-context named
// type.
func notWorkflow(s string) {
	counter++
}

func alsoNotWorkflow(in input) {
	counter++
}
