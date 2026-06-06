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
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func newTestParser() *directiveParser {
	return newDirectiveParser(
		[]querier_dto.DirectiveParameterPrefix{
			{Prefix: '$', IsNamed: false},
		},
		querier_dto.CommentStyle{LinePrefix: "--"},
	)
}

func newTestParserWithNamedPrefix() *directiveParser {
	return newDirectiveParser(
		[]querier_dto.DirectiveParameterPrefix{
			{Prefix: '$', IsNamed: false},
			{Prefix: ':', IsNamed: true},
			{Prefix: '@', IsNamed: true},
		},
		querier_dto.CommentStyle{LinePrefix: "--"},
	)
}

func parseSQL(t *testing.T, parser *directiveParser, sql string) (*querier_dto.DirectiveBlock, []querier_dto.SourceError) {
	t.Helper()
	return parser.Parse(queryBlock{sql: sql, startLine: 1}, "test.sql")
}

func TestDirectiveParser_TopCallNameCommand(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(name: GetUser, command: one)\nSELECT 1;")

	require.Empty(t, diagnostics)
	require.NotNil(t, block.Name)
	require.NotNil(t, block.Command)
	assert.Equal(t, "GetUser", block.Name.Value)
	assert.Equal(t, querier_dto.QueryCommandOne, block.Command.Command)
}

func TestDirectiveParser_TopCallAllKeywordArguments(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(name: SearchOrders, command: many, dynamic: runtime, readonly: false, group_by: orders.id)\nSELECT 1;")

	require.Empty(t, diagnostics)
	require.NotNil(t, block.Name)
	require.NotNil(t, block.Command)

	directives := extractQueryDirectives(block)
	assert.True(t, directives.DynamicRuntime)
	require.NotNil(t, directives.ReadOnlyOverride)
	assert.False(t, *directives.ReadOnlyOverride)
	assert.Equal(t, []string{"orders.id"}, directives.GroupByKeys)
}

func TestDirectiveParser_MultiLineTopCall(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(\n" +
		"--   name: SearchOrders,\n" +
		"--   command: many,\n" +
		"--   dynamic: runtime,\n" +
		"-- )\n" +
		"SELECT 1;"

	block, diagnostics := parseSQL(t, newTestParser(), input)

	require.Empty(t, diagnostics)
	require.NotNil(t, block.Name)
	require.NotNil(t, block.Command)
	assert.Equal(t, "SearchOrders", block.Name.Value)
	assert.Equal(t, querier_dto.QueryCommandMany, block.Command.Command)
}

func TestDirectiveParser_ContinuationLineSegmentOffsets(t *testing.T) {
	t.Parallel()

	parser := newTestParser()
	lines := []string{
		"-- piko.query(",
		"--   name: GetUser,",
		"--   command: one",
		"-- )",
	}
	firstStripped, firstColumnOffset, ok := parser.stripCommentPrefix(lines[0])
	require.True(t, ok)

	logical, lastIndex := parser.buildLogicalLine(
		lines, 0, queryBlock{sql: "", startLine: 1}, firstStripped, firstColumnOffset, 1,
	)

	require.Equal(t, 3, lastIndex, "all four physical lines should contribute")
	require.Len(t, logical.segments, 4)
	require.Equal(t, "piko.query(   name: GetUser,   command: one )", logical.content)

	for index := 1; index < len(logical.segments); index++ {
		segment := logical.segments[index]
		stripped, _, stripOk := parser.stripCommentPrefix(lines[index])
		require.True(t, stripOk)
		assert.Equal(t, stripped, logical.content[segment.contentStart:segment.contentEnd])
	}
}

func TestDirectiveParser_ContinuationLineCap(t *testing.T) {
	t.Parallel()

	parser := newTestParser()

	lines := make([]string, 0, maxContinuationLineCount+100)
	lines = append(lines, "-- piko.query(")
	for range maxContinuationLineCount + 50 {
		lines = append(lines, "--   name: GetUser,")
	}

	firstStripped, firstColumnOffset, ok := parser.stripCommentPrefix(lines[0])
	require.True(t, ok)

	logical, lastIndex := parser.buildLogicalLine(
		lines, 0, queryBlock{sql: "", startLine: 1}, firstStripped, firstColumnOffset, 1,
	)

	assert.LessOrEqual(t, len(logical.segments), maxContinuationLineCount+1,
		"continuation segments must be bounded by the per-line cap")
	assert.Equal(t, maxContinuationLineCount, lastIndex,
		"the builder must stop absorbing lines once the cap is reached")
}

func TestDirectiveParser_ParamBindings(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(name: SearchOrders, command: many)\n" +
		"-- $1 as piko.param(status, optional: true)\n" +
		"-- $2 as piko.param(ids, kind: slice, type: int)\n" +
		"-- piko.sortable(order_by, columns: [name, total, placed_at])\n" +
		"-- $3 as piko.param(page_size, default: 10, max: 100)\n" +
		"-- $4 as piko.param(skip)\n" +
		"SELECT 1;"

	block, diagnostics := parseSQL(t, newTestParser(), input)

	require.Empty(t, diagnostics)
	require.Len(t, block.Parameters, 5)

	assert.Equal(t, querier_dto.ParameterDirectiveParam, block.Parameters[0].Kind)
	assert.Equal(t, "status", block.Parameters[0].Name)
	assert.True(t, block.Parameters[0].IsOptional)

	assert.True(t, block.Parameters[1].IsSlice)
	require.NotNil(t, block.Parameters[1].TypeHint)
	assert.Equal(t, "int", *block.Parameters[1].TypeHint)

	assert.Equal(t, querier_dto.ParameterDirectiveSortable, block.Parameters[2].Kind)
	assert.Equal(t, []string{"name", "total", "placed_at"}, block.Parameters[2].Columns)

	assert.Equal(t, querier_dto.ParameterDirectiveParam, block.Parameters[3].Kind)
	require.NotNil(t, block.Parameters[3].DefaultVal)
	assert.Equal(t, 10, *block.Parameters[3].DefaultVal)
	require.NotNil(t, block.Parameters[3].MaxVal)
	assert.Equal(t, 100, *block.Parameters[3].MaxVal)

	assert.Equal(t, "skip", block.Parameters[4].Name)
}

func TestDirectiveParser_NamedAnchor(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(name: GetUser, command: one)\n" +
		"-- :email as piko.param(user_email, type: text)\n" +
		"SELECT 1;"

	block, diagnostics := parseSQL(t, newTestParserWithNamedPrefix(), input)

	require.Empty(t, diagnostics)
	require.Len(t, block.Parameters, 1)
	assert.True(t, block.Parameters[0].IsNamed)
	assert.Equal(t, "user_email", block.Parameters[0].Name)
}

func TestDirectiveParser_EmbedHeader(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(name: GetBookWithAuthor, command: one)\n" +
		"-- piko.embed(authors, from: a)\n" +
		"SELECT 1;"

	block, diagnostics := parseSQL(t, newTestParser(), input)

	require.Empty(t, diagnostics)
	require.Len(t, block.Embeds, 1)
	assert.Equal(t, "authors", block.Embeds[0].Table)
	assert.Equal(t, "a", block.Embeds[0].From)
}

func TestDirectiveParser_UnknownDirectiveSuggestion(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(name: G, command: one)\n" +
		"-- $1 as piko.parm(status)\n" +
		"SELECT 1;"

	_, diagnostics := parseSQL(t, newTestParser(), input)

	require.NotEmpty(t, diagnostics)
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeUnknownDirective {
			assert.Equal(t, "piko.param", diagnostic.Suggestion)
			found = true
		}
	}
	assert.True(t, found, "expected an unknown-directive diagnostic with Suggestion=piko.param")
}

func TestDirectiveParser_UnknownKeywordArgumentSuggestion(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(naem: G, command: one)\nSELECT 1;"

	_, diagnostics := parseSQL(t, newTestParser(), input)

	require.NotEmpty(t, diagnostics)
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeUnknownKeywordArgument {
			assert.Equal(t, "name", diagnostic.Suggestion)
			found = true
		}
	}
	assert.True(t, found, "expected an unknown-keywordArgument diagnostic with Suggestion=name")
}

func TestDirectiveParser_InvalidEnumSuggestion(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(name: G, command: onee)\nSELECT 1;"

	_, diagnostics := parseSQL(t, newTestParser(), input)

	require.NotEmpty(t, diagnostics)
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeInvalidKeywordArgumentValue {
			assert.Equal(t, "one", diagnostic.Suggestion)
			found = true
		}
	}
	assert.True(t, found, "expected invalid-keywordArgument-value diagnostic with Suggestion=one")
}

func TestDirectiveParser_DuplicateKeywordArgument(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(name: G, name: H, command: one)\nSELECT 1;"

	_, diagnostics := parseSQL(t, newTestParser(), input)

	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeDuplicateKeywordArgument {
			found = true
		}
	}
	assert.True(t, found, "expected duplicate-keywordArgument diagnostic")
}

func TestDirectiveParser_MissingRequired(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(command: one)\nSELECT 1;"

	_, diagnostics := parseSQL(t, newTestParser(), input)

	hasMissingName := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeMissingRequired && diagnostic.Message == "missing required name on piko.query" {
			hasMissingName = true
		}
	}
	assert.True(t, hasMissingName)
}

func TestDirectiveParser_UnclosedCall(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(name: G, command: one\nSELECT 1;"

	_, diagnostics := parseSQL(t, newTestParser(), input)

	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeUnclosedDirective {
			found = true
		}
	}
	assert.True(t, found, "expected unclosed-directive diagnostic")
}

func TestDirectiveParser_EmptyInput(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(), "")

	require.NotNil(t, block)
	assert.Nil(t, block.Name)
	assert.Nil(t, block.Command)

	assert.Len(t, diagnostics, 2)
}

func TestDirectiveParser_PreservesGroupByMultiple(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(name: G, command: many, group_by: orders.id)\nSELECT 1;"

	block, diagnostics := parseSQL(t, newTestParser(), input)

	require.Empty(t, diagnostics)
	directives := extractQueryDirectives(block)
	assert.Equal(t, []string{"orders.id"}, directives.GroupByKeys)
}

func TestDirectiveParser_TopCallPositional(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetUser, one)\nSELECT 1;")

	require.Empty(t, diagnostics)
	require.NotNil(t, block.Name)
	require.NotNil(t, block.Command)
	assert.Equal(t, "GetUser", block.Name.Value)
	assert.Equal(t, querier_dto.QueryCommandOne, block.Command.Command)
}

func TestDirectiveParser_TopCallPositionalWithKeywordArguments(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(SearchOrders, many, dynamic: runtime, group_by: orders.id)\nSELECT 1;")

	require.Empty(t, diagnostics)
	assert.Equal(t, "SearchOrders", block.Name.Value)
	assert.Equal(t, querier_dto.QueryCommandMany, block.Command.Command)

	directives := extractQueryDirectives(block)
	assert.True(t, directives.DynamicRuntime)
	assert.Equal(t, []string{"orders.id"}, directives.GroupByKeys)
}

func TestDirectiveParser_KeywordArgumentAsPositional(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(command: one, name: GetUser)\nSELECT 1;")

	require.Empty(t, diagnostics)
	assert.Equal(t, "GetUser", block.Name.Value)
	assert.Equal(t, querier_dto.QueryCommandOne, block.Command.Command)
}

func TestDirectiveParser_KeywordArgumentAsPositionalDuplicate(t *testing.T) {
	t.Parallel()

	_, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetUser, one, name: Other)\nSELECT 1;")

	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeDuplicateKeywordArgument {
			found = true
		}
	}
	assert.True(t, found, "expected duplicate diagnostic when positional and keywordArgument-as-positional fill the same slot")
}

func TestDirectiveParser_TooManyPositionals(t *testing.T) {
	t.Parallel()

	_, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetUser, one, extra)\nSELECT 1;")

	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeDirectiveSyntax {
			found = true
		}
	}
	assert.True(t, found, "expected directive-syntax diagnostic for extra positional")
}

func TestDirectiveParser_PositionalParameterBinding(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetUser, many)\n-- $1 as piko.param(status, optional: true)\nSELECT 1;")

	require.Empty(t, diagnostics)
	require.Len(t, block.Parameters, 1)
	assert.Equal(t, "status", block.Parameters[0].Name)
	assert.Equal(t, querier_dto.ParameterDirectiveParam, block.Parameters[0].Kind)
	assert.True(t, block.Parameters[0].IsOptional)
}

func TestDirectiveParser_SortablePositionalColumns(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetItems, many)\n-- piko.sortable(order_by, [name, price])\nSELECT 1;")

	require.Empty(t, diagnostics)
	require.Len(t, block.Parameters, 1)
	assert.Equal(t, "order_by", block.Parameters[0].Name)
	assert.Equal(t, []string{"name", "price"}, block.Parameters[0].Columns)
}

func TestDirectiveParser_EmbedPositional(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetBook, one)\n-- piko.embed(authors, from: a)\nSELECT 1;")

	require.Empty(t, diagnostics)
	require.Len(t, block.Embeds, 1)
	assert.Equal(t, "authors", block.Embeds[0].Table)
	assert.Equal(t, "a", block.Embeds[0].From)
}

func TestDirectiveParser_ParameterBindingExpectedPiko(t *testing.T) {
	t.Parallel()

	_, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetUser, many)\n-- $1 as pico.optional(status)\nSELECT 1;")

	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeParameterBindingSyntax {
			assert.Equal(t, "piko", diagnostic.Suggestion)
			found = true
		}
	}
	assert.True(t, found, "expected Q033 with Suggestion=piko")
}

func TestDirectiveParser_InvalidListLiteralBracket(t *testing.T) {
	t.Parallel()

	_, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetUser, many)\n-- piko.sortable(order_by, columns: [a, b, c))\nSELECT 1;")

	foundList := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeInvalidListLiteral {
			foundList = true
		}
	}
	assert.True(t, foundList, "expected Q034 for mismatched list closer")
}

func TestDirectiveParser_UnterminatedString(t *testing.T) {
	t.Parallel()

	_, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(name: \"open, command: many)\nSELECT 1;")

	foundUnterm := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeUnterminatedString {
			foundUnterm = true
		}
	}
	assert.True(t, foundUnterm, "expected Q035 for unterminated string")
}

func TestDirectiveParser_ProseCommentWithApostropheIgnored(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(name: VerifyEmail, command: one)\n"+
			"-- Marks an account's email as verified\n"+
			"SELECT 1;")

	require.Empty(t, diagnostics)
	require.NotNil(t, block.Name)
	require.NotNil(t, block.Command)
	assert.Equal(t, "VerifyEmail", block.Name.Value)
	assert.Equal(t, querier_dto.QueryCommandOne, block.Command.Command)
}

func TestDirectiveParser_ProseCommentBeforeDirectiveIgnored(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- Marks an account's email as verified\n"+
			"-- piko.query(name: VerifyEmail, command: one)\n"+
			"SELECT 1;")

	require.Empty(t, diagnostics)
	require.NotNil(t, block.Name)
	assert.Equal(t, "VerifyEmail", block.Name.Value)
}

func TestDirectiveParser_ProseStartingWithPikoWordIgnored(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko keeps this query fast\n"+
			"-- piko.query(name: GetUser, command: one)\n"+
			"SELECT 1;")

	require.Empty(t, diagnostics)
	require.NotNil(t, block.Name)
	assert.Equal(t, "GetUser", block.Name.Value)
}

func TestDirectiveParser_ProseWithUnbalancedParenDoesNotSwallowDirective(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- returns the latest row (if present\n"+
			"-- piko.query(name: GetLatest, command: one)\n"+
			"SELECT 1;")

	require.Empty(t, diagnostics)
	require.NotNil(t, block.Name)
	require.NotNil(t, block.Command)
	assert.Equal(t, "GetLatest", block.Name.Value)
	assert.Equal(t, querier_dto.QueryCommandOne, block.Command.Command)
}

func TestDirectiveParser_ProseWithUnbalancedBracketDoesNotSwallowDirective(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- see the list [item one for context\n"+
			"-- piko.query(name: ListThings, command: many)\n"+
			"SELECT 1;")

	require.Empty(t, diagnostics)
	require.NotNil(t, block.Name)
	require.NotNil(t, block.Command)
	assert.Equal(t, "ListThings", block.Name.Value)
	assert.Equal(t, querier_dto.QueryCommandMany, block.Command.Command)
}

func TestDirectiveParser_MultiLineCallWithSurroundingProse(t *testing.T) {
	t.Parallel()

	input := "-- Fetch a user's orders (most recent first)\n" +
		"-- piko.query(\n" +
		"--   name: SearchOrders,\n" +
		"--   command: many,\n" +
		"-- )\n" +
		"-- ordering is handled by the caller's cursor\n" +
		"SELECT 1;"

	block, diagnostics := parseSQL(t, newTestParser(), input)

	require.Empty(t, diagnostics)
	require.NotNil(t, block.Name)
	require.NotNil(t, block.Command)
	assert.Equal(t, "SearchOrders", block.Name.Value)
	assert.Equal(t, querier_dto.QueryCommandMany, block.Command.Command)
}

func TestDirectiveParser_GenuineUnterminatedStringInMultiLineCall(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(\n" +
		"--   name: GetUser,\n" +
		"--   command: 'one\n" +
		"-- )\n" +
		"SELECT 1;"

	_, diagnostics := parseSQL(t, newTestParser(), input)

	foundUnterm := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeUnterminatedString {
			foundUnterm = true
		}
	}
	assert.True(t, foundUnterm, "expected Q035 for an unterminated string inside a real directive")
}

func TestDirectiveParser_GenuineUnterminatedStringInParamBinding(t *testing.T) {
	t.Parallel()

	input := "-- piko.query(name: SearchOrders, command: many)\n" +
		"-- $1 as piko.param('status\n" +
		"SELECT 1;"

	_, diagnostics := parseSQL(t, newTestParser(), input)

	foundUnterm := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeUnterminatedString {
			foundUnterm = true
		}
	}
	assert.True(t, foundUnterm, "expected Q035 for an unterminated string inside a parameter binding")
}

func TestDirectiveParser_ProseOnlyHeaderStillRequiresDirective(t *testing.T) {
	t.Parallel()

	_, diagnostics := parseSQL(t, newTestParser(),
		"-- describes the user's most recent activity\n"+
			"SELECT 1;")

	require.NotEmpty(t, diagnostics, "a prose-only header must still report missing directives")
	for _, diagnostic := range diagnostics {
		assert.NotEqual(t, querier_dto.CodeUnterminatedString, diagnostic.Code,
			"a prose apostrophe must not raise Q035")
	}
}

func TestDirectiveParser_ColumnOverrideTypeOnly(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetUser, one)\n-- piko.column(email_lower, type: text)\nSELECT 1;")

	require.Empty(t, diagnostics)
	require.Len(t, block.ColumnOverrides, 1)
	assert.Equal(t, "email_lower", block.ColumnOverrides[0].Name)
	assert.Equal(t, "text", block.ColumnOverrides[0].SQLType)
	assert.Equal(t, "", block.ColumnOverrides[0].GoType)
	assert.Nil(t, block.ColumnOverrides[0].Nullable)
}

func TestDirectiveParser_ColumnOverrideGoTypeOnly(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetUser, one)\n-- piko.column(id, go_type: \"github.com/google/uuid.UUID\")\nSELECT 1;")

	require.Empty(t, diagnostics)
	require.Len(t, block.ColumnOverrides, 1)
	assert.Equal(t, "id", block.ColumnOverrides[0].Name)
	assert.Equal(t, "github.com/google/uuid.UUID", block.ColumnOverrides[0].GoType)
}

func TestDirectiveParser_ColumnOverrideNullable(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetUser, one)\n-- piko.column(id, nullable: true)\nSELECT 1;")

	require.Empty(t, diagnostics)
	require.Len(t, block.ColumnOverrides, 1)
	require.NotNil(t, block.ColumnOverrides[0].Nullable)
	assert.True(t, *block.ColumnOverrides[0].Nullable)
}

func TestDirectiveParser_ColumnOverrideMutuallyExclusive(t *testing.T) {
	t.Parallel()

	_, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetUser, one)\n-- piko.column(id, type: text, go_type: \"foo.Bar\")\nSELECT 1;")

	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeMutuallyExclusiveKeywordArgument {
			found = true
		}
	}
	assert.True(t, found, "expected Q038 mutually-exclusive diagnostic")
}

func TestDirectiveParser_ColumnOverrideMultiple(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetUser, one)\n-- piko.column(id, type: int8)\n-- piko.column(name, nullable: true)\nSELECT 1;")

	require.Empty(t, diagnostics)
	require.Len(t, block.ColumnOverrides, 2)
	assert.Equal(t, "id", block.ColumnOverrides[0].Name)
	assert.Equal(t, "int8", block.ColumnOverrides[0].SQLType)
	assert.Equal(t, "name", block.ColumnOverrides[1].Name)
	require.NotNil(t, block.ColumnOverrides[1].Nullable)
}

func TestDirectiveParser_NegativeNumberLiteral(t *testing.T) {
	t.Parallel()

	block, diagnostics := parseSQL(t, newTestParser(),
		"-- piko.query(GetItems, many)\n-- $1 as piko.param(page_size, default: -5)\nSELECT 1;")

	require.Empty(t, diagnostics)
	require.Len(t, block.Parameters, 1)
	require.NotNil(t, block.Parameters[0].DefaultVal)
	assert.Equal(t, -5, *block.Parameters[0].DefaultVal)
}
