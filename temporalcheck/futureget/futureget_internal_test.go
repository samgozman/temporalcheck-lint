package futureget

import (
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
		{"Future in workflow package", namedIn(workflowPkg, "Future"), "Future", true},
		{"Future in internal package", namedIn(internalPkg, "Future"), "Future", true},
		{"ChildWorkflowFuture in internal", namedIn(internalPkg, "ChildWorkflowFuture"), "ChildWorkflowFuture", true},
		{"EncodedValue in converter", namedIn(converterPkg, "EncodedValue"), "EncodedValue", true},
		{"matching name, wrong package", namedIn("example.com/other", "Future"), "", false},
		{"EncodedValue outside converter", namedIn(workflowPkg, "EncodedValue"), "", false},
		{"unrelated name", namedIn(workflowPkg, "Selector"), "", false},
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
