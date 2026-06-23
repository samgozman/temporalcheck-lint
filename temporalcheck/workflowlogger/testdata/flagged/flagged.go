// Package flagged exercises every logging call the analyzer reports inside
// workflow code: the log, log/slog and fmt standard-library packages (functions
// and *Logger methods), fmt.Fprint* to os.Stdout/os.Stderr, and zerolog chains --
// reached directly, inside a workflow.Go closure, from a method workflow, and from
// a workflow literal nested in an ordinary function.
package flagged

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"go.temporal.io/sdk/workflow"
)

func Workflow(ctx workflow.Context) error {
	// Standard library log: print family plus Fatal/Panic, which log and exit.
	log.Print("a")      // want `logging via log in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	log.Printf("%d", 1) // want `logging via log in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	log.Println("a")    // want `logging via log in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	log.Fatal("a")      // want `logging via log in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	log.Panicf("%d", 1) // want `logging via log in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`

	// A *log.Logger method shares the log package path, so it is flagged too.
	l := log.New(os.Stdout, "", 0)
	l.Printf("%d", 2) // want `logging via log in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`

	// log/slog functions and a *slog.Logger method.
	slog.Info("msg")  // want `logging via slog in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	slog.Error("msg") // want `logging via slog in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	sl := slog.New(nil)
	sl.Warn("w") // want `logging via slog in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`

	// fmt print family always; Fprint* only when writing to os.Stdout/os.Stderr.
	fmt.Print("a")                  // want `logging via fmt in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	fmt.Printf("%d", 1)             // want `logging via fmt in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	fmt.Println("a")                // want `logging via fmt in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	fmt.Fprintln(os.Stdout, "a")    // want `logging via fmt in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	fmt.Fprintf(os.Stderr, "%d", 1) // want `logging via fmt in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`

	// zerolog: the global-logger chain and a Logger value's chain. The diagnostic
	// fires once, at the start of the chain, not per chained method.
	zlog.Info().Msg("x")                // want `logging via zerolog in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	zlog.Error().Str("k", "v").Msg("y") // want `logging via zerolog in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	zl := zerolog.New(os.Stdout)
	zl.Info().Msg("z") // want `logging via zerolog in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`

	// A workflow.Go coroutine is still workflow code, so logging inside its closure
	// is flagged.
	workflow.Go(ctx, func(ctx workflow.Context) {
		log.Println("inside go") // want `logging via log in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	})

	return nil
}

type App struct{}

// A method workflow (first parameter workflow.Context) is a workflow definition
// too, so logging inside it is flagged.
func (a *App) Run(ctx workflow.Context) error {
	slog.Info("from method workflow") // want `logging via slog in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
	return nil
}

// register is not a workflow definition, but the function literal it registers is
// (its first parameter is workflow.Context), so logging inside that literal is
// flagged -- the analyzer reaches workflow definitions nested in ordinary
// functions, not only top-level ones.
func register() {
	run := func(ctx workflow.Context) error {
		fmt.Println("from nested workflow literal") // want `logging via fmt in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`
		return nil
	}
	_ = run
}
