// Package searchattribute flags a Temporal workflow entrypoint whose params
// struct carries a configured field but whose body never upserts the mapped
// search attribute. The mapping (field-name alias -> attribute name) is
// configured, so teams describe their own conventions. The check is opt-in
// because it is a heuristic over what a workflow "should" record.
package searchattribute

import (
	"go/ast"
	"strings"

	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/nolint"
	"github.com/samgozman/temporalcheck-lint/temporalcheck/internal/workflowscope"
	"golang.org/x/tools/go/analysis"
)

// Settings configures the searchattribute analyzer.
type Settings struct {
	Enabled    bool              // master switch (default false)
	Attributes map[string]string // field-name alias -> required search-attribute name
}

// NewAnalyzer builds the searchattribute analyzer for the given settings. The
// alias keys are normalized once here so field matching is case- and
// separator-insensitive (user_id, x-user-id and UserID all collapse to the same
// key).
func NewAnalyzer(settings Settings) *analysis.Analyzer {
	aliases := make(map[string]string, len(settings.Attributes))
	for alias, attr := range settings.Attributes {
		aliases[normalize(alias)] = attr
	}
	c := &checker{enabled: settings.Enabled, aliases: aliases}
	return &analysis.Analyzer{
		Name: "searchattribute",
		Doc:  "flag a Temporal workflow whose params carry a configured field but which never upserts the mapped search attribute",
		URL:  "https://github.com/samgozman/temporalcheck-lint",
		Run:  c.run,
	}
}

// checker threads the analyzer settings through the AST walk.
type checker struct {
	enabled bool
	// aliases maps a normalized field-name alias to the search-attribute name the
	// workflow must upsert when a field matches it.
	aliases map[string]string
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	if !c.enabled || len(c.aliases) == 0 {
		return nil, nil
	}
	for _, file := range pass.Files {
		// Workflows in _test.go files are test-harness workflows exercised with the
		// testsuite, not production entrypoints, so they don't record search
		// attributes; skip test files entirely.
		if strings.HasSuffix(pass.Fset.Position(file.Pos()).Filename, "_test.go") {
			continue
		}
		nolint := nolint.Collect(pass.Fset, file)
		// Only exported top-level functions are treated as workflow entrypoints. A
		// registered/executed workflow is exported (its worker registration lives in
		// another package, so the name must be), whereas an unexported function
		// taking workflow.Context is a workflow *helper* -- written to split up a
		// workflow's logic and called directly by name, not run by the runtime.
		// Checking those flags every helper, so we skip them. Nested closures are
		// likewise not entrypoints, so we do not descend into function bodies.
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !fn.Name.IsExported() || !workflowscope.IsWorkflowFunc(pass, fn.Type) {
				continue
			}
			c.checkWorkflow(pass, nolint, fn)
		}
	}
	return nil, nil
}

// normalize collapses a field name or configured alias to a case- and
// separator-insensitive key, so "x-user-id", "user_id" and "UserID" all match.
func normalize(s string) string {
	return strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(s))
}
