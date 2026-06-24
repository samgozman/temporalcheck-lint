# Contributing

Thanks for taking the time to contribute!

## Getting started

```bash
go mod tidy
make test        # go test -race -v ./...
```

The [`Makefile`](Makefile) is the canonical entry point — run `make help` to list targets:

| Target             | What it does                                                     |
|--------------------|------------------------------------------------------------------|
| `make test`        | Run the test suite with the race detector.                       |
| `make cover-check` | Run tests and fail if coverage drops below 90%.                  |
| `make vet`         | `go vet ./...`.                                                  |
| `make tidy`        | `go mod tidy`.                                                   |
| `make build`       | Build a custom golangci-lint binary with the plugin compiled in. |
| `make run`         | Self-lint this repo with that custom binary.                     |

## How it's organised

The plugin is a thin registration shell around one analyzer per check. New checks slot in without touching the existing ones:

- [`temporalcheck/plugin.go`](temporalcheck/plugin.go) — registers the plugin and maps the `settings:` block onto each analyzer. `BuildAnalyzers` is where new analyzers are added.
- [`temporalcheck/execargs/`](temporalcheck/execargs) — the template analyzer (see below).
- [`temporalcheck/internal/`](temporalcheck/internal) — shared infrastructure:
  - `nolint/` — `//nolint` directive parsing and suppression.
  - `temporalsdk/` — SDK import-path constants and `go/types` matchers.
  - `workflowscope/` — workflow-function detection (used by `workflowstate`, `workflowlogger`).
  - `sdkstub/` — the shared SDK stub for test fixtures.

## Adding a new analyzer

Mirror the `execargs` structure: a sibling package under `temporalcheck/` exposing `NewAnalyzer(Settings) *analysis.Analyzer`, appended to `BuildAnalyzers` in `plugin.go` with its own settings block.

Within the package:

- `<name>.go` — `Settings`, `NewAnalyzer`, the `checker` type, `run` (AST walk), and call dispatch.
- `check.go` — matching logic and any helpers specific to this check (generic helpers go in `internal/`).

The analyzers **never import the Temporal SDK** — they match by package path through `go/types`.

## Tests

Behavior is verified with [`analysistest`](https://pkg.go.dev/golang.org/x/tools/go/analysis/analysistest) fixtures under each analyzer's `testdata/` directory, which is a **self-contained Go module**. The `go.temporal.io/sdk` import is satisfied by a local stub via a `replace` directive, so fixtures resolve offline and with no real SDK dependency.

Each `testdata/` package has `// want` assertions next to expected diagnostics. The fixture is the spec — make it read clearly.

Aim for **100% coverage**; CI enforces a 90% floor via `make cover-check`. When changing behavior:

1. Add or update a `testdata` fixture with `// want` assertions.
2. Add a unit test for any new pure helper.
3. Keep coverage at or above 90%.

### The stub and conformance check

The stub (`temporalcheck/internal/sdkstub/`) is hand-written and could drift. The separate [`conformance/`](conformance) module guards against this: it imports the **real** `go.temporal.io/sdk` and asserts at compile time that the shapes the analyzers depend on still hold. `make conformance` builds it; CI runs it on every PR.

A key subtlety: the SDK re-exports option/config types from `workflow` as **type aliases** from `internal`. The stub mirrors this — option types live in `sdkstub/internal` and are aliased out. An analyzer matching a type literal must call `types.Unalias(t)` and accept both `workflow` and `internal` package paths.

## Layout

```
temporalcheck-lint/
├── temporalcheck/
│   ├── plugin.go                 # register.Plugin("temporalcheck", ...)
│   ├── execargs/                 # check Execute* call argument arity and types
│   ├── stringtarget/             # flag string-named Execute* targets
│   ├── optionsdiscard/           # flag discarded With*Options results
│   ├── activitytimeout/          # flag Activity options missing a required timeout
│   ├── futureget/                # flag discarded Future/EncodedValue .Get errors
│   ├── lossynumber/              # flag any/interface{} params/returns (number precision)
│   ├── nonserializable/          # flag chan/func params/returns (can't serialize)
│   ├── continueasnew/            # flag discarded NewContinueAsNewError results
│   ├── sensitiveargs/            # flag params/fields matching a sensitive-data pattern
│   ├── optionscontext/           # flag Execute* calls fed the wrong With*Options context
│   ├── workeroptions/            # flag worker.Options boot-panic + missing limits
│   ├── workflowstate/            # flag package-level variable mutation from workflow code
│   ├── workflowlogger/           # flag non-replay-aware logging in workflow code
│   └── internal/
│       ├── nolint/               # //nolint directive handling
│       ├── temporalsdk/          # SDK import-path constants and type matchers
│       ├── workflowscope/        # workflow-function detection
│       └── sdkstub/              # shared SDK stub for test fixtures
├── conformance/                  # CI-only: real-SDK contract test
├── .custom-gcl.yml               # custom golangci-lint build config
├── .golangci.yml                 # example consumer config (also self-lints this repo)
├── Makefile
└── go.mod
```

## Roadmap / not yet built

Ideas that aren't implemented yet — contributions welcome. Open an issue first if
you want to take one on.

- **Suggested fixes (auto-fix).** Most diagnostics have an obvious mechanical
  remedy: assign a discarded `With*Options` context back (`ctx = workflow.With…(ctx, o)`),
  add a missing `StartToCloseTimeout`, replace `log`/`fmt` with `workflow.GetLogger(ctx)`,
  or check a discarded `Future.Get` / `NewContinueAsNewError` result. golangci-lint
  surfaces an analyzer's [`SuggestedFixes`](https://pkg.go.dev/golang.org/x/tools/go/analysis#Diagnostic)
  via `--fix`; none of the analyzers attach them yet.
- **Interprocedural determinism analysis.** `workflowstate` and `workflowlogger`
  currently only inspect code lexically inside the workflow function (and its
  closures), not helper functions it calls. Following calls into same-package helpers
  would catch a real class of non-determinism the checks miss today.
- **Dot-import support.** All analyzers match `pkg.Func(...)` selector calls and skip
  dot-imported SDK calls (`import . ".../workflow"`). Handling the bare-identifier form
  would close that gap.
- **More checks.** Candidates noted in `plugin.go`: registration coverage (every
  registered activity/workflow has a matching definition), retry-policy sanity, and
  signal/query/update handler checks.

## Pull requests

- Keep changes small and focused; one behavioral change per PR.
- Make sure `make cover-check`, `make vet`, and `make tidy` are clean — CI runs all three (plus a build-and-self-lint job) on every PR.
- Add an entry to [`CHANGELOG.md`](CHANGELOG.md) and update [`README.md`](README.md) when behavior changes.

## Releasing (maintainers)

1. Move the `Unreleased` notes under a new version heading in `CHANGELOG.md`.
2. Tag the release (`git tag vX.Y.Z && git push --tags`); users pin this tag in their `.custom-gcl.yml`.
3. Keep the golangci-lint version pinned in `.custom-gcl.yml` and the CI workflow in sync.
