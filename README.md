# temporalcheck-lint

A [golangci-lint](https://golangci-lint.run) module plugin for static analysis of [Temporal](https://temporal.io) Go SDK code.

Temporal's APIs take targets and arguments as `interface{}`, erasing compile-time safety. This plugin recovers those checks statically, catching bugs that only surface at run time.

## What it catches

```
workflow.go:14  ExecuteActivity: activity "Greet" expects 1 argument, got 0 (arity)
workflow.go:20  ExecuteActivity: arg 1 of "Greet" has type int, want string (strict-types)
workflow.go:26  WithActivityOptions: the returned context is discarded, so the options never apply (options-discard)
workflow.go:30  ActivityOptions sets no required timeout (required-timeout)
workflow.go:36  Get: the returned error from Future.Get is discarded (future-get)
workflow.go:42  activity "Charge" parameter 1 has dynamic type any; numbers lose precision past 2^53 (lossy-types)
workflow.go:48  activity "Stream" parameter 1 has type chan int; DataConverter cannot serialize a channel (unencodable)
workflow.go:54  NewContinueAsNewError: the continue-as-new error is discarded (continue-as-new)
workflow.go:60  ExecuteActivity: this ctx is configured with WithChildOptions, not WithActivityOptions (options-context)
worker.go:14   worker.Options: MaxConcurrentWorkflowTaskPollers must not be 1 — the worker panics on start (worker-panic)
workflow.go:70  mutates package-level variable counter from workflow code (global-mutation)
workflow.go:76  logging via log in workflow code double-logs on every replay (workflow-logger)
```

## Analyzers

| Analyzer | What it flags | Default |
|---|---|---|
| [`execargs`](temporalcheck/execargs) | Wrong argument count/type for `Execute*` calls | on |
| [`optionsdiscard`](temporalcheck/optionsdiscard) | Discarded `With*Options` return value | on |
| [`activitytimeout`](temporalcheck/activitytimeout) | Activity options missing required timeout | on |
| [`futureget`](temporalcheck/futureget) | Discarded `Future.Get` error | on |
| [`lossynumber`](temporalcheck/lossynumber) | `any`/`interface{}` param/return that loses number precision | on |
| [`nonserializable`](temporalcheck/nonserializable) | `chan`/`func` param/return that can't serialize | on |
| [`continueasnew`](temporalcheck/continueasnew) | `NewContinueAsNewError` result not returned | on |
| [`optionscontext`](temporalcheck/optionscontext) | Context fed wrong `With*Options` type | on |
| [`workeroptions`](temporalcheck/workeroptions) | `worker.Options` fields that panic on start | on |
| [`workflowstate`](temporalcheck/workflowstate) | Mutation of package-level variables from workflow code | on |
| [`stringtarget`](temporalcheck/stringtarget) | String-named `Execute*` targets (blocks `execargs`) | **off** |
| [`sensitiveargs`](temporalcheck/sensitiveargs) | Params/fields whose name matches a sensitive-data pattern | **off** |
| [`workflowlogger`](temporalcheck/workflowlogger) | Non-replay-aware logging in workflow code | **off** |

Each analyzer's README has details, settings, and examples.

## Install

Module plugins are compiled into golangci-lint itself; see the [Module Plugin System docs](https://golangci-lint.run/docs/plugins/module-plugins/).

1. Add `.custom-gcl.yml` to your project:

   ```yaml
   version: v2.12.2
   name: custom-gcl
   destination: ./bin
   plugins:
     - module: github.com/samgozman/temporalcheck-lint
       import: github.com/samgozman/temporalcheck-lint/temporalcheck
       version: vX.Y.Z
   ```

2. Build the custom binary:

   ```bash
   golangci-lint custom   # produces ./bin/custom-gcl
   ```

3. Enable in `.golangci.yml` and run:

   ```bash
   ./bin/custom-gcl run
   ```

<details>
<summary>Full configuration reference</summary>

golangci-lint exposes the plugin as **`temporalcheck`**. Each analyzer has its own settings block:

```yaml
linters-settings:
  custom:
    temporalcheck:
      type: module
      settings:
        execargs:
          disabled: false
          strict-types: false
          strict-pointers: false
          strict-struct-shape: false
          strict-tests: false
        stringtarget:
          enabled: false
          strict-tests: false
        optionsdiscard:
          disabled: false
        activitytimeout:
          disabled: false
          require-start-to-close: false
        futureget:
          disabled: false
        lossynumber:
          disabled: false
        nonserializable:
          disabled: false
          empty-struct: false
        continueasnew:
          disabled: false
        sensitiveargs:
          enabled: false
          pattern: "(?i)cvv|pan|card.?number|password|secret|ssn|token"
        optionscontext:
          disabled: false
        workeroptions:
          disabled: false
          require-options: false
        workflowstate:
          disabled: false
        workflowlogger:
          enabled: false
```

</details>

## Suppressing with `//nolint`

Use the plugin name **`temporalcheck`** (not an analyzer name):

```go
// Suppress one call:
_ = workflow.ExecuteActivity(ctx, a.Greet) //nolint:temporalcheck // registered by name elsewhere

// Suppress an entire file (before the package clause):
//nolint:temporalcheck
package worker
```

A directive naming only other linters (`//nolint:gocritic`) does not suppress this plugin. To disable an analyzer project-wide, use its `disabled` setting.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT. See [LICENSE](LICENSE).
