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

package goastutil

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

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
		"main", "col`name", "null\x00byte", "\xed\xa0\x80", "\xff\xfe", "\ufeffbom",
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

func TestIsValidGoIdentifier(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "plain lower case name", input: "user", expected: true},
		{name: "mixed case name", input: "userID", expected: true},
		{name: "leading underscore", input: "_private", expected: true},
		{name: "predeclared identifier is still legal", input: "error", expected: true},
		{name: "unicode letters", input: "café", expected: true},
		{name: "han characters", input: "日本語", expected: true},
		{name: "empty string", input: "", expected: false},
		{name: "blank identifier", input: "_", expected: false},
		{name: "keyword", input: "range", expected: false},
		{name: "leading digit", input: "2fa", expected: false},
		{name: "hyphen", input: "my-query", expected: false},
		{name: "dot", input: "my.query", expected: false},
		{name: "space", input: "my query", expected: false},
		{name: "dollar sign is legal in JS but not Go", input: "$id", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, IsValidGoIdentifier(testCase.input))
		})
	}
}

func TestIsGoPredeclared(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "builtin type", input: "string", expected: true},
		{name: "builtin function", input: "append", expected: true},
		{name: "builtin constant", input: "iota", expected: true},
		{name: "nil", input: "nil", expected: true},
		{name: "recent builtin", input: "min", expected: true},
		{name: "keyword is not predeclared", input: "range", expected: false},
		{name: "user name", input: "User", expected: false},
		{name: "empty string", input: "", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, IsGoPredeclared(testCase.input))
		})
	}
}

func TestIsGoPackageNameReserved(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "main", input: "main", expected: true},
		{name: "name starting with main", input: "mainly", expected: false},
		{name: "predeclared is not package reserved", input: "string", expected: false},
		{name: "keyword is not package reserved", input: "range", expected: false},
		{name: "empty string", input: "", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, IsGoPackageNameReserved(testCase.input))
		})
	}
}

func TestSanitiseGoIdentifier(t *testing.T) {
	runIdentifierCases(t, []identifierCase{
		{name: "already valid passes through", input: "userID", expected: "userID"},
		{name: "leading underscore kept", input: "_private", expected: "_private"},
		{name: "double underscore kept", input: "__", expected: "__"},
		{name: "keyword suffixed", input: "range", expected: "range_"},
		{name: "type keyword suffixed", input: "type", expected: "type_"},
		{name: "predeclared type suffixed", input: "string", expected: "string_"},
		{name: "predeclared function suffixed", input: "len", expected: "len_"},
		{name: "leading digit prefixed", input: "2fa_enabled", expected: "_2fa_enabled"},
		{name: "hyphen replaced", input: "my-query", expected: "my_query"},
		{name: "dot replaced", input: "my.query", expected: "my_query"},
		{name: "space replaced", input: "my query", expected: "my_query"},
		{name: "slash replaced", input: "my/query", expected: "my_query"},
		{name: "dollar replaced", input: "$id", expected: "_id"},
		{name: "unicode letters kept", input: "café", expected: "café"},
		{name: "han characters kept", input: "日本語", expected: "日本語"},
		{name: "emoji replaced then falls back", input: "😀", expected: DefaultGoIdentifier},
		{name: "empty string falls back", input: "", expected: DefaultGoIdentifier},
		{name: "blank identifier falls back", input: "_", expected: DefaultGoIdentifier},
		{name: "punctuation only falls back", input: "-", expected: DefaultGoIdentifier},
		{name: "newline replaced", input: "a\nb", expected: "a_b"},
	}, SanitiseGoIdentifier)
}

func TestSanitiseGoIdentifierDoesNotCollapseDistinctNames(t *testing.T) {
	assert.NotEqual(t, SanitiseGoIdentifier("2fa"), SanitiseGoIdentifier("3fa"))
	assert.NotEqual(t, SanitiseGoIdentifier("user1"), SanitiseGoIdentifier("user2"))
}

func TestSanitiseGoExportedIdentifier(t *testing.T) {
	runIdentifierCases(t, []identifierCase{
		{name: "already exported passes through", input: "UserQuery", expected: "UserQuery"},
		{name: "lower case is capitalised", input: "userQuery", expected: "UserQuery"},
		{name: "leading underscore dropped", input: "_private", expected: "Private"},
		{name: "leading digit prefixed", input: "2fa", expected: "X2fa"},
		{name: "hyphen replaced", input: "list-users", expected: "List_users"},
		{name: "han characters prefixed", input: "日本語", expected: "X日本語"},
		{name: "keyword is capitalised not suffixed", input: "range", expected: "Range"},
		{name: "predeclared is capitalised", input: "string", expected: "String"},
		{name: "empty string falls back", input: "", expected: "Piko"},
		{name: "blank identifier falls back", input: "_", expected: "Piko"},
		{name: "unicode letters kept", input: "café", expected: "Café"},
	}, SanitiseGoExportedIdentifier)
}

func TestSanitiseGoPackageName(t *testing.T) {
	runIdentifierCases(t, []identifierCase{
		{name: "already valid passes through", input: "posts", expected: "posts"},
		{name: "upper case is folded", input: "Posts", expected: "posts"},
		{name: "space collapses to underscore", input: "My Collection", expected: "my_collection"},
		{name: "hyphen collapses to underscore", input: "user-store", expected: "user_store"},
		{name: "run of punctuation collapses once", input: "user -- store", expected: "user_store"},
		{name: "leading digit prefixed", input: "2fa", expected: "p2fa"},
		{name: "keyword suffixed", input: "range", expected: "range_"},
		{name: "predeclared suffixed", input: "string", expected: "string_"},
		{name: "main is left alone, since only a package clause may not use it", input: "main", expected: "main"},
		{name: "trailing punctuation trimmed", input: "posts!", expected: "posts"},
		{name: "empty string falls back", input: "", expected: DefaultGoPackageName},
		{name: "punctuation only falls back", input: "--", expected: DefaultGoPackageName},
		{name: "emoji only falls back", input: "😀", expected: DefaultGoPackageName},
		{name: "unicode letters kept", input: "Café", expected: "café"},
	}, SanitiseGoPackageName)
}

func TestGoPackageAlias(t *testing.T) {
	alias := GoPackageAlias("acme/user-store")

	assert.True(t, IsValidGoIdentifier(alias), "alias %q must be a legal identifier", alias)
	assert.True(t, strings.HasPrefix(alias, "acme_user_store_"), "alias %q keeps its stem", alias)
	assert.Len(t, alias, len("acme_user_store_")+ShortHashLength)
	assert.Equal(t, alias, GoPackageAlias("acme/user-store"), "alias must be deterministic")
}

func TestGoPackageAliasSeparatesNamesThatFoldTogether(t *testing.T) {
	hyphenated := GoPackageAlias("acme/user-store")
	dotted := GoPackageAlias("acme/user.store")
	spaced := GoPackageAlias("acme/user store")

	assert.NotEqual(t, hyphenated, dotted, "distinct paths must not share an alias")
	assert.NotEqual(t, hyphenated, spaced, "distinct paths must not share an alias")
	assert.NotEqual(t, dotted, spaced, "distinct paths must not share an alias")
}

func TestGoPackageAliasWithStem(t *testing.T) {
	alias := GoPackageAliasWithStem("user store", "acme/user-store")

	assert.True(t, IsValidGoIdentifier(alias))
	assert.True(t, strings.HasPrefix(alias, "user_store_"), "alias %q keeps its stem", alias)
	assert.True(
		t,
		strings.HasSuffix(alias, ShortHash("acme/user-store")),
		"alias %q is keyed on the full path",
		alias,
	)
	assert.NotEqual(
		t,
		alias,
		GoPackageAliasWithStem("user store", "acme/user.store"),
		"a shared stem must still yield distinct aliases",
	)
}

func TestShortHash(t *testing.T) {
	first := ShortHash("acme/user-store")

	assert.Len(t, first, ShortHashLength)
	assert.Equal(t, first, ShortHash("acme/user-store"), "hashing must be deterministic")
	assert.NotEqual(t, first, ShortHash("acme/user.store"))
	assert.Len(t, ShortHash(""), ShortHashLength)
}

func TestDisambiguateIdentifier(t *testing.T) {
	used := map[string]struct{}{"foo": {}, "foo2": {}, "bar": {}}

	assert.Equal(t, "baz", DisambiguateIdentifier("baz", used))
	assert.Equal(t, "foo3", DisambiguateIdentifier("foo", used), "suffix search skips taken names")
	assert.Equal(t, "bar2", DisambiguateIdentifier("bar", used))
	assert.Len(t, used, 3, "the set must not be modified")
}

func TestReserveIdentifier(t *testing.T) {
	used := make(map[string]struct{})

	assert.Equal(t, "foo", ReserveIdentifier("foo", used))
	assert.Equal(t, "foo2", ReserveIdentifier("foo", used))
	assert.Equal(t, "foo3", ReserveIdentifier("foo", used))
	assert.Len(t, used, 3)
}

func TestDisambiguateIdentifiers(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{name: "empty list", input: nil, expected: []string{}},
		{name: "already unique", input: []string{"a", "b"}, expected: []string{"a", "b"}},
		{
			name:     "folded names are suffixed",
			input:    []string{"FooBar", "FooBar", "FooBar"},
			expected: []string{"FooBar", "FooBar2", "FooBar3"},
		},
		{
			name:     "second order collision with a literal suffix",
			input:    []string{"foo", "foo", "foo2"},
			expected: []string{"foo", "foo2", "foo22"},
		},
		{
			name:     "order is preserved",
			input:    []string{"b", "a", "b"},
			expected: []string{"b", "a", "b2"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, DisambiguateIdentifiers(testCase.input))
		})
	}
}

func TestSanitiseJSONTagName(t *testing.T) {
	runIdentifierCases(t, []identifierCase{
		{name: "plain column", input: "user_id", expected: "user_id"},
		{name: "expression alias is legal in a tag", input: "COUNT(*)", expected: "COUNT(*)"},
		{name: "leading digit is legal in a tag", input: "2fa_enabled", expected: "2fa_enabled"},
		{name: "hyphen is legal in a tag", input: "user-id", expected: "user-id"},
		{name: "double quote replaced", input: `col"name`, expected: "col_name"},
		{name: "backtick replaced", input: "col`name", expected: "col_name"},
		{name: "comma replaced", input: "col,name", expected: "col_name"},
		{name: "tab replaced", input: "col\tname", expected: "col_name"},
		{name: "unicode letters kept", input: "naïve", expected: "naïve"},
		{name: "empty string falls back", input: "", expected: DefaultJSONTagName},
	}, SanitiseJSONTagName)
}

func TestSanitisersAlwaysProduceUsableNames(t *testing.T) {
	for _, input := range hostileNames {
		goName := SanitiseGoIdentifier(input)
		assert.True(t, IsValidGoIdentifier(goName), "%q became invalid Go name %q", input, goName)
		assert.False(t, IsGoPredeclared(goName), "%q became predeclared name %q", input, goName)

		exported := SanitiseGoExportedIdentifier(input)
		assert.True(t, IsValidGoIdentifier(exported), "%q became invalid name %q", input, exported)
		assert.True(t, token.IsExported(exported), "%q became unexported name %q", input, exported)

		packageName := SanitiseGoPackageName(input)
		assert.True(t, IsValidGoIdentifier(packageName), "%q became %q", input, packageName)
		assert.False(t, IsGoPredeclared(packageName), "%q became %q", input, packageName)

		alias := GoPackageAlias(input)
		assert.True(t, IsValidGoIdentifier(alias), "%q became invalid alias %q", input, alias)
	}
}

func TestSanitisersAreIdempotent(t *testing.T) {
	for _, input := range hostileNames {
		goName := SanitiseGoIdentifier(input)
		assert.Equal(t, goName, SanitiseGoIdentifier(goName), "input %q", input)

		exported := SanitiseGoExportedIdentifier(input)
		assert.Equal(t, exported, SanitiseGoExportedIdentifier(exported), "input %q", input)

		packageName := SanitiseGoPackageName(input)
		assert.Equal(t, packageName, SanitiseGoPackageName(packageName), "input %q", input)

		tagName := SanitiseJSONTagName(input)
		assert.Equal(t, tagName, SanitiseJSONTagName(tagName), "input %q", input)
	}
}

func TestGoPredeclaredIdentifiersCoverTheUniverseScope(t *testing.T) {
	for _, name := range types.Universe.Names() {
		assert.True(t, IsGoPredeclared(name), "universe name %q is not treated as predeclared", name)
	}

	assert.True(t, IsGoPredeclared("any"))
	assert.True(t, IsGoPredeclared("min"))
	assert.False(t, IsGoPredeclared("range"))
	assert.False(t, IsGoPredeclared(""))
}

func TestSanitiseJSONTagNameNeverOmitsTheField(t *testing.T) {
	assert.Equal(t, DefaultJSONTagName, SanitiseJSONTagName("-"),
		`a tag of "-" tells encoding/json to drop the field`)
	assert.Equal(t, "a-b", SanitiseJSONTagName("a-b"))
	assert.Equal(t, "--", SanitiseJSONTagName("--"))

	for _, input := range hostileNames {
		assert.NotEqual(t, "-", SanitiseJSONTagName(input), "input %q", input)
	}
}

func TestDisambiguateIdentifierProbesPastTakenSuffixes(t *testing.T) {
	used := map[string]struct{}{"foo": {}, "foo2": {}, "foo4": {}}

	assert.Equal(t, "foo3", DisambiguateIdentifier("foo", used))
	assert.Equal(t, "bar", DisambiguateIdentifier("bar", used))

	sequential := map[string]struct{}{}
	for range 5 {
		ReserveIdentifier("name", sequential)
	}
	assert.Equal(t, map[string]struct{}{
		"name": {}, "name2": {}, "name3": {}, "name4": {}, "name5": {},
	}, sequential)
}
