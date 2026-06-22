package execargs

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
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

// TestEntryNouns pins the diagnostic noun each entry point uses: activities say
// "activity", a child workflow says "child workflow", and the workflow-restarting
// or client-launched entries say "workflow".
func TestEntryNouns(t *testing.T) {
	for name, want := range map[string]string{
		"ExecuteActivity":       "activity",
		"ExecuteLocalActivity":  "activity",
		"ExecuteChildWorkflow":  "child workflow",
		"NewContinueAsNewError": "workflow",
	} {
		if got := workflowEntries[name].noun; got != want {
			t.Errorf("workflowEntries[%q].noun = %q, want %q", name, got, want)
		}
	}
	for name := range clientEntries {
		if got := clientEntries[name].noun; got != "workflow" {
			t.Errorf("clientEntries[%q].noun = %q, want %q", name, got, "workflow")
		}
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

func TestJSONName(t *testing.T) {
	tests := []struct {
		name     string
		goName   string
		tag      string
		wantName string
		wantOK   bool
	}{
		{"no tag", "Field", "", "Field", true},
		{"renamed", "Field", `json:"renamed"`, "renamed", true},
		{"renamed with option", "Field", `json:"renamed,omitempty"`, "renamed", true},
		{"empty name with option", "Field", `json:",omitempty"`, "Field", true},
		{"skipped", "Field", `json:"-"`, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := jsonName(tt.goName, tt.tag)
			if got != tt.wantName || ok != tt.wantOK {
				t.Errorf("jsonName(%q, %q) = (%q, %v), want (%q, %v)",
					tt.goName, tt.tag, got, ok, tt.wantName, tt.wantOK)
			}
		})
	}
}

func TestDriftPhrase(t *testing.T) {
	tests := []struct {
		name string
		diff structDiff
		want string
	}{
		{"drops only", structDiff{drops: []string{"Secret"}},
			"serializes by field name but drops {Secret}"},
		{"unset only", structDiff{unset: []string{"Extra"}},
			"serializes by field name but leaves {Extra} unset"},
		{"drops and unset", structDiff{drops: []string{"A"}, unset: []string{"B"}},
			"serializes by field name but drops {A} and leaves {B} unset"},
		{"identical", structDiff{},
			"has identical fields but is a distinct Go type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := driftPhrase(tt.diff); got != tt.want {
				t.Errorf("driftPhrase = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestIsReceiver covers isReceiver, including the false branch for a non-method
// func (nil receiver), which the analysistest dispatch never reaches because it
// only calls isReceiver for OnActivity/OnWorkflow -- both real methods.
func TestIsReceiver(t *testing.T) {
	pkg := types.NewPackage(workflowInternalPkg, "internal")
	env := types.NewNamed(types.NewTypeName(token.NoPos, pkg, testEnvType, nil), types.NewStruct(nil, nil), nil)

	mkMethod := func(recv types.Type) *types.Func {
		var recvVar *types.Var
		if recv != nil {
			recvVar = types.NewVar(token.NoPos, nil, "e", recv)
		}
		sig := types.NewSignatureType(recvVar, nil, nil, types.NewTuple(), types.NewTuple(), false)
		return types.NewFunc(token.NoPos, pkg, "OnActivity", sig)
	}

	tests := []struct {
		name string
		recv types.Type // nil means a non-method func
		want bool
	}{
		{"pointer to env", types.NewPointer(env), true},
		{"value env", env, true},
		{"not a method", nil, false},
		{"non-named receiver", types.Typ[types.Int], false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReceiver(mkMethod(tt.recv), workflowInternalPkg, testEnvType); got != tt.want {
				t.Errorf("isReceiver(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestNamedAlias covers the *types.Alias arm of named, which the analysistest
// fixtures never reach (the test SDK stub resolves workflow.Context to its
// internal named type).
func TestNamedAlias(t *testing.T) {
	pkg := types.NewPackage("example.com/foo", "foo")
	tn := types.NewTypeName(token.NoPos, pkg, "Bar", nil)
	alias := types.NewAlias(tn, types.Typ[types.Int])

	if !named(alias, "example.com/foo", "Bar") {
		t.Error("named did not match the alias by package path and name")
	}
	if named(alias, "example.com/foo", "Other") {
		t.Error("named matched the wrong name")
	}
}

// TestIsWorkflowContextDirect covers the fallback in isWorkflowContext for a
// Context type declared directly in the public workflow package (rather than as
// an alias to the internal type, which the stub uses).
func TestIsWorkflowContextDirect(t *testing.T) {
	pkg := types.NewPackage(workflowPkg, "workflow")
	tn := types.NewTypeName(token.NoPos, pkg, "Context", nil)
	ctxType := types.NewNamed(tn, types.NewStruct(nil, nil), nil)

	if !isWorkflowContext(ctxType) {
		t.Error("isWorkflowContext did not recognize Context declared in the workflow package")
	}
}

// TestSkipCountNoParams covers the zero-parameter guard in skipCount.
func TestSkipCountNoParams(t *testing.T) {
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
	if got := skipCount(sig, kindActivity); got != 0 {
		t.Errorf("skipCount(no params) = %d, want 0", got)
	}
}

// TestCheckAssignableNilType covers the nil-type guard: when the argument has no
// resolved type, checkAssignable returns without reporting.
func TestCheckAssignableNilType(t *testing.T) {
	pass := &analysis.Pass{TypesInfo: &types.Info{}}
	c := &checker{strictTypes: true}
	// An identifier absent from the (empty) TypesInfo resolves to a nil type, so
	// the guard returns before any Reportf (which would need a Fset and panic).
	c.checkAssignable(pass, &ast.Ident{Name: "x"}, "ExecuteActivity", "fn", 1, types.Typ[types.String])
}

// TestCheckExecuteCallTooFewArgs covers the target-index guard: a call with no
// argument at the target index returns before any type work, so the empty
// TypesInfo is never consulted and fn is never dereferenced.
func TestCheckExecuteCallTooFewArgs(t *testing.T) {
	pass := &analysis.Pass{Fset: token.NewFileSet(), TypesInfo: &types.Info{}}
	c := &checker{}
	// targetIdx 2 but only two arguments: ctx and options, no target.
	call := &ast.CallExpr{Fun: &ast.Ident{Name: "f"}, Args: []ast.Expr{&ast.Ident{Name: "ctx"}, &ast.Ident{Name: "opts"}}}
	c.checkExecuteCall(pass, nolintInfo{}, call, nil, entry{noun: "workflow", kind: kindWorkflow, targetIdx: 2})
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
