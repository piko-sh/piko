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

// Provides lexical analysis for expression parsing by tokenizing input into
// identifiers, operators, literals, and keywords. Implements efficient
// single-character token lookup, string interning, and pooled lexer instances
// for performance optimisation.

import (
	"context"
	"fmt"
	"sync"
	"unicode/utf8"

	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// decimalSuffix is the suffix that marks a number as a decimal.
	decimalSuffix = "d"

	// bigIntSuffix is the suffix that marks big integer literals.
	bigIntSuffix = "n"

	// opLen1 is the length of a single-character operator.
	opLen1 = 1

	// opLen2 is the length of two-character operators such as == and !=.
	opLen2 = 2

	// opLen3 is the length of a three-character operator such as "!~=" or "?.[".
	opLen3 = 3

	// singleCharTokenTableSize is the size of the lookup table for
	// single-character tokens.
	singleCharTokenTableSize = 128
)

const (
	// tokenEOF marks the end of input during expression parsing.
	tokenEOF tokenType = iota

	// tokenIdent represents an identifier token in the lexer.
	tokenIdent

	// tokenKeywordIn represents the "in" keyword token.
	tokenKeywordIn

	// tokenNumber represents a numeric literal token in the parser.
	tokenNumber

	// tokenString is the token type name for string literals.
	tokenString

	// tokenSymbol is the token type for operator symbols such as
	// +, -, *, /, ==, !=, &&, and ||.
	tokenSymbol

	// tokenLParen is the left parenthesis token.
	tokenLParen

	// tokenRParen represents a right parenthesis token.
	tokenRParen

	// tokenLBracket is the left bracket character in parsed text.
	tokenLBracket

	// tokenRBracket represents a right bracket character in parsed text.
	tokenRBracket

	// tokenLBrace represents a left brace character ({) in the token stream.
	tokenLBrace

	// tokenRBrace is the token type for a closing brace character.
	tokenRBrace

	// tokenColon is the lexical token for a colon character.
	tokenColon

	// tokenComma is the comma separator token.
	tokenComma

	// tokenDot represents a dot character in path expressions.
	tokenDot

	// tokenOptionalDot is the token type for the ?. optional
	// chaining member access operator.
	tokenOptionalDot

	// tokenOptionalBracket is the token type for the ?.[ optional
	// chaining index operator.
	tokenOptionalBracket

	// tokenError is the error value returned by the lexer for invalid tokens.
	tokenError

	// tokenTemplateLiteral represents a literal text segment in a template.
	tokenTemplateLiteral

	// tokenKeywordTrue represents the boolean true keyword token.
	tokenKeywordTrue

	// tokenKeywordFalse is the token for the false keyword.
	tokenKeywordFalse

	// tokenKeywordNil represents the nil keyword token.
	tokenKeywordNil

	// tokenDecimal is a decimal number literal token.
	tokenDecimal

	// tokenBigInt represents a large integer token type.
	tokenBigInt

	// tokenDateTime is a token type for date and time values.
	tokenDateTime

	// tokenDate is the date format token for parsing.
	tokenDate

	// tokenTime is the token type for time literal values.
	tokenTime

	// tokenDuration is the token type for duration literal values
	// (e.g. du'1h30m').
	tokenDuration

	// tokenRune is the token type for a single rune literal.
	tokenRune

	// tokenAt is the '@' symbol token used to mark linked messages.
	tokenAt
)

var (
	// singleCharTokens maps ASCII bytes to their token types for
	// single-character tokens, where a zero value means the character
	// is not a recognised single-char token.
	//
	// This enables O(1) dispatch for punctuation without switch
	// statements.
	singleCharTokens = [singleCharTokenTableSize]tokenType{
		'(': tokenLParen,
		')': tokenRParen,
		'{': tokenLBrace,
		'}': tokenRBrace,
		'[': tokenLBracket,
		']': tokenRBracket,
		':': tokenColon,
		',': tokenComma,
		'.': tokenDot,
		'+': tokenSymbol,
		'-': tokenSymbol,
		'*': tokenSymbol,
		'/': tokenSymbol,
		'%': tokenSymbol,
		'@': tokenAt,
	}

	keywords = map[string]tokenType{
		"in":    tokenKeywordIn,
		"true":  tokenKeywordTrue,
		"false": tokenKeywordFalse,
		"nil":   tokenKeywordNil,
		"null":  tokenKeywordNil,
	}

	// lexerPool provides pooled lexer instances to reduce allocations.
	lexerPool = sync.Pool{
		New: func() any {
			return &lexer{
				input:    "",
				tokens:   make([]lexerToken, 0, 64),
				position: 0,
				line:     0,
				column:   0,
			}
		},
	}
)

// tokenType represents the kind of a lexical token in the lexer.
type tokenType int

// lexerToken represents a token with zero-copy value storage.
// The value is computed only when needed, using the offset and length.
type lexerToken struct {
	// errorMessage holds the error message for tokenError; empty for other token
	// types. Placed first to reduce GC pointer bitmap size.
	errorMessage string

	// Location is the source position where this token appears.
	Location Location

	// offset is the start position in the input string.
	offset int

	// length is the number of bytes in the token value.
	length int

	// Type is the kind of token, such as keyword, symbol, or literal.
	Type tokenType
}

// getValue returns the token's value by slicing the input string.
// This is a zero-copy operation; no memory allocation occurs.
//
// Takes input (string) which is the source text to slice from.
//
// Returns string which is the token's value extracted from the input.
func (t *lexerToken) getValue(input string) string {
	if t.length == 0 {
		return ""
	}
	return input[t.offset : t.offset+t.length]
}

// lexer holds the state needed to break input text into tokens.
type lexer struct {
	// input is the source text being tokenised.
	input string

	// tokens holds the list of tokens produced from the input text.
	tokens []lexerToken

	// position is the current byte offset in the input string.
	position int

	// line is the current line number in the input; starts at 1.
	line int

	// column is the current column number in the input; resets to 1 on newlines.
	column int
}

// run is the main loop of the lexer, calling each tokenisation
// method in turn.
//
// preferred over chained conditionals.
//
//nolint:revive // linear dispatch
func (l *lexer) run() {
	for l.position < len(l.input) {
		if l.lexWhitespace() {
			continue
		}
		if l.lexComment() {
			continue
		}
		if l.lexPrefixedLiteral() {
			continue
		}
		if l.lexSymbol() {
			continue
		}
		if l.lexNumber() {
			continue
		}
		if l.lexStringLike('\'', '"') {
			continue
		}
		if l.lexStringLike('`') {
			continue
		}
		if l.lexIdentifier() {
			continue
		}

		r, w := utf8.DecodeRuneInString(l.input[l.position:])
		message := fmt.Sprintf("unrecognised character '%s'", string(r))
		l.addErrorToken(l.position, w, message)
		l.position += w
	}
	l.addToken(tokenEOF, l.position, 0)
}

// lexWhitespace reads and skips whitespace characters from the input.
//
// Returns bool which is true when at least one whitespace character was read.
func (l *lexer) lexWhitespace() bool {
	if l.position >= len(l.input) {
		return false
	}
	b := l.input[l.position]
	if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
		return false
	}
	startPosition := l.position
	for l.position < len(l.input) {
		b = l.input[l.position]
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			break
		}
		if b == '\n' {
			l.line++
			l.column = 1
		} else {
			l.column++
		}
		l.position++
	}
	return l.position > startPosition
}

// lexComment consumes a multi-line comment /* ... */ and skips it entirely.
//
// Returns bool which is true if a comment was found and processed, or false
// if no comment starts at the current position.
func (l *lexer) lexComment() bool {
	if l.position+1 >= len(l.input) {
		return false
	}

	if l.input[l.position] != '/' || l.input[l.position+1] != '*' {
		return false
	}

	startOffset := l.position
	startLine := l.line
	startColumn := l.column

	l.position += 2
	l.column += 2

	for l.position+1 < len(l.input) {
		if l.input[l.position] == '*' && l.input[l.position+1] == '/' {
			l.position += 2
			l.column += 2
			return true
		}

		if l.input[l.position] == '\n' {
			l.line++
			l.column = 1
		} else {
			l.column++
		}
		l.position++
	}

	if l.position < len(l.input) {
		if l.input[l.position] == '\n' {
			l.line++
			l.column = 1
		} else {
			l.column++
		}
		l.position++
	}

	l.tokens = append(l.tokens, lexerToken{
		Type:         tokenError,
		offset:       startOffset,
		length:       l.position - startOffset,
		errorMessage: "unterminated multi-line comment",
		Location:     Location{Line: startLine, Column: startColumn, Offset: startOffset},
	})
	return true
}

// lexPrefixedLiteral handles dt'...', d'...', and t'...' style literals.
//
// Returns bool which is true if a prefixed literal was found and handled.
func (l *lexer) lexPrefixedLiteral() bool {
	if l.position+2 > len(l.input) {
		return false
	}

	prefix, tokType := l.matchPrefixedLiteralPrefix()
	if prefix == "" {
		return false
	}

	content, ok := l.lexQuotedContent(prefix)
	if !ok {
		l.addErrorToken(l.position, len(l.input)-l.position, "unterminated prefixed literal")
		l.position = len(l.input)
		return true
	}

	contentStart := l.position + len(prefix)
	l.addToken(tokType, contentStart, len(content))
	l.advance(len(prefix) + len(content) + 1)
	return true
}

// lexSymbol tries to lex a symbol token at the current position.
//
// Returns bool which is true if a symbol was lexed.
func (l *lexer) lexSymbol() bool {
	if l.position >= len(l.input) {
		return false
	}

	b := l.input[l.position]

	if l.lexMultiCharOp(b) {
		return true
	}

	if b < singleCharTokenTableSize {
		if tt := singleCharTokens[b]; tt != 0 {
			l.addToken(tt, l.position, opLen1)
			l.advance(opLen1)
			return true
		}
	}

	return false
}

// lexMultiCharOp handles operators that may be one to three characters long.
//
// Takes b (byte) which is the first character of the possible operator.
//
// Returns bool which is true if a multi-character operator was matched and
// consumed.
func (l *lexer) lexMultiCharOp(b byte) bool {
	switch b {
	case '*':
		return l.lexStarOp()
	case '=':
		return l.lexEqualOp()
	case '!':
		return l.lexBangOp()
	case '~':
		return l.lexTildeOp()
	case '<', '>':
		return l.lexComparisonOp()
	case '&':
		return l.lexDoubleCharOp('&', tokenSymbol)
	case '|':
		return l.lexDoubleCharOp('|', tokenSymbol)
	case '?':
		return l.lexQuestionOp()
	}
	return false
}

// lexStarOp detects stray */ (comment close without open).
//
// Returns bool which is true if */ was found and an error token was emitted.
func (l *lexer) lexStarOp() bool {
	remaining := len(l.input) - l.position
	if remaining >= opLen2 && l.input[l.position+1] == '/' {
		l.addErrorToken(l.position, opLen2, "unexpected '*/' without matching '/*'")
		l.advance(opLen2)
		return true
	}
	return false
}

// lexEqualOp handles the == operator.
//
// Returns bool which is true if an equality operator was found and consumed.
func (l *lexer) lexEqualOp() bool {
	remaining := len(l.input) - l.position
	if remaining >= opLen2 && l.input[l.position:l.position+opLen2] == "==" {
		l.addToken(tokenSymbol, l.position, opLen2)
		l.advance(opLen2)
		return true
	}
	return false
}

// lexBangOp reads a bang operator (!~=, !=, or !) and adds it as a token.
//
// Returns bool which is always true since all paths produce a valid token.
func (l *lexer) lexBangOp() bool {
	remaining := len(l.input) - l.position
	if remaining >= opLen3 && l.input[l.position:l.position+opLen3] == "!~=" {
		l.addToken(tokenSymbol, l.position, opLen3)
		l.advance(opLen3)
		return true
	}
	if remaining >= opLen2 && l.input[l.position:l.position+opLen2] == "!=" {
		l.addToken(tokenSymbol, l.position, opLen2)
		l.advance(opLen2)
		return true
	}
	l.addToken(tokenSymbol, l.position, opLen1)
	l.advance(opLen1)
	return true
}

// lexTildeOp reads a tilde operator (~= or ~) and adds it as a token.
//
// Returns bool which is always true as all paths produce a valid token.
func (l *lexer) lexTildeOp() bool {
	remaining := len(l.input) - l.position
	if remaining >= opLen2 && l.input[l.position:l.position+opLen2] == "~=" {
		l.addToken(tokenSymbol, l.position, opLen2)
		l.advance(opLen2)
		return true
	}
	l.addToken(tokenSymbol, l.position, opLen1)
	l.advance(opLen1)
	return true
}

// lexComparisonOp handles the comparison operators <=, >=, <, and >.
//
// Returns bool which is true when an operator was lexed.
func (l *lexer) lexComparisonOp() bool {
	remaining := len(l.input) - l.position
	if remaining >= opLen2 && l.input[l.position+1] == '=' {
		l.addToken(tokenSymbol, l.position, opLen2)
		l.advance(opLen2)
		return true
	}
	l.addToken(tokenSymbol, l.position, opLen1)
	l.advance(opLen1)
	return true
}

// lexDoubleCharOp handles operators that require exactly two of the same
// character (&&, ||).
//
// Takes char (byte) which is the character to match.
// Takes tt (tokenType) which is the token type to assign on match.
//
// Returns bool which is true if the operator was matched and consumed, or
// false if only one character is present (invalid operator).
func (l *lexer) lexDoubleCharOp(char byte, tt tokenType) bool {
	remaining := len(l.input) - l.position
	if remaining >= opLen2 && l.input[l.position+1] == char {
		l.addToken(tt, l.position, opLen2)
		l.advance(opLen2)
		return true
	}
	return false
}

// lexQuestionOp handles question mark operators: ?.[, ?., ??, and ?.
//
// Returns bool which is always true because a valid token is always produced.
func (l *lexer) lexQuestionOp() bool {
	remaining := len(l.input) - l.position
	if remaining >= opLen3 && l.input[l.position:l.position+opLen3] == "?.[" {
		l.addToken(tokenOptionalBracket, l.position, opLen3)
		l.advance(opLen3)
		return true
	}
	if remaining >= opLen2 {
		switch l.input[l.position+1] {
		case '.':
			l.addToken(tokenOptionalDot, l.position, opLen2)
			l.advance(opLen2)
			return true
		case '?':
			l.addToken(tokenSymbol, l.position, opLen2)
			l.advance(opLen2)
			return true
		}
	}
	l.addToken(tokenSymbol, l.position, opLen1)
	l.advance(opLen1)
	return true
}

// lexNumber reads a number token from the current position.
//
// Returns bool which is true if a number was read, false otherwise.
func (l *lexer) lexNumber() bool {
	if l.position >= len(l.input) {
		return false
	}
	b := l.input[l.position]
	if b < '0' || b > '9' {
		return false
	}

	start := l.position
	end := l.position
	isInteger := true

	for end < len(l.input) {
		b = l.input[end]
		if b >= '0' && b <= '9' {
			end++
			continue
		}
		if b == '.' && isInteger && l.isFloatDecimalASCII(end) {
			isInteger = false
			end++
			continue
		}
		break
	}

	length := end - start

	if l.atSuffix(end, "d") {
		l.addToken(tokenDecimal, start, length)
		l.advance(length + 1)
	} else if l.atSuffix(end, "n") {
		if isInteger {
			l.addToken(tokenBigInt, start, length)
			l.advance(length + 1)
		} else {
			l.addErrorToken(start, length+1, fmt.Sprintf("invalid bigint literal; fractional part not allowed: %s", l.input[start:end+1]))
			l.advance(length + 1)
		}
	} else {
		l.addToken(tokenNumber, start, length)
		l.advance(length)
	}

	return true
}

// lexStringLike handles single, double, and backtick quoted strings.
// The quotes must be ASCII characters (', ", `).
//
// Takes quotes (...rune) which specifies the quote characters to accept.
//
// Returns bool which indicates whether a string was found and consumed.
func (l *lexer) lexStringLike(quotes ...rune) bool {
	if l.position >= len(l.input) {
		return false
	}
	b := l.input[l.position]
	quote := byte(0)
	for _, q := range quotes {
		if b == byte(q) {
			quote = b
			break
		}
	}
	if quote == 0 {
		return false
	}

	content, ok := l.lexQuotedContent(string(rune(quote)))
	if !ok {
		message := "unterminated string literal"
		if quote == '`' {
			message = "unterminated template literal"
		}
		l.addErrorToken(l.position, len(l.input)-l.position, message)
		l.position = len(l.input)
		return true
	}

	tt := tokenString
	if quote == '`' {
		tt = tokenTemplateLiteral
	}
	totalLength := 1 + len(content) + 1
	l.addToken(tt, l.position, totalLength)
	l.advanceWithLineCount(l.input[l.position : l.position+totalLength])

	return true
}

// lexIdentifier scans an identifier or keyword token from the input.
//
// Returns bool which is true when a token was read.
func (l *lexer) lexIdentifier() bool {
	if l.position >= len(l.input) {
		return false
	}

	end := l.scanIdentStart(l.position)
	if end < 0 {
		return false
	}

	end = l.scanIdentChars(end)

	start := l.position
	length := end - start
	value := l.input[start:end]
	tt := tokenIdent
	if keywordType, ok := keywords[value]; ok {
		tt = keywordType
	}

	l.addToken(tt, start, length)
	l.advance(length)
	return true
}

// scanIdentStart scans the first character of an identifier.
//
// Takes position (int) which is the position to start scanning from.
//
// Returns int which is the new position after the character, or -1 if the
// character is not a valid identifier start.
func (l *lexer) scanIdentStart(position int) int {
	b := l.input[position]
	if b < utf8.RuneSelf {
		if identStartTable[b] {
			return position + 1
		}
		return -1
	}
	r, w := utf8.DecodeRuneInString(l.input[position:])
	if isIdentStart(r) {
		return position + w
	}
	return -1
}

// scanIdentChars scans the remaining identifier characters from the given
// position.
//
// Takes position (int) which is the starting position in the input.
//
// Returns int which is the end position after all valid identifier characters.
func (l *lexer) scanIdentChars(position int) int {
	for position < len(l.input) {
		b := l.input[position]
		if b < utf8.RuneSelf {
			if !identCharTable[b] {
				break
			}
			position++
			continue
		}
		r, w := utf8.DecodeRuneInString(l.input[position:])
		if !isIdentChar(r) {
			break
		}
		position += w
	}
	return position
}

// addToken appends a token to the lexer's token list without copying the
// token's value. It uses offset and length to refer to the original source.
//
// Takes tt (tokenType) which specifies the type of token to add.
// Takes offset (int) which marks the start position in the source.
// Takes length (int) which defines the token's length in bytes.
func (l *lexer) addToken(tt tokenType, offset, length int) {
	l.tokens = append(l.tokens, lexerToken{
		errorMessage: "",
		Location:     Location{Line: l.line, Column: l.column, Offset: l.position},
		offset:       offset,
		length:       length,
		Type:         tt,
	})
}

// addErrorToken adds an error token with a message to the token stream.
//
// Takes offset (int) which is the start position of the error.
// Takes length (int) which is the length of the error text.
// Takes errMessage (string) which describes the error.
func (l *lexer) addErrorToken(offset, length int, errMessage string) {
	l.tokens = append(l.tokens, lexerToken{
		Type:         tokenError,
		offset:       offset,
		length:       length,
		errorMessage: errMessage,
		Location:     Location{Line: l.line, Column: l.column, Offset: l.position},
	})
}

// lexQuotedContent extracts content from a prefixed literal or a regular
// string literal, handling escaped quotes.
//
// Takes prefix (string) which specifies the literal prefix ending with the
// quote character.
//
// Returns string which contains the extracted content between quotes.
// Returns bool which indicates whether the content was found.
func (l *lexer) lexQuotedContent(prefix string) (string, bool) {
	quote := prefix[len(prefix)-1]
	contentStart := l.position + len(prefix)

	end := contentStart
	for end < len(l.input) {
		b := l.input[end]
		if b < utf8.RuneSelf {
			if b == quote && l.input[end-1] != '\\' {
				return l.input[contentStart:end], true
			}
			end++
		} else {
			_, w := utf8.DecodeRuneInString(l.input[end:])
			end += w
		}
	}
	return "", false
}

// matchPrefixedLiteralPrefix checks for type-prefixed literal markers.
//
// Returns string which is the matched prefix (e.g. "d'", "t'", "du'"), or
// empty if no match is found.
// Returns tokenType which is the type of literal, or 0 if no match is found.
func (l *lexer) matchPrefixedLiteralPrefix() (string, tokenType) {
	if l.position+opLen2 > len(l.input) {
		return "", 0
	}
	if l.position+opLen3 <= len(l.input) && l.input[l.position:l.position+opLen3] == "du'" {
		return "du'", tokenDuration
	}

	twoCharPrefix := l.input[l.position : l.position+opLen2]
	switch twoCharPrefix {
	case "dt":
		if l.position+opLen2 < len(l.input) && l.input[l.position+opLen2] == '\'' {
			return "dt'", tokenDateTime
		}
	case "d'":
		return "d'", tokenDate
	case "t'":
		return "t'", tokenTime
	case "r'":
		return "r'", tokenRune
	}
	return "", 0
}

// atSuffix checks if the input has the given suffix at position end.
// Returns true only if the suffix matches and is not part of a larger word.
//
// Takes end (int) which is the position where the suffix should start.
// Takes suffix (string) which is the text to match.
//
// Returns bool which is true if the suffix is found and stands alone.
func (l *lexer) atSuffix(end int, suffix string) bool {
	suffixLen := len(suffix)
	if end+suffixLen > len(l.input) {
		return false
	}
	if l.input[end:end+suffixLen] != suffix {
		return false
	}
	if end+suffixLen < len(l.input) {
		nextRune, _ := utf8.DecodeRuneInString(l.input[end+suffixLen:])
		if isIdentChar(nextRune) {
			return false
		}
	}
	return true
}

// isFloatDecimalASCII checks if the character after the decimal point is a
// digit (ASCII fast path).
//
// Takes end (int) which specifies the position of the decimal point.
//
// Returns bool which is true if the next character is an ASCII digit.
func (l *lexer) isFloatDecimalASCII(end int) bool {
	if end+1 >= len(l.input) {
		return false
	}
	b := l.input[end+1]
	return b >= '0' && b <= '9'
}

// advance moves the lexer forward by the given number of bytes.
// It updates both the position and column count for a single-line token.
//
// Takes length (int) which specifies how many bytes to move forward.
func (l *lexer) advance(length int) {
	l.column += utf8.RuneCountInString(l.input[l.position : l.position+length])
	l.position += length
}

// advanceWithLineCount updates the position, line, and column after reading
// text that may contain newlines.
//
// Takes consumed (string) which is the text that was read from the input.
func (l *lexer) advanceWithLineCount(consumed string) {
	newlines := 0
	lastNewline := -1
	for i := range len(consumed) {
		if consumed[i] == '\n' {
			newlines++
			lastNewline = i
		}
	}
	if newlines > 0 {
		l.line += newlines
		l.column = utf8.RuneCountInString(consumed[lastNewline+1:]) + 1
	} else {
		l.column += utf8.RuneCountInString(consumed)
	}
	l.position += len(consumed)
}

// lexInto splits input text into tokens and adds them to the given slice.
// The caller owns the returned slice and may reuse it in later calls.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes input (string) which contains the text to split into tokens.
// Takes tokens ([]lexerToken) which provides the slice to add tokens to.
//
// Returns []lexerToken which may be a new slice if the original was too small.
func lexInto(ctx context.Context, input string, tokens []lexerToken) []lexerToken {
	l, ok := lexerPool.Get().(*lexer)
	if !ok {
		_, ctxLog := logger_domain.From(ctx, log)
		ctxLog.Error("lexerPool returned unexpected type, allocating new instance")
		l = &lexer{}
	}
	l.input = input
	l.position = 0
	l.line = 1
	l.column = 1
	l.tokens = tokens[:0]

	l.run()

	result := l.tokens
	l.tokens = nil
	lexerPool.Put(l)
	return result
}
