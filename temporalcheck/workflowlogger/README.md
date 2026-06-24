# workflowlogger

Flags standard-library or third-party logging calls made from Temporal workflow code.

Temporal workflows **replay**: a worker re-executes workflow code against recorded history. A non-replay-aware logger emits its line on every replay. The SDK's `workflow.GetLogger(ctx)` is wired into the replay machinery and suppresses duplicate output.

## Example

```go
func MyWorkflow(ctx workflow.Context) error {
    log.Printf("processing order %s", id) // re-logged on every replay — flagged
    return nil
}

// Correct
func MyWorkflow(ctx workflow.Context) error {
    workflow.GetLogger(ctx).Info("processing order", "id", id) // replay-aware
    return nil
}
```

```
workflow.go:10  logging via log in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger(ctx) instead (workflow-logger)
```

## What it matches

| Package | Matched calls |
|---------|---------------|
| `log` | `Print`/`Printf`/`Println`, `Fatal*`, `Panic*`, `Output` — package functions and `*log.Logger` methods |
| `log/slog` | `Debug`/`Info`/`Warn`/`Error`/`Log`, their `*Context` forms, `LogAttrs` — package functions and `*slog.Logger` methods |
| `fmt` | `Print`/`Printf`/`Println`; `Fprint`/`Fprintf`/`Fprintln` only when the writer is `os.Stdout`/`os.Stderr` |
| `zerolog` | any logging chain (e.g. `log.Info().Msg(...)`) — matched by import path, reported once at the outermost call |

Activities (first parameter `context.Context`, not `workflow.Context`) are **not** workflow code and are left alone.

## Settings

| Key       | Default | Description |
|-----------|---------|-------------|
| `enabled` | `false` | Turn the analyzer on (opt-in) |

This check is **off by default**: some teams deliberately route workflow logging through a custom logger or other means, so the analyzer stays silent until opted in.

## Limitations

- **Direct logging only — no call graph** — a logging call inside a helper that doesn't take `workflow.Context` is not flagged.
- **`fmt.Fprint*` only to standard streams** — a write to a buffer or non-stdout/stderr writer is not logging and is skipped.
- **zerolog by import path** — any call under `github.com/rs/zerolog`; constructors like `zerolog.New` are excluded.
