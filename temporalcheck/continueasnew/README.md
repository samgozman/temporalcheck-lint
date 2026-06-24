# continueasnew

Flags a `workflow.NewContinueAsNewError` call whose result is discarded rather than returned.

Returning the error is the only way the call has any effect. Constructing it without returning it silently drops the continue-as-new signal — the workflow falls through and ends instead of continuing.

## Example

```go
// Bug: built but not returned
if shouldContinue {
    workflow.NewContinueAsNewError(ctx, MyWorkflow, next) // built, never returned
}
return nil // workflow ends here — the continue-as-new never happens

// Correct
if shouldContinue {
    return workflow.NewContinueAsNewError(ctx, MyWorkflow, next)
}
```

```
workflow.go:14  NewContinueAsNewError: the continue-as-new error is discarded; return it so the workflow continues as new (continue-as-new)
```

## Settings

| Key        | Default | Description |
|------------|---------|-------------|
| `disabled` | `false` | Turn the analyzer off entirely |

## Limitations

- **Bare discard and `_ =` only** — a result assigned to a named variable (`err := workflow.NewContinueAsNewError(...)`) is not flagged, since a `return err` may follow.
- **Call site only** — a continue-as-new error passed into a helper and returned from there is not tracked.
