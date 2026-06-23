package nonserializable

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
	if a.Name != "nonserializable" {
		t.Errorf("analyzer name = %q, want %q", a.Name, "nonserializable")
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

func emptyIface() *types.Interface { return types.NewInterfaceType(nil, nil) }

// fieldStruct builds a struct with one field of the given name and type, so a
// lowercase name yields an unexported field.
func fieldStruct(fieldName string) *types.Struct {
	pkg := types.NewPackage("example.com/foo", "foo")
	v := types.NewField(token.NoPos, pkg, fieldName, types.Typ[types.Int], false)
	return types.NewStruct([]*types.Var{v}, nil)
}

func TestIsUnencodable(t *testing.T) {
	funcSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
	str := types.Typ[types.String]
	tests := []struct {
		name string
		t    types.Type
		want bool
	}{
		{"chan", types.NewChan(types.SendRecv, types.Typ[types.Int]), true},
		{"func", funcSig, true},
		{"named func type", types.NewNamed(types.NewTypeName(token.NoPos, nil, "Handler", nil), funcSig, nil), true},
		{"struct", fieldStruct("v"), false},
		{"map", types.NewMap(str, types.Typ[types.Int]), false},
		{"slice of chan", types.NewSlice(types.NewChan(types.SendRecv, types.Typ[types.Int])), false},
		{"interface", emptyIface(), false},
		{"basic type", types.Typ[types.Int], false},
	}
	for _, tt := range tests {
		if got := isUnencodable(tt.t); got != tt.want {
			t.Errorf("%s: isUnencodable = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsEmptyStruct(t *testing.T) {
	tests := []struct {
		name string
		t    types.Type
		want bool
	}{
		{"only unexported field", fieldStruct("v"), true},
		{"exported field", fieldStruct("V"), false},
		{"fieldless struct{}", types.NewStruct(nil, nil), false},
		{"not a struct", types.Typ[types.Int], false},
		{"chan", types.NewChan(types.SendRecv, types.Typ[types.Int]), false},
	}
	for _, tt := range tests {
		if got := isEmptyStruct(tt.t); got != tt.want {
			t.Errorf("%s: isEmptyStruct = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsMarshalJSON(t *testing.T) {
	pkg := types.NewPackage("example.com/foo", "foo")
	byteSlice := types.NewSlice(types.Typ[types.Uint8])
	str := types.Typ[types.String]

	mkFunc := func(name string, params, results []types.Type) *types.Func {
		toTuple := func(ts []types.Type) *types.Tuple {
			vars := make([]*types.Var, len(ts))
			for i, tp := range ts {
				vars[i] = types.NewVar(token.NoPos, pkg, "", tp)
			}
			return types.NewTuple(vars...)
		}
		sig := types.NewSignatureType(nil, nil, nil, toTuple(params), toTuple(results), false)
		return types.NewFunc(token.NoPos, pkg, name, sig)
	}

	tests := []struct {
		name string
		obj  types.Object
		want bool
	}{
		{"correct shape", mkFunc("MarshalJSON", nil, []types.Type{byteSlice, errorType}), true},
		{"wrong name", mkFunc("String", nil, []types.Type{byteSlice, errorType}), false},
		{"extra param", mkFunc("MarshalJSON", []types.Type{str}, []types.Type{byteSlice, errorType}), false},
		{"one result", mkFunc("MarshalJSON", nil, []types.Type{byteSlice}), false},
		{"first result not []byte", mkFunc("MarshalJSON", nil, []types.Type{str, errorType}), false},
		{"second result not error", mkFunc("MarshalJSON", nil, []types.Type{byteSlice, str}), false},
		{"not a func", types.NewVar(token.NoPos, pkg, "MarshalJSON", str), false},
	}
	for _, tt := range tests {
		if got := isMarshalJSON(tt.obj); got != tt.want {
			t.Errorf("%s: isMarshalJSON = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsByteSlice(t *testing.T) {
	tests := []struct {
		name string
		t    types.Type
		want bool
	}{
		{"[]byte", types.NewSlice(types.Typ[types.Uint8]), true},
		{"[]string", types.NewSlice(types.Typ[types.String]), false},
		{"not a slice", types.Typ[types.Int], false},
	}
	for _, tt := range tests {
		if got := isByteSlice(tt.t); got != tt.want {
			t.Errorf("%s: isByteSlice = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestArgType(t *testing.T) {
	intT := types.Typ[types.Int]
	chanT := types.NewChan(types.SendRecv, intT)

	mkSig := func(variadic bool, params ...types.Type) *types.Signature {
		vars := make([]*types.Var, len(params))
		for i, p := range params {
			vars[i] = types.NewVar(token.NoPos, nil, "", p)
		}
		return types.NewSignatureType(nil, nil, nil, types.NewTuple(vars...), types.NewTuple(), variadic)
	}

	// variadic final parameter: argType unwraps the slice to its element.
	varSig := mkSig(true, intT, types.NewSlice(chanT))
	if got := argType(varSig, 1); got != chanT {
		t.Errorf("variadic final: argType = %v, want chan int", got)
	}
	// variadic, but not the final index: the parameter type is returned as-is.
	if got := argType(varSig, 0); got != intT {
		t.Errorf("variadic non-final: argType = %v, want int", got)
	}
	// non-variadic: the parameter type is returned as-is.
	if got := argType(mkSig(false, chanT), 0); got != chanT {
		t.Errorf("non-variadic: argType = %v, want chan int", got)
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
