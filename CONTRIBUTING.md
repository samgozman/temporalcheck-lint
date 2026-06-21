# Contributing

Thanks for taking the time to contribute! This is a small, focused
golangci-lint module plugin for Temporal, so the workflow is intentionally
simple.

## Getting started

```bash
go mod tidy
make test        # go test -race -v ./...
```

The [`Makefile`](Makefile) is the canonical entry point for every common task —
run `make help` to list the targets:

| Target             | What it does                                                     |
|--------------------|------------------------------------------------------------------|
| `make test`        | Run the test suite with the race detector.                       |
| `make cover-check` | Run tests and fail if coverage drops below 90%.                  |
| `make vet`         | `go vet ./...`.                                                  |
| `make tidy`        | `go mod tidy`.                                                   |
| `make build`       | Build a custom golangci-lint binary with the plugin compiled in. |
| `make run`         | Self-lint this repo with that custom binary.                     |

## How the linter is organised

The plugin is a thin registration shell around one analyzer per check, so new
Temporal checks slot in without touching the existing ones:

- [`temporalcheck/plugin.go`](temporalcheck/plugin.go) — registers the plugin
  with golangci-lint's module system and maps the `settings:` block onto each
  analyzer. `BuildAnalyzers` is where new analyzers are added.
- [`temporalcheck/execargs/`](temporalcheck/execargs) — the first analyzer,
  in its own package:
  - `execargs.go` — settings, the `analysis.Analyzer`, and call dispatch.
  - `check.go` — signature matching, arity/type checks, and helpers.

Adding the next Temporal check means dropping a sibling package next to
`execargs/`, exposing a `NewAnalyzer(Settings)`, and appending it to
`BuildAnalyzers`.

## Tests

Behaviour is verified with [`analysistest`](https://pkg.go.dev/golang.org/x/tools/go/analysis/analysistest)
fixtures under `temporalcheck/execargs/testdata/`, which is a **self-contained
Go module** (`testdata/go.mod`). The `go.temporal.io/sdk` import is satisfied by
a local stub via a `replace` directive, so the fixtures resolve offline (the
test runs with `GOPROXY=off`) and your IDE resolves the import instead of
flagging it. The Go tool ignores `testdata/`, so this module never affects
`go test/vet/build ./...` from the repo root.

One package per scenario:

- `good/` — must produce **zero** diagnostics (correct calls, plus the cases we
  intentionally skip: string-registered targets and spread calls).
- `bad/` — each call carries a `// want` regexp for its expected diagnostic.
- `notypes/` — run with `strict-types` off (the default); type mismatches must be
  silent while arity is still checked.
- `strictptr/` — run with `strict-pointers` on; value-vs-pointer mismatches that
  are allowed by default must be flagged here.
- `temporalsdk/` — a minimal stub module standing in for `go.temporal.io/sdk` so
  fixtures type-check without vendoring the real Temporal SDK.

Pure helpers and the plugin wiring have white-box unit tests alongside the code
(`*_internal_test.go`).

### Keeping the stub honest

The stub is hand-written, so it could drift from the real Temporal SDK. The
separate [`conformance/`](conformance) module guards against that: it imports
the **real** `go.temporal.io/sdk/workflow` and asserts at compile time that each
`Execute*` function still has the `(ctx, target, args...)` shape the analyzer
depends on. `make conformance` builds it; CI runs it on every PR and Dependabot
bumps the SDK version, so an upstream signature change fails loudly on the bump
PR — your cue to update the stub (and possibly the analyzer). It's a standalone
module (with its own `go.mod`/`go.sum`) so the SDK's heavy dependency tree never
enters the main module or the fixtures. It tracks a recent Go, as the SDK does;
the root module stays on Go 1.23.

When you change behaviour:

1. Add or update a `testdata` fixture with the expected `// want` diagnostics
   (the fixture *is* the spec — make it read clearly).
2. Add a unit test for any new pure helper.
3. Keep coverage at or above 90% (`make cover-check` enforces this in CI).

## Pull requests

- Keep changes small and focused; one behavioural change per PR where possible.
- Make sure `make cover-check`, `make vet`, and `make tidy` are clean — CI runs
  all three (plus a build-the-plugin-and-self-lint job) on every PR.
- Add an entry to [`CHANGELOG.md`](CHANGELOG.md) and, when behaviour changes,
  update the [`README.md`](README.md).

## Releasing (maintainers)

1. Move the `Unreleased` notes under a new version heading in `CHANGELOG.md`.
2. Tag the release (`git tag vX.Y.Z && git push --tags`); users pin this tag in
   their `.custom-gcl.yml`.
3. Keep the golangci-lint version pinned in `.custom-gcl.yml` and the CI
   workflow in sync.
