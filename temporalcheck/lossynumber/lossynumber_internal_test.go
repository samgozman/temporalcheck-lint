package lossynumber

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

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

func TestDeref(t *testing.T) {
	if got := deref(types.NewPointer(types.Typ[types.Int])); got != types.Typ[types.Int] {
		t.Errorf("deref(*int) = %v, want int", got)
	}
	if got := deref(types.Typ[types.Int]); got != types.Typ[types.Int] {
		t.Errorf("deref(int) = %v, want int", got)
	}
}

// TestNamedAlias covers the *types.Alias arm and the non-named default of named,
// which the analysistest fixtures never reach (the stub resolves the SDK types to
// their internal named forms).
func TestNamedAlias(t *testing.T) {
	pkg := types.NewPackage("example.com/foo", "foo")
	alias := types.NewAlias(types.NewTypeName(token.NoPos, pkg, "Bar", nil), types.Typ[types.Int])

	if !named(alias, "example.com/foo", "Bar") {
		t.Error("named did not match the alias by package path and name")
	}
	if named(alias, "example.com/foo", "Other") {
		t.Error("named matched the wrong name")
	}
	if named(types.Typ[types.Int], "example.com/foo", "Bar") {
		t.Error("named matched a non-named type")
	}
}

// TestIsWorkflowContextDirect covers the fallback for a Context declared directly
// in the public workflow package (rather than as an alias to the internal type,
// which the stub uses), plus the false case.
func TestIsWorkflowContextDirect(t *testing.T) {
	pkg := types.NewPackage(workflowPkg, "workflow")
	ctx := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Context", nil), types.NewStruct(nil, nil), nil)
	if !isWorkflowContext(ctx) {
		t.Error("isWorkflowContext did not recognize Context declared in the workflow package")
	}
	if isWorkflowContext(types.Typ[types.Int]) {
		t.Error("isWorkflowContext matched a non-context type")
	}
}

// TestIsReceiver covers isReceiver, including the false branch for a non-method
// func and a non-named receiver, which the dispatch never reaches because it only
// calls isReceiver for ExecuteWorkflow on a real interface method.
func TestIsReceiver(t *testing.T) {
	pkg := types.NewPackage(internalPkg, "internal")
	client := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Client", nil), emptyIface(), nil)

	mkFunc := func(recv types.Type) *types.Func {
		var recvVar *types.Var
		if recv != nil {
			recvVar = types.NewVar(token.NoPos, nil, "c", recv)
		}
		sig := types.NewSignatureType(recvVar, nil, nil, types.NewTuple(), types.NewTuple(), false)
		return types.NewFunc(token.NoPos, pkg, "ExecuteWorkflow", sig)
	}

	tests := []struct {
		name string
		recv types.Type
		want bool
	}{
		{"value client", client, true},
		{"pointer to client", types.NewPointer(client), true},
		{"not a method", nil, false},
		{"non-named receiver", types.Typ[types.Int], false},
	}
	for _, tt := range tests {
		if got := isReceiver(mkFunc(tt.recv), internalPkg, "Client"); got != tt.want {
			t.Errorf("%s: isReceiver = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestEntryFor covers the dispatch table directly, including the false paths the
// fixtures don't exercise: a func in an unrelated package, and a method named
// ExecuteWorkflow whose receiver is not client.Client.
func TestEntryFor(t *testing.T) {
	wfPkg := types.NewPackage(workflowPkg, "workflow")
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
	c.checkTarget(pass, nolintInfo{}, call, entry{noun: "workflow", isWorkflow: true, targetIdx: 2})
}

// TestSkipCount covers every branch: the zero-parameter guard, an activity with
// and without the optional leading context, and a workflow with and without the
// injected workflow.Context.
func TestSkipCount(t *testing.T) {
	ctxPkg := types.NewPackage(contextPkg, "context")
	ctxType := types.NewNamed(types.NewTypeName(token.NoPos, ctxPkg, "Context", nil), emptyIface(), nil)
	wfPkg := types.NewPackage(workflowPkg, "workflow")
	wfCtx := types.NewNamed(types.NewTypeName(token.NoPos, wfPkg, "Context", nil), emptyIface(), nil)

	sig := func(first types.Type) *types.Signature {
		var params *types.Tuple
		if first == nil {
			params = types.NewTuple()
		} else {
			params = types.NewTuple(types.NewVar(token.NoPos, nil, "x", first))
		}
		return types.NewSignatureType(nil, nil, nil, params, types.NewTuple(), false)
	}

	tests := []struct {
		name       string
		first      types.Type
		isWorkflow bool
		want       int
	}{
		{"no params", nil, false, 0},
		{"activity with context", ctxType, false, 1},
		{"activity without context", types.Typ[types.Int], false, 0},
		{"workflow with context", wfCtx, true, 1},
		{"workflow without context", types.Typ[types.Int], true, 0},
	}
	for _, tt := range tests {
		if got := skipCount(sig(tt.first), tt.isWorkflow); got != tt.want {
			t.Errorf("%s: skipCount = %d, want %d", tt.name, got, tt.want)
		}
	}
}
