# nonserializable

Flags activity and workflow parameters or non-error returns typed as `chan` or `func` — types that Temporal's `DataConverter` cannot serialize.

`encoding/json` returns an "unsupported type" error for both, so a parameter or result of that type can never make the round trip.

An opt-in rule also flags structs that have fields but none exported: JSON serializes only exported fields, so all data is silently dropped on the wire.

## Example

```go
// Bug: chan parameter can't be serialized
func Stream(ctx context.Context, out chan int) error { ... }
workflow.ExecuteActivity(ctx, Stream, make(chan int)) // fails to serialize at run time

// Correct
func Stream(ctx context.Context, batchSize int) error { ... }
```

```
workflow.go:21  activity "Stream" parameter 1 has type chan int; Temporal's DataConverter cannot serialize a channel or function -- use a serializable type (unencodable)
```

### opt-in: empty-struct

```go
type internalState struct{ counter int } // no exported fields → encodes to {}

func Process(ctx context.Context, s internalState) error { ... }
// activity "Process" parameter 1 has type internalState (no exported fields); JSON encodes to {} and drops all data (empty-struct)
```

## Settings

| Key            | Default | Description |
|----------------|---------|-------------|
| `disabled`     | `false` | Turn the analyzer off entirely |
| `empty-struct` | `false` | Also flag structs with fields but none exported (unless they implement `json.Marshaler`) |

## Limitations

- **Top-level only** — `chan` inside a struct field or `[]chan int` is not flagged.
- **Resolvable targets only** — string-registered targets are skipped.
- **`json.Marshaler` exclusion** for `empty-struct` — types that control their own encoding (`time.Time`, `json.RawMessage`) are not flagged.
