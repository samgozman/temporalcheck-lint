package temporalsdk

import (
	"go/token"
	"go/types"
	"testing"
)

// mkNamed builds a defined type pkgPath.name for matcher tests.
func mkNamed(pkgPath, pkgName, name string) types.Type {
	pkg := types.NewPackage(pkgPath, pkgName)
	obj := types.NewTypeName(token.NoPos, pkg, name, nil)
	return types.NewNamed(obj, types.Typ[types.Int], nil)
}

func TestNamed(t *testing.T) {
	pkg := types.NewPackage("example.com/pkg", "pkg")
	mk := func(name string) types.Type {
		obj := types.NewTypeName(token.NoPos, pkg, name, nil)
		return types.NewNamed(obj, types.Typ[types.Int], nil)
	}

	if !Named(mk("Context"), "example.com/pkg", "Context") {
		t.Error("Named() should match same package and name")
	}
	// An alias type (type Context = internal.Context) is matched on its own object.
	aliasObj := types.NewTypeName(token.NoPos, pkg, "Context", nil)
	if !Named(types.NewAlias(aliasObj, types.Typ[types.Int]), "example.com/pkg", "Context") {
		t.Error("Named() should match an alias by its own package and name")
	}
	if Named(mk("Other"), "example.com/pkg", "Context") {
		t.Error("Named() should reject a name mismatch")
	}
	if Named(mk("Context"), "other/pkg", "Context") {
		t.Error("Named() should reject a package mismatch")
	}
	// error is a named type whose object has no package (universe scope).
	if Named(types.Universe.Lookup("error").Type(), "example.com/pkg", "Context") {
		t.Error("Named() should reject a type with no package")
	}
	if Named(types.Typ[types.String], "example.com/pkg", "Context") {
		t.Error("Named() should reject a non-named type")
	}
}

func TestDeref(t *testing.T) {
	base := mkNamed("example.com/pkg", "pkg", "T")
	if got := Deref(types.NewPointer(base)); got != base {
		t.Errorf("Deref(*T) = %v, want T", got)
	}
	if got := Deref(base); got != base {
		t.Errorf("Deref(T) = %v, want T (unchanged)", got)
	}
}

func TestIsReceiver(t *testing.T) {
	recvType := mkNamed("example.com/pkg", "pkg", "Env")
	method := func(recv *types.Var) *types.Func {
		sig := types.NewSignatureType(recv, nil, nil, types.NewTuple(), types.NewTuple(), false)
		return types.NewFunc(token.NoPos, nil, "M", sig)
	}

	// Value receiver matching the named type.
	v := types.NewVar(token.NoPos, nil, "e", recvType)
	if !IsReceiver(method(v), "example.com/pkg", "Env") {
		t.Error("IsReceiver should match a value receiver of the named type")
	}
	// Pointer receiver is dereferenced before matching.
	pv := types.NewVar(token.NoPos, nil, "e", types.NewPointer(recvType))
	if !IsReceiver(method(pv), "example.com/pkg", "Env") {
		t.Error("IsReceiver should match a pointer receiver of the named type")
	}
	// A function (no receiver) does not match.
	if IsReceiver(method(nil), "example.com/pkg", "Env") {
		t.Error("IsReceiver should reject a function with no receiver")
	}
	// Receiver of a different type does not match.
	other := types.NewVar(token.NoPos, nil, "e", mkNamed("example.com/pkg", "pkg", "Other"))
	if IsReceiver(method(other), "example.com/pkg", "Env") {
		t.Error("IsReceiver should reject a receiver of a different type")
	}
}

func TestIsWorkflowContext(t *testing.T) {
	if IsWorkflowContext(nil) {
		t.Error("IsWorkflowContext(nil) should be false")
	}
	if !IsWorkflowContext(mkNamed(InternalPkg, "internal", "Context")) {
		t.Error("IsWorkflowContext should match internal.Context")
	}
	if !IsWorkflowContext(mkNamed(WorkflowPkg, "workflow", "Context")) {
		t.Error("IsWorkflowContext should match workflow.Context")
	}
	if IsWorkflowContext(mkNamed(ContextPkg, "context", "Context")) {
		t.Error("IsWorkflowContext should not match the stdlib context.Context")
	}
}

func TestSkipCount(t *testing.T) {
	sig := func(first types.Type) *types.Signature {
		var params *types.Tuple
		if first == nil {
			params = types.NewTuple()
		} else {
			params = types.NewTuple(types.NewVar(token.NoPos, nil, "ctx", first))
		}
		return types.NewSignatureType(nil, nil, nil, params, types.NewTuple(), false)
	}

	if got := SkipCount(sig(nil), true); got != 0 {
		t.Errorf("SkipCount(no params) = %d, want 0", got)
	}
	if got := SkipCount(sig(mkNamed(WorkflowPkg, "workflow", "Context")), true); got != 1 {
		t.Errorf("SkipCount(workflow.Context, isWorkflow) = %d, want 1", got)
	}
	if got := SkipCount(sig(types.Typ[types.Int]), true); got != 0 {
		t.Errorf("SkipCount(non-context, isWorkflow) = %d, want 0", got)
	}
	if got := SkipCount(sig(mkNamed(ContextPkg, "context", "Context")), false); got != 1 {
		t.Errorf("SkipCount(context.Context, activity) = %d, want 1", got)
	}
	if got := SkipCount(sig(types.Typ[types.Int]), false); got != 0 {
		t.Errorf("SkipCount(non-context, activity) = %d, want 0", got)
	}
}
