// Package temporalcheck wires the Temporal static checks into golangci-lint as
// a module plugin. Today it exposes a single analyzer (execargs); add more
// analyzers to BuildAnalyzers as the linter grows.
package temporalcheck

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/activitytimeout"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/continueasnew"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/execargs"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/futureget"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/lossynumber"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/nonserializable"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/optionscontext"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/optionsdiscard"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/sensitiveargs"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/stringtarget"
)

func init() {
	register.Plugin("temporalcheck", New)
}

// Settings is the shape of the `settings:` block under this plugin in
// .golangci.yml. Each analyzer gets its own nested block so analyzers added
// later carry their own settings without colliding in a flat namespace.
type Settings struct {
	Execargs        ExecargsSettings        `json:"execargs"`
	StringTarget    StringTargetSettings    `json:"stringtarget"`
	OptionsDiscard  OptionsDiscardSettings  `json:"optionsdiscard"`
	ActivityTimeout ActivityTimeoutSettings `json:"activitytimeout"`
	FutureGet       FutureGetSettings       `json:"futureget"`
	LossyNumber     LossyNumberSettings     `json:"lossynumber"`
	NonSerializable NonSerializableSettings `json:"nonserializable"`
	ContinueAsNew   ContinueAsNewSettings   `json:"continueasnew"`
	SensitiveArgs   SensitiveArgsSettings   `json:"sensitiveargs"`
	OptionsContext  OptionsContextSettings  `json:"optionscontext"`
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
	// Enabled is the master switch: it turns the analyzer on for production
	// Execute* calls (default false). Naming a target by string is a legitimate
	// pattern -- e.g. an activity implemented in another service or language -- so
	// this check is opt-in, like the strict execargs layers. With Enabled off the
	// analyzer reports nothing, regardless of StrictTests.
	Enabled *bool `json:"enabled"`

	// StrictTests extends the check to Temporal testsuite mock setups
	// (OnActivity/OnWorkflow named by string). It is an opt-in layer on top of the
	// production check, gated by Enabled. Opt-in (default false).
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

// ActivityTimeoutSettings configures the activitytimeout analyzer, which flags
// workflow.ActivityOptions/LocalActivityOptions composite literals that set no
// required timeout (StartToCloseTimeout or ScheduleToCloseTimeout).
type ActivityTimeoutSettings struct {
	// Disabled turns the analyzer off entirely (default false). The check is on by
	// default: an activity with neither required timeout is rejected at run time,
	// so there is nothing to opt into.
	Disabled *bool `json:"disabled"`
}

// FutureGetSettings configures the futureget analyzer, which flags a
// workflow.Future/ChildWorkflowFuture/converter.EncodedValue .Get call whose
// returned error is discarded (a bare statement or `_ =`).
type FutureGetSettings struct {
	// Disabled turns the analyzer off entirely (default false). The check is on by
	// default: discarding a .Get error swallows an activity, child-workflow or
	// decode failure, which is always a bug, so there is nothing to opt into.
	Disabled *bool `json:"disabled"`
}

// LossyNumberSettings configures the lossynumber analyzer, which flags
// interface{}/any, map[string]any and []any as activity/workflow parameter or
// return types -- where Temporal's JSON converter decodes numbers as float64 and
// silently loses int64 precision past 2^53.
type LossyNumberSettings struct {
	// Disabled turns the analyzer off entirely (default false). The check is on by
	// default: a dynamically-typed number silently corrupts past 2^53, a latent
	// data-loss bug, so there is nothing to opt into. Disable it only for the rare
	// case of a custom DataConverter that preserves integer precision.
	Disabled *bool `json:"disabled"`
}

// NonSerializableSettings configures the nonserializable analyzer, which flags
// chan and func types (and, opt-in, structs with no exported fields) as
// activity/workflow parameter or return types -- types Temporal's DataConverter
// cannot serialize.
type NonSerializableSettings struct {
	// Disabled turns the analyzer off entirely (default false). The chan/func check
	// is on by default: those types can never be serialized, so there is nothing to
	// opt into. Disable it only for the rare case of a custom DataConverter that can
	// encode them.
	Disabled *bool `json:"disabled"`

	// EmptyStruct opts into also flagging a struct with fields but no exported ones
	// (and not implementing json.Marshaler), which JSON encodes to "{}", silently
	// dropping its data. Off by default: the json.Marshaler exclusion makes it less
	// clear-cut than the always-on chan/func check.
	EmptyStruct *bool `json:"empty-struct"`
}

// ContinueAsNewSettings configures the continueasnew analyzer, which flags a
// workflow.NewContinueAsNewError result that is discarded (a bare statement or
// `_ =`) rather than returned, so the workflow silently ends instead of
// continuing as new.
type ContinueAsNewSettings struct {
	// Disabled turns the analyzer off entirely (default false). The check is on by
	// default: discarding a continue-as-new error aborts the continue-as-new and
	// ends the workflow instead, which is always a bug, so there is nothing to opt
	// into.
	Disabled *bool `json:"disabled"`
}

// SensitiveArgsSettings configures the sensitiveargs analyzer, which flags
// activity/workflow parameters (and the exported fields of struct parameters)
// whose name matches a sensitive-data pattern -- since Temporal records arguments
// in durable workflow history, a useful first line of defence against leaking
// secrets or PII into that history.
type SensitiveArgsSettings struct {
	// Enabled is the master switch (default false). The check is a name-heuristic
	// that can produce false positives, so it is opt-in like the stringtarget
	// analyzer; with Enabled off the analyzer reports nothing.
	Enabled *bool `json:"enabled"`

	// Pattern is the regular expression matched (unanchored) against parameter and
	// struct-field names. Empty uses the built-in default
	// (cvv|pan|card.?number|password|secret|ssn|token, case insensitive).
	Pattern *string `json:"pattern"`
}

// OptionsContextSettings configures the optionscontext analyzer, which flags a
// workflow.Execute{Activity,LocalActivity,ChildWorkflow} call whose context was
// configured with a conflicting With*Options helper in the same function, so the
// options it reads never apply.
type OptionsContextSettings struct {
	// Disabled turns the analyzer off entirely (default false). The check is on by
	// default: it fires only on a seen options/call-kind contradiction, never on
	// absence, so there is nothing to opt into.
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
	activityTimeoutDisabled := false
	if p.settings.ActivityTimeout.Disabled != nil {
		activityTimeoutDisabled = *p.settings.ActivityTimeout.Disabled
	}
	futureGetDisabled := false
	if p.settings.FutureGet.Disabled != nil {
		futureGetDisabled = *p.settings.FutureGet.Disabled
	}
	lossyNumberDisabled := false
	if p.settings.LossyNumber.Disabled != nil {
		lossyNumberDisabled = *p.settings.LossyNumber.Disabled
	}
	nonSerializableDisabled := false
	if p.settings.NonSerializable.Disabled != nil {
		nonSerializableDisabled = *p.settings.NonSerializable.Disabled
	}
	nonSerializableEmptyStruct := false
	if p.settings.NonSerializable.EmptyStruct != nil {
		nonSerializableEmptyStruct = *p.settings.NonSerializable.EmptyStruct
	}
	continueAsNewDisabled := false
	if p.settings.ContinueAsNew.Disabled != nil {
		continueAsNewDisabled = *p.settings.ContinueAsNew.Disabled
	}
	sensitiveArgsEnabled := false
	if p.settings.SensitiveArgs.Enabled != nil {
		sensitiveArgsEnabled = *p.settings.SensitiveArgs.Enabled
	}
	sensitiveArgsPattern := ""
	if p.settings.SensitiveArgs.Pattern != nil {
		sensitiveArgsPattern = *p.settings.SensitiveArgs.Pattern
	}
	optionsContextDisabled := false
	if p.settings.OptionsContext.Disabled != nil {
		optionsContextDisabled = *p.settings.OptionsContext.Disabled
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
		activitytimeout.NewAnalyzer(activitytimeout.Settings{
			Disabled: activityTimeoutDisabled,
		}),
		futureget.NewAnalyzer(futureget.Settings{
			Disabled: futureGetDisabled,
		}),
		lossynumber.NewAnalyzer(lossynumber.Settings{
			Disabled: lossyNumberDisabled,
		}),
		nonserializable.NewAnalyzer(nonserializable.Settings{
			Disabled:    nonSerializableDisabled,
			EmptyStruct: nonSerializableEmptyStruct,
		}),
		continueasnew.NewAnalyzer(continueasnew.Settings{
			Disabled: continueAsNewDisabled,
		}),
		sensitiveargs.NewAnalyzer(sensitiveargs.Settings{
			Enabled: sensitiveArgsEnabled,
			Pattern: sensitiveArgsPattern,
		}),
		optionscontext.NewAnalyzer(optionscontext.Settings{
			Disabled: optionsContextDisabled,
		}),
		// Future Temporal analyzers (e.g. registration coverage, retry-policy
		// sanity, non-determinism heuristics) plug in here.
	}, nil
}

func (p *plugin) GetLoadMode() string {
	// We need full type information to resolve activity/workflow signatures.
	return register.LoadModeTypesInfo
}
