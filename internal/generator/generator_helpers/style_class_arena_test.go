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

func TestBuildClassBytes2Arena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := BuildClassBytes2Arena(arena, "btn", "btn-primary")
	require.NotNil(t, result)
	assert.Equal(t, "btn btn-primary", string(*result))
}

func TestBuildClassBytes4Arena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := BuildClassBytes4Arena(arena, "a", "b", "c", "d")
	require.NotNil(t, result)
	assert.Equal(t, "a b c d", string(*result))
}

func TestBuildClassBytes6Arena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := BuildClassBytes6Arena(arena, "a", "b", "c", "d", "e", "f")
	require.NotNil(t, result)
	assert.Equal(t, "a b c d e f", string(*result))
}

func TestBuildClassBytes8Arena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := BuildClassBytes8Arena(arena, "a", "b", "c", "d", "e", "f", "g", "h")
	require.NotNil(t, result)
	assert.Equal(t, "a b c d e f g h", string(*result))
}

func TestBuildClassBytesVArena(t *testing.T) {
	t.Parallel()

	t.Run("multiple parts", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := BuildClassBytesVArena(arena, "x", "y", "z")
		require.NotNil(t, result)
		assert.Equal(t, "x y z", string(*result))
	})

	t.Run("empty parts returns nil", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := BuildClassBytesVArena(arena)
		assert.Nil(t, result)
	})

	t.Run("deduplication", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := BuildClassBytesVArena(arena, "a b", "b c")
		require.NotNil(t, result)
		assert.Equal(t, "a b c", string(*result))
	})
}

func TestBuildClassBytes2Arena_Empty(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := BuildClassBytes2Arena(arena, "", "")
	assert.Nil(t, result)
}

func TestBuildClassBytes2Arena_Deduplication(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := BuildClassBytes2Arena(arena, "a b", "b c")
	require.NotNil(t, result)
	assert.Equal(t, "a b c", string(*result))
}

func TestMergeClassesBytesArena(t *testing.T) {
	t.Parallel()

	t.Run("string values", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := MergeClassesBytesArena(arena, "btn", "btn-primary")
		require.NotNil(t, result)
		assert.Equal(t, "btn btn-primary", string(*result))
	})

	t.Run("empty values returns nil", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := MergeClassesBytesArena(arena)
		assert.Nil(t, result)
	})

	t.Run("map string bool", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := MergeClassesBytesArena(arena, map[string]bool{"active": true, "disabled": false})
		require.NotNil(t, result)
		assert.Equal(t, "active", string(*result))
	})
}

func TestClassesFromSliceBytesArena(t *testing.T) {
	t.Parallel()

	t.Run("normal slice", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := ClassesFromSliceBytesArena(arena, []string{"btn", "btn-primary"})
		require.NotNil(t, result)
		assert.Equal(t, "btn btn-primary", string(*result))
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := ClassesFromSliceBytesArena(arena, []string{})
		assert.Nil(t, result)
	})
}

func TestClassesFromStringBytesArena(t *testing.T) {
	t.Parallel()

	t.Run("normal string", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := ClassesFromStringBytesArena(arena, "btn btn-primary")
		require.NotNil(t, result)
		assert.Equal(t, "btn btn-primary", string(*result))
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := ClassesFromStringBytesArena(arena, "")
		assert.Nil(t, result)
	})
}

func TestStylesFromStringBytesArena(t *testing.T) {
	t.Parallel()

	t.Run("simple property", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := StylesFromStringBytesArena(arena, "color: red")
		require.NotNil(t, result)
		assert.Equal(t, "color:red;", string(*result))
	})

	t.Run("multiple properties sorted", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := StylesFromStringBytesArena(arena, "font-size: 12px; color: red")
		require.NotNil(t, result)
		assert.Equal(t, "color:red;font-size:12px;", string(*result))
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := StylesFromStringBytesArena(arena, "")
		assert.Nil(t, result)
	})
}

func TestStylesFromStringMapBytesArena(t *testing.T) {
	t.Parallel()

	t.Run("normal map", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := StylesFromStringMapBytesArena(arena, map[string]string{"color": "red"})
		require.NotNil(t, result)
		assert.Equal(t, "color:red;", string(*result))
	})

	t.Run("empty map returns nil", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := StylesFromStringMapBytesArena(arena, map[string]string{})
		assert.Nil(t, result)
	})
}

func TestMergeStylesBytesArena(t *testing.T) {
	t.Parallel()

	t.Run("string value", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := MergeStylesBytesArena(arena, "color: red")
		require.NotNil(t, result)
		assert.Equal(t, "color:red;", string(*result))
	})

	t.Run("empty values returns nil", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := MergeStylesBytesArena(arena)
		assert.Nil(t, result)
	})

	t.Run("map string string", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := MergeStylesBytesArena(arena, map[string]string{"color": "blue"})
		require.NotNil(t, result)
		assert.Equal(t, "color:blue;", string(*result))
	})

	t.Run("map string any", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := MergeStylesBytesArena(arena, map[string]any{"color": "green"})
		require.NotNil(t, result)
		assert.Equal(t, "color:green;", string(*result))
	})

	t.Run("later value overrides earlier", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := MergeStylesBytesArena(arena, "color: red", map[string]string{"color": "blue"})
		require.NotNil(t, result)
		assert.Equal(t, "color:blue;", string(*result))
	})

	t.Run("all empty styles returns nil", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := MergeStylesBytesArena(arena, "", map[string]string{})
		assert.Nil(t, result)
	})
}

func TestBuildStyleStringBytes2Arena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := BuildStyleStringBytes2Arena(arena, "color: ", "red")
	require.NotNil(t, result)
	assert.Equal(t, "color:red;", string(*result))
}

func TestBuildStyleStringBytes3Arena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := BuildStyleStringBytes3Arena(arena, "--var: ", "value", ";")
	require.NotNil(t, result)
	assert.Contains(t, string(*result), "value")
}

func TestBuildStyleStringBytes4Arena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := BuildStyleStringBytes4Arena(arena, "color: ", "red", "; background: ", "blue")
	require.NotNil(t, result)
	s := string(*result)
	assert.Contains(t, s, "color:red;")
	assert.Contains(t, s, "background:blue;")
}

func TestBuildStyleStringBytesVArena(t *testing.T) {
	t.Parallel()

	t.Run("empty returns nil", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := BuildStyleStringBytesVArena(arena)
		assert.Nil(t, result)
	})

	t.Run("one part delegates to StylesFromStringBytesArena", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := BuildStyleStringBytesVArena(arena, "color: red")
		require.NotNil(t, result)
		assert.Equal(t, "color:red;", string(*result))
	})

	t.Run("two parts", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := BuildStyleStringBytesVArena(arena, "color: ", "red")
		require.NotNil(t, result)
		assert.Equal(t, "color:red;", string(*result))
	})

	t.Run("three parts", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := BuildStyleStringBytesVArena(arena, "--var: ", "value", ";")
		require.NotNil(t, result)
		assert.Contains(t, string(*result), "value")
	})

	t.Run("four parts", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := BuildStyleStringBytesVArena(arena, "color: ", "red", "; bg: ", "blue")
		require.NotNil(t, result)
		s := string(*result)
		assert.Contains(t, s, "color:red;")
	})

	t.Run("five parts falls back to concat", func(t *testing.T) {
		t.Parallel()

		arena := ast_domain.GetArena()
		defer ast_domain.PutArena(arena)

		result := BuildStyleStringBytesVArena(arena, "color: ", "red", "; bg: ", "blue", "; opacity: 1")
		require.NotNil(t, result)
		s := string(*result)
		assert.Contains(t, s, "color:red;")
	})
}

func TestMergeClassesBytes(t *testing.T) {
	warmupStylePool()

	t.Run("string values", func(t *testing.T) {
		result := MergeClassesBytes("btn", "btn-primary")
		require.NotNil(t, result)
		defer ast_domain.PutByteBuf(result)
		assert.Equal(t, "btn btn-primary", string(*result))
	})

	t.Run("empty returns nil", func(t *testing.T) {
		result := MergeClassesBytes()
		assert.Nil(t, result)
	})

	t.Run("map string bool", func(t *testing.T) {
		result := MergeClassesBytes(map[string]bool{"active": true, "hidden": false})
		require.NotNil(t, result)
		defer ast_domain.PutByteBuf(result)
		assert.Equal(t, "active", string(*result))
	})

	t.Run("slice of strings", func(t *testing.T) {
		result := MergeClassesBytes([]string{"a", "b"})
		require.NotNil(t, result)
		defer ast_domain.PutByteBuf(result)
		assert.Equal(t, "a b", string(*result))
	})

	t.Run("deduplication across values", func(t *testing.T) {
		result := MergeClassesBytes("a b", "b c")
		require.NotNil(t, result)
		defer ast_domain.PutByteBuf(result)
		assert.Equal(t, "a b c", string(*result))
	})
}

func TestClassesFromSliceBytes(t *testing.T) {
	warmupStylePool()

	t.Run("normal slice", func(t *testing.T) {
		result := ClassesFromSliceBytes([]string{"btn", "btn-primary"})
		require.NotNil(t, result)
		defer ast_domain.PutByteBuf(result)
		assert.Equal(t, "btn btn-primary", string(*result))
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		result := ClassesFromSliceBytes([]string{})
		assert.Nil(t, result)
	})
}

func TestClassesFromStringBytes(t *testing.T) {
	warmupStylePool()

	t.Run("normal string", func(t *testing.T) {
		result := ClassesFromStringBytes("btn active")
		require.NotNil(t, result)
		defer ast_domain.PutByteBuf(result)
		assert.Equal(t, "btn active", string(*result))
	})

	t.Run("deduplication", func(t *testing.T) {
		result := ClassesFromStringBytes("a b a")
		require.NotNil(t, result)
		defer ast_domain.PutByteBuf(result)
		assert.Equal(t, "a b", string(*result))
	})
}

func TestBuildClassBytesV(t *testing.T) {
	warmupStylePool()

	t.Run("multiple parts", func(t *testing.T) {
		result := BuildClassBytesV("x", "y", "z")
		require.NotNil(t, result)
		defer ast_domain.PutByteBuf(result)
		assert.Equal(t, "x y z", string(*result))
	})

	t.Run("empty returns nil", func(t *testing.T) {
		result := BuildClassBytesV()
		assert.Nil(t, result)
	})
}

func TestStylesFromStringMapBytes(t *testing.T) {
	warmupStylePool()

	t.Run("single property", func(t *testing.T) {
		result := StylesFromStringMapBytes(map[string]string{"color": "red"})
		require.NotNil(t, result)
		defer ast_domain.PutByteBuf(result)
		assert.Equal(t, "color:red;", string(*result))
	})

	t.Run("empty map returns nil", func(t *testing.T) {
		result := StylesFromStringMapBytes(map[string]string{})
		assert.Nil(t, result)
	})

	t.Run("multiple properties sorted", func(t *testing.T) {
		result := StylesFromStringMapBytes(map[string]string{"color": "red", "background": "blue"})
		require.NotNil(t, result)
		defer ast_domain.PutByteBuf(result)
		assert.Equal(t, "background:blue;color:red;", string(*result))
	})
}
