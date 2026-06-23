package workflowscope

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
)

func TestFuncBody(t *testing.T) {
	if body, ft := FuncBody(&ast.FuncDecl{Body: &ast.BlockStmt{}, Type: &ast.FuncType{}}); body == nil || ft == nil {
		t.Error("FuncBody(*ast.FuncDecl) should return its body and type")
	}
	if body, ft := FuncBody(&ast.FuncLit{Body: &ast.BlockStmt{}, Type: &ast.FuncType{}}); body == nil || ft == nil {
		t.Error("FuncBody(*ast.FuncLit) should return its body and type")
	}
	if body, ft := FuncBody(&ast.Ident{}); body != nil || ft != nil {
		t.Errorf("FuncBody(non-func) = (%v, %v), want (nil, nil)", body, ft)
	}
}

func TestIsWorkflowFunc_NoParams(t *testing.T) {
	pass := &analysis.Pass{TypesInfo: &types.Info{}}
	if IsWorkflowFunc(pass, nil) {
		t.Error("IsWorkflowFunc(nil type) should be false")
	}
	if IsWorkflowFunc(pass, &ast.FuncType{Params: &ast.FieldList{}}) {
		t.Error("IsWorkflowFunc(no params) should be false")
	}
}

// setup parses src and maps every parameter type identifier named "Ctx" to a
// synthetic workflow.Context, so IsWorkflowFunc treats those funcs as workflows.
func setup(t *testing.T, src string) (*analysis.Pass, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "src.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	wfCtx := mkContext()
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
	ast.Inspect(file, func(n ast.Node) bool {
		ft, ok := n.(*ast.FuncType)
		if !ok || ft.Params == nil {
			return true
		}
		for _, field := range ft.Params.List {
			if id, ok := field.Type.(*ast.Ident); ok && id.Name == "Ctx" {
				info.Types[field.Type] = types.TypeAndValue{Type: wfCtx}
			}
		}
		return true
	})
	return &analysis.Pass{TypesInfo: info}, file
}

func mkContext() types.Type {
	pkg := types.NewPackage(temporalsdk.InternalPkg, "internal")
	obj := types.NewTypeName(token.NoPos, pkg, "Context", nil)
	return types.NewNamed(obj, types.Typ[types.Int], nil)
}

func TestIsWorkflowFunc_Match(t *testing.T) {
	pass, file := setup(t, "package p\nfunc WF(ctx Ctx) {}\nfunc plain(x int) {}\n")
	got := map[string]bool{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got[fn.Name.Name] = IsWorkflowFunc(pass, fn.Type)
	}
	if !got["WF"] {
		t.Error("WF(ctx Ctx) should be a workflow func")
	}
	if got["plain"] {
		t.Error("plain(x int) should not be a workflow func")
	}
}

func TestWalk(t *testing.T) {
	// WF is a workflow; its nested closure inner also takes a Ctx, but Walk must
	// not descend into it (it would be a double report). plain is not a workflow.
	src := "package p\n" +
		"func WF(ctx Ctx) {\n\tinner := func(c2 Ctx) {}\n\t_ = inner\n}\n" +
		"func plain(x int) {}\n"
	pass, file := setup(t, src)

	var bodies int
	Walk(pass, file, func(body *ast.BlockStmt) { bodies++ })
	if bodies != 1 {
		t.Errorf("Walk reported %d workflow bodies, want 1 (nested closure must not be re-entered)", bodies)
	}
}
