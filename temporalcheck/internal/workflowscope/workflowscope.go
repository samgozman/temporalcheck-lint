// Package workflowscope locates Temporal workflow definitions in a file and
// walks their bodies. A workflow is a function whose first parameter is
// workflow.Context; the analyzers that reason about determinism (what may run
// inside workflow code) share this discovery, so it lives here once.
package workflowscope

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
)

// FuncBody returns the body and type of a function declaration or literal, or
// (nil, nil) for any other node.
func FuncBody(n ast.Node) (*ast.BlockStmt, *ast.FuncType) {
	switch fn := n.(type) {
	case *ast.FuncDecl:
		return fn.Body, fn.Type
	case *ast.FuncLit:
		return fn.Body, fn.Type
	default:
		return nil, nil
	}
}

// IsWorkflowFunc reports whether a function with the given type is a Temporal
// workflow definition: its first parameter is workflow.Context.
func IsWorkflowFunc(pass *analysis.Pass, ft *ast.FuncType) bool {
	if ft == nil || ft.Params == nil || len(ft.Params.List) == 0 {
		return false
	}
	// A field may declare several names sharing one type; the first parameter's
	// type is that field's type regardless.
	return temporalsdk.IsWorkflowContext(pass.TypesInfo.TypeOf(ft.Params.List[0].Type))
}

// Walk visits each top-level workflow definition in file and invokes report with
// its body. The first workflow found on a path owns its whole subtree: Walk does
// not descend into a workflow's nested closures, since those run as part of the
// same workflow execution and are already reached by walking the body -- entering
// them again would double-report.
func Walk(pass *analysis.Pass, file *ast.File, report func(body *ast.BlockStmt)) {
	ast.Inspect(file, func(n ast.Node) bool {
		body, ft := FuncBody(n)
		if body == nil {
			return true
		}
		if IsWorkflowFunc(pass, ft) {
			report(body)
			return false
		}
		return true
	})
}
