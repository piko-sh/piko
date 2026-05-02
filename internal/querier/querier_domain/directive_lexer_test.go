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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lexerFromContent(content string) *directiveLexer {
	return newDirectiveLexer(logicalLine{
		content: content,
		segments: []lineSegment{{
			physicalLine: 1,
			contentStart: 0,
			contentEnd:   len(content),
			columnOffset: 1,
		}},
		startLine: 1,
		endLine:   1,
	})
}

func drainTokens(t *testing.T, lexer *directiveLexer) []token {
	t.Helper()
	var tokens []token
	for {
		tok := lexer.next()
		if tok.kind == tokenEOF {
			return tokens
		}
		tokens = append(tokens, tok)
		require.LessOrEqual(t, len(tokens), 1024, "lexer ran past expected token budget")
	}
}

func TestDirectiveLexerScansPikoCallTokens(t *testing.T) {
	lexer := lexerFromContent("piko.embed(orders, from: o)")

	tokens := drainTokens(t, lexer)

	require.Len(t, tokens, 10)
	assert.Equal(t, tokenIdent, tokens[0].kind)
	assert.Equal(t, "piko", tokens[0].lexeme)
	assert.Equal(t, tokenDot, tokens[1].kind)
	assert.Equal(t, tokenIdent, tokens[2].kind)
	assert.Equal(t, "embed", tokens[2].lexeme)
	assert.Equal(t, tokenLParen, tokens[3].kind)
	assert.Equal(t, tokenIdent, tokens[4].kind)
	assert.Equal(t, "orders", tokens[4].lexeme)
	assert.Equal(t, tokenComma, tokens[5].kind)
	assert.Equal(t, tokenIdent, tokens[6].kind)
	assert.Equal(t, "from", tokens[6].lexeme)

	assert.Equal(t, tokenColon, tokens[7].kind)
	assert.Equal(t, tokenIdent, tokens[8].kind)
	assert.Equal(t, "o", tokens[8].lexeme)
	assert.Equal(t, tokenRParen, tokens[9].kind)
}

func TestDirectiveLexerScansDollarSigilAnchor(t *testing.T) {
	lexer := lexerFromContent("$1 as piko.param(status)")

	tokens := drainTokens(t, lexer)

	require.NotEmpty(t, tokens)
	first := tokens[0]
	assert.Equal(t, tokenSigil, first.kind, "first token should be a sigil")
	assert.Equal(t, "$1", first.lexeme)
}

func TestDirectiveLexerScansNamedColonSigil(t *testing.T) {
	lexer := lexerFromContent(":email as piko.param()")

	tokens := drainTokens(t, lexer)

	require.NotEmpty(t, tokens)
	assert.Equal(t, tokenSigil, tokens[0].kind)
	assert.Equal(t, ":email", tokens[0].lexeme)
}

func TestDirectiveLexerScansBracedSigil(t *testing.T) {
	lexer := lexerFromContent("{count:UInt32} as piko.param()")

	tokens := drainTokens(t, lexer)

	require.NotEmpty(t, tokens)
	assert.Equal(t, tokenSigil, tokens[0].kind)
	assert.Equal(t, "{count:UInt32}", tokens[0].lexeme)
}

func TestDirectiveLexerUnterminatedBracedSigilEmitsInvalid(t *testing.T) {
	lexer := lexerFromContent("{count:UInt32")

	tokens := drainTokens(t, lexer)

	require.NotEmpty(t, tokens)
	assert.Equal(t, tokenInvalid, tokens[0].kind, "unterminated brace sigil should emit tokenInvalid")
	assert.Equal(t, "{count:UInt32", tokens[0].lexeme)
}

func TestDirectiveLexerUnterminatedBracedSigilSpanCoversInput(t *testing.T) {
	lexer := lexerFromContent("{count:UInt32")

	tokens := drainTokens(t, lexer)

	require.NotEmpty(t, tokens)
	invalidToken := tokens[0]
	assert.Equal(t, tokenInvalid, invalidToken.kind)
	assert.Equal(t, 1, invalidToken.span.Line)
	assert.Equal(t, 1, invalidToken.span.Column, "unterminated sigil span should anchor at column 1 (the opener)")
	assert.Equal(t, 1, invalidToken.span.EndLine)
	assert.Equal(t, 14, invalidToken.span.EndColumn, "unterminated sigil span should extend to one past the consumed content")
}

func TestDirectiveLexerHonoursDoubledQuoteEscape(t *testing.T) {
	lexer := lexerFromContent(`"he said ""hello"""`)

	tokens := drainTokens(t, lexer)

	require.Len(t, tokens, 1)
	assert.Equal(t, tokenString, tokens[0].kind)
	assert.Equal(t, `he said "hello"`, tokens[0].lexeme, "doubled-quote should decode to a literal quote")
}

func TestDirectiveLexerHonoursBackslashEscape(t *testing.T) {
	lexer := lexerFromContent(`"line\nbreak"`)

	tokens := drainTokens(t, lexer)

	require.Len(t, tokens, 1)
	assert.Equal(t, tokenString, tokens[0].kind)
	assert.Equal(t, `linenbreak`, tokens[0].lexeme, "backslash-n should decode to the literal n character")
}

func TestDirectiveLexerUnterminatedStringEmitsInvalid(t *testing.T) {
	lexer := lexerFromContent(`"never closed`)

	tokens := drainTokens(t, lexer)

	require.Len(t, tokens, 1)
	assert.Equal(t, tokenInvalid, tokens[0].kind, "unterminated string should emit tokenInvalid")
}

func TestDirectiveLexerRejectsUtf8LeadBytesInsideIdentifier(t *testing.T) {

	lexer := lexerFromContent("piko\xc3\xa9")

	tokens := drainTokens(t, lexer)

	require.GreaterOrEqual(t, len(tokens), 2)
	assert.Equal(t, tokenIdent, tokens[0].kind)
	assert.Equal(t, "piko", tokens[0].lexeme)
	assert.Equal(t, tokenInvalid, tokens[1].kind, "UTF-8 lead byte should be rejected as tokenInvalid")
}

func TestDirectiveLexerEOFAtEndOfInput(t *testing.T) {
	lexer := lexerFromContent("")

	first := lexer.peek()
	assert.Equal(t, tokenEOF, first.kind)

	consumed := lexer.next()
	assert.Equal(t, tokenEOF, consumed.kind)
}

func TestDirectiveLexerSpanReportsByteColumns(t *testing.T) {
	lexer := lexerFromContent("piko.embed(orders)")

	pikoToken := lexer.next()
	dotToken := lexer.next()
	embedToken := lexer.next()

	assert.Equal(t, 1, pikoToken.span.Line)
	assert.Equal(t, 1, pikoToken.span.Column)
	assert.Equal(t, 5, pikoToken.span.EndColumn, "piko spans columns 1..5 (exclusive end)")
	assert.Equal(t, 5, dotToken.span.Column, "dot should land at column 5 (one-based, after 'piko')")
	assert.Equal(t, 6, embedToken.span.Column, "ident after dot should start at column 6")
}

func TestDirectiveLexerScansNumericLiteral(t *testing.T) {
	lexer := lexerFromContent("piko(max: -123)")

	var sawNumber bool
	for tok := lexer.next(); tok.kind != tokenEOF; tok = lexer.next() {
		if tok.kind == tokenNumber {
			sawNumber = true
			assert.Equal(t, "-123", tok.lexeme)
			break
		}
	}
	assert.True(t, sawNumber, "expected a negative integer literal to be lexed as tokenNumber")
}

func TestDirectiveLexerPeekDoesNotConsume(t *testing.T) {
	lexer := lexerFromContent("piko(name: Foo)")

	first := lexer.peek()
	again := lexer.peek()
	consumed := lexer.next()

	assert.Equal(t, first.kind, again.kind)
	assert.Equal(t, first.lexeme, again.lexeme)
	assert.Equal(t, first.kind, consumed.kind)
}

func TestDirectiveLexerPeekNReturnsLookahead(t *testing.T) {
	lexer := lexerFromContent("piko(name: Foo)")

	zero := lexer.peekN(0)
	one := lexer.peekN(1)
	two := lexer.peekN(2)

	assert.Equal(t, tokenIdent, zero.kind)
	assert.Equal(t, "piko", zero.lexeme)
	assert.Equal(t, tokenLParen, one.kind)
	assert.Equal(t, tokenIdent, two.kind)
	assert.Equal(t, "name", two.lexeme)
}
