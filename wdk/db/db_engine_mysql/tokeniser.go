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
	"unicode/utf8"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
)

// tokenKind classifies a token produced by the MySQL tokeniser.
type tokenKind uint8

const (
	// tokenIdentifier marks a bare or quoted identifier.
	tokenIdentifier tokenKind = iota

	// tokenNumber marks an integer, decimal, or scientific literal.
	tokenNumber

	// tokenString marks a single-quoted string literal.
	tokenString

	// tokenHexString marks an X'...' hexadecimal literal.
	tokenHexString

	// tokenBitString marks a B'...' bit-string literal.
	tokenBitString

	// tokenOperator marks an arithmetic, comparison, or logical operator.
	tokenOperator

	// tokenLeftParen marks a '(' delimiter.
	tokenLeftParen

	// tokenRightParen marks a ')' delimiter.
	tokenRightParen

	// tokenLeftBracket marks a '[' delimiter.
	tokenLeftBracket

	// tokenRightBracket marks a ']' delimiter.
	tokenRightBracket

	// tokenComma marks a ',' separator.
	tokenComma

	// tokenSemicolon marks a ';' statement terminator.
	tokenSemicolon

	// tokenDot marks a '.' qualifier between identifiers.
	tokenDot

	// tokenStar marks a '*' wildcard or multiplication operator.
	tokenStar

	// tokenQuestionMark marks a positional '?' parameter placeholder.
	tokenQuestionMark

	// tokenNamedParam marks a ':name' named parameter placeholder.
	tokenNamedParam

	// tokenArrow marks the '->' JSON-extract operator.
	tokenArrow

	// tokenDoubleArrow marks the '->>' JSON-extract-and-unquote operator.
	tokenDoubleArrow

	// tokenUserVariable marks an '@name' user variable reference.
	tokenUserVariable

	// tokenSystemVariable marks an '@@name' system variable reference.
	tokenSystemVariable

	// tokenEOF marks the end of the input stream.
	tokenEOF
)

const (
	// substituteCharacter is the MySQL backslash-Z escape value (0x1A).
	substituteCharacter = 0x1A

	// backspaceCharacter is the MySQL backslash-b escape value (0x08).
	backspaceCharacter = 0x08
)

// token is a lexical unit produced by the MySQL tokeniser.
type token struct {
	// value is the textual content of the token.
	value string

	// position is the zero-based byte offset of the token in the source.
	position int

	// kind classifies the token.
	kind tokenKind
}

// tokeniser walks the SQL source and emits tokens for the parser.
type tokeniser struct {
	// input holds the SQL source being scanned.
	input string

	// position is the current zero-based byte offset within input.
	position int
}

var (
	// dialectConfig declares MySQL lexical rules for the shared scanners: "--" line comments
	// that require trailing whitespace, "#" line comments, non-nesting block comments, and
	// 0x/0b base-prefixed integer literals (MySQL has no 0o octal form).
	dialectConfig = engine_shared.DialectConfig{
		Comments: engine_shared.CommentRules{
			DoubleDashRequiresWhitespace: true,
			HashLineComment:              true,
		},
		Numbers: engine_shared.NumberRules{HexPrefix: true, BinaryPrefix: true},
	}
)

// tokenise scans an SQL source string into tokens.
//
// Takes input (string) which is the SQL source to scan.
//
// Returns []token which holds the full token stream including the trailing EOF marker.
// Returns error when scanning fails on a malformed literal.
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

// next advances past whitespace and returns the next token.
//
// Returns token which holds the next lexical unit, or an EOF token at end of input.
// Returns error when a literal cannot be scanned.
func (t *tokeniser) next() (token, error) {
	if err := t.skipWhitespaceAndComments(); err != nil {
		return token{}, err
	}

	if t.position >= len(t.input) {
		return token{kind: tokenEOF, position: t.position}, nil
	}

	character := t.input[t.position]

	if character == '?' {
		return t.readQuestionMarkToken(), nil
	}

	if tok, ok := t.readSingleCharToken(character); ok {
		return tok, nil
	}

	return t.readMultiCharToken(character)
}

// readQuestionMarkToken reads a '?' placeholder, consuming any trailing digits so the
// resulting token covers the full '?N' form when present.
//
// Engines using the wire protocol's anonymous '?' still parse correctly because the lone
// '?' branch leaves value="?", while directive-annotated queries written as '?1', '?2'
// produce a single token with value "?N" that the parser deduplicates by number like a
// named parameter.
//
// Returns token which holds the question-mark token with any captured numeric suffix.
func (t *tokeniser) readQuestionMarkToken() token {
	startPosition := t.position
	t.position++
	for t.position < len(t.input) && t.input[t.position] >= '0' && t.input[t.position] <= '9' {
		t.position++
	}
	return token{kind: tokenQuestionMark, value: t.input[startPosition:t.position], position: startPosition}
}

var (

	// singleCharTokens maps a leading byte to its token kind plus one, where zero indicates
	// "not a single-character token". The +1 offset preserves the zero-value meaning for
	// unmapped bytes.
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

// readSingleCharToken emits a token for a known single-character byte.
//
// Takes character (byte) which is the byte at the current scanner position.
//
// Returns token which is the emitted token when the byte matches.
// Returns bool which is true when the byte produced a token.
func (t *tokeniser) readSingleCharToken(character byte) (token, bool) {
	mapped := singleCharTokens[character]
	if mapped == 0 {
		return token{}, false
	}
	startPosition := t.position
	t.position++
	return token{kind: mapped - 1, value: string(character), position: startPosition}, true
}

// readMultiCharToken dispatches to the reader for the current byte.
//
// Takes character (byte) which is the byte at the current scanner position.
//
// Returns token which holds the scanned multi-character token.
// Returns error when the underlying reader fails.
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
	}
	leading, _ := utf8.DecodeRuneInString(t.input[t.position:])
	if isIdentStart(leading) {
		return t.readIdentifier()
	}
	return t.readOperator()
}

// skipWhitespaceAndComments advances past whitespace and SQL comments.
//
// Returns error when a block comment is left unterminated.
func (t *tokeniser) skipWhitespaceAndComments() error {
	position, err := dialectConfig.Comments.SkipWhitespaceAndComments(t.input, t.position)
	t.position = position
	return err
}

// readString scans a single-quoted string literal with escape handling.
//
// Returns token which holds the decoded string contents.
// Returns error when the literal is not terminated before end of input.
func (t *tokeniser) readString() (token, error) {
	startPosition := t.position
	t.position++

	var builder strings.Builder
	for t.position < len(t.input) {
		character := t.input[t.position]
		if character == '\\' && t.position+1 < len(t.input) {
			t.position++
			escaped := t.input[t.position]

			if escaped == '%' || escaped == '_' {
				builder.WriteByte('\\')
				builder.WriteByte(escaped)
			} else {
				builder.WriteByte(mysqlBackslashEscape(escaped))
			}
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

// mysqlBackslashEscape resolves a single MySQL backslash escape character.
//
// Takes character (byte) which is the byte that follows the backslash.
//
// Returns byte which is the escaped value, or the original byte when no special meaning
// applies.
func mysqlBackslashEscape(character byte) byte {
	switch character {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'r':
		return '\r'
	case 'b':
		return backspaceCharacter
	case '0':
		return 0
	case 'Z':
		return substituteCharacter
	default:
		return character
	}
}

// readHexString scans an X'...' hexadecimal literal.
//
// Returns token which holds the literal payload between the quotes.
// Returns error when the literal is not terminated before end of input.
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

// readBitString scans a B'...' bit-string literal.
//
// Returns token which holds the literal payload between the quotes.
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

// readBacktickIdentifier scans a backtick-delimited identifier.
//
// Returns token which holds the decoded identifier.
// Returns error when the identifier is not terminated before end of input.
func (t *tokeniser) readBacktickIdentifier() (token, error) {
	return t.readDelimitedIdentifier('`')
}

// readQuotedIdentifier scans a double-quoted identifier under ANSI_QUOTES.
//
// Returns token which holds the decoded identifier.
// Returns error when the identifier is not terminated before end of input.
func (t *tokeniser) readQuotedIdentifier() (token, error) {
	return t.readDelimitedIdentifier('"')
}

// readDelimitedIdentifier scans an identifier bounded by a given delimiter.
//
// Takes delimiter (byte) which is the byte that opens and closes the identifier and which
// is doubled to escape itself.
//
// Returns token which holds the decoded identifier.
// Returns error when the identifier is not terminated before end of input.
func (t *tokeniser) readDelimitedIdentifier(delimiter byte) (token, error) {
	startPosition := t.position

	value, position, ok := engine_shared.ScanDoubledDelimiter(t.input, t.position, delimiter)
	if !ok {
		return token{}, fmt.Errorf("unterminated quoted identifier at position %d", startPosition)
	}
	t.position = position

	return token{kind: tokenIdentifier, value: value, position: startPosition}, nil
}

// readNumber scans a numeric literal in integer, decimal, or scientific form.
//
// Returns token which holds the raw numeric text.
// Returns error when the literal cannot be scanned.
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

// consumeDigits advances while the current byte is a decimal digit.
func (t *tokeniser) consumeDigits() {
	for t.position < len(t.input) && isDigit(t.input[t.position]) {
		t.position++
	}
}

// consumeFractionalPart consumes the '.digits' portion of a numeric literal.
func (t *tokeniser) consumeFractionalPart() {
	if t.position >= len(t.input) || t.input[t.position] != '.' {
		return
	}
	t.position++
	t.consumeDigits()
}

// consumeExponentPart consumes an 'e[+-]?digits' exponent of a number.
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

// readIdentifier scans a bare identifier or unquoted keyword.
//
// Advances by UTF-8 rune width so multi-byte code points (Unicode letters or digits
// inside an identifier body, for example accented or CJK characters) are not truncated
// mid-code-point as they would be by a byte-at-a-time scan.
//
// Returns token which holds the identifier text.
// Returns error which is always nil; the signature aligns with peer readers.
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

// readAtToken scans an @user or @@system variable reference.
//
// Returns token which holds the variable token including the @ prefix.
// Returns error which is always nil; the signature aligns with peer readers.
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

// consumeSystemVariableQualifier consumes a global/session/local qualifier.
func (t *tokeniser) consumeSystemVariableQualifier() {
	if t.position >= len(t.input) {
		return
	}
	leading, _ := utf8.DecodeRuneInString(t.input[t.position:])
	if !isIdentStart(leading) {
		return
	}

	saved := t.position
	t.consumeVariableName()
	if t.position < len(t.input) && t.input[t.position] == '.' {
		qualifier := t.input[saved:t.position]
		if qualifier == "global" || qualifier == "session" || qualifier == "local" {
			t.position++
			return
		}
	}
	t.position = saved
}

// consumeVariableName advances over an unquoted variable name.
//
// Advances by UTF-8 rune width so multi-byte code points within the name are not
// truncated mid-code-point.
func (t *tokeniser) consumeVariableName() {
	for t.position < len(t.input) {
		character, width := utf8.DecodeRuneInString(t.input[t.position:])
		if !isIdentPart(character) {
			break
		}
		t.position += width
	}
}

// readColonToken scans a named-parameter placeholder or a colon operator.
//
// Returns token which holds the placeholder or operator token.
// Returns error which is always nil; the signature aligns with peer readers.
func (t *tokeniser) readColonToken() (token, error) {
	startPosition := t.position

	if t.position+1 < len(t.input) {
		leading, _ := utf8.DecodeRuneInString(t.input[t.position+1:])
		if isIdentStart(leading) {
			t.position++
			t.consumeVariableName()
			return token{kind: tokenNamedParam, value: t.input[startPosition:t.position], position: startPosition}, nil
		}
	}

	t.position++
	return token{kind: tokenOperator, value: ":", position: startPosition}, nil
}

// readOperator scans an operator token, preferring longer matches.
//
// Returns token which holds the matched operator.
// Returns error when the current byte does not start a known operator.
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

// readArrowOperator scans -> and ->> JSON-extract operators.
//
// Takes character (byte) which is the byte at the current position.
// Takes startPosition (int) which is the byte offset where the token began.
//
// Returns token which holds the arrow operator when matched.
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

// readThreeCharOperator scans the <=> null-safe comparison operator.
//
// Takes startPosition (int) which is the byte offset where the token began.
//
// Returns token which holds the matched operator.
// Returns bool which is true when the three-character operator matches.
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

var (
	// twoCharOps lists the two-character operators recognised by the tokeniser.
	twoCharOps = map[string]bool{
		"<=": true, ">=": true, "<>": true, "!=": true, "||": true,
		"<<": true, ">>": true,
	}
)

// readTwoCharOperator scans a two-character operator listed in twoCharOps.
//
// Takes startPosition (int) which is the byte offset where the token began.
//
// Returns token which holds the matched operator.
// Returns bool which is true when a two-character operator was consumed.
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

// readSingleCharOperator scans a single-byte operator.
//
// Takes character (byte) which is the byte at the current position.
// Takes startPosition (int) which is the byte offset where the token began.
//
// Returns token which holds the operator when matched.
// Returns error when the byte is not a known operator character.
func (t *tokeniser) readSingleCharOperator(character byte, startPosition int) (token, error) {
	singleCharOps := "=<>+-/%~&|!^"
	if strings.ContainsRune(singleCharOps, rune(character)) {
		t.position++
		return token{kind: tokenOperator, value: string(character), position: startPosition}, nil
	}

	return token{}, fmt.Errorf("unexpected character %q at position %d", string(character), startPosition)
}

// isDigit reports whether a byte is an ASCII decimal digit.
//
// Takes character (byte) which is the byte to test.
//
// Returns bool which is true when the byte is in '0'..'9'.
func isDigit(character byte) bool {
	return character >= '0' && character <= '9'
}

// isIdentStart reports whether a rune may start an identifier.
//
// An ASCII fast-path admits ASCII letters and underscore directly; any rune above the
// ASCII range is admitted when unicode.IsLetter recognises it, so Unicode-named columns
// such as accented or CJK identifiers tokenise as single identifiers.
//
// Takes character (rune) which is the rune to test.
//
// Returns bool which is true for ASCII letters, underscore, or any non-ASCII letter
// recognised by unicode.IsLetter.
func isIdentStart(character rune) bool {
	return engine_shared.IsIdentStart(character)
}

// isIdentPart reports whether a rune may continue an identifier.
//
// This is the identifier-start set plus decimal digits. The ASCII fast-path guards the
// rune into the ASCII range before the digit check so the comparison never loses data;
// non-ASCII runes are admitted when unicode.IsLetter or unicode.IsDigit recognises them.
//
// Takes character (rune) which is the rune to test.
//
// Returns bool which is true for identifier-start runes or decimal digits.
func isIdentPart(character rune) bool {
	return engine_shared.IsIdentPart(character)
}
