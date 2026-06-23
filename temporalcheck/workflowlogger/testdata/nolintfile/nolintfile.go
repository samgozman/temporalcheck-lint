//nolint:temporalcheck // file-wide suppression: directive before the package clause

// Package nolintfile logs from workflow code, but the file-level //nolint
// directive above the package clause suppresses every diagnostic.
package nolintfile

import (
	"log"

	"go.temporal.io/sdk/workflow"
)

func Workflow(ctx workflow.Context) error {
	log.Println("not reported, file suppressed")
	log.Printf("still not reported: %d", 1)
	return nil
}
