//nolint:temporalcheck // file-wide suppression: directive before the package clause

// Package nolintfile mutates a global from workflow code, but the file-level
// //nolint directive above the package clause suppresses every diagnostic.
package nolintfile

import "go.temporal.io/sdk/workflow"

var counter int

func Workflow(ctx workflow.Context) error {
	counter++
	counter = 3
	return nil
}
