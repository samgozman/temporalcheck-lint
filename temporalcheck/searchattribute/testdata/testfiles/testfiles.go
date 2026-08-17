package testfiles

import "go.temporal.io/sdk/workflow"

// A regular (non-test) file: an exported workflow missing the upsert is flagged,
// proving the _test.go skip is scoped to test files, not the whole package.

type Params struct {
	UserID string
}

func Missing(ctx workflow.Context, p Params) error { // want `workflow "Missing" takes field "UserID" but never upserts the "user_id" search attribute \(searchattribute\)`
	return nil
}
