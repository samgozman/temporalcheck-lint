# workeroptions

Flags `worker.Options` composite literals with fields that cause a worker panic or missing concurrency configuration.

## Rules

### `(worker-panic)` — on by default

`MaxConcurrentWorkflowTaskExecutionSize` and `MaxConcurrentWorkflowTaskPollers` must not be `1`: the pollers alternate between sticky and non-sticky queues, so a single one deadlocks the worker at boot.

```go
// Bug: panics on worker.Run
w := worker.New(c, "task-queue", worker.Options{
    MaxConcurrentWorkflowTaskPollers: 1,
})

// Correct: use 0 (SDK default) or >= 2
w := worker.New(c, "task-queue", worker.Options{
    MaxConcurrentWorkflowTaskPollers: 2,
})
```

```
worker.go:14  worker.Options: MaxConcurrentWorkflowTaskPollers must not be 1 — the worker panics on start; use 0 for the default or a value >= 2 (worker-panic)
```

### `(require-options)` — opt-in

Flags a `worker.New` call whose `worker.Options` literal sets none of the five concurrency limits. The SDK defaults (1k concurrent executions, 100k actions/s) can overload a self-hosted cluster or crash a memory-capped pod.

```go
// Flagged with require-options: true — no concurrency limits set
w := worker.New(c, "task-queue", worker.Options{})
```

```
worker.go:22  worker.New: worker.Options sets no concurrency limits, so the worker runs on the SDK defaults (1k executions, 100k/s) that can overload a self-hosted cluster; set MaxConcurrent* limits (require-options)
```

## Settings

| Key               | Default | Description |
|-------------------|---------|-------------|
| `disabled`        | `false` | Turn the analyzer off entirely (also disables `worker-panic`) |
| `require-options` | `false` | Flag `worker.New` calls whose options set no concurrency limits |

## Limitations

- **Constants only for `worker-panic`** — a non-constant value (variable, expression) is skipped rather than risked as a false positive.
- **Literal at call site for `require-options`** — if the third argument to `worker.New` is a variable, the call is skipped.
- **Activity counterparts are not restricted** — only the workflow-task fields have the "must not be 1" constraint; activity fields are never flagged.
