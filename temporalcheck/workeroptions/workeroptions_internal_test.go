package workeroptions

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// TestCheckRequireOptions_TooFewArgs covers the defensive arity guard in
// checkRequireOptions. A compiling worker.New call always has three arguments, so
// this shape is unreachable through analysistest fixtures; it is exercised here
// with a synthetic call whose callee resolves to worker.New but which carries only
// two arguments (as a malformed/partial AST could). The guard must return rather
// than index call.Args[2] out of range, and must report nothing.
func TestCheckRequireOptions_TooFewArgs(t *testing.T) {
	pkg := types.NewPackage(workerPkg, "worker")
	fn := types.NewFunc(token.NoPos, pkg, "New", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	sel := ast.NewIdent("New")
	call := &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: ast.NewIdent("worker"), Sel: sel},
		Args: []ast.Expr{ast.NewIdent("c"), ast.NewIdent("q")}, // worker.New needs three
	}
	pass := &analysis.Pass{
		TypesInfo: &types.Info{Uses: map[*ast.Ident]types.Object{sel: fn}},
		Report:    func(analysis.Diagnostic) { t.Fatal("reported a diagnostic for a worker.New call with fewer than three arguments") },
	}

	c := &checker{requireOptions: true}
	c.checkRequireOptions(pass, nolintInfo{}, call)
}

// namedIn builds a named struct type whose object lives in the given package
// path, so the package/name branches of isWorkerOptions can be exercised without
// a full type-checking pass (the matched cases run through the analysistest stub).
func namedIn(path, name string) types.Type {
	pkg := types.NewPackage(path, "p")
	tn := types.NewTypeName(token.NoPos, pkg, name, nil)
	return types.NewNamed(tn, types.NewStruct(nil, nil), nil)
}

func TestIsWorkerOptions(t *testing.T) {
	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{"nil type", nil, false},
		{"basic (non-named) type", types.Typ[types.Int], false},
		{"nil-package named type", types.NewNamed(types.NewTypeName(token.NoPos, nil, "Options", nil), types.NewStruct(nil, nil), nil), false},
		{"worker.Options matches", namedIn(workerPkg, "Options"), true},
		{"internal.WorkerOptions matches", namedIn(internalPkg, "WorkerOptions"), true},
		{"worker package, wrong name", namedIn(workerPkg, "Worker"), false},
		{"internal package, wrong name", namedIn(internalPkg, "Options"), false},
		{"some other package", namedIn("example.com/other", "Options"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWorkerOptions(tt.typ); got != tt.want {
				t.Errorf("isWorkerOptions(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestKeyedFieldValues(t *testing.T) {
	kv := func(key ast.Expr) ast.Expr { return &ast.KeyValueExpr{Key: key, Value: &ast.Ident{Name: "v"}} }

	tests := []struct {
		name     string
		lit      *ast.CompositeLit
		wantKeys []string // expected field names present
	}{
		{
			name:     "empty literal yields no fields",
			lit:      &ast.CompositeLit{},
			wantKeys: nil,
		},
		{
			name:     "positional literal yields no fields",
			lit:      &ast.CompositeLit{Elts: []ast.Expr{&ast.Ident{Name: "x"}}},
			wantKeys: nil,
		},
		{
			name:     "keyed literal collects field names",
			lit:      &ast.CompositeLit{Elts: []ast.Expr{kv(&ast.Ident{Name: "MaxConcurrentWorkflowTaskPollers"}), kv(&ast.Ident{Name: "Identity"})}},
			wantKeys: []string{"MaxConcurrentWorkflowTaskPollers", "Identity"},
		},
		{
			name:     "non-identifier key is ignored",
			lit:      &ast.CompositeLit{Elts: []ast.Expr{kv(&ast.BasicLit{Kind: token.STRING, Value: `"a"`})}},
			wantKeys: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := keyedFieldValues(tt.lit)
			if len(fields) != len(tt.wantKeys) {
				t.Errorf("keyedFieldValues returned %d fields, want %d (%v)", len(fields), len(tt.wantKeys), fields)
			}
			for _, k := range tt.wantKeys {
				if _, ok := fields[k]; !ok {
					t.Errorf("keyedFieldValues missing expected field %q", k)
				}
			}
		})
	}
}
