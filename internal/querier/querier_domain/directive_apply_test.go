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

func makeTokenSpan(line, column int) querier_dto.TextSpan {
	return querier_dto.TextSpan{Line: line, Column: column, EndLine: line, EndColumn: column + 1}
}

func TestApplyTopCall_PositionalsFillNameAndCommand(t *testing.T) {
	t.Parallel()

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 5),
		closeSpan: makeTokenSpan(1, 20),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "GetUser"}, span: makeTokenSpan(1, 6)},
			{value: parsedValue{raw: "one"}, span: makeTokenSpan(1, 15)},
		},
	}
	result := &querier_dto.DirectiveBlock{}

	applyTopCall(result, call)

	require.NotNil(t, result.Name)
	assert.Equal(t, "GetUser", result.Name.Value)
	require.NotNil(t, result.Command)
	assert.Equal(t, querier_dto.QueryCommandOne, result.Command.Command)
}

func TestApplyTopCall_KeywordsRouteToNameCommandAndMetadata(t *testing.T) {
	t.Parallel()

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 5),
		closeSpan: makeTokenSpan(1, 50),
		closed:    true,
		keywordArguments: []parsedKeywordArgument{
			{key: "name", value: parsedValue{raw: "GetUser"}, span: makeTokenSpan(1, 6), keySpan: makeTokenSpan(1, 6), valueSpan: makeTokenSpan(1, 12)},
			{key: "command", value: parsedValue{raw: "many"}, span: makeTokenSpan(1, 21), keySpan: makeTokenSpan(1, 21), valueSpan: makeTokenSpan(1, 30)},
			{key: "group_by", value: parsedValue{raw: "orders.id"}, span: makeTokenSpan(1, 35), keySpan: makeTokenSpan(1, 35), valueSpan: makeTokenSpan(1, 45)},
		},
	}
	result := &querier_dto.DirectiveBlock{}

	applyTopCall(result, call)

	require.NotNil(t, result.Name)
	assert.Equal(t, "GetUser", result.Name.Value)
	require.NotNil(t, result.Command)
	assert.Equal(t, querier_dto.QueryCommandMany, result.Command.Command)
	require.Len(t, result.Metadata, 1)
	assert.Equal(t, "group_by", result.Metadata[0].Directive)
	assert.Equal(t, "orders.id", result.Metadata[0].Value)
}

func TestApplyTopCall_KeywordDoesNotOverridePositional(t *testing.T) {
	t.Parallel()

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 5),
		closeSpan: makeTokenSpan(1, 40),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "Positional"}, span: makeTokenSpan(1, 6)},
		},
		keywordArguments: []parsedKeywordArgument{
			{key: "name", value: parsedValue{raw: "FromKeyword"}, span: makeTokenSpan(1, 20), keySpan: makeTokenSpan(1, 20), valueSpan: makeTokenSpan(1, 26)},
		},
	}
	result := &querier_dto.DirectiveBlock{}

	applyTopCall(result, call)

	require.NotNil(t, result.Name)
	assert.Equal(t, "Positional", result.Name.Value)
}

func TestApplyTopCall_InvalidCommandStaysUnset(t *testing.T) {
	t.Parallel()

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 5),
		closeSpan: makeTokenSpan(1, 30),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "GetUser"}, span: makeTokenSpan(1, 6)},
			{value: parsedValue{raw: "notACommand"}, span: makeTokenSpan(1, 15)},
		},
	}
	result := &querier_dto.DirectiveBlock{}

	applyTopCall(result, call)

	require.NotNil(t, result.Name)
	assert.Nil(t, result.Command)
}

func TestApplyTopCall_EmptyPositionalsAndKeywordsLeaveResultUntouched(t *testing.T) {
	t.Parallel()

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 5),
		closeSpan: makeTokenSpan(1, 6),
		closed:    true,
	}
	result := &querier_dto.DirectiveBlock{}

	applyTopCall(result, call)

	assert.Nil(t, result.Name)
	assert.Nil(t, result.Command)
	assert.Empty(t, result.Metadata)
}

func TestApplyHeaderCall_DispatchesEmbed(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.embed")
	require.True(t, ok)

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 12),
		closeSpan: makeTokenSpan(1, 28),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "orders"}, span: makeTokenSpan(1, 13)},
		},
		keywordArguments: []parsedKeywordArgument{
			{key: "from", value: parsedValue{raw: "o"}, span: makeTokenSpan(1, 21), keySpan: makeTokenSpan(1, 21), valueSpan: makeTokenSpan(1, 27)},
		},
	}
	result := &querier_dto.DirectiveBlock{}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")
	var diagnostics []querier_dto.SourceError

	applyHeaderCall(result, spec, call, errorBuilder, &diagnostics)

	require.Len(t, result.Embeds, 1)
	assert.Equal(t, "orders", result.Embeds[0].Table)
	assert.Equal(t, "o", result.Embeds[0].From)
	assert.Empty(t, diagnostics)
}

func TestApplyHeaderCall_DispatchesColumn(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.column")
	require.True(t, ok)

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 12),
		closeSpan: makeTokenSpan(1, 35),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "email_lower"}, span: makeTokenSpan(1, 13)},
		},
		keywordArguments: []parsedKeywordArgument{
			{key: "type", value: parsedValue{raw: "text"}, span: makeTokenSpan(1, 25), keySpan: makeTokenSpan(1, 25), valueSpan: makeTokenSpan(1, 31)},
		},
	}
	result := &querier_dto.DirectiveBlock{}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")
	var diagnostics []querier_dto.SourceError

	applyHeaderCall(result, spec, call, errorBuilder, &diagnostics)

	require.Len(t, result.ColumnOverrides, 1)
	assert.Equal(t, "email_lower", result.ColumnOverrides[0].Name)
	assert.Equal(t, "text", result.ColumnOverrides[0].SQLType)
}

func TestApplyEmbedHeaderCall_FillsTableFromKeyword(t *testing.T) {
	t.Parallel()

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 12),
		closeSpan: makeTokenSpan(1, 35),
		closed:    true,
		keywordArguments: []parsedKeywordArgument{
			{key: "table", value: parsedValue{raw: "orders"}, span: makeTokenSpan(1, 13), keySpan: makeTokenSpan(1, 13), valueSpan: makeTokenSpan(1, 20)},
			{key: "as", value: parsedValue{raw: "Orders"}, span: makeTokenSpan(1, 25), keySpan: makeTokenSpan(1, 25), valueSpan: makeTokenSpan(1, 30)},
		},
	}
	result := &querier_dto.DirectiveBlock{}

	applyEmbedHeaderCall(result, call)

	require.Len(t, result.Embeds, 1)
	assert.Equal(t, "orders", result.Embeds[0].Table)
	assert.Equal(t, "Orders", result.Embeds[0].As)
	require.Len(t, result.Metadata, 1)
	assert.Equal(t, "embed", result.Metadata[0].Directive)
	assert.Equal(t, "orders", result.Metadata[0].Value)
}

func TestApplyEmbedHeaderCall_PositionalWins(t *testing.T) {
	t.Parallel()

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 12),
		closeSpan: makeTokenSpan(1, 40),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "positional"}, span: makeTokenSpan(1, 13)},
		},
		keywordArguments: []parsedKeywordArgument{
			{key: "table", value: parsedValue{raw: "keyword"}, span: makeTokenSpan(1, 25), keySpan: makeTokenSpan(1, 25), valueSpan: makeTokenSpan(1, 32)},
		},
	}
	result := &querier_dto.DirectiveBlock{}

	applyEmbedHeaderCall(result, call)

	require.Len(t, result.Embeds, 1)
	assert.Equal(t, "positional", result.Embeds[0].Table)
}

func TestApplyColumnHeaderCall_TypeOnly(t *testing.T) {
	t.Parallel()

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 12),
		closeSpan: makeTokenSpan(1, 35),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "id"}, span: makeTokenSpan(1, 13)},
		},
		keywordArguments: []parsedKeywordArgument{
			{key: "type", value: parsedValue{raw: "int8"}, span: makeTokenSpan(1, 17), keySpan: makeTokenSpan(1, 17), valueSpan: makeTokenSpan(1, 23)},
		},
	}
	result := &querier_dto.DirectiveBlock{}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")
	var diagnostics []querier_dto.SourceError

	applyColumnHeaderCall(result, call, errorBuilder, &diagnostics)

	require.Len(t, result.ColumnOverrides, 1)
	override := result.ColumnOverrides[0]
	assert.Equal(t, "id", override.Name)
	assert.Equal(t, "int8", override.SQLType)
	assert.Equal(t, "", override.GoType)
	assert.Empty(t, diagnostics)
}

func TestApplyColumnHeaderCall_MutuallyExclusive(t *testing.T) {
	t.Parallel()

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 12),
		closeSpan: makeTokenSpan(1, 50),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "id"}, span: makeTokenSpan(1, 13)},
		},
		keywordArguments: []parsedKeywordArgument{
			{key: "type", value: parsedValue{raw: "text"}, span: makeTokenSpan(1, 17), keySpan: makeTokenSpan(1, 17), valueSpan: makeTokenSpan(1, 23)},
			{key: "go_type", value: parsedValue{raw: "foo.Bar"}, span: makeTokenSpan(1, 30), keySpan: makeTokenSpan(1, 30), valueSpan: makeTokenSpan(1, 40)},
		},
	}
	result := &querier_dto.DirectiveBlock{}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")
	var diagnostics []querier_dto.SourceError

	applyColumnHeaderCall(result, call, errorBuilder, &diagnostics)

	require.Len(t, result.ColumnOverrides, 1)
	require.Len(t, diagnostics, 1)
	assert.Equal(t, querier_dto.CodeMutuallyExclusiveKeywordArgument, diagnostics[0].Code)
}

func TestApplyColumnHeaderCall_NullablePointer(t *testing.T) {
	t.Parallel()

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 12),
		closeSpan: makeTokenSpan(1, 35),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "id"}, span: makeTokenSpan(1, 13)},
		},
		keywordArguments: []parsedKeywordArgument{
			{key: "nullable", value: parsedValue{raw: "true"}, span: makeTokenSpan(1, 17), keySpan: makeTokenSpan(1, 17), valueSpan: makeTokenSpan(1, 27)},
		},
	}
	result := &querier_dto.DirectiveBlock{}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")
	var diagnostics []querier_dto.SourceError

	applyColumnHeaderCall(result, call, errorBuilder, &diagnostics)

	require.Len(t, result.ColumnOverrides, 1)
	override := result.ColumnOverrides[0]
	require.NotNil(t, override.Nullable)
	assert.True(t, *override.Nullable)
}

func TestApplyColumnHeaderCall_NullableFalse(t *testing.T) {
	t.Parallel()

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 12),
		closeSpan: makeTokenSpan(1, 35),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "id"}, span: makeTokenSpan(1, 13)},
		},
		keywordArguments: []parsedKeywordArgument{
			{key: "nullable", value: parsedValue{raw: "false"}, span: makeTokenSpan(1, 17), keySpan: makeTokenSpan(1, 17), valueSpan: makeTokenSpan(1, 27)},
		},
	}
	result := &querier_dto.DirectiveBlock{}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")
	var diagnostics []querier_dto.SourceError

	applyColumnHeaderCall(result, call, errorBuilder, &diagnostics)

	require.Len(t, result.ColumnOverrides, 1)
	override := result.ColumnOverrides[0]
	require.NotNil(t, override.Nullable)
	assert.False(t, *override.Nullable)
}

func TestApplyColumnHeaderCall_NameFromKeyword(t *testing.T) {
	t.Parallel()

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 12),
		closeSpan: makeTokenSpan(1, 35),
		closed:    true,
		keywordArguments: []parsedKeywordArgument{
			{key: "name", value: parsedValue{raw: "email_lower"}, span: makeTokenSpan(1, 13), keySpan: makeTokenSpan(1, 13), valueSpan: makeTokenSpan(1, 20)},
			{key: "type", value: parsedValue{raw: "text"}, span: makeTokenSpan(1, 25), keySpan: makeTokenSpan(1, 25), valueSpan: makeTokenSpan(1, 31)},
		},
	}
	result := &querier_dto.DirectiveBlock{}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")
	var diagnostics []querier_dto.SourceError

	applyColumnHeaderCall(result, call, errorBuilder, &diagnostics)

	require.Len(t, result.ColumnOverrides, 1)
	assert.Equal(t, "email_lower", result.ColumnOverrides[0].Name)
}

func TestBuildParameterDirective_PositionalNameFillsName(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.param")
	require.True(t, ok)

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 20),
		closeSpan: makeTokenSpan(1, 35),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "user_id"}, span: makeTokenSpan(1, 21)},
		},
	}
	anchorToken := token{lexeme: "$1", span: makeTokenSpan(1, 4)}
	kindSpan := makeTokenSpan(1, 10)

	directive := buildParameterDirective(spec, call, anchorToken, kindSpan, false, 1)

	require.NotNil(t, directive)
	assert.Equal(t, "user_id", directive.Name)
	assert.Equal(t, "param", directive.DirectiveName)
	assert.Equal(t, 1, directive.Number)
	assert.False(t, directive.IsNamed)
}

func TestBuildParameterDirective_KeywordsFillTypeNullableDefault(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.param")
	require.True(t, ok)

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 20),
		closeSpan: makeTokenSpan(1, 70),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "user_id"}, span: makeTokenSpan(1, 21)},
		},
		keywordArguments: []parsedKeywordArgument{
			{key: "type", value: parsedValue{raw: "int8"}, span: makeTokenSpan(1, 30), keySpan: makeTokenSpan(1, 30), valueSpan: makeTokenSpan(1, 36)},
			{key: "nullable", value: parsedValue{raw: "true"}, span: makeTokenSpan(1, 42), keySpan: makeTokenSpan(1, 42), valueSpan: makeTokenSpan(1, 52)},
			{key: "default", value: parsedValue{raw: "42"}, span: makeTokenSpan(1, 58), keySpan: makeTokenSpan(1, 58), valueSpan: makeTokenSpan(1, 68)},
		},
	}
	anchorToken := token{lexeme: "$1", span: makeTokenSpan(1, 4)}
	kindSpan := makeTokenSpan(1, 10)

	directive := buildParameterDirective(spec, call, anchorToken, kindSpan, false, 1)

	require.NotNil(t, directive)
	require.NotNil(t, directive.TypeHint)
	assert.Equal(t, "int8", *directive.TypeHint)
	require.NotNil(t, directive.Nullable)
	assert.True(t, *directive.Nullable)
	require.NotNil(t, directive.DefaultVal)
	assert.Equal(t, 42, *directive.DefaultVal)
}

func TestBuildParameterDirective_SortableColumnsListFillsColumns(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.sortable")
	require.True(t, ok)

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 20),
		closeSpan: makeTokenSpan(1, 50),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "order_by"}, span: makeTokenSpan(1, 21)},
			{
				value: parsedValue{
					raw:    "",
					asList: []string{"name", "total"},
					isList: true,
				},
				span: makeTokenSpan(1, 31),
			},
		},
	}
	anchorToken := token{lexeme: "$1", span: makeTokenSpan(1, 4)}
	kindSpan := makeTokenSpan(1, 10)

	directive := buildParameterDirective(spec, call, anchorToken, kindSpan, false, 1)

	require.NotNil(t, directive)
	assert.Equal(t, "order_by", directive.Name)
	assert.Equal(t, []string{"name", "total"}, directive.Columns)
}

func TestBuildParameterDirective_MaxIntKeywordFillsMaxVal(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.param")
	require.True(t, ok)

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 20),
		closeSpan: makeTokenSpan(1, 50),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "page_size"}, span: makeTokenSpan(1, 21)},
		},
		keywordArguments: []parsedKeywordArgument{
			{key: "max", value: parsedValue{raw: "100"}, span: makeTokenSpan(1, 32), keySpan: makeTokenSpan(1, 32), valueSpan: makeTokenSpan(1, 45)},
		},
	}
	anchorToken := token{lexeme: "$1", span: makeTokenSpan(1, 4)}
	kindSpan := makeTokenSpan(1, 10)

	directive := buildParameterDirective(spec, call, anchorToken, kindSpan, false, 1)

	require.NotNil(t, directive)
	require.NotNil(t, directive.MaxVal)
	assert.Equal(t, 100, *directive.MaxVal)
}

func TestBuildParameterDirective_NoPositionalsReturnsNil(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.param")
	require.True(t, ok)

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 20),
		closeSpan: makeTokenSpan(1, 21),
		closed:    true,
	}
	anchorToken := token{lexeme: "$1", span: makeTokenSpan(1, 4)}
	kindSpan := makeTokenSpan(1, 10)

	directive := buildParameterDirective(spec, call, anchorToken, kindSpan, false, 1)

	assert.Nil(t, directive)
}

func TestBuildParameterDirective_NamedAnchorMarksIsNamed(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.param")
	require.True(t, ok)

	call := &callArgs{
		openSpan:  makeTokenSpan(1, 20),
		closeSpan: makeTokenSpan(1, 30),
		closed:    true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "status"}, span: makeTokenSpan(1, 21)},
		},
	}
	anchorToken := token{lexeme: ":status", span: makeTokenSpan(1, 4)}
	kindSpan := makeTokenSpan(1, 10)

	directive := buildParameterDirective(spec, call, anchorToken, kindSpan, true, 3)

	require.NotNil(t, directive)
	assert.True(t, directive.IsNamed)
	assert.Equal(t, 3, directive.Number)
	assert.Equal(t, querier_dto.ParameterDirectiveParam, directive.Kind)
}
