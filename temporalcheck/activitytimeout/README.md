# activitytimeout

Flags `workflow.ActivityOptions` and `workflow.LocalActivityOptions` composite literals that set fields but no required timeout.

Temporal requires at least one of `StartToCloseTimeout` or `ScheduleToCloseTimeout`; omitting both causes the activity to be rejected at run time.

## Example

```go
// Missing required timeout — flagged
ao := workflow.ActivityOptions{TaskQueue: "greetings"}
ctx = workflow.WithActivityOptions(ctx, ao)
workflow.ExecuteActivity(ctx, a.Greet) // fails at run time

// Correct
ao := workflow.ActivityOptions{
    TaskQueue:           "greetings",
    StartToCloseTimeout: time.Minute,
}
```

```
workflow.go:14  ActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time (required-timeout)
```

## Settings

| Key                      | Default | Description |
|--------------------------|---------|-------------|
| `disabled`               | `false` | Turn the analyzer off entirely |
| `require-start-to-close` | `false` | Also flag literals that set only `ScheduleToCloseTimeout` without `StartToCloseTimeout` |

The `require-start-to-close` opt-in is for teams that want each attempt bounded, not just the whole schedule window.

```
workflow.go:22  ActivityOptions sets ScheduleToCloseTimeout but not StartToCloseTimeout: bound each attempt with StartToCloseTimeout (require-start-to-close)
```

## Limitations

- **Presence, not value** — `StartToCloseTimeout: 0` is not flagged even though Temporal rejects it; values aren't evaluated statically.
- **Empty literals are skipped** — `workflow.ActivityOptions{}` is commonly populated field-by-field afterwards; this literal-only check can't see that.
- **Positional literals are skipped** — elements without field names can't be mapped to fields.
- **Only composite literals** — options built via `var ao workflow.ActivityOptions; ao.StartToCloseTimeout = ...` are out of scope.
