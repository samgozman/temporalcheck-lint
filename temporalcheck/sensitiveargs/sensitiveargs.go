// Package sensitiveargs implements an opt-in static check for the Temporal Go
// SDK.
//
// Temporal records every activity and workflow argument in its durable event
// history, where it is persisted and replayed for the life of the workflow. A
// secret passed directly as an argument -- a card number, CVV, password, token --
// is therefore written verbatim into that history. The fix is to pass an opaque
// reference (an id, a handle) and fetch the secret inside the activity instead.
//
// This analyzer flags activity/workflow parameters, and the fields of
// struct-typed parameters, whose *name* matches a configurable regular
// expression (default: cvv|pan|card.?number|password|secret|ssn|token, case
// insensitive). Matching on names is a heuristic -- it can flag a benign
// `passwordPolicy` or miss an obfuscated field -- so the whole analyzer is opt-in
// and the pattern is tunable. For teams that must keep PII and secrets out of
// durable history it is a useful first line of defence.
//
// Like the sibling type-predicate analyzers it resolves the target's real
// signature through go/types and looks only at the top level -- a parameter name,
// or the fields of a struct (or pointer-to-struct) parameter -- to stay
// predictable; it does not descend into nested structs, slices or maps.
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
	// Enabled is the master switch (default false). Because the check is a
	// name-heuristic that can produce false positives, it reports nothing unless
	// you opt in -- the same shape as the stringtarget analyzer.
	Enabled bool

	// Pattern is the regular expression matched (unanchored, so a substring match)
	// against parameter and struct-field names. Empty means use defaultPattern.
	Pattern string
}

// NewAnalyzer builds the sensitiveargs analyzer for the given settings. An
// invalid Pattern is reported from Run (NewAnalyzer cannot return an error), so a
// misconfigured regexp surfaces as a clear analysis error rather than silently
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

// checker threads the analyzer settings through the AST walk so the analyzer
// stays free of package-level mutable state.
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
