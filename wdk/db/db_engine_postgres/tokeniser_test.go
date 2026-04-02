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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireTokens(t *testing.T, input string, expected []token) {
	t.Helper()
	tokens, err := tokenise(input)
	require.NoError(t, err)

	if len(tokens) > 0 && tokens[len(tokens)-1].kind == tokenEOF {
		tokens = tokens[:len(tokens)-1]
	}
	require.Equal(t, len(expected), len(tokens), "token count mismatch for input: %q", input)
	for i := range expected {
		assert.Equal(t, expected[i].kind, tokens[i].kind, "token %d kind mismatch for input: %q", i, input)
		assert.Equal(t, expected[i].value, tokens[i].value, "token %d value mismatch for input: %q", i, input)
	}
}

func requireTokeniseError(t *testing.T, input string, messageContains string) {
	t.Helper()
	_, err := tokenise(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), messageContains)
}

func TestTokenise_SingleCharTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected token
	}{
		{name: "left parenthesis", input: "(", expected: token{kind: tokenLeftParen, value: "("}},
		{name: "right parenthesis", input: ")", expected: token{kind: tokenRightParen, value: ")"}},
		{name: "left bracket", input: "[", expected: token{kind: tokenLeftBracket, value: "["}},
		{name: "right bracket", input: "]", expected: token{kind: tokenRightBracket, value: "]"}},
		{name: "comma", input: ",", expected: token{kind: tokenComma, value: ","}},
		{name: "semicolon", input: ";", expected: token{kind: tokenSemicolon, value: ";"}},
		{name: "dot", input: ".", expected: token{kind: tokenDot, value: "."}},
		{name: "star", input: "*", expected: token{kind: tokenStar, value: "*"}},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, []token{testCase.expected})
		})
	}
}

func TestTokenise_Identifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected token
	}{
		{
			name:     "simple keyword",
			input:    "SELECT",
			expected: token{kind: tokenIdentifier, value: "SELECT"},
		},
		{
			name:     "underscore prefix",
			input:    "_col",
			expected: token{kind: tokenIdentifier, value: "_col"},
		},
		{
			name:     "mixed case",
			input:    "MyTable",
			expected: token{kind: tokenIdentifier, value: "MyTable"},
		},
		{
			name:     "with digits",
			input:    "col1",
			expected: token{kind: tokenIdentifier, value: "col1"},
		},
		{
			name:     "quoted identifier with spaces",
			input:    `"my column"`,
			expected: token{kind: tokenIdentifier, value: "my column"},
		},
		{
			name:     "quoted identifier with escaped quotes",
			input:    `"say ""hello"""`,
			expected: token{kind: tokenIdentifier, value: `say "hello"`},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, []token{testCase.expected})
		})
	}
}

func TestTokenise_Numbers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected token
	}{
		{
			name:     "integer",
			input:    "42",
			expected: token{kind: tokenNumber, value: "42"},
		},
		{
			name:     "decimal",
			input:    "3.14",
			expected: token{kind: tokenNumber, value: "3.14"},
		},
		{
			name:     "exponent notation",
			input:    "1e10",
			expected: token{kind: tokenNumber, value: "1e10"},
		},
		{
			name:     "decimal with negative exponent",
			input:    "1.5e-3",
			expected: token{kind: tokenNumber, value: "1.5e-3"},
		},
		{
			name:     "hexadecimal",
			input:    "0xFF",
			expected: token{kind: tokenNumber, value: "0xFF"},
		},
		{
			name:     "octal",
			input:    "0o17",
			expected: token{kind: tokenNumber, value: "0o17"},
		},
		{
			name:     "binary",
			input:    "0b1010",
			expected: token{kind: tokenNumber, value: "0b1010"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, []token{testCase.expected})
		})
	}
}

func TestTokenise_Strings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		expectedKind tokenKind
		expectedVal  string
	}{
		{
			name:         "simple string",
			input:        "'hello'",
			expectedKind: tokenString,
			expectedVal:  "hello",
		},
		{
			name:         "string with escaped single quote",
			input:        "'it''s'",
			expectedKind: tokenString,
			expectedVal:  "it's",
		},
		{
			name:         "escape string with backslash sequence",
			input:        `E'hello\nworld'`,
			expectedKind: tokenEscapeString,
			expectedVal:  "hellonworld",
		},
		{
			name:         "dollar-quoted string with empty tag",
			input:        "$$body$$",
			expectedKind: tokenDollarString,
			expectedVal:  "body",
		},
		{
			name:         "dollar-quoted string with named tag",
			input:        "$tag$body$tag$",
			expectedKind: tokenDollarString,
			expectedVal:  "body",
		},
		{
			name:         "bit string literal",
			input:        "B'1010'",
			expectedKind: tokenBitString,
			expectedVal:  "1010",
		},
		{
			name:         "lowercase escape string prefix",
			input:        `e'tab\there'`,
			expectedKind: tokenEscapeString,
			expectedVal:  "tabthere",
		},
		{
			name:         "lowercase bit string prefix",
			input:        "b'0011'",
			expectedKind: tokenBitString,
			expectedVal:  "0011",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, []token{
				{kind: testCase.expectedKind, value: testCase.expectedVal},
			})
		})
	}
}

func TestTokenise_Parameters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected token
	}{
		{
			name:     "dollar parameter",
			input:    "$1",
			expected: token{kind: tokenDollarParam, value: "$1"},
		},
		{
			name:     "multi-digit dollar parameter",
			input:    "$123",
			expected: token{kind: tokenDollarParam, value: "$123"},
		},
		{
			name:     "named parameter",
			input:    ":name",
			expected: token{kind: tokenNamedParam, value: ":name"},
		},
		{
			name:     "cast operator",
			input:    "::",
			expected: token{kind: tokenCast, value: "::"},
		},
		{
			name:     "bare colon as operator",
			input:    ": ",
			expected: token{kind: tokenOperator, value: ":"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, []token{testCase.expected})
		})
	}
}

func TestTokenise_Operators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected token
	}{

		{name: "less than or equal", input: "<=", expected: token{kind: tokenOperator, value: "<="}},
		{name: "greater than or equal", input: ">=", expected: token{kind: tokenOperator, value: ">="}},
		{name: "not equal angle brackets", input: "<>", expected: token{kind: tokenOperator, value: "<>"}},
		{name: "not equal exclamation", input: "!=", expected: token{kind: tokenOperator, value: "!="}},
		{name: "string concatenation", input: "||", expected: token{kind: tokenOperator, value: "||"}},
		{name: "left shift", input: "<<", expected: token{kind: tokenOperator, value: "<<"}},
		{name: "right shift", input: ">>", expected: token{kind: tokenOperator, value: ">>"}},

		{name: "arrow", input: "->", expected: token{kind: tokenArrow, value: "->"}},
		{name: "double arrow", input: "->>", expected: token{kind: tokenDoubleArrow, value: "->>"}},
		{name: "hash arrow", input: "#>", expected: token{kind: tokenHashArrow, value: "#>"}},
		{name: "hash double arrow", input: "#>>", expected: token{kind: tokenHashDoubleArrow, value: "#>>"}},

		{name: "equals", input: "=", expected: token{kind: tokenOperator, value: "="}},
		{name: "less than", input: "<", expected: token{kind: tokenOperator, value: "<"}},
		{name: "greater than", input: ">", expected: token{kind: tokenOperator, value: ">"}},
		{name: "plus", input: "+", expected: token{kind: tokenOperator, value: "+"}},
		{name: "minus", input: "-", expected: token{kind: tokenOperator, value: "-"}},
		{name: "divide", input: "/", expected: token{kind: tokenOperator, value: "/"}},
		{name: "modulo", input: "%", expected: token{kind: tokenOperator, value: "%"}},
		{name: "tilde", input: "~", expected: token{kind: tokenOperator, value: "~"}},
		{name: "ampersand", input: "&", expected: token{kind: tokenOperator, value: "&"}},
		{name: "pipe", input: "|", expected: token{kind: tokenOperator, value: "|"}},
		{name: "exclamation", input: "!", expected: token{kind: tokenOperator, value: "!"}},
		{name: "hash", input: "#", expected: token{kind: tokenOperator, value: "#"}},
		{name: "caret", input: "^", expected: token{kind: tokenOperator, value: "^"}},

		{name: "logical and", input: "&&", expected: token{kind: tokenOperator, value: "&&"}},
		{name: "contains", input: "@>", expected: token{kind: tokenOperator, value: "@>"}},
		{name: "contained by", input: "<@", expected: token{kind: tokenOperator, value: "<@"}},
		{name: "case-insensitive regex", input: "~*", expected: token{kind: tokenOperator, value: "~*"}},
		{name: "not regex", input: "!~", expected: token{kind: tokenOperator, value: "!~"}},

		{name: "case-insensitive not regex", input: "!~*", expected: token{kind: tokenOperator, value: "!~*"}},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, []token{testCase.expected})
		})
	}
}

func TestTokenise_WhitespaceAndComments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []token
	}{
		{
			name:     "leading and trailing whitespace is stripped",
			input:    "  SELECT  ",
			expected: []token{{kind: tokenIdentifier, value: "SELECT"}},
		},
		{
			name:     "tabs and newlines are treated as whitespace",
			input:    "\t\nSELECT\r\n",
			expected: []token{{kind: tokenIdentifier, value: "SELECT"}},
		},
		{
			name:  "line comment is skipped",
			input: "SELECT -- this is a comment\n1",
			expected: []token{
				{kind: tokenIdentifier, value: "SELECT"},
				{kind: tokenNumber, value: "1"},
			},
		},
		{
			name:  "block comment is skipped",
			input: "SELECT /* comment */ 1",
			expected: []token{
				{kind: tokenIdentifier, value: "SELECT"},
				{kind: tokenNumber, value: "1"},
			},
		},
		{
			name:     "nested block comment is fully consumed",
			input:    "/* outer /* inner */ */",
			expected: []token{},
		},
		{
			name:     "empty input produces no tokens",
			input:    "",
			expected: []token{},
		},
		{
			name:     "whitespace-only input produces no tokens",
			input:    "   \t\n  ",
			expected: []token{},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, testCase.expected)
		})
	}
}

func TestTokenise_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		input           string
		messageContains string
	}{
		{
			name:            "unterminated string literal",
			input:           "'hello",
			messageContains: "unterminated string literal",
		},
		{
			name:            "unterminated escape string",
			input:           "E'hello",
			messageContains: "unterminated escape string",
		},
		{
			name:            "unterminated quoted identifier",
			input:           `"hello`,
			messageContains: "unterminated quoted identifier",
		},
		{
			name:            "unterminated dollar-quoted string",
			input:           "$$hello",
			messageContains: "unterminated dollar-quoted string",
		},
		{
			name:            "unterminated tagged dollar-quoted string",
			input:           "$tag$hello",
			messageContains: "unterminated dollar-quoted string",
		},
		{
			name:            "unterminated bit string",
			input:           "B'1010",
			messageContains: "unterminated bit string",
		},
		{
			name:            "unexpected character",
			input:           "SELECT \x00",
			messageContains: "unexpected character",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokeniseError(t, testCase.input, testCase.messageContains)
		})
	}
}

func TestTokenise_CompleteStatements(t *testing.T) {
	t.Parallel()

	t.Run("SELECT with WHERE clause and dollar parameter", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "SELECT id, name FROM users WHERE id = $1", []token{
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenIdentifier, value: "id"},
			{kind: tokenComma, value: ","},
			{kind: tokenIdentifier, value: "name"},
			{kind: tokenIdentifier, value: "FROM"},
			{kind: tokenIdentifier, value: "users"},
			{kind: tokenIdentifier, value: "WHERE"},
			{kind: tokenIdentifier, value: "id"},
			{kind: tokenOperator, value: "="},
			{kind: tokenDollarParam, value: "$1"},
		})
	})

	t.Run("INSERT with multiple parameters", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "INSERT INTO foo (a, b) VALUES ($1, $2)", []token{
			{kind: tokenIdentifier, value: "INSERT"},
			{kind: tokenIdentifier, value: "INTO"},
			{kind: tokenIdentifier, value: "foo"},
			{kind: tokenLeftParen, value: "("},
			{kind: tokenIdentifier, value: "a"},
			{kind: tokenComma, value: ","},
			{kind: tokenIdentifier, value: "b"},
			{kind: tokenRightParen, value: ")"},
			{kind: tokenIdentifier, value: "VALUES"},
			{kind: tokenLeftParen, value: "("},
			{kind: tokenDollarParam, value: "$1"},
			{kind: tokenComma, value: ","},
			{kind: tokenDollarParam, value: "$2"},
			{kind: tokenRightParen, value: ")"},
		})
	})

	t.Run("multi-statement separated by semicolon", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "SELECT 1; SELECT 2", []token{
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenNumber, value: "1"},
			{kind: tokenSemicolon, value: ";"},
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenNumber, value: "2"},
		})
	})

	t.Run("SELECT with type cast", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "SELECT col::integer FROM t", []token{
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenIdentifier, value: "col"},
			{kind: tokenCast, value: "::"},
			{kind: tokenIdentifier, value: "integer"},
			{kind: tokenIdentifier, value: "FROM"},
			{kind: tokenIdentifier, value: "t"},
		})
	})

	t.Run("JSONB arrow operator chain", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "data->'key'->>'nested'", []token{
			{kind: tokenIdentifier, value: "data"},
			{kind: tokenArrow, value: "->"},
			{kind: tokenString, value: "key"},
			{kind: tokenDoubleArrow, value: "->>"},
			{kind: tokenString, value: "nested"},
		})
	})

	t.Run("function call with star and dot-qualified name", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "SELECT s.name, COUNT(*) FROM schema1.users s", []token{
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenIdentifier, value: "s"},
			{kind: tokenDot, value: "."},
			{kind: tokenIdentifier, value: "name"},
			{kind: tokenComma, value: ","},
			{kind: tokenIdentifier, value: "COUNT"},
			{kind: tokenLeftParen, value: "("},
			{kind: tokenStar, value: "*"},
			{kind: tokenRightParen, value: ")"},
			{kind: tokenIdentifier, value: "FROM"},
			{kind: tokenIdentifier, value: "schema1"},
			{kind: tokenDot, value: "."},
			{kind: tokenIdentifier, value: "users"},
			{kind: tokenIdentifier, value: "s"},
		})
	})

	t.Run("array subscript with bracket notation", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "arr[1]", []token{
			{kind: tokenIdentifier, value: "arr"},
			{kind: tokenLeftBracket, value: "["},
			{kind: tokenNumber, value: "1"},
			{kind: tokenRightBracket, value: "]"},
		})
	})
}
