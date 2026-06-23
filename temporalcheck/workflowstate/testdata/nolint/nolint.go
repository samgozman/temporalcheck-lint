// Package nolint checks //nolint suppression: a directive naming temporalcheck,
// all, or a bare //nolint on the mutation's line suppresses it; a directive
// naming only another linter, or the analyzer name workflowstate, does not.
package nolint

import "go.temporal.io/sdk/workflow"

var counter int

func Workflow(ctx workflow.Context) error {
	counter++ //nolint:temporalcheck // suppressed by plugin name

	counter++ //nolint

	counter++ //nolint:all // all suppresses every linter

	// Naming only another linter does not suppress this one.
	counter++ //nolint:govet // want `mutates package-level variable counter from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`

	// The analyzer name is not the directive name golangci-lint uses (the plugin
	// name temporalcheck is), so it does not suppress.
	counter++ //nolint:workflowstate // want `mutates package-level variable counter from workflow code; shared mutable state breaks replay determinism and races across workflow executions \(global-mutation\)`

	return nil
}
