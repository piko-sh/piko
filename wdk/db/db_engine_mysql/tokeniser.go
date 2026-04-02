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

package db_engine_mysql

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

	tokenHexString

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

	tokenQuestionMark

	tokenNamedParam

	tokenArrow

	tokenDoubleArrow

	tokenUserVariable

	tokenSystemVariable

	tokenEOF
)

const (
	maxASCII = 127

	substituteCharacter = 0x1A
)

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
	singleCharTokens['?'] = tokenQuestionMark + 1
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
	case character == '@':
		return t.readAtToken()
	case character == ':':
		return t.readColonToken()
	case character == '\'':
		return t.readString()
	case character == '"':
		return t.readQuotedIdentifier()
	case character == '`':
		return t.readBacktickIdentifier()
	case (character == 'X' || character == 'x') && t.position+1 < len(t.input) && t.input[t.position+1] == '\'':
		return t.readHexString()
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

		if character == '-' && t.position+1 < len(t.input) && t.input[t.position+1] == '-' &&
			t.position+2 < len(t.input) && (t.input[t.position+2] == ' ' || t.input[t.position+2] == '\t') {
			t.skipLineComment()
			continue
		}

		if character == '#' {
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
	for t.position < len(t.input) && t.input[t.position] != '\n' {
		t.position++
	}
}

func (t *tokeniser) skipBlockComment() {
	t.position += 2
	for t.position+1 < len(t.input) {
		if t.input[t.position] == '*' && t.input[t.position+1] == '/' {
			t.position += 2
			return
		}
		t.position++
	}
	t.position = len(t.input)
}

func (t *tokeniser) readString() (token, error) {
	startPosition := t.position
	t.position++

	var builder strings.Builder
	for t.position < len(t.input) {
		character := t.input[t.position]
		if character == '\\' && t.position+1 < len(t.input) {
			t.position++
			builder.WriteByte(mysqlBackslashEscape(t.input[t.position]))
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
			return token{kind: tokenString, value: builder.String(), position: startPosition}, nil
		}
		builder.WriteByte(character)
		t.position++
	}

	return token{}, fmt.Errorf("unterminated string literal at position %d", startPosition)
}

func mysqlBackslashEscape(character byte) byte {
	switch character {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'r':
		return '\r'
	case '0':
		return 0
	case 'Z':
		return substituteCharacter
	default:
		return character
	}
}

func (t *tokeniser) readHexString() (token, error) {
	startPosition := t.position
	t.position += 2

	var builder strings.Builder
	for t.position < len(t.input) {
		character := t.input[t.position]
		if character == '\'' {
			t.position++
			return token{kind: tokenHexString, value: builder.String(), position: startPosition}, nil
		}
		builder.WriteByte(character)
		t.position++
	}

	return token{}, fmt.Errorf("unterminated hex string at position %d", startPosition)
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

func (t *tokeniser) readBacktickIdentifier() (token, error) {
	return t.readDelimitedIdentifier('`')
}

func (t *tokeniser) readQuotedIdentifier() (token, error) {
	return t.readDelimitedIdentifier('"')
}

func (t *tokeniser) readDelimitedIdentifier(delimiter byte) (token, error) {
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
			return token{kind: tokenIdentifier, value: builder.String(), position: startPosition}, nil
		}
		builder.WriteByte(character)
		t.position++
	}

	return token{}, fmt.Errorf("unterminated quoted identifier at position %d", startPosition)
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

func (t *tokeniser) readAtToken() (token, error) {
	startPosition := t.position
	t.position++

	if t.position < len(t.input) && t.input[t.position] == '@' {
		t.position++
		t.consumeSystemVariableQualifier()
		t.consumeVariableName()
		return token{kind: tokenSystemVariable, value: t.input[startPosition:t.position], position: startPosition}, nil
	}

	t.consumeVariableName()
	return token{kind: tokenUserVariable, value: t.input[startPosition:t.position], position: startPosition}, nil
}

func (t *tokeniser) consumeSystemVariableQualifier() {
	if t.position >= len(t.input) || !isIdentStart(t.input[t.position]) {
		return
	}

	saved := t.position
	for t.position < len(t.input) && isIdentPart(t.input[t.position]) {
		t.position++
	}
	if t.position < len(t.input) && t.input[t.position] == '.' {
		qualifier := t.input[saved:t.position]
		if qualifier == "global" || qualifier == "session" || qualifier == "local" {
			t.position++
			return
		}
	}
	t.position = saved
}

func (t *tokeniser) consumeVariableName() {
	for t.position < len(t.input) && isIdentPart(t.input[t.position]) {
		t.position++
	}
}

func (t *tokeniser) readColonToken() (token, error) {
	startPosition := t.position

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

	if tok, ok := t.readThreeCharOperator(startPosition); ok {
		return tok, nil
	}

	if tok, ok := t.readTwoCharOperator(startPosition); ok {
		return tok, nil
	}

	return t.readSingleCharOperator(character, startPosition)
}

func (t *tokeniser) readArrowOperator(character byte, startPosition int) (token, bool) {
	if character != '-' || t.position+1 >= len(t.input) || t.input[t.position+1] != '>' {
		return token{}, false
	}
	if t.position+2 < len(t.input) && t.input[t.position+2] == '>' {
		t.position += 3 //nolint:revive // 3-char operator
		return token{kind: tokenDoubleArrow, value: "->>", position: startPosition}, true
	}
	t.position += 2
	return token{kind: tokenArrow, value: "->", position: startPosition}, true
}

func (t *tokeniser) readThreeCharOperator(startPosition int) (token, bool) {
	if t.position+2 >= len(t.input) {
		return token{}, false
	}

	threeChar := t.input[t.position : t.position+3] //nolint:revive // 3-char slice
	if threeChar == "<=>" {
		t.position += 3 //nolint:revive // 3-char operator
		return token{kind: tokenOperator, value: "<=>", position: startPosition}, true
	}

	return token{}, false
}

var twoCharOps = map[string]bool{
	"<=": true, ">=": true, "<>": true, "!=": true, "||": true,
	"<<": true, ">>": true,
}

func (t *tokeniser) readTwoCharOperator(startPosition int) (token, bool) {
	if t.position+1 >= len(t.input) {
		return token{}, false
	}

	twoChar := t.input[t.position : t.position+2]
	if twoCharOps[twoChar] {
		t.position += 2
		return token{kind: tokenOperator, value: twoChar, position: startPosition}, true
	}

	return token{}, false
}

func (t *tokeniser) readSingleCharOperator(character byte, startPosition int) (token, error) {
	singleCharOps := "=<>+-/%~&|!^"
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
