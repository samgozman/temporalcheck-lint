# workflowstate

Flags mutation of package-level variables from workflow code.

Temporal workflows replay: the SDK re-executes workflow code against recorded history. Mutating a package-level variable from workflow code breaks replay (the write is not part of the replayed state) and races across concurrent workflow executions in the same worker process.

## Example

```go
var requestCount int // package level

func MyWorkflow(ctx workflow.Context) error {
    requestCount++ // flagged: non-deterministic and races across executions
    return nil
}
```

```
workflow.go:10  mutates package-level variable requestCount from workflow code; shared mutable state breaks replay determinism and races across workflow executions (global-mutation)
```

The idiomatic capture-and-mutate of a **local** — the SDK's pattern for moving data between coroutines — is **not** flagged:

```go
total := 0
workflow.Go(ctx, func(ctx workflow.Context) {
    total++ // captured local — fine, and NOT flagged
})
workflow.Await(ctx, func() bool { return total == 5 })
```

## Settings

| Key        | Default | Description |
|------------|---------|-------------|
| `disabled` | `false` | Turn the analyzer off entirely |

## What counts as workflow code

Any function whose first parameter is `workflow.Context`, including closures lexically nested in it (`workflow.Go` coroutines, `Await` conditions, `Selector` callbacks).

## Limitations

- **Direct mutation only — no call graph** — a package-level variable mutated inside a helper that doesn't take `workflow.Context` is not flagged; detecting that needs transitive analysis.
- **Mutation only, not reads** — reading a package-level variable (e.g. immutable config) is not flagged.
- **Receivers are not globals** — a workflow method can mutate its receiver's fields without triggering this check.
