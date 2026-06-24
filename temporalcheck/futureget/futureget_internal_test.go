package futureget

import (
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
	"go/ast"
	"go/token"
	"go/types"
	"testing"
)

func TestIsBlank(t *testing.T) {
	tests := []struct {
		name string
		expr ast.Expr
		want bool
	}{
		{"blank identifier", &ast.Ident{Name: "_"}, true},
		{"named identifier", &ast.Ident{Name: "f"}, false},
		{"not an identifier", &ast.BasicLit{Kind: token.STRING, Value: `"_"`}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBlank(tt.expr); got != tt.want {
				t.Errorf("isBlank(%v) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestReturnsError(t *testing.T) {
	errType := types.Universe.Lookup("error").Type()
	sig := func(results ...types.Type) *types.Signature {
		vars := make([]*types.Var, len(results))
		for i, r := range results {
			vars[i] = types.NewVar(token.NoPos, nil, "", r)
		}
		return types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(vars...), false)
	}

	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{"not a signature", types.Typ[types.Int], false},
		{"no results", sig(), false},
		{"last result is error", sig(errType), true},
		{"value then error", sig(types.Typ[types.String], errType), true},
		{"last result not error", sig(types.Typ[types.String]), false},
		{"error not last", sig(errType, types.Typ[types.String]), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := returnsError(tt.typ); got != tt.want {
				t.Errorf("returnsError(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestReceiverTypeName(t *testing.T) {
	// A named type in a package whose path matches one of the receiver packages,
	// to exercise the matched branch without the analysistest harness.
	namedIn := func(pkgPath, name string) types.Type {
		pkg := types.NewPackage(pkgPath, "p")
		tn := types.NewTypeName(token.NoPos, pkg, name, nil)
		return types.NewNamed(tn, types.NewInterfaceType(nil, nil), nil)
	}

	// A nil package object, which must be skipped without panicking even when the
	// name would otherwise match.
	nilPkgNamed := func(name string) types.Type {
		tn := types.NewTypeName(token.NoPos, nil, name, nil)
		return types.NewNamed(tn, types.NewInterfaceType(nil, nil), nil)
	}

	tests := []struct {
		name     string
		typ      types.Type
		wantName string
		wantOK   bool
	}{
		{"nil type", nil, "", false},
		{"basic non-named type", types.Typ[types.Int], "", false},
		{"Future in workflow package", namedIn(temporalsdk.WorkflowPkg, "Future"), "Future", true},
		{"Future in internal package", namedIn(temporalsdk.InternalPkg, "Future"), "Future", true},
		{"ChildWorkflowFuture in internal", namedIn(temporalsdk.InternalPkg, "ChildWorkflowFuture"), "ChildWorkflowFuture", true},
		{"EncodedValue in converter", namedIn(temporalsdk.ConverterPkg, "EncodedValue"), "EncodedValue", true},
		{"matching name, wrong package", namedIn("example.com/other", "Future"), "", false},
		{"EncodedValue outside converter", namedIn(temporalsdk.WorkflowPkg, "EncodedValue"), "", false},
		{"unrelated name", namedIn(temporalsdk.WorkflowPkg, "Selector"), "", false},
		{"matching name, nil package", nilPkgNamed("Future"), "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotOK := receiverTypeName(tt.typ)
			if gotName != tt.wantName || gotOK != tt.wantOK {
				t.Errorf("receiverTypeName() = %q, %v; want %q, %v", gotName, gotOK, tt.wantName, tt.wantOK)
			}
		})
	}
}
