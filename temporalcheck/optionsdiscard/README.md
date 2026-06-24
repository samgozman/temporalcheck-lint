# optionsdiscard

Flags `workflow.WithActivityOptions`, `workflow.WithLocalActivityOptions`, and `workflow.WithChildOptions` calls whose returned context is discarded.

These functions do **not** mutate the context you pass — each returns a *new* context that carries the options. Forgetting the `ctx =` means the options silently never apply.

## Example

```go
// Bug: result thrown away
workflow.WithActivityOptions(ctx, ao)  // result discarded
workflow.ExecuteActivity(ctx, a.Greet) // runs with the original ctx — no options

// Correct
ctx = workflow.WithActivityOptions(ctx, ao)
workflow.ExecuteActivity(ctx, a.Greet)
```

```
workflow.go:14  WithActivityOptions: the returned context is discarded, so the options never apply; assign it back with ctx = workflow.WithActivityOptions(ctx, opts) (options-discard)
```

## Settings

| Key        | Default | Description |
|------------|---------|-------------|
| `disabled` | `false` | Turn the analyzer off entirely |

## Limitations

- **Bare discard and `_ =` only** — a result assigned to a named variable is assumed kept, even if never used afterwards (tracking later use is out of scope).
