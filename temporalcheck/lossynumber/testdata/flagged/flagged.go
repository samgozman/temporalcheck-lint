package flagged

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// Activities and workflows whose top-level parameter or return types are lossy
// dynamic types. Each is referenced by an Execute* call below, where the
// analyzer anchors its diagnostic.

func anyParam(ctx context.Context, v any) error { return nil }

func mapParam(ctx context.Context, m map[string]any) error { return nil }

func sliceParam(ctx context.Context, s []any) error { return nil }

func variadicParam(ctx context.Context, xs ...any) error { return nil }

func anyReturn(ctx context.Context) (any, error) { return nil, nil }

// Payload is a named empty interface: zero methods, so it decodes exactly like
// interface{}.
type Payload interface{}

func namedParam(ctx context.Context, p Payload) error { return nil }

// noCtxParam omits the optional leading context.Context, so its first parameter
// is the lossy one.
func noCtxParam(v any) error { return nil }

func childWorkflow(ctx workflow.Context, v any) error { return nil }

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, anyParam, 1)           // want `activity "anyParam" parameter 1 has dynamic type any;.*\(lossy-types\)`
	workflow.ExecuteActivity(ctx, mapParam, nil)         // want `activity "mapParam" parameter 1 has dynamic type map\[string\]any;.*\(lossy-types\)`
	workflow.ExecuteLocalActivity(ctx, sliceParam, nil)  // want `activity "sliceParam" parameter 1 has dynamic type \[\]any;.*\(lossy-types\)`
	workflow.ExecuteActivity(ctx, variadicParam, 1, 2)   // want `activity "variadicParam" parameter 1 has dynamic type \[\]any;.*\(lossy-types\)`
	workflow.ExecuteActivity(ctx, anyReturn)             // want `activity "anyReturn" return 1 has dynamic type any;.*\(lossy-types\)`
	workflow.ExecuteActivity(ctx, namedParam, nil)       // want `activity "namedParam" parameter 1 has dynamic type flagged\.Payload;.*\(lossy-types\)`
	workflow.ExecuteActivity(ctx, noCtxParam, 1)         // want `activity "noCtxParam" parameter 1 has dynamic type any;.*\(lossy-types\)`
	workflow.ExecuteChildWorkflow(ctx, childWorkflow, 1) // want `child workflow "childWorkflow" parameter 1 has dynamic type any;.*\(lossy-types\)`
	// NewContinueAsNewError restarts a workflow, so its target carries the same
	// leading workflow.Context and its lossy parameter is reported the same way.
	_ = workflow.NewContinueAsNewError(ctx, childWorkflow, 1) // want `workflow "childWorkflow" parameter 1 has dynamic type any;.*\(lossy-types\)`
}
