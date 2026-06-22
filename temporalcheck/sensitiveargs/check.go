package sensitiveargs

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

const (
	workflowPkg = "go.temporal.io/sdk/workflow"
	// internalPkg is where the SDK declares workflow.Context, published from the
	// workflow package as an alias (type Context = internal.Context), so the
	// resolved Context type lives here, not in workflowPkg.
	internalPkg = "go.temporal.io/sdk/internal"
	// clientPkg is where the SDK declares the Client interface directly, so
	// client.ExecuteWorkflow's receiver is client.Client. internalPkg.Client is a
	// separate interface client.Client implements; we accept it too, defensively.
	clientPkg  = "go.temporal.io/sdk/client"
	contextPkg = "context"
)

// tagSensitive suffixes every diagnostic, naming the check that produced it.
const tagSensitive = "sensitive"

// explainSensitive is the shared tail of the diagnostic: why a matching name is a
// concern and what to do about it.
const explainSensitive = "Temporal records arguments in durable workflow history — pass an opaque reference and fetch the secret inside the activity instead"

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
// Each names its target as the second argument.
var workflowEntries = map[string]entry{
	"ExecuteActivity":       {noun: "activity", isWorkflow: false, targetIdx: 1},
	"ExecuteLocalActivity":  {noun: "activity", isWorkflow: false, targetIdx: 1},
	"ExecuteChildWorkflow":  {noun: "child workflow", isWorkflow: true, targetIdx: 1},
	"NewContinueAsNewError": {noun: "workflow", isWorkflow: true, targetIdx: 1},
}

// clientEntries are the client.Client methods this analyzer understands. The
// target index differs per method: ExecuteWorkflow(ctx, options, target, args...)
// names it third; SignalWithStartWorkflow(ctx, id, signalName, signalArg, options,
// target, args...) names it sixth.
var clientEntries = map[string]entry{
	"ExecuteWorkflow":         {noun: "workflow", isWorkflow: true, targetIdx: 2},
	"SignalWithStartWorkflow": {noun: "workflow", isWorkflow: true, targetIdx: 5},
}

// entryFor reports whether fn is an Execute*/continue-as-new entry point this
// analyzer inspects. workflow.* are package functions; the client methods are on
// the client.Client interface, so we match them by name and receiver.
func entryFor(fn *types.Func) (entry, bool) {
	if fn.Pkg().Path() == workflowPkg {
		e, ok := workflowEntries[fn.Name()]
		return e, ok
	}
	if e, ok := clientEntries[fn.Name()]; ok &&
		(isReceiver(fn, clientPkg, "Client") || isReceiver(fn, internalPkg, "Client")) {
		return e, true
	}
	return entry{}, false
}

// checkTarget resolves the target reference to its signature and flags any
// user-supplied parameter, or exported field of a struct parameter, whose name
// matches the pattern. A target we cannot resolve to a signature -- a
// string-registered name or any non-function value -- is left alone rather than
// risk a false positive.
func (c *checker) checkTarget(pass *analysis.Pass, nolint nolintInfo, call *ast.CallExpr, e entry) {
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
	if nolint.suppressesCall(pass.Fset, call) {
		return
	}

	c.checkParams(pass, target, e, targetName(target), sig)
}

// checkParams flags each user-supplied parameter (and the exported fields of a
// struct parameter) whose name matches, skipping the framework-injected leading
// context. The parameter number is 1-based over the user parameters, so it matches
// what the author writes after the context.
func (c *checker) checkParams(pass *analysis.Pass, target ast.Expr, e entry, targetNm string, sig *types.Signature) {
	params := sig.Params()
	skip := skipCount(sig, e.isWorkflow)
	for i := skip; i < params.Len(); i++ {
		param := params.At(i)
		num := i - skip + 1

		if c.re.MatchString(param.Name()) {
			pass.Reportf(target.Pos(), "%s %q parameter %d %q matches the sensitive-data pattern; %s (%s)",
				e.noun, targetNm, num, param.Name(), explainSensitive, tagSensitive)
		}

		// A struct parameter carries each of its exported fields into history, so
		// flag any whose name matches. Only exported fields are serialized; the
		// search stays at the top level (no nested structs, slices or maps).
		if s, ok := structFields(param.Type()); ok {
			for j := 0; j < s.NumFields(); j++ {
				f := s.Field(j)
				if f.Exported() && c.re.MatchString(f.Name()) {
					pass.Reportf(target.Pos(), "%s %q parameter %d (type %s) field %q matches the sensitive-data pattern; %s (%s)",
						e.noun, targetNm, num, typeStr(param.Type()), f.Name(), explainSensitive, tagSensitive)
				}
			}
		}
	}
}

// structFields returns the struct underlying t, dereferencing a single pointer
// level (a *PaymentRequest parameter is as exposed as a PaymentRequest), and
// reports whether t is a struct at all. It does not look through slices or maps,
// keeping the field search at the top level.
func structFields(t types.Type) (*types.Struct, bool) {
	s, ok := types.Unalias(deref(t)).Underlying().(*types.Struct)
	return s, ok
}

// skipCount returns how many leading parameters Temporal injects at run time and
// that the author therefore writes but the analyzer must not treat as data: the
// always-injected workflow.Context for a workflow, or the optional context.Context
// for an activity.
func skipCount(sig *types.Signature, isWorkflow bool) int {
	if sig.Params().Len() == 0 {
		return 0
	}
	first := sig.Params().At(0).Type()
	if isWorkflow {
		if isWorkflowContext(first) {
			return 1
		}
		return 0
	}
	if named(first, contextPkg, "Context") {
		return 1
	}
	return 0
}

// isWorkflowContext reports whether t is workflow.Context. The SDK publishes it as
// `type Context = internal.Context`, so depending on the gotypesalias mode t is
// either the alias (named in workflowPkg) or the resolved internal named type
// (named in internalPkg); both must count as the injected context.
func isWorkflowContext(t types.Type) bool {
	if named(types.Unalias(t), internalPkg, "Context") {
		return true
	}
	return named(t, workflowPkg, "Context")
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

// isReceiver reports whether fn is a method whose receiver is (a pointer to) the
// named type pkgPath.name.
func isReceiver(fn *types.Func, pkgPath, name string) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return false
	}
	return named(deref(sig.Recv().Type()), pkgPath, name)
}

// named reports whether t is the named type pkgPath.name. It accepts both defined
// types and aliases, since both carry a *types.TypeName.
func named(t types.Type, pkgPath, name string) bool {
	var obj *types.TypeName
	switch n := t.(type) {
	case *types.Named:
		obj = n.Obj()
	case *types.Alias:
		obj = n.Obj()
	default:
		return false
	}
	return obj != nil && obj.Pkg() != nil &&
		obj.Pkg().Path() == pkgPath && obj.Name() == name
}

// deref strips one level of pointer indirection, leaving non-pointers untouched.
func deref(t types.Type) types.Type {
	if p, ok := t.Underlying().(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}
