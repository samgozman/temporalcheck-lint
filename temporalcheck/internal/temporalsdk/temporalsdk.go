// Package temporalsdk centralizes how the analyzers recognize the Temporal Go
// SDK through go/types. The analyzers never import the SDK; they match calls and
// types by package path, so the SDK's import paths and the type-matching helpers
// that resolve them live here once instead of being copied into each analyzer.
package temporalsdk

import "go/types"

// SDK (and stdlib context) import paths the analyzers match against.
const (
	// WorkflowPkg is the public workflow package.
	WorkflowPkg = "go.temporal.io/sdk/workflow"
	// InternalPkg is where the SDK actually declares the workflow types. The
	// public workflow.Context is published as `type Context = internal.Context`,
	// so depending on the gotypesalias mode a parameter's resolved type surfaces
	// in this package, not in WorkflowPkg. Matchers must accept both.
	InternalPkg = "go.temporal.io/sdk/internal"
	// ClientPkg is where the SDK declares the Client interface directly (unlike
	// the aliased Context type), so client.ExecuteWorkflow's receiver resolves to
	// client.Client. InternalPkg.Client is a separate interface client.Client
	// implements; matchers accept it too, defensively.
	ClientPkg = "go.temporal.io/sdk/client"
	// ConverterPkg holds converter.EncodedValue and friends.
	ConverterPkg = "go.temporal.io/sdk/converter"
	// WorkerPkg holds worker.Options and the worker constructors.
	WorkerPkg = "go.temporal.io/sdk/worker"
	// ContextPkg is the standard library context package, whose Context is the
	// first parameter Temporal injects into an activity (workflows take
	// workflow.Context instead).
	ContextPkg = "context"
)

// Named reports whether t is the named type pkgPath.name. It accepts both defined
// types and aliases, since both carry a *types.TypeName.
func Named(t types.Type, pkgPath, name string) bool {
	var obj *types.TypeName
	switch n := t.(type) {
	case *types.Named:
		obj = n.Obj()
	case *types.Alias:
		obj = n.Obj()
	default:
		return false
	}
	return obj != nil && obj.Pkg() != nil &&
		obj.Pkg().Path() == pkgPath && obj.Name() == name
}

// Deref returns the element type of a pointer, or t unchanged otherwise.
func Deref(t types.Type) types.Type {
	if p, ok := t.Underlying().(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}

// IsReceiver reports whether fn is a method whose receiver is (a pointer to) the
// named type pkgPath.name.
func IsReceiver(fn *types.Func, pkgPath, name string) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return false
	}
	return Named(Deref(sig.Recv().Type()), pkgPath, name)
}

// IsWorkflowContext reports whether t is workflow.Context. The SDK publishes it
// as `type Context = internal.Context`, so depending on the gotypesalias mode t
// is either the alias (named in WorkflowPkg) or the resolved internal named type
// (named in InternalPkg); both count as the injected context.
func IsWorkflowContext(t types.Type) bool {
	if t == nil {
		return false
	}
	if Named(types.Unalias(t), InternalPkg, "Context") {
		return true
	}
	return Named(t, WorkflowPkg, "Context")
}

// SkipCount returns how many leading parameters Temporal injects at run time --
// the context the caller must not supply. A workflow takes workflow.Context, an
// activity takes the standard context.Context; either counts as one injected
// parameter when present as the first one.
func SkipCount(sig *types.Signature, isWorkflow bool) int {
	if sig.Params().Len() == 0 {
		return 0
	}
	first := sig.Params().At(0).Type()
	if isWorkflow {
		if IsWorkflowContext(first) {
			return 1
		}
		return 0
	}
	if Named(first, ContextPkg, "Context") {
		return 1
	}
	return 0
}
