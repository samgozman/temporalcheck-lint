// Shared local stub for go.temporal.io/sdk, used by every analyzer's
// analysistest fixtures via a replace directive (see each
// <analyzer>/testdata/go.mod). It carries the union of the SDK surface the
// fixtures need, declared in the same shape as the real SDK -- option/config
// types and futures live in the internal package and are re-exported from their
// public packages as aliases, so the analyzers are exercised against the SDK's
// real type identities. It is its own module, so the root module's ./... never
// builds it.
module go.temporal.io/sdk

go 1.23.0
