// Package disabled mutates a global from workflow code; with the analyzer
// disabled it must report nothing.
package disabled

import "go.temporal.io/sdk/workflow"

var counter int

func Workflow(ctx workflow.Context) error {
	counter++
	return nil
}
