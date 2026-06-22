# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

Initial proof of concept.

- `execargs` analyzer: checks that the arguments passed to
  `workflow.ExecuteActivity`, `workflow.ExecuteLocalActivity`,
  `workflow.ExecuteChildWorkflow`, `workflow.NewContinueAsNewError` and the
  `client.Client` methods `ExecuteWorkflow` and `SignalWithStartWorkflow` match the
  target function's real signature. The target's argument index differs per entry
  point — second for the `workflow.*` calls, third for `ExecuteWorkflow`, sixth for
  `SignalWithStartWorkflow` — and the client methods are matched by receiver.
  - **Arity** — the number of arguments matches what the target expects, after
    skipping the framework-injected leading parameter (`context.Context` for
    activities, `workflow.Context` for child workflows).
  - **Types** — each argument is assignable to its parameter (opt-in via the
    `strict-types` setting; arity is always checked).
  - Value-vs-pointer mismatches (`T`/`*T` and `[]T`/`[]*T`) are treated as
    compatible by default, matching Temporal's `DataConverter`; opt into flagging
    them with `strict-pointers`.
  - **Struct shape** — opt-in `strict-struct-shape` flags passing one struct type
    where a different struct is wanted. The `DataConverter` serializes by field
    name (JSON-tag aware), so distinct structs can round-trip; the message names
    the fields that silently drop or stay unset. A shared field with an
    incompatible type, or no shared fields at all, is reported as a `strict-types`
    error.
  - Each diagnostic is tagged with the source that produced it: `(arity)`,
    `(strict-types)`, `(strict-pointers)`, or `(strict-struct-shape)`. The three type
    checks are independent opt-ins; arity is always on.
  - Settings are grouped per analyzer under an `execargs:` block, leaving room
    for future analyzers to carry their own settings.
  - Variadic targets, package-level function activities and aliased imports of
    the workflow package are supported; string-registered targets and spread
    (`args...`) calls are intentionally skipped.
  - **Test mocks** — opt-in `strict-tests` extends the arity check to Temporal's
    `testsuite` mock setups, `(*testsuite.TestWorkflowEnvironment).OnActivity` and
    `.OnWorkflow`. Unlike `Execute*`, the matchers must cover **every** declared
    parameter — including the injected context — so the expected count differs by
    one; only arity is checked, since the matchers (`mock.Anything`,
    `mock.MatchedBy`) are opaque. Diagnostics are tagged `(strict-tests)`;
    string-named, spread, and variadic targets are skipped.
- `stringtarget` analyzer (opt-in, off by default): flags
  `workflow.ExecuteActivity`, `workflow.ExecuteLocalActivity`,
  `workflow.ExecuteChildWorkflow`, `workflow.NewContinueAsNewError` and the
  `client.Client` methods `ExecuteWorkflow` and `SignalWithStartWorkflow` calls
  whose target is named by **string** (a literal, a string variable, or a named
  string type) instead of a function reference. A string target can't be resolved to a signature, so it escapes
  `execargs`; this check surfaces those call sites so they can be refactored to a
  function reference that `execargs` *can* verify. Diagnostics are tagged
  `(string-target)`; enable via the `stringtarget.enabled` setting.
  - **Test mocks** — opt-in `strict-tests` extends the check to
    `(*testsuite.TestWorkflowEnvironment).OnActivity` and `.OnWorkflow` setups
    whose target is named by string. It is a layer on top of the production check,
    gated by `enabled` (the master switch): with `enabled` off the analyzer is
    silent regardless of `strict-tests`. Diagnostics are tagged `(strict-tests)`.
- `optionsdiscard` analyzer (on by default): flags
  `workflow.WithActivityOptions`, `workflow.WithLocalActivityOptions` and
  `workflow.WithChildOptions` calls whose returned context is **discarded** — used
  as a bare expression statement or assigned to `_`. Those functions return a new
  context carrying the options rather than mutating the one passed in, so a
  forgotten `ctx =` means the options silently never apply and the call fails at
  run time. Pure AST + types and near-zero false positives, so it runs by default
  (errcheck-style); diagnostics are tagged `(options-discard)`. Turn it off via
  the `optionsdiscard.disabled` setting.
- `activitytimeout` analyzer (on by default): inspects `workflow.ActivityOptions`
  and `workflow.LocalActivityOptions` composite literals and flags any that set
  fields but neither required timeout — `StartToCloseTimeout` nor
  `ScheduleToCloseTimeout`. Temporal requires at least one of the two, so an
  activity configured without either is rejected at run time. Pure AST + types and
  near-zero false positives, so it runs by default (errcheck-style); diagnostics
  are tagged `(required-timeout)`. Presence of the key satisfies the check (the
  value isn't evaluated); empty `{}` literals (typically populated field-by-field
  afterwards) and positional literals are intentionally skipped. Turn it off via
  the `activitytimeout.disabled` setting.
- `futureget` analyzer (on by default): flags a `.Get` call on a
  `workflow.Future`, `workflow.ChildWorkflowFuture` or `converter.EncodedValue`
  whose returned **error is discarded** — used as a bare expression statement or
  assigned to `_`. That error reports a failed activity, a failed child workflow
  or a decode error; dropping it silently swallows the failure (errcheck scoped to
  Temporal's result types). By construction it cannot fire on fire-and-forget,
  which never calls `.Get`. Matching is on the receiver's static type, so a user
  type that merely embeds `Future` is conservatively skipped. Pure AST + types and
  near-zero false positives, so it runs by default (errcheck-style); diagnostics
  are tagged `(future-get)`. Turn it off via the `futureget.disabled` setting.
- `lossynumber` analyzer (on by default): flags `interface{}`/`any`,
  `map[string]any` and `[]any` appearing as a top-level **parameter or return
  type** of an activity or workflow. Temporal's default `DataConverter` is JSON,
  and `encoding/json` decodes every number into a `float64` when the destination
  is the empty interface — so an `int64` past 2^53 round-trips with silent
  precision loss. The analyzer resolves the function referenced by each
  `workflow.ExecuteActivity`/`ExecuteLocalActivity`/`ExecuteChildWorkflow`/
  `NewContinueAsNewError` and `client.ExecuteWorkflow`/`SignalWithStartWorkflow`
  call to its real signature, skips the injected leading
  context and the trailing `error`, and reports any remaining parameter or return
  whose type is one of those lossy forms (a named empty interface counts; a
  non-empty interface such as `error` does not). The check is intentionally
  shallow — a struct that merely contains an `any` field is not flagged — so it
  stays false-positive-free; string-registered targets are skipped. Pure AST +
  types, so it runs by default; diagnostics are tagged `(lossy-types)`. Turn it
  off via the `lossynumber.disabled` setting (e.g. for a custom converter that
  preserves integer precision).
- `nonserializable` analyzer: flags types appearing as a top-level **parameter or
  return type** of an activity or workflow that Temporal's `DataConverter` cannot
  serialize. Sibling of `lossynumber` — same target resolution
  (`workflow.ExecuteActivity`/`ExecuteLocalActivity`/`ExecuteChildWorkflow`/
  `NewContinueAsNewError` and `client.ExecuteWorkflow`/`SignalWithStartWorkflow`) and
  the same shallow, top-level type predicate, but for types that *can't encode at
  all* rather than ones that decode lossily. Two checks:
  - **`chan` / `func`** (on by default): `encoding/json` returns an "unsupported
    type" error for both, so such a parameter or result can never round-trip — always
    a bug, nothing to opt into. A named channel/function type counts; a variadic
    `...chan T` is checked as `chan T`. Diagnostics are tagged `(unencodable)`.
  - **Struct with no exported fields** (opt-in via `empty-struct`): JSON marshals
    only exported fields, so such a struct encodes to `{}` and its data is silently
    dropped. A type implementing `json.Marshaler` controls its own encoding and is
    excluded; a fieldless `struct{}` carries no data and is not flagged. Diagnostics
    are tagged `(empty-struct)`. Opt-in because the `json.Marshaler` exclusion makes
    it less clear-cut than the always-on `chan`/`func` check.

  Deliberately shallow — a struct that merely contains a `chan` field, or a
  `[]chan`, is not flagged — so it stays false-positive-free; string-registered
  targets are skipped. Turn the analyzer off entirely via the
  `nonserializable.disabled` setting.
- `continueasnew` analyzer (on by default): flags a
  `workflow.NewContinueAsNewError` result that is **discarded** — used as a bare
  expression statement or assigned to `_` — instead of being returned. Returning
  that error is the only thing that makes a workflow continue as new; a dropped
  result means the workflow silently ends instead. Pure AST + types and near-zero
  false positives, so it runs by default (errcheck-style); diagnostics are tagged
  `(continue-as-new)`. Only the unambiguous discards are flagged — a result
  assigned to a named variable is left alone, since a `return err` may follow and
  proving otherwise would need data-flow analysis. Turn it off via the
  `continueasnew.disabled` setting.
- `sensitiveargs` analyzer (opt-in): flags an activity/workflow **parameter whose
  name** — or, for a struct parameter, an **exported field whose name** — matches a
  configurable regular expression, since Temporal records arguments in durable
  workflow history. Same target resolution as the sibling analyzers
  (`workflow.ExecuteActivity`/`ExecuteLocalActivity`/`ExecuteChildWorkflow`/
  `NewContinueAsNewError` and `client.ExecuteWorkflow`/`SignalWithStartWorkflow`).
  The default pattern is `(?i)cvv|pan|card.?number|password|secret|ssn|token`,
  overridable via the `sensitiveargs.pattern` setting. A name heuristic, so it is
  off by default (enable with `sensitiveargs.enabled`) — a useful first line of
  defence for keeping secrets and PII out of history. Top level only: it does not
  descend into nested structs, slices or maps, and only exported struct fields are
  considered (unexported fields are never serialized); it checks parameters, not
  return values, and skips string-registered targets. Diagnostics are tagged
  `(sensitive)`.
- Hermetic, offline `analysistest` fixtures: `testdata/` is a self-contained
  module that resolves `go.temporal.io/sdk` via a local stub, so it resolves in
  IDEs without pulling the real SDK.
- `conformance/` module: a compile-time contract test that builds against the
  real Temporal SDK in CI, catching any drift between the stub and the SDK's
  `Execute*`, `workflow.NewContinueAsNewError`, `client.ExecuteWorkflow`,
  `client.SignalWithStartWorkflow`, `testsuite` `OnActivity`/`OnWorkflow`,
  `With*Options`, and `Future`/`ChildWorkflowFuture`/`EncodedValue` `.Get`
  signatures.
