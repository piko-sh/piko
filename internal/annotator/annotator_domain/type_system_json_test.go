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

package annotator_domain

import (
	"testing"

	goast "go/ast"

	"github.com/stretchr/testify/assert"
)

func TestIsJSONPrimitive(t *testing.T) {
	t.Parallel()

	t.Run("should return true for JSON-safe primitives", func(t *testing.T) {
		t.Parallel()

		safePrimitives := []string{
			"string", "bool",
			"int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64",
			"float32", "float64",
			"byte", "rune",
		}

		for _, primitive := range safePrimitives {
			assert.True(t, isJSONPrimitive(primitive),
				"Expected %q to be a JSON-safe primitive", primitive)
		}
	})

	t.Run("should return false for any/interface{}", func(t *testing.T) {
		t.Parallel()

		assert.False(t, isJSONPrimitive("any"),
			"SECURITY: 'any' must NOT be considered a JSON primitive")
		assert.False(t, isJSONPrimitive("interface{}"),
			"SECURITY: 'interface{}' must NOT be considered a JSON primitive")
	})

	t.Run("should return false for complex types", func(t *testing.T) {
		t.Parallel()

		assert.False(t, isJSONPrimitive("complex64"),
			"complex64 cannot be JSON-serialised")
		assert.False(t, isJSONPrimitive("complex128"),
			"complex128 cannot be JSON-serialised")
	})

	t.Run("should return false for user-defined types", func(t *testing.T) {
		t.Parallel()

		userTypes := []string{
			"MyType", "User", "time.Time", "uuid.UUID",
			"MyInt", "MyString", "CustomBool",
		}

		for _, typeName := range userTypes {
			assert.False(t, isJSONPrimitive(typeName),
				"User-defined type %q should not be a JSON primitive", typeName)
		}
	})

	t.Run("should return false for empty string", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isJSONPrimitive(""))
	})
}

func TestIsJSONSafeKeyType(t *testing.T) {
	t.Parallel()

	t.Run("should return true for string keys", func(t *testing.T) {
		t.Parallel()

		expression := &goast.Ident{Name: "string"}
		assert.True(t, isJSONSafeKeyType(expression), "string should be a valid JSON key type")
	})

	t.Run("should return true for integer keys", func(t *testing.T) {
		t.Parallel()

		integerTypes := []string{
			"int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		}

		for _, typeName := range integerTypes {
			expression := &goast.Ident{Name: typeName}
			assert.True(t, isJSONSafeKeyType(expression),
				"%s should be a valid JSON key type (auto-converts to string)", typeName)
		}
	})

	t.Run("should return false for non-string/non-integer keys", func(t *testing.T) {
		t.Parallel()

		invalidKeyTypes := []string{
			"float32", "float64",
			"bool",
			"complex64", "complex128",
			"MyType", "time.Time",
		}

		for _, typeName := range invalidKeyTypes {
			expression := &goast.Ident{Name: typeName}
			assert.False(t, isJSONSafeKeyType(expression),
				"%s should NOT be a valid JSON key type", typeName)
		}
	})

	t.Run("should return false for non-Ident expressions", func(t *testing.T) {
		t.Parallel()

		selectorExpr := &goast.SelectorExpr{
			X:   &goast.Ident{Name: "pkg"},
			Sel: &goast.Ident{Name: "Type"},
		}
		assert.False(t, isJSONSafeKeyType(selectorExpr),
			"Selector expressions should not be valid JSON key types")

		starExpr := &goast.StarExpr{X: &goast.Ident{Name: "string"}}
		assert.False(t, isJSONSafeKeyType(starExpr),
			"Pointer types should not be valid JSON key types")

		arrayExpr := &goast.ArrayType{Elt: &goast.Ident{Name: "byte"}}
		assert.False(t, isJSONSafeKeyType(arrayExpr),
			"Array types should not be valid JSON key types")
	})

	t.Run("should return false for nil", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isJSONSafeKeyType(nil))
	})
}

func TestIsJSONPrimitive_ExhaustiveEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("case sensitivity", func(t *testing.T) {
		t.Parallel()

		assert.False(t, isJSONPrimitive("String"), "String (capital S) is not a primitive")
		assert.False(t, isJSONPrimitive("INT"), "INT (all caps) is not a primitive")
		assert.False(t, isJSONPrimitive("Bool"), "Bool (capital B) is not a primitive")
	})

	t.Run("whitespace handling", func(t *testing.T) {
		t.Parallel()

		assert.False(t, isJSONPrimitive(" string"), "leading whitespace")
		assert.False(t, isJSONPrimitive("string "), "trailing whitespace")
		assert.False(t, isJSONPrimitive(" string "), "surrounding whitespace")
	})

	t.Run("similar but different type names", func(t *testing.T) {
		t.Parallel()

		assert.False(t, isJSONPrimitive("string2"))
		assert.False(t, isJSONPrimitive("myint"))
		assert.False(t, isJSONPrimitive("int_"))
		assert.False(t, isJSONPrimitive("_bool"))
	})
}

func BenchmarkIsJSONPrimitive_String(b *testing.B) {
	for b.Loop() {
		_ = isJSONPrimitive("string")
	}
}

func BenchmarkIsJSONPrimitive_Int64(b *testing.B) {
	for b.Loop() {
		_ = isJSONPrimitive("int64")
	}
}

func BenchmarkIsJSONPrimitive_UserType(b *testing.B) {
	for b.Loop() {
		_ = isJSONPrimitive("MyCustomType")
	}
}

func BenchmarkIsJSONSafeKeyType_String(b *testing.B) {
	expression := &goast.Ident{Name: "string"}
	b.ResetTimer()
	for b.Loop() {
		_ = isJSONSafeKeyType(expression)
	}
}

func BenchmarkIsJSONSafeKeyType_Int(b *testing.B) {
	expression := &goast.Ident{Name: "int"}
	b.ResetTimer()
	for b.Loop() {
		_ = isJSONSafeKeyType(expression)
	}
}
