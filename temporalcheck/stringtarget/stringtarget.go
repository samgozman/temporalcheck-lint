// Package stringtarget flags Temporal Execute*/Signal* calls that name the target
// by its registered string instead of passing the function reference. String
// targets can't be checked statically and blind the execargs analyzer. The check
// is opt-in (off by default).
package stringtarget

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
	"golang.org/x/tools/go/analysis"
)

const (
	testEnvType = "TestWorkflowEnvironment"
	// tagStringTarget suffixes the production-call diagnostic so it is clear which
	// check produced it.
	tagStringTarget = "string-target"
	// tagStrictTests suffixes the test-mock diagnostic so it is distinct from the
	// production string-target one and names the setting that surfaced it.
	tagStrictTests = "strict-tests"
)

// workflowEntries are the workflow.* functions whose target argument this
// analyzer inspects, mapped to that argument's index. A string target is equally
// unresolvable for any of them. The set matches execargs.
var workflowEntries = map[string]int{
	"ExecuteActivity":       1,
	"ExecuteLocalActivity":  1,
	"ExecuteChildWorkflow":  1,
	"NewContinueAsNewError": 1,
}

// clientEntries are the client.Client methods whose workflow target this analyzer
// inspects, mapped to that target's argument index. ExecuteWorkflow(ctx, options,
// target, args...) names it third; SignalWithStartWorkflow(ctx, id, signalName,
// signalArg, options, target, args...) names it sixth.
var clientEntries = map[string]int{
	"ExecuteWorkflow":         2,
	"SignalWithStartWorkflow": 5,
}

// testEntryPoints are the TestWorkflowEnvironment mock-setup methods whose target
// argument StrictTests inspects. A string target is just as unresolvable in a
// mock as in a production call.
var testEntryPoints = map[string]bool{
	"OnActivity": true,
	"OnWorkflow": true,
}

// Settings configures the stringtarget analyzer.
type Settings struct {
	Enabled     bool // master switch (default false)
	StrictTests bool // also check On* mock targets (requires Enabled)
}

// NewAnalyzer builds the stringtarget analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{enabled: settings.Enabled, strictTests: settings.StrictTests}
	return &analysis.Analyzer{
		Name: "stringtarget",
		Doc:  "flag Temporal ExecuteActivity/ExecuteLocalActivity/ExecuteChildWorkflow/NewContinueAsNewError and client ExecuteWorkflow/SignalWithStartWorkflow calls that name the target by its registered string instead of passing the function reference",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk.
type checker struct {
	enabled     bool
	strictTests bool
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
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

	// Resolve via Uses so aliased imports still match.
	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return
	}
	if idx, ok := targetIndex(fn); ok {
		if idx < len(call.Args) {
			c.report(pass, nolint, call, fn.Name(), call.Args[idx], tagStringTarget)
		}
		return
	}
	// Shape is (target, matchers...): the test-mock target is the first argument.
	if c.strictTests && testEntryPoints[fn.Name()] && temporalsdk.IsReceiver(fn, temporalsdk.InternalPkg, testEnvType) {
		c.report(pass, nolint, call, fn.Name(), call.Args[0], tagStrictTests)
	}
}

// targetIndex reports the argument index of the workflow/activity target for fn,
// if fn is an entry point this analyzer inspects.
func targetIndex(fn *types.Func) (int, bool) {
	if fn.Pkg().Path() == temporalsdk.WorkflowPkg {
		idx, ok := workflowEntries[fn.Name()]
		return idx, ok
	}
	if idx, ok := clientEntries[fn.Name()]; ok &&
		(temporalsdk.IsReceiver(fn, temporalsdk.ClientPkg, "Client") || temporalsdk.IsReceiver(fn, temporalsdk.InternalPkg, "Client")) {
		return idx, true
	}
	return 0, false
}

// report flags target when it is named by string, after honoring //nolint. The
// tag distinguishes a production Execute* call from a testsuite mock setup.
func (c *checker) report(pass *analysis.Pass, nolint nolint.Info, call *ast.CallExpr, entry string, target ast.Expr, tag string) {
	// Honor //nolint after confirming this is a call we inspect.
	if nolint.Suppresses(pass.Fset, call) {
		return
	}
	if !isStringType(pass.TypesInfo.TypeOf(target)) {
		return
	}

	subject := "the target"
	if name, ok := literalName(target); ok {
		subject = fmt.Sprintf("target %q", name)
	}
	pass.Reportf(target.Pos(),
		"%s: %s is named by string; pass the function reference instead so its arguments can be checked statically (%s)",
		entry, subject, tag)
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
