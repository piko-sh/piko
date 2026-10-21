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
	"slices"
	"strings"

	parsejs "github.com/tdewolff/parse/v2/js"

	"piko.sh/piko/internal/logger/logger_domain"
)

// exportedFunctionInfo holds details about a top-level function for code generation.
type exportedFunctionInfo struct {
	// name is the function name used when building export wrappers.
	name string

	// isArrow indicates whether the function uses arrow syntax (const name =).
	isArrow bool

	// wasExported indicates whether the original declaration had the export keyword.
	wasExported bool
}

var (
	// topLevelTokenHandlers maps token types to their scanning functions for top-level
	// function detection. Each handler returns a function info if found.
	topLevelTokenHandlers = map[parsejs.TokenType]func(*pkScriptScanner) *exportedFunctionInfo{ //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		parsejs.ExportToken:   scanExportedFunctionWithFlag,
		parsejs.FunctionToken: scanNonExportedFunction,
		parsejs.ConstToken:    scanNonExportedArrowFunction,
		parsejs.AsyncToken:    scanNonExportedAsyncFunction,
	}

	// pkIdentifiersWithoutRefs is the list of PK identifiers excluding refs/_createRefs.
	// User-facing helpers are now accessed via the global piko.* namespace.
	pkIdentifiersWithoutRefs = []string{
		"action",
	}

	// refsPattern matches the identifier "refs" as a whole word.
	refsPattern = regexp.MustCompile(`\brefs\b`)

	// pkContextPattern matches the "pk." prefix, indicating the source uses the pk context
	// object.
	pkContextPattern = regexp.MustCompile(`\bpk\.`)

	// declaredFunctionPattern matches unindented function declarations and captures the
	// function name.
	declaredFunctionPattern = regexp.MustCompile(
		`(?m)^(?:export[ \t]+)?(?:async[ \t]+)?function(?:[ \t]*\*[ \t]*|[ \t]+)([A-Za-z_$][\w$]*)[ \t]*[<(]`)
)

// TransformPKSource scopes script functions and refs to individual component instances.
//
// Empty sources and TypeScript modules declaring exported interfaces are unchanged.
// Sources without top-level functions receive imports only. Undetected function
// declarations produce a warning.
//
// Takes source (string) which is the PK script source code to change.
// Takes componentName (string) which is the partial_name attribute value (e.g.
// "modals/listing_lightbox") or empty for pages. When set, the eager init and __reinit__
// selectors target the specific partial element instead of matching the first
// [partial_name] element on the page.
//
// Returns string which is the changed source with scoped ref support.
func TransformPKSource(source string, componentName string) string {
	if source == "" {
		return source
	}

	if isTypeScriptModuleFile(source) {
		return source
	}

	topLevelFunctions := findTopLevelFunctions(source)
	warnOnUndetectedFunctions(log, source, componentName, topLevelFunctions)
	if len(topLevelFunctions) == 0 {
		if !sourceUsesRefs(source) {
			return addImportsOnly(source)
		}
		return buildInlineRefsSource(source, componentName)
	}

	usedIdentifiers := detectUsedIdentifiersExcludingRefs(source)

	return buildTransformedSource(source, topLevelFunctions, usedIdentifiers, componentName)
}

// warnOnUndetectedFunctions warns about top-level functions missed by the script scanner.
//
// Takes l (logger_domain.Logger) which receives the warning.
// Takes source (string) which is the PK script source that was scanned.
// Takes componentName (string) which is the partial_name of the component, or empty for a
// page.
// Takes found ([]exportedFunctionInfo) which holds the functions the scanner found.
func warnOnUndetectedFunctions(l logger_domain.Logger, source string, componentName string, found []exportedFunctionInfo) {
	missing := undetectedFunctionNames(source, found)
	if len(missing) == 0 {
		return
	}

	component := componentName
	if component == "" {
		component = "(page)"
	}

	l.Warn("PK script declares top-level functions the scanner did not detect, so p-on handlers naming them will not be bound",
		logger_domain.String("component", component),
		logger_domain.Strings("functions", missing))
}

// undetectedFunctionNames lists the functions a source plainly declares at the top level
// that are absent from the scanner's results.
//
// Takes source (string) which is the PK script source that was scanned.
// Takes found ([]exportedFunctionInfo) which holds the functions the scanner found.
//
// Returns []string which holds the missing function names in source order.
func undetectedFunctionNames(source string, found []exportedFunctionInfo) []string {
	var missing []string
	for _, name := range declaredTopLevelFunctionNames(source) {
		wasFound := slices.ContainsFunc(found, func(function exportedFunctionInfo) bool {
			return function.name == name
		})
		if !wasFound {
			missing = append(missing, name)
		}
	}
	return missing
}

// declaredTopLevelFunctionNames lists the functions a source plainly declares at the top
// level, judged by declarations that start a line with no indentation. It is a cheap
// cross-check on the scanner, not a replacement for it.
//
// Takes source (string) which is the PK script source to check.
//
// Returns []string which holds the declared function names in source order.
func declaredTopLevelFunctionNames(source string) []string {
	matches := declaredFunctionPattern.FindAllStringSubmatch(source, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, match[1])
	}
	return names
}

// isTypeScriptModuleFile reports whether the source appears to be a TypeScript module
// file with interface declarations. Such files should be transpiled directly without the
// PK source transformation.
//
// Takes source (string) which is the file content to check.
//
// Returns bool which is true if the source contains export interface declarations.
func isTypeScriptModuleFile(source string) bool {
	return strings.Contains(source, "export interface ")
}

// scanExportedFunctionWithFlag wraps scanExportedFunction and marks the result as
// exported.
//
// Takes scanner (*pkScriptScanner) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which holds the function info with wasExported set to
// true, or nil if no exported function was found.
func scanExportedFunctionWithFlag(scanner *pkScriptScanner) *exportedFunctionInfo {
	function := scanExportedFunction(scanner)
	if function != nil {
		function.wasExported = true
	}
	return function
}

// findTopLevelFunctions finds exported and unexported top-level JavaScript and TypeScript
// functions.
//
// Takes source (string) which contains the source code to parse.
//
// Returns []exportedFunctionInfo which contains details of each top-level function found,
// including both regular function declarations and arrow functions.
func findTopLevelFunctions(source string) []exportedFunctionInfo {
	scanner := newPKScriptScanner(source)

	var functions []exportedFunctionInfo

	for {
		tt, _ := scanner.next()
		if tt == parsejs.ErrorToken {
			break
		}

		if isBraceToken(tt) || scanner.braceDepth > 0 {
			continue
		}

		if function := processTopLevelToken(scanner, tt); function != nil {
			functions = append(functions, *function)
		}
	}

	return functions
}

// updateBraceDepth adjusts brace depth based on the token type.
//
// Takes tt (parsejs.TokenType) which is the token to check for braces.
// Takes depth (int) which is the current brace depth.
//
// Returns int which is the updated depth, incremented for open braces and decremented for
// close braces. It never drops below zero, so a stray close brace cannot make nested
// declarations look top-level.
func updateBraceDepth(tt parsejs.TokenType, depth int) int {
	switch tt {
	case parsejs.OpenBraceToken:
		return depth + 1
	case parsejs.CloseBraceToken:
		return max(depth-1, 0)
	default:
		return depth
	}
}

// isBraceToken reports whether the token is an open or close brace.
//
// Takes tt (parsejs.TokenType) which is the token type to check.
//
// Returns bool which is true if the token is a brace, false otherwise.
func isBraceToken(tt parsejs.TokenType) bool {
	return tt == parsejs.OpenBraceToken || tt == parsejs.CloseBraceToken
}

// processTopLevelToken finds and runs the right handler for a token type.
//
// Takes scanner (*pkScriptScanner) which provides the token stream to process.
// Takes tt (parsejs.TokenType) which specifies the token type to handle.
//
// Returns *exportedFunctionInfo which holds the parsed function details, or nil if no
// handler exists for the token type.
func processTopLevelToken(scanner *pkScriptScanner, tt parsejs.TokenType) *exportedFunctionInfo {
	handler, ok := topLevelTokenHandlers[tt]
	if !ok {
		return nil
	}
	return handler(scanner)
}

// scanExportedFunction scans tokens after an export keyword to check if it is a function
// export and extract the function name.
//
// Takes scanner (*pkScriptScanner) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which contains the function name and whether it is an
// arrow function.
// Returns nil if no function export was found.
func scanExportedFunction(scanner *pkScriptScanner) *exportedFunctionInfo {
	tt, _ := scanner.next()

	if tt == parsejs.AsyncToken {
		tt, _ = scanner.next()
	}

	switch tt { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
	case parsejs.FunctionToken:
		tt, data := scanner.next()
		if tt == parsejs.IdentifierToken {
			name := string(data)
			return &exportedFunctionInfo{
				name:        name,
				isArrow:     false,
				wasExported: false,
			}
		}

	case parsejs.ConstToken:
		tt, data := scanner.next()
		if tt != parsejs.IdentifierToken {
			return nil
		}
		name := string(data)

		tt, _ = scanner.next()
		if tt != parsejs.EqToken {
			return nil
		}

		tt, _ = scanner.next()

		if tt == parsejs.AsyncToken {
			tt, _ = scanner.next()
		}

		if tt == parsejs.LtToken {
			tt = skipGenericTypeParams(scanner)
		}

		if tt == parsejs.OpenParenToken || tt == parsejs.FunctionToken {
			return &exportedFunctionInfo{
				name:        name,
				isArrow:     true,
				wasExported: false,
			}
		}
	}

	return nil
}

// scanNonExportedFunction scans tokens after a function keyword at top level.
//
// Takes scanner (*pkScriptScanner) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which holds the function name, or nil if no function name
// is found.
func scanNonExportedFunction(scanner *pkScriptScanner) *exportedFunctionInfo {
	tt, data := scanner.next()
	if tt == parsejs.IdentifierToken {
		name := string(data)
		return &exportedFunctionInfo{
			name:        name,
			isArrow:     false,
			wasExported: false,
		}
	}
	return nil
}

// scanNonExportedArrowFunction scans for a const name = (...) => pattern at the top level
// of a file.
//
// Takes scanner (*pkScriptScanner) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which contains the function name, or nil if no arrow
// function pattern was found.
func scanNonExportedArrowFunction(scanner *pkScriptScanner) *exportedFunctionInfo {
	tt, data := scanner.next()
	if tt != parsejs.IdentifierToken {
		return nil
	}
	name := string(data)

	tt, _ = scanner.next()
	if tt != parsejs.EqToken {
		return nil
	}

	tt, _ = scanner.next()

	if tt == parsejs.AsyncToken {
		tt, _ = scanner.next()
	}

	if tt == parsejs.LtToken {
		tt = skipGenericTypeParams(scanner)
	}

	if tt == parsejs.OpenParenToken || tt == parsejs.FunctionToken {
		return &exportedFunctionInfo{
			name:        name,
			isArrow:     true,
			wasExported: false,
		}
	}
	return nil
}

// scanNonExportedAsyncFunction scans for a non-exported async function name at the top
// level of the token stream.
//
// Takes scanner (*pkScriptScanner) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which contains the function name, or nil if no async
// function pattern was found.
func scanNonExportedAsyncFunction(scanner *pkScriptScanner) *exportedFunctionInfo {
	tt, _ := scanner.next()
	if tt != parsejs.FunctionToken {
		return nil
	}
	return scanNonExportedFunction(scanner)
}

// skipGenericTypeParams skips over TypeScript generic type parameters such as <T> or <T
// extends Foo, U> and returns the next token after the closing >.
//
// Takes scanner (*pkScriptScanner) which provides the token stream to read from.
//
// Returns parsejs.TokenType which is the next token after the closing angle bracket, or
// ErrorToken if parsing fails.
func skipGenericTypeParams(scanner *pkScriptScanner) parsejs.TokenType {
	depth := 1
	for depth > 0 {
		tt, data := scanner.next()
		if tt == parsejs.ErrorToken {
			return tt
		}
		depth += genericDepthChange(tt, data)
	}
	tt, _ := scanner.next()
	return tt
}

// genericDepthChange returns how far a token moves the nesting depth of generic type
// parameters. The lexer reads adjacent angle brackets, such as the closers of
// Array<Array<T>>, as one shift token, which opens or closes a level per bracket.
//
// Takes tt (parsejs.TokenType) which is the token inside the type parameters.
// Takes data ([]byte) which holds the raw bytes of the token.
//
// Returns int which is the change in depth.
func genericDepthChange(tt parsejs.TokenType, data []byte) int {
	switch tt {
	case parsejs.LtToken, parsejs.LtLtToken:
		return len(data)
	case parsejs.GtToken, parsejs.GtGtToken, parsejs.GtGtGtToken:
		return -len(data)
	default:
		return 0
	}
}

// detectUsedIdentifiersExcludingRefs finds PK identifiers in source code, but does not
// include reference identifiers.
//
// Takes source (string) which contains the PK source code to scan.
//
// Returns []string which contains the identifiers found.
func detectUsedIdentifiersExcludingRefs(source string) []string {
	var used []string
	for _, id := range pkIdentifiersWithoutRefs {
		pattern := identifierPatterns[id]
		if pattern != nil && pattern.MatchString(source) {
			used = append(used, id)
		}
	}
	return used
}

// sourceUsesRefs reports whether the source code references the "refs" identifier or the
// "pk." context prefix, indicating it needs the scoped pk context factory wrapper.
//
// Takes source (string) which contains the PK source code to check.
//
// Returns bool which is true if refs or pk context is used in the source.
func sourceUsesRefs(source string) bool {
	return refsPattern.MatchString(source) || pkContextPattern.MatchString(source)
}

// buildInlineRefsSource wraps source code that uses refs or pk context but has no
// top-level functions, creating a pk context inline from the partial's scope element
// instead of the full factory/WeakMap pattern.
//
// Module scripts execute after DOM parsing, so document.querySelector will find the
// partial container.
//
// Takes source (string) which contains the PK source code to wrap.
// Takes componentName (string) which is the partial_name attribute value for targeted
// scope resolution, or empty for generic fallback.
//
// Returns string which is the source with pk context creation added inline.
func buildInlineRefsSource(source string, componentName string) string {
	usedIdentifiers := detectUsedIdentifiers(source)
	importStmt := buildImportStatement(append(usedIdentifiers, "_createPKContext"))

	selector := `"[partial_name]"`
	if componentName != "" {
		selector = `"[partial_name='` + componentName + `']"`
	}

	var builder strings.Builder
	builder.WriteString(importStmt)
	builder.WriteString("{\n")
	builder.WriteString(`const pk = _createPKContext(document.querySelector(` + selector + `) ?? document.body);`)
	builder.WriteString("\n")
	builder.WriteString(source)
	builder.WriteString("\n}\n")

	return builder.String()
}

// addImportsOnly adds import statements to source code that has no exports.
//
// Takes source (string) which contains the Go source code to process.
//
// Returns string which is the source with import statements added at the start.
func addImportsOnly(source string) string {
	usedIdentifiers := detectUsedIdentifiers(source)
	importStmt := buildImportStatement(usedIdentifiers)
	return importStmt + source
}

// buildTransformedSource creates the full transformed source code using the factory
// pattern and PageContext self-registration. It uses AST-based code generation to apply
// these changes.
//
// Takes source (string) which contains the original source code to transform.
// Takes functions ([]exportedFunctionInfo) which lists the top-level functions to wrap.
// Takes otherImports ([]string) which specifies extra imports to include.
// Takes componentName (string) which names the component in the transformed source.
//
// Returns string which is the transformed source with the factory pattern applied and
// self-registration with PageContext for p-on:* event binding.
func buildTransformedSource(source string, functions []exportedFunctionInfo, otherImports []string, componentName string) string {
	transformedSource := source
	for _, function := range functions {
		if !function.wasExported {
			continue
		}
		if function.isArrow {
			transformedSource = strings.Replace(transformedSource, "export const "+function.name, "const "+function.name, 1)
		} else {
			transformedSource = strings.Replace(transformedSource, "export function "+function.name, "function "+function.name, 1)
			transformedSource = strings.Replace(transformedSource, "export async function "+function.name, "async function "+function.name, 1)
		}
	}

	imports := make([]string, 0, 2+len(otherImports))
	imports = append(imports, "_createPKContext", "getGlobalPageContext")
	imports = append(imports, otherImports...)

	builder := newPKTransformBuilder()
	return builder.buildFullTransform(imports, functions, transformedSource, componentName)
}
