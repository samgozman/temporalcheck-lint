package lossynumber

import (
	"go/ast"
	"go/types"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
	"golang.org/x/tools/go/analysis"
)

// tagLossyTypes suffixes the diagnostic so it is clear which check produced it.
const tagLossyTypes = "lossy-types"

// explain is the shared tail of every diagnostic.
const explain = "Temporal's JSON converter decodes numbers as float64 and silently loses int64 precision past 2^53 -- use a concrete type"

// entry describes one Execute* entry point.
type entry struct {
	noun       string
	isWorkflow bool
	targetIdx  int
}

// workflowEntries are the workflow.* package functions this analyzer understands.
var workflowEntries = map[string]entry{
	"ExecuteActivity":       {noun: "activity", isWorkflow: false, targetIdx: 1},
	"ExecuteLocalActivity":  {noun: "activity", isWorkflow: false, targetIdx: 1},
	"ExecuteChildWorkflow":  {noun: "child workflow", isWorkflow: true, targetIdx: 1},
	"NewContinueAsNewError": {noun: "workflow", isWorkflow: true, targetIdx: 1},
}

// clientEntries are the client.Client methods this analyzer understands.
// Target index: ExecuteWorkflow=2, SignalWithStartWorkflow=5.
var clientEntries = map[string]entry{
	"ExecuteWorkflow":         {noun: "workflow", isWorkflow: true, targetIdx: 2},
	"SignalWithStartWorkflow": {noun: "workflow", isWorkflow: true, targetIdx: 5},
}

// entryFor reports whether fn is an Execute*/continue-as-new entry point this
// analyzer inspects. workflow.* are package functions; the client methods are on
// the client.Client interface, so we match them by name and receiver rather than
// by package path.
func entryFor(fn *types.Func) (entry, bool) {
	if fn.Pkg().Path() == temporalsdk.WorkflowPkg {
		e, ok := workflowEntries[fn.Name()]
		return e, ok
	}
	if e, ok := clientEntries[fn.Name()]; ok &&
		(temporalsdk.IsReceiver(fn, temporalsdk.ClientPkg, "Client") || temporalsdk.IsReceiver(fn, temporalsdk.InternalPkg, "Client")) {
		return e, true
	}
	return entry{}, false
}

// checkTarget resolves the target reference to its signature and flags any
// top-level lossy dynamic parameter or return. A target we cannot resolve to a
// signature -- a string-registered name or any non-function value -- is left
// alone rather than risk a false positive.
func (c *checker) checkTarget(pass *analysis.Pass, nolint nolint.Info, call *ast.CallExpr, e entry) {
	if len(call.Args) <= e.targetIdx {
		return
	}
	target := call.Args[e.targetIdx]
	sig, ok := pass.TypesInfo.TypeOf(target).(*types.Signature)
	if !ok {
		return
	}

	// Honor //nolint directives ourselves so suppression works the same way in
	// standalone/analysistest runs, not only under golangci-lint. Checked after
	// confirming this is a target we resolve, so unrelated calls cost nothing.
	if nolint.Suppresses(pass.Fset, call) {
		return
	}

	name := targetName(target)
	c.checkParams(pass, target, e, name, sig)
	c.checkResults(pass, target, e, name, sig)
}

// checkParams flags each user-supplied parameter whose type is lossy dynamic,
// skipping the framework-injected leading context. The parameter number in the
// message is 1-based over the user parameters, so it matches what the author
// writes after the context.
func (c *checker) checkParams(pass *analysis.Pass, target ast.Expr, e entry, name string, sig *types.Signature) {
	params := sig.Params()
	skip := temporalsdk.SkipCount(sig, e.isWorkflow)
	for i := skip; i < params.Len(); i++ {
		// A variadic final parameter's type is the slice []T; ...any is []any,
		// whose element is the empty interface, so isLossyDynamic classifies it the
		// same as a plain []any parameter -- no special case needed.
		t := params.At(i).Type()
		if isLossyDynamic(t) {
			c.reportf(pass, target, "%s %q parameter %d has dynamic type %s; %s (%s)",
				e.noun, name, i-skip+1, typeStr(t), explain, tagLossyTypes)
		}
	}
}

// checkResults flags each result whose type is lossy dynamic, skipping a single
// trailing error (the conventional last return of an activity/workflow).
func (c *checker) checkResults(pass *analysis.Pass, target ast.Expr, e entry, name string, sig *types.Signature) {
	results := sig.Results()
	n := results.Len()
	if n > 0 && isError(results.At(n-1).Type()) {
		n--
	}
	for i := 0; i < n; i++ {
		t := results.At(i).Type()
		if isLossyDynamic(t) {
			c.reportf(pass, target, "%s %q return %d has dynamic type %s; %s (%s)",
				e.noun, name, i+1, typeStr(t), explain, tagLossyTypes)
		}
	}
}

// isLossyDynamic reports whether t is one of the dynamic types a number decodes
// into lossily: the empty interface itself, or a map/slice whose element is the
// empty interface (map[K]interface{}, []interface{}). The check is intentionally
// shallow -- a struct that merely contains an `any` field is not flagged -- so it
// stays false-positive-free.
func isLossyDynamic(t types.Type) bool {
	if isEmptyInterface(t) {
		return true
	}
	switch u := types.Unalias(t).Underlying().(type) {
	case *types.Map:
		return isEmptyInterface(u.Elem())
	case *types.Slice:
		return isEmptyInterface(u.Elem())
	}
	return false
}

// isEmptyInterface reports whether t is an interface with no methods --
// interface{}, the any alias, or a named `type X interface{}`. A non-empty
// interface (error, io.Reader, a custom interface) is not lossy and never matches.
func isEmptyInterface(t types.Type) bool {
	iface, ok := types.Unalias(t).Underlying().(*types.Interface)
	return ok && iface.NumMethods() == 0
}

// errorType is the universe error, used to recognize the trailing error return.
var errorType = types.Universe.Lookup("error").Type()

func isError(t types.Type) bool {
	return types.Identical(t, errorType)
}

// typeStr renders a type using short package names (context.Context, not the full
// import path).
func typeStr(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Name() })
}

// targetName is the source name of the target reference, used in diagnostics.
func targetName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.Ident:
		return e.Name
	default:
		return "target"
	}
}

// reportf is a thin wrapper over pass.Reportf anchored at the target reference,
// where the offending signature is named.
func (c *checker) reportf(pass *analysis.Pass, target ast.Expr, format string, args ...any) {
	pass.Reportf(target.Pos(), format, args...)
}
