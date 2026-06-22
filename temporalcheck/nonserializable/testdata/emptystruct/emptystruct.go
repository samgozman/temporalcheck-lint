package emptystruct

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// secret has fields but none exported: JSON encodes it to "{}", silently dropping
// its data. With the EmptyStruct check on, it is flagged.
type secret struct{ v int }

// stringerSecret also has no exported fields. Its String method is not
// MarshalJSON, so it does not control its own JSON encoding and is still flagged.
type stringerSecret struct{ v int }

func (s stringerSecret) String() string { return "secret" }

// marshalable has no exported fields but implements json.Marshaler, so it controls
// its own encoding and must NOT be flagged.
type marshalable struct{ v int }

func (m marshalable) MarshalJSON() ([]byte, error) { return []byte(`"ok"`), nil }

// empty has no fields at all, so it carries no data and round-trips fine.
type empty struct{}

// exported has an exported field, so JSON preserves its data.
type exported struct{ V int }

func secretParam(ctx context.Context, s secret) error { return nil }

func stringerParam(ctx context.Context, s stringerSecret) error { return nil }

func marshalableParam(ctx context.Context, m marshalable) error { return nil }

func emptyParam(ctx context.Context, e empty) error { return nil }

func exportedParam(ctx context.Context, e exported) error { return nil }

func secretReturn(ctx context.Context) (secret, error) { return secret{}, nil }

// chanParam confirms the always-on chan/func check still fires when EmptyStruct
// is enabled.
func chanParam(ctx context.Context, ch chan int) error { return nil }

func caller(ctx workflow.Context) {
	workflow.ExecuteActivity(ctx, secretParam, secret{})           // want `activity "secretParam" parameter 1 has type emptystruct\.secret with no exported fields;.*\(empty-struct\)`
	workflow.ExecuteActivity(ctx, stringerParam, stringerSecret{}) // want `activity "stringerParam" parameter 1 has type emptystruct\.stringerSecret with no exported fields;.*\(empty-struct\)`
	workflow.ExecuteActivity(ctx, marshalableParam, marshalable{}) // json.Marshaler: not flagged
	workflow.ExecuteActivity(ctx, emptyParam, empty{})             // fieldless struct{}: not flagged
	workflow.ExecuteActivity(ctx, exportedParam, exported{})       // has an exported field: not flagged
	workflow.ExecuteActivity(ctx, secretReturn)                    // want `activity "secretReturn" return 1 has type emptystruct\.secret with no exported fields;.*\(empty-struct\)`
	workflow.ExecuteActivity(ctx, chanParam, nil)                  // want `activity "chanParam" parameter 1 has type chan int;.*\(unencodable\)`
}
