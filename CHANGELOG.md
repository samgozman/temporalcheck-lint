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
  - **Types** — each argument is assignable to its parameter (toggle with the
    `check-types` setting; arity is always checked).
  - Variadic targets, package-level function activities and aliased imports of
    the workflow package are supported; string-registered targets and spread
    (`args...`) calls are intentionally skipped.
- Hermetic, offline `analysistest` fixtures: `testdata/` is a self-contained
  module that resolves `go.temporal.io/sdk` via a local stub, so it resolves in
  IDEs without pulling the real SDK.
- `conformance/` module: a compile-time contract test that builds against the
  real Temporal SDK in CI, catching any drift between the stub and the SDK's
  `Execute*` signatures.
