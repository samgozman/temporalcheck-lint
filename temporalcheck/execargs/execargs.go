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

	"golang.org/x/tools/go/analysis"
)

const (
	workflowPkg = "go.temporal.io/sdk/workflow"
	contextPkg  = "context"
)

// Settings configures the execargs analyzer.
type Settings struct {
	// CheckTypes also verifies argument types, not just their count. Temporal
	// serializes arguments through its DataConverter, so Go-level assignability
	// is stricter than the wire contract; disable this if the type half is too
	// noisy for your codebase. The arity check always runs.
	CheckTypes bool
}

// kind tells the checker which leading, framework-injected parameter the target
// function carries, so it knows how many parameters to skip at the call site.
type kind int

const (
	kindActivity      kind = iota // leading context.Context is OPTIONAL (skip only if present)
	kindChildWorkflow             // leading workflow.Context is always injected (skip it)
)

// entryPoints are the workflow.* functions this analyzer understands.
// Supporting another one is a single row.
var entryPoints = map[string]kind{
	"ExecuteActivity":      kindActivity,
	"ExecuteLocalActivity": kindActivity,
	"ExecuteChildWorkflow": kindChildWorkflow,
}

// NewAnalyzer builds the execargs analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{checkTypes: settings.CheckTypes}
	return &analysis.Analyzer{
		Name: "execargs",
		Doc:  "check that arguments to Temporal ExecuteActivity/ExecuteLocalActivity/ExecuteChildWorkflow match the target function signature",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk so the analyzer
// stays free of package-level mutable state.
type checker struct {
	checkTypes bool
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				c.checkCall(pass, call)
			}
			return true
		})
	}
	return nil, nil
}

func (c *checker) checkCall(pass *analysis.Pass, call *ast.CallExpr) {
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
	k, ok := entryPoints[fn.Name()]
	if !ok {
		return
	}

	// A spread call -- ExecuteActivity(ctx, fn, slice...) -- can't be matched
	// positionally, so leave it alone instead of emitting a false positive.
	if call.Ellipsis.IsValid() {
		return
	}

	// Shape is always (ctx, target, args...). Fewer than two arguments is a
	// compile error the compiler already reports.
	if len(call.Args) < 2 {
		return
	}

	sig, ok := pass.TypesInfo.TypeOf(call.Args[1]).(*types.Signature)
	if !ok {
		// Target is registered by its string name, or is a value we can't
		// resolve to a signature statically. Out of scope.
		return
	}

	c.checkSignature(pass, call, sel.Sel.Name, k, sig, call.Args[2:])
}
