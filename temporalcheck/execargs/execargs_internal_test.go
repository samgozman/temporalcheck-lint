package execargs

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestArgWord(t *testing.T) {
	if got := argWord(1); got != "argument" {
		t.Errorf("argWord(1) = %q, want %q", got, "argument")
	}
	for _, n := range []int{0, 2, 5} {
		if got := argWord(n); got != "arguments" {
			t.Errorf("argWord(%d) = %q, want %q", n, got, "arguments")
		}
	}
}

func TestNoun(t *testing.T) {
	if got := noun(kindActivity); got != "activity" {
		t.Errorf("noun(kindActivity) = %q, want %q", got, "activity")
	}
	if got := noun(kindChildWorkflow); got != "child workflow" {
		t.Errorf("noun(kindChildWorkflow) = %q, want %q", got, "child workflow")
	}
}

func TestTargetName(t *testing.T) {
	tests := []struct {
		name string
		expr ast.Expr
		want string
	}{
		{"selector", &ast.SelectorExpr{Sel: &ast.Ident{Name: "Greet"}}, "Greet"},
		{"ident", &ast.Ident{Name: "ArchiveAll"}, "ArchiveAll"},
		{"fallback", &ast.BasicLit{Kind: token.STRING, Value: `"x"`}, "target"},
	}
	for _, tt := range tests {
		if got := targetName(tt.expr); got != tt.want {
			t.Errorf("%s: targetName = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestNewAnalyzerMetadata(t *testing.T) {
	a := NewAnalyzer(Settings{StrictTypes: true})
	if a.Name != "execargs" {
		t.Errorf("analyzer name = %q, want %q", a.Name, "execargs")
	}
	if a.Run == nil {
		t.Error("analyzer Run is nil")
	}
}
