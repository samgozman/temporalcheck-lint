package searchattribute

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strconv"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
	"golang.org/x/tools/go/analysis"
)

// tagSearchAttribute suffixes every diagnostic, naming the check that produced it.
const tagSearchAttribute = "searchattribute"

// upsertFuncs are the workflow.* functions that set search attributes. Both name
// the attribute in their arguments -- the typed one through a key constructor, the
// legacy one through map keys.
var upsertFuncs = map[string]bool{
	"UpsertTypedSearchAttributes": true,
	"UpsertSearchAttributes":      true,
}

// checkWorkflow reports each configured attribute required by fn's params struct
// that fn's body never upserts.
func (c *checker) checkWorkflow(pass *analysis.Pass, nolint nolint.Info, fn *ast.FuncDecl) {
	required := c.requiredAttrs(pass, fn.Type)
	if len(required) == 0 {
		return
	}

	seen, opaque := c.upsertedAttrs(pass, fn.Body)
	// An upsert we cannot read statically (a prebuilt map, a spread slice, a key
	// held in a variable) may well set the required attribute; suppress rather than
	// risk a false positive.
	if opaque {
		return
	}

	// The diagnostic is anchored to the signature, so a //nolint on that line (the
	// plugin is "temporalcheck") suppresses it. Checked once, after we know the
	// workflow is a candidate.
	if nolint.Suppresses(pass.Fset, fn.Name) {
		return
	}

	for _, attr := range sortedKeys(required) {
		if seen[attr] {
			continue
		}
		pass.Reportf(fn.Name.Pos(),
			"workflow %q takes field %q but never upserts the %q search attribute (%s)",
			fn.Name.Name, required[attr], attr, tagSearchAttribute)
	}
}

// requiredAttrs maps each attribute name the workflow must upsert to the exported
// struct field that requires it. It reads the exported fields of each struct
// parameter (dereferencing one pointer level), skipping the leading
// workflow.Context, and matches their normalized names against the configured
// aliases. When several fields map to one attribute, the first field found wins.
func (c *checker) requiredAttrs(pass *analysis.Pass, ft *ast.FuncType) map[string]string {
	required := make(map[string]string)
	// ft.Params.List[0] is the workflow.Context (guaranteed by IsWorkflowFunc); the
	// user parameters follow.
	for _, field := range ft.Params.List[1:] {
		s, ok := structFields(pass.TypesInfo.TypeOf(field.Type))
		if !ok {
			continue
		}
		for i := 0; i < s.NumFields(); i++ {
			f := s.Field(i)
			if !f.Exported() {
				continue
			}
			attr, ok := c.aliases[normalize(f.Name())]
			if !ok {
				continue
			}
			if _, dup := required[attr]; !dup {
				required[attr] = f.Name()
			}
		}
	}
	return required
}

// upsertedAttrs walks the workflow body for search-attribute upsert calls and
// returns the set of attribute names they set as string literals. opaque is true
// when some upsert call sets attributes through a source we cannot read (a spread,
// or a call with no literal names), meaning its attributes are unknown.
func (c *checker) upsertedAttrs(pass *analysis.Pass, body *ast.BlockStmt) (seen map[string]bool, opaque bool) {
	seen = make(map[string]bool)
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isUpsertCall(pass, call) {
			return true
		}
		lits := stringLiterals(call.Args)
		// A spread (Upsert(ctx, updates...)) or an absence of literal names means
		// the attributes are built dynamically and cannot be read here.
		if call.Ellipsis != token.NoPos || len(lits) == 0 {
			opaque = true
		}
		for _, l := range lits {
			seen[l] = true
		}
		return true
	})
	return seen, opaque
}

// isUpsertCall reports whether call is workflow.UpsertTypedSearchAttributes or
// workflow.UpsertSearchAttributes, resolved via Uses so aliased imports match.
func isUpsertCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false
	}
	return fn.Pkg().Path() == temporalsdk.WorkflowPkg && upsertFuncs[fn.Name()]
}

// stringLiterals returns the value of every string literal appearing anywhere in
// the given expressions. The attribute names sit in a key constructor argument
// (typed API) or a map key (legacy API); scanning the whole subtree finds either.
// Over-collecting an unrelated literal only ever suppresses a diagnostic, never
// adds one, which keeps the check false-positive-safe.
func stringLiterals(exprs []ast.Expr) []string {
	var out []string
	for _, e := range exprs {
		ast.Inspect(e, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if v, err := strconv.Unquote(lit.Value); err == nil {
					out = append(out, v)
				}
			}
			return true
		})
	}
	return out
}

// structFields returns the struct underlying t, dereferencing a single pointer
// level, and reports whether t is a struct at all. It does not look through slices
// or maps, keeping the field search at the top level.
func structFields(t types.Type) (*types.Struct, bool) {
	if t == nil {
		return nil, false
	}
	s, ok := types.Unalias(temporalsdk.Deref(t)).Underlying().(*types.Struct)
	return s, ok
}

// sortedKeys returns m's keys in sorted order, so diagnostics on one workflow are
// reported deterministically.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
