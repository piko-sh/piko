// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package driven_code_emitter_go_literal

import (
	"cmp"
	goast "go/ast"
	"go/token"
	"slices"
	"strings"
)

// collectUsedQualifiers returns the package qualifiers the code references.
//
// It walks the generated file AST and records every identifier X that appears as the
// left-hand side of a selector expression X.Sel.
//
// The emitter assembles its import block as a UNION of every package any partial in the
// component graph might need, so it over-emits. goimports used to prune the unreferenced
// specs per file (and add missing ones), at the cost of walking the whole module tree on
// every artefact. This set is what lets the emitter do that pruning locally, with no
// filesystem access.
//
// Takes fileAST (*goast.File) which is the generated file AST to walk.
//
// Returns map[string]struct{} which is the set of referenced package qualifiers.
func collectUsedQualifiers(fileAST *goast.File) map[string]struct{} {
	used := make(map[string]struct{})
	goast.Inspect(fileAST, func(n goast.Node) bool {
		if sel, ok := n.(*goast.SelectorExpr); ok {
			if id, ok := sel.X.(*goast.Ident); ok {
				used[id.Name] = struct{}{}
			}
		}
		return true
	})
	return used
}

// importSpecQualifier returns the local name a spec is referenced by.
//
// It uses the spec's explicit alias when present, otherwise the package's assumed name
// (the final path segment). The emitter only leaves a spec unaliased for the standard
// library / runtime packages it adds, whose package name equals the final segment, so the
// segment is an accurate qualifier here.
//
// Takes spec (*goast.ImportSpec) which is the import spec to qualify.
//
// Returns string which is the local name the spec is referenced by.
func importSpecQualifier(spec *goast.ImportSpec) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	return importPathBase(importSpecPath(spec))
}

// importSpecName returns a spec's explicit alias.
//
// Takes spec (*goast.ImportSpec) which is the import spec to inspect.
//
// Returns string which is the explicit alias, or "" when the spec has none.
func importSpecName(spec *goast.ImportSpec) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	return ""
}

// importSpecPath returns a spec's import path with surrounding quotes stripped.
//
// Takes spec (*goast.ImportSpec) which is the import spec to read the path from.
//
// Returns string which is the import path without surrounding quotes.
func importSpecPath(spec *goast.ImportSpec) string {
	return strings.Trim(spec.Path.Value, `"`)
}

// importPathBase returns the final segment of an import path.
//
// Takes path (string) which is the import path to take the final segment of.
//
// Returns string which is the final path segment.
func importPathBase(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}

// resolvedQualifier returns the local name a resolver-recorded import is referenced by.
//
// It uses the alias when set, otherwise the package's assumed name.
//
// Takes alias (string) which is the recorded import alias, or "" when none.
// Takes path (string) which is the recorded import path.
//
// Returns string which is the local name the import is referenced by.
func resolvedQualifier(alias, path string) string {
	if alias != "" {
		return alias
	}
	return importPathBase(path)
}

// synthesisedImportSpec builds the spec goimports would add for a dropped package.
//
// It covers the dual-path case where one path is referenced under two qualifiers and the
// path-keyed import set dropped it. goimports adds such imports minimally, so the alias
// is omitted when it equals the package's assumed name.
//
// Takes path (string) which is the import path of the referenced package.
// Takes alias (string) which is the desired alias, or "" when none.
//
// Returns *goast.ImportSpec which is the synthesised import spec.
func synthesisedImportSpec(path, alias string) *goast.ImportSpec {
	spec := &goast.ImportSpec{Path: strLit(path)}
	if q := resolvedQualifier(alias, path); q != importPathBase(path) {
		spec.Name = cachedIdent(q)
	}
	return spec
}

// importGroup classifies an import path into a gofmt/goimports group.
//
// It returns 0 for the standard library (no dot in the first path segment) and 1 for
// everything else. Groups are rendered separated by a single blank line. (goimports'
// appengine and LocalPrefix groups are unused in this codebase.)
//
// Takes path (string) which is the import path to classify.
//
// Returns int which is 0 for the standard library and 1 for everything else.
func importGroup(path string) int {
	first, _, _ := strings.Cut(path, "/")
	if strings.Contains(first, ".") {
		return 1
	}
	return 0
}

// buildImportDeclFromSpecs assembles the import GenDecl from a slice of specs.
//
// It sorts the specs into goimports order: by group, then path, then alias (an unaliased
// spec sorts before an aliased one for the same path). The block is printed directly by
// formatAndVerify, which inserts the inter-group blank line, so this order is the final
// order.
//
// Takes specs ([]*goast.ImportSpec) which are the specs to assemble and sort.
//
// Returns *goast.GenDecl which is the assembled import declaration.
func buildImportDeclFromSpecs(specs []*goast.ImportSpec) *goast.GenDecl {
	slices.SortFunc(specs, func(a, b *goast.ImportSpec) int {
		pa, pb := importSpecPath(a), importSpecPath(b)
		if c := cmp.Compare(importGroup(pa), importGroup(pb)); c != 0 {
			return c
		}
		if c := cmp.Compare(pa, pb); c != 0 {
			return c
		}
		return cmp.Compare(importSpecName(a), importSpecName(b))
	})

	out := make([]goast.Spec, len(specs))
	for i, s := range specs {
		out[i] = s
	}
	return &goast.GenDecl{Tok: token.IMPORT, Lparen: 1, Specs: out}
}

// insertImportGroupBlankLine inserts a blank line between import groups in source.
//
// It works on already-printed source, reproducing goimports' addImportSpaces without
// re-parsing. Specs are emitted pre-sorted by group, so each group transition gets one
// blank line.
//
// The scan is bounded strictly to the generated import block, which is always the file's
// first declaration (the emitter prepends it and strips any user imports). Once that
// block closes, the pass goes inert for the remainder of the file: user code copied
// verbatim can contain raw-string literals that look like an import block, and a naive
// re-entering scan would inject a blank line inside such a string and silently mutate its
// runtime value.
//
// Takes src ([]byte) which is the already-printed source to process.
//
// Returns []byte which is a fresh slice (the input may be a pooled buffer's bytes).
func insertImportGroupBlankLine(src []byte) []byte {
	lines := strings.Split(string(src), "\n")
	open := importBlockOpenIndex(lines)
	if open < 0 {
		return src
	}

	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:open+1]...)
	prevGroup := -1
	for i, line := range lines[open+1:] {
		if strings.TrimSpace(line) == ")" {
			return []byte(strings.Join(append(out, lines[open+1+i:]...), "\n"))
		}
		group, blankBefore := importGroupTransition(line, prevGroup)
		if blankBefore {
			out = append(out, "")
		}
		prevGroup = group
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n"))
}

// importBlockOpenIndex returns the index of the generated import block's open line.
//
// The generated `import (` line is always the file's first parenthesised import block
// (the emitter prepends it and strips user imports).
//
// Takes lines ([]string) which are the source lines to scan.
//
// Returns int which is the index of the `import (` line, or -1 when there is no
// parenthesised import block.
func importBlockOpenIndex(lines []string) int {
	for i, line := range lines {
		if strings.TrimSpace(line) == "import (" {
			return i
		}
	}
	return -1
}

// importGroupTransition reports a spec line's group and whether a blank precedes it.
//
// A blank line should precede the line on a group change from the previous spec. Non-spec
// lines keep the previous group and never introduce a blank.
//
// Takes line (string) which is the source line to classify.
// Takes prevGroup (int) which is the group of the previous spec, or -1 when none.
//
// Returns int which is the group the line belongs to.
// Returns bool which is true when a blank line should precede the line.
func importGroupTransition(line string, prevGroup int) (group int, blankBefore bool) {
	path, ok := importPathFromSpecLine(strings.TrimSpace(line))
	if !ok {
		return prevGroup, false
	}
	group = importGroup(path)
	return group, prevGroup != -1 && group != prevGroup
}

// importPathFromSpecLine extracts the import path from a single import-spec line.
//
// The path is the last double-quoted token on a line such as `"fmt"` or `alias
// "example.com/pkg"`.
//
// Takes line (string) which is the import-spec line to parse.
//
// Returns string which is the extracted import path, or "" when none is found.
// Returns bool which is true when an import path was found.
func importPathFromSpecLine(line string) (string, bool) {
	end := strings.LastIndexByte(line, '"')
	if end <= 0 {
		return "", false
	}
	start := strings.LastIndexByte(line[:end], '"')
	if start < 0 {
		return "", false
	}
	return line[start+1 : end], true
}
