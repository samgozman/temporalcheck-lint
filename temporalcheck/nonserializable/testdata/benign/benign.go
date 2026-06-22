package benign

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/workflow"
)

// Holder carries a chan field, but a field is deep nesting -- the analyzer looks
// only at the top-level parameter/return type, so passing Holder is fine.
type Holder struct {
	Ch    chan int
	Fn    func()
	Count int64
}

// secret has fields but none exported. JSON would encode it to "{}", but that
// check is opt-in (EmptyStruct) and off here, so it must not be flagged.
type secret struct{ v int }

func concrete(ctx context.Context, v int64) (Holder, error) { return Holder{}, nil }

func structParam(ctx context.Context, h Holder) error { return nil }

func chanSliceParam(ctx context.Context, chs []chan int) error { return nil }

func secretParam(ctx context.Context, s secret) error { return nil }

func onlyErr(ctx context.Context) error { return nil }

func structReturn(ctx context.Context) (Holder, error) { return Holder{}, nil }

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, concrete, 1)
	workflow.ExecuteActivity(ctx, structParam, Holder{})
	// A slice of channels is deeper than the top level, so it is not flagged.
	workflow.ExecuteActivity(ctx, chanSliceParam, nil)
	// secret has no exported fields, but the empty-struct check is opt-in and off.
	workflow.ExecuteActivity(ctx, secretParam, secret{})
	workflow.ExecuteActivity(ctx, onlyErr)
	workflow.ExecuteActivity(ctx, structReturn)
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
