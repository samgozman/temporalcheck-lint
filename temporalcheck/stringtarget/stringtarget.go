// Package stringtarget implements an opt-in static check for the Temporal Go SDK.
//
// Temporal lets you launch an activity or child workflow by its registered
// string name -- workflow.ExecuteActivity(ctx, "MyActivity", args...) -- instead
// of by a reference to the Go function. That string is opaque: it cannot be
// resolved to a signature at compile time, so the number and types of the
// trailing arguments go unchecked, and a typo in the name fails only at run
// time. Worse, it blinds the execargs analyzer, which silently skips any call
// whose target is not a resolvable function value.
//
// This analyzer flags those call sites so they can be refactored to pass the
// function reference instead. Doing so is better on its own terms -- the name is
// then derived from the function rather than duplicated as a fragile string --
// and it is what lets the rest of this linter verify the call's arguments.
//
// The check is off by default; enable it via settings.stringtarget.enabled.
package stringtarget

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/analysis"
)

const (
	workflowPkg = "go.temporal.io/sdk/workflow"
	// tagStringTarget suffixes the diagnostic so it is clear which check, and
	// therefore which setting, produced it -- mirroring the execargs tags.
	tagStringTarget = "string-target"
)

// entryPoints are the workflow.* functions whose target argument this analyzer
// inspects. The set matches execargs: a string target is equally unresolvable
// for any of them.
var entryPoints = map[string]bool{
	"ExecuteActivity":      true,
	"ExecuteLocalActivity": true,
	"ExecuteChildWorkflow": true,
}

// Settings configures the stringtarget analyzer.
type Settings struct {
	// Enabled turns the check on. It is off by default: naming a target by
	// string is a legitimate, sometimes necessary pattern (e.g. an activity
	// implemented in another service or language), so flagging it is opt-in.
	Enabled bool
}

// NewAnalyzer builds the stringtarget analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{enabled: settings.Enabled}
	return &analysis.Analyzer{
		Name: "stringtarget",
		Doc:  "flag Temporal ExecuteActivity/ExecuteLocalActivity/ExecuteChildWorkflow calls that name the target by its registered string instead of passing the function reference",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk so the analyzer
// stays free of package-level mutable state.
type checker struct {
	enabled bool
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	if !c.enabled {
		return nil, nil
	}
	for _, file := range pass.Files {
		nolint := collectNolint(pass.Fset, file)
		ast.Inspect(file, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				c.checkCall(pass, nolint, call)
			}
			return true
		})
	}
	return nil, nil
}

func (c *checker) checkCall(pass *analysis.Pass, nolint nolintInfo, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Resolve via Uses (not the source text), so aliased imports of the
	// workflow package still match.
	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != workflowPkg {
		return
	}
	if !entryPoints[fn.Name()] {
		return
	}

	// Honor //nolint directives ourselves so suppression works the same way in
	// standalone/analysistest runs, not only under golangci-lint. Checked after
	// confirming this is an Execute* call, so unrelated calls cost nothing.
	if nolint.suppressesCall(pass.Fset, call) {
		return
	}

	// Shape is always (ctx, target, args...); a compiling Execute* call therefore
	// has at least two arguments, so call.Args[1] is safe to read.
	target := call.Args[1]
	if !isStringType(pass.TypesInfo.TypeOf(target)) {
		// A function reference (or any non-string value) is exactly what we want
		// callers to pass; execargs takes it from here.
		return
	}

	subject := "the target"
	if name, ok := literalName(target); ok {
		subject = fmt.Sprintf("target %q", name)
	}
	pass.Reportf(target.Pos(),
		"%s: %s is named by string; pass the function reference instead so its arguments can be checked statically (%s)",
		sel.Sel.Name, subject, tagStringTarget)
}

// isStringType reports whether t is a string -- the typed string, an untyped
// string constant, or a named type whose underlying type is string (e.g. a
// `type ActivityName string` holding the registered name).
func isStringType(t types.Type) bool {
	if t == nil {
		return false
	}
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&types.IsString != 0
}

// literalName returns the unquoted value of a string-literal target, so the
// diagnostic can name it. A non-literal string (a variable or constant) has no
// literal to quote, so it falls back to a generic subject.
func literalName(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return v, true
}
