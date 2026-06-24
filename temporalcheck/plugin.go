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
	activityTimeoutRequireStartToClose := false
	if p.settings.ActivityTimeout.RequireStartToClose != nil {
		activityTimeoutRequireStartToClose = *p.settings.ActivityTimeout.RequireStartToClose
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
	workerOptionsDisabled := false
	if p.settings.WorkerOptions.Disabled != nil {
		workerOptionsDisabled = *p.settings.WorkerOptions.Disabled
	}
	workerOptionsRequireOptions := false
	if p.settings.WorkerOptions.RequireOptions != nil {
		workerOptionsRequireOptions = *p.settings.WorkerOptions.RequireOptions
	}
	workflowStateDisabled := false
	if p.settings.WorkflowState.Disabled != nil {
		workflowStateDisabled = *p.settings.WorkflowState.Disabled
	}
	workflowLoggerEnabled := false
	if p.settings.WorkflowLogger.Enabled != nil {
		workflowLoggerEnabled = *p.settings.WorkflowLogger.Enabled
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
			Disabled:            activityTimeoutDisabled,
			RequireStartToClose: activityTimeoutRequireStartToClose,
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
		workeroptions.NewAnalyzer(workeroptions.Settings{
			Disabled:       workerOptionsDisabled,
			RequireOptions: workerOptionsRequireOptions,
		}),
		workflowstate.NewAnalyzer(workflowstate.Settings{
			Disabled: workflowStateDisabled,
		}),
		workflowlogger.NewAnalyzer(workflowlogger.Settings{
			Enabled: workflowLoggerEnabled,
		}),
		// Future Temporal analyzers (e.g. registration coverage, retry-policy
		// sanity, non-determinism heuristics) plug in here.
	}, nil
}

func (p *plugin) GetLoadMode() string {
	// We need full type information to resolve activity/workflow signatures.
	return register.LoadModeTypesInfo
}
