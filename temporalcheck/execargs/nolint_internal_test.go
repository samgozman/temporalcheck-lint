package execargs

import "testing"

func TestNolintForExecargs(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"bare nolint", "//nolint", true},
		{"nolint all", "//nolint:all", true},
		{"names temporalcheck", "//nolint:temporalcheck", true},
		{"temporalcheck in a list", "//nolint:gocritic,temporalcheck,godot", true},
		{"temporalcheck with explanation", "//nolint:temporalcheck // intentional", true},
		{"spaces around names", "//nolint: temporalcheck , gocritic", true},
		{"names the analyzer, not the plugin", "//nolint:execargs", false},
		{"other linters only", "//nolint:gocritic,godot", false},
		{"not a nolint", "// nolint is mentioned in prose", false},
		{"lookalike prefix", "//nolintfoo:temporalcheck", false},
		{"plain comment", "// just a comment", false},
		{"block comment", "/* nolint:temporalcheck */", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nolintForExecargs(tt.text); got != tt.want {
				t.Errorf("nolintForExecargs(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}
