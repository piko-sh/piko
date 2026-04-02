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
	"testing"

	"github.com/stretchr/testify/require"
)

func requireTokens(t *testing.T, input string, expected []token) {
	t.Helper()
	tokens, err := tokenise(input)
	require.NoError(t, err, "tokenise returned an unexpected error for input %q", input)
	require.NotEmpty(t, tokens, "tokenise returned no tokens")
	require.Equal(t, tokenEOF, tokens[len(tokens)-1].kind, "last token should be EOF")

	tokens = tokens[:len(tokens)-1]
	require.Equal(t, expected, tokens)
}

func requireTokeniseError(t *testing.T, input string, expectedSubstring string) {
	t.Helper()
	_, err := tokenise(input)
	require.Error(t, err, "expected tokenise to return an error for input %q", input)
	require.Contains(t, err.Error(), expectedSubstring)
}

func TestTokenise_SingleCharTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected token
	}{
		{
			name:     "left parenthesis",
			input:    "(",
			expected: token{kind: tokenLeftParen, value: "(", position: 0},
		},
		{
			name:     "right parenthesis",
			input:    ")",
			expected: token{kind: tokenRightParen, value: ")", position: 0},
		},
		{
			name:     "left bracket",
			input:    "[",
			expected: token{kind: tokenLeftBracket, value: "[", position: 0},
		},
		{
			name:     "right bracket",
			input:    "]",
			expected: token{kind: tokenRightBracket, value: "]", position: 0},
		},
		{
			name:     "comma",
			input:    ",",
			expected: token{kind: tokenComma, value: ",", position: 0},
		},
		{
			name:     "semicolon",
			input:    ";",
			expected: token{kind: tokenSemicolon, value: ";", position: 0},
		},
		{
			name:     "dot",
			input:    ".",
			expected: token{kind: tokenDot, value: ".", position: 0},
		},
		{
			name:     "star",
			input:    "*",
			expected: token{kind: tokenStar, value: "*", position: 0},
		},
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
		expected []token
	}{
		{
			name:  "simple identifier",
			input: "users",
			expected: []token{
				{kind: tokenIdentifier, value: "users", position: 0},
			},
		},
		{
			name:  "identifier with underscore",
			input: "_column_name",
			expected: []token{
				{kind: tokenIdentifier, value: "_column_name", position: 0},
			},
		},
		{
			name:  "double-quoted identifier",
			input: `"my column"`,
			expected: []token{
				{kind: tokenIdentifier, value: "my column", position: 0},
			},
		},
		{
			name:  "double-quoted identifier with escaped quote",
			input: `"col""name"`,
			expected: []token{
				{kind: tokenIdentifier, value: `col"name`, position: 0},
			},
		},
		{
			name:  "schema-qualified identifier",
			input: "main.users",
			expected: []token{
				{kind: tokenIdentifier, value: "main", position: 0},
				{kind: tokenDot, value: ".", position: 4},
				{kind: tokenIdentifier, value: "users", position: 5},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, testCase.expected)
		})
	}
}

func TestTokenise_Numbers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []token
	}{
		{
			name:  "integer",
			input: "42",
			expected: []token{
				{kind: tokenNumber, value: "42", position: 0},
			},
		},
		{
			name:  "decimal",
			input: "3.14",
			expected: []token{
				{kind: tokenNumber, value: "3.14", position: 0},
			},
		},
		{
			name:  "number with exponent",
			input: "1e10",
			expected: []token{
				{kind: tokenNumber, value: "1e10", position: 0},
			},
		},
		{
			name:  "decimal with signed exponent",
			input: "2.5E-3",
			expected: []token{
				{kind: tokenNumber, value: "2.5E-3", position: 0},
			},
		},
		{
			name:  "hex number",
			input: "0xFF",
			expected: []token{
				{kind: tokenNumber, value: "0xFF", position: 0},
			},
		},
		{
			name:  "octal number",
			input: "0o77",
			expected: []token{
				{kind: tokenNumber, value: "0o77", position: 0},
			},
		},
		{
			name:  "binary number",
			input: "0b1010",
			expected: []token{
				{kind: tokenNumber, value: "0b1010", position: 0},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, testCase.expected)
		})
	}
}

func TestTokenise_Strings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []token
	}{
		{
			name:  "simple string",
			input: "'hello'",
			expected: []token{
				{kind: tokenString, value: "hello", position: 0},
			},
		},
		{
			name:  "string with escaped single quote",
			input: "'it''s'",
			expected: []token{
				{kind: tokenString, value: "it's", position: 0},
			},
		},
		{
			name:  "empty string",
			input: "''",
			expected: []token{
				{kind: tokenString, value: "", position: 0},
			},
		},
		{
			name:  "bit string",
			input: "B'1010'",
			expected: []token{
				{kind: tokenBitString, value: "1010", position: 0},
			},
		},
		{
			name:  "bit string lowercase prefix",
			input: "b'110'",
			expected: []token{
				{kind: tokenBitString, value: "110", position: 0},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, testCase.expected)
		})
	}
}

func TestTokenise_Parameters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []token
	}{
		{
			name:  "dollar parameter",
			input: "$1",
			expected: []token{
				{kind: tokenDollarParam, value: "$1", position: 0},
			},
		},
		{
			name:  "dollar parameter with multiple digits",
			input: "$123",
			expected: []token{
				{kind: tokenDollarParam, value: "$123", position: 0},
			},
		},
		{
			name:  "named parameter with colon",
			input: ":name",
			expected: []token{
				{kind: tokenNamedParam, value: ":name", position: 0},
			},
		},
		{
			name:  "cast operator",
			input: "::",
			expected: []token{
				{kind: tokenCast, value: "::", position: 0},
			},
		},
		{
			name:  "cast used in expression",
			input: "x::int",
			expected: []token{
				{kind: tokenIdentifier, value: "x", position: 0},
				{kind: tokenCast, value: "::", position: 1},
				{kind: tokenIdentifier, value: "int", position: 3},
			},
		},
		{
			name:  "colon as bare operator",
			input: ": ",
			expected: []token{
				{kind: tokenOperator, value: ":", position: 0},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, testCase.expected)
		})
	}
}

func TestTokenise_Operators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []token
	}{
		{
			name:  "less than or equal",
			input: "<=",
			expected: []token{
				{kind: tokenOperator, value: "<=", position: 0},
			},
		},
		{
			name:  "greater than or equal",
			input: ">=",
			expected: []token{
				{kind: tokenOperator, value: ">=", position: 0},
			},
		},
		{
			name:  "not equal angle brackets",
			input: "<>",
			expected: []token{
				{kind: tokenOperator, value: "<>", position: 0},
			},
		},
		{
			name:  "not equal exclamation",
			input: "!=",
			expected: []token{
				{kind: tokenOperator, value: "!=", position: 0},
			},
		},
		{
			name:  "string concatenation",
			input: "||",
			expected: []token{
				{kind: tokenOperator, value: "||", position: 0},
			},
		},
		{
			name:  "left shift",
			input: "<<",
			expected: []token{
				{kind: tokenOperator, value: "<<", position: 0},
			},
		},
		{
			name:  "right shift",
			input: ">>",
			expected: []token{
				{kind: tokenOperator, value: ">>", position: 0},
			},
		},
		{
			name:  "arrow operator",
			input: "->",
			expected: []token{
				{kind: tokenArrow, value: "->", position: 0},
			},
		},
		{
			name:  "double arrow operator",
			input: "->>",
			expected: []token{
				{kind: tokenDoubleArrow, value: "->>", position: 0},
			},
		},
		{
			name:  "contains operator",
			input: "@>",
			expected: []token{
				{kind: tokenOperator, value: "@>", position: 0},
			},
		},
		{
			name:  "contained by operator",
			input: "<@",
			expected: []token{
				{kind: tokenOperator, value: "<@", position: 0},
			},
		},
		{
			name:  "regex match case insensitive",
			input: "~*",
			expected: []token{
				{kind: tokenOperator, value: "~*", position: 0},
			},
		},
		{
			name:  "not regex",
			input: "!~",
			expected: []token{
				{kind: tokenOperator, value: "!~", position: 0},
			},
		},
		{
			name:  "not regex case insensitive",
			input: "!~*",
			expected: []token{
				{kind: tokenOperator, value: "!~*", position: 0},
			},
		},
		{
			name:  "logical and",
			input: "&&",
			expected: []token{
				{kind: tokenOperator, value: "&&", position: 0},
			},
		},
		{
			name:  "plus",
			input: "+",
			expected: []token{
				{kind: tokenOperator, value: "+", position: 0},
			},
		},
		{
			name:  "caret",
			input: "^",
			expected: []token{
				{kind: tokenOperator, value: "^", position: 0},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, testCase.expected)
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
			name:     "whitespace only produces no tokens",
			input:    "   \t\n  ",
			expected: []token{},
		},
		{
			name:  "line comment is skipped",
			input: "-- this is a comment\nSELECT",
			expected: []token{
				{kind: tokenIdentifier, value: "SELECT", position: 21},
			},
		},
		{
			name:  "block comment is skipped",
			input: "/* block */SELECT",
			expected: []token{
				{kind: tokenIdentifier, value: "SELECT", position: 11},
			},
		},
		{
			name:  "nested block comment is skipped",
			input: "/* outer /* inner */ still comment */SELECT",
			expected: []token{
				{kind: tokenIdentifier, value: "SELECT", position: 37},
			},
		},
		{
			name:  "tokens separated by various whitespace",
			input: "a \t b \n c",
			expected: []token{
				{kind: tokenIdentifier, value: "a", position: 0},
				{kind: tokenIdentifier, value: "b", position: 4},
				{kind: tokenIdentifier, value: "c", position: 8},
			},
		},
		{
			name:  "inline block comment between tokens",
			input: "a /* comment */ b",
			expected: []token{
				{kind: tokenIdentifier, value: "a", position: 0},
				{kind: tokenIdentifier, value: "b", position: 16},
			},
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
		name              string
		input             string
		expectedSubstring string
	}{
		{
			name:              "unterminated string literal",
			input:             "'hello",
			expectedSubstring: "unterminated string literal",
		},
		{
			name:              "unterminated quoted identifier",
			input:             `"column`,
			expectedSubstring: "unterminated quoted identifier",
		},
		{
			name:              "bare dollar without digit",
			input:             "$ ",
			expectedSubstring: "unexpected character $",
		},
		{
			name:              "unterminated bit string",
			input:             "B'101",
			expectedSubstring: "unterminated bit string",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokeniseError(t, testCase.input, testCase.expectedSubstring)
		})
	}
}

func TestTokenise_CompleteStatements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []token
	}{
		{
			name:  "select with dollar parameter",
			input: "SELECT id FROM t WHERE id = $1",
			expected: []token{
				{kind: tokenIdentifier, value: "SELECT", position: 0},
				{kind: tokenIdentifier, value: "id", position: 7},
				{kind: tokenIdentifier, value: "FROM", position: 10},
				{kind: tokenIdentifier, value: "t", position: 15},
				{kind: tokenIdentifier, value: "WHERE", position: 17},
				{kind: tokenIdentifier, value: "id", position: 23},
				{kind: tokenOperator, value: "=", position: 26},
				{kind: tokenDollarParam, value: "$1", position: 28},
			},
		},
		{
			name:  "select with cast and arrow",
			input: "SELECT data->>'key' FROM t WHERE val::int > $1",
			expected: []token{
				{kind: tokenIdentifier, value: "SELECT", position: 0},
				{kind: tokenIdentifier, value: "data", position: 7},
				{kind: tokenDoubleArrow, value: "->>", position: 11},
				{kind: tokenString, value: "key", position: 14},
				{kind: tokenIdentifier, value: "FROM", position: 20},
				{kind: tokenIdentifier, value: "t", position: 25},
				{kind: tokenIdentifier, value: "WHERE", position: 27},
				{kind: tokenIdentifier, value: "val", position: 33},
				{kind: tokenCast, value: "::", position: 36},
				{kind: tokenIdentifier, value: "int", position: 38},
				{kind: tokenOperator, value: ">", position: 42},
				{kind: tokenDollarParam, value: "$1", position: 44},
			},
		},
		{
			name:  "insert with named parameter",
			input: "INSERT INTO users (name) VALUES (:name)",
			expected: []token{
				{kind: tokenIdentifier, value: "INSERT", position: 0},
				{kind: tokenIdentifier, value: "INTO", position: 7},
				{kind: tokenIdentifier, value: "users", position: 12},
				{kind: tokenLeftParen, value: "(", position: 18},
				{kind: tokenIdentifier, value: "name", position: 19},
				{kind: tokenRightParen, value: ")", position: 23},
				{kind: tokenIdentifier, value: "VALUES", position: 25},
				{kind: tokenLeftParen, value: "(", position: 32},
				{kind: tokenNamedParam, value: ":name", position: 33},
				{kind: tokenRightParen, value: ")", position: 38},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, testCase.expected)
		})
	}
}
