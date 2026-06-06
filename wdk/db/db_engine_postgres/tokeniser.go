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
	"unicode/utf8"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
)

// tokenKind enumerates the lexical token categories produced by the tokeniser.
type tokenKind uint8

const (
	// tokenIdentifier is an unquoted or double-quoted identifier.
	tokenIdentifier tokenKind = iota

	// tokenNumber is a numeric literal.
	tokenNumber

	// tokenString is a single-quoted string literal.
	tokenString

	// tokenEscapeString is an E'...' escape string literal.
	tokenEscapeString

	// tokenDollarString is a dollar-quoted string literal.
	tokenDollarString

	// tokenBitString is a B'...' bit string literal.
	tokenBitString

	// tokenOperator is a SQL operator token.
	tokenOperator

	// tokenLeftParen is the '(' token.
	tokenLeftParen

	// tokenRightParen is the ')' token.
	tokenRightParen

	// tokenLeftBracket is the '[' token.
	tokenLeftBracket

	// tokenRightBracket is the ']' token.
	tokenRightBracket

	// tokenComma is the ',' token.
	tokenComma

	// tokenSemicolon is the ';' token.
	tokenSemicolon

	// tokenDot is the '.' token.
	tokenDot

	// tokenStar is the '*' token.
	tokenStar

	// tokenDollarParam is a positional parameter such as $1.
	tokenDollarParam

	// tokenNamedParam is a named parameter such as :name.
	tokenNamedParam

	// tokenCast is the '::' cast operator.
	tokenCast

	// tokenArrow is the '->' JSON access operator.
	tokenArrow

	// tokenDoubleArrow is the '->>' JSON access operator.
	tokenDoubleArrow

	// tokenHashArrow is the '#>' JSON path operator.
	tokenHashArrow

	// tokenHashDoubleArrow is the '#>>' JSON path operator.
	tokenHashDoubleArrow

	// tokenEOF marks the end of input.
	tokenEOF
)

// token is a single lexical token produced by the tokeniser.
type token struct {
	// value is the textual content of the token.
	value string

	// position is the byte offset of the token within the input.
	position int

	// kind classifies the token lexically.
	kind tokenKind
}

// tokeniser scans an input SQL string and emits tokens on demand.
type tokeniser struct {
	// input is the source SQL being scanned.
	input string

	// position is the current byte offset within input.
	position int
}

var (
	// dialectConfig declares Postgres lexical rules for the shared scanners: nested block
	// comments and 0x/0o/0b base-prefixed integer literals.
	dialectConfig = engine_shared.DialectConfig{
		Comments: engine_shared.CommentRules{NestedBlockComments: true},
		Numbers:  engine_shared.NumberRules{HexPrefix: true, OctalPrefix: true, BinaryPrefix: true},
	}
)

// tokenise scans input and returns the full token stream.
//
// Takes input (string) which is the SQL text to scan.
//
// Returns []token which is the ordered stream of tokens ending in tokenEOF.
// Returns error when a lexical error occurs.
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

// next advances and returns the next token from the input.
//
// Returns token which is the scanned token.
// Returns error when a lexical error occurs.
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

	// singleCharTokens maps an ASCII byte to its tokenKind (biased by +1 so the zero value
	// indicates "no mapping").
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

// readSingleCharToken returns the single-character token for character if one is mapped.
//
// Takes character (byte) which is the candidate byte to classify.
//
// Returns token which is the produced token when ok is true.
// Returns bool which is true when character maps to a single-char token.
func (t *tokeniser) readSingleCharToken(character byte) (token, bool) {
	mapped := singleCharTokens[character]
	if mapped == 0 {
		return token{}, false
	}
	startPosition := t.position
	t.position++
	return token{kind: mapped - 1, value: string(character), position: startPosition}, true
}

// readMultiCharToken dispatches to the appropriate multi-character reader.
//
// Takes character (byte) which is the leading byte of the token.
//
// Returns token which is the scanned token.
// Returns error when no reader accepts the input.
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
	}

	leading, _ := utf8.DecodeRuneInString(t.input[t.position:])
	if isIdentStart(leading) {
		return t.readIdentifier()
	}
	return t.readOperator()
}

// skipWhitespaceAndComments advances past whitespace and comments.
//
// Returns error when a block comment is left unterminated.
func (t *tokeniser) skipWhitespaceAndComments() error {
	position, err := dialectConfig.Comments.SkipWhitespaceAndComments(t.input, t.position)
	t.position = position
	return err
}

// readString reads a single-quoted string literal.
//
// Returns token which is the string literal token.
// Returns error when the literal is unterminated.
func (t *tokeniser) readString() (token, error) {
	return t.readDelimitedLiteral('\'', tokenString, "string literal")
}

// readDelimitedLiteral reads a literal delimited by delimiter on both sides.
//
// Doubled delimiters inside the literal are treated as escaped delimiters.
//
// Takes delimiter (byte) which opens and closes the literal.
// Takes kind (tokenKind) which classifies the produced token.
// Takes errorDescription (string) which names the literal in error messages.
//
// Returns token which is the produced literal token.
// Returns error when the literal is unterminated.
func (t *tokeniser) readDelimitedLiteral(delimiter byte, kind tokenKind, errorDescription string) (token, error) {
	startPosition := t.position

	value, position, ok := engine_shared.ScanDoubledDelimiter(t.input, t.position, delimiter)
	if !ok {
		return token{}, fmt.Errorf("unterminated %s at position %d", errorDescription, startPosition)
	}
	t.position = position

	return token{kind: kind, value: value, position: startPosition}, nil
}

// readEscapeString reads an E'...' Postgres escape string literal.
//
// Returns token which is the escape string token.
// Returns error when the literal is unterminated.
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

// readBitString reads a B'...' bit string literal.
//
// Returns token which is the bit string token.
// Returns error when the literal is unterminated.
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

// readQuotedIdentifier reads a double-quoted identifier.
//
// Returns token which is the identifier token.
// Returns error when the identifier is unterminated.
func (t *tokeniser) readQuotedIdentifier() (token, error) {
	return t.readDelimitedLiteral('"', tokenIdentifier, "quoted identifier")
}

// readNumber reads a numeric literal, supporting prefixed bases.
//
// Returns token which is the number token.
// Returns error when scanning fails.
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

	t.consumeDigits()
	t.consumeFractionalPart()
	t.consumeExponentPart()

	return token{kind: tokenNumber, value: t.input[startPosition:t.position], position: startPosition}, nil
}

// consumeDigits advances the cursor over decimal digits.
func (t *tokeniser) consumeDigits() {
	for t.position < len(t.input) && isDigit(t.input[t.position]) {
		t.position++
	}
}

// consumeFractionalPart advances past an optional '.digits' fractional part.
func (t *tokeniser) consumeFractionalPart() {
	if t.position >= len(t.input) || t.input[t.position] != '.' {
		return
	}
	t.position++
	t.consumeDigits()
}

// consumeExponentPart advances past an optional 'e[+-]?digits' exponent.
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

// readIdentifier reads an unquoted identifier.
//
// The cursor advances by UTF-8 rune width so multi-byte code points (Unicode letters or
// digits inside an identifier body) are not truncated mid-code-point as they would be by
// a byte-at-a-time scan.
//
// Returns token which is the identifier token.
// Returns error which is always nil; declared for caller uniformity.
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

// readDollarToken dispatches to dollar-parameter or dollar-quoted string.
//
// Returns token which is the produced token.
// Returns error when the dollar-quoted literal is malformed.
func (t *tokeniser) readDollarToken() (token, error) {
	startPosition := t.position

	if t.position+1 < len(t.input) && isDigit(t.input[t.position+1]) {
		t.position++
		number := 0
		for t.position < len(t.input) && isDigit(t.input[t.position]) {
			number = number*decimalBase + int(t.input[t.position]-'0')
			if number > maxDollarParameterNumber {
				return token{}, fmt.Errorf(
					"invalid parameter number at position %d: exceeds maximum of %d",
					startPosition, maxDollarParameterNumber,
				)
			}
			t.position++
		}
		return token{kind: tokenDollarParam, value: t.input[startPosition:t.position], position: startPosition}, nil
	}

	return t.readDollarQuotedString()
}

// readDollarQuotedString reads a $tag$ ... $tag$ dollar-quoted string.
//
// Returns token which is the dollar string token.
// Returns error when the literal is malformed or unterminated.
func (t *tokeniser) readDollarQuotedString() (token, error) {
	startPosition := t.position
	t.position++

	var tag string
	leadingTagRune, _ := utf8.DecodeRuneInString(t.input[t.position:])
	switch {
	case t.position < len(t.input) && t.input[t.position] == '$':
		t.position++
		tag = ""
	case t.position < len(t.input) && isIdentStart(leadingTagRune):
		tagStart := t.position
		for t.position < len(t.input) {
			character, width := utf8.DecodeRuneInString(t.input[t.position:])
			if !isIdentPart(character) {
				break
			}
			t.position += width
		}
		if t.position >= len(t.input) || t.input[t.position] != '$' {
			return token{}, fmt.Errorf("invalid dollar-quoted string at position %d", startPosition)
		}
		tag = t.input[tagStart:t.position]
		t.position++
	default:
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

// readColonToken reads :: cast, :name parameter, or a bare ':' operator.
//
// Returns token which is the produced token.
// Returns error which is always nil; declared for caller uniformity.
func (t *tokeniser) readColonToken() (token, error) {
	startPosition := t.position

	if t.position+1 < len(t.input) && t.input[t.position+1] == ':' {
		t.position += 2
		return token{kind: tokenCast, value: "::", position: startPosition}, nil
	}

	parameterRune, _ := utf8.DecodeRuneInString(t.input[t.position+1:])
	if t.position+1 < len(t.input) && isIdentStart(parameterRune) {
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

	t.position++
	return token{kind: tokenOperator, value: ":", position: startPosition}, nil
}

// readOperator reads an arrow, multi-character, or single-character operator.
//
// Returns token which is the operator token.
// Returns error when no operator pattern matches.
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

// readArrowOperator reads -> or ->> JSON access operators when present.
//
// Takes character (byte) which is the candidate leading byte.
// Takes startPosition (int) which is the start offset of the token.
//
// Returns token which is the produced operator token when matched is true.
// Returns bool which is true when an arrow operator was consumed.
func (t *tokeniser) readArrowOperator(character byte, startPosition int) (token, bool) {
	return t.readArrowLikeOperator(character, '-', startPosition, tokenArrow, "->", tokenDoubleArrow, "->>")
}

// readHashArrowOperator reads #> or #>> JSON path operators when present.
//
// Takes character (byte) which is the candidate leading byte.
// Takes startPosition (int) which is the start offset of the token.
//
// Returns token which is the produced operator token when matched is true.
// Returns bool which is true when a hash-arrow operator was consumed.
func (t *tokeniser) readHashArrowOperator(character byte, startPosition int) (token, bool) {
	return t.readArrowLikeOperator(character, '#', startPosition, tokenHashArrow, "#>", tokenHashDoubleArrow, "#>>")
}

// readArrowLikeOperator reads a prefix> or prefix>> operator pair.
//
// Takes character (byte) which is the candidate leading byte.
// Takes prefix (byte) which is the expected leading byte.
// Takes startPosition (int) which is the start offset of the token.
// Takes singleKind (tokenKind) which classifies the prefix> variant.
// Takes singleValue (string) which is the literal text of the prefix> variant.
// Takes doubleKind (tokenKind) which classifies the prefix>> variant.
// Takes doubleValue (string) which is the literal text of the prefix>> variant.
//
// Returns token which is the produced operator token when matched is true.
// Returns bool which is true when an operator was consumed.
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

var (
	// twoCharOps holds the set of two-character SQL operators recognised by the tokeniser.
	twoCharOps = map[string]bool{
		"<=": true, ">=": true, "<>": true, "!=": true, "||": true,
		"<<": true, ">>": true, "&&": true, "@>": true, "<@": true,
		"~*": true, "!~": true,
		"@@": true,
	}
)

// readTwoOrThreeCharOperator reads a two- or three-character operator.
//
// Takes startPosition (int) which is the start offset of the token.
//
// Returns token which is the produced operator token when matched is true.
// Returns bool which is true when an operator was consumed.
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

// readSingleCharOperator reads a single-character SQL operator.
//
// Takes character (byte) which is the candidate operator byte.
// Takes startPosition (int) which is the start offset of the token.
//
// Returns token which is the operator token.
// Returns error when character is not a recognised operator.
func (t *tokeniser) readSingleCharOperator(character byte, startPosition int) (token, error) {
	singleCharOps := "=<>+-/%~&|!#^"
	if strings.ContainsRune(singleCharOps, rune(character)) {
		t.position++
		return token{kind: tokenOperator, value: string(character), position: startPosition}, nil
	}

	return token{}, fmt.Errorf("unexpected character %q at position %d", string(character), startPosition)
}

// isDigit reports whether character is a decimal digit.
//
// Takes character (byte) which is the byte to classify.
//
// Returns bool which is true when character is between '0' and '9'.
func isDigit(character byte) bool {
	return character >= '0' && character <= '9'
}

// isIdentStart reports whether the rune may begin an identifier.
//
// ASCII letters and the underscore take a fast path; runes beyond the ASCII range are
// accepted when unicode.IsLetter reports them as letters, so identifiers named with
// accented or non-Latin letters begin correctly.
//
// Takes character (rune) which is the rune to classify.
//
// Returns bool which is true for letters, underscore, or non-ASCII letters.
func isIdentStart(character rune) bool {
	return engine_shared.IsIdentStart(character)
}

// isIdentPart reports whether the rune may appear within an identifier after the first
// character, namely an identifier-start rune or a digit.
//
// ASCII runes take a fast path; runes beyond the ASCII range are accepted when
// unicode.IsLetter or unicode.IsDigit reports them, so Unicode letters and digits inside
// an identifier body are preserved.
//
// Takes character (rune) which is the rune to classify.
//
// Returns bool which is true for identifier-start runes or decimal digits.
func isIdentPart(character rune) bool {
	return engine_shared.IsIdentPart(character)
}
