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

package db_engine_postgres

import (
	"fmt"
	"strings"
	"unicode"
)

type tokenKind uint8

const (
	tokenIdentifier tokenKind = iota

	tokenNumber

	tokenString

	tokenEscapeString

	tokenDollarString

	tokenBitString

	tokenOperator

	tokenLeftParen

	tokenRightParen

	tokenLeftBracket

	tokenRightBracket

	tokenComma

	tokenSemicolon

	tokenDot

	tokenStar

	tokenDollarParam

	tokenNamedParam

	tokenCast

	tokenArrow

	tokenDoubleArrow

	tokenHashArrow

	tokenHashDoubleArrow

	tokenEOF
)

const maxASCII = 127

type token struct {
	value string

	position int

	kind tokenKind
}

type tokeniser struct {
	input string

	position int
}

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

func (t *tokeniser) next() (token, error) {
	t.skipWhitespaceAndComments()

	if t.position >= len(t.input) {
		return token{kind: tokenEOF, position: t.position}, nil
	}

	character := t.input[t.position]

	if tok, ok := t.readSingleCharToken(character); ok {
		return tok, nil
	}

	return t.readMultiCharToken(character)
}

var singleCharTokens = [256]tokenKind{}

func init() {
	singleCharTokens['('] = tokenLeftParen + 1
	singleCharTokens[')'] = tokenRightParen + 1
	singleCharTokens['['] = tokenLeftBracket + 1
	singleCharTokens[']'] = tokenRightBracket + 1
	singleCharTokens[','] = tokenComma + 1
	singleCharTokens[';'] = tokenSemicolon + 1
	singleCharTokens['.'] = tokenDot + 1
	singleCharTokens['*'] = tokenStar + 1
}

func (t *tokeniser) readSingleCharToken(character byte) (token, bool) {
	mapped := singleCharTokens[character]
	if mapped == 0 {
		return token{}, false
	}
	startPosition := t.position
	t.position++
	return token{kind: mapped - 1, value: string(character), position: startPosition}, true
}

func (t *tokeniser) readMultiCharToken(character byte) (token, error) {
	switch {
	case character == '$':
		return t.readDollarToken()
	case character == ':':
		return t.readColonToken()
	case character == '\'':
		return t.readString()
	case character == '"':
		return t.readQuotedIdentifier()
	case (character == 'E' || character == 'e') && t.position+1 < len(t.input) && t.input[t.position+1] == '\'':
		return t.readEscapeString()
	case (character == 'B' || character == 'b') && t.position+1 < len(t.input) && t.input[t.position+1] == '\'':
		return t.readBitString()
	case isDigit(character):
		return t.readNumber()
	case isIdentStart(character):
		return t.readIdentifier()
	default:
		return t.readOperator()
	}
}

func (t *tokeniser) skipWhitespaceAndComments() {
	for t.position < len(t.input) {
		character := t.input[t.position]

		if character == ' ' || character == '\t' || character == '\n' || character == '\r' {
			t.position++
			continue
		}

		if character == '-' && t.position+1 < len(t.input) && t.input[t.position+1] == '-' {
			t.skipLineComment()
			continue
		}

		if character == '/' && t.position+1 < len(t.input) && t.input[t.position+1] == '*' {
			t.skipBlockComment()
			continue
		}

		break
	}
}

func (t *tokeniser) skipLineComment() {
	t.position += 2
	for t.position < len(t.input) && t.input[t.position] != '\n' {
		t.position++
	}
}

func (t *tokeniser) skipBlockComment() {
	t.position += 2
	depth := 1
	for t.position+1 < len(t.input) && depth > 0 {
		if t.input[t.position] == '/' && t.input[t.position+1] == '*' {
			depth++
			t.position += 2
			continue
		}
		if t.input[t.position] == '*' && t.input[t.position+1] == '/' {
			depth--
			t.position += 2
			continue
		}
		t.position++
	}
}

func (t *tokeniser) readString() (token, error) {
	return t.readDelimitedLiteral('\'', tokenString, "string literal")
}

func (t *tokeniser) readDelimitedLiteral(delimiter byte, kind tokenKind, errorDescription string) (token, error) {
	startPosition := t.position
	t.position++

	var builder strings.Builder
	for t.position < len(t.input) {
		character := t.input[t.position]
		if character == delimiter {
			t.position++
			if t.position < len(t.input) && t.input[t.position] == delimiter {
				builder.WriteByte(delimiter)
				t.position++
				continue
			}
			return token{kind: kind, value: builder.String(), position: startPosition}, nil
		}
		builder.WriteByte(character)
		t.position++
	}

	return token{}, fmt.Errorf("unterminated %s at position %d", errorDescription, startPosition)
}

func (t *tokeniser) readEscapeString() (token, error) {
	startPosition := t.position
	t.position += 2

	var builder strings.Builder
	for t.position < len(t.input) {
		character := t.input[t.position]
		if character == '\\' && t.position+1 < len(t.input) {
			t.position++
			builder.WriteByte(t.input[t.position])
			t.position++
			continue
		}
		if character == '\'' {
			t.position++
			if t.position < len(t.input) && t.input[t.position] == '\'' {
				builder.WriteByte('\'')
				t.position++
				continue
			}
			return token{kind: tokenEscapeString, value: builder.String(), position: startPosition}, nil
		}
		builder.WriteByte(character)
		t.position++
	}

	return token{}, fmt.Errorf("unterminated escape string at position %d", startPosition)
}

func (t *tokeniser) readBitString() (token, error) {
	startPosition := t.position
	t.position += 2

	var builder strings.Builder
	for t.position < len(t.input) {
		character := t.input[t.position]
		if character == '\'' {
			t.position++
			return token{kind: tokenBitString, value: builder.String(), position: startPosition}, nil
		}
		builder.WriteByte(character)
		t.position++
	}

	return token{}, fmt.Errorf("unterminated bit string at position %d", startPosition)
}

func (t *tokeniser) readQuotedIdentifier() (token, error) {
	return t.readDelimitedLiteral('"', tokenIdentifier, "quoted identifier")
}

func (t *tokeniser) readNumber() (token, error) {
	startPosition := t.position

	if tok, matched := t.tryReadPrefixedNumber(startPosition); matched {
		return tok, nil
	}

	t.consumeDigits()
	t.consumeFractionalPart()
	t.consumeExponentPart()

	return token{kind: tokenNumber, value: t.input[startPosition:t.position], position: startPosition}, nil
}

func (t *tokeniser) tryReadPrefixedNumber(startPosition int) (token, bool) {
	if t.input[t.position] != '0' || t.position+1 >= len(t.input) {
		return token{}, false
	}

	next := t.input[t.position+1]
	validator := prefixedNumberValidator(next)
	if validator == nil {
		return token{}, false
	}

	t.position += 2
	t.consumeWhile(validator)

	return token{kind: tokenNumber, value: t.input[startPosition:t.position], position: startPosition}, true
}

func prefixedNumberValidator(prefix byte) func(byte) bool {
	switch prefix {
	case 'x', 'X':
		return isHexDigit
	case 'o', 'O':
		return isOctalDigit
	case 'b', 'B':
		return isBinaryDigit
	default:
		return nil
	}
}

func (t *tokeniser) consumeWhile(predicate func(byte) bool) {
	for t.position < len(t.input) && predicate(t.input[t.position]) {
		t.position++
	}
}

func isOctalDigit(character byte) bool {
	return character >= '0' && character <= '7'
}

func isBinaryDigit(character byte) bool {
	return character == '0' || character == '1'
}

func (t *tokeniser) consumeDigits() {
	for t.position < len(t.input) && isDigit(t.input[t.position]) {
		t.position++
	}
}

func (t *tokeniser) consumeFractionalPart() {
	if t.position >= len(t.input) || t.input[t.position] != '.' {
		return
	}
	t.position++
	t.consumeDigits()
}

func (t *tokeniser) consumeExponentPart() {
	if t.position >= len(t.input) {
		return
	}
	if t.input[t.position] != 'e' && t.input[t.position] != 'E' {
		return
	}
	t.position++
	if t.position < len(t.input) && (t.input[t.position] == '+' || t.input[t.position] == '-') {
		t.position++
	}
	t.consumeDigits()
}

func (t *tokeniser) readIdentifier() (token, error) {
	startPosition := t.position
	for t.position < len(t.input) && isIdentPart(t.input[t.position]) {
		t.position++
	}
	return token{kind: tokenIdentifier, value: t.input[startPosition:t.position], position: startPosition}, nil
}

func (t *tokeniser) readDollarToken() (token, error) {
	startPosition := t.position

	if t.position+1 < len(t.input) && isDigit(t.input[t.position+1]) {
		t.position++
		for t.position < len(t.input) && isDigit(t.input[t.position]) {
			t.position++
		}
		return token{kind: tokenDollarParam, value: t.input[startPosition:t.position], position: startPosition}, nil
	}

	return t.readDollarQuotedString()
}

func (t *tokeniser) readDollarQuotedString() (token, error) {
	startPosition := t.position
	t.position++

	var tag string
	if t.position < len(t.input) && t.input[t.position] == '$' {
		t.position++
		tag = ""
	} else if t.position < len(t.input) && isIdentStart(t.input[t.position]) {
		tagStart := t.position
		for t.position < len(t.input) && isIdentPart(t.input[t.position]) {
			t.position++
		}
		if t.position >= len(t.input) || t.input[t.position] != '$' {
			return token{}, fmt.Errorf("invalid dollar-quoted string at position %d", startPosition)
		}
		tag = t.input[tagStart:t.position]
		t.position++
	} else {
		return token{}, fmt.Errorf("unexpected character after $ at position %d", startPosition)
	}

	endDelimiter := "$" + tag + "$"
	var builder strings.Builder
	for t.position < len(t.input) {
		if t.input[t.position] == '$' && strings.HasPrefix(t.input[t.position:], endDelimiter) {
			t.position += len(endDelimiter)
			return token{kind: tokenDollarString, value: builder.String(), position: startPosition}, nil
		}
		builder.WriteByte(t.input[t.position])
		t.position++
	}

	return token{}, fmt.Errorf("unterminated dollar-quoted string at position %d", startPosition)
}

func (t *tokeniser) readColonToken() (token, error) {
	startPosition := t.position

	if t.position+1 < len(t.input) && t.input[t.position+1] == ':' {
		t.position += 2
		return token{kind: tokenCast, value: "::", position: startPosition}, nil
	}

	if t.position+1 < len(t.input) && isIdentStart(t.input[t.position+1]) {
		t.position++
		for t.position < len(t.input) && isIdentPart(t.input[t.position]) {
			t.position++
		}
		return token{kind: tokenNamedParam, value: t.input[startPosition:t.position], position: startPosition}, nil
	}

	t.position++
	return token{kind: tokenOperator, value: ":", position: startPosition}, nil
}

func (t *tokeniser) readOperator() (token, error) {
	startPosition := t.position
	character := t.input[t.position]

	if tok, ok := t.readArrowOperator(character, startPosition); ok {
		return tok, nil
	}

	if tok, ok := t.readHashArrowOperator(character, startPosition); ok {
		return tok, nil
	}

	if tok, ok := t.readTwoOrThreeCharOperator(startPosition); ok {
		return tok, nil
	}

	return t.readSingleCharOperator(character, startPosition)
}

func (t *tokeniser) readArrowOperator(character byte, startPosition int) (token, bool) {
	return t.readArrowLikeOperator(character, '-', startPosition, tokenArrow, "->", tokenDoubleArrow, "->>")
}

func (t *tokeniser) readHashArrowOperator(character byte, startPosition int) (token, bool) {
	return t.readArrowLikeOperator(character, '#', startPosition, tokenHashArrow, "#>", tokenHashDoubleArrow, "#>>")
}

func (t *tokeniser) readArrowLikeOperator(
	character byte,
	prefix byte,
	startPosition int,
	singleKind tokenKind,
	singleValue string,
	doubleKind tokenKind,
	doubleValue string,
) (token, bool) {
	if character != prefix || t.position+1 >= len(t.input) || t.input[t.position+1] != '>' {
		return token{}, false
	}
	if t.position+2 < len(t.input) && t.input[t.position+2] == '>' {
		t.position += len(doubleValue)
		return token{kind: doubleKind, value: doubleValue, position: startPosition}, true
	}
	t.position += len(singleValue)
	return token{kind: singleKind, value: singleValue, position: startPosition}, true
}

var twoCharOps = map[string]bool{
	"<=": true, ">=": true, "<>": true, "!=": true, "||": true,
	"<<": true, ">>": true, "&&": true, "@>": true, "<@": true,
	"~*": true, "!~": true,
}

func (t *tokeniser) readTwoOrThreeCharOperator(startPosition int) (token, bool) {
	if t.position+1 >= len(t.input) {
		return token{}, false
	}

	twoChar := t.input[t.position : t.position+2]

	if twoChar == "!~" && t.position+2 < len(t.input) && t.input[t.position+2] == '*' {
		t.position += 3 //nolint:revive // 3-char operator
		return token{kind: tokenOperator, value: "!~*", position: startPosition}, true
	}

	if twoCharOps[twoChar] {
		t.position += 2
		return token{kind: tokenOperator, value: twoChar, position: startPosition}, true
	}

	return token{}, false
}

func (t *tokeniser) readSingleCharOperator(character byte, startPosition int) (token, error) {
	singleCharOps := "=<>+-/%~&|!#^"
	if strings.ContainsRune(singleCharOps, rune(character)) {
		t.position++
		return token{kind: tokenOperator, value: string(character), position: startPosition}, nil
	}

	return token{}, fmt.Errorf("unexpected character %q at position %d", string(character), startPosition)
}

func isDigit(character byte) bool {
	return character >= '0' && character <= '9'
}

func isHexDigit(character byte) bool {
	return isDigit(character) ||
		(character >= 'a' && character <= 'f') ||
		(character >= 'A' && character <= 'F')
}

func isIdentStart(character byte) bool {
	return (character >= 'a' && character <= 'z') ||
		(character >= 'A' && character <= 'Z') ||
		character == '_' ||
		character > maxASCII && unicode.IsLetter(rune(character))
}

func isIdentPart(character byte) bool {
	return isIdentStart(character) || isDigit(character)
}
