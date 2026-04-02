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

package compiler_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTypeAssertions_BasicTypes(t *testing.T) {
	code := `
		const state = {
			count: 0 as number,
			message: "hello" as string,
			active: true as boolean
		};
	`

	assertions := ExtractTypeAssertions(code)

	assert.Equal(t, 3, len(assertions))
	assert.Equal(t, "number", assertions["count"].TypeString)
	assert.Equal(t, "string", assertions["message"].TypeString)
	assert.Equal(t, "boolean", assertions["active"].TypeString)
}

func TestExtractTypeAssertions_Arrays(t *testing.T) {
	code := `
		const state = {
			items: [] as string[],
			numbers: [1, 2, 3] as number[]
		};
	`

	assertions := ExtractTypeAssertions(code)

	assert.Equal(t, 2, len(assertions))
	assert.Equal(t, "string[]", assertions["items"].TypeString)
	assert.Equal(t, "number[]", assertions["numbers"].TypeString)
}

func TestExtractTypeAssertions_Generics(t *testing.T) {
	code := `
		const state = {
			items: [] as Array<string>,
			map: null as Map<string, number>,
			set: new Set() as Set<User>
		};
	`

	assertions := ExtractTypeAssertions(code)

	assert.Equal(t, 3, len(assertions))
	assert.Equal(t, "Array<string>", assertions["items"].TypeString)
	assert.Equal(t, "Map<string,number>", assertions["map"].TypeString)
	assert.Equal(t, "Set<User>", assertions["set"].TypeString)
}

func TestExtractTypeAssertions_NestedGenerics(t *testing.T) {
	code := `
		const state = {
			data: null as Map<string, Array<number>>
		};
	`

	assertions := ExtractTypeAssertions(code)

	assert.Equal(t, 1, len(assertions))
	assert.Equal(t, "Map<string,Array<number>>", assertions["data"].TypeString)
}

func TestExtractTypeAssertions_UnionTypes(t *testing.T) {
	code := `
		const state = {
			optional: null as User | null,
			value: undefined as string | number
		};
	`

	assertions := ExtractTypeAssertions(code)

	assert.Equal(t, 2, len(assertions))
	assert.Contains(t, assertions["optional"].TypeString, "User")
	assert.Contains(t, assertions["optional"].TypeString, "null")
}

func TestExtractTypeAssertions_IgnoresStrings(t *testing.T) {
	code := `
		const state = {
			message: "as number" as string,
			code: 'as boolean' as string
		};
	`

	assertions := ExtractTypeAssertions(code)

	assert.Equal(t, 2, len(assertions))
	assert.Equal(t, "string", assertions["message"].TypeString)
	assert.Equal(t, "string", assertions["code"].TypeString)
}

func TestExtractTypeAssertions_IgnoresComments(t *testing.T) {
	code := `
		const state = {
			count: 0 as number,
			// message: "test" as string,
			active: true as boolean
		};
	`

	assertions := ExtractTypeAssertions(code)

	assert.Equal(t, 2, len(assertions))
	assert.Contains(t, assertions, "count")
	assert.Contains(t, assertions, "active")
	assert.NotContains(t, assertions, "message")
}

func TestExtractTypeAssertions_MixedTyping(t *testing.T) {
	code := `
		const state = {
			explicit: 42 as number,
			inferred: "hello",
			typed: [] as string[]
		};
	`

	assertions := ExtractTypeAssertions(code)

	assert.Equal(t, 2, len(assertions))
	assert.Equal(t, "number", assertions["explicit"].TypeString)
	assert.Equal(t, "string[]", assertions["typed"].TypeString)
	assert.NotContains(t, assertions, "inferred")
}

func TestExtractTypeAssertions_NoState(t *testing.T) {
	code := `
		const other = "not state";
	`

	assertions := ExtractTypeAssertions(code)

	assert.Equal(t, 0, len(assertions))
}

func TestExtractTypeAssertions_EmptyState(t *testing.T) {
	code := `
		const state = {};
	`

	assertions := ExtractTypeAssertions(code)

	assert.Equal(t, 0, len(assertions))
}

func TestExtractTypeAssertions_ComplexRealWorld(t *testing.T) {
	code := `
		const state = {
			// Basic types
			count: 0 as number,
			title: "App" as string,
			loading: false as boolean,

			// Arrays
			items: [] as string[],
			users: [] as Array<User>,

			// Complex types
			cache: new Map() as Map<string, Data>,
			optional: null as User | null,

			// Nested generics
			nestedMap: null as Map<string, Array<number>>
		};

		function increment() {
			state.count++;
		}
	`

	assertions := ExtractTypeAssertions(code)

	assert.Equal(t, 8, len(assertions))
	assert.Equal(t, "number", assertions["count"].TypeString)
	assert.Equal(t, "string", assertions["title"].TypeString)
	assert.Equal(t, "boolean", assertions["loading"].TypeString)
	assert.Equal(t, "string[]", assertions["items"].TypeString)
	assert.Equal(t, "Array<User>", assertions["users"].TypeString)
	assert.Equal(t, "Map<string,Data>", assertions["cache"].TypeString)
	assert.Contains(t, assertions["optional"].TypeString, "User")
	assert.Equal(t, "Map<string,Array<number>>", assertions["nestedMap"].TypeString)
}

func TestParseTypeString_Arrays(t *testing.T) {
	tests := []struct {
		input        string
		expectedType string
		expectedElem string
	}{
		{input: "string[]", expectedType: "array", expectedElem: "string"},
		{input: "number[]", expectedType: "array", expectedElem: "number"},
		{input: "Array<string>", expectedType: "array", expectedElem: "string"},
		{input: "Array<User>", expectedType: "array", expectedElem: "user"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseTypeString(tt.input)
			assert.Equal(t, tt.expectedType, result.JSType)
			assert.Equal(t, tt.expectedElem, result.ElementType)
		})
	}
}

func TestParseTypeString_Primitives(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "number", expected: "number"},
		{input: "string", expected: "string"},
		{input: "boolean", expected: "boolean"},
		{input: "any", expected: "any"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseTypeString(tt.input)
			assert.Equal(t, tt.expected, result.JSType)
		})
	}
}

func TestParseTypeString_Generics(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedType string
		expectedElem string
		expectedKey  string
		expectedVal  string
	}{
		{
			name:         "Array<string>",
			input:        "Array<string>",
			expectedType: "array",
			expectedElem: "string",
		},
		{
			name:         "Map<string,number>",
			input:        "Map<string,number>",
			expectedType: "object",
			expectedKey:  "string",
			expectedVal:  "number",
		},
		{
			name:         "Set<User>",
			input:        "Set<User>",
			expectedType: "object",
			expectedElem: "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTypeString(tt.input)
			assert.Equal(t, tt.expectedType, result.JSType)
			if tt.expectedElem != "" {
				assert.Equal(t, tt.expectedElem, result.ElementType)
			}
			if tt.expectedKey != "" {
				assert.Equal(t, tt.expectedKey, result.KeyType)
			}
			if tt.expectedVal != "" {
				assert.Equal(t, tt.expectedVal, result.ValueType)
			}
		})
	}
}

func TestParseTypeString_Nullable(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedType     string
		expectedNullable bool
	}{
		{
			name:             "string | null",
			input:            "string | null",
			expectedType:     "string",
			expectedNullable: true,
		},
		{
			name:             "number | undefined",
			input:            "number | undefined",
			expectedType:     "number",
			expectedNullable: true,
		},
		{
			name:             "User | null",
			input:            "User | null",
			expectedType:     "user",
			expectedNullable: true,
		},
		{
			name:             "string | null | undefined",
			input:            "string | null | undefined",
			expectedType:     "string",
			expectedNullable: true,
		},
		{
			name:             "plain string (not nullable)",
			input:            "string",
			expectedType:     "string",
			expectedNullable: false,
		},
		{
			name:             "string | number (union without null)",
			input:            "string | number",
			expectedType:     "string",
			expectedNullable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTypeString(tt.input)
			assert.Equal(t, tt.expectedType, result.JSType, "type mismatch")
			assert.Equal(t, tt.expectedNullable, result.IsNullable, "nullable mismatch")
		})
	}
}

func TestParseTypeString_NullableArrays(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedType     string
		expectedElem     string
		expectedNullable bool
	}{
		{
			name:             "string[] | null",
			input:            "string[] | null",
			expectedType:     "array",
			expectedElem:     "string",
			expectedNullable: true,
		},
		{
			name:             "Array<User> | null",
			input:            "Array<User> | null",
			expectedType:     "array",
			expectedElem:     "user",
			expectedNullable: true,
		},
		{
			name:             "string[] (not nullable)",
			input:            "string[]",
			expectedType:     "array",
			expectedElem:     "string",
			expectedNullable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTypeString(tt.input)
			assert.Equal(t, tt.expectedType, result.JSType, "type mismatch")
			assert.Equal(t, tt.expectedElem, result.ElementType, "element type mismatch")
			assert.Equal(t, tt.expectedNullable, result.IsNullable, "nullable mismatch")
		})
	}
}
