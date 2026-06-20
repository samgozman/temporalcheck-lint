package bad

// Aliased import: the analyzer resolves the callee by its package path, not by
// the local import name, so an alias must still be recognised.
import wf "go.temporal.io/sdk/workflow"

func WorkflowAliased(ctx wf.Context) error {
	var a *Activities
	_ = wf.ExecuteActivity(ctx, a.Greet) // want `ExecuteActivity: activity "Greet" expects 1 argument, got 0`
	return nil
}
