// Package workeroptions implements a static check for the Temporal Go SDK.
//
// A Temporal worker is configured with a worker.Options struct passed to
// worker.New. Two of its fields are foot-guns the compiler can't catch:
//
//   - MaxConcurrentWorkflowTaskExecutionSize and MaxConcurrentWorkflowTaskPollers
//     "cannot be 1 and will panic if set to that value" -- the pollers alternate
//     between sticky and non-sticky queues, so a single poller deadlocks the
//     worker at boot. A literal 1 there is a guaranteed crash.
//   - An empty worker.Options leaves the worker on the SDK defaults (1k concurrent
//     executions, 100k/s) that can overload a self-hosted cluster or a
//     memory-capped pod.
//
// This analyzer inspects worker.Options composite literals (and the literal passed
// to worker.New) so both are caught at lint time. The panic check is on by
// default and false-positive-free; the require-options check is opt-in.
package workeroptions

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Settings configures the workeroptions analyzer.
type Settings struct {
	// Disabled turns the analyzer off entirely; it reports nothing (this also
	// disables the default-on worker-panic check).
	Disabled bool

	// RequireOptions opts into flagging a worker.New whose worker.Options literal
	// sets none of the concurrency-limit fields, so the worker runs on the SDK
	// defaults. Off by default: an empty worker.Options is a legitimate choice when
	// the defaults suit the deployment, so this is opt-in.
	RequireOptions bool
}

// NewAnalyzer builds the workeroptions analyzer for the given settings.
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	c := &checker{
		disabled:       settings.Disabled,
		requireOptions: settings.RequireOptions,
	}
	return &analysis.Analyzer{
		Name: "workeroptions",
		Doc:  "flag Temporal worker.Options literals that set MaxConcurrentWorkflowTask{ExecutionSize,Pollers} to 1 (a worker-boot panic), and optionally worker.New calls whose worker.Options sets no concurrency limits",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk so the analyzer
// stays free of package-level mutable state.
type checker struct {
	disabled       bool
	requireOptions bool
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	if c.disabled {
		return nil, nil
	}
	for _, file := range pass.Files {
		nolint := collectNolint(pass.Fset, file)
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.CompositeLit:
				// worker-panic: any worker.Options literal, anywhere.
				c.checkPanic(pass, nolint, node)
			case *ast.CallExpr:
				// require-options: the worker.Options literal passed to worker.New.
				if c.requireOptions {
					c.checkRequireOptions(pass, nolint, node)
				}
			}
			return true
		})
	}
	return nil, nil
}
