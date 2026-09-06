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

package lsp_domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdownCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain text", input: "handleClick", want: "`handleClick`"},
		{name: "empty", input: "", want: "``"},
		{name: "inner backquote", input: "a`b", want: "``a`b``"},
		{name: "template literal", input: "`hi ${name}`", want: "`` `hi ${name}` ``"},
		{name: "longest run wins", input: "a``b`c", want: "```a``b`c```"},
		{name: "line feed becomes a space", input: "a\nb", want: "`a b`"},
		{name: "blank line cannot end the span", input: "a\n\nb", want: "`a b`"},
		{name: "carriage return line feed", input: "a\r\nb\rc", want: "`a b c`"},
		{name: "indentation after a break is kept", input: "{\n  a: 1\n}", want: "`{   a: 1 }`"},
		{name: "text at the limit is kept whole", input: strings.Repeat("x", maxMarkdownCodeRunes), want: "`" + strings.Repeat("x", maxMarkdownCodeRunes) + "`"},
		{name: "long text is cut with an ellipsis", input: strings.Repeat("x", maxMarkdownCodeRunes+1), want: "`" + strings.Repeat("x", maxMarkdownCodeRunes) + markdownCodeEllipsis + "`"},
		{name: "multi-byte text is cut by runes", input: strings.Repeat("é", maxMarkdownCodeRunes+5), want: "`" + strings.Repeat("é", maxMarkdownCodeRunes) + markdownCodeEllipsis + "`"},
		{name: "fence follows backquotes left after truncation", input: strings.Repeat("x", maxMarkdownCodeRunes-2) + "``" + strings.Repeat("`", 10), want: "```" + strings.Repeat("x", maxMarkdownCodeRunes-2) + "``" + markdownCodeEllipsis + "```"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, markdownCode(testCase.input))
		})
	}
}

func TestTruncateRunes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
		limit int
	}{
		{name: "empty text", input: "", limit: 3, want: ""},
		{name: "shorter than the limit", input: "ab", limit: 3, want: "ab"},
		{name: "exactly the limit", input: "abc", limit: 3, want: "abc"},
		{name: "one over the limit", input: "abcd", limit: 3, want: "abc" + markdownCodeEllipsis},
		{name: "zero limit", input: "a", limit: 0, want: markdownCodeEllipsis},
		{name: "multi-byte runes", input: "日本語テキスト", limit: 3, want: "日本語" + markdownCodeEllipsis},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, truncateRunes(testCase.input, testCase.limit))
		})
	}
}
