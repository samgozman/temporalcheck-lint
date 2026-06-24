package workflowlogger

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"golang.org/x/tools/go/analysis"
)

// Logging packages matched by import path through go/types, so aliased imports
// (import slg "log/slog") still resolve and we never import these packages here.
const (
	logPkg     = "log"
	slogPkg    = "log/slog"
	fmtPkg     = "fmt"
	zerologPkg = "github.com/rs/zerolog"
	osPkg      = "os"
)

// logFuncs are the output-producing names in the log package, shared by the
// package-level functions (log.Printf) and the *log.Logger methods (l.Printf).
// Fatal*/Panic* are included: they log and then exit/panic, even worse from a
// workflow. Constructors and configuration (New, Default, SetFlags, ...) are not
// logging output, so they are deliberately absent.
var logFuncs = map[string]bool{
	"Print": true, "Printf": true, "Println": true,
	"Fatal": true, "Fatalf": true, "Fatalln": true,
	"Panic": true, "Panicf": true, "Panicln": true,
	"Output": true,
}

// slogFuncs are the output-producing names in log/slog, shared by the
// package-level functions and the *slog.Logger methods. With/WithGroup/Enabled
// and the Handler accessors do not emit a record, so they are absent.
var slogFuncs = map[string]bool{
	"Debug": true, "Info": true, "Warn": true, "Error": true, "Log": true,
	"DebugContext": true, "InfoContext": true, "WarnContext": true, "ErrorContext": true,
	"LogAttrs": true,
}

// fmtPrintFuncs are the fmt functions that write to standard output. Sprint*/
// Errorf format without emitting, so they are not logging and not listed.
var fmtPrintFuncs = map[string]bool{
	"Print": true, "Printf": true, "Println": true,
}

// fmtFprintFuncs write to an arbitrary io.Writer; they only count as logging when
// that writer is os.Stdout/os.Stderr (see writesToStdStream). Writing to a buffer
// or strings.Builder is not logging, so a blanket match would be a false positive.
var fmtFprintFuncs = map[string]bool{
	"Fprint": true, "Fprintf": true, "Fprintln": true,
}

// reportLogging walks a workflow definition's body -- including the closures
// lexically nested in it, since those run as part of the same workflow execution
// -- and reports each stdlib/zerolog logging call. On a match it stops descending
// so a chained call (zerolog's log.Info().Msg(...)) or a logging call nested in
// another call's arguments yields a single diagnostic.
func (c *checker) reportLogging(pass *analysis.Pass, nolint nolint.Info, body *ast.BlockStmt) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		label, ok := loggingLabel(pass, call)
		if !ok {
			return true
		}
		// Honor //nolint ourselves so suppression works the same way in
		// standalone/analysistest runs, not only under golangci-lint. Either way we
		// stop descending: a suppressed call's inner chain must not be re-reported.
		if !nolint.Suppresses(pass.Fset, call) {
			pass.Reportf(call.Pos(),
				"logging via %s in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger(ctx) instead (%s)",
				label, tagWorkflowLogger)
		}
		return false
	})
}

// loggingLabel reports whether call resolves to a stdlib or zerolog logging call
// and, if so, the package label naming it in the diagnostic. Calls whose callee
// cannot be resolved to a function are skipped rather than guessed at.
func loggingLabel(pass *analysis.Pass, call *ast.CallExpr) (string, bool) {
	fn := calleeFunc(pass, call)
	if fn == nil || fn.Pkg() == nil {
		return "", false
	}
	path := fn.Pkg().Path()
	name := fn.Name()
	switch {
	case path == logPkg && logFuncs[name]:
		return "log", true
	case path == slogPkg && slogFuncs[name]:
		return "slog", true
	case path == fmtPkg && fmtPrintFuncs[name]:
		return "fmt", true
	case path == fmtPkg && fmtFprintFuncs[name] && writesToStdStream(pass, call):
		return "fmt", true
	case (path == zerologPkg || strings.HasPrefix(path, zerologPkg+"/")) && isMethod(fn):
		// zerolog logs through method chains on its Logger/Event types (e.g.
		// log.Info().Msg("x")); the terminal Msg/Send is a method, so the chain is
		// caught at its outermost call. Requiring a receiver excludes package-level
		// constructors and helpers (zerolog.New, zerolog.Nop) that are not logging.
		return "zerolog", true
	}
	return "", false
}

// isMethod reports whether fn is a method (has a receiver) rather than a
// package-level function.
func isMethod(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	return ok && sig.Recv() != nil
}

// calleeFunc resolves the function or method a call invokes, via Uses (not the
// source text) so aliased imports still match. It returns nil for a call whose
// callee is not a plain function/method (a func-typed value, a conversion).
func calleeFunc(pass *analysis.Pass, call *ast.CallExpr) *types.Func {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		fn, _ := pass.TypesInfo.Uses[fun.Sel].(*types.Func)
		return fn
	case *ast.Ident:
		fn, _ := pass.TypesInfo.Uses[fun].(*types.Func)
		return fn
	default:
		return nil
	}
}

// writesToStdStream reports whether a fmt.Fprint* call's first argument is
// os.Stdout or os.Stderr -- the case where it is really logging. A writer that is
// not statically one of those (a buffer, a variable) is skipped, keeping the check
// free of false positives on non-logging Fprint* uses.
func writesToStdStream(pass *analysis.Pass, call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	sel, ok := call.Args[0].(*ast.SelectorExpr)
	if !ok {
		return false
	}
	v, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Var)
	if !ok || v.Pkg() == nil || v.Pkg().Path() != osPkg {
		return false
	}
	return v.Name() == "Stdout" || v.Name() == "Stderr"
}
