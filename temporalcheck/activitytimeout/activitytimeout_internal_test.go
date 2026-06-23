package activitytimeout

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"
)

func TestOptionTypeName(t *testing.T) {
	// A basic (non-named) type is skipped.
	if name, ok := optionTypeName(types.Typ[types.Int]); ok {
		t.Errorf("optionTypeName(int) = %q, true; want skip", name)
	}

	// A named type whose object has no package (a universe-like type) is skipped
	// without panicking on the nil package -- even when its name would otherwise
	// match. (The matched/internal cases are covered by the analysistest stub,
	// whose option types are aliases into the internal package.)
	tn := types.NewTypeName(token.NoPos, nil, "ActivityOptions", nil)
	named := types.NewNamed(tn, types.NewStruct(nil, nil), nil)
	if name, ok := optionTypeName(named); ok {
		t.Errorf("optionTypeName(nil-package named) = %q, true; want skip", name)
	}
}

func TestKeyedFields(t *testing.T) {
	kv := func(key ast.Expr) ast.Expr { return &ast.KeyValueExpr{Key: key, Value: &ast.Ident{Name: "v"}} }

	tests := []struct {
		name     string
		lit      *ast.CompositeLit
		wantOK   bool
		wantKeys []string // expected field names present (when ok)
	}{
		{
			name:   "empty literal is skipped",
			lit:    &ast.CompositeLit{},
			wantOK: false,
		},
		{
			name:   "positional literal is skipped",
			lit:    &ast.CompositeLit{Elts: []ast.Expr{&ast.Ident{Name: "x"}}},
			wantOK: false,
		},
		{
			name:     "keyed literal collects field names",
			lit:      &ast.CompositeLit{Elts: []ast.Expr{kv(&ast.Ident{Name: "StartToCloseTimeout"}), kv(&ast.Ident{Name: "TaskQueue"})}},
			wantOK:   true,
			wantKeys: []string{"StartToCloseTimeout", "TaskQueue"},
		},
		{
			name:     "non-identifier key is ignored, literal still keyed",
			lit:      &ast.CompositeLit{Elts: []ast.Expr{kv(&ast.BasicLit{Kind: token.STRING, Value: `"a"`})}},
			wantOK:   true,
			wantKeys: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, ok := keyedFields(tt.lit)
			if ok != tt.wantOK {
				t.Fatalf("keyedFields ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if len(fields) != len(tt.wantKeys) {
				t.Errorf("keyedFields returned %d fields, want %d (%v)", len(fields), len(tt.wantKeys), fields)
			}
			for _, k := range tt.wantKeys {
				if !fields[k] {
					t.Errorf("keyedFields missing expected field %q", k)
				}
			}
		})
	}
}

func TestHasRequiredTimeout(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]bool
		want   bool
	}{
		{"StartToCloseTimeout satisfies", map[string]bool{"StartToCloseTimeout": true}, true},
		{"ScheduleToCloseTimeout satisfies", map[string]bool{"ScheduleToCloseTimeout": true}, true},
		{"both satisfy", map[string]bool{"StartToCloseTimeout": true, "ScheduleToCloseTimeout": true}, true},
		{"other timeout does not satisfy", map[string]bool{"ScheduleToStartTimeout": true, "HeartbeatTimeout": true}, false},
		{"no fields", map[string]bool{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasRequiredTimeout(tt.fields); got != tt.want {
				t.Errorf("hasRequiredTimeout(%v) = %v, want %v", tt.fields, got, tt.want)
			}
		})
	}
}

func TestScheduleToCloseOnly(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]bool
		want   bool
	}{
		{"schedule-to-close only", map[string]bool{"ScheduleToCloseTimeout": true}, true},
		{"schedule-to-close with other field", map[string]bool{"ScheduleToCloseTimeout": true, "TaskQueue": true}, true},
		{"both timeouts set", map[string]bool{"ScheduleToCloseTimeout": true, "StartToCloseTimeout": true}, false},
		{"start-to-close only", map[string]bool{"StartToCloseTimeout": true}, false},
		{"neither timeout", map[string]bool{"TaskQueue": true}, false},
		{"no fields", map[string]bool{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scheduleToCloseOnly(tt.fields); got != tt.want {
				t.Errorf("scheduleToCloseOnly(%v) = %v, want %v", tt.fields, got, tt.want)
			}
		})
	}
}
