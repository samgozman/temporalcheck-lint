// Package nolint honors //nolint directives for the temporalcheck plugin's analyzers.
// Each analyzer applies suppression itself so it behaves identically in standalone
// analysistest runs (where golangci-lint filtering doesn't happen).
//
// golangci-lint exposes the whole plugin as "temporalcheck", so a bare //nolint,
// //nolint:all, or //nolint:temporalcheck suppresses any of its analyzers.
package nolint

import (
	"go/ast"
	"go/token"
	"strings"
)

// pluginName is the name a //nolint list must use to suppress this plugin.
const pluginName = "temporalcheck"

// directive reports whether text is a //nolint directive that suppresses this plugin.
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

// Info records where suppressing directives appear in a file.
// The zero value suppresses nothing; obtain a populated value from Collect.
type Info struct {
	fileSuppressed bool         // a directive before the package clause suppresses the whole file
	lines          map[int]bool // line numbers carrying a directive
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

// Suppresses reports whether a directive covers node: the whole file is suppressed,
// or a directive sits on any line the node spans.
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
