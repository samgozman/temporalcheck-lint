package optionsdiscard

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestIsBlank(t *testing.T) {
	tests := []struct {
		name string
		expr ast.Expr
		want bool
	}{
		{"blank identifier", &ast.Ident{Name: "_"}, true},
		{"named identifier", &ast.Ident{Name: "ctx"}, false},
		{"not an identifier", &ast.BasicLit{Kind: token.STRING, Value: `"_"`}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBlank(tt.expr); got != tt.want {
				t.Errorf("isBlank(%v) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}
