package activitytimeout

import (
	"go/ast"
	"go/types"
)

// The option structs are declared in the SDK's internal package and re-exported
// from workflow as aliases (type ActivityOptions = internal.ActivityOptions),
// mirroring workflow.Context. We match by path through go/types -- resolving the
// alias to its internal definition -- so aliased imports resolve and we never
// import the SDK.
const (
	workflowPkg = "go.temporal.io/sdk/workflow"
	internalPkg = "go.temporal.io/sdk/internal"
)

// The two timeout fields the checks reason about. Both ActivityOptions and
// LocalActivityOptions carry them.
const (
	fieldStartToClose    = "StartToCloseTimeout"
	fieldScheduleToClose = "ScheduleToCloseTimeout"
)

// requiredTimeouts are the option fields Temporal requires at least one of: an
// activity with neither StartToCloseTimeout nor ScheduleToCloseTimeout set is
// rejected at run time.
var requiredTimeouts = []string{fieldStartToClose, fieldScheduleToClose}

// optionTypeName returns the option-struct name -- "ActivityOptions" or
// "LocalActivityOptions" -- when t is that workflow type, and false for anything
// else. types.Unalias resolves the workflow alias to its internal definition, so
// the literal's type matches whether the type checker surfaces it as the alias or
// the resolved named type (gotypesalias on or off). Matching the package path (not
// the literal's source text) means an aliased import resolves the same way.
func optionTypeName(t types.Type) (string, bool) {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return "", false
	}
	obj := named.Obj()
	if obj.Pkg() == nil {
		return "", false
	}
	switch obj.Pkg().Path() {
	case workflowPkg, internalPkg:
		// The type lives in one of the SDK packages we match; check the name below.
	default:
		return "", false
	}
	switch obj.Name() {
	case "ActivityOptions", "LocalActivityOptions":
		return obj.Name(), true
	default:
		return "", false
	}
}

// keyedFields returns the set of field names a keyed composite literal sets. ok is
// false for two shapes we deliberately skip rather than risk a false positive: an
// empty literal (no elements), which is typically populated field-by-field after
// construction where this literal-only inspection can't see it; and a positional
// literal, whose elements carry no field names to test without the struct layout.
func keyedFields(lit *ast.CompositeLit) (fields map[string]bool, ok bool) {
	if len(lit.Elts) == 0 {
		return nil, false
	}
	fields = make(map[string]bool, len(lit.Elts))
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return nil, false // positional literal
		}
		// Keys in a struct literal are field identifiers; ignore anything else
		// defensively rather than assume.
		if id, ok := kv.Key.(*ast.Ident); ok {
			fields[id.Name] = true
		}
	}
	return fields, true
}

// hasRequiredTimeout reports whether the literal set at least one required timeout
// field. Presence of the key is enough -- we don't evaluate its value (a literal
// 0, a variable, an expression), which keeps the check statically reliable and
// false-positive-free at the cost of not catching an explicit `: 0`.
func hasRequiredTimeout(fields map[string]bool) bool {
	for _, name := range requiredTimeouts {
		if fields[name] {
			return true
		}
	}
	return false
}

// scheduleToCloseOnly reports whether the literal bounds the whole activity with
// ScheduleToCloseTimeout but omits StartToCloseTimeout, leaving a single attempt
// unbounded. Such a literal satisfies hasRequiredTimeout (so it is never the
// always-on diagnostic), but the recommended practice is to also bound each
// attempt with StartToCloseTimeout -- which the opt-in require-start-to-close
// sub-rule nudges. As elsewhere, presence of the key is enough; the value is not
// evaluated.
func scheduleToCloseOnly(fields map[string]bool) bool {
	return fields[fieldScheduleToClose] && !fields[fieldStartToClose]
}
