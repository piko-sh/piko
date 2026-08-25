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

package jsident

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

type identifierCase struct {
	name     string
	input    string
	expected string
}

var (
	hostileNames = []string{
		"", "_", "__", "-", "--", " ", "   ", ".", "2fa", "2fa_enabled",
		"my-query", "my.query", "my query", "my/query", "type", "range", "error",
		"string", "iota", "class", "await", "café", "日本語", "$dollar", "😀",
		"a\nb", "select *", "user__name", "COUNT(*)", `he said "hi"`, "a,b",
		"it's", "back\\slash", "line sep", "para sep", "\ufeffbom",
		"null\x00byte", "bell\a", "tab\there", "\xed\xa0\x80", "\xff\xfe",
	}
)

func runIdentifierCases(t *testing.T, cases []identifierCase, transform func(string) string) {
	t.Helper()
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, transform(testCase.input))
		})
	}
}

func TestIsReservedWord(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "keyword", input: "class", expected: true},
		{name: "control keyword", input: "return", expected: true},
		{name: "strict mode reserved", input: "implements", expected: true},
		{name: "await", input: "await", expected: true},
		{name: "arguments", input: "arguments", expected: true},
		{name: "eval", input: "eval", expected: true},
		{name: "go keyword is fine in TS", input: "range", expected: false},
		{name: "ordinary name", input: "userName", expected: false},
		{name: "empty string", input: "", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, IsReservedWord(testCase.input))
		})
	}
}

func TestIsValidIdentifier(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "plain name", input: "userName", expected: true},
		{name: "dollar sign is legal in JS", input: "$id", expected: true},
		{name: "leading underscore", input: "_private", expected: true},
		{name: "unicode letters", input: "café", expected: true},
		{name: "go keyword is fine in TS", input: "range", expected: true},
		{name: "reserved word", input: "class", expected: false},
		{name: "new is reserved", input: "new", expected: false},
		{name: "leading digit", input: "2fa", expected: false},
		{name: "hyphen", input: "my-query", expected: false},
		{name: "empty string", input: "", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, IsValidIdentifier(testCase.input))
		})
	}
}

func TestSanitiseIdentifier(t *testing.T) {
	runIdentifierCases(t, []identifierCase{
		{name: "already valid passes through", input: "userName", expected: "userName"},
		{name: "dollar sign kept", input: "$id", expected: "$id"},
		{name: "reserved word suffixed", input: "class", expected: "class_"},
		{name: "new is suffixed", input: "new", expected: "new_"},
		{name: "leading digit prefixed", input: "2fa", expected: "_2fa"},
		{name: "hyphen replaced", input: "my-query", expected: "my_query"},
		{name: "empty string falls back", input: "", expected: DefaultIdentifier},
		{name: "go type name is fine in TS", input: "string", expected: "string"},
	}, SanitiseIdentifier)
}

func TestSanitiseIdentifierDoesNotCollapseDistinctNames(t *testing.T) {
	assert.NotEqual(t, SanitiseIdentifier("2fa"), SanitiseIdentifier("3fa"))
}

func TestQuotePropertyName(t *testing.T) {
	runIdentifierCases(t, []identifierCase{
		{name: "identifier is emitted bare", input: "userName", expected: "userName"},
		{name: "reserved word needs no quoting", input: "class", expected: "class"},
		{name: "hyphen is quoted", input: "user-name", expected: `"user-name"`},
		{name: "space is quoted", input: "user name", expected: `"user name"`},
		{name: "leading digit is quoted", input: "2fa", expected: `"2fa"`},
		{name: "empty string is quoted", input: "", expected: `""`},
	}, QuotePropertyName)
}

func TestQuoteStringLiteral(t *testing.T) {
	runIdentifierCases(t, []identifierCase{
		{name: "plain text", input: "hello", expected: `"hello"`},
		{name: "double quote escaped", input: `say "hi"`, expected: `"say \"hi\""`},
		{name: "backslash escaped", input: `a\b`, expected: `"a\\b"`},
		{name: "newline escaped", input: "a\nb", expected: `"a\nb"`},
		{name: "tab escaped", input: "a\tb", expected: `"a\tb"`},
		{name: "single quote needs no escape", input: "it's", expected: `"it's"`},
		{name: "line separator escaped", input: "a\u2028b", expected: `"a\u2028b"`},
		{name: "paragraph separator escaped", input: "a\u2029b", expected: `"a\u2029b"`},
		{name: "empty string", input: "", expected: `""`},
	}, QuoteStringLiteral)
}

func TestQuoteStringLiteralEscapesLoneSurrogates(t *testing.T) {
	quoted := QuoteStringLiteral("\xed\xa0\x80")

	assert.NotContains(t, quoted, "\xed\xa0\x80")
	assert.Contains(t, strings.ToLower(quoted), `\ud800`)
}

func TestIsPrintableUnescaped(t *testing.T) {
	testCases := []struct {
		name     string
		input    rune
		expected bool
	}{
		{name: "letter", input: 'a', expected: true},
		{name: "digit", input: '7', expected: true},
		{name: "space", input: ' ', expected: true},
		{name: "single quote", input: '\'', expected: true},
		{name: "double quote", input: '"', expected: false},
		{name: "backslash", input: '\\', expected: false},
		{name: "newline", input: '\n', expected: false},
		{name: "null", input: 0, expected: false},
		{name: "line separator", input: '\u2028', expected: false},
		{name: "paragraph separator", input: '\u2029', expected: false},
		{name: "byte order mark", input: '\ufeff', expected: false},
		{name: "surrogate", input: 0xD800, expected: false},
		{name: "astral rune", input: '😀', expected: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, IsPrintableUnescaped(testCase.input))
		})
	}
}

func TestSanitiseIdentifierAlwaysProducesUsableNames(t *testing.T) {
	for _, input := range hostileNames {
		name := SanitiseIdentifier(input)
		assert.True(t, IsValidIdentifier(name), "%q became invalid TS name %q", input, name)
		assert.Equal(t, name, SanitiseIdentifier(name), "input %q", input)
	}
}

func TestQuotingAlwaysProducesTerminatedLiterals(t *testing.T) {
	for _, input := range hostileNames {
		quoted := QuoteStringLiteral(input)

		assert.True(t, utf8.ValidString(quoted), "%q produced invalid UTF-8", input)
		assert.GreaterOrEqual(t, len(quoted), 2, "%q produced %q", input, quoted)
		assert.Equal(t, byte('"'), quoted[0], "%q produced %q", input, quoted)
		assert.Equal(t, byte('"'), quoted[len(quoted)-1], "%q produced %q", input, quoted)
		assert.NotContains(t, quoted[1:len(quoted)-1], "\u2028", "%q leaked a separator", input)
		assert.NotContains(t, quoted[1:len(quoted)-1], "\u2029", "%q leaked a separator", input)

		property := QuotePropertyName(input)
		assert.NotEmpty(t, property, "%q produced an empty property name", input)
	}
}
