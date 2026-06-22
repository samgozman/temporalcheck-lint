package continueasnew

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// NewContinueAsNewError is a plain function in the workflow package (not a type,
// so the internal-alias re-export that bites type-matching analyzers does not
// apply). We match it by package path + name through go/types -- never importing
// the SDK -- so an aliased import of the workflow package resolves the same way.
const (
	workflowPkg = "go.temporal.io/sdk/workflow"
	funcName    = "NewContinueAsNewError"
)

// checkDiscarded reports call when it is workflow.NewContinueAsNewError and its
// result is being thrown away, after honoring //nolint.
func (c *checker) checkDiscarded(pass *analysis.Pass, nolint nolintInfo, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Resolve via Uses (not the source text), so an aliased import of the
	// workflow package still matches.
	fn, _ := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !isContinueAsNewError(fn) {
		return
	}

	// Honor //nolint directives ourselves so suppression works the same way in
	// standalone/analysistest runs, not only under golangci-lint. Checked after
	// confirming this is a call we flag, so unrelated calls cost nothing.
	if nolint.suppressesCall(pass.Fset, call) {
		return
	}

	pass.Reportf(call.Pos(),
		"%s: the continue-as-new error is discarded; return it so the workflow continues as new, otherwise the workflow silently ends instead (%s)",
		funcName, tagContinueAsNew)
}

// isContinueAsNewError reports whether fn is workflow.NewContinueAsNewError,
// matched by package path and name so an aliased import resolves the same. A nil
// fn (the Uses entry was not a function) or a package-less builtin is not a match.
func isContinueAsNewError(fn *types.Func) bool {
	return fn != nil && fn.Pkg() != nil &&
		fn.Pkg().Path() == workflowPkg && fn.Name() == funcName
}
