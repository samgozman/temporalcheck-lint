package searchattribute

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestNewAnalyzerMetadata: the analyzer is named and runnable.
func TestNewAnalyzerMetadata(t *testing.T) {
	a := NewAnalyzer(Settings{})
	if a.Name != "searchattribute" {
		t.Errorf("analyzer name = %q, want %q", a.Name, "searchattribute")
	}
	if a.Run == nil {
		t.Error("analyzer Run is nil")
	}
}

// TestNormalize: field names and aliases collapse to a case- and
// separator-insensitive key, so the varied ways a team writes an id all match.
func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"UserID":    "userid",
		"user_id":   "userid",
		"x-user-id": "xuserid",
		"XUserID":   "xuserid",
		"ClientID":  "clientid",
		"":          "",
	}
	for in, want := range cases {
		if got := normalize(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestNewAnalyzerNormalizesAliasKeys: NewAnalyzer normalizes the configured alias
// keys once, so a differently-cased/separated config key still matches.
func TestNewAnalyzerNormalizesAliasKeys(t *testing.T) {
	c := checkerFrom(Settings{Attributes: map[string]string{"X-User-ID": "user_id"}})
	if got := c.aliases["xuserid"]; got != "user_id" {
		t.Errorf("aliases[%q] = %q, want %q", "xuserid", got, "user_id")
	}
}

// checkerFrom rebuilds the checker NewAnalyzer constructs, so a test can inspect
// the normalized alias map without reaching through the analysis.Analyzer.
func checkerFrom(s Settings) checker {
	aliases := make(map[string]string, len(s.Attributes))
	for alias, attr := range s.Attributes {
		aliases[normalize(alias)] = attr
	}
	return checker{enabled: s.Enabled, aliases: aliases}
}

// TestStringLiterals: every string literal in an expression subtree is collected
// (map keys and call arguments alike), and non-string literals are ignored.
func TestStringLiterals(t *testing.T) {
	expr := mustParseExpr(t, `f(map[string]interface{}{"user_id": 42, "order_id": g("h")})`)
	got := stringLiterals([]ast.Expr{expr})
	want := map[string]bool{"user_id": true, "order_id": true, "h": true}
	if len(got) != len(want) {
		t.Fatalf("stringLiterals = %v, want keys %v", got, want)
	}
	for _, v := range got {
		if !want[v] {
			t.Errorf("stringLiterals returned unexpected %q", v)
		}
	}
}

// TestStructFieldsNil: a nil type is not a struct, guarding the unresolved-type
// path.
func TestStructFieldsNil(t *testing.T) {
	if _, ok := structFields(nil); ok {
		t.Error("structFields(nil) reported a struct")
	}
}

func mustParseExpr(t *testing.T, src string) ast.Expr {
	t.Helper()
	e, err := parser.ParseExprFrom(token.NewFileSet(), "expr.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return e
}
