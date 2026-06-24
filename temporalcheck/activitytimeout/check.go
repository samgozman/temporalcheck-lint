package activitytimeout

import (
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
	"go/ast"
	"go/types"
)

// The option structs are declared in sdk/internal and re-exported from workflow as aliases.
// types.Unalias() + accepting both package paths handles both gotypesalias modes.
const (
	fieldStartToClose    = "StartToCloseTimeout"
	fieldScheduleToClose = "ScheduleToCloseTimeout"
)

var requiredTimeouts = []string{fieldStartToClose, fieldScheduleToClose}

// optionTypeName returns "ActivityOptions" or "LocalActivityOptions" when t is
// one of those SDK option types (workflow or internal package path).
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
	case temporalsdk.WorkflowPkg, temporalsdk.InternalPkg:
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

// keyedFields returns the field names set by a keyed composite literal.
// Returns false for empty or positional literals.
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
		if id, ok := kv.Key.(*ast.Ident); ok {
			fields[id.Name] = true
		}
	}
	return fields, true
}

// hasRequiredTimeout reports whether at least one required timeout field is set.
// Presence is enough — values are not evaluated.
func hasRequiredTimeout(fields map[string]bool) bool {
	for _, name := range requiredTimeouts {
		if fields[name] {
			return true
		}
	}
	return false
}

// scheduleToCloseOnly reports whether ScheduleToCloseTimeout is set but StartToCloseTimeout is not.
func scheduleToCloseOnly(fields map[string]bool) bool {
	return fields[fieldScheduleToClose] && !fields[fieldStartToClose]
}
