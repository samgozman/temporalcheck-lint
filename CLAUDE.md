# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A [golangci-lint](https://golangci-lint.run) module plugin that statically checks
[Temporal](https://temporal.io) Go SDK code. It ships one analyzer today
(`execargs`) and is deliberately laid out so more Temporal checks slot in beside it
under the same plugin. Read `README.md` for the user-facing behaviour and
`CONTRIBUTING.md` for the contributor workflow — this file is for working *on* the
code.

## Commands

`make help` lists every target. The common ones:

```bash
make test          # go test -race -v ./...
make cover-check   # tests + coverage gate (CI fails below 90%)
make vet           # go vet ./...
make conformance   # build ./conformance against the real Temporal SDK (needs recent Go)
make build         # build a custom golangci-lint binary with the plugin compiled in
make run           # self-lint this repo with that binary
```

`make build`/`make run` require the upstream `golangci-lint` CLI on `PATH` (it
runs `golangci-lint custom`); `make test`/`vet`/`cover-check` need only Go.

Run a single test: `go test -race -run TestExecArgs_StrictPointers ./temporalcheck/execargs/`.

The `testdata/`, `conformance/`, and `temporalcheck/internal/sdkstub/` directories
are **separate Go modules**, so `./...` from the root never touches them;
`analysistest` loads each analyzer's `testdata/` itself.

## Architecture

The plugin is a thin registration shell around **one analyzer per check**, with
cross-cutting infrastructure factored into shared `internal/` packages:

- `temporalcheck/plugin.go` — `register.Plugin("temporalcheck", New)`, maps the
  `.golangci.yml` `settings:` block onto each analyzer, and lists analyzers in
  `BuildAnalyzers`. `GetLoadMode` returns `LoadModeTypesInfo` — every analyzer gets
  full type information.
- `temporalcheck/execargs/` (and its siblings) — one analyzer per package, holding
  only that check's **domain logic**.
- `temporalcheck/internal/` — code genuinely shared by every analyzer, so it is not
  copied into each:
  - `nolint/` — `//nolint` directive parsing and suppression (`Collect` +
    `Info.Suppresses`). The suppression set is identical for every analyzer
    (golangci-lint knows the whole plugin as `temporalcheck`), so it lives here once.
  - `temporalsdk/` — the SDK import-path constants (`WorkflowPkg`, `InternalPkg`,
    `ClientPkg`, …) and the pure `go/types` matchers (`Named`, `IsWorkflowContext`,
    `IsReceiver`, `Deref`, `SkipCount`) that recognize the SDK.
  - `workflowscope/` — locating workflow definitions (`IsWorkflowFunc`, `FuncBody`,
    `Walk`), used by the determinism checks (workflowlogger, workflowstate).
  - `sdkstub/` — the single shared SDK stub module the fixtures resolve to (below).

Settings use a per-analyzer nested block (`settings.execargs.*`) so analyzers added
later don't collide in a flat namespace. In `plugin.go`, settings fields are
`*bool` (unset vs explicit false) and are flattened to plain `bool` before being
handed to `NewAnalyzer`.

The analyzers **never import the Temporal SDK** — they match calls by package path
(`go.temporal.io/sdk/workflow`) through `go/types`, resolving via
`pass.TypesInfo.Uses` so aliased imports still match. The SDK only appears in test
fixtures via one shared local stub (`temporalcheck/internal/sdkstub/`, a separate
module pulled in by each `testdata/go.mod` via `replace go.temporal.io/sdk =>
../../internal/sdkstub`), and the `conformance/` module asserts at compile time that
the stub still matches the real SDK's signatures.

### Gotcha: the SDK re-exports option/config *types* as aliases from `internal`

The SDK declares its option/config structs in `go.temporal.io/sdk/internal` and
re-exports them from `workflow` as **type aliases** — `type ActivityOptions =
internal.ActivityOptions`, same as `type Context = internal.Context`. This bites
any analyzer that matches an option **type** (not a function):

- `pass.TypesInfo.TypeOf(lit)` resolves to a type whose `Obj().Pkg().Path()` is
  `go.temporal.io/sdk/internal`, **not** `…/workflow`; and under `gotypesalias=1`
  (default in recent Go, which the `golangci-lint custom` binary is built with) the
  literal's type surfaces as `*types.Alias`, not `*types.Named`. So `t.(*types.Named)`
  + a `…/workflow` path check both fail. Call `types.Unalias(t)` first and accept
  **both** the `workflow` and `internal` package paths (see `activitytimeout/check.go`
  `optionTypeName`).
- The shared stub (`internal/sdkstub/`) mirrors the real shape: option/config and
  future types are declared in stub `internal` and re-exported from the public stub
  packages as aliases (`type ActivityOptions = internal.ActivityOptions`), exactly
  as the SDK does. A type-matching analyzer that instead tested against a struct
  declared directly in the public stub package would get **false-green tests** but
  report nothing against the real SDK. `conformance/` can't guard this (Go forbids
  importing `…/sdk/internal` from outside that module), so the stub is the only
  guard — keep new SDK types in `sdkstub/internal` and alias them out. (Function-only
  matchers like `execargs`/`optionsdiscard`/`stringtarget` are unaffected, since the
  functions they match live directly in `workflow`.)
- golangci-lint caches results and does **not** reliably invalidate on a plugin
  rebuild. When verifying a plugin change end-to-end, run `./bin/custom-gcl cache
  clean` after `make build`, or you'll chase stale "0 issues".

## `execargs` is the template for new analyzers

When adding a Temporal check, mirror this structure: a sibling package next to
`execargs/` exposing `NewAnalyzer(Settings) *analysis.Analyzer`, appended to
`BuildAnalyzers` with its own `*bool` settings on the `Settings` struct in
`plugin.go`.

Each analyzer owns its **domain logic**, but anything genuinely shared lives in the
`internal/` packages rather than being copied — `//nolint` suppression
(`internal/nolint`), SDK recognition (`internal/temporalsdk`), workflow discovery
(`internal/workflowscope`), and the test stub (`internal/sdkstub`). Reach for those
first; only add a new shared helper when two analyzers would otherwise duplicate it.

Within the package, follow the same separation of concerns:

- `execargs.go` — the `Settings` struct, `NewAnalyzer`, and the `checker` type that
  threads settings through the walk (no package-level mutable state). `run` walks
  the AST and `checkCall` does call dispatch: confirm the callee, honor `//nolint`
  via `nolint.Collect`/`Info.Suppresses`, bail on shapes we can't resolve, then hand
  off. (golangci-lint exposes the plugin as **`temporalcheck`**, so that, or
  bare/`all`, is the name a `//nolint` directive must use — not the analyzer name.)
- `check.go` — the actual matching logic and any pure helpers specific to this check
  (the generic `go/types` matchers come from `internal/temporalsdk`).

Design principles `execargs` establishes, worth keeping:

- **A false-positive-free baseline that's always on, stricter checks opt-in.** Arity
  always runs; type/pointer/struct-shape checks are independent opt-in layers.
- **Skip what can't be resolved statically** (string-registered targets, spread
  calls) rather than emit a false positive.
- **Tag every diagnostic with its source** — the message ends in `(arity)`,
  `(strict-types)`, etc., naming the setting that produced it.

## Tests

Behaviour is verified with `analysistest` fixtures under
`temporalcheck/execargs/testdata/` — **one package per scenario**, each with
`// want` regexps next to the expected diagnostics. The fixture *is* the spec; make
it read clearly. Pure helpers and plugin wiring have white-box `*_internal_test.go`
unit tests alongside the code.

Aim for **100% test coverage**; CI enforces a 90% floor via `make cover-check`. When
changing behaviour, update or add a fixture, add a unit test for any new pure
helper, and keep coverage up.

## Style

Match the existing code: comments explain *why* a non-obvious thing is done (e.g. why
`workflow.Context` resolves through the internal package), not what the code plainly
says. Don't over-comment or be verbose. Keep changes small and focused — one
behavioural change per PR — and update `CHANGELOG.md` (and `README.md` when
behaviour changes).

### Commit messages

Follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/). Format: `<type>[optional scope]: <description>`

**Types**: `feat`, `fix`, `chore`, `docs`, `test`, `ci`, `refactor`, `perf`, `style`, `build`, `revert`
