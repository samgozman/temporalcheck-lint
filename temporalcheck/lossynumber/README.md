# lossynumber

Flags activity and workflow parameters or non-error returns typed as `interface{}`/`any`, `map[string]any`, or `[]any`.

Temporal's default `DataConverter` uses JSON, which decodes every JSON number into `float64` when the destination is `interface{}`. A `float64` can't represent integers past 2^53 exactly, so large integers silently lose precision on the wire.

## Example

```go
// Bug: any parameter — the worker decodes into interface{}, loses int64 precision
func Charge(ctx context.Context, amount any) error { ... }

var amount int64 = 9007199254740993 // 2^53 + 1
workflow.ExecuteActivity(ctx, Charge, amount)
// Inside Charge: amount is float64(9007199254740992) — off by one, no error

// Correct
func Charge(ctx context.Context, amount int64) error { ... }
```

```
workflow.go:21  activity "Charge" parameter 1 has dynamic type any; Temporal's JSON converter decodes numbers as float64 and silently loses int64 precision past 2^53 — use a concrete type (lossy-types)
```

## Settings

| Key        | Default | Description |
|------------|---------|-------------|
| `disabled` | `false` | Turn the analyzer off entirely |

Disable only if you use a custom `DataConverter` that preserves integer precision (e.g. decodes into `json.Number` or `int64`).

## Limitations

- **Top-level only** — `any` inside a struct field or `[][]any` is not flagged.
- **Named empty interfaces count** — `type Payload interface{}` is flagged; non-empty interfaces (`error`, `io.Reader`) are not.
- **Resolvable targets only** — string-registered targets are skipped.
