package sensitiveargs

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
	"golang.org/x/tools/go/analysis"
)

func TestNewAnalyzerMetadata(t *testing.T) {
	a := NewAnalyzer(Settings{})
	if a.Name != "sensitiveargs" {
		t.Errorf("analyzer name = %q, want %q", a.Name, "sensitiveargs")
	}
	if a.Run == nil {
		t.Error("analyzer Run is nil")
	}
}

// TestInvalidPattern: an unparseable Pattern is surfaced from Run as an error
// (NewAnalyzer cannot return one), so a misconfigured regexp fails loudly rather
// than silently disabling the check.
func TestInvalidPattern(t *testing.T) {
	a := NewAnalyzer(Settings{Enabled: true, Pattern: "("})
	// The compile error is checked before any pass field, so an empty pass is fine.
	if _, err := a.Run(&analysis.Pass{}); err == nil {
		t.Fatal("expected Run to return the regexp compile error, got nil")
	}
}

func TestTargetName(t *testing.T) {
	tests := []struct {
		name string
		expr ast.Expr
		want string
	}{
		{"selector", &ast.SelectorExpr{Sel: &ast.Ident{Name: "Charge"}}, "Charge"},
		{"ident", &ast.Ident{Name: "Login"}, "Login"},
		{"fallback", &ast.BasicLit{Kind: token.STRING, Value: `"x"`}, "target"},
	}
	for _, tt := range tests {
		if got := targetName(tt.expr); got != tt.want {
			t.Errorf("%s: targetName = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func emptyIface() *types.Interface { return types.NewInterfaceType(nil, nil) }

// fieldStruct builds a one-field struct of the given name; a lowercase name yields
// an unexported field.
func fieldStruct(fieldName string) *types.Struct {
	pkg := types.NewPackage("example.com/foo", "foo")
	v := types.NewField(token.NoPos, pkg, fieldName, types.Typ[types.String], false)
	return types.NewStruct([]*types.Var{v}, nil)
}

// TestStructFields covers the struct, pointer-to-struct and alias-to-struct arms
// that reach a struct, plus the non-struct types (int, slice) that do not.
func TestStructFields(t *testing.T) {
	s := fieldStruct("V")
	aliasToStruct := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Req", nil), s)
	tests := []struct {
		name string
		t    types.Type
		want bool
	}{
		{"struct", s, true},
		{"pointer to struct", types.NewPointer(s), true},
		{"alias to struct", aliasToStruct, true},
		{"int", types.Typ[types.Int], false},
		{"slice of struct", types.NewSlice(s), false},
	}
	for _, tt := range tests {
		if _, ok := structFields(tt.t); ok != tt.want {
			t.Errorf("%s: structFields ok = %v, want %v", tt.name, ok, tt.want)
		}
	}
}

// TestEntryFor covers the dispatch table directly, including the false paths the
// fixtures don't exercise: a func in an unrelated package, and a method named
// ExecuteWorkflow whose receiver is not client.Client.
func TestEntryFor(t *testing.T) {
	wfPkg := types.NewPackage(temporalsdk.WorkflowPkg, "workflow")
	otherPkg := types.NewPackage("example.com/foo", "foo")

	plainFunc := func(pkg *types.Package, name string) *types.Func {
		sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
		return types.NewFunc(token.NoPos, pkg, name, sig)
	}
	method := func(recv types.Type, name string) *types.Func {
		rv := types.NewVar(token.NoPos, nil, "c", recv)
		sig := types.NewSignatureType(rv, nil, nil, types.NewTuple(), types.NewTuple(), false)
		return types.NewFunc(token.NoPos, otherPkg, name, sig)
	}
	wrongRecv := types.NewNamed(types.NewTypeName(token.NoPos, otherPkg, "NotClient", nil), emptyIface(), nil)

	tests := []struct {
		name    string
		fn      *types.Func
		wantOK  bool
		wantIdx int
	}{
		{"ExecuteActivity", plainFunc(wfPkg, "ExecuteActivity"), true, 1},
		{"SignalWithStartWorkflow", method(types.NewPointer(wrongRecv), "Go"), false, 0},
		{"unrelated workflow func", plainFunc(wfPkg, "Go"), false, 0},
		{"unrelated package func", plainFunc(otherPkg, "ExecuteActivity"), false, 0},
		{"ExecuteWorkflow wrong receiver", method(wrongRecv, "ExecuteWorkflow"), false, 0},
	}
	for _, tt := range tests {
		e, ok := entryFor(tt.fn)
		if ok != tt.wantOK {
			t.Errorf("%s: entryFor ok = %v, want %v", tt.name, ok, tt.wantOK)
		}
		if ok && e.targetIdx != tt.wantIdx {
			t.Errorf("%s: targetIdx = %d, want %d", tt.name, e.targetIdx, tt.wantIdx)
		}
	}
}

// TestCheckCall covers the three early returns of checkCall that the fixtures do
// not reach: a call whose function is not a selector, a selector that does not
// resolve to a func, and a selector resolving to a func that is not an Execute*
// entry point.
func TestCheckCall(t *testing.T) {
	c := &checker{}

	// Not a selector (e.g. a bare f()): returns before any type lookup.
	c.checkCall(&analysis.Pass{}, nolint.Info{}, &ast.CallExpr{Fun: &ast.Ident{Name: "f"}})

	// Selector, but the selected ident is not in Uses, so it is not a *types.Func.
	sel := &ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "Foo"}}
	unresolved := &analysis.Pass{TypesInfo: &types.Info{Uses: map[*ast.Ident]types.Object{}}}
	c.checkCall(unresolved, nolint.Info{}, &ast.CallExpr{Fun: sel})

	// Selector resolving to a func that is not an entry point: entryFor returns
	// false, so checkTarget is never reached.
	otherPkg := types.NewPackage("example.com/foo", "foo")
	fnSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
	fn := types.NewFunc(token.NoPos, otherPkg, "Foo", fnSig)
	nonEntry := &analysis.Pass{TypesInfo: &types.Info{Uses: map[*ast.Ident]types.Object{sel.Sel: fn}}}
	c.checkCall(nonEntry, nolint.Info{}, &ast.CallExpr{Fun: sel})
}

// TestCheckTargetTooFewArgs covers the defensive arg-count guard, which a
// compiling Execute* call never trips (the target argument is always present).
func TestCheckTargetTooFewArgs(t *testing.T) {
	pass := &analysis.Pass{TypesInfo: &types.Info{}}
	c := &checker{}
	call := &ast.CallExpr{Args: []ast.Expr{&ast.Ident{Name: "ctx"}, &ast.Ident{Name: "opts"}}}
	c.checkTarget(pass, nolint.Info{}, call, entry{noun: "workflow", isWorkflow: true, targetIdx: 2})
}
