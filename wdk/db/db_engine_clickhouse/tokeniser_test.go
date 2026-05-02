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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tokenise_t(t *testing.T, input string) []token {
	t.Helper()
	tokens, err := tokenise(input)
	require.NoError(t, err)
	return tokens
}

func TestTokeniser_EmptyInputEmitsOnlyEOF(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "")
	require.Len(t, tokens, 1)
	assert.Equal(t, tokenEOF, tokens[0].kind)
}

func TestTokeniser_Identifiers(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "users user_id _name col2")
	kinds := tokenKinds(tokens)
	assert.Equal(t, []tokenKind{tokenIdentifier, tokenIdentifier, tokenIdentifier, tokenIdentifier, tokenEOF}, kinds)
	assert.Equal(t, "users", tokens[0].value)
	assert.Equal(t, "user_id", tokens[1].value)
	assert.Equal(t, "_name", tokens[2].value)
	assert.Equal(t, "col2", tokens[3].value)
}

func TestTokeniser_DoubleQuotedIdentifier(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, `"CamelCase"`)
	require.GreaterOrEqual(t, len(tokens), 1)
	assert.Equal(t, tokenIdentifier, tokens[0].kind)
	assert.Equal(t, "CamelCase", tokens[0].value)
}

func TestTokeniser_BacktickIdentifier(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "`weird name`")
	require.GreaterOrEqual(t, len(tokens), 1)
	assert.Equal(t, tokenIdentifier, tokens[0].kind)
	assert.Equal(t, "weird name", tokens[0].value)
}

func TestTokeniser_DoubledBacktickEscape(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "`with``embedded`")
	require.GreaterOrEqual(t, len(tokens), 1)
	assert.Equal(t, tokenIdentifier, tokens[0].kind)
	assert.Equal(t, "with`embedded", tokens[0].value)
}

func TestTokeniser_StringLiteralBasic(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "'hello world'")
	require.GreaterOrEqual(t, len(tokens), 1)
	assert.Equal(t, tokenString, tokens[0].kind)
	assert.Equal(t, "hello world", tokens[0].value)
}

func TestTokeniser_StringDoubledQuoteEscape(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "'O''Reilly'")
	require.GreaterOrEqual(t, len(tokens), 1)
	assert.Equal(t, tokenString, tokens[0].kind)
	assert.Equal(t, "O'Reilly", tokens[0].value)
}

func TestTokeniser_StringBackslashEscapes(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, `'tab\there\nnewline\\slash\'quote'`)
	require.GreaterOrEqual(t, len(tokens), 1)
	assert.Equal(t, tokenString, tokens[0].kind)
	assert.Equal(t, "tab\there\nnewline\\slash'quote", tokens[0].value)
}

func TestTokeniser_UnterminatedString(t *testing.T) {
	t.Parallel()

	_, err := tokenise("'unterminated")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unterminated string")
}

func TestTokeniser_Numbers(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "1 42 3.14 1.5e10 1.5E-3")
	for index := range 5 {
		assert.Equal(t, tokenNumber, tokens[index].kind, "token %d should be number", index)
	}
	assert.Equal(t, "1", tokens[0].value)
	assert.Equal(t, "42", tokens[1].value)
	assert.Equal(t, "3.14", tokens[2].value)
	assert.Equal(t, "1.5e10", tokens[3].value)
	assert.Equal(t, "1.5E-3", tokens[4].value)
}

func TestTokeniser_HexAndBinaryNumbers(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "0xFF 0xdeadbeef 0b1010 0o777")
	for index := range 4 {
		assert.Equal(t, tokenNumber, tokens[index].kind, "token %d should be number", index)
	}
	assert.Equal(t, "0xFF", tokens[0].value)
	assert.Equal(t, "0xdeadbeef", tokens[1].value)
	assert.Equal(t, "0b1010", tokens[2].value)
	assert.Equal(t, "0o777", tokens[3].value)
}

func TestTokeniser_ClickHouseParameter(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "{user_id:UInt64}")
	require.GreaterOrEqual(t, len(tokens), 1)
	assert.Equal(t, tokenClickHouseParam, tokens[0].kind)
	assert.Equal(t, "user_id:UInt64", tokens[0].value)
}

func TestTokeniser_ClickHouseParameterInContext(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "SELECT id FROM users WHERE id = {uid:UInt32}")
	hasParam := false
	for _, tok := range tokens {
		if tok.kind == tokenClickHouseParam {
			hasParam = true
			assert.Equal(t, "uid:UInt32", tok.value)
		}
	}
	assert.True(t, hasParam, "expected to see a ClickHouse placeholder token")
}

func TestTokeniser_UnterminatedClickHouseParameter(t *testing.T) {
	t.Parallel()

	_, err := tokenise("{user_id:UInt64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unterminated placeholder")
}

func TestTokeniser_LambdaArrow(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "x -> x + 1")
	hasArrow := false
	for _, tok := range tokens {
		if tok.kind == tokenArrow {
			hasArrow = true
			assert.Equal(t, "->", tok.value)
		}
	}
	assert.True(t, hasArrow, "expected lambda arrow token")
}

func TestTokeniser_CastOperator(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "x::String")
	require.GreaterOrEqual(t, len(tokens), 3)
	assert.Equal(t, tokenIdentifier, tokens[0].kind)
	assert.Equal(t, tokenCast, tokens[1].kind)
	assert.Equal(t, "::", tokens[1].value)
	assert.Equal(t, tokenIdentifier, tokens[2].kind)
	assert.Equal(t, "String", tokens[2].value)
}

func TestTokeniser_TwoCharOperators(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "a <= b >= c <> d != e || f && g << h >> i")
	wantOps := []string{"<=", ">=", "<>", "!=", "||", "&&", "<<", ">>"}
	foundOps := []string{}
	for _, tok := range tokens {
		if tok.kind == tokenOperator {
			foundOps = append(foundOps, tok.value)
		}
	}
	assert.Equal(t, wantOps, foundOps)
}

func TestTokeniser_LineComment(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "SELECT -- comment\n  id")
	identTokens := 0
	for _, tok := range tokens {
		if tok.kind == tokenIdentifier {
			identTokens++
		}
	}
	assert.Equal(t, 2, identTokens, "comment should not produce a token")
}

func TestTokeniser_BlockComment(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "SELECT /* block\n comment */ id")
	identTokens := 0
	for _, tok := range tokens {
		if tok.kind == tokenIdentifier {
			identTokens++
		}
	}
	assert.Equal(t, 2, identTokens)
}

func TestTokeniser_NestedBlockComment(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "SELECT /* outer /* inner */ still in outer */ id")
	identTokens := 0
	for _, tok := range tokens {
		if tok.kind == tokenIdentifier {
			identTokens++
		}
	}
	assert.Equal(t, 2, identTokens)
}

func TestTokeniser_Punctuation(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "a.b,c;d(e)[f]*g")
	wantKinds := []tokenKind{
		tokenIdentifier, tokenDot, tokenIdentifier,
		tokenComma, tokenIdentifier,
		tokenSemicolon, tokenIdentifier,
		tokenLeftParen, tokenIdentifier, tokenRightParen,
		tokenLeftBracket, tokenIdentifier, tokenRightBracket,
		tokenStar, tokenIdentifier,
		tokenEOF,
	}
	assert.Equal(t, wantKinds, tokenKinds(tokens))
}

func TestTokeniser_UnexpectedCharacter(t *testing.T) {
	t.Parallel()

	_, err := tokenise("SELECT \x00 FROM users")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected character")
}

func TestTokeniser_FullSelectShape(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "SELECT id, name FROM users WHERE id = {uid:UInt32} ORDER BY id LIMIT 10")

	wantIdentSubstrings := []string{"SELECT", "id", "name", "FROM", "users", "WHERE", "id", "ORDER", "BY", "id", "LIMIT"}
	identValues := []string{}
	for _, tok := range tokens {
		if tok.kind == tokenIdentifier {
			identValues = append(identValues, tok.value)
		}
	}
	assert.Equal(t, wantIdentSubstrings, identValues)

	hasParam := false
	for _, tok := range tokens {
		if tok.kind == tokenClickHouseParam && tok.value == "uid:UInt32" {
			hasParam = true
		}
	}
	assert.True(t, hasParam, "expected the placeholder to be tokenised")
}

func tokenKinds(tokens []token) []tokenKind {
	kinds := make([]tokenKind, len(tokens))
	for i, tok := range tokens {
		kinds[i] = tok.kind
	}
	return kinds
}

func TestTokeniser_NonASCIIIdentifierKeepsCodepoints(t *testing.T) {
	t.Parallel()

	tokens := tokenise_t(t, "é_col")

	require.GreaterOrEqual(t, len(tokens), 2, "expected at least one identifier token plus EOF")
	assert.Equal(t, tokenIdentifier, tokens[0].kind)
	assert.Equal(t, "é_col", tokens[0].value)
	assert.Equal(t, tokenEOF, tokens[len(tokens)-1].kind)
}

func TestTokeniser_HexAndUnicodeEscapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "hex", input: `'\x41\x42'`, want: "AB"},
		{name: "unicode escape", input: "'\\u00e9'", want: "é"},
		{name: "truncated hex falls back", input: `'\xZZ'`, want: "xZZ"},
		{name: "legacy single char", input: `'a\tb'`, want: "a\tb"},
	}
	for index := range cases {
		testCase := cases[index]
		t.Run(testCase.name, func(testRunner *testing.T) {
			testRunner.Parallel()
			tokens, err := tokenise(testCase.input)
			require.NoError(testRunner, err)
			require.NotEmpty(testRunner, tokens)
			assert.Equal(testRunner, testCase.want, tokens[0].value)
		})
	}
}
