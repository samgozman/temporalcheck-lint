package optionscontext

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestNolintForOptionsContext(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"not a nolint comment", "// just a comment", false},
		{"bare nolint", "//nolint", true},
		{"nolint with no colon", "//nolintfoo", false},
		{"nolint all", "//nolint:all", true},
		{"names the plugin", "//nolint:temporalcheck", true},
		{"names plugin with explanation", "//nolint:temporalcheck // on purpose", true},
		{"names another linter only", "//nolint:gocritic", false},
		{"names the analyzer, not the plugin", "//nolint:optionscontext", false},
		{"plugin among several", "//nolint:gocritic,temporalcheck", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nolintForOptionsContext(tt.text); got != tt.want {
				t.Errorf("nolintForOptionsContext(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestKindMetadata(t *testing.T) {
	tests := []struct {
		k      kind
		helper string
		noun   string
	}{
		{kindActivity, "WithActivityOptions", "activity"},
		{kindLocalActivity, "WithLocalActivityOptions", "local activity"},
		{kindChild, "WithChildOptions", "child workflow"},
	}
	for _, tt := range tests {
		if got := tt.k.helper(); got != tt.helper {
			t.Errorf("kind(%d).helper() = %q, want %q", tt.k, got, tt.helper)
		}
		if got := tt.k.noun(); got != tt.noun {
			t.Errorf("kind(%d).noun() = %q, want %q", tt.k, got, tt.noun)
		}
		var s kindSet
		if s.has(tt.k) {
			t.Errorf("empty kindSet should not contain kind(%d)", tt.k)
		}
		s |= tt.k.bit()
		if !s.has(tt.k) {
			t.Errorf("kindSet should contain kind(%d) after setting its bit", tt.k)
		}
	}
}

func TestAssignTargets(t *testing.T) {
	x := &ast.Ident{Name: "x"}
	y := &ast.Ident{Name: "y"}

	assign := &ast.AssignStmt{Lhs: []ast.Expr{x, y}}
	if got := assignTargets(assign); len(got) != 2 {
		t.Errorf("assignTargets(AssignStmt) = %d targets, want 2", len(got))
	}

	inc := &ast.IncDecStmt{X: x}
	if got := assignTargets(inc); len(got) != 1 {
		t.Errorf("assignTargets(IncDecStmt) = %d targets, want 1", len(got))
	}

	rangeBoth := &ast.RangeStmt{Key: x, Value: y}
	if got := assignTargets(rangeBoth); len(got) != 2 {
		t.Errorf("assignTargets(RangeStmt key+value) = %d targets, want 2", len(got))
	}

	rangeNone := &ast.RangeStmt{}
	if got := assignTargets(rangeNone); len(got) != 0 {
		t.Errorf("assignTargets(RangeStmt no vars) = %d targets, want 0", len(got))
	}

	other := &ast.ExprStmt{X: &ast.BasicLit{Kind: token.INT, Value: "1"}}
	if got := assignTargets(other); got != nil {
		t.Errorf("assignTargets(ExprStmt) = %v, want nil", got)
	}
}
