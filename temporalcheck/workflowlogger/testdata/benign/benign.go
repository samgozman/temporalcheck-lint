// Package benign collects the patterns the analyzer must NOT flag: logging from a
// non-workflow function and from an activity (neither replays), the SDK's
// replay-aware workflow.GetLogger, fmt formatting helpers that do not emit, calls
// with no logging callee, and fmt.Fprint* to a writer that is not a standard
// stream.
package benign

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"

	"go.temporal.io/sdk/workflow"
)

// helper is not workflow code, so stdlib logging here is fine.
func helper() {
	log.Println("fine here")
	fmt.Println("also fine")
}

// doWork is a plain, non-logging function; calling it from a workflow is fine and
// exercises the resolved-but-not-logging path.
func doWork() {}

// sink holds an io.Writer field, a writer that is not os.Stdout/os.Stderr.
type sink struct{ w io.Writer }

// Activity's first parameter is the standard context.Context, not
// workflow.Context, so it is not a workflow definition: activities do not replay,
// and logging in them is not a determinism hazard.
func Activity(ctx context.Context) error {
	log.Printf("activities may log freely: %v", ctx.Err())
	return nil
}

func Workflow(ctx workflow.Context) error {
	// The SDK's replay-aware logger is the correct way to log from a workflow.
	workflow.GetLogger(ctx).Info("starting")

	// fmt formatting helpers that return a value instead of emitting are fine.
	_ = fmt.Sprintf("x=%d", 1)
	_ = fmt.Errorf("boom: %d", 2)

	// A resolved non-logging function call and an immediately-invoked function
	// literal (no named callee) are both skipped.
	doWork()
	func() { _ = 1 }()

	// Fprint* to a buffer or another non-standard writer builds/forwards bytes; it
	// is not logging, so it is not flagged.
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "n=%d", 3)
	_ = buf.String()

	s := sink{}
	fmt.Fprintln(s.w, "to a field writer, not a standard stream")

	return nil
}
