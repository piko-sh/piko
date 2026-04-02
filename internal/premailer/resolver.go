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

package premailer

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_lexer"
)

// maxResolveDepth is the maximum number of iterations for resolving CSS
// variables. This limit prevents infinite loops from circular references
// where variables refer to each other.
const maxResolveDepth = 20

// resolverContext holds state needed during CSS variable resolution.
// It groups related data together to reduce parameter counts.
type resolverContext struct {
	// theme maps CSS variable names to their resolved values.
	theme map[string]string

	// diagnostics collects warnings about unresolved or invalid CSS variables.
	diagnostics *[]*ast_domain.Diagnostic

	// sourcePath is the file path used when reporting errors.
	sourcePath string

	// symbols holds the symbol table used to resolve variable references.
	symbols []ast.Symbol

	// symbolMap provides symbol name lookup for turning tokens into strings.
	symbolMap ast.SymbolMap
}

// resolveCSSVariables resolves all var() functions in CSS tokens to their
// final static values using the theme map. This is critical for email
// compatibility as email clients cannot evaluate CSS variables at runtime.
//
// Resolution algorithm:
//  1. Scan tokens to find TFunction tokens with name "var"
//  2. Extract variable name and optional fallback from function arguments
//  3. Look up variable in theme map and substitute
//  4. Repeat until no more var() functions remain or max depth reached
//
// This iterative approach naturally handles nested variables because each
// substitution may introduce new var() calls to be resolved in the next
// iteration.
//
// Takes tokens ([]css_ast.Token) which are the esbuild CSS tokens to process.
// Takes symbols ([]ast.Symbol) which provides symbol information for lookups.
// Takes symbolMap (ast.SymbolMap) which maps symbol references.
// Takes theme (map[string]string) which contains CSS variable values.
// Takes diagnostics (*[]*ast_domain.Diagnostic) which collects any diagnostics.
// Takes sourcePath (string) which identifies the source file for errors.
//
// Returns []css_ast.Token which are the tokens with all var() functions
// resolved to static values.
func resolveCSSVariables(
	tokens []css_ast.Token,
	symbols []ast.Symbol,
	symbolMap ast.SymbolMap,
	theme map[string]string,
	diagnostics *[]*ast_domain.Diagnostic,
	sourcePath string,
) []css_ast.Token {
	ctx := resolverContext{
		symbols:     symbols,
		symbolMap:   symbolMap,
		theme:       theme,
		diagnostics: diagnostics,
		sourcePath:  sourcePath,
	}

	return resolveCSSVariablesWithResolverContext(tokens, &ctx)
}

// revive:enable:argument-limit

// resolveCSSVariablesWithContext resolves CSS variables using a
// declarationContext. This is a helper for internal callers that already have
// a declarationContext.
//
// Takes tokens ([]css_ast.Token) which contains the CSS tokens to process.
// Takes ctx (*declarationContext) which provides the context for resolution.
//
// Returns []css_ast.Token which contains the tokens with variables resolved.
func resolveCSSVariablesWithContext(tokens []css_ast.Token, ctx *declarationContext) []css_ast.Token {
	resolverCtx := resolverContext{
		symbols:     ctx.symbols,
		symbolMap:   ctx.symbolMap,
		theme:       ctx.options.Theme,
		diagnostics: ctx.diagnostics,
		sourcePath:  ctx.sourcePath,
	}

	return resolveCSSVariablesWithResolverContext(tokens, &resolverCtx)
}

// resolveCSSVariablesWithResolverContext replaces CSS variables with their
// values in a list of tokens.
//
// Takes tokens ([]css_ast.Token) which contains the CSS tokens to process.
// Takes ctx (*resolverContext) which provides variable definitions and state.
//
// Returns []css_ast.Token which contains the tokens with all variables
// replaced by their values.
func resolveCSSVariablesWithResolverContext(tokens []css_ast.Token, ctx *resolverContext) []css_ast.Token {
	if !shouldResolveVariables(tokens, ctx) {
		return tokens
	}

	return resolveVariablesIteratively(tokens, ctx)
}

// shouldResolveVariables checks whether variable resolution should be tried.
//
// Takes tokens ([]css_ast.Token) which contains the CSS tokens to check.
// Takes ctx (*resolverContext) which provides the theme variables.
//
// Returns bool which is true if the tokens contain var() functions and the
// theme has variables to resolve.
func shouldResolveVariables(tokens []css_ast.Token, ctx *resolverContext) bool {
	if len(ctx.theme) == 0 {
		return false
	}

	for _, token := range tokens {
		if token.Kind == css_lexer.TFunction && strings.EqualFold(token.Text, "var") {
			return true
		}
	}
	return false
}

// resolveVariablesIteratively replaces CSS variables in a loop until no more
// changes happen or the maximum depth is reached.
//
// Takes tokens ([]css_ast.Token) which contains the CSS tokens to process.
// Takes ctx (*resolverContext) which provides the resolution context.
//
// Returns []css_ast.Token which contains the processed tokens.
func resolveVariablesIteratively(tokens []css_ast.Token, ctx *resolverContext) []css_ast.Token {
	resolvedTokens := tokens

	for range maxResolveDepth {
		newTokens, changed := resolveVarFunctionsOnce(resolvedTokens, ctx)
		if !changed {
			return newTokens
		}
		resolvedTokens = newTokens
	}

	reportMaxDepthExceeded(tokens, resolvedTokens, ctx)
	return resolvedTokens
}

// reportMaxDepthExceeded creates a diagnostic when variable resolution
// exceeds the maximum depth limit.
//
// This is kept separate from the main resolution logic to make the code
// easier to maintain.
//
// Takes originalTokens ([]css_ast.Token) which contains the original CSS
// tokens before resolution.
// Takes partialTokens ([]css_ast.Token) which contains the tokens after
// partial resolution was tried.
// Takes ctx (*resolverContext) which provides the resolution context and
// collects diagnostics.
func reportMaxDepthExceeded(originalTokens, partialTokens []css_ast.Token, ctx *resolverContext) {
	originalValue := tokensToString(originalTokens, ctx.symbols, ctx.symbolMap)
	partialValue := tokensToString(partialTokens, ctx.symbols, ctx.symbolMap)

	*ctx.diagnostics = append(*ctx.diagnostics, ast_domain.NewDiagnostic(
		ast_domain.Warning,
		fmt.Sprintf("CSS variable resolution exceeded max depth of %d iterations. "+
			"This likely indicates a circular reference (e.g., 'var-a: var(--var-b); var-b: var(--var-a);'). "+
			"Original value: '%s'. Partially resolved value: '%s'",
			maxResolveDepth, originalValue, partialValue),
		originalValue,
		ast_domain.Location{},
		ctx.sourcePath,
	))
}

// resolveVarFunctionsOnce makes a single pass over the token stream to replace
// any var() functions it finds with their values.
//
// Takes tokens ([]css_ast.Token) which is the token stream to process.
// Takes ctx (*resolverContext) which holds variable values and state.
//
// Returns []css_ast.Token which is the token stream with var() functions
// replaced.
// Returns bool which is true if any changes were made.
func resolveVarFunctionsOnce(tokens []css_ast.Token, ctx *resolverContext) ([]css_ast.Token, bool) {
	var result []css_ast.Token
	changed := false

	for i := range len(tokens) {
		token := tokens[i]

		if isVarFunction(token) {
			processedTokens := processVarFunction(token, ctx, &changed)
			result = append(result, processedTokens...)
		} else {
			result = append(result, token)
		}
	}

	return result, changed
}

// isVarFunction checks whether a token is a CSS var() function call.
//
// Takes token (css_ast.Token) which is the CSS token to check.
//
// Returns bool which is true if the token is a var() function call.
func isVarFunction(token css_ast.Token) bool {
	return token.Kind == css_lexer.TFunction && strings.EqualFold(token.Text, "var")
}

// processVarFunction resolves a single var() function token.
//
// Takes token (css_ast.Token) which is the var() function token to
// resolve.
// Takes ctx (*resolverContext) which provides theme data and symbol
// mappings.
// Takes changed (*bool) which is set to true when any value is
// resolved.
//
// Returns []css_ast.Token which contains the resolved token values,
// or nil if the variable is not defined and has no fallback.
func processVarFunction(token css_ast.Token, ctx *resolverContext, changed *bool) []css_ast.Token {
	var argTokens []css_ast.Token
	if token.Children != nil {
		argTokens = *token.Children
	}

	varName, fallback := parseVarArguments(argTokens, ctx.symbols, ctx.symbolMap)

	if themeValue, ok := ctx.theme[varName]; ok {
		*changed = true
		return parseThemeValue(themeValue)
	}

	if fallback != nil {
		*changed = true
		return fallback
	}

	reportUndefinedVariable(varName, ctx)
	*changed = true
	return nil
}

// reportUndefinedVariable creates a warning for a CSS variable that is not
// defined. This keeps diagnostic creation separate from CSS variable handling.
//
// Takes varName (string) which is the CSS variable name without the -- prefix.
// Takes ctx (*resolverContext) which provides the diagnostics list and source
// path.
func reportUndefinedVariable(varName string, ctx *resolverContext) {
	*ctx.diagnostics = append(*ctx.diagnostics, ast_domain.NewDiagnostic(
		ast_domain.Warning,
		fmt.Sprintf("Undefined CSS variable '--%s' with no fallback value. This will likely cause rendering issues in email clients.", varName),
		fmt.Sprintf("var(--%s)", varName),
		ast_domain.Location{},
		ctx.sourcePath,
	))
}

// parseVarArguments extracts the variable name and optional
// fallback value from a var() function call.
//
// Arguments are in the form: --variable-name or
// --variable-name, fallback-value.
//
// Takes argTokens ([]css_ast.Token) which contains the tokens inside the
// var() function call.
//
// Returns varName (string) which is the extracted CSS variable name
// without the leading dashes.
// Returns fallback ([]css_ast.Token) which contains the fallback value
// tokens, or nil when no fallback is provided.
func parseVarArguments(argTokens []css_ast.Token, _ []ast.Symbol, _ ast.SymbolMap) (varName string, fallback []css_ast.Token) {
	if len(argTokens) == 0 {
		return "", nil
	}

	i := skipWhitespace(argTokens, 0)
	varName, i = parseVariableName(argTokens, i)

	fallback = parseFallbackValue(argTokens, i)

	return varName, fallback
}

// skipWhitespace moves past any whitespace tokens in the slice.
//
// Takes tokens ([]css_ast.Token) which is the slice of CSS tokens to scan.
// Takes startIndex (int) which is the position to start scanning from.
//
// Returns int which is the index of the first non-whitespace token, or the
// length of the slice if all remaining tokens are whitespace.
func skipWhitespace(tokens []css_ast.Token, startIndex int) int {
	i := startIndex
	for i < len(tokens) && tokens[i].Kind == css_lexer.TWhitespace {
		i++
	}
	return i
}

// parseVariableName extracts the CSS variable name from tokens. It handles two
// formats: separate dashes (TDelim("-") TDelim("-") TIdent("name")) and combined
// dashes (TIdent("--name")).
//
// Takes tokens ([]css_ast.Token) which contains the CSS tokens to parse.
// Takes startIndex (int) which specifies where to begin parsing.
//
// Returns varName (string) which is the variable name without leading dashes,
// or empty if no valid name is found.
// Returns nextIndex (int) which is the position after the parsed variable name.
func parseVariableName(tokens []css_ast.Token, startIndex int) (varName string, nextIndex int) {
	i := startIndex
	if i >= len(tokens) {
		return "", i
	}

	if hasSeparateDoubleDashPrefix(tokens, i) {
		i += 2

		i = skipWhitespace(tokens, i)

		if i < len(tokens) && tokens[i].Kind == css_lexer.TIdent {
			varName = tokens[i].Text
			i++
		}
		return varName, i
	}

	if tokens[i].Kind == css_lexer.TIdent {
		name := tokens[i].Text
		varName = strings.TrimPrefix(name, "--")
		i++
		return varName, i
	}

	return "", i
}

// hasSeparateDoubleDashPrefix checks if the tokens at the given position start
// with two separate dash tokens that form a "--" prefix.
//
// Takes tokens ([]css_ast.Token) which is the token slice to check.
// Takes startIndex (int) which is the position to start checking from.
//
// Returns bool which is true if two separate delimiter tokens at startIndex
// form a "--" prefix.
func hasSeparateDoubleDashPrefix(tokens []css_ast.Token, startIndex int) bool {
	if startIndex+2 >= len(tokens) {
		return false
	}

	return tokens[startIndex].Kind == css_lexer.TDelim &&
		tokens[startIndex].Text == literalDash &&
		tokens[startIndex+1].Kind == css_lexer.TDelim &&
		tokens[startIndex+1].Text == literalDash
}

// parseFallbackValue extracts the fallback value after a comma, if present.
//
// Takes tokens ([]css_ast.Token) which contains the CSS tokens to parse.
// Takes startIndex (int) which specifies where to begin searching.
//
// Returns []css_ast.Token which contains the fallback tokens, or nil if no
// comma is found.
func parseFallbackValue(tokens []css_ast.Token, startIndex int) []css_ast.Token {
	i := skipWhitespace(tokens, startIndex)

	if i >= len(tokens) || tokens[i].Kind != css_lexer.TComma {
		return nil
	}

	i++
	i = skipWhitespace(tokens, i)

	if i < len(tokens) {
		return tokens[i:]
	}

	return nil
}

// parseThemeValue parses a theme value string into CSS tokens.
// This is a simplified parser that creates tokens from a string value.
//
// When the value contains a var() function call, it parses the nested
// variable. Otherwise, it parses the value as regular CSS tokens.
//
// Takes value (string) which is the theme value to parse.
//
// Returns []css_ast.Token which contains the parsed CSS tokens, or nil if
// the value is empty.
func parseThemeValue(value string) []css_ast.Token {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	if strings.HasPrefix(value, "var(") && strings.HasSuffix(value, ")") {
		return parseNestedVarFunction(value)
	}

	return parseRegularThemeValue(value)
}

// parseNestedVarFunction parses a nested var() function call into tokens.
//
// Takes value (string) which contains the var() function string to parse.
//
// Returns []css_ast.Token which contains a single TFunction token with the
// parsed var() call and its arguments as children.
func parseNestedVarFunction(value string) []css_ast.Token {
	inner := value[varFunctionPrefix : len(value)-1]

	parts := strings.Split(inner, ",")

	return []css_ast.Token{
		{
			Kind:     css_lexer.TFunction,
			Text:     "var",
			Children: new(buildVarFunctionChildren(parts)),
		},
	}
}

// buildVarFunctionChildren builds the child tokens for a var() function.
//
// Takes parts ([]string) which contains the variable name and an optional
// fallback value.
//
// Returns []css_ast.Token which contains the tokens for the var() function.
func buildVarFunctionChildren(parts []string) []css_ast.Token {
	var children []css_ast.Token

	varName := strings.TrimSpace(parts[indexFirst])
	if strings.HasPrefix(varName, "--") {
		children = append(children,
			css_ast.Token{Kind: css_lexer.TDelim, Text: literalDash},
			css_ast.Token{Kind: css_lexer.TDelim, Text: literalDash},
			css_ast.Token{Kind: css_lexer.TIdent, Text: varName[2:]},
		)
	} else {
		children = append(children, css_ast.Token{Kind: css_lexer.TIdent, Text: varName})
	}

	if len(parts) > 1 {
		children = append(children, css_ast.Token{Kind: css_lexer.TComma, Text: ","})
		fallbackValue := strings.TrimSpace(parts[indexSecond])
		children = append(children, parseThemeValue(fallbackValue)...)
	}

	return children
}

// parseRegularThemeValue parses a standard CSS value (not a var reference) into
// tokens.
//
// Takes value (string) which contains the CSS value to parse.
//
// Returns []css_ast.Token which contains the parsed tokens with whitespace
// kept between words.
func parseRegularThemeValue(value string) []css_ast.Token {
	var tokens []css_ast.Token

	words := strings.Fields(value)
	for i, word := range words {
		if i > indexFirst {
			tokens = append(tokens, css_ast.Token{Kind: css_lexer.TWhitespace, Text: literalSpace})
		}

		tokens = append(tokens, parseThemeValueWord(word)...)
	}

	return tokens
}

// parseThemeValueWord parses a single word from a CSS value into tokens.
//
// Takes word (string) which is the CSS value word to parse.
//
// Returns []css_ast.Token which contains the parsed tokens for the word.
func parseThemeValueWord(word string) []css_ast.Token {
	if strings.HasPrefix(word, "#") {
		return []css_ast.Token{{Kind: css_lexer.THash, Text: word[1:]}}
	}

	if parenIndex := strings.Index(word, "("); parenIndex > 0 {
		return parseFunctionToken(word, parenIndex)
	}

	return []css_ast.Token{{Kind: css_lexer.TIdent, Text: word}}
}

// parseFunctionToken parses a CSS function such as rgb() or url() into tokens.
//
// Takes word (string) which is the CSS function text to parse.
// Takes parenIndex (int) which is the position of the opening parenthesis.
//
// Returns []css_ast.Token which contains the parsed function tokens.
func parseFunctionToken(word string, parenIndex int) []css_ast.Token {
	var tokens []css_ast.Token

	functionName := word[:parenIndex]
	tokens = append(tokens, css_ast.Token{Kind: css_lexer.TFunction, Text: functionName})

	rest := word[parenIndex+1:]
	if rest != "" && rest != ")" {
		tokens = append(tokens, css_ast.Token{Kind: css_lexer.TIdent, Text: strings.TrimSuffix(rest, ")")})
	}

	if strings.HasSuffix(word, ")") {
		tokens = append(tokens, css_ast.Token{Kind: css_lexer.TCloseParen, Text: ")"})
	}

	return tokens
}
