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

package jsscan

import (
	"strings"

	parsejs "github.com/tdewolff/parse/v2/js"
)

const (
	// minimumQuotedLength is the length of the shortest quoted literal, a pair of quotes.
	minimumQuotedLength = 2
)

// Token is one significant token of a script.
type Token struct {
	// Text is the source text of the token.
	Text string

	// Start is the byte offset in the source where the token begins.
	Start int

	// Type is the kind of token.
	Type parsejs.TokenType
}

// Literal is a string literal, or a property name, read from a script.
type Literal struct {
	// Value is the text of the literal without its quotes.
	Value string

	// Start is the byte offset in the source where the literal's token begins.
	Start int

	// Length is the number of bytes the literal's token takes in the source.
	Length int
}

// PropertyRead is a property of an object path that a script names with a literal, such
// as name in pk.refs.name or pk.refs["name"].
type PropertyRead struct {
	Literal

	// Assigned is true when the script assigns to the property, as in pk.refs.name = x.
	Assigned bool
}

// Tokens reads the significant tokens of a script.
//
// Takes source (string) which is the script to read.
// Takes limit (int) which is the most tokens to read.
//
// Returns []Token which holds the tokens in source order.
// Returns bool which is true when the script has more tokens than limit, so the result
// covers only its start.
func Tokens(source string, limit int) ([]Token, bool) {
	scanner := New(source)
	var tokens []Token
	for {
		tokenType, data := scanner.Next()
		if tokenType == parsejs.ErrorToken {
			return tokens, false
		}
		if len(tokens) >= limit {
			return tokens, true
		}
		start := scanner.TokenStart()
		tokens = append(tokens, Token{Text: source[start : start+len(data)], Start: start, Type: tokenType})
	}
}

// FindPropertyReads finds the properties of an object path that a script names with a
// literal, as in path.name, path?.name, path["name"] and path?.["name"]. A computed name,
// such as path["a" + b], is skipped, and so is a path that is itself a member of another
// object, such as other.path.name.
//
// Takes tokens ([]Token) which are the script's tokens.
// Takes path ([]string) which names the object, such as ["pk", "refs"].
//
// Returns []PropertyRead which holds the properties in source order.
func FindPropertyReads(tokens []Token, path ...string) []PropertyRead {
	var reads []PropertyRead
	for index := range tokens {
		next, found := matchPath(tokens, index, path)
		if !found {
			continue
		}
		if read, ok := propertyAfterPath(tokens, next); ok {
			reads = append(reads, read)
		}
	}
	return reads
}

// FindCallArguments finds the calls of a function path whose first argument is a string
// literal, such as piko.partial("name"), and returns those literals. A call whose first
// argument is computed, such as piko.partial("a" + b), is skipped.
//
// Takes tokens ([]Token) which are the script's tokens.
// Takes path ([]string) which names the function, such as ["piko", "partial"].
//
// Returns []Literal which holds the first arguments in source order.
func FindCallArguments(tokens []Token, path ...string) []Literal {
	var literals []Literal
	for index := range tokens {
		next, found := matchPath(tokens, index, path)
		if !found || !isTokenAt(tokens, next, parsejs.OpenParenToken) {
			continue
		}
		literal, ok := stringLiteralAt(tokens, next+1)
		if ok && isArgumentEnd(tokens, next+2) {
			literals = append(literals, literal)
		}
	}
	return literals
}

// FindCallArrayArguments finds the calls of a function path whose first argument is an
// array, such as piko.partials.reloadGroup(["a", "b"]), and returns the elements of those
// arrays that are string literals. Computed elements are skipped.
//
// Takes tokens ([]Token) which are the script's tokens.
// Takes path ([]string) which names the function.
//
// Returns []Literal which holds the literal elements in source order.
func FindCallArrayArguments(tokens []Token, path ...string) []Literal {
	var literals []Literal
	for index := range tokens {
		next, found := matchPath(tokens, index, path)
		if !found || !isTokenAt(tokens, next, parsejs.OpenParenToken) || !isTokenAt(tokens, next+1, parsejs.OpenBracketToken) {
			continue
		}
		literals = append(literals, arrayLiterals(tokens, next+2)...)
	}
	return literals
}

// matchPath reports whether an object path, such as pk.refs, starts at a token and is not
// itself a member of another object.
//
// Takes tokens ([]Token) which are the script's tokens.
// Takes index (int) which is the token where the path may start.
// Takes path ([]string) which names the path's parts.
//
// Returns int which is the index of the token after the path.
// Returns bool which is true when the path starts at index.
func matchPath(tokens []Token, index int, path []string) (int, bool) {
	if len(path) == 0 || (index > 0 && isMemberAccess(tokens[index-1].Type)) {
		return 0, false
	}
	position := index
	for partIndex, part := range path {
		if partIndex > 0 {
			if !isTokenAt(tokens, position, parsejs.DotToken) {
				return 0, false
			}
			position++
		}
		if position >= len(tokens) || tokens[position].Text != part || !parsejs.IsIdentifierName(tokens[position].Type) {
			return 0, false
		}
		position++
	}
	return position, true
}

// propertyAfterPath reads the literal property named straight after an object path.
//
// Takes tokens ([]Token) which are the script's tokens.
// Takes index (int) which is the token after the path.
//
// Returns PropertyRead which is the property read.
// Returns bool which is false when no literal property follows.
func propertyAfterPath(tokens []Token, index int) (PropertyRead, bool) {
	position := index
	if position < len(tokens) && isMemberAccess(tokens[position].Type) {
		position++
		if position < len(tokens) && parsejs.IsIdentifierName(tokens[position].Type) {
			token := tokens[position]
			literal := Literal{Value: token.Text, Start: token.Start, Length: len(token.Text)}
			return PropertyRead{Literal: literal, Assigned: isTokenAt(tokens, position+1, parsejs.EqToken)}, true
		}
	}
	if !isTokenAt(tokens, position, parsejs.OpenBracketToken) || isTokenAt(tokens, index, parsejs.DotToken) {
		return PropertyRead{}, false
	}
	literalPosition := position + 1
	closePosition := literalPosition + 1
	literal, ok := stringLiteralAt(tokens, literalPosition)
	if !ok || !isTokenAt(tokens, closePosition, parsejs.CloseBracketToken) {
		return PropertyRead{}, false
	}
	return PropertyRead{Literal: literal, Assigned: isTokenAt(tokens, closePosition+1, parsejs.EqToken)}, true
}

// arrayLiterals reads the string literal elements of an array, from its first element to
// its closing bracket. Only elements that are a single string literal count, so those
// inside nested calls, arrays or conditions are skipped.
//
// Takes tokens ([]Token) which are the script's tokens.
// Takes index (int) which is the token after the opening bracket.
//
// Returns []Literal which holds the literal elements.
func arrayLiterals(tokens []Token, index int) []Literal {
	var literals []Literal
	depth := 0
	for position := index; position < len(tokens); position++ {
		depth = updateNestingDepth(tokens[position].Type, depth)
		if depth < 0 {
			return literals
		}
		if depth > 0 || !isElementStart(tokens, position, index) {
			continue
		}
		literal, ok := stringLiteralAt(tokens, position)
		if ok && isElementEnd(tokens, position+1) {
			literals = append(literals, literal)
		}
	}
	return literals
}

// updateNestingDepth tracks how deep a token is inside brackets, parentheses and braces.
//
// Takes tokenType (parsejs.TokenType) which is the token to apply.
// Takes depth (int) which is the depth before the token.
//
// Returns int which is the depth after the token, below zero once the enclosing bracket
// closes.
func updateNestingDepth(tokenType parsejs.TokenType, depth int) int {
	switch tokenType {
	case parsejs.OpenBracketToken, parsejs.OpenParenToken, parsejs.OpenBraceToken:
		return depth + 1
	case parsejs.CloseBracketToken, parsejs.CloseParenToken, parsejs.CloseBraceToken:
		return depth - 1
	default:
		return depth
	}
}

// isElementStart reports whether a token begins an array element. That is when it is the
// first token in the array or follows a comma.
//
// Takes tokens ([]Token) which are the script's tokens.
// Takes position (int) which is the token to check.
// Takes first (int) which is the first token in the array.
//
// Returns bool which is true when the token begins an element.
func isElementStart(tokens []Token, position, first int) bool {
	return position == first || tokens[position-1].Type == parsejs.CommaToken
}

// isElementEnd reports whether a token ends an array element.
//
// Takes tokens ([]Token) which are the script's tokens.
// Takes position (int) which is the token after the element.
//
// Returns bool which is true for a comma or the closing bracket.
func isElementEnd(tokens []Token, position int) bool {
	return isTokenAt(tokens, position, parsejs.CommaToken) || isTokenAt(tokens, position, parsejs.CloseBracketToken)
}

// isArgumentEnd reports whether a token ends a call argument.
//
// Takes tokens ([]Token) which are the script's tokens.
// Takes position (int) which is the token after the argument.
//
// Returns bool which is true for a comma or the closing parenthesis.
func isArgumentEnd(tokens []Token, position int) bool {
	return isTokenAt(tokens, position, parsejs.CommaToken) || isTokenAt(tokens, position, parsejs.CloseParenToken)
}

// stringLiteralAt reads a plain string literal, which is a quoted string or a template
// literal with no substitutions. A literal with an escape sequence is not read, so the
// value is always the text written in the source.
//
// Takes tokens ([]Token) which are the script's tokens.
// Takes position (int) which is the token to read.
//
// Returns Literal which is the literal read.
// Returns bool which is false when the token is not a plain string literal.
func stringLiteralAt(tokens []Token, position int) (Literal, bool) {
	if position >= len(tokens) {
		return Literal{}, false
	}
	token := tokens[position]
	if token.Type != parsejs.StringToken && token.Type != parsejs.TemplateToken {
		return Literal{}, false
	}
	text := token.Text
	if len(text) < minimumQuotedLength || strings.ContainsRune(text, '\\') || strings.Contains(text, "${") {
		return Literal{}, false
	}
	return Literal{Value: text[1 : len(text)-1], Start: token.Start, Length: len(text)}, true
}

// isTokenAt reports whether the token at a position has a given type.
//
// Takes tokens ([]Token) which are the script's tokens.
// Takes position (int) which is the token to check.
// Takes tokenType (parsejs.TokenType) which is the type to look for.
//
// Returns bool which is true when the position holds a token of that type.
func isTokenAt(tokens []Token, position int, tokenType parsejs.TokenType) bool {
	return position < len(tokens) && tokens[position].Type == tokenType
}

// isMemberAccess reports whether a token is . or ?., which come before a property name.
//
// Takes tokenType (parsejs.TokenType) which is the token to check.
//
// Returns bool which is true for . and ?.
func isMemberAccess(tokenType parsejs.TokenType) bool {
	return tokenType == parsejs.DotToken || tokenType == parsejs.OptChainToken
}
