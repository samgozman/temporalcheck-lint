package nolint

import "go.temporal.io/sdk/workflow"

// The diagnostic is anchored to the workflow's signature line, so a //nolint
// directive suppressing the plugin belongs there.

type Params struct {
	UserID string
}

func Suppressed(ctx workflow.Context, p Params) error { //nolint:temporalcheck // names the plugin: suppressed
	return nil
}

func Bare(ctx workflow.Context, p Params) error { //nolint
	return nil
}

func OtherLinter(ctx workflow.Context, p Params) error { //nolint:otherlinter // want `workflow "OtherLinter" takes field "UserID" but never upserts the "user_id" search attribute \(searchattribute\)`
	return nil
}

func AnalyzerName(ctx workflow.Context, p Params) error { //nolint:searchattribute // analyzer name, not the plugin: not suppressed // want `workflow "AnalyzerName" takes field "UserID" but never upserts the "user_id" search attribute \(searchattribute\)`
	return nil
}
