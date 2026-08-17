// Package temporal is a stand-in for go.temporal.io/sdk/temporal. It exists only
// so the analyzers' fixtures type-check without vendoring the real SDK. The
// search-attribute key types and SearchAttributeUpdate are aliases to the internal
// definitions, exactly as the real SDK re-exports them, so a fixture builds a
// typed update via temporal.NewSearchAttributeKeyKeyword("attr").ValueSet(v) just
// as production code does.
package temporal

import "go.temporal.io/sdk/internal"

// SearchAttributeUpdate and the key types are aliases to the internal definitions,
// exactly as the real SDK re-exports them (temporal.SearchAttributeUpdate =
// internal.SearchAttributeUpdate).
type (
	SearchAttributeUpdate     = internal.SearchAttributeUpdate
	SearchAttributeKeyString  = internal.SearchAttributeKeyString
	SearchAttributeKeyKeyword = internal.SearchAttributeKeyKeyword
)

// NewSearchAttributeKeyString creates a new string-based key.
func NewSearchAttributeKeyString(name string) SearchAttributeKeyString {
	return internal.NewSearchAttributeKeyString(name)
}

// NewSearchAttributeKeyKeyword creates a new keyword-based key.
func NewSearchAttributeKeyKeyword(name string) SearchAttributeKeyKeyword {
	return internal.NewSearchAttributeKeyKeyword(name)
}
