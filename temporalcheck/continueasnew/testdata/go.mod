// analysistest fixture module. The replace points the Temporal SDK import at a
// local stub, so fixtures resolve offline and in IDEs. The Go tool ignores
// testdata/, so this never affects the root module.
module continueasnewfixtures

go 1.23.0

require go.temporal.io/sdk v0.0.0

replace go.temporal.io/sdk => ./temporalsdk
