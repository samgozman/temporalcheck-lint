// Package flagged exercises the discarded-result shapes the analyzer reports:
// a bare call statement (the classic "forgot the ctx =") and an explicit blank
// assignment, across all three With*Options entry points, including an aliased
// import of the workflow package.
package flagged

import (
	"go.temporal.io/sdk/workflow"

	wf "go.temporal.io/sdk/workflow"
)

func Workflow(ctx workflow.Context) error {
	var (
		ao  workflow.ActivityOptions
		lao workflow.LocalActivityOptions
		cwo workflow.ChildWorkflowOptions
	)

	// Bare expression statements: the returned context is thrown away, so the
	// options never apply.
	workflow.WithActivityOptions(ctx, ao)       // want `WithActivityOptions: the returned context is discarded, so the options never apply; assign it back with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-discard\)`
	workflow.WithLocalActivityOptions(ctx, lao) // want `WithLocalActivityOptions: the returned context is discarded, so the options never apply; assign it back with ctx = workflow.WithLocalActivityOptions\(ctx, opts\) \(options-discard\)`
	workflow.WithChildOptions(ctx, cwo)         // want `WithChildOptions: the returned context is discarded, so the options never apply; assign it back with ctx = workflow.WithChildOptions\(ctx, opts\) \(options-discard\)`

	// Explicit blank assignment: also a discard.
	_ = workflow.WithActivityOptions(ctx, ao) // want `WithActivityOptions: the returned context is discarded, so the options never apply; assign it back with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-discard\)`

	// An aliased import resolves to the same workflow function, so it is flagged
	// just the same.
	wf.WithActivityOptions(ctx, ao) // want `WithActivityOptions: the returned context is discarded, so the options never apply; assign it back with ctx = workflow.WithActivityOptions\(ctx, opts\) \(options-discard\)`

	return nil
}
