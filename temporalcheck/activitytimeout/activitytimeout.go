// Package activitytimeout flags ActivityOptions/LocalActivityOptions literals
// that set fields but no required timeout (StartToCloseTimeout or ScheduleToCloseTimeout),
// causing the activity to be rejected at run time.
package activitytimeout

import (
	"go/ast"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"golang.org/x/tools/go/analysis"
)

const (
	tagRequiredTimeout     = "required-timeout"
	tagRequireStartToClose = "require-start-to-close"
)

// Settings configures the activitytimeout analyzer.
type Settings struct {
	Disabled            bool
	RequireStartToClose bool // also flag ScheduleToCloseTimeout-only literals (no StartToCloseTimeout)
}

// NewAnalyzer builds the activitytimeout analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{disabled: settings.Disabled, requireStartToClose: settings.RequireStartToClose}
	return &analysis.Analyzer{
		Name: "activitytimeout",
		Doc:  "flag Temporal workflow.ActivityOptions/LocalActivityOptions composite literals that set no required timeout (StartToCloseTimeout or ScheduleToCloseTimeout), which the activity is rejected for at run time",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk.
type checker struct {
	disabled            bool
	requireStartToClose bool
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	if c.disabled {
		return nil, nil
	}
	for _, file := range pass.Files {
		nolint := nolint.Collect(pass.Fset, file)
		ast.Inspect(file, func(n ast.Node) bool {
			if lit, ok := n.(*ast.CompositeLit); ok {
				c.checkLiteral(pass, nolint, lit)
			}
			return true
		})
	}
	return nil, nil
}

// checkLiteral reports an ActivityOptions/LocalActivityOptions literal with no required timeout.
func (c *checker) checkLiteral(pass *analysis.Pass, nolint nolint.Info, lit *ast.CompositeLit) {
	// Resolve via the type system so aliased imports match.
	name, ok := optionTypeName(pass.TypesInfo.TypeOf(lit))
	if !ok {
		return
	}

	// Empty and positional literals are skipped (see keyedFields).
	fields, ok := keyedFields(lit)
	if !ok {
		return
	}

	// The two reports are mutually exclusive, so one //nolint suppresses whichever fires.
	if !hasRequiredTimeout(fields) {
		if nolint.Suppresses(pass.Fset, lit) {
			return
		}
		pass.Reportf(lit.Pos(),
			"%s sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time (%s)",
			name, tagRequiredTimeout)
		return
	}

	if c.requireStartToClose && scheduleToCloseOnly(fields) {
		if nolint.Suppresses(pass.Fset, lit) {
			return
		}
		pass.Reportf(lit.Pos(),
			"%s sets ScheduleToCloseTimeout but not StartToCloseTimeout: bound each attempt with StartToCloseTimeout (%s)",
			name, tagRequireStartToClose)
	}
}
