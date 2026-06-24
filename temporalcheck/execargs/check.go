package execargs

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
	"sort"
	"strings"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
	"golang.org/x/tools/go/analysis"
)

// Diagnostics are suffixed with the source so it's clear which setting produced each report.
const (
	tagArity          = "arity"
	tagStrictTypes    = "strict-types"
	tagStrictPointers = "strict-pointers"
	tagStructShape    = "strict-struct-shape"
	tagStrictTests    = "strict-tests"
)

// checkSignature matches call-site arguments against the resolved target signature,
// accounting for the framework-injected leading parameter.
func (c *checker) checkSignature(
	pass *analysis.Pass,
	call *ast.CallExpr,
	fnName string,
	e entry,
	sig *types.Signature,
	args []ast.Expr,
) {
	params := sig.Params()
	skip := skipCount(sig, e.kind)
	name := targetName(call.Args[e.targetIdx])

	if sig.Variadic() {
		c.checkVariadic(pass, call, fnName, e, name, sig, skip, args)
		return
	}

	want := params.Len() - skip
	if len(args) != want {
		pass.Reportf(call.Lparen, "%s: %s %q expects %d %s, got %d (%s)",
			fnName, e.noun, name, want, argWord(want), len(args), tagArity)
		return
	}
	if !c.typeChecksEnabled() {
		return
	}
	for i, arg := range args {
		c.checkAssignable(pass, arg, fnName, name, i+1, params.At(skip+i).Type())
	}
}

func (c *checker) checkVariadic(
	pass *analysis.Pass,
	call *ast.CallExpr,
	fnName string,
	e entry,
	name string,
	sig *types.Signature,
	skip int,
	args []ast.Expr,
) {
	params := sig.Params()
	variadicIdx := params.Len() - 1
	fixed := variadicIdx - skip

	if len(args) < fixed {
		pass.Reportf(call.Lparen, "%s: %s %q expects at least %d %s, got %d (%s)",
			fnName, e.noun, name, fixed, argWord(fixed), len(args), tagArity)
		return
	}
	if !c.typeChecksEnabled() {
		return
	}
	for i := 0; i < fixed; i++ {
		c.checkAssignable(pass, args[i], fnName, name, i+1, params.At(skip+i).Type())
	}
	// A variadic parameter's type is always a slice.
	elem := params.At(variadicIdx).Type().(*types.Slice).Elem()
	for i := fixed; i < len(args); i++ {
		c.checkAssignable(pass, args[i], fnName, name, i+1, elem)
	}
}

func (c *checker) checkAssignable(pass *analysis.Pass, arg ast.Expr, fnName, name string, pos int, want types.Type) {
	got := pass.TypesInfo.TypeOf(arg)
	if got == nil || want == nil {
		return
	}
	if types.AssignableTo(got, want) {
		return
	}

	// A difference that is only pointer indirection (T vs *T, []T vs []*T) is
	// allowed unless StrictPointers is set.
	if pointerInsensitiveMatch(got, want) {
		if c.strictPointers {
			c.reportf(pass, arg, "%s: arg %d of %q has type %s, want %s (%s)",
				fnName, pos, name, typeStr(got), typeStr(want), tagStrictPointers)
		}
		return
	}

	// Two distinct struct types: Temporal serializes by field name, so they may round-trip.
	if gs, ws := structUnder(got), structUnder(want); gs != nil && ws != nil {
		c.reportStructMismatch(pass, arg, fnName, name, pos, got, want, compareStructs(gs, ws))
		return
	}

	if c.strictTypes {
		c.reportf(pass, arg, "%s: arg %d of %q has type %s, want %s (%s)",
			fnName, pos, name, typeStr(got), typeStr(want), tagStrictTypes)
	}
}

// reportStructMismatch emits the right diagnostic for passing struct type got
// where struct type want is expected, given how their fields line up.
func (c *checker) reportStructMismatch(pass *analysis.Pass, arg ast.Expr, fnName, name string, pos int, got, want types.Type, d structDiff) {
	switch {
	case d.conflict != nil:
		if c.strictTypes || c.structShape {
			c.reportf(pass, arg, "%s: arg %d of %q sends %s, target wants %s -- field %q is incompatible (%s vs %s) (%s)",
				fnName, pos, name, typeStr(got), typeStr(want), d.conflict.field,
				typeStr(d.conflict.got), typeStr(d.conflict.want), tagStrictTypes)
		}
	case d.overlap == 0:
		if c.strictTypes || c.structShape {
			c.reportf(pass, arg, "%s: arg %d of %q sends %s, target wants %s -- no fields in common (%s)",
				fnName, pos, name, typeStr(got), typeStr(want), tagStrictTypes)
		}
	default:
		// Wire-compatible but distinct: silently drops/zeroes mismatched fields.
		if c.structShape {
			c.reportf(pass, arg, "%s: arg %d of %q sends %s, target wants %s -- %s (%s)",
				fnName, pos, name, typeStr(got), typeStr(want), driftPhrase(d), tagStructShape)
		}
	}
}

// reportf anchors a diagnostic at the argument position.
func (c *checker) reportf(pass *analysis.Pass, arg ast.Expr, format string, args ...any) {
	pass.Reportf(arg.Pos(), format, args...)
}

// structUnder returns the struct t denotes after stripping one pointer, or nil.
func structUnder(t types.Type) *types.Struct {
	if p, ok := types.Unalias(t).Underlying().(*types.Pointer); ok {
		t = p.Elem()
	}
	if s, ok := types.Unalias(t).Underlying().(*types.Struct); ok {
		return s
	}
	return nil
}

// structDiff describes how a sent struct's fields line up with a wanted struct's (JSON names).
type structDiff struct {
	overlap  int            // shared fields whose types are compatible
	drops    []string       // sent fields the target ignores
	unset    []string       // target fields left zero because the sender omits them
	conflict *fieldConflict // first shared field whose types are incompatible
}

type fieldConflict struct {
	field     string // Go field name on the target
	got, want types.Type
}

// compareStructs matches the two structs' serialized fields by JSON name.
func compareStructs(got, want *types.Struct) structDiff {
	gf, wf := structFields(got), structFields(want)
	var d structDiff
	for _, jsonName := range sortedKeys(gf) {
		gi := gf[jsonName]
		wi, shared := wf[jsonName]
		if !shared {
			d.drops = append(d.drops, gi.goName)
			continue
		}
		if !fieldsCompatible(gi.typ, wi.typ) {
			if d.conflict == nil {
				d.conflict = &fieldConflict{field: wi.goName, got: gi.typ, want: wi.typ}
			}
			continue
		}
		d.overlap++
	}
	for _, jsonName := range sortedKeys(wf) {
		if _, shared := gf[jsonName]; !shared {
			d.unset = append(d.unset, wf[jsonName].goName)
		}
	}
	return d
}

// fieldsCompatible mirrors the top-level check: T vs *T is not a conflict.
func fieldsCompatible(a, b types.Type) bool {
	return types.AssignableTo(a, b) || pointerInsensitiveMatch(a, b)
}

type fieldEntry struct {
	goName string
	typ    types.Type
}

// structFields maps each exported, non-embedded field's JSON name to its entry.
// Fields tagged json:"-" are excluded. Embedded fields are not modeled.
func structFields(s *types.Struct) map[string]fieldEntry {
	out := make(map[string]fieldEntry, s.NumFields())
	for i := 0; i < s.NumFields(); i++ {
		f := s.Field(i)
		if !f.Exported() || f.Embedded() {
			continue
		}
		name, ok := jsonName(f.Name(), s.Tag(i))
		if !ok {
			continue
		}
		out[name] = fieldEntry{goName: f.Name(), typ: f.Type()}
	}
	return out
}

// jsonName returns the wire (JSON) field name and whether the field is serialized,
// mirroring encoding/json's reading of the `json` struct tag.
func jsonName(goName, tag string) (string, bool) {
	v, ok := reflect.StructTag(tag).Lookup("json")
	if !ok {
		return goName, true
	}
	if v == "-" {
		return "", false
	}
	if i := strings.IndexByte(v, ','); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return goName, true
	}
	return v, true
}

func sortedKeys(m map[string]fieldEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// driftPhrase describes what silently changes when a wire-compatible but distinct struct is passed.
func driftPhrase(d structDiff) string {
	switch {
	case len(d.drops) > 0 && len(d.unset) > 0:
		return fmt.Sprintf("serializes by field name but drops {%s} and leaves {%s} unset",
			strings.Join(d.drops, ", "), strings.Join(d.unset, ", "))
	case len(d.drops) > 0:
		return fmt.Sprintf("serializes by field name but drops {%s}", strings.Join(d.drops, ", "))
	case len(d.unset) > 0:
		return fmt.Sprintf("serializes by field name but leaves {%s} unset", strings.Join(d.unset, ", "))
	default:
		return "has identical fields but is a distinct Go type"
	}
}

func pointerInsensitiveMatch(got, want types.Type) bool {
	if types.AssignableTo(temporalsdk.Deref(got), temporalsdk.Deref(want)) {
		return true
	}
	gs, gok := got.Underlying().(*types.Slice)
	ws, wok := want.Underlying().(*types.Slice)
	if gok && wok {
		return pointerInsensitiveMatch(gs.Elem(), ws.Elem())
	}
	return false
}

// skipCount returns how many leading parameters Temporal injects (0 or 1).
func skipCount(sig *types.Signature, k kind) int {
	if sig.Params().Len() == 0 {
		return 0
	}
	first := sig.Params().At(0).Type()
	switch k {
	case kindWorkflow:
		if temporalsdk.IsWorkflowContext(first) {
			return 1
		}
	case kindActivity:
		if temporalsdk.Named(first, temporalsdk.ContextPkg, "Context") {
			return 1
		}
	}
	return 0
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

// typeStr renders a type with short package names (e.g. context.Context).
func typeStr(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Name() })
}
