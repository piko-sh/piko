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

package db_engine_sqlite

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
)

// tokenKind classifies a lexed SQL token.
type tokenKind uint8

const (
	// tokenIdentifier marks bare or quoted identifiers and unreserved words.
	tokenIdentifier tokenKind = iota

	// tokenNumber marks integer, decimal, hex, or exponent numeric literals.
	tokenNumber

	// tokenString marks a single-quoted string literal.
	tokenString

	// tokenBlobLiteral marks an x'...' hexadecimal blob literal.
	tokenBlobLiteral

	// tokenOperator marks an arithmetic, comparison, or bitwise operator.
	tokenOperator

	// tokenLeftParen marks the '(' punctuation.
	tokenLeftParen

	// tokenRightParen marks the ')' punctuation.
	tokenRightParen

	// tokenComma marks the ',' punctuation.
	tokenComma

	// tokenSemicolon marks the ';' statement terminator.
	tokenSemicolon

	// tokenDot marks the '.' qualifier between schema, table, and column.
	tokenDot

	// tokenStar marks the '*' wildcard or multiplication operator.
	tokenStar

	// tokenQuestionMark marks an anonymous '?' positional parameter.
	tokenQuestionMark

	// tokenNumberedParam marks a numbered '?N' positional parameter.
	tokenNumberedParam

	// tokenNamedParam marks a named ':', '@', or '$' parameter.
	tokenNamedParam

	// tokenArrow marks the '->' JSON path operator.
	tokenArrow

	// tokenDoubleArrow marks the '->>' JSON value operator.
	tokenDoubleArrow

	// tokenEOF marks the end of the token stream.
	tokenEOF
)

// token holds a single lexed SQL token together with its source position.
type token struct {
	// value holds the raw text of the token as it appeared in the input.
	value string

	// position records the byte offset of the token's first character.
	position int

	// kind classifies what the token represents.
	kind tokenKind
}

// tokeniser tracks scan progress through a SQL input string.
type tokeniser struct {
	// input is the SQL source being scanned.
	input string

	// position is the current byte offset within input.
	position int
}

var (
	// dialectConfig declares SQLite lexical rules for the shared scanners: plain "--" line
	// comments, non-nesting block comments, and 0x hexadecimal integer literals (SQLite has
	// no 0o octal or 0b binary forms).
	dialectConfig = engine_shared.DialectConfig{
		Comments: engine_shared.CommentRules{},
		Numbers:  engine_shared.NumberRules{HexPrefix: true},
	}
)

// tokenise scans a SQL string into a flat sequence of tokens.
//
// Takes input (string) which is the SQL source text to lex.
//
// Returns []token which is the lexed sequence terminated by tokenEOF.
// Returns error when the input contains an unterminated literal or an unexpected
// character.
func tokenise(input string) ([]token, error) {
	lexer := &tokeniser{input: input}
	var tokens []token

	for {
		tok, err := lexer.next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.kind == tokenEOF {
			break
		}
	}

	return tokens, nil
}

// next returns the next token from the input stream.
//
// Returns token which is the lexed token, or tokenEOF when input is exhausted.
// Returns error when an unterminated literal or unexpected character is encountered.
func (t *tokeniser) next() (token, error) {
	if err := t.skipWhitespaceAndComments(); err != nil {
		return token{}, err
	}

	if t.position >= len(t.input) {
		return token{kind: tokenEOF, position: t.position}, nil
	}

	character := t.input[t.position]

	if tok, ok := t.readSingleCharToken(character); ok {
		return tok, nil
	}

	return t.readMultiCharToken(character)
}

var (
	// singleCharTokens maps ASCII byte values to their tokenKind plus one, so that a zero
	// entry signals "not a single-character token".
	singleCharTokens = [256]tokenKind{}
)

func init() {
	singleCharTokens['('] = tokenLeftParen + 1
	singleCharTokens[')'] = tokenRightParen + 1
	singleCharTokens[','] = tokenComma + 1
	singleCharTokens[';'] = tokenSemicolon + 1
	singleCharTokens['.'] = tokenDot + 1
	singleCharTokens['*'] = tokenStar + 1
}

// readSingleCharToken consumes one byte if it maps to a single-character punctuation
// token.
//
// Takes character (byte) which is the current byte at the scan position.
//
// Returns token which is the produced token when matched, else the zero value.
// Returns bool which is true when a single-character token was consumed.
func (t *tokeniser) readSingleCharToken(character byte) (token, bool) {
	mapped := singleCharTokens[character]
	if mapped == 0 {
		return token{}, false
	}
	startPosition := t.position
	t.position++
	return token{kind: mapped - 1, value: string(character), position: startPosition}, true
}

// readMultiCharToken dispatches to the appropriate scanner for tokens longer than one
// byte.
//
// Takes character (byte) which is the current byte at the scan position.
//
// Returns token which is the lexed multi-character token.
// Returns error when the underlying scanner reports a malformed literal.
func (t *tokeniser) readMultiCharToken(character byte) (token, error) {
	switch {
	case character == '?':
		return t.readQuestionParam()
	case character == ':' || character == '@' || character == '$':
		return t.readNamedParam()
	case character == '\'':
		return t.readString()
	case character == '"':
		return t.readQuotedIdentifier('"')
	case character == '`':
		return t.readQuotedIdentifier('`')
	case character == '[':
		return t.readBracketIdentifier()
	case isDigit(character):
		return t.readNumber()
	case (character == 'x' || character == 'X') && t.position+1 < len(t.input) && t.input[t.position+1] == '\'':
		return t.readBlobLiteral()
	}

	leading, _ := utf8.DecodeRuneInString(t.input[t.position:])
	if isIdentStart(leading) {
		return t.readIdentifier()
	}
	return t.readOperator()
}

// skipWhitespaceAndComments advances past spaces, tabs, newlines, line comments, and
// block comments.
//
// Returns error when a block comment is left unterminated.
func (t *tokeniser) skipWhitespaceAndComments() error {
	position, err := dialectConfig.Comments.SkipWhitespaceAndComments(t.input, t.position)
	t.position = position
	return err
}

// readString lexes a single-quoted SQL string literal with doubled-quote escaping.
//
// Returns token which is the lexed string token without surrounding quotes.
// Returns error when the literal is unterminated.
func (t *tokeniser) readString() (token, error) {
	startPosition := t.position

	value, position, ok := engine_shared.ScanDoubledDelimiter(t.input, t.position, '\'')
	if !ok {
		return token{}, fmt.Errorf("unterminated string literal at position %d", startPosition)
	}
	t.position = position

	return token{kind: tokenString, value: value, position: startPosition}, nil
}

// readQuotedIdentifier lexes a double-quote or backtick identifier with doubled-quote
// escaping.
//
// Takes quote (byte) which is the opening and closing quote character.
//
// Returns token which is the lexed identifier without surrounding quotes.
// Returns error when the identifier is unterminated.
func (t *tokeniser) readQuotedIdentifier(quote byte) (token, error) {
	startPosition := t.position

	value, position, ok := engine_shared.ScanDoubledDelimiter(t.input, t.position, quote)
	if !ok {
		return token{}, fmt.Errorf("unterminated quoted identifier at position %d", startPosition)
	}
	t.position = position

	return token{kind: tokenIdentifier, value: value, position: startPosition}, nil
}

// readBracketIdentifier lexes a `[name]` bracketed identifier.
//
// Returns token which is the lexed identifier without surrounding brackets.
// Returns error when the identifier is unterminated.
func (t *tokeniser) readBracketIdentifier() (token, error) {
	startPosition := t.position
	t.position++

	var builder strings.Builder
	for t.position < len(t.input) {
		character := t.input[t.position]
		if character == ']' {
			t.position++
			return token{kind: tokenIdentifier, value: builder.String(), position: startPosition}, nil
		}
		builder.WriteByte(character)
		t.position++
	}

	return token{}, fmt.Errorf("unterminated bracket identifier at position %d", startPosition)
}

// readNumber lexes an integer, hexadecimal, decimal, or exponent numeric literal.
//
// Returns token which is the lexed numeric token.
// Returns error which is currently always nil.
func (t *tokeniser) readNumber() (token, error) {
	startPosition := t.position

	position, matched, err := dialectConfig.Numbers.TryReadPrefixedNumber(t.input, t.position)
	if err != nil {
		return token{}, err
	}
	if matched {
		t.position = position
		return token{kind: tokenNumber, value: t.input[startPosition:t.position], position: startPosition}, nil
	}

	t.consumeWhile(isDigit)
	t.readFractionalPart()
	t.readExponentPart()

	return token{kind: tokenNumber, value: t.input[startPosition:t.position], position: startPosition}, nil
}

// readFractionalPart consumes the `.digits` portion of a numeric literal when present.
func (t *tokeniser) readFractionalPart() {
	if t.position >= len(t.input) || t.input[t.position] != '.' {
		return
	}
	t.position++
	t.consumeWhile(isDigit)
}

// readExponentPart consumes the `e[+-]?digits` exponent of a numeric literal when
// present.
func (t *tokeniser) readExponentPart() {
	if t.position >= len(t.input) {
		return
	}
	character := t.input[t.position]
	if character != 'e' && character != 'E' {
		return
	}
	t.position++
	if t.position < len(t.input) && (t.input[t.position] == '+' || t.input[t.position] == '-') {
		t.position++
	}
	t.consumeWhile(isDigit)
}

// consumeWhile advances the scan position while predicate matches the current byte.
//
// Takes predicate (func(byte) bool) which decides whether a byte is part of the current
// token.
func (t *tokeniser) consumeWhile(predicate func(byte) bool) {
	for t.position < len(t.input) && predicate(t.input[t.position]) {
		t.position++
	}
}

// readIdentifier lexes an unquoted identifier.
//
// Advances by UTF-8 rune width so multi-byte code points (Unicode letters or digits
// inside an identifier body, for example accented or non-Latin letters) are not truncated
// mid-code-point as they would be by a byte-at-a-time scan.
//
// Returns token which is the lexed identifier token.
// Returns error which is currently always nil.
func (t *tokeniser) readIdentifier() (token, error) {
	startPosition := t.position
	for t.position < len(t.input) {
		character, width := utf8.DecodeRuneInString(t.input[t.position:])
		if !isIdentPart(character) {
			break
		}
		t.position += width
	}
	return token{kind: tokenIdentifier, value: t.input[startPosition:t.position], position: startPosition}, nil
}

// readQuestionParam lexes an anonymous `?` or numbered `?N` parameter.
//
// Returns token which is the lexed parameter token.
// Returns error which is currently always nil.
func (t *tokeniser) readQuestionParam() (token, error) {
	startPosition := t.position
	t.position++

	if t.position < len(t.input) && isDigit(t.input[t.position]) {
		for t.position < len(t.input) && isDigit(t.input[t.position]) {
			t.position++
		}
		return token{kind: tokenNumberedParam, value: t.input[startPosition:t.position], position: startPosition}, nil
	}

	return token{kind: tokenQuestionMark, value: "?", position: startPosition}, nil
}

// readNamedParam lexes a named `:`, `@`, or `$` parameter including its identifier body.
//
// Returns token which is the lexed parameter token.
// Returns error which is currently always nil.
func (t *tokeniser) readNamedParam() (token, error) {
	startPosition := t.position
	t.position++
	for t.position < len(t.input) {
		character, width := utf8.DecodeRuneInString(t.input[t.position:])
		if !isIdentPart(character) {
			break
		}
		t.position += width
	}
	return token{kind: tokenNamedParam, value: t.input[startPosition:t.position], position: startPosition}, nil
}

// readBlobLiteral lexes an `x'...'` hexadecimal blob literal.
//
// Returns token which is the lexed blob token without surrounding quotes.
// Returns error when the literal is unterminated.
func (t *tokeniser) readBlobLiteral() (token, error) {
	startPosition := t.position
	t.position += 2

	var builder strings.Builder
	for t.position < len(t.input) {
		character := t.input[t.position]
		if character == '\'' {
			t.position++
			return token{kind: tokenBlobLiteral, value: builder.String(), position: startPosition}, nil
		}
		builder.WriteByte(character)
		t.position++
	}

	return token{}, fmt.Errorf("unterminated blob literal at position %d", startPosition)
}

var (
	// twoCharOperators lists every two-character SQL operator recognised by the lexer. It is
	// package-level so readOperator does not allocate a fresh map on each operator token.
	twoCharOperators = map[string]bool{
		"<=": true, ">=": true, "<>": true, "!=": true, "||": true,
		"<<": true, ">>": true,
	}
)

// readOperator lexes a one- or two-character operator including the JSON arrow operators.
//
// Returns token which is the lexed operator token.
// Returns error when the current byte is not a recognised operator.
func (t *tokeniser) readOperator() (token, error) {
	startPosition := t.position
	character := t.input[t.position]

	if character == '-' && t.position+1 < len(t.input) && t.input[t.position+1] == '>' {
		if t.position+2 < len(t.input) && t.input[t.position+2] == '>' {
			t.position += doubleArrowOperatorLength
			return token{kind: tokenDoubleArrow, value: "->>", position: startPosition}, nil
		}
		t.position += 2
		return token{kind: tokenArrow, value: "->", position: startPosition}, nil
	}

	if t.position+1 < len(t.input) {
		twoChar := t.input[t.position : t.position+2]
		if twoCharOperators[twoChar] {
			t.position += 2
			return token{kind: tokenOperator, value: twoChar, position: startPosition}, nil
		}
	}

	singleCharOps := "=<>+-/%~&|!"
	if strings.ContainsRune(singleCharOps, rune(character)) {
		t.position++
		return token{kind: tokenOperator, value: string(character), position: startPosition}, nil
	}

	return token{}, fmt.Errorf("unexpected character %q at position %d", string(character), startPosition)
}

// isDigit reports whether character is an ASCII decimal digit.
//
// Takes character (byte) which is the byte to test.
//
// Returns bool which is true when character is '0' through '9'.
func isDigit(character byte) bool {
	return character >= '0' && character <= '9'
}

// isIdentStart reports whether the rune may start an unquoted identifier.
//
// Runes below utf8.RuneSelf take an ASCII fast-path covering letters and the underscore;
// any other rune is accepted when unicode.IsLetter reports it as a letter, so non-ASCII
// identifiers using accented or non-Latin letters tokenise correctly.
//
// Takes character (rune) which is the decoded code point to test.
//
// Returns bool which is true for ASCII letters, the underscore, or any Unicode letter.
func isIdentStart(character rune) bool {
	return engine_shared.IsIdentStart(character)
}

// isIdentPart reports whether the rune may continue an unquoted identifier after the
// first character.
//
// This extends the identifier-start set with decimal digits, so for runes below
// utf8.RuneSelf the ASCII digit check is inlined, while any other rune is accepted when
// unicode.IsLetter or unicode.IsDigit reports it.
//
// Takes character (rune) which is the decoded code point to test.
//
// Returns bool which is true for identifier starts and decimal digits.
func isIdentPart(character rune) bool {
	return engine_shared.IsIdentPart(character)
}
