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

The `testdata/` and `conformance/` directories are **separate Go modules**, so
`./...` from the root never touches them; `analysistest` loads `testdata/` itself.

## Architecture

The plugin is a thin registration shell around **one analyzer per check**:

- `temporalcheck/plugin.go` — `register.Plugin("temporalcheck", New)`, maps the
  `.golangci.yml` `settings:` block onto each analyzer, and lists analyzers in
  `BuildAnalyzers`. `GetLoadMode` returns `LoadModeTypesInfo` — every analyzer gets
  full type information.
- `temporalcheck/execargs/` — the first analyzer, in its own package.

Settings use a per-analyzer nested block (`settings.execargs.*`) so analyzers added
later don't collide in a flat namespace. In `plugin.go`, settings fields are
`*bool` (unset vs explicit false) and are flattened to plain `bool` before being
handed to `NewAnalyzer`.

The analyzer **never imports the Temporal SDK** — it matches calls by package path
(`go.temporal.io/sdk/workflow`) through `go/types`, resolving via
`pass.TypesInfo.Uses` so aliased imports still match. The SDK only appears in test
fixtures via a local stub (`testdata/temporalsdk/`), and the `conformance/` module
asserts at compile time that the stub still matches the real SDK's signatures.

## `execargs` is the template for new analyzers

When adding a Temporal check, mirror this structure: a sibling package next to
`execargs/` exposing `NewAnalyzer(Settings) *analysis.Analyzer`, appended to
`BuildAnalyzers` with its own `*bool` settings on the `Settings` struct in
`plugin.go`.

Within the package, follow the same separation of concerns:

- `execargs.go` — the `Settings` struct, `NewAnalyzer`, and the `checker` type that
  threads settings through the walk (no package-level mutable state). `run` walks
  the AST and `checkCall` does call dispatch: confirm the callee, honor `//nolint`,
  bail on shapes we can't resolve, then hand off.
- `check.go` — the actual matching logic and pure helpers (signature comparison,
  type rendering, etc.).
- `nolint.go` — `//nolint` suppression, honored by the analyzer itself so it works
  in standalone/`analysistest` runs, not only under golangci-lint. golangci-lint
  exposes the plugin as **`temporalcheck`**, so that (or bare/`all`) is the name a
  directive must use — not the analyzer name `execargs`.

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
