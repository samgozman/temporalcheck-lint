package structshape

import "go.temporal.io/sdk/workflow"

// Run with StructShape on (and StrictTypes/StrictPointers off), proving the check
// is independent: only struct-shape findings, plus the hard struct cases that the
// setting also surfaces.
func Workflow(ctx workflow.Context) error {
	var a *Activities

	// Wire-compatible but distinct: drops a source-only field and leaves a
	// target-only field unset.
	_ = workflow.ExecuteActivity(ctx, a.CreateTransfer, &TransferParams{}) // want `ExecuteActivity: arg 1 of "CreateTransfer" sends \*structshape.TransferParams, target wants \*structshape.EnrichedTransferParams -- serializes by field name but drops \{OnlyOnSource\} and leaves \{OnlyOnTarget\} unset \(strict-struct-shape\)`

	// Shared field with an incompatible type: a hard error, tagged strict-types
	// even though struct-shape surfaced it.
	_ = workflow.ExecuteActivity(ctx, a.NeedTarget, &ConflictParams{}) // want `ExecuteActivity: arg 1 of "NeedTarget" sends \*structshape.ConflictParams, target wants \*structshape.TargetParams -- field "SourceAccountID" is incompatible \(int vs string\) \(strict-types\)`

	// No fields in common: almost certainly the wrong type entirely.
	_ = workflow.ExecuteActivity(ctx, a.NeedTarget, &Unrelated{}) // want `ExecuteActivity: arg 1 of "NeedTarget" sends \*structshape.Unrelated, target wants \*structshape.TargetParams -- no fields in common \(strict-types\)`

	// json tags: ID matches via "identifier"; Secret (json:"-") and internal
	// (unexported) never reach the wire, so only Extra is left unset.
	_ = workflow.ExecuteActivity(ctx, a.NeedTagged, &TaggedSend{}) // want `ExecuteActivity: arg 1 of "NeedTagged" sends \*structshape.TaggedSend, target wants \*structshape.TaggedWant -- serializes by field name but leaves \{Extra\} unset \(strict-struct-shape\)`

	// Identical fields, distinct Go type.
	_ = workflow.ExecuteActivity(ctx, a.NeedB, &AParams{}) // want `ExecuteActivity: arg 1 of "NeedB" sends \*structshape.AParams, target wants \*structshape.BParams -- has identical fields but is a distinct Go type \(strict-struct-shape\)`

	// Correct type: silent.
	_ = workflow.ExecuteActivity(ctx, a.NeedB, &BParams{})

	return nil
}
