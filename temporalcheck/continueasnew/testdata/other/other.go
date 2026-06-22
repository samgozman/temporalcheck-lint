// Package other declares a function named NewContinueAsNewError in a package
// other than the SDK's workflow, so fixtures can confirm the analyzer matches by
// package path -- a same-named call here must not be flagged.
package other

// NewContinueAsNewError is an unrelated namesake; its result being discarded is
// not a Temporal continue-as-new bug.
func NewContinueAsNewError(ctx interface{}, wfn interface{}) error { return nil }
