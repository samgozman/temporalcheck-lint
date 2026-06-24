# execargs

Checks that arguments passed to `workflow.ExecuteActivity`, `workflow.ExecuteLocalActivity`, `workflow.ExecuteChildWorkflow`, `workflow.NewContinueAsNewError`, `client.ExecuteWorkflow`, and `client.SignalWithStartWorkflow` match the target function's real signature.

These APIs take the target as `interface{}` and arguments as `...interface{}`, so wrong argument count or type compiles cleanly and only fails at run time.

## Example

```go
// Greet(ctx context.Context, name string) (string, error)
var a *Activities

workflow.ExecuteActivity(ctx, a.Greet)               // missing name → arity error
workflow.ExecuteActivity(ctx, a.Greet, 42)           // int, want string → strict-types error
workflow.ExecuteActivity(ctx, a.Greet, "x", "extra") // too many args → arity error
```

```
workflow.go:14  ExecuteActivity: activity "Greet" expects 1 argument, got 0 (arity)
workflow.go:17  ExecuteActivity: activity "Greet" expects 1 argument, got 2 (arity)
workflow.go:20  ExecuteActivity: arg 1 of "Greet" has type int, want string (strict-types)
```

## Settings

| Key                   | Default | Description |
|-----------------------|---------|-------------|
| `disabled`            | `false` | Turn the analyzer off entirely |
| `strict-types`        | `false` | Also check argument types, not just count |
| `strict-pointers`     | `false` | Flag `T` vs `*T` mismatches (Temporal's default DataConverter treats them as equivalent) |
| `strict-struct-shape` | `false` | Flag passing one struct type where a different struct is wanted (serializes by field name — drops/zeroes mismatched fields) |
| `strict-tests`        | `false` | Also check `testsuite.OnActivity`/`OnWorkflow` mock matcher arity |

The arity check is always on; the others are independent opt-in layers.

### struct-shape example

```go
// Charge(ctx context.Context, p *ChargeParams) error
// type ChargeParams struct { Amount int; Currency string }
// type PayParams struct { Amount int; Note string }

workflow.ExecuteActivity(ctx, a.Charge, &PayParams{Amount: 10})
// strict-struct-shape: sends *PayParams, target wants *ChargeParams — drops {Note} and leaves {Currency} unset
```

### strict-tests example

```go
// testsuite mock: OnActivity(target, matchers...)
// matchers must cover ALL parameters including the injected context
env.OnActivity(a.Greet, mock.Anything) // missing 1 matcher for `name` parameter
// OnActivity: mock for activity "Greet" expects 2 arguments (one per parameter), got 1 (strict-tests)
```

## Limitations

- **String-registered targets are skipped** — `ExecuteActivity(ctx, "MyActivity", ...)` can't be resolved to a signature. Use the [`stringtarget`](../stringtarget) analyzer to flag those.
- **Spread calls are skipped** — `ExecuteActivity(ctx, fn, slice...)` can't be matched positionally.
- **Type check is stricter than Temporal** — Temporal's `DataConverter` can round-trip types that Go's assignability rejects (e.g. `int` vs `int32` via JSON). The arity check is the false-positive-free baseline.
- **`strict-struct-shape` models JSON only** — exported fields matched by `json` tag name; embedded fields and nested structs are not followed.
