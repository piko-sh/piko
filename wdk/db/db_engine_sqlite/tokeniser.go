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
	"unicode"
)

type tokenKind uint8

const (
	tokenIdentifier tokenKind = iota

	tokenNumber

	tokenString

	tokenBlobLiteral

	tokenOperator

	tokenLeftParen

	tokenRightParen

	tokenComma

	tokenSemicolon

	tokenDot

	tokenStar

	tokenQuestionMark

	tokenNumberedParam

	tokenNamedParam

	tokenArrow

	tokenDoubleArrow

	tokenEOF
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

		if t.skipLineComment() {
			continue
		}

		if t.skipBlockComment() {
			continue
		}

		break
	}
}

func (t *tokeniser) skipLineComment() bool {
	if t.position+1 >= len(t.input) || t.input[t.position] != '-' || t.input[t.position+1] != '-' {
		return false
	}
	t.position += 2
	for t.position < len(t.input) && t.input[t.position] != '\n' {
		t.position++
	}
	return true
}

func (t *tokeniser) skipBlockComment() bool {
	if t.position+1 >= len(t.input) || t.input[t.position] != '/' || t.input[t.position+1] != '*' {
		return false
	}
	t.position += 2
	for t.position+1 < len(t.input) {
		if t.input[t.position] == '*' && t.input[t.position+1] == '/' {
			t.position += 2
			return true
		}
		t.position++
	}
	return true
}

func (t *tokeniser) readString() (token, error) {
	startPosition := t.position
	t.position++

	var builder strings.Builder
	for t.position < len(t.input) {
		character := t.input[t.position]
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

func (t *tokeniser) readQuotedIdentifier(quote byte) (token, error) {
	startPosition := t.position
	t.position++

	var builder strings.Builder
	for t.position < len(t.input) {
		character := t.input[t.position]
		if character == quote {
			t.position++
			if t.position < len(t.input) && t.input[t.position] == quote {
				builder.WriteByte(quote)
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

func (t *tokeniser) readNumber() (token, error) {
	startPosition := t.position

	if t.readHexPrefix() {
		t.consumeWhile(isHexDigit)
		return token{kind: tokenNumber, value: t.input[startPosition:t.position], position: startPosition}, nil
	}

	t.consumeWhile(isDigit)
	t.readFractionalPart()
	t.readExponentPart()

	return token{kind: tokenNumber, value: t.input[startPosition:t.position], position: startPosition}, nil
}

func (t *tokeniser) readHexPrefix() bool {
	if t.input[t.position] != '0' || t.position+1 >= len(t.input) {
		return false
	}
	next := t.input[t.position+1]
	if next != 'x' && next != 'X' {
		return false
	}
	t.position += 2
	return true
}

func (t *tokeniser) readFractionalPart() {
	if t.position >= len(t.input) || t.input[t.position] != '.' {
		return
	}
	t.position++
	t.consumeWhile(isDigit)
}

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

func (t *tokeniser) consumeWhile(predicate func(byte) bool) {
	for t.position < len(t.input) && predicate(t.input[t.position]) {
		t.position++
	}
}

func (t *tokeniser) readIdentifier() (token, error) {
	startPosition := t.position
	for t.position < len(t.input) && isIdentPart(t.input[t.position]) {
		t.position++
	}
	return token{kind: tokenIdentifier, value: t.input[startPosition:t.position], position: startPosition}, nil
}

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

func (t *tokeniser) readNamedParam() (token, error) {
	startPosition := t.position
	t.position++
	for t.position < len(t.input) && isIdentPart(t.input[t.position]) {
		t.position++
	}
	return token{kind: tokenNamedParam, value: t.input[startPosition:t.position], position: startPosition}, nil
}

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

	twoCharOps := map[string]bool{
		"<=": true, ">=": true, "<>": true, "!=": true, "||": true,
		"<<": true, ">>": true,
	}

	if t.position+1 < len(t.input) {
		twoChar := t.input[t.position : t.position+2]
		if twoCharOps[twoChar] {
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
		character > maxASCIICodePoint && unicode.IsLetter(rune(character))
}

func isIdentPart(character byte) bool {
	return isIdentStart(character) || isDigit(character)
}
