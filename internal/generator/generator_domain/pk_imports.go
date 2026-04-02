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

package generator_domain

import (
	"regexp"
)

const (
	// pkFrameworkPath is the import path for the PK runtime framework.
	pkFrameworkPath = "/_piko/dist/ppframework.core.es.js"

	// pkActionsGenPath is the import path for the generated actions file.
	pkActionsGenPath = "/_piko/assets/pk-js/pk/actions.gen.js"
)

var (
	// pkIdentifiers lists identifiers that may appear in PK source code and need
	// to be imported from the runtime.
	//
	// User-facing helpers (refs, navigate, bus, etc.) are no longer listed here
	// because they are accessed via the global piko.* namespace (e.g. piko.refs,
	// piko.nav.navigate()). Internal identifiers like _createRefs and
	// getGlobalPageContext are added automatically by the source transformer.
	pkIdentifiers = []string{
		"action",
	}

	// identifierPatterns caches compiled regex patterns for each identifier.
	// Uses word boundaries to avoid false positives (e.g., "preferences"
	// should not match "refs").
	identifierPatterns = make(map[string]*regexp.Regexp)
)

// prepareSourceWithImports scans the source for PK runtime identifiers and
// adds the matching import statement at the start.
//
// The action namespace is always imported from pk/actions.gen.js so that
// typed actions like action.echo.message() are available.
//
// Takes source (string) which is the source code to scan.
//
// Returns string which is the source with the import statements added.
func prepareSourceWithImports(source string) string {
	if source == "" {
		return source
	}

	usedIdentifiers := detectUsedIdentifiers(source)
	importStmt := buildImportStatement(usedIdentifiers)
	return importStmt + source
}

// detectUsedIdentifiers scans source code for PK runtime identifiers.
//
// Takes source (string) which contains the source code to scan.
//
// Returns []string which contains the PK identifiers found in the source.
func detectUsedIdentifiers(source string) []string {
	var used []string
	for _, id := range pkIdentifiers {
		pattern := identifierPatterns[id]
		if pattern.MatchString(source) {
			used = append(used, id)
		}
	}
	return used
}

// buildImportStatement creates ES module import statements for the given
// identifiers using AST-based code generation.
//
// The function always imports `action` from the generated actions file
// (pk/actions.gen.js) so that the typed action namespace is available.
// Other identifiers are imported from the PK framework.
//
// When the identifiers slice is empty, the function still returns the action
// import.
//
// Takes identifiers ([]string) which lists the symbols to import from the
// PK framework.
//
// Returns string which contains the formatted import statements.
func buildImportStatement(identifiers []string) string {
	ast := newJSASTBuilder()
	var result string

	actionImport := ast.newImport([]string{"action"}, pkActionsGenPath)
	result += ast.renderStmt(actionImport) + "\n"

	var frameworkIdentifiers []string
	for _, id := range identifiers {
		if id != "action" {
			frameworkIdentifiers = append(frameworkIdentifiers, id)
		}
	}

	if len(frameworkIdentifiers) > 0 {
		frameworkImport := ast.newImport(frameworkIdentifiers, pkFrameworkPath)
		result += ast.renderStmt(frameworkImport) + "\n"
	}

	return result
}

func init() {
	for _, id := range pkIdentifiers {
		identifierPatterns[id] = regexp.MustCompile(`\b` + regexp.QuoteMeta(id) + `\b`)
	}
}
