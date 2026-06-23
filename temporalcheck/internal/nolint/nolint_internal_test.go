package nolint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestDirective(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"bare nolint", "//nolint", true},
		{"nolint all", "//nolint:all", true},
		{"names temporalcheck", "//nolint:temporalcheck", true},
		{"temporalcheck in a list", "//nolint:gocritic,temporalcheck,godot", true},
		{"temporalcheck with explanation", "//nolint:temporalcheck // intentional", true},
		{"spaces around names", "//nolint: temporalcheck , gocritic", true},
		{"names an analyzer, not the plugin", "//nolint:execargs", false},
		{"other linters only", "//nolint:gocritic,godot", false},
		{"not a nolint", "// nolint is mentioned in prose", false},
		{"lookalike prefix", "//nolintfoo:temporalcheck", false},
		{"plain comment", "// just a comment", false},
		{"block comment", "/* nolint:temporalcheck */", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := directive(tt.text); got != tt.want {
				t.Errorf("directive(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

// parseFile parses src and returns its fset and AST, failing the test on error.
func parseFile(t *testing.T, src string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return fset, file
}

func TestCollectAndSuppresses(t *testing.T) {
	const src = `package p

// an ordinary comment that is not a directive
func f() {
	g() //nolint:temporalcheck
	h()
}

func multi(
	x int,
) { //nolint
	_ = x
}
`
	fset, file := parseFile(t, src)
	info := Collect(fset, file)

	// Locate the three calls by name.
	calls := map[string]ast.Node{}
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok {
				calls[id.Name] = call
			}
		}
		return true
	})

	if !info.Suppresses(fset, calls["g"]) {
		t.Error("g() should be suppressed by trailing //nolint:temporalcheck")
	}
	if info.Suppresses(fset, calls["h"]) {
		t.Error("h() should not be suppressed")
	}
}

func TestSuppressesNodeSpanningLines(t *testing.T) {
	// A directive on the closing line of a multi-line call suppresses it.
	const src = `package p

func f() {
	g(
		1,
		2,
	) //nolint
}
`
	fset, file := parseFile(t, src)
	info := Collect(fset, file)

	var call ast.Node
	ast.Inspect(file, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			call = c
		}
		return true
	})

	if !info.Suppresses(fset, call) {
		t.Error("multi-line call should be suppressed by //nolint on its closing line")
	}
}

func TestFileSuppressed(t *testing.T) {
	const src = `//nolint:temporalcheck
package p

func f() { g() }
`
	fset, file := parseFile(t, src)
	info := Collect(fset, file)

	var call ast.Node
	ast.Inspect(file, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			call = c
		}
		return true
	})

	if !info.Suppresses(fset, call) {
		t.Error("a directive before the package clause should suppress the whole file")
	}
}
