// Package disabled mutates nothing special; it logs from workflow code, but the
// analyzer is constructed with Enabled=false in the disabled test, so it reports
// nothing here.
package disabled

import (
	"log"

	"go.temporal.io/sdk/workflow"
)

func Workflow(ctx workflow.Context) error {
	log.Println("not reported because the analyzer is disabled")
	return nil
}
