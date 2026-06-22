package workeroptions

import (
	"go/ast"
	"go/constant"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// Diagnostics are suffixed with the rule that produced them, so it is clear which
// setting controls a given report.
const (
	tagWorkerPanic    = "worker-panic"
	tagRequireOptions = "require-options"
)

// The worker options struct is declared in the SDK's internal package as
// WorkerOptions and re-exported from worker as `type Options =
// internal.WorkerOptions`, mirroring workflow.Context. We match by path through
// go/types -- resolving the alias to its internal definition -- so aliased imports
// resolve and we never import the SDK. The two packages name the type differently
// (worker.Options vs internal.WorkerOptions), so each path checks its own name.
const (
	workerPkg   = "go.temporal.io/sdk/worker"
	internalPkg = "go.temporal.io/sdk/internal"
)

// panicFields are the two options fields Temporal documents as unable to hold the
// value 1: the pollers alternate between sticky and non-sticky queues, so a single
// one deadlocks the worker, which panics on start. The activity counterparts carry
// no such restriction, so they are deliberately absent here.
var panicFields = []string{
	"MaxConcurrentWorkflowTaskExecutionSize",
	"MaxConcurrentWorkflowTaskPollers",
}

// concurrencyFields are the five worker.Options knobs that bound a worker's
// resource use. require-options is satisfied by any one of them being set
// (value-irrelevant): requiring a specific field would falsely flag an
// activity-only or local-activity-only worker that correctly sets just its own
// knob, so "sets none of the five" is the unambiguous failure mode.
var concurrencyFields = []string{
	"MaxConcurrentActivityExecutionSize",
	"MaxConcurrentWorkflowTaskExecutionSize",
	"MaxConcurrentActivityTaskPollers",
	"MaxConcurrentWorkflowTaskPollers",
	"MaxConcurrentLocalActivityExecutionSize",
}

// checkPanic reports each workflow-task field a worker.Options literal sets to a
// constant 1 -- a guaranteed worker-boot panic. The diagnostic anchors on the
// offending value expression, not the literal.
func (c *checker) checkPanic(pass *analysis.Pass, nolint nolintInfo, lit *ast.CompositeLit) {
	if !isWorkerOptions(pass.TypesInfo.TypeOf(lit)) {
		return
	}

	fields := keyedFieldValues(lit)
	for _, name := range panicFields {
		val, ok := fields[name]
		if !ok || !constEqualsOne(pass, val) {
			continue
		}
		// Honor //nolint ourselves so suppression works the same way in
		// standalone/analysistest runs, not only under golangci-lint. Checked after
		// confirming this value is flagged, so unrelated values cost nothing.
		if nolint.suppressesNode(pass.Fset, val) {
			continue
		}
		pass.Reportf(val.Pos(),
			"worker.Options: %s must not be 1 — the worker panics on start; use 0 for the default or a value >= 2 (%s)",
			name, tagWorkerPanic)
	}
}

// checkRequireOptions reports a worker.New(c, q, worker.Options{...}) whose options
// literal sets none of the five concurrency limits -- the worker then runs on the
// SDK defaults. Only the literal passed directly to worker.New is inspected; a
// variable argument is skipped since its fields aren't visible at the call site.
func (c *checker) checkRequireOptions(pass *analysis.Pass, nolint nolintInfo, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	// Resolve via Uses (not the source text), so aliased imports of the worker
	// package still match.
	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != workerPkg || fn.Name() != "New" {
		return
	}

	// worker.New(client, taskQueue, options): the options literal is the third
	// argument. Guard the index so a malformed/partial AST can't panic.
	if len(call.Args) < 3 {
		return
	}
	lit, ok := call.Args[2].(*ast.CompositeLit)
	if !ok || !isWorkerOptions(pass.TypesInfo.TypeOf(lit)) {
		return
	}

	fields := keyedFieldValues(lit)
	for _, name := range concurrencyFields {
		if _, ok := fields[name]; ok {
			return // at least one limit set -> satisfied
		}
	}

	if nolint.suppressesNode(pass.Fset, lit) {
		return
	}
	pass.Reportf(lit.Pos(),
		"worker.New: worker.Options sets no concurrency limits, so the worker runs on the SDK defaults (1k executions, 100k/s) that can overload a self-hosted cluster; set MaxConcurrent* limits (%s)",
		tagRequireOptions)
}

// isWorkerOptions reports whether t is the worker.Options struct. types.Unalias
// resolves the worker alias to its internal definition, so the literal's type
// matches whether the type checker surfaces it as the alias (worker.Options) or
// the resolved named type (internal.WorkerOptions) -- the two name the type
// differently, so each package path checks its own name.
func isWorkerOptions(t types.Type) bool {
	if t == nil {
		return false
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj.Pkg() == nil {
		return false
	}
	switch obj.Pkg().Path() {
	case workerPkg:
		return obj.Name() == "Options"
	case internalPkg:
		return obj.Name() == "WorkerOptions"
	default:
		return false
	}
}

// keyedFieldValues maps each field a keyed composite literal sets to its value
// expression. A positional literal (no field names to map without the struct
// layout) and an empty literal both yield an empty map, so callers simply find no
// field of interest -- the same skip the activitytimeout analyzer makes, avoiding
// a false positive on shapes we can't resolve.
func keyedFieldValues(lit *ast.CompositeLit) map[string]ast.Expr {
	fields := make(map[string]ast.Expr, len(lit.Elts))
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue // positional element
		}
		if id, ok := kv.Key.(*ast.Ident); ok {
			fields[id.Name] = kv.Value
		}
	}
	return fields
}

// constEqualsOne reports whether expr is a constant integer equal to 1. A
// non-constant value (a variable or expression the analyzer can't resolve
// statically, e.g. cfg.Pollers) returns false, so it is skipped rather than risked
// as a false positive.
func constEqualsOne(pass *analysis.Pass, expr ast.Expr) bool {
	v := pass.TypesInfo.Types[expr].Value
	if v == nil || v.Kind() != constant.Int {
		return false
	}
	n, ok := constant.Int64Val(v)
	return ok && n == 1
}
