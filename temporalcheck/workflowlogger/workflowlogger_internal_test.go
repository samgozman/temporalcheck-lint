package workflowlogger

import (
	"go/ast"
	"testing"
)

// TestWritesToStdStream_NoArgs covers the defensive guard for a Fprint* call with
// no arguments (only reachable on a malformed/partial AST, never in compiling
// source). The guard returns before touching the pass, so a nil pass is safe.
func TestWritesToStdStream_NoArgs(t *testing.T) {
	if writesToStdStream(nil, &ast.CallExpr{}) {
		t.Error("writesToStdStream(call with no args) = true, want false")
	}
}

// TestCalleeFunc_NonFuncCallee covers the branch where a call's callee is neither
// a selector nor an identifier (an immediately-invoked function literal); it has
// no named function to resolve, so calleeFunc returns nil without using the pass.
func TestCalleeFunc_NonFuncCallee(t *testing.T) {
	if fn := calleeFunc(nil, &ast.CallExpr{Fun: &ast.FuncLit{}}); fn != nil {
		t.Errorf("calleeFunc(IIFE) = %v, want nil", fn)
	}
}
