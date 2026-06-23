package lossynumber

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
	if a.Name != "lossynumber" {
		t.Errorf("analyzer name = %q, want %q", a.Name, "lossynumber")
	}
	if a.Run == nil {
		t.Error("analyzer Run is nil")
	}
}

func TestTargetName(t *testing.T) {
	tests := []struct {
		name string
		expr ast.Expr
		want string
	}{
		{"selector", &ast.SelectorExpr{Sel: &ast.Ident{Name: "Greet"}}, "Greet"},
		{"ident", &ast.Ident{Name: "Archive"}, "Archive"},
		{"fallback", &ast.BasicLit{Kind: token.STRING, Value: `"x"`}, "target"},
	}
	for _, tt := range tests {
		if got := targetName(tt.expr); got != tt.want {
			t.Errorf("%s: targetName = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// emptyIface and a non-empty interface used across the type tests.
func emptyIface() *types.Interface    { return types.NewInterfaceType(nil, nil) }
func nonEmptyIface() *types.Interface { return errorType.Underlying().(*types.Interface) }

func TestIsEmptyInterface(t *testing.T) {
	pkg := types.NewPackage("example.com/foo", "foo")
	namedEmpty := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Payload", nil), emptyIface(), nil)

	tests := []struct {
		name string
		t    types.Type
		want bool
	}{
		{"interface{}", emptyIface(), true},
		{"named empty interface", namedEmpty, true},
		{"error (non-empty)", errorType, false},
		{"basic type", types.Typ[types.Int], false},
	}
	for _, tt := range tests {
		if got := isEmptyInterface(tt.t); got != tt.want {
			t.Errorf("%s: isEmptyInterface = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsLossyDynamic(t *testing.T) {
	str := types.Typ[types.String]
	tests := []struct {
		name string
		t    types.Type
		want bool
	}{
		{"interface{}", emptyIface(), true},
		{"map[string]any", types.NewMap(str, emptyIface()), true},
		{"[]any", types.NewSlice(emptyIface()), true},
		{"map[string]int", types.NewMap(str, types.Typ[types.Int]), false},
		{"[]byte", types.NewSlice(types.Typ[types.Byte]), false},
		{"non-empty interface", nonEmptyIface(), false},
		{"basic type", types.Typ[types.Int64], false},
		{"struct", types.NewStruct(nil, nil), false},
	}
	for _, tt := range tests {
		if got := isLossyDynamic(tt.t); got != tt.want {
			t.Errorf("%s: isLossyDynamic = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsError(t *testing.T) {
	if !isError(errorType) {
		t.Error("isError(error) = false, want true")
	}
	if isError(types.Typ[types.Int]) {
		t.Error("isError(int) = true, want false")
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
		{"ExecuteChildWorkflow", plainFunc(wfPkg, "ExecuteChildWorkflow"), true, 1},
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

// TestCheckTargetTooFewArgs covers the defensive arg-count guard, which a
// compiling Execute* call never trips (the target argument is always present).
func TestCheckTargetTooFewArgs(t *testing.T) {
	pass := &analysis.Pass{TypesInfo: &types.Info{}}
	c := &checker{}
	// targetIdx 2 but only two arguments: the guard returns before any type work,
	// so the empty TypesInfo is never consulted.
	call := &ast.CallExpr{Args: []ast.Expr{&ast.Ident{Name: "ctx"}, &ast.Ident{Name: "opts"}}}
	c.checkTarget(pass, nolint.Info{}, call, entry{noun: "workflow", isWorkflow: true, targetIdx: 2})
}
