// Package nolint checks //nolint suppression: a directive naming temporalcheck,
// all, or a bare //nolint on the logging call's line suppresses it; a directive
// naming only another linter, or the analyzer name workflowlogger, does not.
package nolint

import (
	"log"

	"go.temporal.io/sdk/workflow"
)

func Workflow(ctx workflow.Context) error {
	log.Println("a") //nolint:temporalcheck // suppressed by plugin name

	log.Println("b") //nolint

	log.Println("c") //nolint:all // all suppresses every linter

	// Naming only another linter does not suppress this one.
	log.Println("d") //nolint:govet // want `logging via log in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`

	// The analyzer name is not the directive name golangci-lint uses (the plugin
	// name temporalcheck is), so it does not suppress.
	log.Println("e") //nolint:workflowlogger // want `logging via log in workflow code double-logs on every replay and is not replay-aware; use workflow.GetLogger\(ctx\) instead \(workflow-logger\)`

	return nil
}
