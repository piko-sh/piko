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

package ast_domain

// Provides lexical analysis for CSS selector parsing by tokenizing selector
// strings into identifiers, operators, and punctuation. Implements pooled
// QueryLexer instances with efficient token recognition for classes, IDs,
// attributes, pseudo-classes, and combinators.

import (
	"sync"
	"unicode"
)

const (
	// TokenIllegal marks a token that is not valid or not known.
	TokenIllegal QueryTokenType = iota

	// TokenEOF represents the end of file token.
	TokenEOF

	// TokenIdent represents identifiers like main, div, card.
	TokenIdent

	// TokenString represents quoted values like "value" or 'value'.
	TokenString

	// TokenHash represents the hash symbol (#) for ID selectors.
	TokenHash

	// TokenDot represents the dot symbol (.) for class selectors.
	TokenDot

	// TokenStar represents the universal selector (*).
	TokenStar

	// TokenLBracket represents the left bracket ([) for attribute selectors.
	TokenLBracket

	// TokenRBracket represents the right bracket (]).
	TokenRBracket

	// TokenLParen represents the left parenthesis (().
	TokenLParen

	// TokenRParen represents the right parenthesis ()).
	TokenRParen

	// TokenColon represents the colon (:) for pseudo-classes.
	TokenColon

	// TokenCombinator represents the child combinator (>).
	TokenCombinator

	// TokenPlus represents the adjacent sibling combinator (+).
	TokenPlus

	// TokenTilde represents the general sibling combinator (~).
	TokenTilde

	// TokenComma represents the comma (,) for selector lists.
	TokenComma

	// TokenWhitespace represents whitespace for descendant combinators.
	TokenWhitespace

	// TokenEquals represents the equals operator (=).
	TokenEquals

	// TokenIncludes represents the includes operator (~=).
	TokenIncludes

	// TokenDashMatch represents the dash match operator (|=).
	TokenDashMatch

	// TokenPrefix represents the prefix match operator (^=).
	TokenPrefix

	// TokenSuffix represents the suffix match operator ($=).
	TokenSuffix

	// TokenContains represents the contains operator (*=).
	TokenContains
)

var (
	// queryLexerPool pools QueryLexer instances to reduce allocations.
	queryLexerPool = sync.Pool{
		New: func() any {
			return &QueryLexer{}
		},
	}

	// simpleTokens maps ASCII characters to their token info.
	//
	// A zero value means the character is not a simple token. Using an array
	// instead of a map avoids allocation and is faster for ASCII lookups. The
	// size is defined by singleCharTokenTableSize in
	// expression_lexer.go.
	simpleTokens [singleCharTokenTableSize]simpleTokenInfo
)

// simpleTokenInfo holds pre-allocated token type and literal for single-char
// tokens.
type simpleTokenInfo struct {
	// literal is the string value of the token.
	literal string

	// tokenType is the query token type this character represents.
	tokenType QueryTokenType
}

// QueryTokenType represents the kind of token found in a selector query.
type QueryTokenType int

// QueryToken represents a single token read from an input string.
type QueryToken struct {
	// Literal is the exact text of the token as it appears in the source.
	Literal string

	// Location is the position of this token in the source text.
	Location Location

	// Type is the kind of token this query element represents.
	Type QueryTokenType
}

// QueryLexer holds the state of the lexer for turning query strings into
// tokens.
type QueryLexer struct {
	// input holds the raw query string being tokenised.
	input string

	// position is the current position in the input string.
	position int

	// readPosition is the next position to read from in the input string.
	readPosition int

	// character is the current character being examined.
	character rune

	// line is the current line number in the input; starts at 1.
	line int

	// column is the current column position within the line; starts at 1.
	column int
}

// NewQueryLexer creates a new lexer for the given input string.
// The returned lexer should be released with Release when done.
//
// Takes input (string) which is the query text to tokenise.
//
// Returns *QueryLexer which is ready to produce tokens from the input.
func NewQueryLexer(input string) *QueryLexer {
	l, ok := queryLexerPool.Get().(*QueryLexer)
	if !ok {
		l = &QueryLexer{}
	}
	l.input = input
	l.position = 0
	l.readPosition = 0
	l.character = 0
	l.line = 1
	l.column = 1
	l.readChar()
	return l
}

// Release returns the lexer to the pool for reuse.
func (l *QueryLexer) Release() {
	l.input = ""
	queryLexerPool.Put(l)
}

// NextToken reads the input and returns the next token.
//
// Returns QueryToken which holds the parsed token with its type, literal
// value, and position in the source.
func (l *QueryLexer) NextToken() QueryToken {
	location := Location{Line: l.line, Column: l.column, Offset: 0}

	if l.isWhitespace() {
		l.skipWhitespace()
		return QueryToken{Type: TokenWhitespace, Literal: " ", Location: location}
	}

	if l.character < singleCharTokenTableSize {
		if info := simpleTokens[l.character]; info.tokenType != 0 {
			token := QueryToken{Type: info.tokenType, Literal: info.literal, Location: location}
			l.readChar()
			return token
		}
	}

	return l.lexComplexToken(location)
}

// isWhitespace checks if the current character is whitespace.
//
// Returns bool which is true when the current character is a space, tab,
// newline, or carriage return.
func (l *QueryLexer) isWhitespace() bool {
	return l.character == ' ' || l.character == '\t' || l.character == '\n' || l.character == '\r'
}

// lexComplexToken parses tokens that need lookahead or special handling.
//
// Takes location (Location) which specifies the source position for the token.
//
// Returns QueryToken which contains the parsed token with its type and value.
func (l *QueryLexer) lexComplexToken(location Location) QueryToken {
	switch l.character {
	case '~':
		return l.lexTwoCharToken('=', TokenIncludes, "~=", TokenTilde, "~", location)
	case '|':
		return l.lexTwoCharToken('=', TokenDashMatch, "|=", TokenIllegal, "|", location)
	case '^':
		return l.lexTwoCharToken('=', TokenPrefix, "^=", TokenIllegal, "^", location)
	case '$':
		return l.lexTwoCharToken('=', TokenSuffix, "$=", TokenIllegal, "$", location)
	case '*':
		return l.lexTwoCharToken('=', TokenContains, "*=", TokenStar, "*", location)
	case '"', '\'':
		token := QueryToken{Type: TokenString, Literal: l.readString(l.character), Location: location}
		return token
	case 0:
		return QueryToken{Type: TokenEOF, Literal: "", Location: location}
	default:
		if isQueryIdentChar(l.character) {
			return QueryToken{Type: TokenIdent, Literal: l.readIdentifier(), Location: location}
		}
		token := QueryToken{Type: TokenIllegal, Literal: string(l.character), Location: location}
		l.readChar()
		return token
	}
}

// lexTwoCharToken handles tokens that may be one or two characters.
//
// Takes secondChar (rune) which is the second character to check for.
// Takes twoCharType (QueryTokenType) which is the token type for two-char
// match.
// Takes twoCharLiteral (string) which is the literal for two-char match.
// Takes oneCharType (QueryTokenType) which is the token type for one-char
// match.
// Takes oneCharLiteral (string) which is the literal for one-char match.
// Takes location (Location) which is the source location of the token.
//
// Returns QueryToken which is the resulting token based on the match.
func (l *QueryLexer) lexTwoCharToken(
	secondChar rune,
	twoCharType QueryTokenType,
	twoCharLiteral string,
	oneCharType QueryTokenType,
	oneCharLiteral string,
	location Location,
) QueryToken {
	if l.peekChar() == secondChar {
		l.readChar()
		l.readChar()
		return QueryToken{Type: twoCharType, Literal: twoCharLiteral, Location: location}
	}
	l.readChar()
	return QueryToken{Type: oneCharType, Literal: oneCharLiteral, Location: location}
}

// readChar moves the lexer forward by one character.
func (l *QueryLexer) readChar() {
	if l.character == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}

	if l.readPosition >= len(l.input) {
		l.character = 0
	} else {
		l.character = rune(l.input[l.readPosition])
	}
	l.position = l.readPosition
	l.readPosition++
}

// readIdentifier reads a sequence of valid identifier characters.
// Handles the special case of the "piko:" namespace prefix, treating
// "piko:elementname" as a single identifier for Piko framework elements.
//
// Returns string which is the identifier text from the input.
func (l *QueryLexer) readIdentifier() string {
	start := l.position
	for isQueryIdentChar(l.character) {
		l.readChar()
	}

	identifier := l.input[start:l.position]
	if identifier == "piko" && l.character == ':' && isQueryIdentChar(l.peekChar()) {
		l.readChar()
		for isQueryIdentChar(l.character) {
			l.readChar()
		}
	}

	return l.input[start:l.position]
}

// readString reads a quoted string literal.
//
// Takes quote (rune) which is the quote character that marks the start and end
// of the string.
//
// Returns string which is the content between the quotes. Escaped quotes are
// kept as they are.
func (l *QueryLexer) readString(quote rune) string {
	l.readChar()
	start := l.position
	for {
		if l.character == '\\' && l.peekChar() == quote {
			l.readChar()
			l.readChar()
			continue
		}
		if l.character == quote || l.character == 0 {
			break
		}
		l.readChar()
	}
	end := l.position
	l.readChar()
	return l.input[start:end]
}

// skipWhitespace moves past any space, tab, or newline characters in the input.
func (l *QueryLexer) skipWhitespace() {
	for l.character == ' ' || l.character == '\t' || l.character == '\n' || l.character == '\r' {
		l.readChar()
	}
}

// peekChar looks at the next character without advancing the lexer's position.
//
// Returns rune which is the next character, or 0 if at end of input.
func (l *QueryLexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return rune(l.input[l.readPosition])
}

// isQueryIdentChar reports whether a rune is valid within an identifier.
//
// Takes character (rune) which is the character to check.
//
// Returns bool which is true if the rune is a letter, digit, hyphen, or
// underscore.
func isQueryIdentChar(character rune) bool {
	return unicode.IsLetter(character) || unicode.IsDigit(character) || character == '-' || character == '_'
}

func init() {
	simpleTokens['='] = simpleTokenInfo{tokenType: TokenEquals, literal: "="}
	simpleTokens['+'] = simpleTokenInfo{tokenType: TokenPlus, literal: "+"}
	simpleTokens[','] = simpleTokenInfo{tokenType: TokenComma, literal: ","}
	simpleTokens['>'] = simpleTokenInfo{tokenType: TokenCombinator, literal: ">"}
	simpleTokens['['] = simpleTokenInfo{tokenType: TokenLBracket, literal: "["}
	simpleTokens[']'] = simpleTokenInfo{tokenType: TokenRBracket, literal: "]"}
	simpleTokens['('] = simpleTokenInfo{tokenType: TokenLParen, literal: "("}
	simpleTokens[')'] = simpleTokenInfo{tokenType: TokenRParen, literal: ")"}
	simpleTokens[':'] = simpleTokenInfo{tokenType: TokenColon, literal: ":"}
	simpleTokens['.'] = simpleTokenInfo{tokenType: TokenDot, literal: "."}
	simpleTokens['#'] = simpleTokenInfo{tokenType: TokenHash, literal: "#"}
}
