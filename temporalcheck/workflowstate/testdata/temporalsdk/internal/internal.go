// Package internal is a stand-in for go.temporal.io/sdk/internal. The real SDK
// declares Context, Future and Selector here and re-publishes them from the
// workflow package as aliases (type Context = internal.Context). The fixtures
// reproduce that shape so the analyzer is exercised against the SDK's real type
// identities: a workflow.Context parameter resolves to internal.Context, not a
// fresh type declared in the workflow stub. The workflow-scope detection must
// resolve that alias to match a workflow definition.
package internal

// Context mirrors the SDK's workflow context, the first parameter that marks a
// function as a workflow definition.
type Context interface{ isWorkflowContext() }

// Future is the minimal shape needed for the Selector.AddFuture idiom in the
// fixtures, where a captured local is written from the callback.
type Future interface {
	Get(ctx Context, valuePtr interface{}) error
}

// Selector mirrors workflow.Selector: AddFuture registers a callback the
// deterministic runner invokes, the canonical place a captured local is mutated.
type Selector interface {
	AddFuture(future Future, f func(f Future)) Selector
	Select(ctx Context)
}
