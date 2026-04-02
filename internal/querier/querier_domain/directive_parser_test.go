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

func TestDirectiveParser_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		input              string
		startLine          int
		filename           string
		expectName         *string
		expectCommand      *querier_dto.QueryCommand
		expectParamCount   int
		expectMetadataKeys []string
		expectDiagCount    int
		expectDiagMessages []string
	}{
		{
			name: "complete block with name command and parameter",
			input: "-- piko.name: GetUser\n" +
				"-- piko.command: one\n" +
				"-- $1 as piko.param(user_id)\n",
			startLine:        1,
			filename:         "test.sql",
			expectName:       new("GetUser"),
			expectCommand:    commandPtr(querier_dto.QueryCommandOne),
			expectParamCount: 1,
			expectDiagCount:  0,
		},
		{
			name:            "missing name produces diagnostic",
			input:           "-- piko.command: many\n",
			startLine:       1,
			filename:        "test.sql",
			expectCommand:   commandPtr(querier_dto.QueryCommandMany),
			expectDiagCount: 1,
			expectDiagMessages: []string{
				"missing piko.name directive",
			},
		},
		{
			name:            "missing command produces diagnostic",
			input:           "-- piko.name: ListUsers\n",
			startLine:       1,
			filename:        "test.sql",
			expectName:      new("ListUsers"),
			expectDiagCount: 1,
			expectDiagMessages: []string{
				"missing piko.command directive",
			},
		},
		{
			name:            "name and command only with no parameters or errors",
			input:           "-- piko.name: DeleteUser\n-- piko.command: exec\n",
			startLine:       1,
			filename:        "test.sql",
			expectName:      new("DeleteUser"),
			expectCommand:   commandPtr(querier_dto.QueryCommandExec),
			expectDiagCount: 0,
		},
		{
			name: "multiple numbered parameters",
			input: "-- piko.name: CreateUser\n" +
				"-- piko.command: one\n" +
				"-- $1 as piko.param(name)\n" +
				"-- $2 as piko.param(email)\n" +
				"-- $3 as piko.optional(bio)\n",
			startLine:        1,
			filename:         "test.sql",
			expectName:       new("CreateUser"),
			expectCommand:    commandPtr(querier_dto.QueryCommandOne),
			expectParamCount: 3,
			expectDiagCount:  0,
		},
		{
			name: "with query directives in metadata",
			input: "-- piko.name: ListPosts\n" +
				"-- piko.command: many\n" +
				"-- piko.nullable\n" +
				"-- piko.readonly\n",
			startLine:          1,
			filename:           "test.sql",
			expectName:         new("ListPosts"),
			expectCommand:      commandPtr(querier_dto.QueryCommandMany),
			expectMetadataKeys: []string{"nullable", "readonly"},
			expectDiagCount:    0,
		},
		{
			name:            "empty block produces two diagnostics for missing name and command",
			input:           "-- \n",
			startLine:       1,
			filename:        "test.sql",
			expectDiagCount: 2,
			expectDiagMessages: []string{
				"missing piko.name directive",
				"missing piko.command directive",
			},
		},
		{
			name: "parameter with options",
			input: "-- piko.name: Search\n" +
				"-- piko.command: many\n" +
				"-- $1 as piko.param(query) type:text nullable:true\n",
			startLine:        1,
			filename:         "test.sql",
			expectName:       new("Search"),
			expectCommand:    commandPtr(querier_dto.QueryCommandMany),
			expectParamCount: 1,
			expectDiagCount:  0,
		},
		{
			name: "non-comment line terminates parsing",
			input: "-- piko.name: GetUser\n" +
				"-- piko.command: one\n" +
				"SELECT * FROM users WHERE id = $1;\n",
			startLine:       1,
			filename:        "test.sql",
			expectName:      new("GetUser"),
			expectCommand:   commandPtr(querier_dto.QueryCommandOne),
			expectDiagCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := newTestParser()
			block := queryBlock{sql: tt.input, startLine: tt.startLine}

			result, diagnostics := parser.Parse(block, tt.filename)

			require.NotNil(t, result, "result should never be nil")

			if tt.expectName != nil {
				require.NotNil(t, result.Name, "expected Name directive to be set")
				assert.Equal(t, *tt.expectName, result.Name.Value)
			}

			if tt.expectCommand != nil {
				require.NotNil(t, result.Command, "expected Command directive to be set")
				assert.Equal(t, *tt.expectCommand, result.Command.Command)
			}

			assert.Len(t, result.Parameters, tt.expectParamCount,
				"parameter count mismatch")

			if tt.expectMetadataKeys != nil {
				require.Len(t, result.Metadata, len(tt.expectMetadataKeys),
					"metadata count mismatch")
				for i, expectedKey := range tt.expectMetadataKeys {
					assert.Equal(t, expectedKey, result.Metadata[i].Directive,
						"metadata[%d] directive key mismatch", i)
				}
			}

			assert.Len(t, diagnostics, tt.expectDiagCount,
				"diagnostic count mismatch")
			for i, expectedMessage := range tt.expectDiagMessages {
				if i < len(diagnostics) {
					assert.Equal(t, expectedMessage, diagnostics[i].Message,
						"diagnostic[%d] message mismatch", i)
				}
			}
		})
	}
}

func TestParseQueryCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		expected      querier_dto.QueryCommand
		expectedValid bool
	}{
		{
			name:          "one maps to QueryCommandOne",
			input:         "one",
			expected:      querier_dto.QueryCommandOne,
			expectedValid: true,
		},
		{
			name:          "many maps to QueryCommandMany",
			input:         "many",
			expected:      querier_dto.QueryCommandMany,
			expectedValid: true,
		},
		{
			name:          "exec maps to QueryCommandExec",
			input:         "exec",
			expected:      querier_dto.QueryCommandExec,
			expectedValid: true,
		},
		{
			name:          "execresult maps to QueryCommandExecResult",
			input:         "execresult",
			expected:      querier_dto.QueryCommandExecResult,
			expectedValid: true,
		},
		{
			name:          "execrows maps to QueryCommandExecRows",
			input:         "execrows",
			expected:      querier_dto.QueryCommandExecRows,
			expectedValid: true,
		},
		{
			name:          "batch maps to QueryCommandBatch",
			input:         "batch",
			expected:      querier_dto.QueryCommandBatch,
			expectedValid: true,
		},
		{
			name:          "stream maps to QueryCommandStream",
			input:         "stream",
			expected:      querier_dto.QueryCommandStream,
			expectedValid: true,
		},
		{
			name:          "copyfrom maps to QueryCommandCopyFrom",
			input:         "copyfrom",
			expected:      querier_dto.QueryCommandCopyFrom,
			expectedValid: true,
		},
		{
			name:          "case-insensitive One maps to QueryCommandOne",
			input:         "One",
			expected:      querier_dto.QueryCommandOne,
			expectedValid: true,
		},
		{
			name:          "uppercase MANY maps to QueryCommandMany",
			input:         "MANY",
			expected:      querier_dto.QueryCommandMany,
			expectedValid: true,
		},
		{
			name:          "unknown command returns invalid",
			input:         "unknown",
			expected:      0,
			expectedValid: false,
		},
		{
			name:          "empty string returns invalid",
			input:         "",
			expected:      0,
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			command, valid := parseQueryCommand(tt.input)

			assert.Equal(t, tt.expectedValid, valid,
				"validity mismatch for input %q", tt.input)
			if valid {
				assert.Equal(t, tt.expected, command,
					"command mismatch for input %q", tt.input)
			}
		})
	}
}

func TestParseParameterDirectiveKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		expected      querier_dto.ParameterDirectiveKind
		expectedValid bool
	}{
		{
			name:          "param maps to ParameterDirectiveParam",
			input:         "param",
			expected:      querier_dto.ParameterDirectiveParam,
			expectedValid: true,
		},
		{
			name:          "optional maps to ParameterDirectiveOptional",
			input:         "optional",
			expected:      querier_dto.ParameterDirectiveOptional,
			expectedValid: true,
		},
		{
			name:          "slice maps to ParameterDirectiveSlice",
			input:         "slice",
			expected:      querier_dto.ParameterDirectiveSlice,
			expectedValid: true,
		},
		{
			name:          "sortable maps to ParameterDirectiveSortable",
			input:         "sortable",
			expected:      querier_dto.ParameterDirectiveSortable,
			expectedValid: true,
		},
		{
			name:          "limit maps to ParameterDirectiveLimit",
			input:         "limit",
			expected:      querier_dto.ParameterDirectiveLimit,
			expectedValid: true,
		},
		{
			name:          "offset maps to ParameterDirectiveOffset",
			input:         "offset",
			expected:      querier_dto.ParameterDirectiveOffset,
			expectedValid: true,
		},
		{
			name:          "unknown kind returns invalid",
			input:         "foobar",
			expected:      0,
			expectedValid: false,
		},
		{
			name:          "empty string returns invalid",
			input:         "",
			expected:      0,
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			kind, valid := parseParameterDirectiveKind(tt.input)

			assert.Equal(t, tt.expectedValid, valid,
				"validity mismatch for input %q", tt.input)
			if valid {
				assert.Equal(t, tt.expected, kind,
					"kind mismatch for input %q", tt.input)
			}
		})
	}
}

func TestParseParameterOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		optionKey      string
		optionValue    string
		expectTypeHint *string
		expectNullable *bool
		expectColumns  []string
		expectDefault  *int
		expectMax      *int
	}{
		{
			name:           "type hint sets type override",
			optionKey:      "type",
			optionValue:    "integer",
			expectTypeHint: new("integer"),
		},
		{
			name:           "nullable true sets nullable",
			optionKey:      "nullable",
			optionValue:    "true",
			expectNullable: new(true),
		},
		{
			name:           "nullable false sets not nullable",
			optionKey:      "nullable",
			optionValue:    "false",
			expectNullable: new(false),
		},
		{
			name:          "columns sets sortable columns",
			optionKey:     "columns",
			optionValue:   "name,email",
			expectColumns: []string{"name", "email"},
		},
		{
			name:          "columns with spaces are trimmed",
			optionKey:     "columns",
			optionValue:   "name, email, age",
			expectColumns: []string{"name", "email", "age"},
		},
		{
			name:          "default sets default value",
			optionKey:     "default",
			optionValue:   "10",
			expectDefault: new(10),
		},
		{
			name:        "max sets max value",
			optionKey:   "max",
			optionValue: "100",
			expectMax:   new(100),
		},
		{
			name:        "unknown option key is silently ignored",
			optionKey:   "unknown",
			optionValue: "whatever",
		},
		{
			name:        "default with non-numeric value is ignored",
			optionKey:   "default",
			optionValue: "abc",
		},
		{
			name:        "max with non-numeric value is ignored",
			optionKey:   "max",
			optionValue: "xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			directive := &querier_dto.ParameterDirective{}
			option := &querier_dto.DirectiveOption{
				Key:   tt.optionKey,
				Value: tt.optionValue,
			}

			resolveOption(directive, option)

			if tt.expectTypeHint != nil {
				require.NotNil(t, directive.TypeHint, "expected TypeHint to be set")
				assert.Equal(t, *tt.expectTypeHint, *directive.TypeHint)
			} else {
				assert.Nil(t, directive.TypeHint, "expected TypeHint to remain nil")
			}

			if tt.expectNullable != nil {
				require.NotNil(t, directive.Nullable, "expected Nullable to be set")
				assert.Equal(t, *tt.expectNullable, *directive.Nullable)
			} else {
				assert.Nil(t, directive.Nullable, "expected Nullable to remain nil")
			}

			if tt.expectColumns != nil {
				assert.Equal(t, tt.expectColumns, directive.Columns)
			} else {
				assert.Nil(t, directive.Columns, "expected Columns to remain nil")
			}

			if tt.expectDefault != nil {
				require.NotNil(t, directive.DefaultVal, "expected DefaultVal to be set")
				assert.Equal(t, *tt.expectDefault, *directive.DefaultVal)
			} else {
				assert.Nil(t, directive.DefaultVal, "expected DefaultVal to remain nil")
			}

			if tt.expectMax != nil {
				require.NotNil(t, directive.MaxVal, "expected MaxVal to be set")
				assert.Equal(t, *tt.expectMax, *directive.MaxVal)
			} else {
				assert.Nil(t, directive.MaxVal, "expected MaxVal to remain nil")
			}
		})
	}
}

func TestParseQueryDirectives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		metadata               []*querier_dto.MetadataDirective
		expectNullableOverride *bool
		expectReadOnlyOverride *bool
		expectDynamicRuntime   bool
		expectGroupByKeys      []string
	}{
		{
			name: "piko.nullable sets nullable override to true",
			metadata: []*querier_dto.MetadataDirective{
				{Directive: "nullable", Value: "true"},
			},
			expectNullableOverride: new(true),
		},
		{
			name: "piko.nullable false sets nullable override to false",
			metadata: []*querier_dto.MetadataDirective{
				{Directive: "nullable", Value: "false"},
			},
			expectNullableOverride: new(false),
		},
		{
			name: "piko.readonly sets readonly override to true",
			metadata: []*querier_dto.MetadataDirective{
				{Directive: "readonly", Value: "true"},
			},
			expectReadOnlyOverride: new(true),
		},
		{
			name: "piko.readonly false sets readonly override to false",
			metadata: []*querier_dto.MetadataDirective{
				{Directive: "readonly", Value: "false"},
			},
			expectReadOnlyOverride: new(false),
		},
		{
			name: "piko.runtime true sets dynamic runtime",
			metadata: []*querier_dto.MetadataDirective{
				{Directive: "runtime", Value: "true"},
			},
			expectDynamicRuntime: true,
		},
		{
			name: "piko.runtime false does not set dynamic runtime",
			metadata: []*querier_dto.MetadataDirective{
				{Directive: "runtime", Value: "false"},
			},
			expectDynamicRuntime: false,
		},
		{
			name: "piko.dynamic runtime sets dynamic runtime",
			metadata: []*querier_dto.MetadataDirective{
				{Directive: "dynamic", Value: "runtime"},
			},
			expectDynamicRuntime: true,
		},
		{
			name: "piko.group_by sets group by keys",
			metadata: []*querier_dto.MetadataDirective{
				{Directive: "group_by", Value: "author_id"},
			},
			expectGroupByKeys: []string{"author_id"},
		},
		{
			name: "multiple piko.group_by directives accumulate",
			metadata: []*querier_dto.MetadataDirective{
				{Directive: "group_by", Value: "author_id"},
				{Directive: "group_by", Value: "category"},
			},
			expectGroupByKeys: []string{"author_id", "category"},
		},
		{
			name:     "empty metadata produces empty directives",
			metadata: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			block := &querier_dto.DirectiveBlock{
				Metadata: tt.metadata,
			}

			directives := extractQueryDirectives(block)

			require.NotNil(t, directives, "directives should never be nil")

			if tt.expectNullableOverride != nil {
				require.NotNil(t, directives.NullableOverride,
					"expected NullableOverride to be set")
				assert.Equal(t, *tt.expectNullableOverride, *directives.NullableOverride)
			} else {
				assert.Nil(t, directives.NullableOverride,
					"expected NullableOverride to remain nil")
			}

			if tt.expectReadOnlyOverride != nil {
				require.NotNil(t, directives.ReadOnlyOverride,
					"expected ReadOnlyOverride to be set")
				assert.Equal(t, *tt.expectReadOnlyOverride, *directives.ReadOnlyOverride)
			} else {
				assert.Nil(t, directives.ReadOnlyOverride,
					"expected ReadOnlyOverride to remain nil")
			}

			assert.Equal(t, tt.expectDynamicRuntime, directives.DynamicRuntime)

			if tt.expectGroupByKeys != nil {
				assert.Equal(t, tt.expectGroupByKeys, directives.GroupByKeys)
			} else {
				assert.Nil(t, directives.GroupByKeys,
					"expected GroupByKeys to remain nil")
			}
		})
	}
}

func TestDirectiveParser_Parse_NumberedParameters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		expectParams   []expectedParam
		expectDiagMsgs []string
	}{
		{
			name: "single numbered parameter",
			input: "-- piko.name: GetUser\n" +
				"-- piko.command: one\n" +
				"-- $1 as piko.param(user_id)\n",
			expectParams: []expectedParam{
				{number: 1, name: "user_id", kind: querier_dto.ParameterDirectiveParam},
			},
		},
		{
			name: "multiple numbered parameters",
			input: "-- piko.name: CreateUser\n" +
				"-- piko.command: exec\n" +
				"-- $1 as piko.param(name)\n" +
				"-- $2 as piko.param(email)\n" +
				"-- $3 as piko.param(age)\n",
			expectParams: []expectedParam{
				{number: 1, name: "name", kind: querier_dto.ParameterDirectiveParam},
				{number: 2, name: "email", kind: querier_dto.ParameterDirectiveParam},
				{number: 3, name: "age", kind: querier_dto.ParameterDirectiveParam},
			},
		},
		{
			name: "parameter with type hint",
			input: "-- piko.name: GetById\n" +
				"-- piko.command: one\n" +
				"-- $1 as piko.param(id) type:integer\n",
			expectParams: []expectedParam{
				{number: 1, name: "id", kind: querier_dto.ParameterDirectiveParam, typeHint: new("integer")},
			},
		},
		{
			name: "optional parameter",
			input: "-- piko.name: Search\n" +
				"-- piko.command: many\n" +
				"-- $1 as piko.optional(search)\n",
			expectParams: []expectedParam{
				{number: 1, name: "search", kind: querier_dto.ParameterDirectiveOptional},
			},
		},
		{
			name: "limit parameter with default and max",
			input: "-- piko.name: ListPaged\n" +
				"-- piko.command: many\n" +
				"-- $1 as piko.limit(page_size) default:25 max:100\n",
			expectParams: []expectedParam{
				{
					number:     1,
					name:       "page_size",
					kind:       querier_dto.ParameterDirectiveLimit,
					defaultVal: new(25),
					maxVal:     new(100),
				},
			},
		},
		{
			name: "sortable parameter with columns",
			input: "-- piko.name: SortedList\n" +
				"-- piko.command: many\n" +
				"-- $1 as piko.sortable(sort_col) columns:name,email,created_at\n",
			expectParams: []expectedParam{
				{
					number:  1,
					name:    "sort_col",
					kind:    querier_dto.ParameterDirectiveSortable,
					columns: []string{"name", "email", "created_at"},
				},
			},
		},
		{
			name: "parameter with nullable option",
			input: "-- piko.name: NullableSearch\n" +
				"-- piko.command: many\n" +
				"-- $1 as piko.param(term) nullable:true\n",
			expectParams: []expectedParam{
				{number: 1, name: "term", kind: querier_dto.ParameterDirectiveParam, nullable: new(true)},
			},
		},
		{
			name: "slice parameter",
			input: "-- piko.name: GetByIds\n" +
				"-- piko.command: many\n" +
				"-- $1 as piko.slice(ids)\n",
			expectParams: []expectedParam{
				{number: 1, name: "ids", kind: querier_dto.ParameterDirectiveSlice},
			},
		},
		{
			name: "offset parameter",
			input: "-- piko.name: Paginate\n" +
				"-- piko.command: many\n" +
				"-- $1 as piko.offset(page_offset) default:0\n",
			expectParams: []expectedParam{
				{
					number:     1,
					name:       "page_offset",
					kind:       querier_dto.ParameterDirectiveOffset,
					defaultVal: new(0),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := newTestParser()
			block := queryBlock{sql: tt.input, startLine: 1}

			result, diagnostics := parser.Parse(block, "test.sql")

			for _, diag := range diagnostics {
				if diag.Code == querier_dto.CodeDirectiveSyntax {
					t.Errorf("unexpected syntax diagnostic: %s", diag.Message)
				}
			}

			require.Len(t, result.Parameters, len(tt.expectParams),
				"parameter count mismatch")

			for i, expected := range tt.expectParams {
				param := result.Parameters[i]
				assert.Equal(t, expected.number, param.Number,
					"param[%d] number mismatch", i)
				assert.Equal(t, expected.name, param.Name,
					"param[%d] name mismatch", i)
				assert.Equal(t, expected.kind, param.Kind,
					"param[%d] kind mismatch", i)

				if expected.typeHint != nil {
					require.NotNil(t, param.TypeHint,
						"param[%d] expected TypeHint to be set", i)
					assert.Equal(t, *expected.typeHint, *param.TypeHint)
				}

				if expected.nullable != nil {
					require.NotNil(t, param.Nullable,
						"param[%d] expected Nullable to be set", i)
					assert.Equal(t, *expected.nullable, *param.Nullable)
				}

				if expected.defaultVal != nil {
					require.NotNil(t, param.DefaultVal,
						"param[%d] expected DefaultVal to be set", i)
					assert.Equal(t, *expected.defaultVal, *param.DefaultVal)
				}

				if expected.maxVal != nil {
					require.NotNil(t, param.MaxVal,
						"param[%d] expected MaxVal to be set", i)
					assert.Equal(t, *expected.maxVal, *param.MaxVal)
				}

				if expected.columns != nil {
					assert.Equal(t, expected.columns, param.Columns,
						"param[%d] columns mismatch", i)
				}
			}
		})
	}
}

func TestDirectiveParser_Parse_NamedParameters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		expectParams []expectedNamedParam
	}{
		{
			name: "colon-prefixed named parameter",
			input: "-- piko.name: FindByEmail\n" +
				"-- piko.command: one\n" +
				"-- :email as piko.param\n",
			expectParams: []expectedNamedParam{
				{
					number:        1,
					name:          "email",
					directiveName: "email",
					kind:          querier_dto.ParameterDirectiveParam,
					isNamed:       true,
				},
			},
		},
		{
			name: "at-prefixed named parameter",
			input: "-- piko.name: FindByName\n" +
				"-- piko.command: one\n" +
				"-- @name as piko.optional\n",
			expectParams: []expectedNamedParam{
				{
					number:        1,
					name:          "name",
					directiveName: "name",
					kind:          querier_dto.ParameterDirectiveOptional,
					isNamed:       true,
				},
			},
		},
		{
			name: "named parameter with explicit name override",
			input: "-- piko.name: FindByEmail\n" +
				"-- piko.command: one\n" +
				"-- :email as piko.param(user_email)\n",
			expectParams: []expectedNamedParam{
				{
					number:        1,
					name:          "user_email",
					directiveName: "email",
					kind:          querier_dto.ParameterDirectiveParam,
					isNamed:       true,
				},
			},
		},
		{
			name: "multiple named parameters get sequential numbers",
			input: "-- piko.name: Search\n" +
				"-- piko.command: many\n" +
				"-- :email as piko.param\n" +
				"-- :name as piko.optional\n",
			expectParams: []expectedNamedParam{
				{
					number:        1,
					name:          "email",
					directiveName: "email",
					kind:          querier_dto.ParameterDirectiveParam,
					isNamed:       true,
				},
				{
					number:        2,
					name:          "name",
					directiveName: "name",
					kind:          querier_dto.ParameterDirectiveOptional,
					isNamed:       true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := newTestParserWithNamedPrefix()
			block := queryBlock{sql: tt.input, startLine: 1}

			result, diagnostics := parser.Parse(block, "test.sql")

			for _, diag := range diagnostics {
				if diag.Code == querier_dto.CodeDirectiveSyntax {
					t.Errorf("unexpected syntax diagnostic: %s", diag.Message)
				}
			}

			require.Len(t, result.Parameters, len(tt.expectParams),
				"parameter count mismatch")

			for i, expected := range tt.expectParams {
				param := result.Parameters[i]
				assert.Equal(t, expected.number, param.Number,
					"param[%d] number mismatch", i)
				assert.Equal(t, expected.name, param.Name,
					"param[%d] name mismatch", i)
				assert.Equal(t, expected.directiveName, param.DirectiveName,
					"param[%d] directive name mismatch", i)
				assert.Equal(t, expected.kind, param.Kind,
					"param[%d] kind mismatch", i)
				assert.Equal(t, expected.isNamed, param.IsNamed,
					"param[%d] IsNamed mismatch", i)
			}
		})
	}
}

func TestParseCommandValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		value         string
		expectCommand querier_dto.QueryCommand
		expectError   bool
	}{
		{
			name:          "valid one command",
			value:         "one",
			expectCommand: querier_dto.QueryCommandOne,
		},
		{
			name:          "valid many command",
			value:         "many",
			expectCommand: querier_dto.QueryCommandMany,
		},
		{
			name:          "valid exec command",
			value:         "exec",
			expectCommand: querier_dto.QueryCommandExec,
		},
		{
			name:          "valid execresult command",
			value:         "execresult",
			expectCommand: querier_dto.QueryCommandExecResult,
		},
		{
			name:          "valid execrows command",
			value:         "execrows",
			expectCommand: querier_dto.QueryCommandExecRows,
		},
		{
			name:          "valid batch command",
			value:         "batch",
			expectCommand: querier_dto.QueryCommandBatch,
		},
		{
			name:          "valid stream command",
			value:         "stream",
			expectCommand: querier_dto.QueryCommandStream,
		},
		{
			name:          "valid copyfrom command",
			value:         "copyfrom",
			expectCommand: querier_dto.QueryCommandCopyFrom,
		},
		{
			name:        "unknown command produces error",
			value:       "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lineSpan := querier_dto.TextSpan{Line: 1, Column: 1, EndLine: 1, EndColumn: 30}
			keySpan := querier_dto.TextSpan{Line: 1, Column: 1, EndLine: 1, EndColumn: 15}
			valueSpan := querier_dto.TextSpan{Line: 1, Column: 16, EndLine: 1, EndColumn: 30}

			directive, parseError := parseCommandValue(
				tt.value, lineSpan, keySpan, valueSpan, 1, "test.sql",
			)

			if tt.expectError {
				require.NotNil(t, parseError,
					"expected an error for unknown command %q", tt.value)
				assert.Contains(t, parseError.Message, tt.value,
					"error message should contain the unknown command value")
				assert.Equal(t, querier_dto.CodeDirectiveSyntax, parseError.Code)
				assert.Nil(t, directive, "directive should be nil on error")
			} else {
				assert.Nil(t, parseError, "expected no error for valid command %q", tt.value)
				require.NotNil(t, directive, "directive should not be nil for valid command")
				assert.Equal(t, tt.expectCommand, directive.Command)
				assert.Equal(t, tt.value, directive.Value)
			}
		})
	}
}

func TestSetBoolOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected *bool
	}{
		{
			name:     "true sets pointer to true",
			value:    "true",
			expected: new(true),
		},
		{
			name:     "false sets pointer to false",
			value:    "false",
			expected: new(false),
		},
		{
			name:     "unrecognised value leaves pointer nil",
			value:    "maybe",
			expected: nil,
		},
		{
			name:     "empty string leaves pointer nil",
			value:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var target *bool
			setBoolOverride(&target, tt.value)

			if tt.expected != nil {
				require.NotNil(t, target, "expected target to be set")
				assert.Equal(t, *tt.expected, *target)
			} else {
				assert.Nil(t, target, "expected target to remain nil")
			}
		})
	}
}

func TestSyntaxError(t *testing.T) {
	t.Parallel()

	result := syntaxError("queries/test.sql", 10, 5, "unexpected token")

	assert.Equal(t, "queries/test.sql", result.Filename)
	assert.Equal(t, 10, result.Line)
	assert.Equal(t, 5, result.Column)
	assert.Equal(t, "unexpected token", result.Message)
	assert.Equal(t, querier_dto.SeverityError, result.Severity)
	assert.Equal(t, querier_dto.CodeDirectiveSyntax, result.Code)
}

func TestDirectiveParser_Parse_BlockSpan(t *testing.T) {
	t.Parallel()

	parser := newTestParser()
	input := "-- piko.name: GetUser\n-- piko.command: one\n"
	block := queryBlock{sql: input, startLine: 5}

	result, _ := parser.Parse(block, "test.sql")

	assert.Equal(t, 5, result.Span.Line, "span should start at the first comment line")
	assert.Equal(t, 1, result.Span.Column, "span column should always be 1")
	assert.Equal(t, 6, result.Span.EndLine, "span should end at the last comment line")
}

func TestDirectiveParser_Parse_EmptyCommentLines(t *testing.T) {
	t.Parallel()

	parser := newTestParser()
	input := "-- piko.name: GetUser\n--\n-- piko.command: one\n"
	block := queryBlock{sql: input, startLine: 1}

	result, diagnostics := parser.Parse(block, "test.sql")

	require.NotNil(t, result.Name, "name should be parsed despite empty comment line")
	assert.Equal(t, "GetUser", result.Name.Value)
	require.NotNil(t, result.Command, "command should be parsed despite empty comment line")
	assert.Equal(t, querier_dto.QueryCommandOne, result.Command.Command)

	for _, diag := range diagnostics {
		if diag.Code == querier_dto.CodeDirectiveSyntax {
			t.Errorf("unexpected syntax diagnostic: %s", diag.Message)
		}
	}
}

func TestDirectiveParser_Parse_MetadataDirectives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		expectMetadata []expectedMetadata
	}{
		{
			name: "bare directive treated as boolean true",
			input: "-- piko.name: ListPosts\n" +
				"-- piko.command: many\n" +
				"-- piko.nullable\n",
			expectMetadata: []expectedMetadata{
				{directive: "nullable", value: "true"},
			},
		},
		{
			name: "colon directive with value",
			input: "-- piko.name: GroupedPosts\n" +
				"-- piko.command: many\n" +
				"-- piko.group_by: author_id\n",
			expectMetadata: []expectedMetadata{
				{directive: "group_by", value: "author_id"},
			},
		},
		{
			name: "parenthesised directive",
			input: "-- piko.name: EmbedTest\n" +
				"-- piko.command: one\n" +
				"-- piko.embed(users)\n",
			expectMetadata: []expectedMetadata{
				{directive: "embed", value: "users"},
			},
		},
		{
			name: "multiple metadata directives",
			input: "-- piko.name: FullTest\n" +
				"-- piko.command: many\n" +
				"-- piko.nullable\n" +
				"-- piko.readonly\n" +
				"-- piko.group_by: category\n",
			expectMetadata: []expectedMetadata{
				{directive: "nullable", value: "true"},
				{directive: "readonly", value: "true"},
				{directive: "group_by", value: "category"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := newTestParser()
			block := queryBlock{sql: tt.input, startLine: 1}

			result, _ := parser.Parse(block, "test.sql")

			require.Len(t, result.Metadata, len(tt.expectMetadata),
				"metadata count mismatch")

			for i, expected := range tt.expectMetadata {
				assert.Equal(t, expected.directive, result.Metadata[i].Directive,
					"metadata[%d] directive mismatch", i)
				assert.Equal(t, expected.value, result.Metadata[i].Value,
					"metadata[%d] value mismatch", i)
			}
		})
	}
}

func TestDirectiveParser_Parse_DiagnosticFilename(t *testing.T) {
	t.Parallel()

	parser := newTestParser()

	input := "-- some unrecognised line\n"
	block := queryBlock{sql: input, startLine: 1}

	_, diagnostics := parser.Parse(block, "queries/users/get.sql")

	require.NotEmpty(t, diagnostics, "expected at least one diagnostic")
	for _, diag := range diagnostics {
		assert.Equal(t, "queries/users/get.sql", diag.Filename,
			"diagnostic filename should match the provided filename")
	}
}

func TestDirectiveParser_Parse_StartLineOffset(t *testing.T) {
	t.Parallel()

	parser := newTestParser()
	input := "-- piko.name: GetUser\n-- piko.command: one\n"
	block := queryBlock{sql: input, startLine: 42}

	result, _ := parser.Parse(block, "test.sql")

	require.NotNil(t, result.Name, "name should be parsed")
	assert.Equal(t, 42, result.Name.Span.Line,
		"name directive span should use the provided startLine offset")
	require.NotNil(t, result.Command, "command should be parsed")
	assert.Equal(t, 43, result.Command.Span.Line,
		"command directive span should be offset from startLine")
}

type expectedParam struct {
	number     int
	name       string
	kind       querier_dto.ParameterDirectiveKind
	typeHint   *string
	nullable   *bool
	defaultVal *int
	maxVal     *int
	columns    []string
}

type expectedNamedParam struct {
	number        int
	name          string
	directiveName string
	kind          querier_dto.ParameterDirectiveKind
	isNamed       bool
}

type expectedMetadata struct {
	directive string
	value     string
}

func commandPtr(command querier_dto.QueryCommand) *querier_dto.QueryCommand {
	return new(command)
}
