// Minimal stand-in module for go.temporal.io/sdk, used only by the fixtures via
// a local replace directive. It carries just enough of the workflow package's
// surface for the fixtures to type-check; it is never published or fetched.
module go.temporal.io/sdk

go 1.23.0
