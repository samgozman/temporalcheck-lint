// Package workflow is a minimal stand-in for go.temporal.io/sdk/workflow.
// It exists only so the analyzer's testdata type-checks without vendoring the
// real Temporal SDK. The signatures mirror the SDK's: the target is interface{}
// and the arguments are variadic interface{}, which is exactly the type erasure
// that lets a bare string stand in for the target.
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
