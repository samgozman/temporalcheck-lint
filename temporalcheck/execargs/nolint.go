package execargs

import (
	"go/ast"
	"go/token"
	"strings"
)

// nolintForExecargs reports whether a comment is a golangci-lint nolint
// directive that suppresses this linter: a bare "//nolint", "//nolint:all", or
// a "//nolint:..." list naming "temporalcheck" (the plugin name golangci-lint
// knows this analyzer by), ignoring any trailing "// explanation".
func nolintForExecargs(text string) bool {
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
		case "all", "temporalcheck":
			return true
		}
	}

	return false
}

// nolintInfo records where execargs nolint directives appear in a file.
type nolintInfo struct {
	fileSuppressed bool         // a directive before the package clause suppresses the whole file
	lines          map[int]bool // line numbers carrying a directive, to suppress a call on that line
}

// collectNolint scans the file's comments for execargs nolint directives.
func collectNolint(fset *token.FileSet, file *ast.File) nolintInfo {
	info := nolintInfo{lines: make(map[int]bool)}

	for _, group := range file.Comments {
		for _, c := range group.List {
			if !nolintForExecargs(c.Text) {
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

// suppressesCall reports whether a nolint directive covers the call: either the
// whole file is suppressed, or a directive sits on any line the call spans. The
// line-range check means a trailing //nolint works wherever the diagnostic
// anchors -- the opening paren for arity, or a specific argument for a type
// mismatch -- including calls written across several lines.
func (info nolintInfo) suppressesCall(fset *token.FileSet, call *ast.CallExpr) bool {
	if info.fileSuppressed {
		return true
	}

	start := fset.Position(call.Pos()).Line
	end := fset.Position(call.End()).Line
	for line := start; line <= end; line++ {
		if info.lines[line] {
			return true
		}
	}

	return false
}
