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

package compiler_domain

import (
	"strings"

	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/js_lexer"
	"piko.sh/piko/internal/esbuild/logger"
)

const (
	// doubleAngleBracketDepth is the amount to subtract from the angle bracket
	// depth when a ">>" token closes two brackets at once.
	doubleAngleBracketDepth = 2

	// tripleAngleBracketDepth is the depth value to subtract for a >>> token.
	tripleAngleBracketDepth = 3

	// unionTypeSeparator is the delimiter for union types in TypeScript.
	unionTypeSeparator = "|"
)

const (
	// tokenActionContinue tells the parser to keep reading the current value.
	tokenActionContinue tokenAction = iota

	// tokenActionIncDepth signals that a bracket depth counter should increase.
	tokenActionIncDepth

	// tokenActionDecDepth decreases the nesting depth when a closing bracket is found.
	tokenActionDecDepth

	// tokenActionEndProperty signals that a property value has ended.
	tokenActionEndProperty
)

// TypeAssertion represents a TypeScript type assertion discovered in source code.
type TypeAssertion struct {
	// PropertyName is the property name used in the type assertion expression.
	PropertyName string

	// TypeString is the raw type annotation from the source code.
	TypeString string

	// Line is the source line number where the type assertion occurs.
	Line int

	// Column is the column number within the line.
	Column int
}

// tokenAction represents what to do with a token after it is classified.
type tokenAction int

// typeTokenState tracks position and nesting depth during type token extraction.
type typeTokenState struct {
	// tokens holds the type tokens that are joined to form the final type string.
	tokens []string

	// angleDepth tracks how deeply nested the parser is inside angle brackets.
	angleDepth int

	// bracketDepth tracks the nesting level of square brackets; 0 means top level.
	bracketDepth int
}

// isAtTopLevel reports whether the parser is at the top level of a type.
//
// Returns bool which is true when not inside angle brackets or square brackets.
func (s *typeTokenState) isAtTopLevel() bool {
	return s.angleDepth == 0 && s.bracketDepth == 0
}

// genericTypeInfo holds the parsed details of a generic type.
type genericTypeInfo struct {
	// JSType is the JavaScript type name for this generic type.
	JSType string

	// ElemType is the element type for slices, arrays, or maps.
	ElemType string

	// KeyType is the type of map keys; empty for non-map types.
	KeyType string

	// ValueType is the type of map values; empty for non-map types.
	ValueType string

	// OK indicates whether the generic type was parsed correctly.
	OK bool
}

// ParsedTypeInfo holds the result of parsing a TypeScript type string.
type ParsedTypeInfo struct {
	// JSType is the JavaScript type name, such as "string", "number", or "Array".
	JSType string

	// ElementType is the type of elements when JSType is an array or slice.
	ElementType string

	// KeyType is the key type for map types; empty for non-map types.
	KeyType string

	// ValueType is the value type for map types.
	ValueType string

	// IsNullable indicates whether the type can hold a nil value.
	IsNullable bool
}

// ExtractTypeAssertions finds and extracts "as Type" assertions from the state
// object in TypeScript source code using esbuild's lexer.
//
// This function must run BEFORE esbuild parsing because esbuild removes all
// type information during parsing.
//
// The function uses a state machine approach to:
//  1. Find the "const state = {" declaration.
//  2. Track brace depth to stay within the state object.
//  3. For each property, detect the "propName: value as Type" pattern.
//  4. Extract type tokens, including nested generics, arrays, and unions.
//
// When the source contains strings with "as" inside them, the lexer handles
// them as string literals and ignores them. Comments containing "as" are also
// skipped by the lexer. Array literals in values like [1,2,3] do not affect
// type bracket detection. Nested generics like Map<string, Array<number>> and
// union types like User | nil are handled correctly.
//
// Takes source (string) which is the TypeScript source code to parse.
//
// Returns map[string]TypeAssertion which maps property names to their type
// assertion information.
func ExtractTypeAssertions(source string) map[string]TypeAssertion {
	assertions := make(map[string]TypeAssertion)

	log := logger.NewDeferLog(logger.DeferLogAll, nil)
	lexer := js_lexer.NewLexer(
		log,
		logger.Source{
			KeyPath:  logger.Path{Text: "extract.ts"},
			Contents: source,
		},
		config.TSOptions{},
	)

	if !seekToStateObject(&lexer) {
		return assertions
	}

	extractStateProperties(&lexer, assertions)

	return assertions
}

// ParseTypeString parses a TypeScript type string into structured metadata.
//
// Takes typeString (string) which is the TypeScript type to parse.
//
// Returns ParsedTypeInfo which holds the base type, element type for arrays,
// key and value types for maps, and whether the type is nullable.
func ParseTypeString(typeString string) ParsedTypeInfo {
	typeString = strings.TrimSpace(typeString)

	baseTypeString, isNullable := parseNullableType(typeString)

	if elemType, found := strings.CutSuffix(baseTypeString, "[]"); found {
		return ParsedTypeInfo{
			JSType:      "array",
			ElementType: strings.ToLower(strings.TrimSpace(elemType)),
			KeyType:     "",
			ValueType:   "",
			IsNullable:  isNullable,
		}
	}

	if info := parseGenericTypeString(baseTypeString); info.OK {
		return ParsedTypeInfo{
			JSType:      info.JSType,
			ElementType: info.ElemType,
			KeyType:     info.KeyType,
			ValueType:   info.ValueType,
			IsNullable:  isNullable,
		}
	}

	return ParsedTypeInfo{
		JSType:      strings.ToLower(extractBaseType(baseTypeString)),
		ElementType: "",
		KeyType:     "",
		ValueType:   "",
		IsNullable:  isNullable,
	}
}

// seekToStateObject moves the lexer forward to find the opening brace of a
// `const state = {` pattern.
//
// Takes lexer (*js_lexer.Lexer) which is the JavaScript lexer to move forward.
//
// Returns bool which is true if the pattern was found, or false if the end of
// the file was reached first.
func seekToStateObject(lexer *js_lexer.Lexer) bool {
	for lexer.Token != js_lexer.TEndOfFile {
		if !isConstStatePattern(lexer) {
			lexer.Next()
			continue
		}
		lexer.Next()
		return true
	}
	return false
}

// isConstStatePattern checks if the lexer is at a "const state = {" pattern
// and moves through it.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to check.
//
// Returns bool which is true if the pattern was found and consumed.
func isConstStatePattern(lexer *js_lexer.Lexer) bool {
	if lexer.Token != js_lexer.TConst {
		return false
	}
	lexer.Next()

	if lexer.Token != js_lexer.TIdentifier || lexer.Raw() != propState {
		return false
	}
	lexer.Next()

	if lexer.Token != js_lexer.TEquals {
		return false
	}
	lexer.Next()

	return lexer.Token == js_lexer.TOpenBrace
}

// extractStateProperties parses type assertions from state object properties.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to parse.
// Takes assertions (map[string]TypeAssertion) which stores the extracted type
// assertions.
func extractStateProperties(lexer *js_lexer.Lexer, assertions map[string]TypeAssertion) {
	braceDepth := 1

	for lexer.Token != js_lexer.TEndOfFile {
		switch lexer.Token {
		case js_lexer.TOpenBrace:
			braceDepth++
			lexer.Next()
			continue
		case js_lexer.TCloseBrace:
			braceDepth--
			if braceDepth == 0 {
				return
			}
			lexer.Next()
			continue
		default:
		}

		if braceDepth != 1 {
			lexer.Next()
			continue
		}

		extractPropertyAssertion(lexer, assertions)
	}
}

// extractPropertyAssertion reads a single property from the token stream and
// extracts any type assertion it contains.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to read from.
// Takes assertions (map[string]TypeAssertion) which stores the extracted type
// assertions keyed by property name.
func extractPropertyAssertion(lexer *js_lexer.Lexer, assertions map[string]TypeAssertion) {
	if lexer.Token != js_lexer.TIdentifier {
		lexer.Next()
		return
	}

	propName := lexer.Raw()
	lexer.Next()

	if lexer.Token != js_lexer.TColon {
		return
	}
	lexer.Next()

	foundAs := scanForTypeAssertion(lexer, propName, assertions)

	if !foundAs && lexer.Token == js_lexer.TComma {
		lexer.Next()
	}
}

// scanForTypeAssertion scans tokens to find an "as Type" pattern.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to scan.
// Takes propName (string) which is the property name being checked.
// Takes assertions (map[string]TypeAssertion) which stores any found assertions.
//
// Returns bool which is true if a type assertion was found.
func scanForTypeAssertion(lexer *js_lexer.Lexer, propName string, assertions map[string]TypeAssertion) bool {
	valueDepth := 0

	for {
		action := classifyValueToken(lexer.Token, valueDepth)

		switch action {
		case tokenActionIncDepth:
			valueDepth++
			lexer.Next()
			continue
		case tokenActionDecDepth:
			valueDepth--
			lexer.Next()
			continue
		case tokenActionEndProperty:
			return false
		default:
		}

		if valueDepth == 0 && isAsKeyword(lexer) {
			typeString := extractTypeTokens(lexer)
			assertions[propName] = TypeAssertion{
				PropertyName: propName,
				TypeString:   typeString,
				Line:         0,
				Column:       0,
			}
			return true
		}

		lexer.Next()
	}
}

// classifyValueToken decides what action to take for a token when scanning a
// value.
//
// Takes token (js_lexer.T) which is the token type to check.
// Takes valueDepth (int) which tracks nesting depth within brackets, braces,
// or parentheses.
//
// Returns tokenAction which shows whether to increase depth, decrease depth,
// end the property, or keep scanning.
func classifyValueToken(token js_lexer.T, valueDepth int) tokenAction {
	switch token {
	case js_lexer.TOpenBracket, js_lexer.TOpenBrace, js_lexer.TOpenParen:
		return tokenActionIncDepth
	case js_lexer.TCloseBracket, js_lexer.TCloseParen:
		if valueDepth > 0 {
			return tokenActionDecDepth
		}
		return tokenActionContinue
	case js_lexer.TCloseBrace:
		if valueDepth > 0 {
			return tokenActionDecDepth
		}
		return tokenActionEndProperty
	case js_lexer.TComma:
		if valueDepth == 0 {
			return tokenActionEndProperty
		}
		return tokenActionContinue
	default:
		return tokenActionContinue
	}
}

// isAsKeyword checks whether the lexer is at an "as" keyword.
//
// Takes lexer (*js_lexer.Lexer) which holds the current token state.
//
// Returns bool which is true when the current token is the "as" identifier.
func isAsKeyword(lexer *js_lexer.Lexer) bool {
	return lexer.Token == js_lexer.TIdentifier && lexer.Raw() == "as"
}

// extractTypeTokens reads tokens from a TypeScript type expression and joins
// them into a single string.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to read from.
//
// Returns string which is the combined type expression.
func extractTypeTokens(lexer *js_lexer.Lexer) string {
	state := &typeTokenState{}
	lexer.Next()

	for lexer.Token != js_lexer.TEndOfFile {
		if done := processTypeToken(lexer, state); done {
			break
		}
	}

	return strings.Join(state.tokens, "")
}

// processTypeToken handles a single token during type extraction.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to read.
// Takes state (*typeTokenState) which tracks the extraction progress.
//
// Returns bool which is true when extraction should stop.
func processTypeToken(lexer *js_lexer.Lexer, state *typeTokenState) bool {
	if handleAngleBrackets(lexer, state) {
		return false
	}

	if handleSquareBrackets(lexer, state) {
		return false
	}

	if state.isAtTopLevel() && isPropertyDelimiter(lexer.Token) {
		return true
	}

	if tokenString := collectTypeToken(lexer); tokenString != "" {
		state.tokens = append(state.tokens, tokenString)
	}

	lexer.Next()
	return false
}

// handleAngleBrackets processes angle bracket tokens in generic types.
// It handles single, double, and triple angle brackets, and updates the
// bracket depth and token list.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to read from.
// Takes state (*typeTokenState) which tracks bracket depth and collected tokens.
//
// Returns bool which is true if a bracket was handled.
func handleAngleBrackets(lexer *js_lexer.Lexer, state *typeTokenState) bool {
	if lexer.Token == js_lexer.TLessThan {
		state.angleDepth++
		state.tokens = append(state.tokens, "<")
		lexer.Next()
		return true
	}

	if lexer.Token == js_lexer.TGreaterThan {
		state.tokens = append(state.tokens, ">")
		state.angleDepth--
		lexer.Next()
		return true
	}

	if lexer.Raw() == ">>" {
		state.tokens = append(state.tokens, ">>")
		state.angleDepth -= doubleAngleBracketDepth
		lexer.Next()
		return true
	}

	if lexer.Raw() == ">>>" {
		state.tokens = append(state.tokens, ">>>")
		state.angleDepth -= tripleAngleBracketDepth
		lexer.Next()
		return true
	}

	return false
}

// handleSquareBrackets processes square bracket tokens for array types.
//
// Takes lexer (*js_lexer.Lexer) which provides the token stream to read from.
// Takes state (*typeTokenState) which tracks bracket depth and collected
// tokens.
//
// Returns bool which is true if a bracket was processed.
func handleSquareBrackets(lexer *js_lexer.Lexer, state *typeTokenState) bool {
	if lexer.Token == js_lexer.TOpenBracket {
		state.bracketDepth++
		state.tokens = append(state.tokens, "[")
		lexer.Next()
		return true
	}

	if lexer.Token == js_lexer.TCloseBracket {
		state.tokens = append(state.tokens, "]")
		state.bracketDepth--
		lexer.Next()
		return true
	}

	return false
}

// isPropertyDelimiter checks if a token marks the end of a property value.
//
// Takes token (js_lexer.T) which is the token to check.
//
// Returns bool which is true if the token is a comma, closing brace, or
// semicolon.
func isPropertyDelimiter(token js_lexer.T) bool {
	return token == js_lexer.TComma || token == js_lexer.TCloseBrace || token == js_lexer.TSemicolon
}

// collectTypeToken returns the raw token text for type-related tokens.
//
// Takes lexer (*js_lexer.Lexer) which provides the current token to check.
//
// Returns string which is the raw text for identifiers, a fixed string for
// type operators (|, &, ,), or an empty string for other tokens.
func collectTypeToken(lexer *js_lexer.Lexer) string {
	switch lexer.Token {
	case js_lexer.TIdentifier:
		return lexer.Raw()
	case js_lexer.TNull:
		return "null"
	case js_lexer.TBar:
		return "|"
	case js_lexer.TAmpersand:
		return "&"
	case js_lexer.TComma:
		return ","
	default:
		return ""
	}
}

// parseNullableType splits a union type string once and returns the base type
// with null/undefined removed, along with whether it was nullable.
//
// Takes typeString (string) which is the type expression to process.
//
// Returns string which is the type with null and undefined removed, or "any"
// if no types remain.
// Returns bool which is true if the type includes null or undefined.
func parseNullableType(typeString string) (string, bool) {
	parts := strings.Split(typeString, unionTypeSeparator)
	nonNullParts := make([]string, 0, len(parts))
	isNullable := false
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		lower := strings.ToLower(trimmed)
		if lower == "null" || lower == "undefined" {
			isNullable = true
			continue
		}
		nonNullParts = append(nonNullParts, trimmed)
	}
	if len(nonNullParts) == 0 {
		return "any", isNullable
	}
	return strings.Join(nonNullParts, " "+unionTypeSeparator+" "), isNullable
}

// parseGenericTypeString parses generic type patterns such as Array<T>,
// Map<K,V>, and Set<T>.
//
// Takes typeString (string) which contains the type string to parse.
//
// Returns genericTypeInfo which holds the parsed type details, with OK set to
// true when parsing succeeds, or an empty struct when the string is not a valid
// generic type.
func parseGenericTypeString(typeString string) genericTypeInfo {
	openIndex := strings.Index(typeString, "<")
	if openIndex < 0 {
		return genericTypeInfo{}
	}

	closeIndex := strings.LastIndex(typeString, ">")
	if closeIndex <= openIndex {
		return genericTypeInfo{}
	}

	baseName := strings.ToLower(strings.TrimSpace(typeString[:openIndex]))
	genericPart := typeString[openIndex+1 : closeIndex]

	switch baseName {
	case "array":
		return genericTypeInfo{JSType: "array", ElemType: strings.ToLower(strings.TrimSpace(genericPart)), KeyType: "", ValueType: "", OK: true}
	case "map":
		return parseMapGenericString(genericPart)
	case "set":
		return genericTypeInfo{JSType: "object", ElemType: strings.ToLower(strings.TrimSpace(genericPart)), KeyType: "", ValueType: "", OK: true}
	default:
		return genericTypeInfo{}
	}
}

// parseMapGenericString parses a Map<K, V> generic part into its key and value
// types.
//
// Takes genericPart (string) which contains the generic parameters, for example
// "String, Number" from "Map<String, Number>".
//
// Returns genericTypeInfo which holds the parsed key and value types, or an
// empty struct if the format is not valid.
func parseMapGenericString(genericPart string) genericTypeInfo {
	parts := strings.SplitN(genericPart, ",", 2)
	if len(parts) != 2 {
		return genericTypeInfo{}
	}
	return genericTypeInfo{
		JSType:    "object",
		ElemType:  "",
		KeyType:   strings.ToLower(strings.TrimSpace(parts[0])),
		ValueType: strings.ToLower(strings.TrimSpace(parts[1])),
		OK:        true,
	}
}

// extractBaseType returns the base type from a union or intersection type
// string.
//
// Takes typeString (string) which is the type string to extract the base from.
//
// Returns string which is the first type in the union or intersection, or the
// original string if no separator is found.
func extractBaseType(typeString string) string {
	if !strings.Contains(typeString, unionTypeSeparator) && !strings.Contains(typeString, "&") {
		return typeString
	}
	parts := strings.FieldsFunc(typeString, func(r rune) bool {
		return r == '|' || r == '&'
	})
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return typeString
}
