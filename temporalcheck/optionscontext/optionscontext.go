// Package optionscontext implements a static check for the Temporal Go SDK.
//
// Activities, local activities and child workflows each read their options from
// the workflow context under a DIFFERENT key. workflow.WithActivityOptions,
// WithLocalActivityOptions and WithChildOptions return a context carrying the
// corresponding options; ExecuteActivity, ExecuteLocalActivity and
// ExecuteChildWorkflow each read the matching one back out. Crossing the wires
// compiles cleanly but blows up at run time:
//
//	ctx = workflow.WithChildOptions(ctx, cwo)
//	workflow.ExecuteActivity(ctx, a.Greet) // child options, activity call -- boom
//
// Asking "does this ctx have the RIGHT options?" needs full dataflow and
// cross-function visibility, which is where false positives come from. This
// analyzer asks the narrower, decidable question instead: "does this ctx carry a
// CONFLICTING options helper, applied in this same function, with no matching one
// in sight?" It only ever fires on a seen contradiction, never on absence, so it
// stays near-zero-false-positive and is on by default.
//
// The analysis is intra-procedural and AST+types only. It tracks, per context
// variable, the set of option kinds applied to it through its visible derivation
// chain, and bails to "unknown" (reports nothing) the moment it loses sight of
// the full story: a bare function parameter, an opaque reassignment, a capture
// in a closure, or a branch-dependent value.
package optionscontext

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

const (
	workflowPkg = "go.temporal.io/sdk/workflow"
	// tagOptionsContext suffixes the diagnostic so it is clear which check
	// produced it.
	tagOptionsContext = "options-context"
)

// withFuncs maps each workflow.With*Options helper to the option kind it applies
// to the context it returns.
var withFuncs = map[string]kind{
	"WithActivityOptions":      kindActivity,
	"WithLocalActivityOptions": kindLocalActivity,
	// WithChildOptions is the public name the SDK exports for the child-workflow
	// options setter (the underlying internal function is WithChildWorkflowOptions).
	"WithChildOptions": kindChild,
}

// executeFuncs maps each workflow.Execute* function to the option kind it reads
// back out of the context it receives.
var executeFuncs = map[string]kind{
	"ExecuteActivity":      kindActivity,
	"ExecuteLocalActivity": kindLocalActivity,
	"ExecuteChildWorkflow": kindChild,
}

// Settings configures the optionscontext analyzer.
type Settings struct {
	// Disabled turns the analyzer off entirely; it reports nothing. The check is
	// on by default: it fires only on a seen options/call-kind contradiction, never
	// on absence, so there is nothing to opt into.
	Disabled bool
}

// NewAnalyzer builds the optionscontext analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{disabled: settings.Disabled}
	return &analysis.Analyzer{
		Name: "optionscontext",
		Doc:  "flag a Temporal workflow.Execute{Activity,LocalActivity,ChildWorkflow} call whose context carries a conflicting With*Options helper applied in the same function, so the options it reads never apply",
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
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			c.analyzeFunc(pass, nolint, fn)
		}
	}
	return nil, nil
}

// analyzeFunc runs the per-function flow analysis: identify context variables
// that are unsafe to track at all (captured in a closure), then interpret the
// body in source order, firing on any seen conflict.
func (c *checker) analyzeFunc(pass *analysis.Pass, nolint nolintInfo, fn *ast.FuncDecl) {
	wc := &walkCtx{
		pass:     pass,
		nolint:   nolint,
		poisoned: collectPoisoned(pass, fn.Body),
	}
	wc.walkStmts(fn.Body.List, &state{applied: map[*types.Var]ctxInfo{}})
}

// walkCtx carries the per-function context for the walk. state (the mutable flow
// facts) is threaded separately so it can be cloned at control-flow boundaries.
type walkCtx struct {
	pass   *analysis.Pass
	nolint nolintInfo
	// poisoned vars are never tracked and never fired on -- a context variable
	// assigned inside a closure could be reconfigured at an unknown time, so we
	// cannot reason about its value.
	poisoned map[*types.Var]bool
}

// state holds the flow facts at a point in the walk: for each tracked context
// variable, the option kinds applied to its current value. Absence means
// "unknown" -- we never fire on it.
type state struct {
	applied map[*types.Var]ctxInfo
}

// ctxInfo records what is known about a context variable's current value.
type ctxInfo struct {
	set  kindSet // option kinds applied along the visible chain
	last kind    // the most recently applied kind, named in the diagnostic
}

func (s *state) clone() *state {
	out := &state{applied: make(map[*types.Var]ctxInfo, len(s.applied))}
	for k, v := range s.applied {
		out.applied[k] = v
	}
	return out
}

// walkStmts interprets a straight-line statement list in source order.
func (wc *walkCtx) walkStmts(stmts []ast.Stmt, st *state) {
	for _, stmt := range stmts {
		wc.walkStmt(stmt, st)
	}
}

// walkStmt interprets one statement, updating st and reporting conflicts.
func (wc *walkCtx) walkStmt(stmt ast.Stmt, st *state) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		wc.handleAssign(s, st)
	case *ast.BlockStmt:
		// A bare nested block is sequential with its surroundings; no join to merge.
		wc.walkStmts(s.List, st)
	case *ast.LabeledStmt:
		wc.walkStmt(s.Stmt, st)
	case *ast.IfStmt:
		wc.walkIf(s, st)
	case *ast.ForStmt:
		wc.walkFor(s, st)
	case *ast.RangeStmt:
		wc.walkRange(s, st)
	case *ast.SwitchStmt:
		wc.walkSwitch(s, st)
	case *ast.TypeSwitchStmt:
		wc.walkTypeSwitch(s, st)
	case *ast.SelectStmt:
		wc.walkSelect(s, st)
	default:
		// Any other statement (ExprStmt, ReturnStmt, GoStmt, DeferStmt, ...) cannot
		// branch and declares no tracked reassignment; just scan it for Execute*
		// calls to check against the current state.
		wc.scanExecute(stmt, st)
	}
}

// handleAssign updates the flow facts for an assignment and checks any Execute*
// call on its right-hand side.
func (wc *walkCtx) handleAssign(s *ast.AssignStmt, st *state) {
	// An Execute* call can be the RHS of an assignment (future := ExecuteActivity(...));
	// it reads the pre-assignment context, so check it before applying the update.
	for _, rhs := range s.Rhs {
		wc.scanExecute(rhs, st)
	}

	// The only shape that derives a tracked context is a single
	// `lhs = workflow.With*Options(base, opts)`.
	if len(s.Lhs) == 1 && len(s.Rhs) == 1 {
		if k, base, ok := wc.withKind(s.Rhs[0]); ok {
			if v := wc.identVar(s.Lhs[0]); v != nil && !wc.poisoned[v] {
				var baseSet kindSet
				if base != nil {
					baseSet = st.applied[base].set
				}
				st.applied[v] = ctxInfo{set: baseSet | k.bit(), last: k}
			}
			return
		}
		// A single assignment from anything else (an opaque helper returning a
		// Context, another variable, ...) makes the target's value unknown.
		if v := wc.identVar(s.Lhs[0]); v != nil {
			delete(st.applied, v)
		}
		return
	}

	// A multi-value assignment to a context variable is equally opaque.
	for _, l := range s.Lhs {
		if v := wc.identVar(l); v != nil {
			delete(st.applied, v)
		}
	}
}

// The control-flow walkers below all follow the same shape: interpret the branch
// bodies on a CLONE of the entry state (so a conflict inside a branch is still
// caught), then, since we cannot know which branch ran, reset to "unknown" every
// context variable assigned anywhere in the construct. That makes "assigned in
// different branches with different kinds" collapse to unknown -- no false fire.

func (wc *walkCtx) walkIf(s *ast.IfStmt, st *state) {
	entry := st.clone()
	if s.Init != nil {
		wc.walkStmt(s.Init, entry)
	}
	wc.scanExecute(s.Cond, entry)
	wc.walkStmts(s.Body.List, entry.clone())
	if s.Else != nil {
		wc.walkStmt(s.Else, entry.clone())
	}
	wc.resetAssigned(s, st)
}

func (wc *walkCtx) walkFor(s *ast.ForStmt, st *state) {
	entry := st.clone()
	if s.Init != nil {
		wc.walkStmt(s.Init, entry)
	}
	wc.scanExecute(s.Cond, entry)
	if s.Post != nil {
		wc.walkStmt(s.Post, entry)
	}
	wc.walkStmts(s.Body.List, entry.clone())
	wc.resetAssigned(s, st)
}

func (wc *walkCtx) walkRange(s *ast.RangeStmt, st *state) {
	entry := st.clone()
	wc.scanExecute(s.X, entry)
	wc.walkStmts(s.Body.List, entry.clone())
	wc.resetAssigned(s, st)
}

func (wc *walkCtx) walkSwitch(s *ast.SwitchStmt, st *state) {
	entry := st.clone()
	if s.Init != nil {
		wc.walkStmt(s.Init, entry)
	}
	wc.scanExecute(s.Tag, entry)
	for _, cc := range s.Body.List {
		// A switch body's statements are always case clauses, so this holds.
		clause := cc.(*ast.CaseClause)
		for _, e := range clause.List {
			wc.scanExecute(e, entry)
		}
		wc.walkStmts(clause.Body, entry.clone())
	}
	wc.resetAssigned(s, st)
}

func (wc *walkCtx) walkTypeSwitch(s *ast.TypeSwitchStmt, st *state) {
	entry := st.clone()
	if s.Init != nil {
		wc.walkStmt(s.Init, entry)
	}
	if s.Assign != nil {
		wc.walkStmt(s.Assign, entry)
	}
	for _, cc := range s.Body.List {
		// A type-switch body's statements are always case clauses, so this holds.
		clause := cc.(*ast.CaseClause)
		wc.walkStmts(clause.Body, entry.clone())
	}
	wc.resetAssigned(s, st)
}

func (wc *walkCtx) walkSelect(s *ast.SelectStmt, st *state) {
	entry := st.clone()
	for _, cc := range s.Body.List {
		// A select body's statements are always comm clauses, so this holds.
		clause := cc.(*ast.CommClause)
		branch := entry.clone()
		if clause.Comm != nil {
			wc.walkStmt(clause.Comm, branch)
		}
		wc.walkStmts(clause.Body, branch)
	}
	wc.resetAssigned(s, st)
}
