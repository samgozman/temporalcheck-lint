package temporalcheck

import (
	"testing"

	"github.com/golangci/plugin-module-register/register"
)

func TestNew(t *testing.T) {
	p, err := New(map[string]any{
		"execargs": map[string]any{
			"strict-types":        true,
			"strict-pointers":     true,
			"strict-struct-shape": true,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	analyzers, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}
	if len(analyzers) != 1 {
		t.Fatalf("BuildAnalyzers returned %d analyzers, want 1", len(analyzers))
	}
	if got := analyzers[0].Name; got != "execargs" {
		t.Errorf("analyzer name = %q, want %q", got, "execargs")
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

func TestNew_InvalidSettings(t *testing.T) {
	// DecodeSettings JSON-decodes into Settings; a scalar cannot decode into
	// the struct, so New must surface the error rather than panic.
	if _, err := New(12345); err == nil {
		t.Fatal("expected error decoding invalid settings, got nil")
	}
}
