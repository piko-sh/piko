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

package jsscan

import (
	"testing"

	"github.com/stretchr/testify/require"
	parsejs "github.com/tdewolff/parse/v2/js"
)

type scannedToken struct {
	text  string
	start int
}

func TestScanner_ReadsRegExpLiteralsWhole(t *testing.T) {
	t.Parallel()

	scanner := New(`if (!/^[a-z][a-z0-9_]{0,31}$/.test(name)) text.replace(/\//g, '_');`)

	var literals []string
	for {
		tt, data := scanner.Next()
		if tt == parsejs.ErrorToken {
			break
		}
		if tt == parsejs.RegExpToken {
			literals = append(literals, string(data))
		}
	}

	require.Equal(t, []string{`/^[a-z][a-z0-9_]{0,31}$/`, `/\//g`}, literals)
	require.Zero(t, scanner.BraceDepth())
}

func TestScanner_TokenStart(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   []scannedToken
	}{
		{
			name:   "tokens after whitespace and comments",
			source: "a /* note */ . b // end\n c",
			want:   []scannedToken{{"a", 0}, {".", 13}, {"b", 15}, {"c", 25}},
		},
		{
			name:   "regular expression literal starts at its slash",
			source: "x = /a+/g",
			want:   []scannedToken{{"x", 0}, {"=", 2}, {"/a+/g", 4}},
		},
		{
			name:   "string literal keeps its quotes",
			source: `f('é', "b")`,
			want:   []scannedToken{{"f", 0}, {"(", 1}, {"'é'", 2}, {",", 6}, {`"b"`, 8}, {")", 11}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.want, scanAll(testCase.source))
		})
	}
}

func TestScanner_BraceDepth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   []int
	}{
		{name: "nested braces", source: "{ { } }", want: []int{1, 2, 1, 0}},
		{name: "stray close brace stays at zero", source: "} {", want: []int{0, 1}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scanner := New(testCase.source)
			var depths []int
			for {
				tt, _ := scanner.Next()
				if tt == parsejs.ErrorToken {
					break
				}
				depths = append(depths, scanner.BraceDepth())
			}
			require.Equal(t, testCase.want, depths)
		})
	}
}

func scanAll(source string) []scannedToken {
	scanner := New(source)
	var tokens []scannedToken
	for {
		tt, data := scanner.Next()
		if tt == parsejs.ErrorToken {
			return tokens
		}
		tokens = append(tokens, scannedToken{text: string(data), start: scanner.TokenStart()})
	}
}
