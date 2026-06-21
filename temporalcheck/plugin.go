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
// .golangci.yml. Each analyzer gets its own nested block so analyzers added
// later carry their own settings without colliding in a flat namespace.
type Settings struct {
	Execargs ExecargsSettings `json:"execargs"`
}

// ExecargsSettings configures the execargs analyzer. Pointers distinguish
// "unset" (use the default) from an explicit false.
type ExecargsSettings struct {
	// StrictTypes also verifies argument types, not just their count. Temporal
	// serializes arguments through its DataConverter, so Go-level assignability
	// is stricter than the wire contract; this is opt-in so the always-on arity
	// check stays the false-positive-free baseline. Defaults to false when unset.
	StrictTypes *bool `json:"strict-types"`

	// StrictPointers makes the type check flag a value passed where a pointer is
	// expected (or vice versa), including the []T vs []*T slice forms. Temporal's
	// default DataConverter serializes both identically, so these are allowed by
	// default; set this to true to be warned about them anyway. Defaults to false
	// when unset, and only applies while StrictTypes is on.
	StrictPointers *bool `json:"strict-pointers"`
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
	strictTypes := false
	if p.settings.Execargs.StrictTypes != nil {
		strictTypes = *p.settings.Execargs.StrictTypes
	}
	strictPointers := false
	if p.settings.Execargs.StrictPointers != nil {
		strictPointers = *p.settings.Execargs.StrictPointers
	}
	return []*analysis.Analyzer{
		execargs.NewAnalyzer(execargs.Settings{StrictTypes: strictTypes, StrictPointers: strictPointers}),
		// Future Temporal analyzers (e.g. registration coverage, retry-policy
		// sanity, non-determinism heuristics) plug in here.
	}, nil
}

func (p *plugin) GetLoadMode() string {
	// We need full type information to resolve activity/workflow signatures.
	return register.LoadModeTypesInfo
}
