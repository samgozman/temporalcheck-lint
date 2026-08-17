# searchattribute

Flags a Temporal **workflow entrypoint** whose params struct carries a configured field but whose body never upserts the mapped **search attribute**.

Search attributes are how a workflow is made findable in the Temporal UI and via `ListWorkflowExecutions`. A common convention is: "every workflow started for a user records that user's id as a `user_id` search attribute." This check enforces *your* convention — you declare which params field implies which attribute, and it flags workflows that take the field but forget to upsert the attribute.

The mapping is project-specific, so the analyzer is **off by default** and does nothing until you configure `attributes`.

## Example

With settings:

```yaml
searchattribute:
  enabled: true
  attributes:
    user_id: user_id      # a UserID field must upsert the "user_id" attribute
    client_id: user_id    # alias: ClientID also satisfies "user_id"
```

```go
type Params struct {
    UserID  string
    OrderID string
}

func ProcessOrder(ctx workflow.Context, p Params) error {
    // never calls workflow.UpsertTypedSearchAttributes / UpsertSearchAttributes
    return nil
}
```

```
workflow.go:9  workflow "ProcessOrder" takes field "UserID" but never upserts the "user_id" search attribute (searchattribute)
```

A workflow that upserts the attribute — via the typed or the legacy API, with the attribute name as a string literal — is not flagged:

```go
func ProcessOrder(ctx workflow.Context, p Params) error {
    return workflow.UpsertTypedSearchAttributes(ctx,
        temporal.NewSearchAttributeKeyKeyword("user_id").ValueSet(p.UserID))
}
```

## Settings

| Key          | Default | Description |
|--------------|---------|-------------|
| `enabled`    | `false` | Master switch — analyzer is silent until this is `true` |
| `attributes` | `{}`    | Map of field-name alias → required search-attribute name. With no entries the analyzer reports nothing. |

**Field matching is case- and separator-insensitive.** The keys `user_id`, `x-user-id`, and `UserID` all match a Go field named `UserID`, so you can write the key in whatever form reads best. Several aliases may map to one attribute name (the "aliases" case): both `client_id` and `user_id` mapping to `user_id` means either field satisfies the requirement.

## Limitations

- **Exported workflow functions only** — a candidate entrypoint is an *exported* top-level function (or method) whose first parameter is `workflow.Context`. Registered/executed workflows are exported (their worker registration lives in another package). Unexported `workflow.Context` functions are treated as workflow *helpers* — written to split up a workflow's logic and called directly by name — and are not checked, which avoids flagging every helper. Nested closures and coroutines (`workflow.Go`) are not entrypoints either.
- **Test files are skipped** — workflows defined in `_test.go` files are test-harness workflows exercised with the testsuite, not production entrypoints, so they are not checked.
- **Top-level struct fields only** — exported fields of a struct parameter, dereferencing one pointer level; nested structs, slices, and maps are not descended.
- **Attribute names must be string literals** — the check reads the attribute name from a `NewSearchAttributeKey*` argument (typed API) or a map key (legacy API). A workflow that upserts through a source we can't read statically — a prebuilt map, a spread slice (`updates...`), or a key held in a variable — is **left alone entirely**, to avoid false positives.
- **Heuristic** — it enforces a naming convention you supply, not a correctness property; it can't know a workflow legitimately shouldn't record a field.
