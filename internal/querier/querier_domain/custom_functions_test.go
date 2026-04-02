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

func TestMergeCustomFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		catalogue       *querier_dto.Catalogue
		customFunctions []querier_dto.CustomFunctionConfig
		expectFunctions map[string]int
	}{
		{
			name:            "empty list leaves catalogue unchanged",
			catalogue:       newTestCatalogue("public"),
			customFunctions: nil,
			expectFunctions: map[string]int{},
		},
		{
			name:      "adds function to default schema",
			catalogue: newTestCatalogue("public"),
			customFunctions: []querier_dto.CustomFunctionConfig{
				{
					Name:       "my_func",
					ReturnType: "integer",
					Arguments:  []string{"text"},
				},
			},
			expectFunctions: map[string]int{
				"my_func": 1,
			},
		},
		{
			name: "nil default schema does not panic",
			catalogue: &querier_dto.Catalogue{
				DefaultSchema: "missing",
				Schemas:       map[string]*querier_dto.Schema{},
			},
			customFunctions: []querier_dto.CustomFunctionConfig{
				{
					Name:       "some_func",
					ReturnType: "text",
				},
			},
			expectFunctions: nil,
		},
		{
			name:      "multiple functions are all added",
			catalogue: newTestCatalogue("public"),
			customFunctions: []querier_dto.CustomFunctionConfig{
				{
					Name:       "func_a",
					ReturnType: "integer",
				},
				{
					Name:       "func_b",
					ReturnType: "text",
					Arguments:  []string{"integer", "text"},
				},
				{
					Name:       "func_c",
					ReturnType: "boolean",
				},
			},
			expectFunctions: map[string]int{
				"func_a": 1,
				"func_b": 1,
				"func_c": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			engine := &mockEngine{}

			mergeCustomFunctions(tt.catalogue, engine, tt.customFunctions)

			if tt.expectFunctions == nil {

				return
			}

			schema := tt.catalogue.Schemas[tt.catalogue.DefaultSchema]
			require.NotNil(t, schema, "default schema should exist")

			for functionName, expectedCount := range tt.expectFunctions {
				signatures, exists := schema.Functions[functionName]
				assert.True(t, exists, "function %q should be present", functionName)
				assert.Len(t, signatures, expectedCount,
					"function %q should have %d overload(s)", functionName, expectedCount)
			}

			assert.Len(t, schema.Functions, len(tt.expectFunctions),
				"schema should contain exactly the expected number of functions")
		})
	}
}

func TestConvertCustomFunction(t *testing.T) {
	t.Parallel()

	engine := &mockEngine{}

	tests := []struct {
		name            string
		config          querier_dto.CustomFunctionConfig
		expectNil       bool
		expectName      string
		expectArgs      int
		expectMinArgs   int
		expectAggregate bool
		expectVariadic  bool
	}{
		{
			name: "empty name returns nil",
			config: querier_dto.CustomFunctionConfig{
				Name:       "",
				ReturnType: "integer",
			},
			expectNil: true,
		},
		{
			name: "empty return type returns nil",
			config: querier_dto.CustomFunctionConfig{
				Name:       "my_func",
				ReturnType: "",
			},
			expectNil: true,
		},
		{
			name: "valid with arguments maps arguments correctly",
			config: querier_dto.CustomFunctionConfig{
				Name:       "concat_ws",
				ReturnType: "text",
				Arguments:  []string{"text", "integer"},
			},
			expectNil:     false,
			expectName:    "concat_ws",
			expectArgs:    2,
			expectMinArgs: 2,
		},
		{
			name: "MinArguments explicitly set uses provided value",
			config: querier_dto.CustomFunctionConfig{
				Name:         "variadic_fn",
				ReturnType:   "text",
				Arguments:    []string{"text", "integer", "boolean"},
				MinArguments: 1,
			},
			expectNil:     false,
			expectName:    "variadic_fn",
			expectArgs:    3,
			expectMinArgs: 1,
		},
		{
			name: "MinArguments zero defaults to len of arguments",
			config: querier_dto.CustomFunctionConfig{
				Name:         "all_required",
				ReturnType:   "integer",
				Arguments:    []string{"text", "text"},
				MinArguments: 0,
			},
			expectNil:     false,
			expectName:    "all_required",
			expectArgs:    2,
			expectMinArgs: 2,
		},
		{
			name: "aggregate function sets IsAggregate",
			config: querier_dto.CustomFunctionConfig{
				Name:        "my_sum",
				ReturnType:  "numeric",
				Arguments:   []string{"numeric"},
				IsAggregate: true,
			},
			expectNil:       false,
			expectName:      "my_sum",
			expectArgs:      1,
			expectMinArgs:   1,
			expectAggregate: true,
		},
		{
			name: "variadic function sets IsVariadic",
			config: querier_dto.CustomFunctionConfig{
				Name:       "format_str",
				ReturnType: "text",
				Arguments:  []string{"text"},
				IsVariadic: true,
			},
			expectNil:      false,
			expectName:     "format_str",
			expectArgs:     1,
			expectMinArgs:  1,
			expectVariadic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := convertCustomFunction(tt.config, engine)

			if tt.expectNil {
				assert.Nil(t, result, "expected nil result for invalid config")
				return
			}

			require.NotNil(t, result, "expected non-nil function signature")
			assert.Equal(t, tt.expectName, result.Name)
			assert.Len(t, result.Arguments, tt.expectArgs)
			assert.Equal(t, tt.expectMinArgs, result.MinArguments)
			assert.Equal(t, tt.expectAggregate, result.IsAggregate)
			assert.Equal(t, tt.expectVariadic, result.IsVariadic)

			for argIndex, arg := range result.Arguments {
				expectedArgName := "arg" + string(rune('1'+argIndex))
				if argIndex >= 9 {

					assert.Contains(t, arg.Name, "arg")
				} else {
					assert.Equal(t, expectedArgName, arg.Name,
						"argument %d should be named %q", argIndex, expectedArgName)
				}

				assert.Equal(t, tt.config.Arguments[argIndex], arg.Type.EngineName,
					"argument type should be normalised via the engine")
			}

			assert.Equal(t, tt.config.ReturnType, result.ReturnType.EngineName,
				"return type should be normalised via the engine")
		})
	}
}

func TestParseNullableBehaviour(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected querier_dto.FunctionNullableBehaviour
	}{
		{
			name:     "never_null returns FunctionNullableNeverNull",
			input:    "never_null",
			expected: querier_dto.FunctionNullableNeverNull,
		},
		{
			name:     "called_on_null returns FunctionNullableCalledOnNull",
			input:    "called_on_null",
			expected: querier_dto.FunctionNullableCalledOnNull,
		},
		{
			name:     "returns_null_on_null returns FunctionNullableReturnsNullOnNull",
			input:    "returns_null_on_null",
			expected: querier_dto.FunctionNullableReturnsNullOnNull,
		},
		{
			name:     "empty string returns FunctionNullableReturnsNullOnNull",
			input:    "",
			expected: querier_dto.FunctionNullableReturnsNullOnNull,
		},
		{
			name:     "unknown value returns FunctionNullableReturnsNullOnNull",
			input:    "something_unexpected",
			expected: querier_dto.FunctionNullableReturnsNullOnNull,
		},
		{
			name:     "case insensitive NEVER_NULL returns FunctionNullableNeverNull",
			input:    "NEVER_NULL",
			expected: querier_dto.FunctionNullableNeverNull,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := parseNullableBehaviour(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
