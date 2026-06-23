// analysistest fixture module. The replaces point the Temporal SDK and zerolog
// imports at local stubs, so fixtures resolve offline and in IDEs. The Go tool
// ignores testdata/, so this never affects the root module.
module workflowloggerfixtures

go 1.23.0

require (
	github.com/rs/zerolog v0.0.0
	go.temporal.io/sdk v0.0.0
)

replace go.temporal.io/sdk => ./temporalsdk

replace github.com/rs/zerolog => ./zerolog
