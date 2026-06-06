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

package querier_domain

import (
	"fmt"
	"slices"
	"unicode/utf8"

	"piko.sh/piko/internal/querier/querier_dto"
)

// tokenKind enumerates the lexer's output tokens.
type tokenKind uint8

const (
	// tokenEOF marks the end of the input.
	tokenEOF tokenKind = iota

	// tokenIdent is an identifier token.
	tokenIdent

	// tokenNumber is a numeric literal token.
	tokenNumber

	// tokenString is a quoted string literal token.
	tokenString

	// tokenLParen is the opening parenthesis token.
	tokenLParen

	// tokenRParen is the closing parenthesis token.
	tokenRParen

	// tokenLBracket is the opening square bracket token.
	tokenLBracket

	// tokenRBracket is the closing square bracket token.
	tokenRBracket

	// tokenColon is the colon token used as a keyword argument separator.
	tokenColon

	// tokenComma is the comma token separating arguments.
	tokenComma

	// tokenDot is the dot token used in dotted paths.
	tokenDot

	// tokenSigil is a parameter anchor token such as $1, :name, @name or ?2.
	tokenSigil

	// tokenInvalid marks a byte or rune that begins no recognised token.
	tokenInvalid
)

// token is a single lexed unit carrying its lexeme text and source span.
type token struct {
	// lexeme is the raw text of the token.
	lexeme string

	// span is the source span covering the token in file coordinates.
	span querier_dto.TextSpan

	// kind is the classification of the token.
	kind tokenKind
}

// firstByte returns the first byte of the token's lexeme or zero when empty.
//
// The parser uses it to detect parameter-prefix sigils without inspecting kinds.
//
// Returns byte which is the first byte of the lexeme, or zero when the lexeme is empty.
func (t token) firstByte() byte {
	if len(t.lexeme) == 0 {
		return 0
	}
	return t.lexeme[0]
}

// directiveLexer turns the joined content of a logicalLine into a stream of tokens.
// Positions track back to the original physical (line, column) coordinates via the
// logicalLine's lineSegment source map.
//
// Sigil scanning (`$1`, `:name`, `@name`, `?2`) is enabled only for the very first
// non-whitespace token, because every subsequent occurrence of `:` is a keyword argument
// separator and every `?`/`@` is invalid outside an anchor position.
type directiveLexer struct {
	// peeked buffers tokens that have been scanned ahead but not yet consumed.
	peeked []token

	// logical is the joined logical line being tokenised.
	logical logicalLine

	// pos is the current byte offset into the logical content.
	pos int

	// scannedAny records whether any non-whitespace token has been scanned yet.
	scannedAny bool
}

// newDirectiveLexer constructs a lexer for the given logical line.
//
// Takes logical (logicalLine) which is the joined logical line to tokenise.
//
// Returns *directiveLexer which is the lexer ready to scan the line.
func newDirectiveLexer(logical logicalLine) *directiveLexer {
	return &directiveLexer{logical: logical}
}

// next returns the next token, consuming it.
//
// Returns token which is the next token in the stream.
func (l *directiveLexer) next() token {
	if len(l.peeked) > 0 {
		tok := l.peeked[0]
		l.peeked = l.peeked[1:]
		return tok
	}
	return l.scan()
}

// peek returns the next token without consuming it.
//
// Returns token which is the next token in the stream.
func (l *directiveLexer) peek() token {
	if len(l.peeked) == 0 {
		l.peeked = append(l.peeked, l.scan())
	}
	return l.peeked[0]
}

// peekN returns the (offset+1)-th upcoming token without consuming any tokens.
//
// Argument parsing uses it to disambiguate `ident :` (a keyword argument) from a bare
// value.
//
// Takes offset (int) which is the zero-based lookahead distance into the token stream.
//
// Returns token which is the token at the requested lookahead position.
func (l *directiveLexer) peekN(offset int) token {
	for len(l.peeked) <= offset {
		l.peeked = append(l.peeked, l.scan())
	}
	return l.peeked[offset]
}

// scan reads one token from the input position.
//
// Returns token which is the token read at the current position.
func (l *directiveLexer) scan() token {
	l.skipWhitespace()
	if l.pos >= len(l.logical.content) {
		return token{kind: tokenEOF, span: l.spanAt(l.pos, l.pos)}
	}
	start := l.pos
	ch := l.logical.content[l.pos]
	switch {
	case ch == '(':
		l.pos++
		return token{kind: tokenLParen, lexeme: "(", span: l.spanAt(start, l.pos)}
	case ch == ')':
		l.pos++
		return token{kind: tokenRParen, lexeme: ")", span: l.spanAt(start, l.pos)}
	case ch == '[':
		l.pos++
		return token{kind: tokenLBracket, lexeme: "[", span: l.spanAt(start, l.pos)}
	case ch == ']':
		l.pos++
		return token{kind: tokenRBracket, lexeme: "]", span: l.spanAt(start, l.pos)}
	case ch == ',':
		l.pos++
		return token{kind: tokenComma, lexeme: ",", span: l.spanAt(start, l.pos)}
	case !l.scannedAny && isParameterSigil(ch):
		l.scannedAny = true
		return l.scanSigil(start)
	case ch == ':':
		l.pos++
		return token{kind: tokenColon, lexeme: ":", span: l.spanAt(start, l.pos)}
	case ch == '.':
		l.pos++
		return token{kind: tokenDot, lexeme: ".", span: l.spanAt(start, l.pos)}
	case ch == '\'' || ch == '"':
		l.scannedAny = true
		return l.scanString(start)
	case isDigit(ch):
		l.scannedAny = true
		return l.scanNumber(start)
	case ch == '-' && l.pos+1 < len(l.logical.content) && isDigit(l.logical.content[l.pos+1]):
		l.scannedAny = true
		l.pos++
		return l.scanNumber(start)
	case isIdentStart(ch):
		l.scannedAny = true
		return l.scanIdent(start)
	default:
		return l.scanInvalid(start, ch)
	}
}

// scanInvalid emits a tokenInvalid for a byte that begins no recognised token.
//
// It decodes a full rune so a multibyte UTF-8 codepoint is reported intact rather than
// re-encoded byte-by-byte into mojibake, and a genuinely invalid byte is shown as hex.
//
// Takes start (int) which is the byte offset where the token began.
// Takes ch (byte) which is the leading byte (used for the hex fallback).
//
// Returns token which is the tokenInvalid carrying the decoded rune or hex byte.
func (l *directiveLexer) scanInvalid(start int, ch byte) token {
	decoded, width := utf8.DecodeRuneInString(l.logical.content[l.pos:])
	lexeme := string(decoded)
	if decoded == utf8.RuneError && width <= 1 {
		lexeme = fmt.Sprintf("0x%02x", ch)
	}
	l.pos += width
	l.scannedAny = true
	return token{kind: tokenInvalid, lexeme: lexeme, span: l.spanAt(start, l.pos)}
}

// scanString reads a quoted string literal.
//
// The opening quote may be either a single or a double quote, and doubled-quote escapes
// are honoured.
//
// Takes start (int) which is the byte offset where the token began.
//
// Returns token which is the string token, or a tokenInvalid when the literal is
// unterminated.
func (l *directiveLexer) scanString(start int) token {
	quote := l.logical.content[l.pos]
	l.pos++
	var sb []byte
	for l.pos < len(l.logical.content) {
		ch := l.logical.content[l.pos]
		if ch == '\\' && l.pos+1 < len(l.logical.content) {
			sb = append(sb, l.logical.content[l.pos+1])
			l.pos += 2
			continue
		}
		if ch == quote {
			if l.pos+1 < len(l.logical.content) && l.logical.content[l.pos+1] == quote {
				sb = append(sb, quote)
				l.pos += 2
				continue
			}
			l.pos++
			return token{kind: tokenString, lexeme: string(sb), span: l.spanAt(start, l.pos)}
		}
		sb = append(sb, ch)
		l.pos++
	}
	return token{kind: tokenInvalid, lexeme: string(sb), span: l.spanAt(start, l.pos)}
}

// scanSigil reads a parameter anchor in the `$N`, `:name`, `@name`, `?N` or `{name:Type}`
// form.
//
// For the brace form the scanner consumes through the matching `}`; for the others it
// consumes the sigil byte plus immediately-following ident or digit characters. The
// returned token keeps the sigil prefix on its lexeme so the parser can recognise which
// prefix style was used. An unterminated `{name:Type` form produces a tokenInvalid so the
// upstream parser surfaces it instead of the lexer sliding past end-of-input.
//
// Takes start (int) which is the byte offset where the token began.
//
// Returns token which is the sigil token, or a tokenInvalid for an unterminated brace
// form.
func (l *directiveLexer) scanSigil(start int) token {
	opener := l.logical.content[l.pos]
	l.pos++
	if opener == '{' {
		for l.pos < len(l.logical.content) && l.logical.content[l.pos] != '}' {
			l.pos++
		}
		if l.pos >= len(l.logical.content) {
			return token{kind: tokenInvalid, lexeme: l.logical.content[start:l.pos], span: l.spanAt(start, l.pos)}
		}
		l.pos++
		return token{kind: tokenSigil, lexeme: l.logical.content[start:l.pos], span: l.spanAt(start, l.pos)}
	}
	for l.pos < len(l.logical.content) && isIdentPart(l.logical.content[l.pos]) {
		l.pos++
	}
	return token{kind: tokenSigil, lexeme: l.logical.content[start:l.pos], span: l.spanAt(start, l.pos)}
}

// scanNumber reads a run of ASCII digits into a numeric token.
//
// Takes start (int) which is the byte offset where the token began.
//
// Returns token which is the number token covering the consumed digits.
func (l *directiveLexer) scanNumber(start int) token {
	for l.pos < len(l.logical.content) && isDigit(l.logical.content[l.pos]) {
		l.pos++
	}
	return token{kind: tokenNumber, lexeme: l.logical.content[start:l.pos], span: l.spanAt(start, l.pos)}
}

// scanIdent reads a run of identifier characters into an identifier token.
//
// Takes start (int) which is the byte offset where the token began.
//
// Returns token which is the identifier token covering the consumed characters.
func (l *directiveLexer) scanIdent(start int) token {
	for l.pos < len(l.logical.content) && isIdentPart(l.logical.content[l.pos]) {
		l.pos++
	}
	return token{kind: tokenIdent, lexeme: l.logical.content[start:l.pos], span: l.spanAt(start, l.pos)}
}

// skipWhitespace advances the current position past spaces and tabs.
func (l *directiveLexer) skipWhitespace() {
	for l.pos < len(l.logical.content) {
		ch := l.logical.content[l.pos]
		if ch == ' ' || ch == '\t' {
			l.pos++
			continue
		}
		break
	}
}

// spanAt maps a half-open content range to file coordinates via the logical line's
// segment map.
//
// The reported column is a byte column, not a rune column, because the directive grammar
// only recognises ASCII identifiers, sigils and punctuation. Editors that expect rune
// columns for non-ASCII content can be a few columns off, but no diagnostic from this
// lexer ever points at non-ASCII content because the lexer rejects it as tokenInvalid
// first.
//
// Takes start (int) which is the inclusive byte offset where the range begins.
// Takes end (int) which is the exclusive byte offset where the range ends.
//
// Returns querier_dto.TextSpan which is the file-coordinate span for the range.
func (l *directiveLexer) spanAt(start, end int) querier_dto.TextSpan {
	startLine, startCol := l.locate(start)
	endLine, endCol := l.locate(end)
	return querier_dto.TextSpan{
		Line:      startLine,
		Column:    startCol,
		EndLine:   endLine,
		EndColumn: endCol,
	}
}

// locate finds the physical line and column for a position in the logical content.
//
// It falls back to the first segment when the position precedes any segment, which is a
// defensive case that does not arise on valid scanner input.
//
// Takes offset (int) which is the byte offset into the logical content.
//
// Returns physicalLine (int) which is the one-based physical line number.
// Returns column (int) which is the one-based byte column on that line.
func (l *directiveLexer) locate(offset int) (physicalLine, column int) {
	if len(l.logical.segments) == 0 {
		return l.logical.startLine, 1
	}
	for _, seg := range slices.Backward(l.logical.segments) {
		if offset >= seg.contentStart {
			return seg.physicalLine, seg.columnOffset + (offset - seg.contentStart)
		}
	}
	first := l.logical.segments[0]
	return first.physicalLine, first.columnOffset
}

// isParameterSigil reports whether ch is an engine parameter sigil byte that anchors a
// parameter binding directive.
//
// It is kept independent of the engine prefix table so the lexer remains engine-agnostic,
// and the parser later validates the byte against the supplied prefix lookup. The `{`
// byte is included for ClickHouse's `{name:Type}` form.
//
// Takes ch (byte) which is the candidate leading byte.
//
// Returns bool which is true when ch is one of the recognised sigil bytes.
func isParameterSigil(ch byte) bool {
	return ch == '$' || ch == '?' || ch == ':' || ch == '@' || ch == '{'
}

// isIdentStart reports whether ch may start an identifier.
//
// Takes ch (byte) which is the candidate byte.
//
// Returns bool which is true when ch is a letter or underscore.
func isIdentStart(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_'
}

// isIdentPart reports whether ch may continue an identifier.
//
// Takes ch (byte) which is the candidate byte.
//
// Returns bool which is true when ch may appear after the first identifier byte.
func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch)
}

// isDigit reports whether ch is an ASCII digit.
//
// Takes ch (byte) which is the candidate byte.
//
// Returns bool which is true when ch is in the range 0 to 9.
func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
