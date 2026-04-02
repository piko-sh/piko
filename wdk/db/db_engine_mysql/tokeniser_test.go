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
		{
			name:  "left parenthesis",
			input: "(",
			expected: token{
				kind:  tokenLeftParen,
				value: "(",
			},
		},
		{
			name:  "right parenthesis",
			input: ")",
			expected: token{
				kind:  tokenRightParen,
				value: ")",
			},
		},
		{
			name:  "left bracket",
			input: "[",
			expected: token{
				kind:  tokenLeftBracket,
				value: "[",
			},
		},
		{
			name:  "right bracket",
			input: "]",
			expected: token{
				kind:  tokenRightBracket,
				value: "]",
			},
		},
		{
			name:  "comma",
			input: ",",
			expected: token{
				kind:  tokenComma,
				value: ",",
			},
		},
		{
			name:  "semicolon",
			input: ";",
			expected: token{
				kind:  tokenSemicolon,
				value: ";",
			},
		},
		{
			name:  "dot",
			input: ".",
			expected: token{
				kind:  tokenDot,
				value: ".",
			},
		},
		{
			name:  "star",
			input: "*",
			expected: token{
				kind:  tokenStar,
				value: "*",
			},
		},
		{
			name:  "question mark",
			input: "?",
			expected: token{
				kind:  tokenQuestionMark,
				value: "?",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireTokens(t, testCase.input, []token{testCase.expected})
		})
	}

	t.Run("all single-character tokens in sequence", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "( ) [ ] , ; . * ?", []token{
			{kind: tokenLeftParen, value: "("},
			{kind: tokenRightParen, value: ")"},
			{kind: tokenLeftBracket, value: "["},
			{kind: tokenRightBracket, value: "]"},
			{kind: tokenComma, value: ","},
			{kind: tokenSemicolon, value: ";"},
			{kind: tokenDot, value: "."},
			{kind: tokenStar, value: "*"},
			{kind: tokenQuestionMark, value: "?"},
		})
	})
}

func TestTokenise_Identifiers(t *testing.T) {
	t.Parallel()

	t.Run("simple identifier", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "users", []token{
			{kind: tokenIdentifier, value: "users"},
		})
	})

	t.Run("identifier with digits and underscores", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "user_id_2", []token{
			{kind: tokenIdentifier, value: "user_id_2"},
		})
	})

	t.Run("identifier starting with underscore", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "_private", []token{
			{kind: tokenIdentifier, value: "_private"},
		})
	})

	t.Run("backtick-quoted identifier", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "`column`", []token{
			{kind: tokenIdentifier, value: "column"},
		})
	})

	t.Run("backtick-quoted identifier with spaces", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "`my column`", []token{
			{kind: tokenIdentifier, value: "my column"},
		})
	})

	t.Run("backtick-quoted identifier with escaped backtick", func(t *testing.T) {
		t.Parallel()

		requireTokens(t, "`col``name`", []token{
			{kind: tokenIdentifier, value: "col`name"},
		})
	})

	t.Run("double-quoted identifier", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, `"table_name"`, []token{
			{kind: tokenIdentifier, value: "table_name"},
		})
	})

	t.Run("double-quoted identifier with escaped double quote", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, `"table""name"`, []token{
			{kind: tokenIdentifier, value: `table"name`},
		})
	})

	t.Run("multiple identifiers separated by dot", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "db.users", []token{
			{kind: tokenIdentifier, value: "db"},
			{kind: tokenDot, value: "."},
			{kind: tokenIdentifier, value: "users"},
		})
	})

	t.Run("mixed bare and backtick identifiers", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "`schema`.table_name", []token{
			{kind: tokenIdentifier, value: "schema"},
			{kind: tokenDot, value: "."},
			{kind: tokenIdentifier, value: "table_name"},
		})
	})
}

func TestTokenise_Numbers(t *testing.T) {
	t.Parallel()

	t.Run("integer", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "42", []token{
			{kind: tokenNumber, value: "42"},
		})
	})

	t.Run("decimal", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "3.14", []token{
			{kind: tokenNumber, value: "3.14"},
		})
	})

	t.Run("decimal starting with zero", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "0.5", []token{
			{kind: tokenNumber, value: "0.5"},
		})
	})

	t.Run("exponent notation", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "1e10", []token{
			{kind: tokenNumber, value: "1e10"},
		})
	})

	t.Run("exponent with uppercase E", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "2E5", []token{
			{kind: tokenNumber, value: "2E5"},
		})
	})

	t.Run("exponent with positive sign", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "1.5e+3", []token{
			{kind: tokenNumber, value: "1.5e+3"},
		})
	})

	t.Run("exponent with negative sign", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "6.022e-23", []token{
			{kind: tokenNumber, value: "6.022e-23"},
		})
	})

	t.Run("hexadecimal number", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "0xFF", []token{
			{kind: tokenNumber, value: "0xFF"},
		})
	})

	t.Run("hexadecimal number lowercase prefix", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "0xDEAD", []token{
			{kind: tokenNumber, value: "0xDEAD"},
		})
	})

	t.Run("binary number", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "0b1010", []token{
			{kind: tokenNumber, value: "0b1010"},
		})
	})

	t.Run("binary number uppercase prefix", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "0B110", []token{
			{kind: tokenNumber, value: "0B110"},
		})
	})
}

func TestTokenise_Strings(t *testing.T) {
	t.Parallel()

	t.Run("simple string", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "'hello'", []token{
			{kind: tokenString, value: "hello"},
		})
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "''", []token{
			{kind: tokenString, value: ""},
		})
	})

	t.Run("string with escaped single quote (doubled)", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "'it''s'", []token{
			{kind: tokenString, value: "it's"},
		})
	})

	t.Run("string with backslash escape newline", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, `'line\none'`, []token{
			{kind: tokenString, value: "line\none"},
		})
	})

	t.Run("string with backslash escape tab", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, `'col\tcol'`, []token{
			{kind: tokenString, value: "col\tcol"},
		})
	})

	t.Run("string with backslash escape carriage return", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, `'cr\rhere'`, []token{
			{kind: tokenString, value: "cr\rhere"},
		})
	})

	t.Run("string with backslash escape null", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, `'null\0byte'`, []token{
			{kind: tokenString, value: "null\x00byte"},
		})
	})

	t.Run("string with backslash escape substitute character", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, `'sub\Z'`, []token{
			{kind: tokenString, value: "sub\x1A"},
		})
	})

	t.Run("string with backslash escaped backslash", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, `'back\\slash'`, []token{
			{kind: tokenString, value: `back\slash`},
		})
	})

	t.Run("string with backslash escaped single quote", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, `'it\'s'`, []token{
			{kind: tokenString, value: "it's"},
		})
	})

	t.Run("hex string uppercase prefix", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "X'DEADBEEF'", []token{
			{kind: tokenHexString, value: "DEADBEEF"},
		})
	})

	t.Run("hex string lowercase prefix", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "x'ab01'", []token{
			{kind: tokenHexString, value: "ab01"},
		})
	})

	t.Run("empty hex string", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "X''", []token{
			{kind: tokenHexString, value: ""},
		})
	})

	t.Run("bit string uppercase prefix", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "B'1010'", []token{
			{kind: tokenBitString, value: "1010"},
		})
	})

	t.Run("bit string lowercase prefix", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "b'110'", []token{
			{kind: tokenBitString, value: "110"},
		})
	})
}

func TestTokenise_Parameters(t *testing.T) {
	t.Parallel()

	t.Run("question mark parameter", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "?", []token{
			{kind: tokenQuestionMark, value: "?"},
		})
	})

	t.Run("named parameter with colon prefix", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, ":name", []token{
			{kind: tokenNamedParam, value: ":name"},
		})
	})

	t.Run("user variable", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "@myvar", []token{
			{kind: tokenUserVariable, value: "@myvar"},
		})
	})

	t.Run("user variable with underscores and digits", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "@user_id_1", []token{
			{kind: tokenUserVariable, value: "@user_id_1"},
		})
	})

	t.Run("system variable with global qualifier", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "@@global.max_connections", []token{
			{kind: tokenSystemVariable, value: "@@global.max_connections"},
		})
	})

	t.Run("system variable with session qualifier", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "@@session.wait_timeout", []token{
			{kind: tokenSystemVariable, value: "@@session.wait_timeout"},
		})
	})

	t.Run("system variable with local qualifier", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "@@local.sort_buffer_size", []token{
			{kind: tokenSystemVariable, value: "@@local.sort_buffer_size"},
		})
	})

	t.Run("system variable without qualifier", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "@@version", []token{
			{kind: tokenSystemVariable, value: "@@version"},
		})
	})

	t.Run("colon without following identifier is operator", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, ":", []token{
			{kind: tokenOperator, value: ":"},
		})
	})

	t.Run("multiple question mark parameters", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "?, ?", []token{
			{kind: tokenQuestionMark, value: "?"},
			{kind: tokenComma, value: ","},
			{kind: tokenQuestionMark, value: "?"},
		})
	})
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
				{kind: tokenOperator, value: "<="},
			},
		},
		{
			name:  "greater than or equal",
			input: ">=",
			expected: []token{
				{kind: tokenOperator, value: ">="},
			},
		},
		{
			name:  "not equal (angle brackets)",
			input: "<>",
			expected: []token{
				{kind: tokenOperator, value: "<>"},
			},
		},
		{
			name:  "not equal (exclamation)",
			input: "!=",
			expected: []token{
				{kind: tokenOperator, value: "!="},
			},
		},
		{
			name:  "null-safe equality",
			input: "<=>",
			expected: []token{
				{kind: tokenOperator, value: "<=>"},
			},
		},
		{
			name:  "left shift",
			input: "<<",
			expected: []token{
				{kind: tokenOperator, value: "<<"},
			},
		},
		{
			name:  "right shift",
			input: ">>",
			expected: []token{
				{kind: tokenOperator, value: ">>"},
			},
		},
		{
			name:  "arrow operator",
			input: "->",
			expected: []token{
				{kind: tokenArrow, value: "->"},
			},
		},
		{
			name:  "double arrow operator",
			input: "->>",
			expected: []token{
				{kind: tokenDoubleArrow, value: "->>"},
			},
		},
		{
			name:  "logical or (double pipe)",
			input: "||",
			expected: []token{
				{kind: tokenOperator, value: "||"},
			},
		},
		{
			name:  "equals",
			input: "=",
			expected: []token{
				{kind: tokenOperator, value: "="},
			},
		},
		{
			name:  "less than",
			input: "<",
			expected: []token{
				{kind: tokenOperator, value: "<"},
			},
		},
		{
			name:  "greater than",
			input: ">",
			expected: []token{
				{kind: tokenOperator, value: ">"},
			},
		},
		{
			name:  "plus",
			input: "+",
			expected: []token{
				{kind: tokenOperator, value: "+"},
			},
		},
		{
			name:  "minus",
			input: "-",
			expected: []token{
				{kind: tokenOperator, value: "-"},
			},
		},
		{
			name:  "forward slash (division)",
			input: "/",
			expected: []token{
				{kind: tokenOperator, value: "/"},
			},
		},
		{
			name:  "percent (modulo)",
			input: "%",
			expected: []token{
				{kind: tokenOperator, value: "%"},
			},
		},
		{
			name:  "tilde (bitwise not)",
			input: "~",
			expected: []token{
				{kind: tokenOperator, value: "~"},
			},
		},
		{
			name:  "ampersand (bitwise and)",
			input: "&",
			expected: []token{
				{kind: tokenOperator, value: "&"},
			},
		},
		{
			name:  "pipe (bitwise or)",
			input: "|",
			expected: []token{
				{kind: tokenOperator, value: "|"},
			},
		},
		{
			name:  "exclamation (logical not)",
			input: "!",
			expected: []token{
				{kind: tokenOperator, value: "!"},
			},
		},
		{
			name:  "caret (bitwise xor)",
			input: "^",
			expected: []token{
				{kind: tokenOperator, value: "^"},
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

	t.Run("spaces between tokens are skipped", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "  a   b  ", []token{
			{kind: tokenIdentifier, value: "a"},
			{kind: tokenIdentifier, value: "b"},
		})
	})

	t.Run("tabs and newlines are skipped", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "a\t\nb", []token{
			{kind: tokenIdentifier, value: "a"},
			{kind: tokenIdentifier, value: "b"},
		})
	})

	t.Run("carriage return is skipped", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "a\r\nb", []token{
			{kind: tokenIdentifier, value: "a"},
			{kind: tokenIdentifier, value: "b"},
		})
	})

	t.Run("line comment with double dash and space", func(t *testing.T) {
		t.Parallel()

		requireTokens(t, "a -- this is a comment\nb", []token{
			{kind: tokenIdentifier, value: "a"},
			{kind: tokenIdentifier, value: "b"},
		})
	})

	t.Run("line comment with double dash and tab", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "a --\tcomment here\nb", []token{
			{kind: tokenIdentifier, value: "a"},
			{kind: tokenIdentifier, value: "b"},
		})
	})

	t.Run("double dash without space is not a comment", func(t *testing.T) {
		t.Parallel()

		requireTokens(t, "a --b", []token{
			{kind: tokenIdentifier, value: "a"},
			{kind: tokenOperator, value: "-"},
			{kind: tokenOperator, value: "-"},
			{kind: tokenIdentifier, value: "b"},
		})
	})

	t.Run("hash line comment", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "a # comment\nb", []token{
			{kind: tokenIdentifier, value: "a"},
			{kind: tokenIdentifier, value: "b"},
		})
	})

	t.Run("hash comment at end of input", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "a # trailing comment", []token{
			{kind: tokenIdentifier, value: "a"},
		})
	})

	t.Run("block comment", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "a /* block */ b", []token{
			{kind: tokenIdentifier, value: "a"},
			{kind: tokenIdentifier, value: "b"},
		})
	})

	t.Run("block comment spanning multiple lines", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "a /* line1\nline2 */ b", []token{
			{kind: tokenIdentifier, value: "a"},
			{kind: tokenIdentifier, value: "b"},
		})
	})

	t.Run("unterminated block comment consumes to end", func(t *testing.T) {
		t.Parallel()

		requireTokens(t, "a /* never closed", []token{
			{kind: tokenIdentifier, value: "a"},
		})
	})

	t.Run("empty input yields no tokens", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "", []token{})
	})

	t.Run("whitespace-only input yields no tokens", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "   \t\n\r  ", []token{})
	})

	t.Run("consecutive comments of different styles", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "-- comment\n# another\n/* block */\na", []token{

			{kind: tokenIdentifier, value: "a"},
		})
	})
}

func TestTokenise_Errors(t *testing.T) {
	t.Parallel()

	t.Run("unterminated single-quoted string", func(t *testing.T) {
		t.Parallel()
		requireTokeniseError(t, "'hello", "unterminated string literal")
	})

	t.Run("unterminated string with escaped quote at end", func(t *testing.T) {
		t.Parallel()
		requireTokeniseError(t, `'hello\'`, "unterminated string literal")
	})

	t.Run("unterminated backtick identifier", func(t *testing.T) {
		t.Parallel()
		requireTokeniseError(t, "`column", "unterminated quoted identifier")
	})

	t.Run("unterminated double-quoted identifier", func(t *testing.T) {
		t.Parallel()
		requireTokeniseError(t, `"column`, "unterminated quoted identifier")
	})

	t.Run("unterminated hex string", func(t *testing.T) {
		t.Parallel()
		requireTokeniseError(t, "X'DEAD", "unterminated hex string")
	})

	t.Run("unterminated bit string", func(t *testing.T) {
		t.Parallel()
		requireTokeniseError(t, "B'101", "unterminated bit string")
	})

	t.Run("unexpected character", func(t *testing.T) {
		t.Parallel()
		requireTokeniseError(t, "$", "unexpected character")
	})
}

func TestTokenise_CompleteStatements(t *testing.T) {
	t.Parallel()

	t.Run("simple select with question mark parameter", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "SELECT id FROM users WHERE id = ?", []token{
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenIdentifier, value: "id"},
			{kind: tokenIdentifier, value: "FROM"},
			{kind: tokenIdentifier, value: "users"},
			{kind: tokenIdentifier, value: "WHERE"},
			{kind: tokenIdentifier, value: "id"},
			{kind: tokenOperator, value: "="},
			{kind: tokenQuestionMark, value: "?"},
		})
	})

	t.Run("insert with multiple question mark parameters", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "INSERT INTO foo VALUES (?, ?)", []token{
			{kind: tokenIdentifier, value: "INSERT"},
			{kind: tokenIdentifier, value: "INTO"},
			{kind: tokenIdentifier, value: "foo"},
			{kind: tokenIdentifier, value: "VALUES"},
			{kind: tokenLeftParen, value: "("},
			{kind: tokenQuestionMark, value: "?"},
			{kind: tokenComma, value: ","},
			{kind: tokenQuestionMark, value: "?"},
			{kind: tokenRightParen, value: ")"},
		})
	})

	t.Run("multi-statement input", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "SELECT 1; SELECT 2", []token{
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenNumber, value: "1"},
			{kind: tokenSemicolon, value: ";"},
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenNumber, value: "2"},
		})
	})

	t.Run("select star with backtick-quoted table", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "SELECT * FROM `order`", []token{
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenStar, value: "*"},
			{kind: tokenIdentifier, value: "FROM"},
			{kind: tokenIdentifier, value: "order"},
		})
	})

	t.Run("select with user variable assignment", func(t *testing.T) {
		t.Parallel()

		requireTokens(t, "SELECT @total := COUNT(*) FROM items", []token{
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenUserVariable, value: "@total"},
			{kind: tokenOperator, value: ":"},
			{kind: tokenOperator, value: "="},
			{kind: tokenIdentifier, value: "COUNT"},
			{kind: tokenLeftParen, value: "("},
			{kind: tokenStar, value: "*"},
			{kind: tokenRightParen, value: ")"},
			{kind: tokenIdentifier, value: "FROM"},
			{kind: tokenIdentifier, value: "items"},
		})
	})

	t.Run("JSON extraction with arrow operators", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "SELECT data->'$.name', data->>'$.email' FROM profiles", []token{
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenIdentifier, value: "data"},
			{kind: tokenArrow, value: "->"},
			{kind: tokenString, value: "$.name"},
			{kind: tokenComma, value: ","},
			{kind: tokenIdentifier, value: "data"},
			{kind: tokenDoubleArrow, value: "->>"},
			{kind: tokenString, value: "$.email"},
			{kind: tokenIdentifier, value: "FROM"},
			{kind: tokenIdentifier, value: "profiles"},
		})
	})

	t.Run("null-safe equality in where clause", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "SELECT * FROM t WHERE a <=> b", []token{
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenStar, value: "*"},
			{kind: tokenIdentifier, value: "FROM"},
			{kind: tokenIdentifier, value: "t"},
			{kind: tokenIdentifier, value: "WHERE"},
			{kind: tokenIdentifier, value: "a"},
			{kind: tokenOperator, value: "<=>"},
			{kind: tokenIdentifier, value: "b"},
		})
	})

	t.Run("select with system variable", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "SELECT @@global.max_connections", []token{
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenSystemVariable, value: "@@global.max_connections"},
		})
	})

	t.Run("insert with hex string value", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "INSERT INTO blobs (data) VALUES (X'FF00')", []token{
			{kind: tokenIdentifier, value: "INSERT"},
			{kind: tokenIdentifier, value: "INTO"},
			{kind: tokenIdentifier, value: "blobs"},
			{kind: tokenLeftParen, value: "("},
			{kind: tokenIdentifier, value: "data"},
			{kind: tokenRightParen, value: ")"},
			{kind: tokenIdentifier, value: "VALUES"},
			{kind: tokenLeftParen, value: "("},
			{kind: tokenHexString, value: "FF00"},
			{kind: tokenRightParen, value: ")"},
		})
	})

	t.Run("statement with inline comment", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "SELECT /* hint */ id FROM users", []token{
			{kind: tokenIdentifier, value: "SELECT"},
			{kind: tokenIdentifier, value: "id"},
			{kind: tokenIdentifier, value: "FROM"},
			{kind: tokenIdentifier, value: "users"},
		})
	})

	t.Run("create table with various column types", func(t *testing.T) {
		t.Parallel()
		requireTokens(t, "CREATE TABLE `t` (`id` INT, `name` VARCHAR(255))", []token{
			{kind: tokenIdentifier, value: "CREATE"},
			{kind: tokenIdentifier, value: "TABLE"},
			{kind: tokenIdentifier, value: "t"},
			{kind: tokenLeftParen, value: "("},
			{kind: tokenIdentifier, value: "id"},
			{kind: tokenIdentifier, value: "INT"},
			{kind: tokenComma, value: ","},
			{kind: tokenIdentifier, value: "name"},
			{kind: tokenIdentifier, value: "VARCHAR"},
			{kind: tokenLeftParen, value: "("},
			{kind: tokenNumber, value: "255"},
			{kind: tokenRightParen, value: ")"},
			{kind: tokenRightParen, value: ")"},
		})
	})
}
