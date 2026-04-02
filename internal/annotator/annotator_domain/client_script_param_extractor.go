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

package annotator_domain

// Extracts TypeScript function parameter type annotations from client scripts
// using esbuild's lexer directly. This runs BEFORE the parser (which discards
// all type information) so that parameter types can be used for compile-time
// validation of event handler calls.

import (
	"strings"

	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/js_lexer"
	"piko.sh/piko/internal/esbuild/logger"
)

const (
	// categoryString represents the string type category.
	categoryString = "string"

	// categoryNumber represents the number type category.
	categoryNumber = "number"

	// categoryBoolean represents the boolean type category.
	categoryBoolean = "boolean"

	// categoryObject represents the object type category.
	categoryObject = "object"

	// categoryAny represents the any/unknown type category.
	categoryAny = "any"
)

// typeDepth tracks nesting depth of angle brackets, parentheses, and square
// brackets during type expression parsing.
type typeDepth struct {
	// angle tracks the nesting depth of angle brackets (<>).
	angle int

	// paren tracks the nesting depth of parentheses.
	paren int

	// bracket tracks the nesting depth of square brackets.
	bracket int
}

// isTopLevel reports whether the parser is at the top-level nesting depth.
//
// Returns bool which is true when no grouping tokens are open.
func (d *typeDepth) isTopLevel() bool {
	return d.angle == 0 && d.paren == 0 && d.bracket == 0
}

// handleGroupingToken checks whether the current token is a grouping delimiter
// (angle, paren, bracket) and adjusts depth accordingly. When the token is a
// top-level close-paren, it signals the caller to stop.
//
// Takes lexer (*js_lexer.Lexer) which provides the current token.
// Takes tokens (*[]string) which accumulates the type expression tokens.
//
// Returns handled (bool) which is true when the token was a grouping delimiter.
// Returns stop (bool) which is true when parsing should stop (unmatched close
// paren).
func (d *typeDepth) handleGroupingToken(lexer *js_lexer.Lexer, tokens *[]string) (handled, stop bool) {
	switch lexer.Token {
	case js_lexer.TLessThan:
		d.angle++
		*tokens = append(*tokens, "<")
		lexer.Next()
		return true, false
	case js_lexer.TGreaterThan:
		*tokens = append(*tokens, ">")
		d.angle--
		lexer.Next()
		return true, false
	case js_lexer.TGreaterThanGreaterThan:
		*tokens = append(*tokens, ">>")
		d.angle -= 2
		lexer.Next()
		return true, false
	case js_lexer.TGreaterThanGreaterThanGreaterThan:
		const tripleAngleCount = 3
		*tokens = append(*tokens, ">>>")
		d.angle -= tripleAngleCount
		lexer.Next()
		return true, false
	case js_lexer.TOpenParen:
		d.paren++
		*tokens = append(*tokens, "(")
		lexer.Next()
		return true, false
	case js_lexer.TCloseParen:
		if d.paren > 0 {
			d.paren--
			*tokens = append(*tokens, ")")
			lexer.Next()
			return true, false
		}
		return false, true
	case js_lexer.TOpenBracket:
		d.bracket++
		*tokens = append(*tokens, "[")
		lexer.Next()
		return true, false
	case js_lexer.TCloseBracket:
		d.bracket--
		*tokens = append(*tokens, "]")
		lexer.Next()
		return true, false
	default:
		return false, false
	}
}

// ExtractFunctionParams uses esbuild's lexer to find function declarations and
// arrow function assignments in TypeScript source code. For each function it
// extracts parameter names, type annotations, and optional/rest markers.
//
// This must run BEFORE esbuild parsing because esbuild removes all type
// information during parsing.
//
// Supported patterns:
//   - [export] function NAME(params) { ... }
//   - [export] async function NAME(params) { ... }
//   - [export] const NAME = (params) => { ... }
//   - [export] const NAME = async (params) => { ... }
//
// Takes source (string) which is the TypeScript source code to scan.
//
// Returns map[string][]ParamInfo which maps function names to their parameter
// lists. Functions without parameters are included with an empty slice.
func ExtractFunctionParams(source string) (result map[string][]ParamInfo) {
	if source == "" {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			if result == nil {
				result = make(map[string][]ParamInfo)
			}
		}
	}()

	result = make(map[string][]ParamInfo)

	log := logger.NewDeferLog(logger.DeferLogAll, nil)
	lexer := js_lexer.NewLexer(
		log,
		logger.Source{
			KeyPath:  logger.Path{Text: "params.ts"},
			Contents: source,
		},
		config.TSOptions{},
	)

	for lexer.Token != js_lexer.TEndOfFile {
		if lexer.Token == js_lexer.TExport {
			lexer.Next()
		}

		switch {
		case isFunctionKeyword(&lexer):
			extractFunctionDeclParams(&lexer, result)
		case isAsyncFunctionKeyword(&lexer):
			extractAsyncFunctionDeclParams(&lexer, result)
		case isConstKeyword(&lexer):
			extractConstArrowParams(&lexer, result)
		default:
			lexer.Next()
		}
	}

	return result
}

// isFunctionKeyword checks if the lexer is at a "function" keyword (not
// preceded by "async", which is handled separately).
//
// Takes lexer (*js_lexer.Lexer) which provides the current token to check.
//
// Returns bool which is true if the current token is a function keyword.
func isFunctionKeyword(lexer *js_lexer.Lexer) bool {
	return lexer.Token == js_lexer.TFunction
}

// isAsyncFunctionKeyword checks if the lexer is at "async" which may be
// followed by "function".
//
// Takes lexer (*js_lexer.Lexer) which provides the current token to check.
//
// Returns bool which is true if the current token is an "async" identifier.
func isAsyncFunctionKeyword(lexer *js_lexer.Lexer) bool {
	return lexer.Token == js_lexer.TIdentifier && lexer.Raw() == "async"
}

// isConstKeyword checks if the lexer is at a "const" keyword.
//
// Takes lexer (*js_lexer.Lexer) which provides the current token to check.
//
// Returns bool which is true if the current token is a const keyword.
func isConstKeyword(lexer *js_lexer.Lexer) bool {
	return lexer.Token == js_lexer.TConst
}

// extractFunctionDeclParams handles: function NAME(params) { ... }.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to read from.
// Takes result (map[string][]ParamInfo) which collects extracted function
// parameters keyed by function name.
func extractFunctionDeclParams(lexer *js_lexer.Lexer, result map[string][]ParamInfo) {
	lexer.Next()

	if lexer.Token == js_lexer.TIdentifier {
		name := lexer.Raw()
		lexer.Next()

		skipTypeParameters(lexer)

		if lexer.Token == js_lexer.TOpenParen {
			parameters := extractParamList(lexer)
			result[name] = parameters
		}
	}

	skipToNextStatement(lexer)
}

// extractAsyncFunctionDeclParams handles: async function NAME(params) { ... }.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to read from.
// Takes result (map[string][]ParamInfo) which collects extracted function
// parameters keyed by function name.
func extractAsyncFunctionDeclParams(lexer *js_lexer.Lexer, result map[string][]ParamInfo) {
	lexer.Next()

	if lexer.Token == js_lexer.TFunction {
		extractFunctionDeclParams(lexer, result)
		return
	}

	skipToNextStatement(lexer)
}

// extractConstArrowParams handles const arrow and async const
// arrow function patterns.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to read from.
// Takes result (map[string][]ParamInfo) which collects extracted function
// parameters keyed by function name.
func extractConstArrowParams(lexer *js_lexer.Lexer, result map[string][]ParamInfo) {
	lexer.Next()

	if lexer.Token != js_lexer.TIdentifier {
		return
	}

	name := lexer.Raw()
	lexer.Next()

	if lexer.Token != js_lexer.TEquals {
		return
	}
	lexer.Next()

	if lexer.Token == js_lexer.TIdentifier && lexer.Raw() == "async" {
		lexer.Next()
	}

	skipTypeParameters(lexer)

	if lexer.Token == js_lexer.TOpenParen {
		parameters := extractParamList(lexer)
		result[name] = parameters
	}
}

// extractParamList reads a complete parameter list from opening to closing
// parenthesis, extracting parameter names and type annotations.
//
// Takes lexer (*js_lexer.Lexer) which must be positioned at TOpenParen.
//
// Returns []ParamInfo which holds the extracted parameter information.
func extractParamList(lexer *js_lexer.Lexer) []ParamInfo {
	lexer.Next()

	var parameters []ParamInfo

	for lexer.Token != js_lexer.TCloseParen && lexer.Token != js_lexer.TEndOfFile {
		param := extractSingleParam(lexer)
		parameters = append(parameters, param)

		if lexer.Token == js_lexer.TComma {
			lexer.Next()
		}
	}

	if lexer.Token == js_lexer.TCloseParen {
		lexer.Next()
	}

	return parameters
}

// extractSingleParam reads one parameter declaration including optional rest
// operator, name, optional marker, and type annotation.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream positioned at
// the start of a parameter.
//
// Returns ParamInfo which contains the extracted parameter name, type, and
// flags.
func extractSingleParam(lexer *js_lexer.Lexer) ParamInfo {
	var param ParamInfo

	if lexer.Token == js_lexer.TDotDotDot {
		param.IsRest = true
		lexer.Next()
	}

	switch lexer.Token {
	case js_lexer.TOpenBrace, js_lexer.TOpenBracket:
		param.Name = "(destructured)"
		skipDestructurePattern(lexer)
	case js_lexer.TIdentifier:
		param.Name = lexer.Raw()
		lexer.Next()
	default:
		lexer.Next()
		param.Category = categoryAny
		return param
	}

	if lexer.Token == js_lexer.TQuestion {
		param.Optional = true
		lexer.Next()
	}

	if lexer.Token == js_lexer.TColon {
		lexer.Next()
		param.TypeName = extractParamType(lexer)
		param.Category = classifyTSType(param.TypeName)
	} else {
		param.Category = categoryAny
	}

	if lexer.Token == js_lexer.TEquals {
		param.Optional = true
		lexer.Next()
		skipDefaultValue(lexer)
	}

	return param
}

// extractParamType reads a TypeScript type expression from the lexer and
// returns it as a string. Stops at a top-level comma, closing paren, or
// equals sign (default value).
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream positioned
// after the colon of a type annotation.
//
// Returns string which is the reconstructed type expression text.
func extractParamType(lexer *js_lexer.Lexer) string {
	var tokens []string
	var depth typeDepth

	for lexer.Token != js_lexer.TEndOfFile {
		if handled, stop := depth.handleGroupingToken(lexer, &tokens); handled || stop {
			if stop {
				break
			}
			continue
		}

		if depth.isTopLevel() && (lexer.Token == js_lexer.TComma || lexer.Token == js_lexer.TEquals) {
			break
		}

		appendTypeToken(lexer, &tokens)
		lexer.Next()
	}

	return strings.Join(tokens, "")
}

// appendTypeToken appends the string representation of a non-grouping type
// token to the tokens slice.
//
// Takes lexer (*js_lexer.Lexer) which provides the current token.
// Takes tokens (*[]string) which accumulates the type expression tokens.
func appendTypeToken(lexer *js_lexer.Lexer, tokens *[]string) {
	switch lexer.Token {
	case js_lexer.TIdentifier, js_lexer.TStringLiteral, js_lexer.TNumericLiteral:
		*tokens = append(*tokens, lexer.Raw())
	case js_lexer.TNull:
		*tokens = append(*tokens, "null")
	case js_lexer.TBar:
		*tokens = append(*tokens, "|")
	case js_lexer.TAmpersand:
		*tokens = append(*tokens, "&")
	case js_lexer.TComma:
		*tokens = append(*tokens, ",")
	case js_lexer.TDot:
		*tokens = append(*tokens, ".")
	case js_lexer.TDotDotDot:
		*tokens = append(*tokens, "...")
	case js_lexer.TOpenBrace:
		*tokens = append(*tokens, "{")
	case js_lexer.TCloseBrace:
		*tokens = append(*tokens, "}")
	case js_lexer.TColon:
		*tokens = append(*tokens, ":")
	case js_lexer.TQuestion:
		*tokens = append(*tokens, "?")
	default:
	}
}

// skipTypeParameters skips past generic type parameters like <T> or <T, U>.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to advance
// past any generic type parameter list.
func skipTypeParameters(lexer *js_lexer.Lexer) {
	if lexer.Token != js_lexer.TLessThan {
		return
	}

	depth := 1
	lexer.Next()

	for depth > 0 && lexer.Token != js_lexer.TEndOfFile {
		switch lexer.Token {
		case js_lexer.TLessThan:
			depth++
		case js_lexer.TGreaterThan:
			depth--
		default:
		}
		lexer.Next()
	}
}

// skipDestructurePattern skips past a destructured parameter pattern like
// { a, b } or [ x, y ].
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream positioned at
// the opening brace or bracket of a destructure pattern.
func skipDestructurePattern(lexer *js_lexer.Lexer) {
	var openTok, closeTok js_lexer.T
	if lexer.Token == js_lexer.TOpenBrace {
		openTok = js_lexer.TOpenBrace
		closeTok = js_lexer.TCloseBrace
	} else {
		openTok = js_lexer.TOpenBracket
		closeTok = js_lexer.TCloseBracket
	}

	depth := 1
	lexer.Next()

	for depth > 0 && lexer.Token != js_lexer.TEndOfFile {
		switch lexer.Token {
		case openTok:
			depth++
		case closeTok:
			depth--
		}
		if depth > 0 {
			lexer.Next()
		}
	}

	if lexer.Token == closeTok {
		lexer.Next()
	}
}

// skipDefaultValue skips past a default value expression in a parameter.
// Stops at a top-level comma or closing paren.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream positioned
// after the equals sign of a default value.
func skipDefaultValue(lexer *js_lexer.Lexer) {
	depth := 0

	for lexer.Token != js_lexer.TEndOfFile {
		switch lexer.Token {
		case js_lexer.TOpenParen, js_lexer.TOpenBrace, js_lexer.TOpenBracket:
			depth++
		case js_lexer.TCloseParen:
			if depth == 0 {
				return
			}
			depth--
		case js_lexer.TCloseBrace, js_lexer.TCloseBracket:
			if depth > 0 {
				depth--
			}
		case js_lexer.TComma:
			if depth == 0 {
				return
			}
		default:
		}
		lexer.Next()
	}
}

// skipToNextStatement advances the lexer past the current statement by
// skipping until a matching closing brace at the top level or end of file.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to advance
// past the current statement.
func skipToNextStatement(lexer *js_lexer.Lexer) {
	depth := 0

	for lexer.Token != js_lexer.TEndOfFile {
		switch lexer.Token {
		case js_lexer.TOpenBrace:
			depth++
		case js_lexer.TCloseBrace:
			if depth <= 0 {
				break
			}
			depth--
			if depth == 0 {
				lexer.Next()
				return
			}
		case js_lexer.TSemicolon:
			if depth == 0 {
				lexer.Next()
				return
			}
		default:
		}
		lexer.Next()
	}
}

// classifyTSType maps a raw TypeScript type string to a simplified
// category, classifying nullable types by their non-null part.
//
// Takes rawType (string) which is the TypeScript type annotation.
//
// Returns string which is one of: "string", "number", "boolean", "object",
// or "any".
func classifyTSType(rawType string) string {
	if rawType == "" {
		return categoryAny
	}

	base := stripNullableFromType(rawType)
	if base == "" {
		return categoryAny
	}

	lower := strings.ToLower(base)

	switch lower {
	case "string":
		return categoryString
	case "number", "bigint":
		return categoryNumber
	case "boolean", "bool":
		return categoryBoolean
	case "any", "unknown", "void", "never", "undefined", "null":
		return categoryAny
	default:
		return categoryObject
	}
}

// stripNullableFromType removes "null" and "undefined" from a union type
// string and returns the remaining base type.
//
// Takes typeString (string) which is the type string that may contain nullable
// union members.
//
// Returns string which is the base type with null and undefined removed.
func stripNullableFromType(typeString string) string {
	if !strings.Contains(typeString, "|") {
		return typeString
	}

	for part := range strings.SplitSeq(typeString, "|") {
		trimmed := strings.TrimSpace(part)
		lower := strings.ToLower(trimmed)
		if lower != "null" && lower != "undefined" && trimmed != "" {
			return trimmed
		}
	}

	return ""
}
