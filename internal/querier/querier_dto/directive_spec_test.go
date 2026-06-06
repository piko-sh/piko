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

package querier_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupDirective_ResolvesKnownNamesAndRejectsUnknown(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		lookup      string
		expectFound bool
		expectRole  DirectiveRole
	}{
		{name: "top-level piko.query resolves to top role", lookup: "piko.query", expectFound: true, expectRole: DirectiveRoleTop},
		{name: "piko.param resolves to param role", lookup: "piko.param", expectFound: true, expectRole: DirectiveRoleParam},
		{name: "piko.embed resolves to header role", lookup: "piko.embed", expectFound: true, expectRole: DirectiveRoleHeader},
		{name: "piko.migration resolves to migration role", lookup: "piko.migration", expectFound: true, expectRole: DirectiveRoleMigration},
		{name: "unknown name is not found", lookup: "piko.nope", expectFound: false},
		{name: "wrong case is not found because lookup is case-sensitive", lookup: "Piko", expectFound: false},
		{name: "empty name is not found", lookup: "", expectFound: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec, found := LookupDirective(testCase.lookup)
			assert.Equal(t, testCase.expectFound, found)
			if testCase.expectFound {
				require.NotNil(t, spec)
				assert.Equal(t, testCase.lookup, spec.Name)
				assert.Equal(t, testCase.expectRole, spec.Role)
			} else {
				assert.Nil(t, spec)
			}
		})
	}
}

func TestLookupDirective_ReturnsPointerIntoRegistrySoParamKindIsCarried(t *testing.T) {
	t.Parallel()

	spec, found := LookupDirective("piko.sortable")
	require.True(t, found)
	require.NotNil(t, spec)
	assert.Equal(t, ParameterDirectiveSortable, spec.ParamKind)
}

func TestLookupKeywordArgument_ResolvesAcceptedKeysAndRejectsUnknown(t *testing.T) {
	t.Parallel()

	spec, found := LookupDirective("piko.query")
	require.True(t, found)

	keywordArgument, ok := LookupKeywordArgument(spec, "readonly")
	require.True(t, ok)
	require.NotNil(t, keywordArgument)
	assert.Equal(t, "readonly", keywordArgument.Name)
	assert.Equal(t, KeywordArgumentBool, keywordArgument.Kind)

	_, ok = LookupKeywordArgument(spec, "does_not_exist")
	assert.False(t, ok)
}

func TestLookupKeywordArgument_ReturnsFalseForNilSpec(t *testing.T) {
	t.Parallel()

	keywordArgument, ok := LookupKeywordArgument(nil, "readonly")
	assert.False(t, ok)
	assert.Nil(t, keywordArgument)
}

func TestLookupPositional_ResolvesByNameAndReportsIndex(t *testing.T) {
	t.Parallel()

	spec, found := LookupDirective("piko.query")
	require.True(t, found)

	first, firstIndex, ok := LookupPositional(spec, "name")
	require.True(t, ok)
	require.NotNil(t, first)
	assert.Equal(t, "name", first.Name)
	assert.Equal(t, 0, firstIndex)

	second, secondIndex, ok := LookupPositional(spec, "command")
	require.True(t, ok)
	require.NotNil(t, second)
	assert.Equal(t, "command", second.Name)
	assert.Equal(t, 1, secondIndex)
}

func TestLookupPositional_ReturnsFalseForUnknownNameAndNilSpec(t *testing.T) {
	t.Parallel()

	spec, found := LookupDirective("piko.query")
	require.True(t, found)

	positional, index, ok := LookupPositional(spec, "missing")
	assert.False(t, ok)
	assert.Nil(t, positional)
	assert.Equal(t, 0, index)

	positional, index, ok = LookupPositional(nil, "name")
	assert.False(t, ok)
	assert.Nil(t, positional)
	assert.Equal(t, 0, index)
}

func TestPositionalNames_ReturnsDeclarationOrder(t *testing.T) {
	t.Parallel()

	spec, found := LookupDirective("piko.query")
	require.True(t, found)

	assert.Equal(t, []string{"name", "command"}, PositionalNames(spec))
}

func TestPositionalNames_ReturnsNilForNilSpec(t *testing.T) {
	t.Parallel()

	assert.Nil(t, PositionalNames(nil))
}

func TestPositionalNames_ReturnsEmptySliceForDirectiveWithoutPositionals(t *testing.T) {
	t.Parallel()

	spec, found := LookupDirective("piko.param")
	require.True(t, found)

	specNoPositionals := &DirectiveSpec{Name: "synthetic"}
	assert.Empty(t, PositionalNames(specNoPositionals))

	assert.Equal(t, []string{"name"}, PositionalNames(spec))
}

func TestDirectiveNames_ContainsEveryRegisteredDirectiveInOrder(t *testing.T) {
	t.Parallel()

	names := DirectiveNames()
	assert.Equal(t, []string{
		"piko.query",
		"piko.param",
		"piko.sortable",
		"piko.embed",
		"piko.column",
		"piko.migration",
	}, names)
	assert.Len(t, names, len(DirectiveSpecs))
}

func TestKeywordArgumentNames_ReturnsAcceptedKeysInDisplayOrder(t *testing.T) {
	t.Parallel()

	spec, found := LookupDirective("piko.query")
	require.True(t, found)

	assert.Equal(t, []string{"dynamic", "readonly", "nullable", "group_by"}, KeywordArgumentNames(spec))
}

func TestKeywordArgumentNames_ReturnsNilForNilSpec(t *testing.T) {
	t.Parallel()

	assert.Nil(t, KeywordArgumentNames(nil))
}

func TestKeywordArgumentNames_ReturnsEmptyForDirectiveWithoutKeywordArguments(t *testing.T) {
	t.Parallel()

	spec, found := LookupDirective("piko.sortable")
	require.True(t, found)

	assert.Empty(t, KeywordArgumentNames(spec))
}

func TestDirectiveSpecs_CommandPositionalCarriesClosedEnum(t *testing.T) {
	t.Parallel()

	spec, found := LookupDirective("piko.query")
	require.True(t, found)

	command, _, ok := LookupPositional(spec, "command")
	require.True(t, ok)
	assert.Equal(t, KeywordArgumentIdent, command.Kind)
	assert.Equal(t, []string{
		"one", "many", "exec", "execresult", "execrows",
		"batch", "stream", "copyfrom", "asyncexec",
	}, command.AllowedValues)
	assert.True(t, command.Required)
}
