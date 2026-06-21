# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

Initial proof of concept.

- `execargs` analyzer: checks that the arguments passed to
  `workflow.ExecuteActivity`, `workflow.ExecuteLocalActivity` and
  `workflow.ExecuteChildWorkflow` match the target function's real signature.
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
- `stringtarget` analyzer (opt-in, off by default): flags
  `workflow.ExecuteActivity`, `workflow.ExecuteLocalActivity` and
  `workflow.ExecuteChildWorkflow` calls whose target is named by **string** (a
  literal, a string variable, or a named string type) instead of a function
  reference. A string target can't be resolved to a signature, so it escapes
  `execargs`; this check surfaces those call sites so they can be refactored to a
  function reference that `execargs` *can* verify. Diagnostics are tagged
  `(string-target)`; enable via the `stringtarget.enabled` setting.
- Hermetic, offline `analysistest` fixtures: `testdata/` is a self-contained
  module that resolves `go.temporal.io/sdk` via a local stub, so it resolves in
  IDEs without pulling the real SDK.
- `conformance/` module: a compile-time contract test that builds against the
  real Temporal SDK in CI, catching any drift between the stub and the SDK's
  `Execute*` signatures.
