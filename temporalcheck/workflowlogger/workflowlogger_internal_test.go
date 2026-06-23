package workflowlogger

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"
)

// TestNolintForWorkflowlogger covers the directive-parsing edges: a bare
// directive, all/temporalcheck lists, a trailing explanation, and the strings
// that must NOT count (a non-directive comment, "//nolintfoo", another linter,
// the analyzer name).
func TestNolintForWorkflowlogger(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"//nolint", true},
		{"//nolint:all", true},
		{"//nolint:temporalcheck", true},
		{"//nolint:temporalcheck // disabled on purpose", true},
		{"//nolint:govet,temporalcheck", true},
		{"// ordinary comment", false},
		{"//nolintfoo", false},
		{"//nolint:govet", false},
		{"//nolint:workflowlogger", false},
	}
	for _, tt := range tests {
		if got := nolintForWorkflowlogger(tt.text); got != tt.want {
			t.Errorf("nolintForWorkflowlogger(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

// TestIsWorkflowContext_Nil guards the nil-type path: an unresolved parameter
// type must not be treated as a workflow context.
func TestIsWorkflowContext_Nil(t *testing.T) {
	if isWorkflowContext(nil) {
		t.Error("isWorkflowContext(nil) = true, want false")
	}
}

// TestNamed covers the matcher's branches directly: a matching named type, an
// alias matched on its own object, a name mismatch, a package mismatch, a type
// from the universe scope (nil pkg), and a non-named type.
func TestNamed(t *testing.T) {
	pkg := types.NewPackage("example.com/pkg", "pkg")
	mkNamed := func(name string) types.Type {
		obj := types.NewTypeName(token.NoPos, pkg, name, nil)
		return types.NewNamed(obj, types.Typ[types.Int], nil)
	}

	if !named(mkNamed("Context"), "example.com/pkg", "Context") {
		t.Error("named() should match same package and name")
	}
	aliasObj := types.NewTypeName(token.NoPos, pkg, "Context", nil)
	if !named(types.NewAlias(aliasObj, types.Typ[types.Int]), "example.com/pkg", "Context") {
		t.Error("named() should match an alias by its own package and name")
	}
	if named(mkNamed("Other"), "example.com/pkg", "Context") {
		t.Error("named() should reject a name mismatch")
	}
	if named(mkNamed("Context"), "other/pkg", "Context") {
		t.Error("named() should reject a package mismatch")
	}
	if named(types.Universe.Lookup("error").Type(), "example.com/pkg", "Context") {
		t.Error("named() should reject a type with no package")
	}
	if named(types.Typ[types.String], "example.com/pkg", "Context") {
		t.Error("named() should reject a non-named type")
	}
}

// TestFuncBody covers the non-function branch: any other node yields no body.
func TestFuncBody(t *testing.T) {
	if body, ft := funcBody(&ast.Ident{}); body != nil || ft != nil {
		t.Errorf("funcBody(non-func) = (%v, %v), want (nil, nil)", body, ft)
	}
}

// TestWritesToStdStream_NoArgs covers the defensive guard for a Fprint* call with
// no arguments (only reachable on a malformed/partial AST, never in compiling
// source). The guard returns before touching the pass, so a nil pass is safe.
func TestWritesToStdStream_NoArgs(t *testing.T) {
	if writesToStdStream(nil, &ast.CallExpr{}) {
		t.Error("writesToStdStream(call with no args) = true, want false")
	}
}

// TestCalleeFunc_NonFuncCallee covers the branch where a call's callee is neither
// a selector nor an identifier (an immediately-invoked function literal); it has
// no named function to resolve, so calleeFunc returns nil without using the pass.
func TestCalleeFunc_NonFuncCallee(t *testing.T) {
	if fn := calleeFunc(nil, &ast.CallExpr{Fun: &ast.FuncLit{}}); fn != nil {
		t.Errorf("calleeFunc(IIFE) = %v, want nil", fn)
	}
}
