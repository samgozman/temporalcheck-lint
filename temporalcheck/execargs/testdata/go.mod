// This is the analysistest fixture module. It is intentionally self-contained:
// the Temporal SDK import is satisfied by a local stub via the replace below, so
// the fixtures resolve offline (analysistest runs with GOPROXY=off) and your
// IDE can resolve "go.temporal.io/sdk/workflow" instead of flagging it.
//
// It is NOT part of the main module's build — the Go tool ignores testdata
// directories, so `go test/vet/build ./...` from the repo root never sees this.
module temporalcheckfixtures

go 1.23.0

require go.temporal.io/sdk v0.0.0

replace go.temporal.io/sdk => ./temporalsdk
