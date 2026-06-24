// Package sensitiveargs flags Temporal activity/workflow parameters (and the
// exported fields of struct parameters) whose name matches a configurable regexp.
// Temporal records arguments in durable workflow history, so a secret passed
// directly becomes persistent. The check is opt-in because name matching is a
// heuristic and can produce false positives.
package sensitiveargs

import (
	"go/ast"
	"go/types"
	"regexp"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"golang.org/x/tools/go/analysis"
)

// defaultPattern is the built-in heuristic, used when Pattern is empty. The
// leading (?i) makes the whole alternation case insensitive, so it matches
// CardNumber and CVV as well as their lower-case forms. `card.?number` allows an
// optional separator (cardNumber, card_number, "card number").
const defaultPattern = `(?i)cvv|pan|card.?number|password|secret|ssn|token`

// Settings configures the sensitiveargs analyzer.
type Settings struct {
	Enabled bool   // master switch (default false)
	Pattern string // regexp matched against param/field names; empty uses defaultPattern
}

// NewAnalyzer builds the sensitiveargs analyzer for the given settings. An
// invalid Pattern surfaces as an analysis error from Run rather than silently
// disabling the check.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	pattern := settings.Pattern
	if pattern == "" {
		pattern = defaultPattern
	}
	re, err := regexp.Compile(pattern)
	c := &checker{enabled: settings.Enabled, re: re, compileErr: err}
	return &analysis.Analyzer{
		Name: "sensitiveargs",
		Doc:  "flag Temporal activity/workflow parameters (and struct fields) whose name matches a sensitive-data pattern, since Temporal records arguments in durable workflow history",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk.
type checker struct {
	enabled    bool
	re         *regexp.Regexp
	compileErr error
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	if c.compileErr != nil {
		return nil, c.compileErr
	}
	if !c.enabled {
		return nil, nil
	}
	for _, file := range pass.Files {
		nolint := nolint.Collect(pass.Fset, file)
		ast.Inspect(file, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				c.checkCall(pass, nolint, call)
			}
			return true
		})
	}
	return nil, nil
}

func (c *checker) checkCall(pass *analysis.Pass, nolint nolint.Info, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Resolve via Uses (not the source text), so aliased imports of the
	// workflow/client packages still match.
	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return
	}

	e, ok := entryFor(fn)
	if !ok {
		return
	}
	c.checkTarget(pass, nolint, call, e)
}
