package flagged

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// Activities and workflows whose top-level parameter or return types are chan or
// func -- types the DataConverter can never serialize. Each is referenced by an
// Execute* call below, where the analyzer anchors its diagnostic.

func chanParam(ctx context.Context, ch chan int) error { return nil }

func funcParam(ctx context.Context, f func() error) error { return nil }

func variadicChan(ctx context.Context, chs ...chan int) error { return nil }

func chanReturn(ctx context.Context) (chan int, error) { return nil, nil }

func funcReturn(ctx context.Context) (func(), error) { return nil, nil }

// Stopper is a named channel type: its underlying type is a channel, so it is
// just as unserializable as a bare `chan struct{}`.
type Stopper chan struct{}

func namedChanParam(ctx context.Context, s Stopper) error { return nil }

// Handler is a named function type, unserializable for the same reason.
type Handler func(int) int

func namedFuncParam(ctx context.Context, h Handler) error { return nil }

// noCtxParam omits the optional leading context.Context, so its first parameter
// is the unserializable one.
func noCtxParam(ch chan int) error { return nil }

func childWorkflow(ctx workflow.Context, f func()) error { return nil }

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, chanParam, nil)          // want `activity "chanParam" parameter 1 has type chan int;.*\(unencodable\)`
	workflow.ExecuteActivity(ctx, funcParam, nil)          // want `activity "funcParam" parameter 1 has type func\(\) error;.*\(unencodable\)`
	workflow.ExecuteLocalActivity(ctx, variadicChan, nil)  // want `activity "variadicChan" parameter 1 has type chan int;.*\(unencodable\)`
	workflow.ExecuteActivity(ctx, chanReturn)              // want `activity "chanReturn" return 1 has type chan int;.*\(unencodable\)`
	workflow.ExecuteActivity(ctx, funcReturn)              // want `activity "funcReturn" return 1 has type func\(\);.*\(unencodable\)`
	workflow.ExecuteActivity(ctx, namedChanParam, nil)     // want `activity "namedChanParam" parameter 1 has type flagged\.Stopper;.*\(unencodable\)`
	workflow.ExecuteActivity(ctx, namedFuncParam, nil)     // want `activity "namedFuncParam" parameter 1 has type flagged\.Handler;.*\(unencodable\)`
	workflow.ExecuteActivity(ctx, noCtxParam, nil)         // want `activity "noCtxParam" parameter 1 has type chan int;.*\(unencodable\)`
	workflow.ExecuteChildWorkflow(ctx, childWorkflow, nil) // want `child workflow "childWorkflow" parameter 1 has type func\(\);.*\(unencodable\)`
	// NewContinueAsNewError restarts a workflow, so its target carries the same
	// leading workflow.Context and its unserializable parameter is reported the same way.
	_ = workflow.NewContinueAsNewError(ctx, childWorkflow, nil) // want `workflow "childWorkflow" parameter 1 has type func\(\);.*\(unencodable\)`
}
