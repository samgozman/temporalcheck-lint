package aliases

import "go.temporal.io/sdk/workflow"

// Field-name matching is case- and separator-insensitive, and several field
// aliases can map to the same attribute. Config maps client_id -> user_id and
// x-user-id -> user_id, so a ClientID or XUserID field requires the user_id
// attribute just as UserID does.

type ClientParams struct {
	ClientID string
}

type KebabParams struct {
	XUserID string
}

// ByClientID: the ClientID field (alias) requires user_id, which is missing.
func ByClientID(ctx workflow.Context, p ClientParams) error { // want `workflow "ByClientID" takes field "ClientID" but never upserts the "user_id" search attribute \(searchattribute\)`
	return nil
}

// ByKebabAlias: the config key "x-user-id" normalizes to match the XUserID field.
func ByKebabAlias(ctx workflow.Context, p KebabParams) error { // want `workflow "ByKebabAlias" takes field "XUserID" but never upserts the "user_id" search attribute \(searchattribute\)`
	return nil
}
