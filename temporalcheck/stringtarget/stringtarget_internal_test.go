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
