// Package execargs checks that arguments to Temporal Execute*/Signal* calls
// match the target function's real signature (arity, types, struct shape).
package execargs

import (
	"go/ast"
	"go/types"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/temporalsdk"
	"golang.org/x/tools/go/analysis"
)

// Settings configures the execargs analyzer.
type Settings struct {
	Disabled       bool
	StrictTypes    bool // also check arg types, not just count
	StrictPointers bool // flag T vs *T mismatches (DataConverter treats them as equivalent by default)
	StructShape    bool // flag distinct struct types (serializes by field name — drops/zeroes mismatches)
	StrictTests    bool // also check OnActivity/OnWorkflow mock matcher arity
}

// kind tells the checker which leading parameter Temporal injects into the target.
type kind int

const (
	kindActivity kind = iota // leading context.Context is OPTIONAL (skip only if present)
	kindWorkflow             // leading workflow.Context is always injected (skip it)
)

// entry describes one entry point: the diagnostic noun, the kind of injected
// context, and which call argument is the target reference.
type entry struct {
	noun      string
	kind      kind
	targetIdx int
}

// workflowEntries are the workflow.* package functions this analyzer checks.
var workflowEntries = map[string]entry{
	"ExecuteActivity":       {noun: "activity", kind: kindActivity, targetIdx: 1},
	"ExecuteLocalActivity":  {noun: "activity", kind: kindActivity, targetIdx: 1},
	"ExecuteChildWorkflow":  {noun: "child workflow", kind: kindWorkflow, targetIdx: 1},
	"NewContinueAsNewError": {noun: "workflow", kind: kindWorkflow, targetIdx: 1},
}

// clientEntries are the client.Client methods this analyzer checks.
// Target index differs: ExecuteWorkflow names it third; SignalWithStartWorkflow names it sixth.
var clientEntries = map[string]entry{
	"ExecuteWorkflow":         {noun: "workflow", kind: kindWorkflow, targetIdx: 2},
	"SignalWithStartWorkflow": {noun: "workflow", kind: kindWorkflow, targetIdx: 5},
}

// entryFor reports whether fn is an entry point this analyzer checks.
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

// testEnvType is the SDK type whose OnActivity/OnWorkflow methods StrictTests checks.
// The SDK declares it in internal and re-publishes it as an alias, so the resolved
// method's receiver lives in temporalsdk.InternalPkg.
const testEnvType = "TestWorkflowEnvironment"

// testEntryPoints maps mock-setup method names to the noun used in diagnostics.
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

// checker threads the analyzer settings through the AST walk.
type checker struct {
	disabled       bool
	strictTypes    bool
	strictPointers bool
	structShape    bool
	strictTests    bool
}

// typeChecksEnabled reports whether any opt-in type check is on.
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

	// Resolve via Uses (not source text) so aliased imports still match.
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
	// Honor //nolint after confirming this is an entry-point call.
	if nolint.Suppresses(pass.Fset, call) {
		return
	}

	// A spread call can't be matched positionally.
	if call.Ellipsis.IsValid() {
		return
	}

	if len(call.Args) <= e.targetIdx {
		return
	}
	sig, ok := pass.TypesInfo.TypeOf(call.Args[e.targetIdx]).(*types.Signature)
	if !ok {
		// String-named target or otherwise unresolvable — out of scope.
		return
	}

	c.checkSignature(pass, call, fn.Name(), e, sig, call.Args[e.targetIdx+1:])
}

// checkTestCall verifies the matcher arity of OnActivity/OnWorkflow mock setups.
// Unlike Execute*, matchers must cover ALL parameters including the injected context.
func (c *checker) checkTestCall(pass *analysis.Pass, nolint nolint.Info, call *ast.CallExpr, fn *types.Func) {
	// Confirm the method is OnActivity/OnWorkflow on TestWorkflowEnvironment,
	// not an unrelated internal method (e.g. MockCallWrapper.Return/Once).
	noun, ok := testEntryPoints[fn.Name()]
	if !ok || !temporalsdk.IsReceiver(fn, temporalsdk.InternalPkg, testEnvType) {
		return
	}
	if nolint.Suppresses(pass.Fset, call) {
		return
	}

	// A spread call can't be matched positionally.
	if call.Ellipsis.IsValid() {
		return
	}

	// Shape is (target, matchers...); a compiling On* call has at least the target.
	sig, ok := pass.TypesInfo.TypeOf(call.Args[0]).(*types.Signature)
	if !ok {
		return // string-named target or unresolvable
	}
	// A variadic target makes matcher count unknowable; skip rather than risk a false positive.
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
