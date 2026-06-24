# stringtarget

Flags `ExecuteActivity`, `ExecuteLocalActivity`, `ExecuteChildWorkflow`, `NewContinueAsNewError`, `client.ExecuteWorkflow`, and `client.SignalWithStartWorkflow` calls whose target argument is a string.

A string target is opaque: it can't be resolved to a signature, so argument count and types go unchecked, and it silently disables the [`execargs`](../execargs) analyzer for that call.

Passing the function reference instead — `workflow.ExecuteActivity(ctx, a.MyActivity, ...)` — derives the name from the function rather than duplicating it as a fragile string, and lets `execargs` verify the call.

## Example

```go
// Flagged
workflow.ExecuteActivity(ctx, "MyActivity", arg1, arg2)

// Correct
workflow.ExecuteActivity(ctx, a.MyActivity, arg1, arg2)
```

```
workflow.go:14  ExecuteActivity: target "MyActivity" is named by string; pass the function reference instead so its arguments can be checked statically (string-target)
```

## Settings

| Key            | Default | Description |
|----------------|---------|-------------|
| `enabled`      | `false` | Master switch — analyzer is silent until this is `true` |
| `strict-tests` | `false` | Also flag string-named targets in `testsuite.OnActivity`/`OnWorkflow` (requires `enabled: true`) |

This check is **off by default**: naming a target by string is sometimes necessary (an activity implemented in another service or language).

## Limitations

- **It does not resolve the named target** — reports only that the target is a string; use the function reference form so `execargs` can check the call.
