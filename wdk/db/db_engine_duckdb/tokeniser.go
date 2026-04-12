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

package db_engine_duckdb

import (
	"fmt"
	"strings"
	"unicode"
)

// tokenKind classifies a single lexical token in a SQL statement.
type tokenKind uint8

const (
	// tokenIdentifier denotes an unquoted or quoted SQL identifier.
	tokenIdentifier tokenKind = iota

	// tokenNumber denotes a numeric literal.
	tokenNumber

	// tokenString denotes a single-quoted string literal.
	tokenString

	// tokenBitString denotes a B'...' bit-string literal.
	tokenBitString

	// tokenOperator denotes an operator token such as "=" or "<=".
	tokenOperator

	// tokenLeftParen denotes the "(" punctuation token.
	tokenLeftParen

	// tokenRightParen denotes the ")" punctuation token.
	tokenRightParen

	// tokenLeftBracket denotes the "[" punctuation token.
	tokenLeftBracket

	// tokenRightBracket denotes the "]" punctuation token.
	tokenRightBracket

	// tokenComma denotes the "," punctuation token.
	tokenComma

	// tokenSemicolon denotes the ";" statement terminator token.
	tokenSemicolon

	// tokenDot denotes the "." punctuation token.
	tokenDot

	// tokenStar denotes the "*" punctuation or wildcard token.
	tokenStar

	// tokenDollarParam denotes a $N positional parameter reference.
	tokenDollarParam

	// tokenNamedParam denotes a :name named parameter reference.
	tokenNamedParam

	// tokenCast denotes the "::" cast operator.
	tokenCast

	// tokenArrow denotes the "->" JSON arrow operator.
	tokenArrow

	// tokenDoubleArrow denotes the "->>" JSON double-arrow operator.
	tokenDoubleArrow

	// tokenEOF denotes the end-of-input sentinel token.
	tokenEOF
)
const (

	// maxASCII caps the inclusive upper bound of the 7-bit ASCII byte range.
	maxASCII = 127
)

// token holds a single lexed SQL token along with its source position.
type token struct {
	// value is the raw textual content of the token as it appears in input.
	value string

	// position is the zero-based byte offset of the token in the input.
	position int

	// kind classifies the token's lexical category.
	kind tokenKind
}

// tokeniser walks the SQL input string and produces a sequence of tokens.
type tokeniser struct {
	// input is the SQL source text being scanned.
	input string

	// position is the current zero-based byte offset within input.
	position int
}

// tokenise splits a SQL string into a sequence of tokens.
//
// Takes input (string) which is the SQL source text to tokenise.
//
// Returns []token which is the ordered token stream ending with tokenEOF.
// Returns error when an unterminated literal or unexpected character is found.
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

// next reads and returns the next token from the input stream.
//
// Returns token which is the next lexed token, or tokenEOF at end of input.
// Returns error when an unterminated literal or unexpected character is found.
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

var (

	// singleCharTokens maps an ASCII byte to its token kind plus one, where zero indicates
	// the byte does not start a single-character token.
	singleCharTokens = [256]tokenKind{}
)

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

// readSingleCharToken attempts to consume a one-byte punctuation token.
//
// Takes character (byte) which is the current byte at the cursor.
//
// Returns token which is the produced punctuation token when matched.
// Returns bool which is true when a punctuation token was produced.
func (t *tokeniser) readSingleCharToken(character byte) (token, bool) {
	mapped := singleCharTokens[character]
	if mapped == 0 {
		return token{}, false
	}
	startPosition := t.position
	t.position++
	return token{kind: mapped - 1, value: string(character), position: startPosition}, true
}

// readMultiCharToken dispatches to the appropriate reader for non-trivial token starts
// such as identifiers, numbers, and string literals.
//
// Takes character (byte) which is the current byte at the cursor.
//
// Returns token which is the lexed token produced by the dispatched reader.
// Returns error when the dispatched reader fails to recognise the input.
func (t *tokeniser) readMultiCharToken(character byte) (token, error) {
	switch {
	case character == '$':
		return t.readDollarParam()
	case character == ':':
		return t.readColonToken()
	case character == '\'':
		return t.readString()
	case character == '"':
		return t.readQuotedIdentifier()
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

// skipWhitespaceAndComments advances the cursor past whitespace, line comments, and block
// comments in the input.
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

// skipLineComment advances the cursor past a "--" comment to end of line.
func (t *tokeniser) skipLineComment() {
	t.position += 2
	for t.position < len(t.input) && t.input[t.position] != '\n' {
		t.position++
	}
}

// skipBlockComment advances the cursor past a nested "/* ... */" comment.
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

// readString reads a single-quoted string literal.
//
// Returns token which is the lexed tokenString.
// Returns error when the literal is not terminated before end of input.
func (t *tokeniser) readString() (token, error) {
	return t.readDelimitedLiteral('\'', tokenString, "string literal")
}

// readDelimitedLiteral reads a literal bounded by a single-byte delimiter, treating a
// doubled delimiter as an embedded literal occurrence.
//
// Takes delimiter (byte) which is the quoting byte that opens and closes the literal.
// Takes kind (tokenKind) which is the token kind to assign to the result.
// Takes errorDescription (string) which names the literal in error messages.
//
// Returns token which is the lexed token of the requested kind.
// Returns error when the literal is not terminated before end of input.
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

// readBitString reads a B'...' bit-string literal.
//
// Returns token which is the lexed tokenBitString.
// Returns error when the literal is not terminated before end of input.
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

// readQuotedIdentifier reads a double-quoted SQL identifier.
//
// Returns token which is the lexed tokenIdentifier with quoted content.
// Returns error when the identifier is not terminated before end of input.
func (t *tokeniser) readQuotedIdentifier() (token, error) {
	return t.readDelimitedLiteral('"', tokenIdentifier, "quoted identifier")
}

// readNumber reads a numeric literal including hex, octal, binary, decimal, fractional
// and exponent forms.
//
// Returns token which is the lexed tokenNumber.
// Returns error when no valid numeric form can be recognised.
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

// tryReadPrefixedNumber attempts to read a 0x, 0o, or 0b prefixed integer.
//
// Takes startPosition (int) which is the start byte offset of the candidate literal.
//
// Returns token which is the lexed tokenNumber when a prefix matches.
// Returns bool which is true when a prefixed literal was consumed.
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

// prefixedNumberValidator returns a digit predicate appropriate for the numeric base
// implied by the prefix byte.
//
// Takes prefix (byte) which is the byte following the leading "0".
//
// Returns func(byte) bool which validates digits for the matching base, or nil when the
// prefix does not introduce a known base.
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

// consumeWhile advances the cursor while the predicate accepts the current byte of input.
//
// Takes predicate (func(byte) bool) which decides whether to keep consuming.
func (t *tokeniser) consumeWhile(predicate func(byte) bool) {
	for t.position < len(t.input) && predicate(t.input[t.position]) {
		t.position++
	}
}

// isOctalDigit reports whether the byte is an ASCII octal digit.
//
// Takes character (byte) which is the byte to classify.
//
// Returns bool which is true for bytes in the range '0' to '7'.
func isOctalDigit(character byte) bool {
	return character >= '0' && character <= '7'
}

// isBinaryDigit reports whether the byte is an ASCII binary digit.
//
// Takes character (byte) which is the byte to classify.
//
// Returns bool which is true for '0' or '1'.
func isBinaryDigit(character byte) bool {
	return character == '0' || character == '1'
}

// consumeDigits advances the cursor over a run of decimal digit bytes.
func (t *tokeniser) consumeDigits() {
	for t.position < len(t.input) && isDigit(t.input[t.position]) {
		t.position++
	}
}

// consumeFractionalPart advances the cursor over a "." followed by digits.
func (t *tokeniser) consumeFractionalPart() {
	if t.position >= len(t.input) || t.input[t.position] != '.' {
		return
	}
	t.position++
	t.consumeDigits()
}

// consumeExponentPart advances the cursor over an "e" or "E" exponent suffix with an
// optional sign and digits.
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

// readIdentifier reads an unquoted SQL identifier.
//
// Returns token which is the lexed tokenIdentifier.
// Returns error which is always nil for the unquoted identifier path.
func (t *tokeniser) readIdentifier() (token, error) {
	startPosition := t.position
	for t.position < len(t.input) && isIdentPart(t.input[t.position]) {
		t.position++
	}
	return token{kind: tokenIdentifier, value: t.input[startPosition:t.position], position: startPosition}, nil
}

// readDollarParam reads a "$N" positional parameter reference.
//
// Returns token which is the lexed tokenDollarParam.
// Returns error when "$" is not followed by a digit.
func (t *tokeniser) readDollarParam() (token, error) {
	startPosition := t.position

	if t.position+1 < len(t.input) && isDigit(t.input[t.position+1]) {
		t.position++
		for t.position < len(t.input) && isDigit(t.input[t.position]) {
			t.position++
		}
		return token{kind: tokenDollarParam, value: t.input[startPosition:t.position], position: startPosition}, nil
	}

	return token{}, fmt.Errorf("unexpected character $ at position %d", startPosition)
}

// readColonToken reads a "::" cast, a ":name" parameter, or a single ":" operator
// depending on what follows the colon.
//
// Returns token which is the lexed token of the matching kind.
// Returns error which is always nil for the colon dispatch path.
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

// readOperator reads an operator token, trying arrow forms and multi-byte operators
// before falling back to a single-byte operator.
//
// Returns token which is the lexed tokenOperator or arrow variant.
// Returns error when the byte does not begin any recognised operator.
func (t *tokeniser) readOperator() (token, error) {
	startPosition := t.position
	character := t.input[t.position]

	if tok, ok := t.readArrowOperator(character, startPosition); ok {
		return tok, nil
	}

	if tok, ok := t.readTwoOrThreeCharOperator(startPosition); ok {
		return tok, nil
	}

	return t.readSingleCharOperator(character, startPosition)
}

// readArrowOperator reads the "->" or "->>" JSON arrow operator.
//
// Takes character (byte) which is the current byte at the cursor.
// Takes startPosition (int) which is the start byte offset of the operator.
//
// Returns token which is the lexed tokenArrow or tokenDoubleArrow.
// Returns bool which is true when an arrow operator was consumed.
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

var (
	// twoCharOps lists every two-character SQL operator recognised by the lexer.
	twoCharOps = map[string]bool{
		"<=": true, ">=": true, "<>": true, "!=": true, "||": true,
		"<<": true, ">>": true, "&&": true, "@>": true, "<@": true,
		"~*": true, "!~": true,
	}
)

// readTwoOrThreeCharOperator reads a multi-byte operator such as "<=" or "!~*" when one
// matches at the cursor.
//
// Takes startPosition (int) which is the start byte offset of the operator.
//
// Returns token which is the lexed tokenOperator when matched.
// Returns bool which is true when a multi-byte operator was consumed.
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

// readSingleCharOperator reads a one-byte operator such as "=" or "+".
//
// Takes character (byte) which is the current byte at the cursor.
// Takes startPosition (int) which is the start byte offset of the operator.
//
// Returns token which is the lexed tokenOperator.
// Returns error when the byte is not a recognised single-byte operator.
func (t *tokeniser) readSingleCharOperator(character byte, startPosition int) (token, error) {
	singleCharOps := "=<>+-/%~&|!^"
	if strings.ContainsRune(singleCharOps, rune(character)) {
		t.position++
		return token{kind: tokenOperator, value: string(character), position: startPosition}, nil
	}

	return token{}, fmt.Errorf("unexpected character %q at position %d", string(character), startPosition)
}

// isDigit reports whether the byte is an ASCII decimal digit.
//
// Takes character (byte) which is the byte to classify.
//
// Returns bool which is true for bytes in the range '0' to '9'.
func isDigit(character byte) bool {
	return character >= '0' && character <= '9'
}

// isHexDigit reports whether the byte is an ASCII hexadecimal digit.
//
// Takes character (byte) which is the byte to classify.
//
// Returns bool which is true for '0'-'9', 'a'-'f', or 'A'-'F'.
func isHexDigit(character byte) bool {
	return isDigit(character) ||
		(character >= 'a' && character <= 'f') ||
		(character >= 'A' && character <= 'F')
}

// isIdentStart reports whether the byte can start an unquoted identifier.
//
// Takes character (byte) which is the byte to classify.
//
// Returns bool which is true for letters, underscore, or non-ASCII letters.
func isIdentStart(character byte) bool {
	return (character >= 'a' && character <= 'z') ||
		(character >= 'A' && character <= 'Z') ||
		character == '_' ||
		character > maxASCII && unicode.IsLetter(rune(character))
}

// isIdentPart reports whether the byte can continue an unquoted identifier.
//
// Takes character (byte) which is the byte to classify.
//
// Returns bool which is true for identifier-start bytes or decimal digits.
func isIdentPart(character byte) bool {
	return isIdentStart(character) || isDigit(character)
}
