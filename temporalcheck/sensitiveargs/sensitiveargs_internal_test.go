package sensitiveargs

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

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

// TestIsReceiver covers isReceiver, including the false branches for a non-method
// func and a non-named receiver, which dispatch never reaches because it only
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
	c.checkCall(&analysis.Pass{}, nolintInfo{}, &ast.CallExpr{Fun: &ast.Ident{Name: "f"}})

	// Selector, but the selected ident is not in Uses, so it is not a *types.Func.
	sel := &ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "Foo"}}
	unresolved := &analysis.Pass{TypesInfo: &types.Info{Uses: map[*ast.Ident]types.Object{}}}
	c.checkCall(unresolved, nolintInfo{}, &ast.CallExpr{Fun: sel})

	// Selector resolving to a func that is not an entry point: entryFor returns
	// false, so checkTarget is never reached.
	otherPkg := types.NewPackage("example.com/foo", "foo")
	fnSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
	fn := types.NewFunc(token.NoPos, otherPkg, "Foo", fnSig)
	nonEntry := &analysis.Pass{TypesInfo: &types.Info{Uses: map[*ast.Ident]types.Object{sel.Sel: fn}}}
	c.checkCall(nonEntry, nolintInfo{}, &ast.CallExpr{Fun: sel})
}

// TestCheckTargetTooFewArgs covers the defensive arg-count guard, which a
// compiling Execute* call never trips (the target argument is always present).
func TestCheckTargetTooFewArgs(t *testing.T) {
	pass := &analysis.Pass{TypesInfo: &types.Info{}}
	c := &checker{}
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
