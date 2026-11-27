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
)

func TestTokens_StopsAtLimit(t *testing.T) {
	t.Parallel()

	tokens, truncated := Tokens("a.b.c", 2)
	require.True(t, truncated)
	require.Len(t, tokens, 2)

	tokens, truncated = Tokens("a.b", 10)
	require.False(t, truncated)
	require.Equal(t, []Token{{Text: "a", Start: 0, Type: tokens[0].Type}, {Text: ".", Start: 1, Type: tokens[1].Type}, {Text: "b", Start: 2, Type: tokens[2].Type}}, tokens)
}

func TestFindPropertyReads(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		source   string
		want     []string
		assigned []string
	}{
		{name: "dot access", source: "pk.refs.title.textContent = 'x';", want: []string{"title"}},
		{name: "optional chaining", source: "pk.refs?.title?.focus();", want: []string{"title"}},
		{name: "quoted index", source: `pk.refs["title"].focus(); pk.refs?.['body']`, want: []string{"title", "body"}},
		{name: "no substitution template index", source: "pk.refs[`title`]", want: []string{"title"}},
		{name: "keyword spelt property", source: "pk.refs.default", want: []string{"default"}},
		{name: "computed index is skipped", source: "pk.refs['panel' + index]; pk.refs[name]", want: nil},
		{name: "template with substitution is skipped", source: "pk.refs[`panel${index}`]", want: nil},
		{name: "escaped literal is skipped", source: `pk.refs["ti\x74le"]`, want: nil},
		{name: "member of another object is skipped", source: "other.pk.refs.title; other?.pk.refs.body", want: nil},
		{name: "comments and strings are skipped", source: "// pk.refs.one\n/* pk.refs.two */ const s = 'pk.refs.three';", want: nil},
		{name: "regular expression is skipped", source: "const re = /pk.refs.one/; pk.refs.two", want: []string{"two"}},
		{name: "assignment is marked", source: "pk.refs.made = el; pk.refs['other'] = el; pk.refs.read == el", want: []string{"made", "other", "read"}, assigned: []string{"made", "other"}},
		{name: "other paths are ignored", source: "pk.refsX.title; refs.title; pk.other.title", want: nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			tokens, truncated := Tokens(testCase.source, 1000)
			require.False(t, truncated)
			var names, assigned []string
			for _, read := range FindPropertyReads(tokens, "pk", "refs") {
				names = append(names, read.Value)
				require.Equal(t, testCase.source[read.Start:read.Start+read.Length], tokens[indexOfStart(tokens, read.Start)].Text)
				if read.Assigned {
					assigned = append(assigned, read.Value)
				}
			}
			require.Equal(t, testCase.want, names)
			require.Equal(t, testCase.assigned, assigned)
		})
	}
}

func TestFindCallArguments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   []string
	}{
		{name: "single argument", source: `piko.partial("cart")`, want: []string{"cart"}},
		{name: "argument followed by options", source: `piko.partial('cart', {debounce: 50})`, want: []string{"cart"}},
		{name: "computed argument is skipped", source: `piko.partial("cart-" + id); piko.partial(name)`, want: nil},
		{name: "literal inside a nested call is skipped", source: `piko.partial(pick("cart"))`, want: nil},
		{name: "other functions are ignored", source: `piko.partials("cart"); window.piko.partial("cart")`, want: nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			tokens, _ := Tokens(testCase.source, 1000)
			require.Equal(t, testCase.want, literalValues(FindCallArguments(tokens, "piko", "partial")))
		})
	}
}

func TestFindCallArrayArguments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   []string
	}{
		{name: "literal elements", source: `piko.partials.reloadGroup(["a", 'b'], {})`, want: []string{"a", "b"}},
		{name: "computed elements are skipped", source: `piko.partials.reloadGroup([x, "a" + y, ok ? "b" : "c", "d"])`, want: []string{"d"}},
		{name: "nested literals are skipped", source: `piko.partials.reloadGroup([f(a, "x", b), ["y"], {name: "z"}, "w"])`, want: []string{"w"}},
		{name: "spread is skipped", source: `piko.partials.reloadGroup([...names, "last"])`, want: []string{"last"}},
		{name: "array in a variable is skipped", source: `piko.partials.reloadGroup(names)`, want: nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			tokens, _ := Tokens(testCase.source, 1000)
			require.Equal(t, testCase.want, literalValues(FindCallArrayArguments(tokens, "piko", "partials", "reloadGroup")))
		})
	}
}

func literalValues(literals []Literal) []string {
	var values []string
	for _, literal := range literals {
		values = append(values, literal.Value)
	}
	return values
}

func indexOfStart(tokens []Token, start int) int {
	for index, token := range tokens {
		if token.Start == start {
			return index
		}
	}
	return -1
}
