// Package execargs implements a static check for the Temporal Go SDK.
//
// Temporal's workflow.ExecuteActivity / ExecuteLocalActivity /
// ExecuteChildWorkflow take the target as interface{} and its arguments as a
// variadic ...interface{}. That erases all compile-time checking: passing the
// wrong number of arguments, or arguments of the wrong type, compiles cleanly
// and only fails at run time. This analyzer resolves the referenced function's
// real signature and checks each Execute* call site against it.
package execargs

import (
	"go/ast"
	"go/types"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
	"golang.org/x/tools/go/analysis"
)

// Settings configures the execargs analyzer. The three checks below are
// independent, opt-in layers on top of the always-on arity check; enabling any
// of them turns on the per-argument type comparison.
type Settings struct {
	// Disabled turns the analyzer off entirely; it reports nothing.
	Disabled bool

	// StrictTypes verifies argument types, not just their count -- a genuine
	// mismatch such as int where string is wanted. Temporal serializes arguments
	// through its DataConverter, so Go-level assignability is stricter than the
	// wire contract; this is off by default so the always-on arity check stays the
	// false-positive-free baseline.
	StrictTypes bool

	// StrictPointers reports a value passed where a pointer is wanted (T vs *T,
	// and []T vs []*T). Temporal's default DataConverter serializes both to the
	// same wire form, so this is off by default; enable it to be warned anyway,
	// e.g. before a DataConverter change could break that equivalence.
	StrictPointers bool

	// StructShape reports passing one struct type where a different struct type is
	// wanted. The DataConverter serializes by field name, so two distinct structs
	// can round-trip whenever their fields line up -- the call works today but
	// silently drops or zero-fills any field that does not match, and breaks the
	// moment a field is renamed or starts to matter. Off by default; this is the
	// rarest but most dangerous case, so it has its own knob.
	StructShape bool

	// StrictTests extends the arity check to Temporal's testsuite mock setups:
	// (*testsuite.TestWorkflowEnvironment).OnActivity / .OnWorkflow. Those take the
	// target as interface{} and the matchers as variadic interface{}, erasing the
	// same arity the Execute* check covers. Unlike Execute*, the matchers must
	// cover EVERY declared parameter -- including the injected context -- so the
	// count differs by one. Only arity is checked: the matchers are opaque
	// (mock.Anything / mock.MatchedBy), never the real typed value, so the
	// strict-type/pointer/struct layers cannot apply. Off by default.
	StrictTests bool
}

// kind tells the checker which leading, framework-injected parameter the target
// function carries, so it knows how many parameters to skip at the call site.
type kind int

const (
	kindActivity kind = iota // leading context.Context is OPTIONAL (skip only if present)
	kindWorkflow             // leading workflow.Context is always injected (skip it)
)

// entry describes one target+args entry point: how the diagnostic names the
// target, which leading parameter the target carries (kind), and which call
// argument is the target reference.
type entry struct {
	noun      string
	kind      kind
	targetIdx int
}

// workflowEntries are the workflow.* package functions this analyzer understands.
// Each names its target as the second argument: ExecuteActivity(ctx, target,
// args...) / NewContinueAsNewError(ctx, target, args...). Supporting another is a
// single row.
var workflowEntries = map[string]entry{
	"ExecuteActivity":       {noun: "activity", kind: kindActivity, targetIdx: 1},
	"ExecuteLocalActivity":  {noun: "activity", kind: kindActivity, targetIdx: 1},
	"ExecuteChildWorkflow":  {noun: "child workflow", kind: kindWorkflow, targetIdx: 1},
	"NewContinueAsNewError": {noun: "workflow", kind: kindWorkflow, targetIdx: 1},
}

// clientEntries are the client.Client methods this analyzer understands. The
// target index differs per method: ExecuteWorkflow(ctx, options, target, args...)
// names it third; SignalWithStartWorkflow(ctx, id, signalName, signalArg, options,
// target, args...) names it sixth.
var clientEntries = map[string]entry{
	"ExecuteWorkflow":         {noun: "workflow", kind: kindWorkflow, targetIdx: 2},
	"SignalWithStartWorkflow": {noun: "workflow", kind: kindWorkflow, targetIdx: 5},
}

// entryFor reports whether fn is a target+args entry point this analyzer checks.
// workflow.* are package functions; the client methods are matched by name and
// receiver, since the SDK declares them on client.Client rather than in a package
// we can match by path.
func entryFor(fn *types.Func) (entry, bool) {
	if fn.Pkg().Path() == temporalsdk.WorkflowPkg {
		e, ok := workflowEntries[fn.Name()]
		return e, ok
	}
	if e, ok := clientEntries[fn.Name()]; ok &&
		(temporalsdk.IsReceiver(fn, temporalsdk.ClientPkg, "Client") || temporalsdk.IsReceiver(fn, temporalsdk.InternalPkg, "Client")) {
		return e, true
	}
	return entry{}, false
}

// testEnvType is the testsuite type whose mock-setup methods StrictTests checks.
// The SDK declares it in the internal package and re-publishes it from testsuite
// as an alias, so the resolved method's receiver lives in temporalsdk.InternalPkg.
const testEnvType = "TestWorkflowEnvironment"

// testEntryPoints maps the TestWorkflowEnvironment mock-setup methods to the noun
// used in their diagnostics. Both take (target, matchers...), so a missing or
// extra matcher is an arity bug the same way a misargued Execute* call is.
var testEntryPoints = map[string]string{
	"OnActivity": "activity",
	"OnWorkflow": "workflow",
}

// NewAnalyzer builds the execargs analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{
		disabled:       settings.Disabled,
		strictTypes:    settings.StrictTypes,
		strictPointers: settings.StrictPointers,
		structShape:    settings.StructShape,
		strictTests:    settings.StrictTests,
	}
	return &analysis.Analyzer{
		Name: "execargs",
		Doc:  "check that arguments to Temporal ExecuteActivity/ExecuteLocalActivity/ExecuteChildWorkflow/NewContinueAsNewError and client ExecuteWorkflow/SignalWithStartWorkflow match the target function signature",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk so the analyzer
// stays free of package-level mutable state.
type checker struct {
	disabled       bool
	strictTypes    bool
	strictPointers bool
	structShape    bool
	strictTests    bool
}

// typeChecksEnabled reports whether any of the opt-in type checks is on, i.e.
// whether the per-argument type comparison should run at all.
func (c *checker) typeChecksEnabled() bool {
	return c.strictTypes || c.strictPointers || c.structShape
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	if c.disabled {
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
	// workflow/testsuite packages still match.
	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return
	}
	if e, ok := entryFor(fn); ok {
		c.checkExecuteCall(pass, nolint, call, fn, e)
		return
	}
	if c.strictTests && fn.Pkg().Path() == temporalsdk.InternalPkg {
		c.checkTestCall(pass, nolint, call, fn)
	}
}

func (c *checker) checkExecuteCall(pass *analysis.Pass, nolint nolint.Info, call *ast.CallExpr, fn *types.Func, e entry) {
	// Honor //nolint directives ourselves so suppression works the same way it
	// does in standalone/analysistest runs, not only under golangci-lint. Checked
	// after confirming this is an entry-point call, so unrelated calls cost nothing.
	if nolint.Suppresses(pass.Fset, call) {
		return
	}

	// A spread call -- ExecuteActivity(ctx, fn, slice...) -- can't be matched
	// positionally, so leave it alone instead of emitting a false positive.
	if call.Ellipsis.IsValid() {
		return
	}

	// A compiling call always has the target present, but guard the index so a
	// malformed/partial AST can't panic.
	if len(call.Args) <= e.targetIdx {
		return
	}
	sig, ok := pass.TypesInfo.TypeOf(call.Args[e.targetIdx]).(*types.Signature)
	if !ok {
		// Target is registered by its string name, or is a value we can't
		// resolve to a signature statically. Out of scope.
		return
	}

	c.checkSignature(pass, call, fn.Name(), e, sig, call.Args[e.targetIdx+1:])
}

// checkTestCall verifies the matcher arity of a TestWorkflowEnvironment mock
// setup -- OnActivity/OnWorkflow. The matchers must cover every declared
// parameter, so there is no injected context to skip (the way Execute* does);
// the count is simply the target's parameter count.
func (c *checker) checkTestCall(pass *analysis.Pass, nolint nolint.Info, call *ast.CallExpr, fn *types.Func) {
	// Confirm the method is OnActivity/OnWorkflow on testsuite's
	// TestWorkflowEnvironment, so an unrelated internal method -- e.g. the
	// MockCallWrapper.Return/Once chained after the setup -- can't match.
	noun, ok := testEntryPoints[fn.Name()]
	if !ok || !temporalsdk.IsReceiver(fn, temporalsdk.InternalPkg, testEnvType) {
		return
	}
	if nolint.Suppresses(pass.Fset, call) {
		return
	}

	// A spread call -- OnActivity(fn, matchers...) -- can't be matched positionally.
	if call.Ellipsis.IsValid() {
		return
	}

	// Shape is (target, matchers...); a compiling On* call has at least the
	// required target argument, so call.Args[0] is safe to read.
	sig, ok := pass.TypesInfo.TypeOf(call.Args[0]).(*types.Signature)
	if !ok {
		// String-named target (stringtarget's job) or otherwise unresolvable.
		return
	}
	// A variadic target makes the matcher count unknowable statically; the mock
	// framework flattens the variadic, so skip rather than risk a false positive.
	if sig.Variadic() {
		return
	}

	want := sig.Params().Len()
	got := len(call.Args) - 1
	if got != want {
		pass.Reportf(call.Lparen, "%s: mock for %s %q expects %d %s (one per parameter), got %d (%s)",
			fn.Name(), noun, targetName(call.Args[0]), want, argWord(want), got, tagStrictTests)
	}
}
