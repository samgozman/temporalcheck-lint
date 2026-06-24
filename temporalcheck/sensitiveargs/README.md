# sensitiveargs

Flags activity and workflow parameters (and exported fields of struct parameters) whose name matches a configurable regular expression.

Temporal records every argument in **durable workflow history**, persisted for the life of the workflow. Passing a secret or PII value directly means it enters that history. The safer pattern is to pass an opaque reference (an ID, a vault key) and fetch the secret inside the activity.

## Example

```go
workflow.ExecuteActivity(ctx, ChargeCard, cardNumber, cvv)
// Both `cardNumber` and `cvv` parameters match the default pattern
```

```
workflow.go:14  activity "ChargeCard" parameter 2 "cvv" matches the sensitive-data pattern; Temporal records arguments in durable workflow history -- pass an opaque reference and fetch the secret inside the activity instead (sensitive)
```

Struct fields are also inspected:

```go
type PaymentRequest struct {
    CardNumber string
    Amount     int
}
workflow.ExecuteActivity(ctx, Charge, req)
// workflow.go:21  activity "Charge" parameter 1 (type PaymentRequest) field "CardNumber" matches the sensitive-data pattern (sensitive)
```

## Settings

| Key       | Default | Description |
|-----------|---------|-------------|
| `enabled` | `false` | Master switch — analyzer is silent until this is `true` |
| `pattern` | `(?i)cvv\|pan\|card.?number\|password\|secret\|ssn\|token` | Regexp matched (unanchored) against parameter and field names |

This check is **off by default**: naming a parameter `password` is a heuristic, not proof of a sensitive value.

## Limitations

- **Name heuristic only** — a sensitive value with an innocuously named parameter is not caught; tune `pattern` to your conventions.
- **Top level only** — parameter names and the exported fields of struct parameters; nested structs, slices, and maps are not descended.
- **Parameters only** — return values are not checked.
- **Resolvable targets only** — string-registered targets are skipped.
