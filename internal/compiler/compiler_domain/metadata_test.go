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
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestNewComponentMetadata(t *testing.T) {
	t.Run("creates empty metadata with initialised maps", func(t *testing.T) {
		metadata := NewComponentMetadata()

		require.NotNil(t, metadata)
		require.NotNil(t, metadata.StateProperties)
		require.NotNil(t, metadata.Methods)
		require.NotNil(t, metadata.BooleanProps)
		assert.Empty(t, metadata.StateProperties)
		assert.Empty(t, metadata.Methods)
		assert.Empty(t, metadata.BooleanProps)
	})

	t.Run("maps are usable immediately", func(t *testing.T) {
		metadata := NewComponentMetadata()
		metadata.StateProperties["count"] = &PropertyMetadata{Name: "count", JSType: "number"}
		metadata.Methods["increment"] = &MethodMetadata{Name: "increment"}
		metadata.BooleanProps = append(metadata.BooleanProps, "active")

		assert.Len(t, metadata.StateProperties, 1)
		assert.Len(t, metadata.Methods, 1)
		assert.Len(t, metadata.BooleanProps, 1)
	})
}

func TestPropertyMetadata_GetPropType_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		prop     PropertyMetadata
	}{
		{
			name:     "string type",
			prop:     PropertyMetadata{JSType: "string"},
			expected: "String",
		},
		{
			name:     "number type",
			prop:     PropertyMetadata{JSType: "number"},
			expected: "Number",
		},
		{
			name:     "boolean type",
			prop:     PropertyMetadata{JSType: "boolean"},
			expected: "Boolean",
		},
		{
			name:     "array type without element type",
			prop:     PropertyMetadata{JSType: "array"},
			expected: "Array",
		},
		{
			name:     "array with string element type",
			prop:     PropertyMetadata{JSType: "array", ElementType: "string"},
			expected: "Array<String>",
		},
		{
			name:     "array with number element type",
			prop:     PropertyMetadata{JSType: "array", ElementType: "number"},
			expected: "Array<Number>",
		},
		{
			name:     "array with custom element type",
			prop:     PropertyMetadata{JSType: "array", ElementType: "user"},
			expected: "Array<User>",
		},
		{
			name:     "object type without key/value types",
			prop:     PropertyMetadata{JSType: "object"},
			expected: "Object",
		},
		{
			name:     "map type with key and value types",
			prop:     PropertyMetadata{JSType: "object", KeyType: "string", ValueType: "number"},
			expected: "Map<String,Number>",
		},
		{
			name:     "map with complex value type",
			prop:     PropertyMetadata{JSType: "object", KeyType: "string", ValueType: "user"},
			expected: "Map<String,User>",
		},
		{
			name:     "unknown type",
			prop:     PropertyMetadata{JSType: "unknown"},
			expected: "Any",
		},
		{
			name:     "empty type",
			prop:     PropertyMetadata{JSType: ""},
			expected: "Any",
		},
		{
			name:     "any type",
			prop:     PropertyMetadata{JSType: "any"},
			expected: "Any",
		},
		{
			name:     "object with only key type (not a map)",
			prop:     PropertyMetadata{JSType: "object", KeyType: "string"},
			expected: "Object",
		},
		{
			name:     "object with only value type (not a map)",
			prop:     PropertyMetadata{JSType: "object", ValueType: "number"},
			expected: "Object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.prop.GetPropType()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPropertyMetadata_GetDefaultValue_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		prop     PropertyMetadata
	}{
		{
			name:     "with initial value",
			prop:     PropertyMetadata{JSType: "string", InitialValue: `"hello"`},
			expected: `"hello"`,
		},
		{
			name:     "with numeric initial value",
			prop:     PropertyMetadata{JSType: "number", InitialValue: "42"},
			expected: "42",
		},
		{
			name:     "with array initial value",
			prop:     PropertyMetadata{JSType: "array", InitialValue: "[1, 2, 3]"},
			expected: "[1, 2, 3]",
		},
		{
			name:     "string fallback",
			prop:     PropertyMetadata{JSType: "string"},
			expected: `""`,
		},
		{
			name:     "number fallback",
			prop:     PropertyMetadata{JSType: "number"},
			expected: "0",
		},
		{
			name:     "boolean fallback",
			prop:     PropertyMetadata{JSType: "boolean"},
			expected: "false",
		},
		{
			name:     "array fallback",
			prop:     PropertyMetadata{JSType: "array"},
			expected: "[]",
		},
		{
			name:     "object fallback",
			prop:     PropertyMetadata{JSType: "object"},
			expected: "{}",
		},
		{
			name:     "unknown type fallback",
			prop:     PropertyMetadata{JSType: "unknown"},
			expected: "null",
		},
		{
			name:     "empty type fallback",
			prop:     PropertyMetadata{JSType: ""},
			expected: "null",
		},
		{
			name:     "any type fallback",
			prop:     PropertyMetadata{JSType: "any"},
			expected: "null",
		},
		{
			name:     "initial value takes precedence over type fallback",
			prop:     PropertyMetadata{JSType: "string", InitialValue: "null"},
			expected: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.prop.GetDefaultValue()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPropertyMetadata_IsBoolean(t *testing.T) {
	tests := []struct {
		name     string
		prop     PropertyMetadata
		expected bool
	}{
		{
			name:     "boolean type",
			prop:     PropertyMetadata{JSType: "boolean"},
			expected: true,
		},
		{
			name:     "string type",
			prop:     PropertyMetadata{JSType: "string"},
			expected: false,
		},
		{
			name:     "number type",
			prop:     PropertyMetadata{JSType: "number"},
			expected: false,
		},
		{
			name:     "array type",
			prop:     PropertyMetadata{JSType: "array"},
			expected: false,
		},
		{
			name:     "object type",
			prop:     PropertyMetadata{JSType: "object"},
			expected: false,
		},
		{
			name:     "empty type",
			prop:     PropertyMetadata{JSType: ""},
			expected: false,
		},
		{
			name:     "Boolean uppercase (not matching)",
			prop:     PropertyMetadata{JSType: "Boolean"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.prop.IsBoolean()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPropertyMetadata_FullProperties(t *testing.T) {
	t.Run("fully populated property", func(t *testing.T) {
		prop := PropertyMetadata{
			Name:         "items",
			JSType:       "array",
			ElementType:  "string",
			InitialValue: `["a", "b"]`,
			Location:     ast_domain.Location{Line: 10, Column: 5},
			IsNullable:   true,
		}

		assert.Equal(t, "items", prop.Name)
		assert.Equal(t, "Array<String>", prop.GetPropType())
		assert.Equal(t, `["a", "b"]`, prop.GetDefaultValue())
		assert.False(t, prop.IsBoolean())
		assert.True(t, prop.IsNullable)
		assert.Equal(t, 10, prop.Location.Line)
		assert.Equal(t, 5, prop.Location.Column)
	})

	t.Run("map property", func(t *testing.T) {
		prop := PropertyMetadata{
			Name:       "cache",
			JSType:     "object",
			KeyType:    "string",
			ValueType:  "user",
			IsNullable: false,
		}

		assert.Equal(t, "Map<String,User>", prop.GetPropType())
		assert.Equal(t, "{}", prop.GetDefaultValue())
	})
}

func TestMethodMetadata(t *testing.T) {
	t.Run("basic method", func(t *testing.T) {
		method := MethodMetadata{
			Name:     "increment",
			Location: ast_domain.Location{Line: 20, Column: 1},
		}

		assert.Equal(t, "increment", method.Name)
		assert.Equal(t, 20, method.Location.Line)
		assert.Equal(t, 1, method.Location.Column)
	})
}

func TestComponentMetadata_Integration(t *testing.T) {
	t.Run("realistic component metadata", func(t *testing.T) {
		metadata := NewComponentMetadata()

		metadata.StateProperties["count"] = &PropertyMetadata{
			Name:         "count",
			JSType:       "number",
			InitialValue: "0",
		}
		metadata.StateProperties["message"] = &PropertyMetadata{
			Name:         "message",
			JSType:       "string",
			InitialValue: `"Hello"`,
		}
		metadata.StateProperties["active"] = &PropertyMetadata{
			Name:   "active",
			JSType: "boolean",
		}
		metadata.StateProperties["items"] = &PropertyMetadata{
			Name:        "items",
			JSType:      "array",
			ElementType: "string",
		}

		metadata.BooleanProps = append(metadata.BooleanProps, "active")

		metadata.Methods["increment"] = &MethodMetadata{Name: "increment"}
		metadata.Methods["decrement"] = &MethodMetadata{Name: "decrement"}
		metadata.Methods["toggle"] = &MethodMetadata{Name: "toggle"}

		assert.Len(t, metadata.StateProperties, 4)
		assert.Len(t, metadata.Methods, 3)
		assert.Len(t, metadata.BooleanProps, 1)

		assert.Equal(t, "Number", metadata.StateProperties["count"].GetPropType())
		assert.Equal(t, "String", metadata.StateProperties["message"].GetPropType())
		assert.Equal(t, "Boolean", metadata.StateProperties["active"].GetPropType())
		assert.Equal(t, "Array<String>", metadata.StateProperties["items"].GetPropType())

		assert.Equal(t, "0", metadata.StateProperties["count"].GetDefaultValue())
		assert.Equal(t, `"Hello"`, metadata.StateProperties["message"].GetDefaultValue())
		assert.Equal(t, "false", metadata.StateProperties["active"].GetDefaultValue())
		assert.Equal(t, "[]", metadata.StateProperties["items"].GetDefaultValue())

		assert.True(t, metadata.StateProperties["active"].IsBoolean())
		assert.False(t, metadata.StateProperties["count"].IsBoolean())
	})
}
