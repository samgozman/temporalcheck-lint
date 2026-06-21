package benign

import (
	"context"
	"fmt"
	"io"

	"go.temporal.io/sdk/workflow"
)

// Result carries an `any` field, but a field is deep nesting -- the analyzer
// looks only at the top-level parameter/return type, so passing Result is fine.
type Result struct {
	Data  any
	Count int64
}

// Stringer is a non-empty interface: it has a method, so it is not the lossy
// empty interface and must never be flagged.
type Stringer interface{ String() string }

func concrete(ctx context.Context, v int64) (Result, error) { return Result{}, nil }

func structParam(ctx context.Context, r Result) error { return nil }

func ifaceParam(ctx context.Context, rd io.Reader) error { return nil }

func errParam(ctx context.Context, e error) error { return nil }

func customIface(ctx context.Context, s Stringer) error { return nil }

func concreteCollections(ctx context.Context, m map[string]int, b []byte, ss []string) error {
	return nil
}

func onlyErr(ctx context.Context) error { return nil }

func structReturn(ctx context.Context) (Result, error) { return Result{}, nil }

func ifaceReturn(ctx context.Context) (Stringer, error) { return nil, nil }

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, concrete, 1)
	workflow.ExecuteActivity(ctx, structParam, Result{})
	workflow.ExecuteActivity(ctx, ifaceParam, nil)
	workflow.ExecuteActivity(ctx, errParam, nil)
	workflow.ExecuteActivity(ctx, customIface, nil)
	workflow.ExecuteActivity(ctx, concreteCollections, nil, nil, nil)
	workflow.ExecuteActivity(ctx, onlyErr)
	workflow.ExecuteActivity(ctx, structReturn)
	workflow.ExecuteActivity(ctx, ifaceReturn)
	// A string-registered target resolves to no signature, so it is out of scope
	// -- skipped rather than guessed at.
	workflow.ExecuteActivity(ctx, "StringRegistered", 1)
}

type holder struct{ fn func() }

func localHelper() {}

// misc exercises call shapes that are not Execute* entry points: a non-selector
// call, a selector whose name resolves to a struct field (not a function), and a
// selector into another package -- none of which the analyzer touches.
func misc(h holder) {
	localHelper()
	h.fn()
	_ = fmt.Sprint("x")
}
