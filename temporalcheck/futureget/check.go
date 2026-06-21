package futureget

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// The receiver types live in three SDK packages. Future and ChildWorkflowFuture
// are declared in internal and re-exported from workflow as aliases (type Future
// = internal.Future), mirroring workflow.Context, so a workflow.Future value
// resolves through go/types to the internal named type. EncodedValue is declared
// directly in converter. We match by package path + name through go/types --
// never importing the SDK -- so aliased imports resolve the same way.
const (
	workflowPkg  = "go.temporal.io/sdk/workflow"
	internalPkg  = "go.temporal.io/sdk/internal"
	converterPkg = "go.temporal.io/sdk/converter"
)

// receiverTypes maps the matched receiver type name to the SDK package paths it
// may surface from. Future/ChildWorkflowFuture resolve to internal but may also
// appear as the workflow alias depending on how the type checker reports them;
// accepting both paths covers gotypesalias on or off. EncodedValue lives only in
// converter. Adding another result type is a single row.
var receiverTypes = map[string][]string{
	"Future":              {workflowPkg, internalPkg},
	"ChildWorkflowFuture": {workflowPkg, internalPkg},
	"EncodedValue":        {converterPkg},
}

// checkDiscarded reports call when it is a .Get on one of the Temporal receiver
// types whose returned error is being thrown away, after honoring //nolint.
func (c *checker) checkDiscarded(pass *analysis.Pass, nolint nolintInfo, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Get" {
		return
	}

	// Match on the receiver's static type, not the method object: only the
	// Temporal result types carry the Get we own, and matching the static type
	// means a user type that merely embeds Future is conservatively skipped.
	typeName, ok := receiverTypeName(pass.TypesInfo.TypeOf(sel.X))
	if !ok {
		return
	}

	// Honor //nolint directives ourselves so suppression works the same way in
	// standalone/analysistest runs, not only under golangci-lint. Checked after
	// confirming this is a call we flag, so unrelated calls cost nothing.
	if nolint.suppressesCall(pass.Fset, call) {
		return
	}

	pass.Reportf(call.Pos(),
		"Get: the returned error from %s.Get is discarded; check it or assign it to a variable you inspect (%s)",
		typeName, tagFutureGet)
}

// receiverTypeName returns the matched receiver type name -- "Future",
// "ChildWorkflowFuture" or "EncodedValue" -- when t is that Temporal type, and
// false for anything else. types.Unalias resolves the workflow alias to its
// internal definition, so the receiver matches whether the type checker surfaces
// it as the alias or the resolved named type. Matching the package path (not the
// source text) means an aliased import resolves the same way.
func receiverTypeName(t types.Type) (string, bool) {
	if t == nil {
		return "", false
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return "", false
	}
	obj := named.Obj()
	if obj.Pkg() == nil {
		return "", false
	}
	pkgs, ok := receiverTypes[obj.Name()]
	if !ok {
		return "", false
	}
	for _, p := range pkgs {
		if obj.Pkg().Path() == p {
			return obj.Name(), true
		}
	}
	return "", false
}
