// Package temporalcheck wires the Temporal static checks into golangci-lint as
// a module plugin. Today it exposes a single analyzer (execargs); add more
// analyzers to BuildAnalyzers as the linter grows.
package temporalcheck

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/execargs"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/optionsdiscard"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/stringtarget"
)

func init() {
	register.Plugin("temporalcheck", New)
}

// Settings is the shape of the `settings:` block under this plugin in
// .golangci.yml. Each analyzer gets its own nested block so analyzers added
// later carry their own settings without colliding in a flat namespace.
type Settings struct {
	Execargs       ExecargsSettings       `json:"execargs"`
	StringTarget   StringTargetSettings   `json:"stringtarget"`
	OptionsDiscard OptionsDiscardSettings `json:"optionsdiscard"`
}

// ExecargsSettings configures the execargs analyzer. Pointers distinguish
// "unset" (use the default) from an explicit false.
type ExecargsSettings struct {
	// Disabled turns the analyzer off entirely (default false).
	Disabled *bool `json:"disabled"`

	// The three checks below are independent, opt-in layers on top of the
	// always-on arity check; each defaults to false when unset.

	// StrictTypes verifies argument types, not just their count. Temporal
	// serializes arguments through its DataConverter, so Go-level assignability
	// is stricter than the wire contract; this is opt-in so the always-on arity
	// check stays the false-positive-free baseline.
	StrictTypes *bool `json:"strict-types"`

	// StrictPointers flags a value passed where a pointer is expected (or vice
	// versa), including the []T vs []*T slice forms. Temporal's default
	// DataConverter serializes both identically, so these are allowed unless you
	// opt in here.
	StrictPointers *bool `json:"strict-pointers"`

	// StructShape flags passing one struct type where a different struct type is
	// expected. The DataConverter serializes by field name, so distinct structs
	// can round-trip while silently dropping or zero-filling mismatched fields;
	// this is the rarest but most dangerous case, hence its own opt-in.
	StructShape *bool `json:"strict-struct-shape"`

	// StrictTests extends the arity check to Temporal testsuite mock setups
	// (OnActivity/OnWorkflow). The matchers must cover every parameter, including
	// the injected context, so the count differs from an Execute* call by one;
	// only arity is checked since matchers are opaque. Opt-in (default false).
	StrictTests *bool `json:"strict-tests"`
}

// StringTargetSettings configures the stringtarget analyzer, which flags
// Execute* calls that name the target by its registered string instead of
// passing the function reference.
type StringTargetSettings struct {
	// Enabled turns the analyzer on for production Execute* calls (default false).
	// Naming a target by string is a legitimate pattern -- e.g. an activity
	// implemented in another service or language -- so this check is opt-in, like
	// the strict execargs layers.
	Enabled *bool `json:"enabled"`

	// StrictTests turns the check on for Temporal testsuite mock setups
	// (OnActivity/OnWorkflow named by string). Independent of Enabled, so test
	// mocks can be checked on their own. Opt-in (default false).
	StrictTests *bool `json:"strict-tests"`
}

// OptionsDiscardSettings configures the optionsdiscard analyzer, which flags
// workflow.With{Activity,LocalActivity,ChildWorkflow}Options calls whose returned
// context is discarded -- the options then silently never apply.
type OptionsDiscardSettings struct {
	// Disabled turns the analyzer off entirely (default false). The check is on by
	// default: discarding a With*Options result is always a bug, so there is
	// nothing to opt into.
	Disabled *bool `json:"disabled"`
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
	disabled := false
	if p.settings.Execargs.Disabled != nil {
		disabled = *p.settings.Execargs.Disabled
	}
	strictTypes := false
	if p.settings.Execargs.StrictTypes != nil {
		strictTypes = *p.settings.Execargs.StrictTypes
	}
	strictPointers := false
	if p.settings.Execargs.StrictPointers != nil {
		strictPointers = *p.settings.Execargs.StrictPointers
	}
	structShape := false
	if p.settings.Execargs.StructShape != nil {
		structShape = *p.settings.Execargs.StructShape
	}
	strictTests := false
	if p.settings.Execargs.StrictTests != nil {
		strictTests = *p.settings.Execargs.StrictTests
	}
	stringTargetEnabled := false
	if p.settings.StringTarget.Enabled != nil {
		stringTargetEnabled = *p.settings.StringTarget.Enabled
	}
	stringTargetStrictTests := false
	if p.settings.StringTarget.StrictTests != nil {
		stringTargetStrictTests = *p.settings.StringTarget.StrictTests
	}
	optionsDiscardDisabled := false
	if p.settings.OptionsDiscard.Disabled != nil {
		optionsDiscardDisabled = *p.settings.OptionsDiscard.Disabled
	}
	return []*analysis.Analyzer{
		execargs.NewAnalyzer(execargs.Settings{
			Disabled:       disabled,
			StrictTypes:    strictTypes,
			StrictPointers: strictPointers,
			StructShape:    structShape,
			StrictTests:    strictTests,
		}),
		stringtarget.NewAnalyzer(stringtarget.Settings{
			Enabled:     stringTargetEnabled,
			StrictTests: stringTargetStrictTests,
		}),
		optionsdiscard.NewAnalyzer(optionsdiscard.Settings{
			Disabled: optionsDiscardDisabled,
		}),
		// Future Temporal analyzers (e.g. registration coverage, retry-policy
		// sanity, non-determinism heuristics) plug in here.
	}, nil
}

func (p *plugin) GetLoadMode() string {
	// We need full type information to resolve activity/workflow signatures.
	return register.LoadModeTypesInfo
}
