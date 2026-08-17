package testfiles

import "go.temporal.io/sdk/workflow"

// Workflows defined in _test.go files -- test harness workflows exercised with the
// testsuite -- are not production entrypoints and don't record search attributes,
// so the analyzer skips test files entirely. No diagnostic here despite the
// missing upsert.

func MissingInTest(ctx workflow.Context, p Params) error {
	return nil
}
