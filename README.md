# temporalcheck-lint (PoC)

A [golangci-lint](https://golangci-lint.run) module plugin for static analysis
of [Temporal](https://temporal.io) Go SDK code. This is a **proof of concept**
with a single analyzer (`execargs`); it's laid out so more Temporal checks can
be added under the same plugin later.

## The problem

`workflow.ExecuteActivity`, `workflow.ExecuteLocalActivity` and
`workflow.ExecuteChildWorkflow` take the target as `interface{}` and its
arguments as `...interface{}`:

```go
func ExecuteActivity(ctx Context, activity interface{}, args ...interface{}) Future
```

So the compiler can't help you. Given this activity:

```go
// Greet(ctx context.Context, name string) (string, error)
```

all of these build fine and only blow up at run time:

```go
var a *Activities

workflow.ExecuteActivity(ctx, a.Greet)               // missing name
workflow.ExecuteActivity(ctx, a.Greet, 42)           // name is a string, not int
workflow.ExecuteActivity(ctx, a.Greet, "x", "extra") // one argument too many
```

## What this checks

The `execargs` analyzer resolves the **real signature** of the referenced
activity / child-workflow function (across files and packages) and checks the
call site against it:

- **Arity** — the number of arguments passed matches the number the target
  expects, after accounting for the framework-injected leading parameter. Always
  on; this is the false-positive-free baseline.
- **Types** — each argument is assignable to the corresponding parameter. Opt-in
  via `strict-types` (off by default).

The injected leading parameter is handled per entry point:

| Entry point            | Injected first parameter         | Skipped when matching args |
|------------------------|----------------------------------|----------------------------|
| `ExecuteActivity`      | `context.Context` (**optional**) | only if present            |
| `ExecuteLocalActivity` | `context.Context` (**optional**) | only if present            |
| `ExecuteChildWorkflow` | `workflow.Context` (required)    | always                     |

So for `Greet(ctx context.Context, name string)`, the checker expects exactly
**one** call-site argument (`name`), of type `string`.

### Example diagnostics

```
workflow.go:14  ExecuteActivity: activity "Greet" expects 1 argument, got 0
workflow.go:17  ExecuteActivity: activity "Greet" expects 1 argument, got 2
workflow.go:20  ExecuteActivity: arg 1 of "Greet" has type int, want string
workflow.go:23  ExecuteActivity: arg 2 of "ProcessOrder" has type string, want int
workflow.go:38  ExecuteChildWorkflow: child workflow "ShipmentWorkflow" expects 1 argument, got 0
```

The message names the **entry point** (golangci-lint already appends the linter
name, e.g. `(execargs)`). `arg N` numbers the arguments **you write at the call
site** (after the target), not the target's parameter positions.

## Use it as a golangci-lint plugin

Module plugins are compiled into golangci-lint itself; see the
[Module Plugin System docs](https://golangci-lint.run/docs/plugins/module-plugins/).

1. Add `.custom-gcl.yml` to the project you want to lint (an example is in this
   repo). For local development against this unpublished checkout, use `path:`
   instead of `version:`:

   ```yaml
   version: v2.12.2
   name: custom-gcl
   destination: ./bin
   plugins:
     - module: github.com/samgozman/temporalcheck-lint
       import: github.com/samgozman/temporalcheck-lint/temporalcheck
       path: /absolute/path/to/temporalcheck-lint
   ```

2. Build the custom binary (produces `./bin/custom-gcl`):

   ```bash
   golangci-lint custom   # or: make build
   ```

3. Enable it in the target project's `.golangci.yml` (example in this repo),
   then run:

   ```bash
   ./bin/custom-gcl run   # or: make run
   ```

## Settings

Settings are grouped per analyzer, so each analyzer keeps its own block as the
linter grows:

```yaml
settings:
  execargs:
    strict-types: true
    strict-pointers: false
```

### `execargs`

By default the analyzer only checks **arity** (the false-positive-free part);
the stricter checks are opt-in.

| Key               | Type | Default | Description                                                                                       |
|-------------------|------|---------|---------------------------------------------------------------------------------------------------|
| `strict-types`    | bool | `false` | Also check argument *types*, not just the argument count                                           |
| `strict-pointers` | bool | `false` | Flag a value passed where a pointer is expected (and vice versa), including `[]T` vs `[]*T`        |

Temporal's default `DataConverter` serializes `T` and `*T` (and `[]T` and
`[]*T`) to the same wire form, so even with `strict-types` on the type check
treats them as interchangeable. Set `strict-pointers: true` to be warned about
such mismatches anyway — handy if you rely on that equivalence and want a
heads-up before a `DataConverter` change could break it. It has no effect while
`strict-types` is off.

## How it works

`GetLoadMode()` returns `LoadModeTypesInfo`, so the pass has full type
information. For each call the analyzer:

1. Confirms the callee is a function in `go.temporal.io/sdk/workflow` named
   `ExecuteActivity` / `ExecuteLocalActivity` / `ExecuteChildWorkflow`
   (resolved via `pass.TypesInfo.Uses`, so aliased imports still match).
2. Reads the target argument's type. A **method value** like `a.Greet`
   resolves to a `*types.Signature` with the receiver already stripped, which
   is exactly what we want.
3. Computes how many leading parameters Temporal injects (table above) and
   compares the remaining parameters against the call-site arguments using
   `types.AssignableTo`.

## Known limitations (it's a PoC)

- **Type check is stricter than Temporal.** Temporal serializes arguments
  through its `DataConverter`, so the wire-level contract is looser than Go
  assignability (e.g. `int` vs `int32` may round-trip fine via JSON yet be
  flagged here). Value-vs-pointer mismatches are allowed even under
  `strict-types` (see `strict-pointers`), but other looseness is not modeled.
  The **arity** check is the false-positive-free part and runs by default; the
  type check is opt-in via `strict-types`.
- **String-registered targets are skipped.** `ExecuteActivity(ctx, "MyActivity", ...)`
  can't be resolved to a signature statically, so it's ignored.
- **Spread calls are skipped.** `ExecuteActivity(ctx, fn, slice...)` can't be
  matched positionally.
- **Method expressions and dot-imports are out of scope.**
  `Activities.Greet` (receiver as first param) and a dot-imported
  `ExecuteActivity` aren't recognized. Neither is idiomatic Temporal usage.
- Variadic activities get a basic check (fixed prefix + element type).

## Development

```bash
make test          # go test -race -v ./...
make cover-check   # tests + 90% coverage gate
make vet
make conformance   # build ./conformance against the real Temporal SDK
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for how the linter and its fixtures are
organised.

### The stub and the conformance check

The analyzer never imports the Temporal SDK — it matches calls by package path
through `go/types`. The SDK only appears in the **test fixtures**, which use a
tiny local stub (`testdata/temporalsdk/`) so the tests stay hermetic and
offline.

To make sure that stub doesn't silently drift from the real SDK, the separate
[`conformance/`](conformance) module imports the **real**
`go.temporal.io/sdk/workflow` and asserts — at compile time — that each
`Execute*` function still has the `(ctx, target, args...)` shape the analyzer
relies on. CI builds it against the latest SDK and Dependabot bumps the version,
so a breaking SDK change shows up as a failed `make conformance` on the bump PR.
It's its own module so the SDK's large dependency tree never touches the main
module or the fixtures.

## Layout

```
temporalcheck-lint/
├── temporalcheck/
│   ├── plugin.go                 # register.Plugin("temporalcheck", ...)
│   ├── plugin_internal_test.go
│   └── execargs/
│       ├── execargs.go           # settings, analyzer, call dispatch
│       ├── check.go              # signature matching + helpers
│       ├── execargs_test.go      # analysistest
│       ├── execargs_internal_test.go
│       └── testdata/                  # self-contained fixture module (see below)
│           ├── go.mod                 # replace go.temporal.io/sdk => ./temporalsdk
│           ├── good/ bad/ notypes/    # fixture packages (// want assertions)
│           └── temporalsdk/           # local stub module for the Temporal SDK
├── conformance/                  # CI-only module: real-SDK contract test (see below)
├── .custom-gcl.yml               # custom golangci-lint build config
├── .golangci.yml                 # example consumer config (also self-lints this repo)
├── Makefile
└── go.mod
```

Adding the next Temporal check means dropping a new analyzer package next to
`execargs/` and appending it to `BuildAnalyzers` in `plugin.go`.

## License

MIT. See [LICENSE](LICENSE).
