// Package benign exercises the shapes the analyzer must leave alone: a checked
// error, an error assigned to a real variable, a non-Get method on a matched
// receiver, a Get on a non-Temporal type, a Get on a user type that merely embeds
// workflow.Future, and a selector into another package. None produce
// diagnostics.
package benign

import (
	"strings"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

// myFuture is an unrelated local type whose Get returns an error; a discarded
// call to it must not be flagged -- only the Temporal receiver types are.
type myFuture struct{}

func (myFuture) Get(ctx workflow.Context, valuePtr interface{}) error { return nil }

// wrapped embeds workflow.Future. A discarded w.Get has static receiver type
// `wrapped` (this package), not Future, so the analyzer conservatively skips it.
type wrapped struct{ workflow.Future }

func Workflow(ctx workflow.Context, ev converter.EncodedValue) error {
	var result string

	f := workflow.ExecuteActivity(ctx, "Activity")

	// Checked error: the canonical correct usage.
	if err := f.Get(ctx, &result); err != nil {
		return err
	}

	// Assigned to a real variable and returned: not a discard.
	err := f.Get(ctx, &result)
	if err != nil {
		return err
	}

	// Non-Get method on a matched receiver: returns bool, named IsReady, so it is
	// outside what this analyzer owns.
	f.IsReady()

	// Get on a non-Temporal type: resolved receiver is out of scope.
	var mf myFuture
	mf.Get(ctx, nil)

	// Get on a user type that only embeds workflow.Future: static type is local.
	var w wrapped
	w.Get(ctx, &result)

	// EncodedValue error kept in a variable that is then inspected.
	if verr := ev.Get(&result); verr != nil {
		return verr
	}

	// Selector into a non-Temporal package.
	_ = strings.ToUpper("x")

	return nil
}
