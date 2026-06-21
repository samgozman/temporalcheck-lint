// Package activitytimeout implements a static check for the Temporal Go SDK.
//
// Activity options carry the timeouts that bound an activity execution. Temporal
// requires at least one of two of them -- StartToCloseTimeout or
// ScheduleToCloseTimeout -- on every activity; without either, the activity is
// rejected at run time:
//
//	ao := workflow.ActivityOptions{TaskQueue: "greetings"} // no timeout set
//	ctx = workflow.WithActivityOptions(ctx, ao)
//	workflow.ExecuteActivity(ctx, a.Greet)                 // fails at run time
//
// The Go idiom is a workflow.ActivityOptions{...} (or LocalActivityOptions)
// composite literal fed to WithActivityOptions before ExecuteActivity, so the
// mistake is visible in the literal itself. This analyzer inspects those literals
// and flags any that set fields but neither required timeout, so the bug is caught
// at lint time. It is errcheck-style: pure AST + types, near-zero false positives,
// so it is on by default.
package activitytimeout

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// tagRequiredTimeout suffixes the diagnostic so it is clear which check
// produced it.
const tagRequiredTimeout = "required-timeout"

// Settings configures the activitytimeout analyzer.
type Settings struct {
	// Disabled turns the analyzer off entirely; it reports nothing. The check is
	// on by default: an activity with neither required timeout is always rejected
	// at run time, never a deliberate pattern, so there is nothing to opt into.
	Disabled bool
}

// NewAnalyzer builds the activitytimeout analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{disabled: settings.Disabled}
	return &analysis.Analyzer{
		Name: "activitytimeout",
		Doc:  "flag Temporal workflow.ActivityOptions/LocalActivityOptions composite literals that set no required timeout (StartToCloseTimeout or ScheduleToCloseTimeout), which the activity is rejected for at run time",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk so the analyzer
// stays free of package-level mutable state.
type checker struct {
	disabled bool
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	if c.disabled {
		return nil, nil
	}
	for _, file := range pass.Files {
		nolint := collectNolint(pass.Fset, file)
		ast.Inspect(file, func(n ast.Node) bool {
			// Composite literals are visited wherever they appear -- including the
			// inner literal of &workflow.ActivityOptions{...} and the elided element
			// literals of a []workflow.ActivityOptions{{...}} -- so each is checked.
			if lit, ok := n.(*ast.CompositeLit); ok {
				c.checkLiteral(pass, nolint, lit)
			}
			return true
		})
	}
	return nil, nil
}

// checkLiteral reports lit when it is an ActivityOptions/LocalActivityOptions
// literal that sets fields but no required timeout, after honoring //nolint.
func (c *checker) checkLiteral(pass *analysis.Pass, nolint nolintInfo, lit *ast.CompositeLit) {
	// Resolve via the type system (not the source text), so aliased imports of the
	// workflow package still match.
	name, ok := optionTypeName(pass.TypesInfo.TypeOf(lit))
	if !ok {
		return
	}

	// Empty and positional literals are deliberately skipped (see keyedFields):
	// an empty literal is typically populated field-by-field afterwards, and a
	// positional one can't be mapped to field names without the struct layout.
	fields, ok := keyedFields(lit)
	if !ok {
		return
	}

	if hasRequiredTimeout(fields) {
		return
	}

	// Honor //nolint ourselves so suppression works the same way in
	// standalone/analysistest runs, not only under golangci-lint. Checked after
	// confirming this is a literal we flag, so unrelated literals cost nothing.
	if nolint.suppressesNode(pass.Fset, lit) {
		return
	}

	pass.Reportf(lit.Pos(),
		"%s sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time (%s)",
		name, tagRequiredTimeout)
}
