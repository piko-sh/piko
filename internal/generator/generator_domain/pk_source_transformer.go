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
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	parsejs "github.com/tdewolff/parse/v2/js"

	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/jsscan"
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
	topLevelTokenHandlers = map[parsejs.TokenType]func(*jsscan.Scanner) *exportedFunctionInfo{ //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
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
// A plain module, an empty source and a TypeScript module declaring exported interfaces
// are returned unchanged. A script that neither uses pk or refs nor declares top-level
// functions receives imports only. Undetected function declarations produce a warning.
//
// A page script gets one instance per page visit, scoped to the page root. A partial
// script gets one instance per root element of the partial.
//
// Takes source (string) which is the PK script source code to change.
// Takes target (generator_dto.ClientScriptTarget) which says whether the script belongs
// to a page, a partial or a plain module, and names the partial.
//
// Returns string which is the changed source with scoped ref support.
// Returns error when the target is not valid.
func TransformPKSource(ctx context.Context, source string, target generator_dto.ClientScriptTarget) (string, error) {
	if err := target.Validate(); err != nil {
		return "", fmt.Errorf("transforming PK script: %w", err)
	}
	if isLeftUnchanged(source, target) {
		return source, nil
	}

	scriptTarget := newPKScriptTarget(target)
	topLevelFunctions := findTopLevelFunctions(source)
	warnOnUndetectedFunctions(ctx, source, scriptTarget.label(), topLevelFunctions)

	if len(topLevelFunctions) > 0 {
		return buildTransformedSource(source, topLevelFunctions, detectUsedIdentifiersExcludingRefs(source), scriptTarget), nil
	}
	if sourceUsesRefs(source) {
		return scriptTarget.transformWithoutFunctions(source), nil
	}
	return addImportsOnly(source), nil
}

// isLeftUnchanged reports whether a script is used as written. That is the case for a
// plain module, an empty source, or a TypeScript module that declares exported
// interfaces.
//
// Takes source (string) which is the script source.
// Takes target (generator_dto.ClientScriptTarget) which says what the script belongs to.
//
// Returns bool which is true when the script needs no transformation.
func isLeftUnchanged(source string, target generator_dto.ClientScriptTarget) bool {
	return target.Kind == generator_dto.ClientScriptModule || source == "" || isTypeScriptModuleFile(source)
}

// warnOnUndetectedFunctions warns about top-level functions missed by the script scanner.
//
// Takes source (string) which is the PK script source that was scanned.
// Takes component (string) which names the script's component in the warning.
// Takes found ([]exportedFunctionInfo) which holds the functions the scanner found.
func warnOnUndetectedFunctions(ctx context.Context, source string, component string, found []exportedFunctionInfo) {
	missing := undetectedFunctionNames(source, found)
	if len(missing) == 0 {
		return
	}

	_, l := logger_domain.From(ctx, log)
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
// Takes scanner (*jsscan.Scanner) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which holds the function info with wasExported set to
// true, or nil if no exported function was found.
func scanExportedFunctionWithFlag(scanner *jsscan.Scanner) *exportedFunctionInfo {
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
	scanner := jsscan.New(source)

	var functions []exportedFunctionInfo

	for {
		tt, _ := scanner.Next()
		if tt == parsejs.ErrorToken {
			break
		}

		if isBraceToken(tt) || scanner.BraceDepth() > 0 {
			continue
		}

		if function := processTopLevelToken(scanner, tt); function != nil {
			functions = append(functions, *function)
		}
	}

	return functions
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
// Takes scanner (*jsscan.Scanner) which provides the token stream to process.
// Takes tt (parsejs.TokenType) which specifies the token type to handle.
//
// Returns *exportedFunctionInfo which holds the parsed function details, or nil if no
// handler exists for the token type.
func processTopLevelToken(scanner *jsscan.Scanner, tt parsejs.TokenType) *exportedFunctionInfo {
	handler, ok := topLevelTokenHandlers[tt]
	if !ok {
		return nil
	}
	return handler(scanner)
}

// scanExportedFunction scans tokens after an export keyword to check if it is a function
// export and extract the function name.
//
// Takes scanner (*jsscan.Scanner) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which contains the function name and whether it is an
// arrow function.
// Returns nil if no function export was found.
func scanExportedFunction(scanner *jsscan.Scanner) *exportedFunctionInfo {
	tt, _ := scanner.Next()

	if tt == parsejs.AsyncToken {
		tt, _ = scanner.Next()
	}

	switch tt { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
	case parsejs.FunctionToken:
		tt, data := scanner.Next()
		if tt == parsejs.IdentifierToken {
			name := string(data)
			return &exportedFunctionInfo{
				name:        name,
				isArrow:     false,
				wasExported: false,
			}
		}

	case parsejs.ConstToken:
		tt, data := scanner.Next()
		if tt != parsejs.IdentifierToken {
			return nil
		}
		name := string(data)

		tt, _ = scanner.Next()
		if tt != parsejs.EqToken {
			return nil
		}

		tt, _ = scanner.Next()

		if tt == parsejs.AsyncToken {
			tt, _ = scanner.Next()
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
// Takes scanner (*jsscan.Scanner) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which holds the function name, or nil if no function name
// is found.
func scanNonExportedFunction(scanner *jsscan.Scanner) *exportedFunctionInfo {
	tt, data := scanner.Next()
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
// Takes scanner (*jsscan.Scanner) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which contains the function name, or nil if no arrow
// function pattern was found.
func scanNonExportedArrowFunction(scanner *jsscan.Scanner) *exportedFunctionInfo {
	tt, data := scanner.Next()
	if tt != parsejs.IdentifierToken {
		return nil
	}
	name := string(data)

	tt, _ = scanner.Next()
	if tt != parsejs.EqToken {
		return nil
	}

	tt, _ = scanner.Next()

	if tt == parsejs.AsyncToken {
		tt, _ = scanner.Next()
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
// Takes scanner (*jsscan.Scanner) which provides the token stream to scan.
//
// Returns *exportedFunctionInfo which contains the function name, or nil if no async
// function pattern was found.
func scanNonExportedAsyncFunction(scanner *jsscan.Scanner) *exportedFunctionInfo {
	tt, _ := scanner.Next()
	if tt != parsejs.FunctionToken {
		return nil
	}
	return scanNonExportedFunction(scanner)
}

// skipGenericTypeParams skips over TypeScript generic type parameters such as <T> or <T
// extends Foo, U> and returns the next token after the closing >.
//
// Takes scanner (*jsscan.Scanner) which provides the token stream to read from.
//
// Returns parsejs.TokenType which is the next token after the closing angle bracket, or
// ErrorToken if parsing fails.
func skipGenericTypeParams(scanner *jsscan.Scanner) parsejs.TokenType {
	depth := 1
	for depth > 0 {
		tt, data := scanner.Next()
		if tt == parsejs.ErrorToken {
			return tt
		}
		depth += genericDepthChange(tt, data)
	}
	tt, _ := scanner.Next()
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

// buildInlineRefsSource wraps a partial script that uses refs or the pk context but has
// no top-level functions. It creates the pk context inline from the partial's first root
// element instead of using the factory.
//
// Module scripts execute after DOM parsing, so document.querySelector will find the
// partial container.
//
// Takes source (string) which contains the PK source code to wrap.
// Takes target (partialPKScript) which names the partial.
//
// Returns string which is the source with pk context creation added inline.
func buildInlineRefsSource(source string, target partialPKScript) string {
	usedIdentifiers := detectUsedIdentifiers(source)
	importStmt := buildImportStatement(append(usedIdentifiers, jsFuncCreatePKContext))
	builder := newPKTransformBuilder()

	var result strings.Builder
	result.WriteString(importStmt)
	result.WriteString("{" + jsNewline)
	result.WriteString(builder.buildContextDeclaration(target, target.scopeExpression(builder.ast)))
	result.WriteString(jsNewline)
	result.WriteString(source)
	result.WriteString(jsNewline + "}" + jsNewline)

	return result.String()
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
// pattern. It uses AST-based code generation to apply these changes.
//
// Takes source (string) which contains the original source code to transform.
// Takes functions ([]exportedFunctionInfo) which lists the top-level functions to wrap.
// Takes otherImports ([]string) which specifies extra imports to include.
// Takes target (pkScriptTarget) which shapes the code that differs between pages and
// partials.
//
// Returns string which is the transformed source, whose exported functions run on the
// instance of the current page visit or partial element.
func buildTransformedSource(source string, functions []exportedFunctionInfo, otherImports []string, target pkScriptTarget) string {
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

	imports := slices.Concat(target.runtimeImports(), otherImports)

	builder := newPKTransformBuilder()
	return builder.buildFullTransform(imports, functions, transformedSource, target)
}
