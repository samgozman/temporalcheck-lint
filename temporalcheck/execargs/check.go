package execargs

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// Diagnostics are suffixed with the source that produced them, so it is clear
// which setting controls a given report. "arity" is the always-on baseline; the
// others name the opt-in setting that surfaced the mismatch.
const (
	tagArity          = "arity"
	tagStrictTypes    = "strict-types"
	tagStrictPointers = "strict-pointers"
)

// checkSignature matches the call-site arguments against the resolved target
// signature, accounting for the framework-injected leading parameter.
func (c *checker) checkSignature(
	pass *analysis.Pass,
	call *ast.CallExpr,
	entry string,
	k kind,
	sig *types.Signature,
	args []ast.Expr,
) {
	params := sig.Params()
	skip := skipCount(sig, k)
	name := targetName(call.Args[1])

	if sig.Variadic() {
		c.checkVariadic(pass, call, entry, k, name, sig, skip, args)
		return
	}

	want := params.Len() - skip
	if want < 0 {
		want = 0
	}
	if len(args) != want {
		pass.Reportf(call.Lparen, "%s: %s %q expects %d %s, got %d (%s)",
			entry, noun(k), name, want, argWord(want), len(args), tagArity)
		return
	}
	if !c.strictTypes {
		return
	}
	for i, arg := range args {
		c.checkAssignable(pass, arg, entry, name, i+1, params.At(skip+i).Type())
	}
}

// checkVariadic handles a variadic target: a fixed prefix of parameters
// followed by a final ...T that absorbs the trailing arguments.
func (c *checker) checkVariadic(
	pass *analysis.Pass,
	call *ast.CallExpr,
	entry string,
	k kind,
	name string,
	sig *types.Signature,
	skip int,
	args []ast.Expr,
) {
	params := sig.Params()
	variadicIdx := params.Len() - 1 // the variadic parameter is always last
	fixed := variadicIdx - skip
	if fixed < 0 {
		fixed = 0
	}

	if len(args) < fixed {
		pass.Reportf(call.Lparen, "%s: %s %q expects at least %d %s, got %d (%s)",
			entry, noun(k), name, fixed, argWord(fixed), len(args), tagArity)
		return
	}
	if !c.strictTypes {
		return
	}
	for i := 0; i < fixed; i++ {
		c.checkAssignable(pass, args[i], entry, name, i+1, params.At(skip+i).Type())
	}
	slice, ok := params.At(variadicIdx).Type().(*types.Slice)
	if !ok {
		return
	}
	elem := slice.Elem()
	for i := fixed; i < len(args); i++ {
		c.checkAssignable(pass, args[i], entry, name, i+1, elem)
	}
}

func (c *checker) checkAssignable(pass *analysis.Pass, arg ast.Expr, entry, name string, pos int, want types.Type) {
	got := pass.TypesInfo.TypeOf(arg)
	if got == nil || want == nil {
		return
	}
	if types.AssignableTo(got, want) {
		return
	}
	// Attribute the mismatch to the setting that surfaced it, so the message
	// tells you which knob to turn. A difference that is only pointer indirection
	// (T vs *T, []T vs []*T) is allowed unless StrictPointers is set; anything
	// else is a genuine type mismatch under StrictTypes.
	setting := tagStrictTypes
	if pointerInsensitiveMatch(got, want) {
		if !c.strictPointers {
			return
		}
		setting = tagStrictPointers
	}
	pass.Reportf(arg.Pos(), "%s: arg %d of %q has type %s, want %s (%s)",
		entry, pos, name, typeStr(got), typeStr(want), setting)
}

func pointerInsensitiveMatch(got, want types.Type) bool {
	if types.AssignableTo(deref(got), deref(want)) {
		return true
	}
	gs, gok := got.Underlying().(*types.Slice)
	ws, wok := want.Underlying().(*types.Slice)
	if gok && wok {
		return pointerInsensitiveMatch(gs.Elem(), ws.Elem())
	}
	return false
}

// deref strips one level of pointer indirection, leaving non-pointers untouched.
func deref(t types.Type) types.Type {
	if p, ok := t.Underlying().(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}

// skipCount returns how many leading parameters Temporal injects at run time and
// that the caller therefore must not supply.
func skipCount(sig *types.Signature, k kind) int {
	if sig.Params().Len() == 0 {
		return 0
	}
	first := sig.Params().At(0).Type()
	switch k {
	case kindChildWorkflow:
		if isWorkflowContext(first) {
			return 1
		}
	case kindActivity:
		if named(first, contextPkg, "Context") {
			return 1
		}
	}
	return 0
}

// isWorkflowContext reports whether t is workflow.Context. The SDK publishes it
// as `type Context = internal.Context`, so depending on the gotypesalias mode t
// is either the alias (named in workflowPkg) or the resolved internal named type
// (named in workflowInternalPkg); both must count as the injected context.
func isWorkflowContext(t types.Type) bool {
	// The resolved type lives in the internal package (or the workflow package
	// itself, for a direct declaration like the test stub once used).
	if named(types.Unalias(t), workflowInternalPkg, "Context") {
		return true
	}
	// The unresolved alias is named in the public workflow package.
	return named(t, workflowPkg, "Context")
}

func noun(k kind) string {
	if k == kindChildWorkflow {
		return "child workflow"
	}
	return "activity"
}

func argWord(n int) string {
	if n == 1 {
		return "argument"
	}
	return "arguments"
}

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

// named reports whether t is the named type pkgPath.name. It accepts both
// defined types and aliases, since both carry an *types.TypeName.
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

// typeStr renders a type using short package names (context.Context, not the
// full import path).
func typeStr(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Name() })
}
