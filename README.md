# temporalcheck-lint

A [golangci-lint](https://golangci-lint.run) module plugin for static analysis
of [Temporal](https://temporal.io) Go SDK code. Several Temporal SDK APIs take
their arguments as `interface{}`, which erases compile-time checking; this plugin
provides analyzers that recover those checks statically.

## Install

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

## Configuration

golangci-lint exposes the plugin under the name **`temporalcheck`**. Settings are
grouped per analyzer, so each analyzer keeps its own block:

```yaml
settings:
  execargs:
    disabled: false
    strict-types: true
    strict-pointers: true
    strict-struct-shape: true
  stringtarget:
    enabled: true
```

Each analyzer's keys are documented in its section under [Analyzers](#analyzers).

## Suppressing with `//nolint`

Suppression is handled by the plugin itself, so a single call can be exempted
whether you run it under golangci-lint or standalone. Use the plugin name
**`temporalcheck`** (not an analyzer name like `execargs`). A directive suppresses
the call when it is bare (`//nolint`), names `all`, or names `temporalcheck`:

```go
// Suppress one call (anchor the directive anywhere on the call's lines):
_ = workflow.ExecuteActivity(ctx, a.Greet) //nolint:temporalcheck // registered by name elsewhere

// Suppress an entire file: put the directive before the package clause:
//nolint:temporalcheck
package worker
```

A directive that names only other linters (e.g. `//nolint:gocritic`), or an
analyzer name, does not suppress this plugin. To turn an analyzer off across the
whole project, use its `disabled` setting instead.

## Analyzers

Each analyzer is documented below with the same shape: **the problem** it
addresses, **what it checks**, its **settings**, and its **limitations**.

### `execargs`

#### The problem

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

#### What it checks

`execargs` resolves the **real signature** of the referenced activity /
child-workflow function (across files and packages) and checks the call site
against it:

- **Arity** — the number of arguments passed matches the number the target
  expects, after accounting for the framework-injected leading parameter. Always
  on; this is the false-positive-free baseline.
- **Types** — each argument is assignable to the corresponding parameter. Opt-in
  via `strict-types` (off by default).
- **Struct shape** — passing one struct type where a *different* struct type is
  wanted. The `DataConverter` serializes by field name, so distinct structs can
  round-trip while silently dropping or zero-filling mismatched fields — it works
  until a field is renamed or starts to matter. Opt-in via `strict-struct-shape`
  (off by default).

The injected leading parameter is handled per entry point:

| Entry point            | Injected first parameter         | Skipped when matching args |
|------------------------|----------------------------------|----------------------------|
| `ExecuteActivity`      | `context.Context` (**optional**) | only if present            |
| `ExecuteLocalActivity` | `context.Context` (**optional**) | only if present            |
| `ExecuteChildWorkflow` | `workflow.Context` (required)    | always                     |

So for `Greet(ctx context.Context, name string)`, the checker expects exactly
**one** call-site argument (`name`), of type `string`.

##### Example diagnostics

```
workflow.go:14  ExecuteActivity: activity "Greet" expects 1 argument, got 0 (arity)
workflow.go:17  ExecuteActivity: activity "Greet" expects 1 argument, got 2 (arity)
workflow.go:20  ExecuteActivity: arg 1 of "Greet" has type int, want string (strict-types)
workflow.go:23  ExecuteActivity: arg 2 of "ProcessOrder" has type string, want int (strict-types)
workflow.go:30  ExecuteActivity: arg 1 of "Save" has type []*Tier, want []Tier (strict-pointers)
workflow.go:34  ExecuteActivity: arg 1 of "Charge" sends *PayParams, target wants *ChargeParams — serializes by field name but drops {Note} and leaves {Currency} unset (strict-struct-shape)
workflow.go:38  ExecuteChildWorkflow: child workflow "ShipmentWorkflow" expects 1 argument, got 0 (arity)
```

The message names the **entry point** and ends with the **source** in
parentheses — `(arity)`, `(strict-types)`, `(strict-pointers)`, or
`(strict-struct-shape)` — so you can see which check fired and which setting controls it
(golangci-lint then appends the linter name, e.g. `(execargs)`, after that).
`arg N` numbers the arguments **you write at the call site** (after the target),
not the target's parameter positions.

#### Settings

By default the analyzer only checks **arity** (the false-positive-free part).
The three checks below are independent, opt-in layers — enable any combination.

| Key                   | Type | Default | Description                                                                                       |
|-----------------------|------|---------|---------------------------------------------------------------------------------------------------|
| `disabled`            | bool | `false` | Turn the `execargs` analyzer off entirely — it reports nothing regardless of the keys below        |
| `strict-types`        | bool | `false` | Also check argument *types*, not just the argument count                                           |
| `strict-pointers`     | bool | `false` | Flag a value passed where a pointer is expected (and vice versa), including `[]T` vs `[]*T`        |
| `strict-struct-shape` | bool | `false` | Flag passing one struct type where a *different* struct type is wanted                             |

Temporal's default `DataConverter` serializes `T` and `*T` (and `[]T` and
`[]*T`) to the same wire form, so the type check treats them as interchangeable.
Set `strict-pointers: true` to be warned about such mismatches anyway — handy if
you rely on that equivalence and want a heads-up before a `DataConverter` change
could break it.

The converter also serializes structs **by field name**, so passing a distinct
struct type can quietly round-trip — overlapping fields map across, while fields
only on the sender are dropped and fields only on the target are left zero. That
works until a field is renamed or starts to matter. `strict-struct-shape`
surfaces these, naming exactly what drifts; a shared field with an incompatible
type, or no shared fields at all, is reported as a `strict-types` error instead.

The three settings are orthogonal: each can be enabled on its own (e.g.
`strict-struct-shape` without `strict-types`), and every diagnostic is tagged
with the setting that produced it.

#### How it works

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

#### Limitations

- **Type check is stricter than Temporal.** Temporal serializes arguments
  through its `DataConverter`, so the wire-level contract is looser than Go
  assignability (e.g. `int` vs `int32` may round-trip fine via JSON yet be
  flagged here). Value-vs-pointer mismatches are allowed even under
  `strict-types` (see `strict-pointers`), but other looseness is not modeled.
  The **arity** check is the false-positive-free part and runs by default; the
  type check is opt-in via `strict-types`.
- **String-registered targets are skipped.** `ExecuteActivity(ctx, "MyActivity", ...)`
  can't be resolved to a signature statically, so it's ignored here — the opt-in
  [`stringtarget`](#stringtarget) analyzer flags those call sites instead, so you
  can refactor them to a function reference that `execargs` *can* check.
- **Spread calls are skipped.** `ExecuteActivity(ctx, fn, slice...)` can't be
  matched positionally.
- **Method expressions and dot-imports are out of scope.**
  `Activities.Greet` (receiver as first param) and a dot-imported
  `ExecuteActivity` aren't recognized. Neither is idiomatic Temporal usage.
- Variadic activities get a basic check (fixed prefix + element type).
- **`strict-struct-shape` models JSON, and only one level.** It matches exported
  fields by their `json` tag name (the default `DataConverter`); a custom
  converter with different semantics isn't modeled. Embedded-field promotion is
  not followed, and slices/maps of distinct structs fall back to `strict-types`
  rather than a per-field drift report.

### `stringtarget`

#### The problem

Temporal lets you launch an activity or child workflow by its registered
**string name** instead of by a reference to the Go function:

```go
workflow.ExecuteActivity(ctx, "MyActivity", arg1, arg2)
```

That string is opaque. It can't be resolved to a signature at compile time, so:

- the number and types of the trailing arguments go **unchecked**, and a typo in
  the name fails only at run time; and
- it **blinds `execargs`**, which silently skips any call whose target isn't a
  resolvable function value.

Passing the function reference instead — `workflow.ExecuteActivity(ctx, a.MyActivity, …)`
— is better on its own terms (the name is derived from the function rather than
duplicated as a fragile string), and it's exactly what lets the rest of this
plugin verify the call.

#### What it checks

`stringtarget` flags every `ExecuteActivity` / `ExecuteLocalActivity` /
`ExecuteChildWorkflow` call whose target argument is a string — a literal, a
string variable, or a named string type — so you can refactor it to a function
reference. A call that already passes a function reference is never flagged.

This check is **off by default**: naming a target by string is a legitimate,
sometimes necessary pattern (for example an activity implemented in another
service or language), so flagging it is opt-in.

##### Example diagnostics

```
workflow.go:14  ExecuteActivity: target "MyActivity" is named by string; pass the function reference instead so its arguments can be checked statically (string-target)
workflow.go:18  ExecuteChildWorkflow: the target is named by string; pass the function reference instead so its arguments can be checked statically (string-target)
```

The message names the **entry point** and ends with the source `(string-target)`
(golangci-lint then appends the linter name, `(stringtarget)`, after that). When
the target is a string literal the diagnostic quotes its value; for a string
variable it falls back to a generic subject.

#### Settings

| Key       | Type | Default | Description                                                                          |
|-----------|------|---------|--------------------------------------------------------------------------------------|
| `enabled` | bool | `false` | Turn the `stringtarget` analyzer on — it reports nothing unless explicitly enabled    |

#### Limitations

- **It does not resolve the named target.** This analyzer only reports *that* a
  target is named by string; it can't map the name to a signature (see the
  `execargs` limitation above for why that's not statically possible across
  packages). Its purpose is to steer you toward the form `execargs` can check.

## Development

```bash
make test          # go test -race -v ./...
make cover-check   # tests + 90% coverage gate
make vet
make conformance   # build ./conformance against the real Temporal SDK
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for how the plugin and its fixtures are
organised.

### The stub and the conformance check

The analyzers never import the Temporal SDK — they match calls by package path
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
│   ├── execargs/
│   │   ├── execargs.go           # settings, analyzer, call dispatch
│   │   ├── check.go              # signature matching + helpers
│   │   ├── nolint.go             # //nolint directive suppression
│   │   ├── execargs_test.go      # analysistest
│   │   ├── execargs_internal_test.go
│   │   ├── nolint_internal_test.go
│   │   └── testdata/                  # self-contained fixture module (see below)
│   │       ├── go.mod                 # replace go.temporal.io/sdk => ./temporalsdk
│   │       ├── good/ bad/ notypes/    # fixture packages (// want assertions)
│   │       └── temporalsdk/           # local stub module for the Temporal SDK
│   └── stringtarget/             # opt-in: flag string-named Execute* targets
│       ├── stringtarget.go       # settings, analyzer, call dispatch
│       ├── nolint.go             # //nolint directive suppression
│       ├── stringtarget_test.go  # analysistest
│       ├── stringtarget_internal_test.go
│       ├── nolint_internal_test.go
│       └── testdata/             # self-contained fixture module
├── conformance/                  # CI-only module: real-SDK contract test (see below)
├── .custom-gcl.yml               # custom golangci-lint build config
├── .golangci.yml                 # example consumer config (also self-lints this repo)
├── Makefile
└── go.mod
```

Each analyzer lives in its own package under `temporalcheck/` with the same
internal layout as `execargs/` (analyzer + dispatch, matching logic, `//nolint`
handling, and a `testdata/` fixture module) and is registered in
`BuildAnalyzers` in `plugin.go`.

## License

MIT. See [LICENSE](LICENSE).
