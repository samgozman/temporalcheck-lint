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
    strict-tests: true
  stringtarget:
    enabled: true
    strict-tests: true
  optionsdiscard:
    disabled: false
  activitytimeout:
    disabled: false
  futureget:
    disabled: false
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

`workflow.ExecuteActivity`, `workflow.ExecuteLocalActivity`,
`workflow.ExecuteChildWorkflow`, `workflow.NewContinueAsNewError` and the
`client.Client` methods `ExecuteWorkflow` and `SignalWithStartWorkflow` take the
target as `interface{}` and its arguments as `...interface{}`:

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
workflow function (across files and packages) and checks the call site
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
- **Test mocks** — the matcher arity of `testsuite` mock setups,
  `(*testsuite.TestWorkflowEnvironment).OnActivity` and `.OnWorkflow`. Opt-in via
  `strict-tests` (off by default); see [Settings](#settings) below for the
  context-counting difference and why only arity is checked.

The target sits at a different argument position per entry point — second for the
`workflow.*` calls, third for `ExecuteWorkflow` (after the options), sixth for
`SignalWithStartWorkflow` (after the signal fields and options) — and the
`client.Client` methods are matched by receiver. Arguments are matched against the
target's parameters after the position of the target reference.

The injected leading parameter of the target is handled per entry point:

| Entry point               | Injected first parameter         | Skipped when matching args |
|---------------------------|----------------------------------|----------------------------|
| `ExecuteActivity`         | `context.Context` (**optional**) | only if present            |
| `ExecuteLocalActivity`    | `context.Context` (**optional**) | only if present            |
| `ExecuteChildWorkflow`    | `workflow.Context` (required)    | always                     |
| `NewContinueAsNewError`   | `workflow.Context` (required)    | always                     |
| `ExecuteWorkflow`         | `workflow.Context` (required)    | always                     |
| `SignalWithStartWorkflow` | `workflow.Context` (required)    | always                     |

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
parentheses — `(arity)`, `(strict-types)`, `(strict-pointers)`,
`(strict-struct-shape)`, or `(strict-tests)` — so you can see which check fired and which setting controls it
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
| `strict-tests`        | bool | `false` | Also check the matcher arity of `testsuite` `OnActivity`/`OnWorkflow` mock setups                  |

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

`strict-tests` extends the **arity** check to Temporal's `testsuite` mock setups
— `(*testsuite.TestWorkflowEnvironment).OnActivity` and `.OnWorkflow`. These take
the target as `interface{}` and the matchers as `...interface{}`, the same type
erasure `Execute*` suffers. One difference matters: the matchers must cover
**every** declared parameter, **including** the injected `context.Context` /
`workflow.Context` (you pass `mock.Anything` for it) — so the expected count is
the target's full parameter count, with nothing skipped. Only arity is checked,
because the matchers (`mock.Anything`, `mock.MatchedBy`) are opaque and never the
real typed value. String-named, spread (`matchers...`), and variadic targets are
skipped. Diagnostics are tagged `(strict-tests)`, e.g.:

```
workflow_test.go:42  OnActivity: mock for activity "Greet" expects 2 arguments (one per parameter), got 1 (strict-tests)
```

The settings are orthogonal: each can be enabled on its own (e.g.
`strict-struct-shape` without `strict-types`, or `strict-tests` on its own), and
every diagnostic is tagged with the setting that produced it.

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
`ExecuteChildWorkflow` / `NewContinueAsNewError` and `client` `ExecuteWorkflow` /
`SignalWithStartWorkflow` call whose target argument is a string — a literal, a
string variable, or a named string type — so you can refactor it to a function
reference. A call that already passes a function reference is never flagged.

This check is **off by default**: naming a target by string is a legitimate,
sometimes necessary pattern (for example an activity implemented in another
service or language), so flagging it is opt-in.

With `strict-tests` on, the same string-target check also runs over Temporal's
`testsuite` mock setups — `(*testsuite.TestWorkflowEnvironment).OnActivity` and
`.OnWorkflow` — whose target is named by string. It is an opt-in layer **gated by
`enabled`**: `enabled` is the master switch, so with it off the analyzer reports
nothing regardless of `strict-tests`. Those diagnostics are tagged
`(strict-tests)` rather than `(string-target)`.

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

| Key            | Type | Default | Description                                                                                       |
|----------------|------|---------|---------------------------------------------------------------------------------------------------|
| `enabled`      | bool | `false` | Master switch — flag string-named targets in production `Execute*` calls; with this off the analyzer is silent |
| `strict-tests` | bool | `false` | Also flag string-named targets in `testsuite` `OnActivity`/`OnWorkflow` mock setups (gated by `enabled`) |

#### Limitations

- **It does not resolve the named target.** This analyzer only reports *that* a
  target is named by string; it can't map the name to a signature (see the
  `execargs` limitation above for why that's not statically possible across
  packages). Its purpose is to steer you toward the form `execargs` can check.

### `optionsdiscard`

#### The problem

`workflow.WithActivityOptions`, `workflow.WithLocalActivityOptions` and
`workflow.WithChildOptions` do **not** mutate the context you give them — each
returns a *new* context that carries the options:

```go
func WithActivityOptions(ctx Context, options ActivityOptions) Context
```

The classic mistake is to forget the `ctx =`:

```go
workflow.WithActivityOptions(ctx, ao)  // result thrown away
workflow.ExecuteActivity(ctx, a.Greet) // runs with the OLD ctx — no options set
```

This compiles cleanly, but the options silently never apply and the activity
blows up at run time with a missing-`StartToCloseTimeout` error.

#### What it checks

`optionsdiscard` flags any `WithActivityOptions` / `WithLocalActivityOptions` /
`WithChildOptions` call whose returned context is **discarded** — used as a bare
expression statement, or assigned to the blank identifier `_`. The fix is to
assign the result back:

```go
ctx = workflow.WithActivityOptions(ctx, ao) // correct — never flagged
```

The check is **on by default**. It is pure AST + types with near-zero false
positives (errcheck-style): discarding a `With*Options` result is always a bug,
never a deliberate pattern, so there is nothing to opt into — only a `disabled`
switch to turn it off.

##### Example diagnostics

```
workflow.go:14  WithActivityOptions: the returned context is discarded, so the options never apply; assign it back with ctx = workflow.WithActivityOptions(ctx, opts) (options-discard)
```

The message names the **entry point** and ends with the source `(options-discard)`
(golangci-lint then appends the linter name, `(optionsdiscard)`, after that).

#### Settings

| Key        | Type | Default | Description                                       |
|------------|------|---------|---------------------------------------------------|
| `disabled` | bool | `false` | Turn the `optionsdiscard` analyzer off entirely    |

#### Limitations

- **Only the discard forms are flagged.** A bare call statement and `_ =` are
  caught; a result assigned to a *named* variable is assumed kept, even if that
  variable is never used afterwards. Tracking later use is out of scope — it would
  trade the near-zero false-positive rate for marginal extra coverage.

### `activitytimeout`

#### The problem

Every Temporal activity needs a timeout. The SDK requires at least one of
`StartToCloseTimeout` or `ScheduleToCloseTimeout` on both `ActivityOptions` and
`LocalActivityOptions`; configure an activity with neither and Temporal rejects it
at run time. The options are an ordinary struct literal, so the compiler is happy
to let you omit them:

```go
ao := workflow.ActivityOptions{TaskQueue: "greetings"} // no timeout set
ctx = workflow.WithActivityOptions(ctx, ao)
workflow.ExecuteActivity(ctx, a.Greet)                 // fails at run time
```

#### What it checks

`activitytimeout` inspects `workflow.ActivityOptions` and
`workflow.LocalActivityOptions` composite literals and flags any that set fields
but neither required timeout. The fix is to set one:

```go
ao := workflow.ActivityOptions{
	TaskQueue:           "greetings",
	StartToCloseTimeout: time.Minute, // correct — never flagged
}
```

The check is **on by default**. It is pure AST + types with near-zero false
positives: an activity without a required timeout is always rejected at run time,
never a deliberate pattern, so there is nothing to opt into — only a `disabled`
switch to turn it off.

##### Example diagnostics

```
workflow.go:14  ActivityOptions sets no required timeout: set StartToCloseTimeout or ScheduleToCloseTimeout, or the activity is rejected at run time (required-timeout)
```

The message names the **options type** and ends with the source
`(required-timeout)` (golangci-lint then appends the linter name,
`(activitytimeout)`, after that).

#### Settings

| Key        | Type | Default | Description                                        |
|------------|------|---------|----------------------------------------------------|
| `disabled` | bool | `false` | Turn the `activitytimeout` analyzer off entirely    |

#### Limitations

- **Presence, not value.** A required timeout *key* in the literal satisfies the
  check; its value isn't evaluated. An explicit `StartToCloseTimeout: 0` (also
  rejected by Temporal) is **not** flagged, since the value is often a variable or
  expression the analyzer can't resolve statically.
- **Empty literals are skipped.** `workflow.ActivityOptions{}` is left alone — it
  is commonly populated field-by-field afterwards (`ao.StartToCloseTimeout = …`),
  which this literal-only inspection can't see. Flagging it would be a false
  positive.
- **Positional literals are skipped.** A literal without field names can't be
  mapped to fields without the struct layout (and `go vet` already flags unkeyed
  imported-struct literals).
- **Only composite literals are inspected.** Options built some other way — e.g.
  `var ao workflow.ActivityOptions` then field assignments, or a value returned
  from a helper — are out of scope.

### `futureget`

#### The problem

`workflow.Future`, `workflow.ChildWorkflowFuture` and `converter.EncodedValue`
all surface a result through a `.Get` that returns an `error`:

```go
func (Future) Get(ctx workflow.Context, valuePtr interface{}) error
```

That error reports a failed activity, a failed child workflow, or a decode error.
Dropping it silently swallows the failure — the workflow carries on as if the
call succeeded:

```go
future := workflow.ExecuteActivity(ctx, a.Publish, batch)
_ = future.Get(ctx, nil) // activity error swallowed
```

#### What it checks

`futureget` flags a `.Get` call on one of those three receiver types whose
returned error is **discarded** — used as a bare expression statement, or
assigned to `_`. The fix is to check it:

```go
if err := future.Get(ctx, &result); err != nil {
	return err // correct — never flagged
}
```

The check is **on by default**. It is errcheck scoped to Temporal's result
types: pure AST + types with near-zero false positives. By construction it cannot
fire on fire-and-forget, which never calls `.Get` — exactly the case a generic
errcheck would wrongly flag.

##### Example diagnostics

```
workflow.go:21  Get: the returned error from Future.Get is discarded; check it or assign it to a variable you inspect (future-get)
```

The message names the **receiver type** and ends with the source `(future-get)`
(golangci-lint then appends the linter name, `(futureget)`, after that).

#### Settings

| Key        | Type | Default | Description                                   |
|------------|------|---------|-----------------------------------------------|
| `disabled` | bool | `false` | Turn the `futureget` analyzer off entirely     |

#### Limitations

- **Syntactic discard only.** Only a bare statement and `_ =` are flagged. An
  error assigned to a real variable that is then never checked (`err := f.Get(…)`
  with no following use) is **not** flagged — that needs data-flow analysis and
  would risk false positives.
- **Static receiver type.** Matching is on the receiver's declared type, so a
  user type that merely embeds `workflow.Future` and discards a promoted `.Get`
  is conservatively skipped.
- **`.Get` only.** Sibling methods (`IsReady`, `GetChildWorkflowExecution`,
  `SignalChildWorkflow`) don't return a must-check error and are out of scope.
  `converter.EncodedValues` (plural) is also out of scope.

### `lossynumber`

#### The problem

Temporal serializes activity and workflow arguments and results through its
`DataConverter`, whose default is JSON. Go's `encoding/json` decodes **every JSON
number into a `float64`** when the destination type is the empty interface, and a
`float64` cannot represent integers past 2^53 exactly. So a number carried through
a dynamically-typed parameter or return — `interface{}`/`any`, `map[string]any`,
`[]any` — round-trips with silent precision loss:

```go
// Activity parameter typed `any`: the worker decodes the argument into interface{}.
func Charge(ctx context.Context, amount any) error { /* ... */ }

var amount int64 = 9007199254740993 // 2^53 + 1
workflow.ExecuteActivity(ctx, Charge, amount)
// Inside Charge, amount is float64(9007199254740992) — off by one, no error.
```

#### What it checks

`lossynumber` resolves the function referenced by each `workflow.ExecuteActivity`,
`workflow.ExecuteLocalActivity`, `workflow.ExecuteChildWorkflow`,
`workflow.NewContinueAsNewError`, `client.ExecuteWorkflow` and
`client.SignalWithStartWorkflow` call to its real signature, then flags any **top-level**
parameter or **non-error return** whose type is one of those lossy dynamic forms.
The framework-injected leading context (`context.Context` for activities,
`workflow.Context` for workflows) and a trailing `error` are skipped. The fix is a
concrete type:

```go
func Charge(ctx context.Context, amount int64) error { /* ... */ } // never flagged
```

The check is **on by default** and pure AST + types. It is deliberately shallow to
stay false-positive-free:

- A named empty interface (`type Payload interface{}`) counts; a **non-empty**
  interface (`error`, `io.Reader`, any interface with methods) does not.
- A struct that merely **contains** an `any` field is **not** flagged — only the
  top-level parameter/return type, or the element of a top-level `map`/slice, is
  examined.
- A string-registered target resolves to no signature and is skipped.

##### Example diagnostics

```
workflow.go:21  activity "Charge" parameter 1 has dynamic type any; Temporal's JSON converter decodes numbers as float64 and silently loses int64 precision past 2^53 — use a concrete type (lossy-types)
```

The message names the **target** and the offending parameter/return and ends with
the source `(lossy-types)` (golangci-lint then appends the linter name,
`(lossynumber)`, after that).

#### Settings

| Key        | Type | Default | Description                                      |
|------------|------|---------|--------------------------------------------------|
| `disabled` | bool | `false` | Turn the `lossynumber` analyzer off entirely      |

Disable it only for the rare case of a custom `DataConverter` that preserves
integer precision (e.g. one that decodes numbers into `json.Number` or `int64`).

#### Limitations

- **Top-level only.** Lossy types nested inside a struct field, or below the first
  level of a `map`/slice (e.g. `[][]any`), are not flagged — that would risk false
  positives on types that never actually carry a number.
- **Resolvable targets only.** A target registered and executed by its string name
  has no static signature, so it is skipped (the [`stringtarget`](#stringtarget)
  analyzer addresses string targets directly).
- **Execution sites.** The signature is inspected wherever the function is passed
  to an `Execute*`/`client.ExecuteWorkflow` call; an activity that is registered
  but never executed in the analyzed code is not reached.

### `continueasnew`

#### The problem

A workflow continues as new by **returning** the error built by
`workflow.NewContinueAsNewError`:

```go
func NewContinueAsNewError(ctx workflow.Context, wfn interface{}, args ...interface{}) error
```

The returned error *is* the signal: returning it ends the current run and starts
a fresh one with the given arguments. Constructing it without returning it drops
the signal silently — the workflow falls through and just **ends** instead of
continuing:

```go
if shouldContinue {
	workflow.NewContinueAsNewError(ctx, MyWorkflow, next) // built, never returned
}
return nil // workflow ends here — the continue-as-new never happens
```

#### What it checks

`continueasnew` flags a `workflow.NewContinueAsNewError` call whose result is
**discarded** — used as a bare expression statement, or assigned to `_`. The fix
is to return it:

```go
if shouldContinue {
	return workflow.NewContinueAsNewError(ctx, MyWorkflow, next) // correct — never flagged
}
```

The check is **on by default**. It is pure AST + types with near-zero false
positives: returning the error is the only way the call has any effect, so a
discarded result is always a bug — there is nothing to opt into, only a
`disabled` switch.

##### Example diagnostics

```
workflow.go:14  NewContinueAsNewError: the continue-as-new error is discarded; return it so the workflow continues as new, otherwise the workflow silently ends instead (continue-as-new)
```

The message ends with the source `(continue-as-new)` (golangci-lint then appends
the linter name, `(continueasnew)`, after that).

#### Settings

| Key        | Type | Default | Description                                       |
|------------|------|---------|---------------------------------------------------|
| `disabled` | bool | `false` | Turn the `continueasnew` analyzer off entirely     |

#### Limitations

- **Syntactic discard only.** Only a bare statement and `_ =` are flagged. A
  result assigned to a real variable (`err := workflow.NewContinueAsNewError(…)`)
  is **not** flagged — a `return err` may follow, and proving it never does needs
  data-flow analysis that would risk false positives.
- **Constructed-and-returned shapes only.** The analyzer reasons about the
  `NewContinueAsNewError` call site itself; it does not track a continue-as-new
  error passed into a helper and returned from there.

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
`go.temporal.io/sdk/workflow`, `go.temporal.io/sdk/converter` and
`go.temporal.io/sdk/testsuite` and asserts — at compile time — that each
`Execute*` function still has the `(ctx, target, args...)` shape, that
`TestWorkflowEnvironment.OnActivity`/`.OnWorkflow` still have the `(target,
matchers...)` shape, that each `With*Options` function still returns a new
`Context`, and that `Future`/`ChildWorkflowFuture`/`EncodedValue` still expose a
`.Get` returning `error` — the shapes the analyzers rely on. CI builds it
against the latest SDK and Dependabot bumps the version,
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
│   ├── stringtarget/             # opt-in: flag string-named Execute* targets
│   │   ├── stringtarget.go       # settings, analyzer, call dispatch
│   │   ├── nolint.go             # //nolint directive suppression
│   │   ├── stringtarget_test.go  # analysistest
│   │   ├── stringtarget_internal_test.go
│   │   ├── nolint_internal_test.go
│   │   └── testdata/             # self-contained fixture module
│   ├── optionsdiscard/           # flag discarded With*Options results
│   │   ├── optionsdiscard.go     # settings, analyzer, discard dispatch
│   │   ├── nolint.go             # //nolint directive suppression
│   │   ├── optionsdiscard_test.go  # analysistest
│   │   ├── optionsdiscard_internal_test.go
│   │   ├── nolint_internal_test.go
│   │   └── testdata/             # self-contained fixture module
│   ├── activitytimeout/          # flag Activity options missing a required timeout
│   │   ├── activitytimeout.go    # settings, analyzer, literal dispatch
│   │   ├── check.go              # option-type + field matching helpers
│   │   ├── nolint.go             # //nolint directive suppression
│   │   ├── activitytimeout_test.go  # analysistest
│   │   ├── activitytimeout_internal_test.go
│   │   ├── nolint_internal_test.go
│   │   └── testdata/             # self-contained fixture module
│   ├── futureget/                # flag discarded Future/EncodedValue .Get errors
│   │   ├── futureget.go          # settings, analyzer, discard dispatch
│   │   ├── check.go              # receiver-type matching helpers
│   │   ├── nolint.go             # //nolint directive suppression
│   │   ├── futureget_test.go     # analysistest
│   │   ├── futureget_internal_test.go
│   │   ├── nolint_internal_test.go
│   │   └── testdata/             # self-contained fixture module
│   ├── lossynumber/              # flag dynamic-typed (any) activity/workflow params/returns
│   │   ├── lossynumber.go        # settings, analyzer, call dispatch
│   │   ├── check.go              # signature inspection + lossy-type predicate
│   │   ├── nolint.go             # //nolint directive suppression
│   │   ├── lossynumber_test.go   # analysistest
│   │   ├── lossynumber_internal_test.go
│   │   ├── nolint_internal_test.go
│   │   └── testdata/             # self-contained fixture module
│   └── continueasnew/            # flag discarded (not-returned) NewContinueAsNewError results
│       ├── continueasnew.go      # settings, analyzer, discard dispatch
│       ├── check.go              # NewContinueAsNewError call matching
│       ├── nolint.go             # //nolint directive suppression
│       ├── continueasnew_test.go # analysistest
│       ├── continueasnew_internal_test.go
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
