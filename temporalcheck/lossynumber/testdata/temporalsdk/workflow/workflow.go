// Package workflow is a minimal stand-in for go.temporal.io/sdk/workflow. It
// exists only so the analyzer's testdata type-checks without vendoring the real
// Temporal SDK. The Execute* signatures take the target as interface{} and the
// arguments as variadic interface{}, which is exactly the type erasure the
// analyzer resolves by inspecting the target's real signature instead.
package workflow

import "go.temporal.io/sdk/internal"

// Context mirrors the real SDK, which publishes workflow.Context as an alias to
// an internal type rather than declaring it directly in this package.
type Context = internal.Context

type Future interface {
	Get(ctx Context, valuePtr any) error
}

type ChildWorkflowFuture interface{ Future }

func ExecuteActivity(ctx Context, activity any, args ...any) Future { return nil }

func ExecuteLocalActivity(ctx Context, activity any, args ...any) Future { return nil }

func ExecuteChildWorkflow(ctx Context, childWorkflow any, args ...any) ChildWorkflowFuture {
	return nil
}
