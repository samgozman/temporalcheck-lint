// Package flagged exercises the mutations the analyzer reports: assignments and
// ++/-- whose root object is a package-level variable, reached directly, through
// a field, an index, or a cross-package selector, and from inside a workflow.Go
// closure, an Await/Selector callback, and a method workflow.
package flagged

import (
	"go.temporal.io/sdk/workflow"

	"workflowstatefixtures/shared"
)

// Package-level state -- mutating any of this from workflow code is the hazard.
var (
	counter int
	cfg     config
	cache   = map[string]int{}
	items   []int
	ptr     = &counter
)

type config struct{ Retries int }

func Workflow(ctx workflow.Context) error {
	// Bare scalar assignment and increment/decrement.
	counter = 5  // want `mutates package-level variable counter from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`
	counter++    // want `mutates package-level variable counter from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`
	counter--    // want `mutates package-level variable counter from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`
	counter += 2 // want `mutates package-level variable counter from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`

	// Field of a package-level struct: the root object is still the global.
	cfg.Retries = 3 // want `mutates package-level variable cfg from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`

	// Index into a package-level map and slice.
	cache["k"] = 1 // want `mutates package-level variable cache from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`
	items[0] = 9   // want `mutates package-level variable items from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`

	// Reassigning the global itself (append result) is a mutation of the global.
	items = append(items, 1) // want `mutates package-level variable items from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`

	// Writing through a global pointer; the root object is still the global ptr.
	*ptr = 7 // want `mutates package-level variable ptr from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`

	// A parenthesized target peels to the same global.
	(counter) = 0 // want `mutates package-level variable counter from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`

	// Cross-package globals, reached through a qualified selector.
	shared.Global++          // want `mutates package-level variable Global from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`
	shared.Registry["x"] = 2 // want `mutates package-level variable Registry from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`

	// A workflow.Go coroutine is still workflow code: a global mutated inside its
	// closure is flagged (unlike a captured local, see the benign fixture).
	workflow.Go(ctx, func(ctx workflow.Context) {
		counter++ // want `mutates package-level variable counter from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`
	})

	// An Await condition closure that writes a global (not just reads one).
	_ = workflow.Await(ctx, func() bool {
		counter++ // want `mutates package-level variable counter from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`
		return counter > 10
	})

	return nil
}

type App struct{}

// A method workflow (first parameter workflow.Context) is a workflow definition
// too, so global mutation inside it is flagged.
func (a *App) Run(ctx workflow.Context) error {
	counter += 100 // want `mutates package-level variable counter from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`
	return nil
}

// register is not a workflow definition, but the function literal it registers
// is (its first parameter is workflow.Context), so a global mutated inside that
// literal is flagged -- the analyzer reaches workflow definitions nested in
// ordinary functions, not only top-level ones.
func register() {
	run := func(ctx workflow.Context) error {
		counter++ // want `mutates package-level variable counter from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`
		return nil
	}
	_ = run
}
