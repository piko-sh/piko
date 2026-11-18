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

package runeutil

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		want     string
		maxRunes int
	}{
		{name: "ascii that fits is unchanged", input: "hello", maxRunes: 10, want: "hello"},
		{name: "ascii at the limit is unchanged", input: "hello", maxRunes: 5, want: "hello"},
		{name: "ascii over the limit is cut", input: "hello world", maxRunes: 5, want: "hello"},
		{name: "cyrillic is cut by runes", input: "ПРИВЕТ", maxRunes: 3, want: "ПРИ"},
		{name: "accented latin is cut by runes", input: "éléphant", maxRunes: 3, want: "élé"},
		{name: "emoji is kept whole", input: "abc😀def", maxRunes: 4, want: "abc😀"},
		{name: "cut lands on a rune boundary", input: "ab中cd", maxRunes: 3, want: "ab中"},
		{name: "multi-byte text that fits by runes is unchanged", input: "中中中", maxRunes: 3, want: "中中中"},
		{name: "zero gives an empty string", input: "ПРИВЕТ", maxRunes: 0, want: ""},
		{name: "negative gives an empty string", input: "ПРИВЕТ", maxRunes: -1, want: ""},
		{name: "empty input stays empty", input: "", maxRunes: 5, want: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := Truncate(testCase.input, testCase.maxRunes)

			assert.Equal(t, testCase.want, got)
			assert.True(t, utf8.ValidString(got))
		})
	}
}

func TestTruncate_LongMultiByteInput(t *testing.T) {
	t.Parallel()

	input := "https://example.com/path?q=" + strings.Repeat("\U0001F600", 1000)

	got := Truncate(input, 50)

	assert.True(t, utf8.ValidString(got))
	assert.Equal(t, 50, utf8.RuneCountInString(got))
}

func TestTruncateWithEllipsis(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		want     string
		maxRunes int
	}{
		{name: "short ascii is unchanged", input: "hello", maxRunes: 10, want: "hello"},
		{name: "ascii at the limit is unchanged", input: "abcdef", maxRunes: 6, want: "abcdef"},
		{name: "ascii over the limit ends with an ellipsis", input: "abcdefghij", maxRunes: 7, want: "abcd..."},
		{name: "a limit too small for the ellipsis cuts without one", input: "abcdef", maxRunes: 2, want: "ab"},
		{name: "a limit equal to the ellipsis cuts without one", input: "abcdef", maxRunes: 3, want: "abc"},
		{name: "zero gives an empty string", input: "anything", maxRunes: 0, want: ""},
		{name: "negative gives an empty string", input: "anything", maxRunes: -3, want: ""},
		{name: "multi-byte runes are kept whole", input: strings.Repeat("é", 50), maxRunes: 10, want: strings.Repeat("é", 7) + "..."},
		{name: "long multi-byte input is cut by runes", input: strings.Repeat("文", 200), maxRunes: 50, want: strings.Repeat("文", 47) + "..."},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := TruncateWithEllipsis(testCase.input, testCase.maxRunes)

			assert.Equal(t, testCase.want, got)
			assert.True(t, utf8.ValidString(got))
		})
	}
}

func TestTruncate_InvalidUTF8IsReplaced(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
		maxRunes int
	}{
		{name: "invalid byte that fits", input: "ab\xffc", maxRunes: 10, expected: "ab�c"},
		{name: "invalid byte inside the cut", input: "ab\xffcdef", maxRunes: 4, expected: "ab�c"},
		{name: "invalid byte after the cut", input: "abc\xff", maxRunes: 3, expected: "abc"},
		{name: "each invalid byte counts as one rune", input: "\xff\xfe\xfd", maxRunes: 2, expected: "��"},
		{name: "cut multi-byte sequence", input: "a\xe2\x82", maxRunes: 5, expected: "a��"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := Truncate(testCase.input, testCase.maxRunes)

			assert.Equal(t, testCase.expected, got)
			assert.True(t, utf8.ValidString(got))
		})
	}
}

func TestTruncateWithEllipsis_InvalidUTF8IsReplaced(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
		maxRunes int
	}{
		{name: "invalid byte that fits", input: "ab\xff", maxRunes: 10, expected: "ab�"},
		{name: "invalid byte inside the cut", input: "a\xffcdefgh", maxRunes: 5, expected: "a�..."},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := TruncateWithEllipsis(testCase.input, testCase.maxRunes)

			assert.Equal(t, testCase.expected, got)
			assert.True(t, utf8.ValidString(got))
		})
	}
}

func TestTruncateBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
		maxBytes int
	}{
		{name: "under limit", input: "hello", maxBytes: 10, expected: "hello"},
		{name: "at limit", input: "hello", maxBytes: 5, expected: "hello"},
		{name: "over limit", input: "hello world", maxBytes: 5, expected: "hello"},
		{name: "empty", input: "", maxBytes: 10, expected: ""},
		{name: "zero limit", input: "hello", maxBytes: 0, expected: ""},
		{name: "negative limit", input: "hello", maxBytes: -1, expected: ""},
		{name: "cut inside a rune drops it", input: "世", maxBytes: 2, expected: ""},
		{name: "cut after a whole rune keeps it", input: "a世b", maxBytes: 4, expected: "a世"},
		{name: "cut inside the second rune", input: "世界", maxBytes: 5, expected: "世"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := TruncateBytes(testCase.input, testCase.maxBytes)

			assert.Equal(t, testCase.expected, got)
			assert.LessOrEqual(t, len(got), max(testCase.maxBytes, 0))
			assert.True(t, utf8.ValidString(got))
		})
	}
}
