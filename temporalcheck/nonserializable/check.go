package nonserializable

import (
	"go/ast"
	"go/types"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
	"golang.org/x/tools/go/analysis"
)

// Diagnostics are suffixed with the source that produced them. The chan/func
// check is always on; the struct check is opt-in via EmptyStruct.
const (
	tagUnencodable = "unencodable"
	tagEmptyStruct = "empty-struct"
)

// The shared tails of the two diagnostics: why the type can't round-trip and what
// to do about it.
const (
	explainUnencodable = "Temporal's DataConverter cannot serialize a channel or function — use a serializable type"
	explainEmptyStruct = "Temporal's JSON converter serializes a struct with no exported fields to {} and silently drops its data — export fields or implement json.Marshaler"
)

// entry describes one Execute* entry point: how the diagnostic names the target,
// whether the target is a workflow (leading workflow.Context, always injected) or
// an activity (leading context.Context, optional), and which call argument is the
// target reference.
type entry struct {
	noun       string
	isWorkflow bool
	targetIdx  int
}

// workflowEntries are the workflow.* package functions this analyzer understands.
// Each names its target as the second argument: ExecuteActivity(ctx, target,
// args...) / NewContinueAsNewError(ctx, target, args...).
var workflowEntries = map[string]entry{
	"ExecuteActivity":       {noun: "activity", isWorkflow: false, targetIdx: 1},
	"ExecuteLocalActivity":  {noun: "activity", isWorkflow: false, targetIdx: 1},
	"ExecuteChildWorkflow":  {noun: "child workflow", isWorkflow: true, targetIdx: 1},
	"NewContinueAsNewError": {noun: "workflow", isWorkflow: true, targetIdx: 1},
}

// clientEntries are the client.Client methods this analyzer understands, keyed by
// method name. The target index differs per method: ExecuteWorkflow(ctx, options,
// target, args...) names it third; SignalWithStartWorkflow(ctx, id, signalName,
// signalArg, options, target, args...) names it sixth.
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
// top-level non-serializable parameter or return. A target we cannot resolve to a
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

// checkParams flags each user-supplied parameter whose type can't serialize,
// skipping the framework-injected leading context. The parameter number in the
// message is 1-based over the user parameters, so it matches what the author
// writes after the context.
func (c *checker) checkParams(pass *analysis.Pass, target ast.Expr, e entry, name string, sig *types.Signature) {
	params := sig.Params()
	skip := temporalsdk.SkipCount(sig, e.isWorkflow)
	for i := skip; i < params.Len(); i++ {
		c.report(pass, target, e.noun, name, "parameter", i-skip+1, argType(sig, i))
	}
}

// checkResults flags each result whose type can't serialize, skipping a single
// trailing error (the conventional last return of an activity/workflow).
func (c *checker) checkResults(pass *analysis.Pass, target ast.Expr, e entry, name string, sig *types.Signature) {
	results := sig.Results()
	n := results.Len()
	if n > 0 && isError(results.At(n-1).Type()) {
		n--
	}
	for i := 0; i < n; i++ {
		c.report(pass, target, e.noun, name, "return", i+1, results.At(i).Type())
	}
}

// report emits at most one diagnostic for a single parameter or return type. The
// chan/func case is always reported; the empty-struct case only when opted in.
// A type can be at most one of the two, so order does not matter.
func (c *checker) report(pass *analysis.Pass, target ast.Expr, noun, name, kind string, num int, t types.Type) {
	if isUnencodable(t) {
		c.reportf(pass, target, "%s %q %s %d has type %s; %s (%s)",
			noun, name, kind, num, typeStr(t), explainUnencodable, tagUnencodable)
		return
	}
	if c.emptyStruct && isEmptyStruct(t) {
		c.reportf(pass, target, "%s %q %s %d has type %s with no exported fields; %s (%s)",
			noun, name, kind, num, typeStr(t), explainEmptyStruct, tagEmptyStruct)
	}
}

// argType returns the type of a single value passed for parameter i: the element
// type for the final variadic parameter (so ...chan int is checked as chan int,
// the type of each actual argument), otherwise the parameter type itself.
func argType(sig *types.Signature, i int) types.Type {
	t := sig.Params().At(i).Type()
	if sig.Variadic() && i == sig.Params().Len()-1 {
		if s, ok := t.(*types.Slice); ok {
			return s.Elem()
		}
	}
	return t
}

// isUnencodable reports whether t is a type the DataConverter can never serialize:
// a channel or a function. encoding/json returns an "unsupported type" error for
// both, so a parameter or result of that type is always a bug. The check is
// intentionally shallow -- a struct that merely contains a chan field, or a
// []chan, is not flagged -- so it stays false-positive-free.
func isUnencodable(t types.Type) bool {
	switch types.Unalias(t).Underlying().(type) {
	case *types.Chan, *types.Signature:
		return true
	}
	return false
}

// isEmptyStruct reports whether t is a struct that has fields but none exported,
// and does not implement json.Marshaler. JSON marshals only exported fields, so
// such a struct encodes to "{}" and all of its data is silently lost. A type that
// implements json.Marshaler controls its own encoding and is excluded. A fieldless
// struct{} carries no data and round-trips fine, so it is not flagged.
func isEmptyStruct(t types.Type) bool {
	s, ok := types.Unalias(t).Underlying().(*types.Struct)
	if !ok || s.NumFields() == 0 {
		return false
	}
	for i := 0; i < s.NumFields(); i++ {
		if s.Field(i).Exported() {
			return false
		}
	}
	return !implementsJSONMarshaler(t)
}

// implementsJSONMarshaler reports whether t implements json.Marshaler. We inspect
// the method set of *t, which includes both value- and pointer-receiver methods,
// so a MarshalJSON declared on either receiver counts. Matching by method shape
// rather than the json.Marshaler type itself keeps the analyzer from importing
// encoding/json (the same reason it never imports the Temporal SDK).
func implementsJSONMarshaler(t types.Type) bool {
	ms := types.NewMethodSet(types.NewPointer(t))
	for i := 0; i < ms.Len(); i++ {
		if isMarshalJSON(ms.At(i).Obj()) {
			return true
		}
	}
	return false
}

// isMarshalJSON reports whether obj is a method with the json.Marshaler shape:
// MarshalJSON() ([]byte, error).
func isMarshalJSON(obj types.Object) bool {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Name() != "MarshalJSON" {
		return false
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Params().Len() != 0 || sig.Results().Len() != 2 {
		return false
	}
	return isByteSlice(sig.Results().At(0).Type()) && isError(sig.Results().At(1).Type())
}

// isByteSlice reports whether t is []byte.
func isByteSlice(t types.Type) bool {
	s, ok := t.Underlying().(*types.Slice)
	if !ok {
		return false
	}
	b, ok := s.Elem().Underlying().(*types.Basic)
	return ok && b.Kind() == types.Uint8
}

// errorType is the universe error, used to recognize the trailing error return
// and the second result of MarshalJSON.
var errorType = types.Universe.Lookup("error").Type()

func isError(t types.Type) bool {
	return types.Identical(t, errorType)
}

// typeStr renders a type using short package names (workflow.Context, not the full
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
