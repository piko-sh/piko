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
	"strings"

	"github.com/tdewolff/parse/v2"
	parsejs "github.com/tdewolff/parse/v2/js"
)

// exportedFunctionInfo holds details about a top-level function for code
// generation.
type exportedFunctionInfo struct {
	// name is the function name used when building export wrappers.
	name string

	// isArrow indicates whether the function uses arrow syntax (const name =).
	isArrow bool

	// wasExported indicates whether the original declaration had the export
	// keyword.
	wasExported bool
}

var (
	// topLevelTokenHandlers maps token types to their scanning functions for
	// top-level function detection. Each handler returns a function info if found.
	topLevelTokenHandlers = map[parsejs.TokenType]func(*parsejs.Lexer) *exportedFunctionInfo{
		parsejs.ExportToken:   scanExportedFunctionWithFlag,
		parsejs.FunctionToken: scanNonExportedFunction,
		parsejs.ConstToken:    scanNonExportedArrowFunction,
		parsejs.AsyncToken:    scanNonExportedAsyncFunction,
	}

	// pkIdentifiersWithoutRefs is the list of PK identifiers excluding
	// refs/_createRefs. User-facing helpers are now accessed via the global piko.*
	// namespace.
	pkIdentifiersWithoutRefs = []string{
		"action",
	}

	// refsPattern matches the identifier "refs" as a whole word.
	refsPattern = regexp.MustCompile(`\brefs\b`)

	// pkContextPattern matches the "pk." prefix, indicating the source uses the
	// pk context object.
	pkContextPattern = regexp.MustCompile(`\bpk\.`)
)

// TransformPKSource changes PK script source to support scoped refs.
//
// The change process:
//  1. Wraps all top-level functions in a factory that creates scoped refs.
//  2. Creates wrapper exports that route to the correct scoped instance.
//  3. Uses WeakMap to store instances for each scope element.
//
// This lets users write simple code like:
//
//	function handleClick(event) {
//	    refs.button.click();
//	}
//
// Or with export (backwards compatible):
//
//	export function handleClick(event) {
//	    refs.button.click();
//	}
//
// And have it work correctly with multiple partial instances, each with their
// own scoped refs.
//
// When the source is empty, returns it unchanged.
//
// When the source is a TypeScript module file with export interface
// declarations, returns it unchanged. These ES modules should be transpiled
// directly without the scoped refs change.
//
// When no top-level functions are found, only adds the required imports.
//
// Takes source (string) which is the PK script source code to change.
// Takes componentName (string) which is the partial_name attribute value
// (e.g. "modals/listing_lightbox") or empty for pages. When set, the eager
// init and __reinit__ selectors target the specific partial element instead
// of matching the first [partial_name] element on the page.
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
	if len(topLevelFunctions) == 0 {
		if !sourceUsesRefs(source) {
			return addImportsOnly(source)
		}
		return buildInlineRefsSource(source, componentName)
	}

	usedIdentifiers := detectUsedIdentifiersExcludingRefs(source)

	return buildTransformedSource(source, topLevelFunctions, usedIdentifiers, componentName)
}

// isTypeScriptModuleFile reports whether the source appears to be a TypeScript
// module file with interface declarations. Such files should be transpiled
// directly without the PK source transformation.
//
// Takes source (string) which is the file content to check.
//
// Returns bool which is true if the source contains export interface
// declarations.
func isTypeScriptModuleFile(source string) bool {
	return strings.Contains(source, "export interface ")
}

// scanExportedFunctionWithFlag wraps scanExportedFunction and marks the
// result as exported.
//
// Takes lexer (*parsejs.Lexer) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which holds the function info with wasExported
// set to true, or nil if no exported function was found.
func scanExportedFunctionWithFlag(lexer *parsejs.Lexer) *exportedFunctionInfo {
	function := scanExportedFunction(lexer)
	if function != nil {
		function.wasExported = true
	}
	return function
}

// findTopLevelFunctions extracts all top-level function declarations from source
// code using token scanning. Works with both JavaScript and TypeScript, unlike
// full AST parsing which fails on TypeScript type hints.
//
// The function finds both exported and non-exported functions, and tracks
// whether each function had an export keyword via the wasExported field.
//
// Takes source (string) which contains the source code to parse.
//
// Returns []exportedFunctionInfo which contains details of each top-level
// function found, including both regular function declarations and arrow
// functions.
func findTopLevelFunctions(source string) []exportedFunctionInfo {
	input := parse.NewInputString(source)
	lexer := parsejs.NewLexer(input)

	var functions []exportedFunctionInfo
	braceDepth := 0

	for {
		tt, _ := lexer.Next()
		if tt == parsejs.ErrorToken {
			break
		}

		braceDepth = updateBraceDepth(tt, braceDepth)
		if isBraceToken(tt) || braceDepth > 0 {
			continue
		}

		if function := processTopLevelToken(lexer, tt); function != nil {
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
// Returns int which is the updated depth, incremented for open braces and
// decremented for close braces.
func updateBraceDepth(tt parsejs.TokenType, depth int) int {
	switch tt {
	case parsejs.OpenBraceToken:
		return depth + 1
	case parsejs.CloseBraceToken:
		return depth - 1
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
// Takes lexer (*parsejs.Lexer) which provides the token stream to process.
// Takes tt (parsejs.TokenType) which specifies the token type to handle.
//
// Returns *exportedFunctionInfo which holds the parsed function details,
// or nil if no handler exists for the token type.
func processTopLevelToken(lexer *parsejs.Lexer, tt parsejs.TokenType) *exportedFunctionInfo {
	handler, ok := topLevelTokenHandlers[tt]
	if !ok {
		return nil
	}
	return handler(lexer)
}

// scanExportedFunction scans tokens after an export keyword to check if it is
// a function export and extract the function name.
//
// Takes lexer (*parsejs.Lexer) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which contains the function name and whether
// it is an arrow function. Returns nil if no function export was found.
func scanExportedFunction(lexer *parsejs.Lexer) *exportedFunctionInfo {
	tt, _ := skipWhitespace(lexer)

	if tt == parsejs.AsyncToken {
		tt, _ = skipWhitespace(lexer)
	}

	switch tt {
	case parsejs.FunctionToken:
		tt, data := skipWhitespace(lexer)
		if tt == parsejs.IdentifierToken {
			name := string(data)
			return &exportedFunctionInfo{
				name:    name,
				isArrow: false,
			}
		}

	case parsejs.ConstToken:
		tt, data := skipWhitespace(lexer)
		if tt != parsejs.IdentifierToken {
			return nil
		}
		name := string(data)

		tt, _ = skipWhitespace(lexer)
		if tt != parsejs.EqToken {
			return nil
		}

		tt, _ = skipWhitespace(lexer)

		if tt == parsejs.AsyncToken {
			tt, _ = skipWhitespace(lexer)
		}

		if tt == parsejs.LtToken {
			tt = skipGenericTypeParams(lexer)
		}

		if tt == parsejs.OpenParenToken || tt == parsejs.FunctionToken {
			return &exportedFunctionInfo{
				name:    name,
				isArrow: true,
			}
		}
	}

	return nil
}

// scanNonExportedFunction scans tokens after a function keyword at top level.
//
// Takes lexer (*parsejs.Lexer) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which holds the function name, or nil if no
// function name is found.
func scanNonExportedFunction(lexer *parsejs.Lexer) *exportedFunctionInfo {
	tt, data := skipWhitespace(lexer)
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

// scanNonExportedArrowFunction scans for a const name = (...) => pattern at
// the top level of a file.
//
// Takes lexer (*parsejs.Lexer) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which contains the function name, or nil if no
// arrow function pattern was found.
func scanNonExportedArrowFunction(lexer *parsejs.Lexer) *exportedFunctionInfo {
	tt, data := skipWhitespace(lexer)
	if tt != parsejs.IdentifierToken {
		return nil
	}
	name := string(data)

	tt, _ = skipWhitespace(lexer)
	if tt != parsejs.EqToken {
		return nil
	}

	tt, _ = skipWhitespace(lexer)

	if tt == parsejs.AsyncToken {
		tt, _ = skipWhitespace(lexer)
	}

	if tt == parsejs.LtToken {
		tt = skipGenericTypeParams(lexer)
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

// scanNonExportedAsyncFunction scans for a non-exported async function name
// at the top level of the token stream.
//
// Takes lexer (*parsejs.Lexer) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which contains the function name, or nil if no
// async function pattern was found.
func scanNonExportedAsyncFunction(lexer *parsejs.Lexer) *exportedFunctionInfo {
	tt, _ := skipWhitespace(lexer)
	if tt != parsejs.FunctionToken {
		return nil
	}
	return scanNonExportedFunction(lexer)
}

// skipGenericTypeParams skips over TypeScript generic type parameters such as
// <T> or <T extends Foo, U> and returns the next token after the closing >.
//
// Takes lexer (*parsejs.Lexer) which provides the token stream to read from.
//
// Returns parsejs.TokenType which is the next token after the closing angle
// bracket, or ErrorToken if parsing fails.
func skipGenericTypeParams(lexer *parsejs.Lexer) parsejs.TokenType {
	depth := 1
	for depth > 0 {
		tt, _ := lexer.Next()
		switch tt {
		case parsejs.LtToken:
			depth++
		case parsejs.GtToken:
			depth--
		case parsejs.ErrorToken:
			return tt
		}
	}
	tt, _ := skipWhitespace(lexer)
	return tt
}

// skipWhitespace advances the lexer past whitespace and line terminator
// tokens, returning the next meaningful token.
//
// Takes lexer (*parsejs.Lexer) which provides the token stream to advance.
//
// Returns parsejs.TokenType which is the type of the next non-whitespace token.
// Returns []byte which contains the raw bytes of that token.
func skipWhitespace(lexer *parsejs.Lexer) (parsejs.TokenType, []byte) {
	for {
		tt, data := lexer.Next()
		if tt != parsejs.WhitespaceToken && tt != parsejs.LineTerminatorToken {
			return tt, data
		}
	}
}

// detectUsedIdentifiersExcludingRefs finds PK identifiers in source code,
// but does not include reference identifiers.
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

// sourceUsesRefs reports whether the source code references the "refs"
// identifier or the "pk." context prefix, indicating it needs the scoped
// pk context factory wrapper.
//
// Takes source (string) which contains the PK source code to check.
//
// Returns bool which is true if refs or pk context is used in the source.
func sourceUsesRefs(source string) bool {
	return refsPattern.MatchString(source) || pkContextPattern.MatchString(source)
}

// buildInlineRefsSource wraps source code that uses refs or pk context
// but has no top-level functions, creating a pk context inline from the
// partial's scope element instead of the full factory/WeakMap pattern.
//
// Module scripts execute after DOM parsing, so document.querySelector
// will find the partial container.
//
// Takes source (string) which contains the PK source code to wrap.
// Takes componentName (string) which is the partial_name attribute value
// for targeted scope resolution, or empty for generic fallback.
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
// Returns string which is the source with import statements added at the
// start.
func addImportsOnly(source string) string {
	usedIdentifiers := detectUsedIdentifiers(source)
	importStmt := buildImportStatement(usedIdentifiers)
	return importStmt + source
}

// buildTransformedSource creates the full transformed source code using the
// factory pattern and PageContext self-registration. It uses AST-based code
// generation to apply these changes.
//
// Takes source (string) which contains the original source code to transform.
// Takes functions ([]exportedFunctionInfo) which lists the top-level functions
// to wrap.
// Takes otherImports ([]string) which specifies extra imports to include.
//
// Returns string which is the transformed source with the factory pattern
// applied and self-registration with PageContext for p-on:* event binding.
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
