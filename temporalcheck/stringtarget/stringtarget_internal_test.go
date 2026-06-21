package stringtarget

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"
)

func TestIsStringType(t *testing.T) {
	named := types.NewNamed(
		types.NewTypeName(token.NoPos, nil, "ActivityName", nil),
		types.Typ[types.String], nil,
	)

	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{"nil", nil, false},
		{"string", types.Typ[types.String], true},
		{"untyped string", types.Typ[types.UntypedString], true},
		{"named string", named, true},
		{"int", types.Typ[types.Int], false},
		{"slice", types.NewSlice(types.Typ[types.String]), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStringType(tt.typ); got != tt.want {
				t.Errorf("isStringType(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestIsReceiver(t *testing.T) {
	const pkgPath = "go.temporal.io/sdk/internal"
	pkg := types.NewPackage(pkgPath, "internal")

	mkMethod := func(recv types.Type) *types.Func {
		var recvVar *types.Var
		if recv != nil {
			recvVar = types.NewVar(token.NoPos, nil, "e", recv)
		}
		sig := types.NewSignatureType(recvVar, nil, nil, types.NewTuple(), types.NewTuple(), false)
		return types.NewFunc(token.NoPos, pkg, "OnActivity", sig)
	}
	named := func(name string) *types.Named {
		return types.NewNamed(types.NewTypeName(token.NoPos, pkg, name, nil), types.NewStruct(nil, nil), nil)
	}

	env := named("TestWorkflowEnvironment")
	tests := []struct {
		name string
		recv types.Type // nil means a non-method func
		want bool
	}{
		{"pointer to env", types.NewPointer(env), true},
		{"value env", env, true},
		{"not a method", nil, false},
		{"non-named receiver", types.Typ[types.Int], false},
		{"wrong type name", named("Other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReceiver(mkMethod(tt.recv), pkgPath, "TestWorkflowEnvironment"); got != tt.want {
				t.Errorf("isReceiver(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestLiteralName(t *testing.T) {
	tests := []struct {
		name     string
		expr     ast.Expr
		wantName string
		wantOK   bool
	}{
		{"string literal", &ast.BasicLit{Kind: token.STRING, Value: `"Greet"`}, "Greet", true},
		{"int literal", &ast.BasicLit{Kind: token.INT, Value: "42"}, "", false},
		{"unparseable string literal", &ast.BasicLit{Kind: token.STRING, Value: `"unterminated`}, "", false},
		{"not a literal", &ast.Ident{Name: "name"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := literalName(tt.expr)
			if got != tt.wantName || ok != tt.wantOK {
				t.Errorf("literalName(%v) = (%q, %v), want (%q, %v)", tt.expr, got, ok, tt.wantName, tt.wantOK)
			}
		})
	}
}
