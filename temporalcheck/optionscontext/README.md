# optionscontext

Flags `ExecuteActivity`, `ExecuteLocalActivity`, and `ExecuteChildWorkflow` calls whose context was configured with a **conflicting** `With*Options` helper in the same function.

Each execute function reads its options from a distinct context key. `WithActivityOptions`, `WithLocalActivityOptions`, and `WithChildOptions` each set a different key; crossing them compiles cleanly but fails at run time.

## Example

```go
// Bug: ctx has child options, but we're calling ExecuteActivity
ctx = workflow.WithChildOptions(ctx, cwo)
workflow.ExecuteActivity(ctx, a.Greet) // child options set, activity options never applied → fails at run time

// Correct
ctx = workflow.WithActivityOptions(ctx, ao)
workflow.ExecuteActivity(ctx, a.Greet)
```

```
workflow.go:14  ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions, so the activity options never apply; derive it with ctx = workflow.WithActivityOptions(ctx, opts) (options-context)
```

## Settings

| Key        | Default | Description |
|------------|---------|-------------|
| `disabled` | `false` | Turn the analyzer off entirely |

## How it works

The check is **intra-procedural** and fires only on a **seen contradiction** — a context that carries the wrong helper, applied in the same function, with no matching one in sight. It never fires on absence (a context missing the right helper), keeping it near-zero false-positive.

It bails to "unknown" (reports nothing) when it loses track: a bare function parameter, an opaque reassignment, a closure capture, or a branch with different kinds.

## Limitations

- **Intra-procedural** — a context configured in a helper and passed in is treated as unknown.
- **Seen contradictions only** — absence of the right helper is never flagged.
- **Plain variables only** — a context in a struct field or returned from a call is skipped.
