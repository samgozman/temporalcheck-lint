package optionscontext

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// kind identifies one of the three execution flavors, each with its own options
// type, its own With*Options helper, and its own Execute* function.
type kind uint8

const (
	kindActivity kind = iota
	kindLocalActivity
	kindChild
)

// kindSet is a set of kinds applied to a context, one bit per kind.
type kindSet uint8

func (k kind) bit() kindSet { return 1 << k }

// has reports whether the set contains k.
func (s kindSet) has(k kind) bool { return s&k.bit() != 0 }

// helper is the With*Options function name that applies this kind, used in the
// diagnostic both to name the conflict and to spell out the fix.
func (k kind) helper() string {
	switch k {
	case kindActivity:
		return "WithActivityOptions"
	case kindLocalActivity:
		return "WithLocalActivityOptions"
	default: // kindChild
		return "WithChildOptions"
	}
}

// noun describes the options this kind reads, for the diagnostic's "the X options
// never apply" clause.
func (k kind) noun() string {
	switch k {
	case kindActivity:
		return "activity"
	case kindLocalActivity:
		return "local activity"
	default: // kindChild
		return "child workflow"
	}
}

// withKind reports whether expr is a workflow.With*Options call, returning the
// kind it applies and the context variable it derives from (nil if that base is
// not a plain identifier we can track).
func (wc *walkCtx) withKind(expr ast.Expr) (kind, *types.Var, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return 0, nil, false
	}
	fn := wc.calleeFunc(call)
	if fn == nil || fn.Pkg() == nil || fn.Pkg().Path() != workflowPkg {
		return 0, nil, false
	}
	k, ok := withFuncs[fn.Name()]
	if !ok {
		return 0, nil, false
	}
	var base *types.Var
	if len(call.Args) > 0 {
		base = wc.identVar(call.Args[0])
	}
	return k, base, true
}

// executeKind reports whether call is a workflow.Execute* function, returning the
// option kind it reads out of the context it receives.
func (wc *walkCtx) executeKind(call *ast.CallExpr) (kind, bool) {
	fn := wc.calleeFunc(call)
	if fn == nil || fn.Pkg() == nil || fn.Pkg().Path() != workflowPkg {
		return 0, false
	}
	k, ok := executeFuncs[fn.Name()]
	return k, ok
}

// scanExecute checks every workflow.Execute* call within node against the current
// flow facts. It does not descend into nested function literals: a captured
// context is out of scope (its variable is poisoned), and a closure's own context
// is analyzed when its enclosing function declaration is walked.
func (wc *walkCtx) scanExecute(node ast.Node, st *state) {
	if node == nil {
		return
	}
	ast.Inspect(node, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if k, ok := wc.executeKind(call); ok {
			wc.checkConflict(call, k, st)
		}
		return true
	})
}

// checkConflict fires when the context passed to an Execute* call carries a
// conflicting options kind with no matching one in sight. It is deliberately
// silent on absence: an untracked or empty context (a bare parameter, an opaque
// reassignment) is left alone.
func (wc *walkCtx) checkConflict(call *ast.CallExpr, needed kind, st *state) {
	// executeKind matched a real workflow.Execute*, whose leading context parameter
	// is required, so a type-checked call always has Args[0].
	v := wc.identVar(call.Args[0])
	if v == nil || wc.poisoned[v] {
		return
	}
	info, ok := st.applied[v]
	if !ok || info.set == 0 {
		return // unknown: never fire on absence
	}
	if info.set.has(needed) {
		return // the correct helper appears in the visible chain
	}

	// Honor //nolint ourselves so suppression works the same way in
	// standalone/analysistest runs, not only under golangci-lint. Checked after
	// confirming a real conflict, so unrelated calls cost nothing.
	if wc.nolint.suppressesCall(wc.pass.Fset, call) {
		return
	}

	// Seen contradiction: the context carries a conflicting helper and the matching
	// one was never applied. last is necessarily a conflicting kind (needed is not
	// in the set), so it names the culprit precisely.
	conflict := info.last
	fn := wc.calleeFunc(call)
	wc.pass.Reportf(call.Lparen,
		"%s: this ctx is configured with %s, not %s, so the %s options never apply; derive it with ctx = workflow.%s(ctx, opts) (%s)",
		fn.Name(), conflict.helper(), needed.helper(), needed.noun(), needed.helper(), tagOptionsContext)
}

// calleeFunc resolves a call's callee to the function it names, via Uses (not the
// source text) so aliased imports of the workflow package still match. It returns
// nil for non-selector calls and for selectors whose object is not a function
// (e.g. a func-typed field).
func (wc *walkCtx) calleeFunc(call *ast.CallExpr) *types.Func {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	fn, _ := wc.pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	return fn
}

// identVar returns the variable an expression names, or nil if the expression is
// not a plain identifier bound to a variable (a struct field, a call result, the
// blank identifier). Only plain variables are trackable.
func (wc *walkCtx) identVar(expr ast.Expr) *types.Var {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return nil
	}
	v, _ := wc.pass.TypesInfo.ObjectOf(id).(*types.Var)
	return v
}

// collectPoisoned finds every variable assigned inside a function literal in the
// body. Such a variable may be a captured context reconfigured by the closure at
// an unknown time, so we refuse to track it at all -- the closure-capture bail.
func collectPoisoned(pass *analysis.Pass, body *ast.BlockStmt) map[*types.Var]bool {
	poisoned := map[*types.Var]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		lit, ok := n.(*ast.FuncLit)
		if !ok {
			return true
		}
		ast.Inspect(lit.Body, func(m ast.Node) bool {
			for _, target := range assignTargets(m) {
				id, ok := target.(*ast.Ident)
				if !ok {
					continue
				}
				if v, _ := pass.TypesInfo.ObjectOf(id).(*types.Var); v != nil {
					poisoned[v] = true
				}
			}
			return true
		})
		return true
	})
	return poisoned
}

// resetAssigned drops to "unknown" every tracked context variable assigned
// anywhere within node. Called after a control-flow construct: since we cannot
// know which branch executed, any value assigned inside is no longer certain.
func (wc *walkCtx) resetAssigned(node ast.Node, st *state) {
	ast.Inspect(node, func(n ast.Node) bool {
		for _, target := range assignTargets(n) {
			if v := wc.identVar(target); v != nil {
				delete(st.applied, v)
			}
		}
		return true
	})
}

// assignTargets returns the identifiers a statement assigns to, across the
// assignment forms that can carry a context variable.
func assignTargets(n ast.Node) []ast.Expr {
	switch s := n.(type) {
	case *ast.AssignStmt:
		return s.Lhs
	case *ast.IncDecStmt:
		return []ast.Expr{s.X}
	case *ast.RangeStmt:
		var out []ast.Expr
		if s.Key != nil {
			out = append(out, s.Key)
		}
		if s.Value != nil {
			out = append(out, s.Value)
		}
		return out
	default:
		return nil
	}
}
