package workflowstate

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"
)

// TestNolintForWorkflowstate covers the directive-parsing edges: a bare
// directive, all/temporalcheck lists, a trailing explanation, and the strings
// that must NOT count (a non-directive comment, "//nolintfoo", another linter,
// the analyzer name).
func TestNolintForWorkflowstate(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"//nolint", true},
		{"//nolint:all", true},
		{"//nolint:temporalcheck", true},
		{"//nolint:temporalcheck // discarded on purpose", true},
		{"//nolint:govet,temporalcheck", true},
		{"// ordinary comment", false},
		{"//nolintfoo", false},
		{"//nolint:govet", false},
		{"//nolint:workflowstate", false},
	}
	for _, tt := range tests {
		if got := nolintForWorkflowstate(tt.text); got != tt.want {
			t.Errorf("nolintForWorkflowstate(%q) = %v, want %v", tt.text, got, tt.want)
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

// TestNamed covers the matcher's branches directly: a matching named type, a
// name mismatch, a package mismatch, a type from the universe scope (nil pkg),
// and a non-named type.
func TestNamed(t *testing.T) {
	pkg := types.NewPackage("example.com/pkg", "pkg")
	mkNamed := func(name string) types.Type {
		obj := types.NewTypeName(token.NoPos, pkg, name, nil)
		return types.NewNamed(obj, types.Typ[types.Int], nil)
	}

	if !named(mkNamed("Context"), "example.com/pkg", "Context") {
		t.Error("named() should match same package and name")
	}
	// An alias type (type Context = internal.Context) is matched on its own object.
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
	// types.Error is a named type whose object has no package (universe scope).
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
