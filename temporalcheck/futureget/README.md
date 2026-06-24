# futureget

Flags a `.Get` call on `workflow.Future`, `workflow.ChildWorkflowFuture`, or `converter.EncodedValue` whose returned error is discarded.

That error reports a failed activity, failed child workflow, or decode error. Dropping it silently ignores the failure.

## Example

```go
// Bug: error discarded
future := workflow.ExecuteActivity(ctx, a.Publish, batch)
_ = future.Get(ctx, nil) // activity error swallowed

// Correct
if err := future.Get(ctx, &result); err != nil {
    return err
}
```

```
workflow.go:21  Get: the returned error from Future.Get is discarded; check it or assign it to a variable you inspect (future-get)
```

## Settings

| Key        | Default | Description |
|------------|---------|-------------|
| `disabled` | `false` | Turn the analyzer off entirely |

## Limitations

- **Bare discard and `_ =` only** — `err := f.Get(...)` with no following check is not flagged (needs data-flow analysis).
- **Static receiver type** — a user type that embeds `workflow.Future` is not matched.
- **`.Get` only** — `IsReady`, `GetChildWorkflowExecution`, `SignalChildWorkflow` and `converter.EncodedValues` (plural) are out of scope.
