// Package nolint honors golangci-lint //nolint directives for the temporalcheck
// plugin's analyzers. Each analyzer applies suppression itself (rather than
// relying on golangci-lint to filter results) so it behaves identically under
// standalone/analysistest runs, where no such filtering happens.
//
// golangci-lint exposes the whole plugin under the single name "temporalcheck",
// not under each analyzer's name, so the suppression set is the same for every
// analyzer: a bare //nolint, //nolint:all, or a //nolint:... list naming
// "temporalcheck". That is why this logic lives here once instead of in each
// analyzer.
package nolint

import (
	"go/ast"
	"go/token"
	"strings"
)

// pluginName is the name golangci-lint knows this plugin by; a //nolint list
// must name it (or "all") to suppress, never an individual analyzer's name.
const pluginName = "temporalcheck"

// directive reports whether text is a //nolint directive that suppresses this
// plugin: a bare "//nolint", "//nolint:all", or a "//nolint:..." list naming
// "temporalcheck", ignoring any trailing "// explanation".
func directive(text string) bool {
	if !strings.HasPrefix(text, "//nolint") {
		return false
	}

	rest := strings.TrimPrefix(text, "//nolint")
	if rest == "" {
		return true // bare //nolint suppresses every linter
	}

	if !strings.HasPrefix(rest, ":") {
		return false // e.g. "//nolintfoo"
	}

	list := strings.TrimPrefix(rest, ":")
	if i := strings.Index(list, "//"); i >= 0 {
		list = list[:i]
	}

	for _, name := range strings.Split(list, ",") {
		switch strings.TrimSpace(name) {
		case "all", pluginName:
			return true
		}
	}

	return false
}

// Info records where suppressing directives appear in a file. The zero value
// suppresses nothing; obtain a populated value from Collect.
type Info struct {
	fileSuppressed bool         // a directive before the package clause suppresses the whole file
	lines          map[int]bool // line numbers carrying a directive, to suppress a node on that line
}

// Collect scans the file's comments for suppressing directives.
func Collect(fset *token.FileSet, file *ast.File) Info {
	info := Info{lines: make(map[int]bool)}

	for _, group := range file.Comments {
		for _, c := range group.List {
			if !directive(c.Text) {
				continue
			}

			info.lines[fset.Position(c.Pos()).Line] = true
			if c.Pos() < file.Package {
				info.fileSuppressed = true
			}
		}
	}

	return info
}

// Suppresses reports whether a directive covers node: either the whole file is
// suppressed, or a directive sits on any line the node spans. The line-range
// check means a trailing //nolint works wherever the diagnostic anchors --
// including nodes written across several lines.
func (info Info) Suppresses(fset *token.FileSet, node ast.Node) bool {
	if info.fileSuppressed {
		return true
	}

	start := fset.Position(node.Pos()).Line
	end := fset.Position(node.End()).Line
	for line := start; line <= end; line++ {
		if info.lines[line] {
			return true
		}
	}

	return false
}
