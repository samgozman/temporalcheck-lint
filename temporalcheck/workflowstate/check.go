package workflowstate

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// isWorkflowFunc reports whether a function with the given type is a Temporal
// workflow definition: its first parameter is workflow.Context.
func isWorkflowFunc(pass *analysis.Pass, ft *ast.FuncType) bool {
	if ft == nil || ft.Params == nil || len(ft.Params.List) == 0 {
		return false
	}
	// A field may declare several names sharing one type; the first parameter's
	// type is that field's type regardless.
	return isWorkflowContext(pass.TypesInfo.TypeOf(ft.Params.List[0].Type))
}

// isWorkflowContext reports whether t is workflow.Context. The SDK publishes it
// as `type Context = internal.Context`, so depending on the gotypesalias mode t
// is either the alias (named in workflowPkg) or the resolved internal named type
// (named in workflowInternalPkg); both count.
func isWorkflowContext(t types.Type) bool {
	if t == nil {
		return false
	}
	if named(types.Unalias(t), workflowInternalPkg, "Context") {
		return true
	}
	return named(t, workflowPkg, "Context")
}

// named reports whether t is the named (or aliased) type pkgPath.name.
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

// reportMutations walks a workflow definition's body -- including the closures
// lexically nested in it, since those run as part of the same workflow execution
// -- and reports every assignment or ++/-- whose root object is a package-level
// variable.
func (c *checker) reportMutations(pass *analysis.Pass, nolint nolintInfo, body *ast.BlockStmt) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			// `:=` always introduces fresh locals in the current scope; it can never
			// name a package-level variable, so it is not a mutation of shared state.
			if s.Tok == token.DEFINE {
				return true
			}
			for _, lhs := range s.Lhs {
				c.checkTarget(pass, nolint, s, lhs)
			}
		case *ast.IncDecStmt:
			c.checkTarget(pass, nolint, s, s.X)
		}
		return true
	})
}

// checkTarget reports a mutation when target's root object is a package-level
// variable. stmt is the enclosing statement, used for diagnostic position and
// //nolint suppression so a directive on that line works.
func (c *checker) checkTarget(pass *analysis.Pass, nolint nolintInfo, stmt ast.Node, target ast.Expr) {
	v := rootVar(pass, target)
	if v == nil || !isPackageVar(v) {
		return
	}
	// Honor //nolint ourselves so suppression works the same way in
	// standalone/analysistest runs, not only under golangci-lint. Checked after
	// confirming a real mutation, so unrelated statements cost nothing.
	if nolint.suppressesNode(pass.Fset, stmt) {
		return
	}
	pass.Reportf(target.Pos(),
		"mutates package-level variable %s from workflow code; shared mutable state breaks replay determinism and races across workflow executions (%s)",
		v.Name(), tagGlobalMutation)
}

// rootVar resolves the variable an assignment target is rooted at, peeling
// parentheses, pointer dereferences, indexing and field selection. It returns nil
// when the root is not a plain variable we can name (a call result, the blank
// identifier, a package-qualified function) -- those are skipped rather than
// guessed at.
func rootVar(pass *analysis.Pass, expr ast.Expr) *types.Var {
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return rootVar(pass, e.X)
	case *ast.StarExpr:
		return rootVar(pass, e.X)
	case *ast.IndexExpr:
		return rootVar(pass, e.X)
	case *ast.Ident:
		v, _ := pass.TypesInfo.ObjectOf(e).(*types.Var)
		return v
	case *ast.SelectorExpr:
		// A selector is either pkg.Var (the variable lives in another package and
		// is named by the selector itself) or value.Field (the root is value's own
		// root). Distinguish by whether the base names an imported package.
		if id, ok := e.X.(*ast.Ident); ok {
			if _, isPkg := pass.TypesInfo.ObjectOf(id).(*types.PkgName); isPkg {
				v, _ := pass.TypesInfo.ObjectOf(e.Sel).(*types.Var)
				return v
			}
		}
		return rootVar(pass, e.X)
	default:
		return nil
	}
}

// isPackageVar reports whether v is declared at package scope. Package-level
// variables have the package scope as their parent; locals, parameters and
// receivers are parented by a function or block scope, which is exactly the
// capture case we must not flag.
func isPackageVar(v *types.Var) bool {
	return v.Pkg() != nil && v.Parent() == v.Pkg().Scope()
}
