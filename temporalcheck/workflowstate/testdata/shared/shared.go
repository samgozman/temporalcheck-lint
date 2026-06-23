// Package shared holds a package-level variable mutated from another package's
// workflow code, so the analyzer is exercised on cross-package global mutation
// (the qualified selector form pkg.Global), not only same-package vars.
package shared

// Global is a package-level variable. A workflow in another package mutating it
// is exactly the shared-state hazard the analyzer reports.
var Global int

// Registry is a package-level map; mutating an entry from a workflow races
// across executions just as a scalar assignment does.
var Registry = map[string]int{}
