package temporalcheck

import (
	"testing"

	"github.com/golangci/plugin-module-register/register"
)

func TestNew(t *testing.T) {
	p, err := New(map[string]any{
		"execargs": map[string]any{
			"disabled":            false,
			"strict-types":        true,
			"strict-pointers":     true,
			"strict-struct-shape": true,
			"strict-tests":        true,
		},
		"stringtarget": map[string]any{
			"enabled":      true,
			"strict-tests": true,
		},
		"optionsdiscard": map[string]any{
			"disabled": false,
		},
		"activitytimeout": map[string]any{
			"disabled": false,
		},
		"futureget": map[string]any{
			"disabled": false,
		},
		"lossynumber": map[string]any{
			"disabled": false,
		},
		"continueasnew": map[string]any{
			"disabled": false,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	analyzers, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}
	want := []string{"execargs", "stringtarget", "optionsdiscard", "activitytimeout", "futureget", "lossynumber", "continueasnew"}
	if len(analyzers) != len(want) {
		t.Fatalf("BuildAnalyzers returned %d analyzers, want %d", len(analyzers), len(want))
	}
	for i, name := range want {
		if got := analyzers[i].Name; got != name {
			t.Errorf("analyzer[%d] name = %q, want %q", i, got, name)
		}
	}

	if got := p.GetLoadMode(); got != register.LoadModeTypesInfo {
		t.Errorf("GetLoadMode = %q, want %q", got, register.LoadModeTypesInfo)
	}
}

func TestNew_Defaults(t *testing.T) {
	// An empty settings block must still build the analyzer (strict-types and
	// strict-pointers both default to false).
	p, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := p.BuildAnalyzers(); err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}
}

func TestNew_Disabled(t *testing.T) {
	// disabled: true must still build the analyzer (the plugin stays wired); the
	// analyzer itself reports nothing.
	p, err := New(map[string]any{
		"execargs": map[string]any{"disabled": true},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	analyzers, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}
	if len(analyzers) != 7 {
		t.Fatalf("BuildAnalyzers returned %d analyzers, want 7", len(analyzers))
	}
}

func TestNew_InvalidSettings(t *testing.T) {
	// DecodeSettings JSON-decodes into Settings; a scalar cannot decode into
	// the struct, so New must surface the error rather than panic.
	if _, err := New(12345); err == nil {
		t.Fatal("expected error decoding invalid settings, got nil")
	}
}
