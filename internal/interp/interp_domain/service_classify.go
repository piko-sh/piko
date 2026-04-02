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

package interp_domain

import "strings"

// classifiedLines holds source lines separated into imports,
// declarations, and executable statements for mixed-mode evaluation.
type classifiedLines struct {
	// imports holds import declaration lines.
	imports []string

	// decls holds top-level declaration lines (func, type, var, const).
	decls []string

	// statements holds executable statement lines.
	statements []string
}

// classifyLines separates source lines into imports, declarations,
// and executable statements.
//
// Takes lines ([]string) which are the source lines to classify.
//
// Returns classifiedLines with lines separated by category.
func classifyLines(lines []string) classifiedLines {
	var cl classifiedLines

	i := 0
	for i < len(lines) {
		trimmed := strings.TrimLeft(lines[i], " \t")

		if isImportLine(trimmed) {
			i = collectImportLines(lines, i, &cl.imports)
			continue
		}

		if isDeclarationLine(trimmed) {
			i = collectDeclarationLines(lines, i, &cl.decls)
			continue
		}

		cl.statements = append(cl.statements, lines[i])
		i++
	}
	return cl
}

// isImportLine reports whether a trimmed line starts an import.
//
// Takes trimmed (string) which is the left-trimmed source line.
//
// Returns true when the line begins with an import keyword.
func isImportLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "import\t")
}

// isDeclarationLine reports whether a trimmed line starts a
// top-level declaration (func, type, var, const).
//
// Takes trimmed (string) which is the left-trimmed source line.
//
// Returns true when the line begins with a declaration keyword.
func isDeclarationLine(trimmed string) bool {
	isFunc := strings.HasPrefix(trimmed, "func ") && !strings.HasPrefix(trimmed, "func(")
	return isFunc ||
		strings.HasPrefix(trimmed, "type ") ||
		strings.HasPrefix(trimmed, "var ") ||
		strings.HasPrefix(trimmed, "const ")
}

// collectImportLines collects a single or grouped import starting at
// index i, appending lines to dst.
//
// Takes lines ([]string) which are all source lines.
// Takes i (int) which is the starting index of the import.
// Takes dst (*[]string) which is the destination slice for
// collected import lines.
//
// Returns int which is the new index after the collected import lines.
func collectImportLines(lines []string, i int, dst *[]string) int {
	if strings.Contains(lines[i], "(") {
		for i < len(lines) {
			*dst = append(*dst, lines[i])
			if strings.Contains(lines[i], ")") {
				i++
				break
			}
			i++
		}
		return i
	}
	*dst = append(*dst, lines[i])
	return i + 1
}

// collectDeclarationLines collects a brace-delimited declaration
// starting at index i, appending lines to dst.
//
// Takes lines ([]string) which are all source lines.
// Takes i (int) which is the starting index of the declaration.
// Takes dst (*[]string) which is the destination slice for
// collected declaration lines.
//
// Returns int which is the new index after the collected lines.
func collectDeclarationLines(lines []string, i int, dst *[]string) int {
	depth := 0
	start := i
	for i < len(lines) {
		for _, character := range lines[i] {
			switch character {
			case '{':
				depth++
			case '}':
				depth--
			}
		}
		i++
		if depth == 0 && i > start {
			break
		}
	}
	*dst = append(*dst, lines[start:i]...)
	return i
}

// buildMixedSource reconstructs a Go source file from classified lines,
// wrapping statements in a synthetic _eval_ function.
//
// Takes cl (classifiedLines) which holds the classified import,
// declaration, and statement lines.
//
// Returns string which is the reconstructed Go source file.
func buildMixedSource(cl classifiedLines) string {
	var src strings.Builder
	src.WriteString("package main\n")
	for _, l := range cl.imports {
		src.WriteString(l)
		src.WriteString(newlineSep)
	}
	for _, l := range cl.decls {
		src.WriteString(l)
		src.WriteString(newlineSep)
	}
	src.WriteString("func _eval_() {\n")
	for _, l := range cl.statements {
		src.WriteString(l)
		src.WriteString(newlineSep)
	}
	src.WriteString("}\n")
	return src.String()
}
