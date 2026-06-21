// Package converter is a minimal stand-in for go.temporal.io/sdk/converter.
// Unlike Future, the real SDK declares EncodedValue directly in this package (it
// is not re-exported from internal), so the stub does the same -- the analyzer
// matches it by the converter package path, not the internal one.
package converter

// EncodedValue mirrors the SDK interface. Its Get takes no context (unlike
// Future.Get) but likewise returns an error that must not be dropped.
type EncodedValue interface {
	HasValue() bool
	Get(valuePtr interface{}) error
}
