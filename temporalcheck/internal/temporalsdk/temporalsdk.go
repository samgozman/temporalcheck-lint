// Package temporalsdk centralizes how analyzers recognize the Temporal Go SDK
// through go/types. Analyzers match by package path, never by importing the SDK.
package temporalsdk

import "go/types"

// SDK (and stdlib context) import paths the analyzers match against.
const (
	WorkflowPkg  = "go.temporal.io/sdk/workflow"
	// InternalPkg is where the SDK declares its types. workflow.Context is published as
	// `type Context = internal.Context`, so depending on gotypesalias mode the resolved
	// type may surface here; matchers must accept both.
	InternalPkg  = "go.temporal.io/sdk/internal"
	ClientPkg    = "go.temporal.io/sdk/client"
	ConverterPkg = "go.temporal.io/sdk/converter"
	WorkerPkg    = "go.temporal.io/sdk/worker"
	// ContextPkg is the stdlib context; activities take context.Context, not workflow.Context.
	ContextPkg   = "context"
)

// Named reports whether t is the named type pkgPath.name (defined type or alias).
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

// IsReceiver reports whether fn is a method whose receiver is (a pointer to) pkgPath.name.
func IsReceiver(fn *types.Func, pkgPath, name string) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return false
	}
	return Named(Deref(sig.Recv().Type()), pkgPath, name)
}

// IsWorkflowContext reports whether t is workflow.Context.
// The SDK publishes it as `type Context = internal.Context`, so t may resolve
// to either the alias (WorkflowPkg) or the underlying named type (InternalPkg).
func IsWorkflowContext(t types.Type) bool {
	if t == nil {
		return false
	}
	if Named(types.Unalias(t), InternalPkg, "Context") {
		return true
	}
	return Named(t, WorkflowPkg, "Context")
}

// SkipCount returns how many leading parameters Temporal injects at run time (0 or 1):
// a workflow takes workflow.Context; an activity takes context.Context.
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
