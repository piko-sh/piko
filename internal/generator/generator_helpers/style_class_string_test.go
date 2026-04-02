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

package generator_helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestClassesFromString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "single class", input: "btn", want: "btn"},
		{name: "multiple classes", input: "btn btn-primary", want: "btn btn-primary"},
		{name: "duplicates removed", input: "a b a", want: "a b"},
		{name: "extra whitespace trimmed", input: "  a   b  ", want: "a b"},
		{name: "empty string", input: "", want: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ClassesFromString(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestClassesFromSlice(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		want  string
		input []string
	}{
		{name: "empty slice", input: []string{}, want: ""},
		{name: "nil slice", input: nil, want: ""},
		{name: "single class", input: []string{"btn"}, want: "btn"},
		{name: "multiple classes", input: []string{"btn", "btn-primary"}, want: "btn btn-primary"},
		{name: "duplicates removed", input: []string{"a", "b", "a"}, want: "a b"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ClassesFromSlice(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestClassesFromMapStringBool(t *testing.T) {
	t.Parallel()

	t.Run("empty map", func(t *testing.T) {
		t.Parallel()

		got := ClassesFromMapStringBool(map[string]bool{})
		assert.Equal(t, "", got)
	})

	t.Run("all false", func(t *testing.T) {
		t.Parallel()

		got := ClassesFromMapStringBool(map[string]bool{"a": false, "b": false})
		assert.Equal(t, "", got)
	})

	t.Run("mixed true and false", func(t *testing.T) {
		t.Parallel()

		got := ClassesFromMapStringBool(map[string]bool{"b": true, "a": true, "c": false})
		assert.Equal(t, "a b", got)
	})

	t.Run("single true", func(t *testing.T) {
		t.Parallel()

		got := ClassesFromMapStringBool(map[string]bool{"active": true})
		assert.Equal(t, "active", got)
	})
}

func TestMergeClasses(t *testing.T) {
	t.Parallel()

	t.Run("empty values", func(t *testing.T) {
		t.Parallel()

		got := MergeClasses()
		assert.Equal(t, "", got)
	})

	t.Run("string values", func(t *testing.T) {
		t.Parallel()

		got := MergeClasses("btn", "btn-primary")
		assert.Equal(t, "btn btn-primary", got)
	})

	t.Run("string slice value", func(t *testing.T) {
		t.Parallel()

		got := MergeClasses([]string{"a", "b"})
		assert.Equal(t, "a b", got)
	})

	t.Run("map string bool", func(t *testing.T) {
		t.Parallel()

		got := MergeClasses(map[string]bool{"active": true, "disabled": false})
		assert.Equal(t, "active", got)
	})

	t.Run("map string any with truthiness", func(t *testing.T) {
		t.Parallel()

		got := MergeClasses(map[string]any{"visible": true, "hidden": false})
		assert.Equal(t, "visible", got)
	})

	t.Run("slice any", func(t *testing.T) {
		t.Parallel()

		got := MergeClasses([]any{"x", "y"})
		assert.Equal(t, "x y", got)
	})

	t.Run("nil value handled", func(t *testing.T) {
		t.Parallel()

		got := MergeClasses("btn", nil)
		assert.Equal(t, "btn", got)
	})

	t.Run("deduplicated across values", func(t *testing.T) {
		t.Parallel()

		got := MergeClasses("a b", "b c")
		assert.Equal(t, "a b c", got)
	})
}

func TestStylesFromString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "single property", input: "color: red", want: "color:red;"},
		{name: "multiple properties sorted", input: "font-size: 12px; color: red", want: "color:red;font-size:12px;"},
		{name: "trailing semicolon", input: "color: red;", want: "color:red;"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := StylesFromString(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStylesFromStringMap(t *testing.T) {
	t.Parallel()

	t.Run("empty map", func(t *testing.T) {
		t.Parallel()

		got := StylesFromStringMap(map[string]string{})
		assert.Equal(t, "", got)
	})

	t.Run("single property", func(t *testing.T) {
		t.Parallel()

		got := StylesFromStringMap(map[string]string{"color": "red"})
		assert.Equal(t, "color:red;", got)
	})

	t.Run("camelCase converted to kebab-case", func(t *testing.T) {
		t.Parallel()

		got := StylesFromStringMap(map[string]string{"fontSize": "12px"})
		assert.Equal(t, "font-size:12px;", got)
	})
}

func TestMergeStyles(t *testing.T) {
	t.Parallel()

	t.Run("empty values", func(t *testing.T) {
		t.Parallel()

		got := MergeStyles()
		assert.Equal(t, "", got)
	})

	t.Run("string value", func(t *testing.T) {
		t.Parallel()

		got := MergeStyles("color: red; font-size: 12px")
		assert.Equal(t, "color:red;font-size:12px;", got)
	})

	t.Run("map string string", func(t *testing.T) {
		t.Parallel()

		got := MergeStyles(map[string]string{"color": "blue"})
		assert.Equal(t, "color:blue;", got)
	})

	t.Run("map string any", func(t *testing.T) {
		t.Parallel()

		got := MergeStyles(map[string]any{"color": "green"})
		assert.Equal(t, "color:green;", got)
	})

	t.Run("later value overrides earlier", func(t *testing.T) {
		t.Parallel()

		got := MergeStyles("color: red", map[string]string{"color": "blue"})
		assert.Equal(t, "color:blue;", got)
	})

	t.Run("map string any with nil value removes key", func(t *testing.T) {
		t.Parallel()

		got := MergeStyles("color: red; font-size: 12px", map[string]any{"color": nil})
		assert.Equal(t, "font-size:12px;", got)
	})

	t.Run("unsupported type ignored", func(t *testing.T) {
		t.Parallel()

		got := MergeStyles("color: red", 42)
		assert.Equal(t, "color:red;", got)
	})
}

func TestBuildClassBytes2(t *testing.T) {
	t.Parallel()

	warmupStylePool()
	result := BuildClassBytes2("btn", "btn-primary")
	require.NotNil(t, result)
	defer ast_domain.PutByteBuf(result)
	assert.Equal(t, "btn btn-primary", string(*result))
}

func TestBuildClassBytes4(t *testing.T) {
	t.Parallel()

	warmupStylePool()
	result := BuildClassBytes4("a", "b", "c", "d")
	require.NotNil(t, result)
	defer ast_domain.PutByteBuf(result)
	assert.Equal(t, "a b c d", string(*result))
}

func TestBuildClassBytes6(t *testing.T) {
	t.Parallel()

	warmupStylePool()
	result := BuildClassBytes6("a", "b", "c", "d", "e", "f")
	require.NotNil(t, result)
	defer ast_domain.PutByteBuf(result)
	assert.Equal(t, "a b c d e f", string(*result))
}

func TestBuildClassBytes8(t *testing.T) {
	t.Parallel()

	warmupStylePool()
	result := BuildClassBytes8("a", "b", "c", "d", "e", "f", "g", "h")
	require.NotNil(t, result)
	defer ast_domain.PutByteBuf(result)
	assert.Equal(t, "a b c d e f g h", string(*result))
}

func TestBuildClassBytes2Deduplication(t *testing.T) {
	t.Parallel()

	warmupStylePool()
	result := BuildClassBytes2("a b", "b c")
	require.NotNil(t, result)
	defer ast_domain.PutByteBuf(result)
	assert.Equal(t, "a b c", string(*result))
}

func TestBuildClassBytes2Empty(t *testing.T) {
	t.Parallel()

	warmupStylePool()
	result := BuildClassBytes2("", "")
	assert.Nil(t, result)
}

func TestAppendHiddenToStyleBytes(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns hidden style", func(t *testing.T) {
		t.Parallel()

		warmupStylePool()
		result := AppendHiddenToStyleBytes(nil)
		require.NotNil(t, result)
		defer ast_domain.PutByteBuf(result)
		assert.Contains(t, string(*result), "display")
	})

	t.Run("existing style gets hidden appended", func(t *testing.T) {
		t.Parallel()

		warmupStylePool()
		existing := StylesFromStringBytes("color:red;")
		require.NotNil(t, existing)
		result := AppendHiddenToStyleBytes(existing)
		defer ast_domain.PutByteBuf(result)
		s := string(*result)
		assert.Contains(t, s, "color:red")
		assert.Contains(t, s, "display")
	})
}

func TestMergeStylesBytes(t *testing.T) {
	t.Parallel()

	warmupStylePool()

	t.Run("string style parsed correctly", func(t *testing.T) {
		t.Parallel()

		result := MergeStylesBytes("color: red", nil, nil)
		require.NotNil(t, result)
		defer ast_domain.PutByteBuf(result)
		assert.Equal(t, "color:red;", string(*result))
	})

	t.Run("nil static and nil dynamic returns nil", func(t *testing.T) {
		t.Parallel()

		result := MergeStylesBytes("", nil, nil)
		assert.Nil(t, result)
	})
}

func TestClassesFromSliceAnyEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty slice any", func(t *testing.T) {
		t.Parallel()

		got := MergeClasses([]any{})
		assert.Equal(t, "", got)
	})

	t.Run("slice with non-string items", func(t *testing.T) {
		t.Parallel()

		got := MergeClasses([]any{42, true, ""})
		assert.Equal(t, "", got)
	})
}

func TestClassesFromMapStringAnyEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty map any", func(t *testing.T) {
		t.Parallel()

		got := MergeClasses(map[string]any{})
		assert.Equal(t, "", got)
	})

	t.Run("all falsy values", func(t *testing.T) {
		t.Parallel()

		got := MergeClasses(map[string]any{"a": false, "b": 0, "c": ""})
		assert.Equal(t, "", got)
	})
}
