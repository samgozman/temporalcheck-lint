// Package workflowstate implements a static check for the Temporal Go SDK.
//
// A Temporal workflow must be deterministic: replay re-executes its code against
// recorded history and must produce the same commands. Worker processes also run
// many workflow executions concurrently. Mutating a package-level variable from
// workflow code breaks both guarantees -- the mutation is not part of the replay
// state, and the variable is shared across every execution in the worker, so the
// writes race. Temporal's own workflowcheck tool documents this exact gap: "this
// will not catch all cases of non-determinism such as global var mutation".
//
// This analyzer fills it, narrowly. Inside any workflow definition (a function
// whose first parameter is workflow.Context) -- including closures lexically
// nested in it, such as workflow.Go goroutines and Selector callbacks -- it
// resolves the root object of every assignment and ++/-- and reports the ones
// rooted at a package-level variable (this package or another).
//
// It deliberately does NOT flag mutation of a captured local. Capturing a local
// from the enclosing workflow function and writing it from a workflow.Go /
// Await / Selector callback is the SDK's documented idiom for moving data
// between deterministic coroutines, so flagging it would be near-100% false
// positive. The discriminator is the variable's scope: package-level is flagged,
// a local or parameter (the capture case) is not. Mutations whose root cannot be
// resolved to a plain variable (a call result, an opaque receiver) are skipped
// rather than guessed at. That keeps the check near-zero-false-positive, so it is
// on by default.
package workflowstate

import (
	"go/ast"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/workflowscope"
	"golang.org/x/tools/go/analysis"
)

const (
	// tagGlobalMutation suffixes the diagnostic so it is clear which check
	// produced it.
	tagGlobalMutation = "global-mutation"
)

// Settings configures the workflowstate analyzer.
type Settings struct {
	// Disabled turns the analyzer off entirely; it reports nothing. The check is
	// on by default: mutating shared package state from a workflow breaks replay
	// determinism and races across executions, which is never legitimate, so there
	// is nothing to opt into.
	Disabled bool
}

// NewAnalyzer builds the workflowstate analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{disabled: settings.Disabled}
	return &analysis.Analyzer{
		Name: "workflowstate",
		Doc:  "flag mutation of a package-level variable from Temporal workflow code, which breaks replay determinism and races across concurrent workflow executions",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk so the analyzer
// stays free of package-level mutable state.
type checker struct {
	disabled bool
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	if c.disabled {
		return nil, nil
	}
	for _, file := range pass.Files {
		nolint := nolint.Collect(pass.Fset, file)
		workflowscope.Walk(pass, file, func(body *ast.BlockStmt) {
			c.reportMutations(pass, nolint, body)
		})
	}
	return nil, nil
}
