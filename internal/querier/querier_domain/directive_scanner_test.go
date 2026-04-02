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

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestDirectiveLineScanner_AtEnd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		position int
		expected bool
	}{
		{
			name:     "empty string is immediately at end",
			line:     "",
			position: 0,
			expected: true,
		},
		{
			name:     "non-empty string at start is not at end",
			line:     "hello",
			position: 0,
			expected: false,
		},
		{
			name:     "after advancing past end returns true",
			line:     "ab",
			position: 2,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := newDirectiveLineScanner(tt.line, 1)
			scanner.position = tt.position

			assert.Equal(t, tt.expected, scanner.atEnd())
		})
	}
}

func TestDirectiveLineScanner_Current(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		position int
		expected byte
	}{
		{
			name:     "returns byte at current position",
			line:     "abc",
			position: 1,
			expected: 'b',
		},
		{
			name:     "returns zero at end of line",
			line:     "abc",
			position: 3,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := newDirectiveLineScanner(tt.line, 1)
			scanner.position = tt.position

			assert.Equal(t, tt.expected, scanner.current())
		})
	}
}

func TestDirectiveLineScanner_Advance(t *testing.T) {
	t.Parallel()

	scanner := newDirectiveLineScanner("xyz", 1)

	got := scanner.advance()

	assert.Equal(t, byte('x'), got, "advance should return the current byte")
	assert.Equal(t, 1, scanner.position, "position should be incremented after advance")
}

func TestDirectiveLineScanner_Column(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		position       int
		expectedColumn int
	}{
		{
			name:           "column is position plus one at start",
			position:       0,
			expectedColumn: 1,
		},
		{
			name:           "column is position plus one after advancing",
			position:       4,
			expectedColumn: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := newDirectiveLineScanner("hello", 1)
			scanner.position = tt.position

			assert.Equal(t, tt.expectedColumn, scanner.column())
		})
	}
}

func TestDirectiveLineScanner_SkipWhitespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		line             string
		expectedPosition int
	}{
		{
			name:             "skips spaces only",
			line:             "   abc",
			expectedPosition: 3,
		},
		{
			name:             "skips tabs only",
			line:             "\t\tabc",
			expectedPosition: 2,
		},
		{
			name:             "skips mixed spaces and tabs",
			line:             " \t \tabc",
			expectedPosition: 4,
		},
		{
			name:             "no whitespace leaves position unchanged",
			line:             "abc",
			expectedPosition: 0,
		},
		{
			name:             "leading non-whitespace character leaves position unchanged",
			line:             "x  abc",
			expectedPosition: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := newDirectiveLineScanner(tt.line, 1)
			scanner.skipWhitespace()

			assert.Equal(t, tt.expectedPosition, scanner.position)
		})
	}
}

func TestDirectiveLineScanner_SpanFrom(t *testing.T) {
	t.Parallel()

	scanner := newDirectiveLineScanner("hello world", 5)
	scanner.position = 7

	span := scanner.spanFrom(3)

	expected := querier_dto.TextSpan{
		Line:      5,
		Column:    3,
		EndLine:   5,
		EndColumn: 8,
	}

	assert.Equal(t, expected, span)
}

func TestDirectiveLineScanner_MatchString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		line             string
		match            string
		expectedResult   bool
		expectedPosition int
	}{
		{
			name:             "exact match advances position and returns true",
			line:             "hello",
			match:            "hello",
			expectedResult:   true,
			expectedPosition: 5,
		},
		{
			name:             "prefix match advances position and returns true",
			line:             "hello world",
			match:            "hello",
			expectedResult:   true,
			expectedPosition: 5,
		},
		{
			name:             "mismatch does not advance and returns false",
			line:             "hello",
			match:            "world",
			expectedResult:   false,
			expectedPosition: 0,
		},
		{
			name:             "insufficient length returns false without advancing",
			line:             "hi",
			match:            "hello",
			expectedResult:   false,
			expectedPosition: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := newDirectiveLineScanner(tt.line, 1)
			result := scanner.matchString(tt.match)

			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.expectedPosition, scanner.position)
		})
	}
}

func TestDirectiveLineScanner_MatchKeyword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		line             string
		keyword          string
		expectedResult   bool
		expectedPosition int
	}{
		{
			name:             "case-insensitive match advances and returns true",
			line:             "SELECT * FROM users",
			keyword:          "select",
			expectedResult:   true,
			expectedPosition: 6,
		},
		{
			name:             "word boundary prevents partial match",
			line:             "SELECTED",
			keyword:          "SEL",
			expectedResult:   false,
			expectedPosition: 0,
		},
		{
			name:             "keyword at end of line matches",
			line:             "SELECT",
			keyword:          "select",
			expectedResult:   true,
			expectedPosition: 6,
		},
		{
			name:             "mismatch returns false without advancing",
			line:             "INSERT INTO",
			keyword:          "select",
			expectedResult:   false,
			expectedPosition: 0,
		},
		{
			name:             "keyword followed by space matches at word boundary",
			line:             "FROM users",
			keyword:          "from",
			expectedResult:   true,
			expectedPosition: 4,
		},
		{
			name:             "insufficient length returns false",
			line:             "SE",
			keyword:          "SELECT",
			expectedResult:   false,
			expectedPosition: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := newDirectiveLineScanner(tt.line, 1)
			result := scanner.matchKeyword(tt.keyword)

			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.expectedPosition, scanner.position)
		})
	}
}

func TestDirectiveLineScanner_MatchByte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		line             string
		matchByte        byte
		expectedResult   bool
		expectedPosition int
	}{
		{
			name:             "matching byte advances and returns true",
			line:             "(hello)",
			matchByte:        '(',
			expectedResult:   true,
			expectedPosition: 1,
		},
		{
			name:             "mismatched byte does not advance and returns false",
			line:             "hello",
			matchByte:        '(',
			expectedResult:   false,
			expectedPosition: 0,
		},
		{
			name:             "at end of line returns false",
			line:             "",
			matchByte:        '(',
			expectedResult:   false,
			expectedPosition: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := newDirectiveLineScanner(tt.line, 1)
			result := scanner.matchByte(tt.matchByte)

			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.expectedPosition, scanner.position)
		})
	}
}

func TestDirectiveLineScanner_LookingAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		line             string
		prefix           string
		expectedResult   bool
		expectedPosition int
	}{
		{
			name:             "prefix present returns true without advancing",
			line:             "hello world",
			prefix:           "hello",
			expectedResult:   true,
			expectedPosition: 0,
		},
		{
			name:             "prefix absent returns false",
			line:             "hello world",
			prefix:           "world",
			expectedResult:   false,
			expectedPosition: 0,
		},
		{
			name:             "insufficient length returns false",
			line:             "hi",
			prefix:           "hello",
			expectedResult:   false,
			expectedPosition: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := newDirectiveLineScanner(tt.line, 1)
			result := scanner.lookingAt(tt.prefix)

			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.expectedPosition, scanner.position,
				"lookingAt must not advance the position")
		})
	}
}

func TestDirectiveLineScanner_ReadWord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		line             string
		lineNumber       int
		expectedWord     string
		expectedSpan     querier_dto.TextSpan
		expectedPosition int
	}{
		{
			name:         "reads alphanumeric and underscore word",
			line:         "my_var123 rest",
			lineNumber:   3,
			expectedWord: "my_var123",
			expectedSpan: querier_dto.TextSpan{
				Line:      3,
				Column:    1,
				EndLine:   3,
				EndColumn: 10,
			},
			expectedPosition: 9,
		},
		{
			name:         "stops at space",
			line:         "hello world",
			lineNumber:   1,
			expectedWord: "hello",
			expectedSpan: querier_dto.TextSpan{
				Line:      1,
				Column:    1,
				EndLine:   1,
				EndColumn: 6,
			},
			expectedPosition: 5,
		},
		{
			name:         "stops at parenthesis",
			line:         "func(arg)",
			lineNumber:   1,
			expectedWord: "func",
			expectedSpan: querier_dto.TextSpan{
				Line:      1,
				Column:    1,
				EndLine:   1,
				EndColumn: 5,
			},
			expectedPosition: 4,
		},
		{
			name:         "empty word at non-word character",
			line:         "(something)",
			lineNumber:   2,
			expectedWord: "",
			expectedSpan: querier_dto.TextSpan{
				Line:      2,
				Column:    1,
				EndLine:   2,
				EndColumn: 1,
			},
			expectedPosition: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := newDirectiveLineScanner(tt.line, tt.lineNumber)
			word, span := scanner.readWord()

			assert.Equal(t, tt.expectedWord, word)
			assert.Equal(t, tt.expectedSpan, span)
			assert.Equal(t, tt.expectedPosition, scanner.position)
		})
	}
}

func TestDirectiveLineScanner_ReadDigits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		line             string
		lineNumber       int
		expectedDigits   string
		expectedSpan     querier_dto.TextSpan
		expectedPosition int
	}{
		{
			name:           "reads digits only",
			line:           "12345 rest",
			lineNumber:     1,
			expectedDigits: "12345",
			expectedSpan: querier_dto.TextSpan{
				Line:      1,
				Column:    1,
				EndLine:   1,
				EndColumn: 6,
			},
			expectedPosition: 5,
		},
		{
			name:           "stops at non-digit character",
			line:           "42abc",
			lineNumber:     2,
			expectedDigits: "42",
			expectedSpan: querier_dto.TextSpan{
				Line:      2,
				Column:    1,
				EndLine:   2,
				EndColumn: 3,
			},
			expectedPosition: 2,
		},
		{
			name:           "empty digits at non-digit character",
			line:           "abc123",
			lineNumber:     1,
			expectedDigits: "",
			expectedSpan: querier_dto.TextSpan{
				Line:      1,
				Column:    1,
				EndLine:   1,
				EndColumn: 1,
			},
			expectedPosition: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := newDirectiveLineScanner(tt.line, tt.lineNumber)
			digits, span := scanner.readDigits()

			assert.Equal(t, tt.expectedDigits, digits)
			assert.Equal(t, tt.expectedSpan, span)
			assert.Equal(t, tt.expectedPosition, scanner.position)
		})
	}
}

func TestDirectiveLineScanner_ReadUntilByte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		line             string
		delimiter        byte
		lineNumber       int
		expectedText     string
		expectedSpan     querier_dto.TextSpan
		expectedPosition int
	}{
		{
			name:         "reads up to delimiter",
			line:         "key=value",
			delimiter:    '=',
			lineNumber:   1,
			expectedText: "key",
			expectedSpan: querier_dto.TextSpan{
				Line:      1,
				Column:    1,
				EndLine:   1,
				EndColumn: 4,
			},
			expectedPosition: 3,
		},
		{
			name:         "delimiter not found reads to end",
			line:         "no delimiter here",
			delimiter:    '=',
			lineNumber:   1,
			expectedText: "no delimiter here",
			expectedSpan: querier_dto.TextSpan{
				Line:      1,
				Column:    1,
				EndLine:   1,
				EndColumn: 18,
			},
			expectedPosition: 17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := newDirectiveLineScanner(tt.line, tt.lineNumber)
			text, span := scanner.readUntilByte(tt.delimiter)

			assert.Equal(t, tt.expectedText, text)
			assert.Equal(t, tt.expectedSpan, span)
			assert.Equal(t, tt.expectedPosition, scanner.position)
		})
	}
}

func TestDirectiveLineScanner_ReadUntilWhitespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		line             string
		lineNumber       int
		expectedText     string
		expectedSpan     querier_dto.TextSpan
		expectedPosition int
	}{
		{
			name:         "stops at space",
			line:         "hello world",
			lineNumber:   1,
			expectedText: "hello",
			expectedSpan: querier_dto.TextSpan{
				Line:      1,
				Column:    1,
				EndLine:   1,
				EndColumn: 6,
			},
			expectedPosition: 5,
		},
		{
			name:         "stops at tab",
			line:         "hello\tworld",
			lineNumber:   1,
			expectedText: "hello",
			expectedSpan: querier_dto.TextSpan{
				Line:      1,
				Column:    1,
				EndLine:   1,
				EndColumn: 6,
			},
			expectedPosition: 5,
		},
		{
			name:         "no whitespace reads to end",
			line:         "nospaces",
			lineNumber:   2,
			expectedText: "nospaces",
			expectedSpan: querier_dto.TextSpan{
				Line:      2,
				Column:    1,
				EndLine:   2,
				EndColumn: 9,
			},
			expectedPosition: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := newDirectiveLineScanner(tt.line, tt.lineNumber)
			text, span := scanner.readUntilWhitespace()

			assert.Equal(t, tt.expectedText, text)
			assert.Equal(t, tt.expectedSpan, span)
			assert.Equal(t, tt.expectedPosition, scanner.position)
		})
	}
}

func TestDirectiveLineScanner_ReadRemainder(t *testing.T) {
	t.Parallel()

	scanner := newDirectiveLineScanner("hello world", 3)

	scanner.position = 6

	text, span := scanner.readRemainder()

	assert.Equal(t, "world", text)
	assert.Equal(t, querier_dto.TextSpan{
		Line:      3,
		Column:    7,
		EndLine:   3,
		EndColumn: 12,
	}, span)
	assert.True(t, scanner.atEnd(), "scanner should be at end after readRemainder")
}

func TestIsWordCharacter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		character byte
		expected  bool
	}{
		{
			name:      "lowercase letter",
			character: 'a',
			expected:  true,
		},
		{
			name:      "lowercase letter z",
			character: 'z',
			expected:  true,
		},
		{
			name:      "uppercase letter",
			character: 'A',
			expected:  true,
		},
		{
			name:      "uppercase letter Z",
			character: 'Z',
			expected:  true,
		},
		{
			name:      "digit zero",
			character: '0',
			expected:  true,
		},
		{
			name:      "digit nine",
			character: '9',
			expected:  true,
		},
		{
			name:      "underscore",
			character: '_',
			expected:  true,
		},
		{
			name:      "space is not a word character",
			character: ' ',
			expected:  false,
		},
		{
			name:      "hyphen is not a word character",
			character: '-',
			expected:  false,
		},
		{
			name:      "dot is not a word character",
			character: '.',
			expected:  false,
		},
		{
			name:      "open parenthesis is not a word character",
			character: '(',
			expected:  false,
		},
		{
			name:      "at sign is not a word character",
			character: '@',
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, isWordCharacter(tt.character))
		})
	}
}
