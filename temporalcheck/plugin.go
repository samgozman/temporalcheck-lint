// Package temporalcheck wires the Temporal static checks into golangci-lint as a module plugin.
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
	"github.com/samgozman/temporalcheck-lint/temporalcheck/workeroptions"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/workflowlogger"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/workflowstate"
)

func init() {
	register.Plugin("temporalcheck", New)
}

// Settings is the shape of the `settings:` block under this plugin in .golangci.yml.
// Each analyzer has its own nested block to avoid naming collisions.
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
	WorkerOptions   WorkerOptionsSettings   `json:"workeroptions"`
	WorkflowState   WorkflowStateSettings   `json:"workflowstate"`
	WorkflowLogger  WorkflowLoggerSettings  `json:"workflowlogger"`
}

// ExecargsSettings configures the execargs analyzer.
// Pointers distinguish "unset" (use the default) from an explicit false.
type ExecargsSettings struct {
	Disabled       *bool `json:"disabled"`
	StrictTypes    *bool `json:"strict-types"`         // check arg types, not just count
	StrictPointers *bool `json:"strict-pointers"`       // flag T vs *T mismatches
	StructShape    *bool `json:"strict-struct-shape"`   // flag distinct struct types
	StrictTests    *bool `json:"strict-tests"`          // check OnActivity/OnWorkflow matcher arity
}

// StringTargetSettings configures the stringtarget analyzer.
type StringTargetSettings struct {
	Enabled     *bool `json:"enabled"`       // master switch (default false)
	StrictTests *bool `json:"strict-tests"`  // also check On* mock targets
}

// OptionsDiscardSettings configures the optionsdiscard analyzer.
type OptionsDiscardSettings struct {
	Disabled *bool `json:"disabled"`
}

// ActivityTimeoutSettings configures the activitytimeout analyzer.
type ActivityTimeoutSettings struct {
	Disabled            *bool `json:"disabled"`
	RequireStartToClose *bool `json:"require-start-to-close"` // also flag schedule-to-close-only literals
}

// FutureGetSettings configures the futureget analyzer.
type FutureGetSettings struct {
	Disabled *bool `json:"disabled"`
}

// LossyNumberSettings configures the lossynumber analyzer.
type LossyNumberSettings struct {
	Disabled *bool `json:"disabled"`
}

// NonSerializableSettings configures the nonserializable analyzer.
type NonSerializableSettings struct {
	Disabled    *bool `json:"disabled"`
	EmptyStruct *bool `json:"empty-struct"` // also flag structs with no exported fields
}

// ContinueAsNewSettings configures the continueasnew analyzer.
type ContinueAsNewSettings struct {
	Disabled *bool `json:"disabled"`
}

// SensitiveArgsSettings configures the sensitiveargs analyzer.
type SensitiveArgsSettings struct {
	Enabled *bool   `json:"enabled"`  // master switch (default false)
	Pattern *string `json:"pattern"`  // regexp matched against param/field names
}

// OptionsContextSettings configures the optionscontext analyzer.
type OptionsContextSettings struct {
	Disabled *bool `json:"disabled"`
}

// WorkerOptionsSettings configures the workeroptions analyzer.
type WorkerOptionsSettings struct {
	Disabled       *bool `json:"disabled"`
	RequireOptions *bool `json:"require-options"` // flag worker.New with no concurrency limits set
}

// WorkflowStateSettings configures the workflowstate analyzer.
type WorkflowStateSettings struct {
	Disabled *bool `json:"disabled"`
}

// WorkflowLoggerSettings configures the workflowlogger analyzer.
type WorkflowLoggerSettings struct {
	Enabled *bool `json:"enabled"` // master switch (default false)
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

// deref returns *p when set, or def when the pointer is nil. Settings fields are
// pointers so an unset field (use the default) is distinct from an explicit
// false/empty; this flattens each to the value the analyzer wants, with the
// default sitting next to the field it applies to.
func deref[T any](p *T, def T) T {
	if p != nil {
		return *p
	}
	return def
}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	s := p.settings
	return []*analysis.Analyzer{
		execargs.NewAnalyzer(execargs.Settings{
			Disabled:       deref(s.Execargs.Disabled, false),
			StrictTypes:    deref(s.Execargs.StrictTypes, false),
			StrictPointers: deref(s.Execargs.StrictPointers, false),
			StructShape:    deref(s.Execargs.StructShape, false),
			StrictTests:    deref(s.Execargs.StrictTests, false),
		}),
		stringtarget.NewAnalyzer(stringtarget.Settings{
			Enabled:     deref(s.StringTarget.Enabled, false),
			StrictTests: deref(s.StringTarget.StrictTests, false),
		}),
		optionsdiscard.NewAnalyzer(optionsdiscard.Settings{
			Disabled: deref(s.OptionsDiscard.Disabled, false),
		}),
		activitytimeout.NewAnalyzer(activitytimeout.Settings{
			Disabled:            deref(s.ActivityTimeout.Disabled, false),
			RequireStartToClose: deref(s.ActivityTimeout.RequireStartToClose, false),
		}),
		futureget.NewAnalyzer(futureget.Settings{
			Disabled: deref(s.FutureGet.Disabled, false),
		}),
		lossynumber.NewAnalyzer(lossynumber.Settings{
			Disabled: deref(s.LossyNumber.Disabled, false),
		}),
		nonserializable.NewAnalyzer(nonserializable.Settings{
			Disabled:    deref(s.NonSerializable.Disabled, false),
			EmptyStruct: deref(s.NonSerializable.EmptyStruct, false),
		}),
		continueasnew.NewAnalyzer(continueasnew.Settings{
			Disabled: deref(s.ContinueAsNew.Disabled, false),
		}),
		sensitiveargs.NewAnalyzer(sensitiveargs.Settings{
			Enabled: deref(s.SensitiveArgs.Enabled, false),
			Pattern: deref(s.SensitiveArgs.Pattern, ""),
		}),
		optionscontext.NewAnalyzer(optionscontext.Settings{
			Disabled: deref(s.OptionsContext.Disabled, false),
		}),
		workeroptions.NewAnalyzer(workeroptions.Settings{
			Disabled:       deref(s.WorkerOptions.Disabled, false),
			RequireOptions: deref(s.WorkerOptions.RequireOptions, false),
		}),
		workflowstate.NewAnalyzer(workflowstate.Settings{
			Disabled: deref(s.WorkflowState.Disabled, false),
		}),
		workflowlogger.NewAnalyzer(workflowlogger.Settings{
			Enabled: deref(s.WorkflowLogger.Enabled, false),
		}),
		// Future Temporal analyzers (e.g. registration coverage, retry-policy
		// sanity, non-determinism heuristics) plug in here.
	}, nil
}

func (p *plugin) GetLoadMode() string {
	// We need full type information to resolve activity/workflow signatures.
	return register.LoadModeTypesInfo
}
