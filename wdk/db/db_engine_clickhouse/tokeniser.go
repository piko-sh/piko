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

package db_engine_clickhouse

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
)

// tokenKind classifies a single lexical token.
//
// Each kind maps to a small, well-defined parser state, so the set is kept narrow on
// purpose. ClickHouse's `{name:Type}` placeholder gets its own kind so the parser can
// pluck the name and type tag out without rescanning.
type tokenKind uint8

const (
	// tokenIdentifier is a bare, quoted, or backtick-quoted identifier.
	tokenIdentifier tokenKind = iota

	// tokenNumber is a numeric literal.
	tokenNumber

	// tokenString is a single-quoted string literal.
	tokenString

	// tokenOperator is an arithmetic, bitwise, or comparison operator.
	tokenOperator

	// tokenLeftParen is the `(` punctuation token.
	tokenLeftParen

	// tokenRightParen is the `)` punctuation token.
	tokenRightParen

	// tokenLeftBracket is the `[` punctuation token.
	tokenLeftBracket

	// tokenRightBracket is the `]` punctuation token.
	tokenRightBracket

	// tokenComma is the `,` punctuation token.
	tokenComma

	// tokenSemicolon is the `;` statement separator token.
	tokenSemicolon

	// tokenDot is the `.` qualifier and tuple-field-access token.
	tokenDot

	// tokenStar is the `*` wildcard and multiplication token.
	tokenStar

	// tokenClickHouseParam is the brace-delimited placeholder `{name:Type}`.
	//
	// The token's value is the placeholder body without braces (`name:Type`); the parser
	// splits on the embedded colon.
	tokenClickHouseParam

	// tokenArrow is the lambda arrow `->` (e.g. `arrayMap(x -> x + 1, arr)`).
	//
	// ClickHouse does not use `->` for JSON access (which postgres does); JSONExtract*
	// functions handle that.
	tokenArrow

	// tokenCast is the postfix `::` cast operator.
	//
	// ClickHouse accepts both `CAST(expr AS type)` and `expr::type`, with `::` being
	// idiomatic for short casts.
	tokenCast

	// tokenEOF marks the end of the input stream.
	tokenEOF
)

const (
	// escapeMarkerWidth is the single byte (x / u / U) following the backslash that selects
	// the escape form.
	escapeMarkerWidth = 1

	// hexByteEscapeDigits is the digit count of the \xHH escape form.
	hexByteEscapeDigits = 2

	// hexRuneEscapeDigits is the digit count of the \uHHHH escape form.
	hexRuneEscapeDigits = 4

	// hexLongRuneEscapeDigits is the digit count of the \UHHHHHHHH escape form.
	hexLongRuneEscapeDigits = 8

	// hexBase is the radix used when decoding a hexadecimal ASCII digit.
	hexBase = 16

	// hexAlphaOffset is the value added to an alphabetic hexadecimal digit (a-f / A-F) when
	// decoding it to its 0-15 numeric value.
	hexAlphaOffset = 10
)

// token carries the lexed value plus its original input byte offset for diagnostics.
//
// The position field is a half-open byte index, not a rune index; the directive parser
// maps it to (line, column) via the engine's line/column tracker.
type token struct {
	// value is the lexed text of the token.
	value string

	// position is the half-open byte offset of the token within the input.
	position int

	// kind classifies the token for the parser.
	kind tokenKind
}

// tokeniser advances over the input byte by byte.
//
// The struct is intentionally tiny: a one-pass, allocation-light scan keeps parser
// throughput within target of the existing engines.
type tokeniser struct {
	// input is the SQL text being scanned.
	input string

	// position is the current byte offset of the scan cursor.
	position int
}

var (
	// dialectConfig declares ClickHouse lexical rules for the shared scanners: nested block
	// comments, 0x/0o/0b base-prefixed integer literals, and a strict requirement that a
	// base prefix be followed by at least one digit.
	dialectConfig = engine_shared.DialectConfig{
		Comments: engine_shared.CommentRules{NestedBlockComments: true},
		Numbers: engine_shared.NumberRules{
			HexPrefix:                true,
			OctalPrefix:              true,
			BinaryPrefix:             true,
			RequireDigitsAfterPrefix: true,
		},
	}

	// singleCharTokens maps a byte to its token kind, paired with hasSingleCharToken so a
	// zero tokenKind for a real mapping (the iota base) is distinguishable from the absence
	// of a mapping.
	singleCharTokens [256]tokenKind

	// hasSingleCharToken marks which singleCharTokens entries are valid mappings rather than
	// the zero default.
	hasSingleCharToken [256]bool

	// twoCharOps lists the two-character operators ClickHouse exposes: the comparisons `<=`,
	// `>=`, `<>`, `!=`, the booleans `&&` and `||`, and the bitwise shifts `<<` and `>>`.
	twoCharOps = map[string]bool{
		"<=": true, ">=": true, "<>": true, "!=": true, "||": true,
		"<<": true, ">>": true, "&&": true,
	}
)

// tokenise produces the full token stream for the input.
//
// Takes input (string) which is the SQL text to scan.
//
// Returns []token which is the lexed stream terminated by a tokenEOF sentinel.
// Returns error when a string or placeholder is unterminated or an unexpected character
// does not start any valid token.
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

// next emits the next token from the input stream.
//
// Whitespace and comments are skipped first, then the leading byte dispatches to the
// single-char or multi-char reader. Identifier-start dispatch decodes the leading UTF-8
// codepoint so a multi-byte rune (for example a Unicode letter at the start of a
// quoted-or-bare identifier) routes through readIdentifier rather than falling into
// readOperator's "unexpected character" path.
//
// Returns token which is the next lexed token, or a tokenEOF sentinel when the input is
// exhausted.
// Returns error when a string, placeholder, or comment is malformed or unterminated, in
// which case the token is the zero value.
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

func init() {
	registerSingleCharToken('(', tokenLeftParen)
	registerSingleCharToken(')', tokenRightParen)
	registerSingleCharToken('[', tokenLeftBracket)
	registerSingleCharToken(']', tokenRightBracket)
	registerSingleCharToken(',', tokenComma)
	registerSingleCharToken(';', tokenSemicolon)
	registerSingleCharToken('.', tokenDot)
	registerSingleCharToken('*', tokenStar)
}

// registerSingleCharToken records a byte to tokenKind mapping in the dispatch tables used
// by readSingleCharToken.
//
// It is pulled out so the init body reads as a registration list rather than a paired set
// of index writes.
//
// Takes character (byte) which is the source byte to map.
// Takes kind (tokenKind) which is the token kind the byte produces.
func registerSingleCharToken(character byte, kind tokenKind) {
	singleCharTokens[character] = kind
	hasSingleCharToken[character] = true
}

// readSingleCharToken emits a punctuation token when the leading byte maps to one in the
// dispatch tables.
//
// Takes character (byte) which is the leading byte under the cursor.
//
// Returns token which is the lexed punctuation token when one matches.
// Returns bool which is true when the byte mapped to a single-character token.
func (t *tokeniser) readSingleCharToken(character byte) (token, bool) {
	if !hasSingleCharToken[character] {
		return token{}, false
	}
	startPosition := t.position
	t.position++
	return token{kind: singleCharTokens[character], value: string(character), position: startPosition}, true
}

// readMultiCharToken routes to the appropriate scanner for multi-byte constructs.
//
// Order matters: the colon must be tried before generic operators because `::` and `:`
// need separate handling, and the placeholder `{...}` must be tried before any treatment
// of `{` as invalid. For the identifier-start branch the leading UTF-8 codepoint is
// decoded so a multi-byte rune is correctly recognised; without the decode the function
// only inspects the high byte and the dispatch falls into readOperator on a Unicode
// letter.
//
// Takes character (byte) which is the leading byte under the cursor.
//
// Returns token which is the lexed multi-byte token.
// Returns error when the construct is malformed or unterminated.
func (t *tokeniser) readMultiCharToken(character byte) (token, error) {
	switch {
	case character == '{':
		return t.readClickHouseParam()
	case character == ':':
		return t.readColonToken()
	case character == '\'':
		return t.readString()
	case character == '"':
		return t.readQuotedIdentifier()
	case character == '`':
		return t.readBacktickIdentifier()
	case isDigit(character):
		return t.readNumber()
	}
	leading, _ := utf8.DecodeRuneInString(t.input[t.position:])
	if isIdentStart(leading) {
		return t.readIdentifier()
	}
	return t.readOperator()
}

// skipWhitespaceAndComments advances the cursor past insignificant whitespace and SQL
// comments.
//
// ClickHouse uses standard `--` line comments and `/* */` block comments. Block comments
// are non-nested in upstream ClickHouse, but nesting is accepted because the cost is
// trivial and the only failure mode is being too permissive on input other tools also
// tolerate.
//
// Returns error when a block comment is unterminated.
func (t *tokeniser) skipWhitespaceAndComments() error {
	position, err := dialectConfig.Comments.SkipWhitespaceAndComments(t.input, t.position)
	t.position = position
	return err
}

// readString reads a single-quoted ClickHouse string literal.
//
// It supports both doubled single-quote escapes and backslash escapes (`\\`, `\'`, `\n`,
// `\t`, `\r`, `\0`). The returned value carries the unescaped string body without the
// enclosing quotes.
//
// Returns token which is the lexed string token holding the unescaped body.
// Returns error when the string literal is unterminated.
func (t *tokeniser) readString() (token, error) {
	startPosition := t.position
	t.position++

	var builder strings.Builder
	for t.position < len(t.input) {
		character := t.input[t.position]
		if character == '\\' && t.position+1 < len(t.input) {
			t.readStringEscape(&builder)
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

// readStringEscape consumes the escape sequence at the cursor, writes the decoded bytes
// into builder, and advances the cursor past the whole sequence.
//
// The cursor must point at the leading backslash. Beyond the single-character escapes
// handled by decodeStringEscape it recognises the hex (\xHH) and unicode (\uHHHH,
// \UHHHHHHHH) forms ClickHouse accepts. A truncated or non-hex sequence falls back to the
// single-character behaviour so legacy data still round-trips rather than erroring.
//
// Takes builder (*strings.Builder) which receives the decoded bytes.
func (t *tokeniser) readStringEscape(builder *strings.Builder) {
	t.position++
	if t.position >= len(t.input) {
		builder.WriteByte('\\')
		return
	}
	marker := t.input[t.position]
	switch marker {
	case 'x':
		if value, ok := t.decodeHexEscape(t.position+escapeMarkerWidth, hexByteEscapeDigits); ok {
			builder.WriteByte(byte(value)) //nolint:gosec // two hex digits guarantee 0-255
			t.position += escapeMarkerWidth + hexByteEscapeDigits
			return
		}
	case 'u':
		if t.writeUnicodeEscape(builder, hexRuneEscapeDigits) {
			return
		}
	case 'U':
		if t.writeUnicodeEscape(builder, hexLongRuneEscapeDigits) {
			return
		}
	}
	builder.WriteByte(decodeStringEscape(marker))
	t.position++
}

// writeUnicodeEscape decodes a \u / \U escape of the given digit width and writes the
// resulting rune, advancing the cursor.
//
// It leaves the cursor untouched and returns false when the digits are missing or the
// value is not a valid Unicode code point, so the caller falls back to literal handling.
// The unicode.MaxRune guard also keeps the rune conversion within int32 range.
//
// Takes builder (*strings.Builder) which receives the decoded rune.
// Takes digits (int) which is the number of hex digits in the escape.
//
// Returns bool which is true when the escape was decoded and written.
func (t *tokeniser) writeUnicodeEscape(builder *strings.Builder, digits int) bool {
	value, ok := t.decodeHexEscape(t.position+escapeMarkerWidth, digits)
	if !ok || value > unicode.MaxRune {
		return false
	}
	_, _ = builder.WriteRune(rune(value)) //nolint:gosec // guarded by value > unicode.MaxRune above, so it fits a rune
	t.position += escapeMarkerWidth + digits
	return true
}

// decodeHexEscape reads exactly count hexadecimal digits starting at start and returns
// their integer value.
//
// It falls back to treating the escape literally when fewer than count digits remain or a
// non-hex byte is encountered.
//
// Takes start (int) which is the index of the first hex digit.
// Takes count (int) which is the exact number of digits to read.
//
// Returns value (int) which is the decoded code point.
// Returns ok (bool) which is true when count valid hex digits were read.
func (t *tokeniser) decodeHexEscape(start, count int) (value int, ok bool) {
	if start+count > len(t.input) {
		return 0, false
	}
	for offset := range count {
		digit, valid := hexDigitValue(t.input[start+offset])
		if !valid {
			return 0, false
		}
		value = value*hexBase + digit
	}
	return value, true
}

// hexDigitValue maps a single hexadecimal ASCII byte to its 0-15 value.
//
// Takes character (byte) which is the candidate hexadecimal digit.
//
// Returns digit (int) which is the 0-15 value of the byte.
// Returns valid (bool) which is false for any non-hex byte.
func hexDigitValue(character byte) (digit int, valid bool) {
	switch {
	case character >= '0' && character <= '9':
		return int(character - '0'), true
	case character >= 'a' && character <= 'f':
		return int(character-'a') + hexAlphaOffset, true
	case character >= 'A' && character <= 'F':
		return int(character-'A') + hexAlphaOffset, true
	default:
		return 0, false
	}
}

// decodeStringEscape translates a single backslash-escape character to its expanded byte.
//
// Unrecognised escapes pass through unchanged, matching ClickHouse server tolerance for
// legacy data.
//
// Takes escape (byte) which is the character following the backslash.
//
// Returns byte which is the decoded escape value.
func decodeStringEscape(escape byte) byte {
	switch escape {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'r':
		return '\r'
	case '0':
		return 0
	case '\\':
		return '\\'
	case '\'':
		return '\''
	case '"':
		return '"'
	default:
		return escape
	}
}

// readQuotedIdentifier reads a double-quoted identifier.
//
// ClickHouse accepts both double-quoted and backtick-quoted identifiers; this helper
// handles the double-quote form. A doubled-quote ("") escapes to a single double-quote in
// the value.
//
// Returns token which is the lexed identifier token.
// Returns error when the quoted identifier is unterminated.
func (t *tokeniser) readQuotedIdentifier() (token, error) {
	return t.readDelimitedIdentifier('"')
}

// readBacktickIdentifier reads a backtick-quoted identifier.
//
// A doubled backtick escapes to a single backtick in the value.
//
// Returns token which is the lexed identifier token.
// Returns error when the quoted identifier is unterminated.
func (t *tokeniser) readBacktickIdentifier() (token, error) {
	return t.readDelimitedIdentifier('`')
}

// readDelimitedIdentifier reads an identifier enclosed by the given delimiter where a
// doubled delimiter escapes to a single one in the value.
//
// Takes delimiter (byte) which is the opening and closing quote character.
//
// Returns token which is the lexed identifier token.
// Returns error when the quoted identifier is unterminated.
func (t *tokeniser) readDelimitedIdentifier(delimiter byte) (token, error) {
	startPosition := t.position

	value, position, ok := engine_shared.ScanDoubledDelimiter(t.input, t.position, delimiter)
	if !ok {
		return token{}, fmt.Errorf("unterminated quoted identifier at position %d", startPosition)
	}
	t.position = position

	return token{kind: tokenIdentifier, value: value, position: startPosition}, nil
}

// readClickHouseParam reads a placeholder of the form `{name:Type}`.
//
// The returned token's value is the placeholder body without braces, so callers can split
// on the embedded colon to recover the name and the ClickHouse type tag. The body must
// match `name:Type` where name is a valid identifier and Type is non-empty. Newlines and
// embedded `{` inside the body are rejected because they would surface as confusing
// parser errors downstream.
//
// Returns token which is the lexed placeholder token carrying the body without braces.
// Returns error when the closing brace is missing, the colon separating name from type is
// missing, the name or type is empty, or a newline or `{` appears inside the body.
func (t *tokeniser) readClickHouseParam() (token, error) {
	startPosition := t.position
	t.position++

	var builder strings.Builder
	for t.position < len(t.input) {
		character := t.input[t.position]
		if character == '}' {
			body := builder.String()
			t.position++
			if err := validateClickHouseParamBody(body, startPosition); err != nil {
				return token{}, err
			}
			return token{kind: tokenClickHouseParam, value: body, position: startPosition}, nil
		}
		if character == '\n' {
			return token{}, fmt.Errorf("unexpected newline inside placeholder at position %d", startPosition)
		}
		if character == '{' {
			return token{}, fmt.Errorf("unexpected '{' inside placeholder at position %d", startPosition)
		}
		builder.WriteByte(character)
		t.position++
	}

	return token{}, fmt.Errorf("unterminated placeholder at position %d", startPosition)
}

// validateClickHouseParamBody enforces the `name:Type` shape on a stripped parameter
// body.
//
// The colon must be present and divide the body into a non-empty identifier (name) and
// non-empty type tag.
//
// Takes body (string) which is the placeholder body without braces.
// Takes position (int) which is the byte offset used in error messages.
//
// Returns error when the body lacks a colon, has an empty name or type, or has a name
// that does not start with an identifier character.
func validateClickHouseParamBody(body string, position int) error {
	nameSegment, typeSegment, found := strings.Cut(body, ":")
	if !found {
		return fmt.Errorf("placeholder at position %d is missing ':' separator", position)
	}
	name := strings.TrimSpace(nameSegment)
	typeTag := strings.TrimSpace(typeSegment)
	if name == "" {
		return fmt.Errorf("placeholder at position %d has empty parameter name", position)
	}
	if typeTag == "" {
		return fmt.Errorf("placeholder at position %d has empty type tag", position)
	}
	leading, _ := utf8.DecodeRuneInString(name)
	if !isIdentStart(leading) {
		return fmt.Errorf("placeholder at position %d: parameter name %q does not start with an identifier character", position, name)
	}
	return nil
}

// readNumber reads a numeric literal, accepting base-prefixed integers (0x / 0o / 0b) as
// well as decimal integers with optional fractional and exponent parts.
//
// Returns token which is the lexed numeric token.
// Returns error when a base prefix is not followed by at least one digit.
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

// consumeDigits advances the cursor past any run of ASCII decimal digits. Used inside the
// numeric scanner to read the integer and fractional digit groups.
func (t *tokeniser) consumeDigits() {
	for t.position < len(t.input) && isDigit(t.input[t.position]) {
		t.position++
	}
}

// consumeFractionalPart consumes the `.<digits>` portion of a numeric literal if present.
// No-op when the current byte is not a decimal point.
func (t *tokeniser) consumeFractionalPart() {
	if t.position >= len(t.input) || t.input[t.position] != '.' {
		return
	}
	t.position++
	t.consumeDigits()
}

// consumeExponentPart consumes the `[eE][+-]?<digits>` portion of a numeric literal if
// present. Accepts both signed and unsigned exponents; no-op when the current byte is not
// `e` or `E`.
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

// readIdentifier consumes an identifier from the current cursor.
//
// It advances by UTF-8 rune width so multi-byte codepoints (Unicode letters or digits
// inside an identifier body) are not truncated mid-codepoint as they would be by a
// byte-at-a-time scan.
//
// Returns token which is the lexed identifier token.
// Returns error which is always nil and present for interface symmetry with the other
// readers.
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

// readColonToken handles `::` (cast) and a single `:` (operator).
//
// ClickHouse does not use `:name` for named parameters (those are `{name:Type}` instead),
// so this scanner is simpler than postgres'.
//
// Returns token which is the cast or operator token.
// Returns error which is always nil and present for interface symmetry with the other
// readers.
func (t *tokeniser) readColonToken() (token, error) {
	startPosition := t.position

	if t.position+1 < len(t.input) && t.input[t.position+1] == ':' {
		t.position += 2
		return token{kind: tokenCast, value: "::", position: startPosition}, nil
	}

	t.position++
	return token{kind: tokenOperator, value: ":", position: startPosition}, nil
}

// readOperator scans the next operator token.
//
// ClickHouse's operator surface is a subset of postgres' (no JSON `->>`, `#>`, `#>>`
// since JSON access uses functions). The lambda arrow `->` becomes its own token kind so
// the parser can dispatch directly.
//
// Returns token which is the lexed operator token.
// Returns error when the byte does not start any valid operator.
func (t *tokeniser) readOperator() (token, error) {
	startPosition := t.position
	character := t.input[t.position]

	if tok, ok := t.readArrowOperator(character, startPosition); ok {
		return tok, nil
	}

	if tok, ok := t.readTwoCharOperator(startPosition); ok {
		return tok, nil
	}

	return t.readSingleCharOperator(character, startPosition)
}

// readArrowOperator handles the lambda arrow `->`.
//
// Takes character (byte) which is the leading byte under the cursor.
// Takes startPosition (int) which is the byte offset recorded on the token.
//
// Returns token which is the arrow token when one matches.
// Returns bool which is false when the cursor is not on a `-` or no `>` follows.
func (t *tokeniser) readArrowOperator(character byte, startPosition int) (token, bool) {
	if character != '-' || t.position+1 >= len(t.input) || t.input[t.position+1] != '>' {
		return token{}, false
	}
	t.position += 2
	return token{kind: tokenArrow, value: "->", position: startPosition}, true
}

// readTwoCharOperator matches a two-character operator from the twoCharOps table.
//
// Takes startPosition (int) which is the byte offset recorded on the token.
//
// Returns token which is the operator token when one matches.
// Returns bool which is false when the next two bytes do not form a known two-character
// operator.
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

// readSingleCharOperator scans a single-character operator.
//
// The allowed set covers the SQL arithmetic, bitwise, and comparison operators ClickHouse
// exposes. JDBC-style `?` placeholders are deliberately excluded because ClickHouse SQL
// uses `{name:Type}` for parameter binding; a stray `?` produces an explicit "unexpected
// character" diagnostic rather than silently tokenising as an operator.
//
// Takes character (byte) which is the candidate operator byte.
// Takes startPosition (int) which is the byte offset recorded on the token.
//
// Returns token which is the lexed operator token.
// Returns error when the byte is not a recognised single-character operator.
func (t *tokeniser) readSingleCharOperator(character byte, startPosition int) (token, error) {
	if strings.IndexByte("=<>+-/%~&|!^", character) >= 0 {
		t.position++
		return token{kind: tokenOperator, value: string(character), position: startPosition}, nil
	}

	return token{}, fmt.Errorf("unexpected character %q at position %d", string(character), startPosition)
}

// isDigit reports whether the byte is an ASCII decimal digit (0-9).
//
// Takes character (byte) which is the candidate digit.
//
// Returns bool which is true when the byte is in the range 0-9.
func isDigit(character byte) bool {
	return character >= '0' && character <= '9'
}

// isIdentStart reports whether the rune can begin a SQL identifier.
//
// An ASCII letter, an underscore, or a non-ASCII letter (per unicode.IsLetter) qualifies,
// matching ClickHouse's permissive identifier rules where Unicode-named columns are
// accepted.
//
// Takes character (rune) which is the candidate leading rune.
//
// Returns bool which is true when the rune may start an identifier.
func isIdentStart(character rune) bool {
	return engine_shared.IsIdentStart(character)
}

// isIdentPart reports whether the rune can appear in a SQL identifier after the first
// character (identifier-start plus digits).
//
// Takes character (rune) which is the candidate body rune.
//
// Returns bool which is true when the rune may appear after the first character.
func isIdentPart(character rune) bool {
	return engine_shared.IsIdentPart(character)
}
