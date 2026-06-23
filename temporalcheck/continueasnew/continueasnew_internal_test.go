package continueasnew

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
		{"named identifier", &ast.Ident{Name: "err"}, false},
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

func TestIsContinueAsNewError(t *testing.T) {
	// A function object in a given package, to exercise the match branches without
	// the analysistest harness.
	funcIn := func(pkgPath, name string) *types.Func {
		pkg := types.NewPackage(pkgPath, "p")
		sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		return types.NewFunc(token.NoPos, pkg, name, sig)
	}

	// A package-less builtin-style func, which must be skipped without panicking.
	noPkg := func(name string) *types.Func {
		sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		return types.NewFunc(token.NoPos, nil, name, sig)
	}

	tests := []struct {
		name string
		fn   *types.Func
		want bool
	}{
		{"nil func", nil, false},
		{"workflow.NewContinueAsNewError", funcIn(temporalsdk.WorkflowPkg, funcName), true},
		{"right name, wrong package", funcIn("example.com/other", funcName), false},
		{"right package, wrong name", funcIn(temporalsdk.WorkflowPkg, "ExecuteActivity"), false},
		{"matching name, nil package", noPkg(funcName), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isContinueAsNewError(tt.fn); got != tt.want {
				t.Errorf("isContinueAsNewError(%v) = %v, want %v", tt.fn, got, tt.want)
			}
		})
	}
}
