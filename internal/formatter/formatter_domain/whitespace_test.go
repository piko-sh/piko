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

package formatter_domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormaliseWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty string", input: "", expected: ""},
		{name: "single word", input: "Hello", expected: "Hello"},
		{name: "two words single space", input: "Hello World", expected: "Hello World"},
		{name: "double space", input: "Hello  World", expected: "Hello World"},
		{name: "triple space", input: "Hello   World", expected: "Hello World"},
		{name: "many spaces", input: "Hello          World", expected: "Hello World"},
		{name: "leading space", input: " Hello", expected: " Hello"},
		{name: "trailing space", input: "Hello ", expected: "Hello "},
		{name: "leading and trailing", input: " Hello ", expected: " Hello "},
		{name: "multiple leading", input: "    Hello", expected: " Hello"},
		{name: "multiple trailing", input: "Hello    ", expected: "Hello "},
		{name: "multiple both", input: "    Hello    ", expected: " Hello "},
		{name: "single newline", input: "Hello\nWorld", expected: "Hello World"},
		{name: "multiple newlines", input: "Hello\n\n\nWorld", expected: "Hello World"},
		{name: "newline with spaces", input: "Hello  \n  World", expected: "Hello World"},
		{name: "windows newline", input: "Hello\r\nWorld", expected: "Hello World"},
		{name: "mac newline", input: "Hello\rWorld", expected: "Hello World"},
		{name: "single tab", input: "Hello\tWorld", expected: "Hello World"},
		{name: "multiple tabs", input: "Hello\t\t\tWorld", expected: "Hello World"},
		{name: "mixed tabs and spaces", input: "Hello \t \t World", expected: "Hello World"},
		{name: "space newline space", input: "Hello \n World", expected: "Hello World"},
		{name: "tab newline tab", input: "Hello\t\n\tWorld", expected: "Hello World"},
		{name: "complex mix", input: "Hello   \n\t  \r\n   World", expected: "Hello World"},
		{name: "three words", input: "One  Two  Three", expected: "One Two Three"},
		{name: "many words with mixed whitespace", input: "A\t\nB  \r\nC   D", expected: "A B C D"},
		{name: "only spaces", input: "     ", expected: " "},
		{name: "only newlines", input: "\n\n\n", expected: " "},
		{name: "only tabs", input: "\t\t\t", expected: " "},
		{name: "mixed whitespace only", input: " \n\t\r\n ", expected: " "},
		{
			name:     "paragraph with line breaks",
			input:    "First paragraph\n\nSecond paragraph",
			expected: "First paragraph Second paragraph",
		},
		{
			name:     "indented code",
			input:    "    function hello() {\n        return 'world';\n    }",
			expected: " function hello() { return 'world'; }",
		},
		{
			name:     "HTML with newlines",
			input:    "Hello <strong>bold</strong> text",
			expected: "Hello <strong>bold</strong> text",
		},
		{
			name:     "text with extra spacing around tags",
			input:    "Text   <strong>   bold   </strong>   more",
			expected: "Text <strong> bold </strong> more",
		},
		{name: "single space", input: " ", expected: " "},
		{name: "single character", input: "A", expected: "A"},
		{name: "two characters", input: "AB", expected: "AB"},
		{name: "two characters with space", input: "A B", expected: "A B"},
		{name: "unicode characters", input: "Hello　World", expected: "Hello　World"},
		{name: "emoji with spaces", input: "Hello  👋  World", expected: "Hello 👋 World"},
		{name: "form label", input: "First Name:    ", expected: "First Name: "},
		{name: "button text", input: "  Click  Here  ", expected: " Click Here "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normaliseWhitespace(tt.input)
			assert.Equal(t, tt.expected, result,
				"normaliseWhitespace(%q) should produce %q, got %q",
				tt.input, tt.expected, result)
		})
	}
}

func TestNormaliseWhitespace_Properties(t *testing.T) {
	testInputs := []string{
		"Hello    World",
		"  Multiple   spaces  ",
		"Tabs\t\tand\t\tnewlines\n\n",
		"",
		" ",
		"SingleWord",
		strings.Repeat(" ", 100),
		strings.Repeat("Word  ", 50),
	}

	for _, input := range testInputs {
		t.Run("input: "+input, func(t *testing.T) {
			result := normaliseWhitespace(input)

			assert.LessOrEqual(t, len(result), len(input),
				"Normalised string should not be longer than input")

			assert.NotContains(t, result, "  ",
				"Result should not contain consecutive spaces")

			assert.NotContains(t, result, "\t",
				"Result should not contain tabs")

			assert.NotContains(t, result, "\n",
				"Result should not contain newlines")
			assert.NotContains(t, result, "\r",
				"Result should not contain carriage returns")

			doubleNormalised := normaliseWhitespace(result)
			assert.Equal(t, result, doubleNormalised,
				"Normalisation should be idempotent")
		})
	}
}
