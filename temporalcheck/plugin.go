// Package temporalcheck wires the Temporal static checks into golangci-lint as
// a module plugin. Today it exposes a single analyzer (execargs); add more
// analyzers to BuildAnalyzers as the linter grows.
package temporalcheck

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/execargs"
)

func init() {
	register.Plugin("temporalcheck", New)
}

// Settings is the shape of the `settings:` block under this plugin in
// .golangci.yml. Pointers distinguish "unset" (use the default) from an
// explicit false.
type Settings struct {
	// CheckTypes also verifies argument types, not just their count. Temporal
	// serializes arguments through its DataConverter, so Go-level assignability
	// is stricter than the wire contract; set this to false if the type check
	// is too noisy for your codebase. Defaults to true when unset.
	CheckTypes *bool `json:"check-types"`
}

type plugin struct {
	settings Settings
}

var _ register.LinterPlugin = (*plugin)(nil)

func New(raw any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[Settings](raw)
	if err != nil {
		return nil, err
	}
	return &plugin{settings: s}, nil
}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	checkTypes := true
	if p.settings.CheckTypes != nil {
		checkTypes = *p.settings.CheckTypes
	}
	return []*analysis.Analyzer{
		execargs.NewAnalyzer(execargs.Settings{CheckTypes: checkTypes}),
		// Future Temporal analyzers (e.g. registration coverage, retry-policy
		// sanity, non-determinism heuristics) plug in here.
	}, nil
}

func (p *plugin) GetLoadMode() string {
	// We need full type information to resolve activity/workflow signatures.
	return register.LoadModeTypesInfo
}
