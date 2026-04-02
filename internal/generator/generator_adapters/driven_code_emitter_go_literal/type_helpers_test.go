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

package driven_code_emitter_go_literal

import (
	"strings"
	"testing"

	goast "go/ast"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestIsBoolType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeExpr goast.Expr
		name     string
		want     bool
	}{
		{
			name:     "bool ident",
			typeExpr: cachedIdent("bool"),
			want:     true,
		},
		{
			name:     "int ident",
			typeExpr: cachedIdent("int"),
			want:     false,
		},
		{
			name:     "string ident",
			typeExpr: cachedIdent("string"),
			want:     false,
		},
		{
			name:     "pointer to bool is not bool type",
			typeExpr: &goast.StarExpr{X: cachedIdent("bool")},
			want:     false,
		},
		{
			name:     "nil type expr",
			typeExpr: nil,
			want:     false,
		},
		{
			name:     "selector expr pkg.bool is not bool",
			typeExpr: &goast.SelectorExpr{X: cachedIdent("pkg"), Sel: cachedIdent("bool")},
			want:     false,
		},
		{
			name:     "map type",
			typeExpr: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("bool")},
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isBoolType(tc.typeExpr)

			assert.Equal(t, tc.want, got, "isBoolType(%v) = %v, want %v", tc.typeExpr, got, tc.want)
		})
	}
}

func TestIsStringType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeExpr goast.Expr
		name     string
		want     bool
	}{
		{name: "string ident", typeExpr: cachedIdent("string"), want: true},
		{name: "int ident", typeExpr: cachedIdent("int"), want: false},
		{name: "bool ident", typeExpr: cachedIdent("bool"), want: false},
		{name: "pointer to string", typeExpr: &goast.StarExpr{X: cachedIdent("string")}, want: false},
		{name: "nil type", typeExpr: nil, want: false},
		{name: "pkg.string selector", typeExpr: &goast.SelectorExpr{X: cachedIdent("pkg"), Sel: cachedIdent("string")}, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isStringType(tc.typeExpr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsNillableType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeExpr goast.Expr
		name     string
		want     bool
	}{

		{name: "nil type expr is nillable", typeExpr: nil, want: true},

		{name: "*int pointer", typeExpr: &goast.StarExpr{X: cachedIdent("int")}, want: true},
		{name: "*string pointer", typeExpr: &goast.StarExpr{X: cachedIdent("string")}, want: true},
		{name: "**int double pointer", typeExpr: &goast.StarExpr{X: &goast.StarExpr{X: cachedIdent("int")}}, want: true},

		{name: "[]int slice", typeExpr: &goast.ArrayType{Len: nil, Elt: cachedIdent("int")}, want: true},
		{name: "[]string slice", typeExpr: &goast.ArrayType{Len: nil, Elt: cachedIdent("string")}, want: true},

		{name: "[5]int array", typeExpr: &goast.ArrayType{Len: &goast.BasicLit{Value: "5"}, Elt: cachedIdent("int")}, want: false},
		{name: "[10]string array", typeExpr: &goast.ArrayType{Len: &goast.BasicLit{Value: "10"}, Elt: cachedIdent("string")}, want: false},

		{name: "map[string]int", typeExpr: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")}, want: true},
		{name: "map[int]bool", typeExpr: &goast.MapType{Key: cachedIdent("int"), Value: cachedIdent("bool")}, want: true},

		{name: "func() type", typeExpr: &goast.FuncType{Params: &goast.FieldList{}}, want: true},

		{name: "chan int", typeExpr: &goast.ChanType{Value: cachedIdent("int")}, want: true},

		{name: "interface{}", typeExpr: &goast.InterfaceType{Methods: &goast.FieldList{}}, want: true},

		{name: "int", typeExpr: cachedIdent("int"), want: false},
		{name: "string", typeExpr: cachedIdent("string"), want: false},
		{name: "bool", typeExpr: cachedIdent("bool"), want: false},
		{name: "float64", typeExpr: cachedIdent("float64"), want: false},

		{name: "MyStruct", typeExpr: cachedIdent("MyStruct"), want: false},
		{name: "pkg.MyStruct selector", typeExpr: &goast.SelectorExpr{X: cachedIdent("pkg"), Sel: cachedIdent("MyStruct")}, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isNillableType(tc.typeExpr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsNumeric(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		typeName string
		want     bool
	}{

		{name: "int", typeName: "int", want: true},
		{name: "int8", typeName: "int8", want: true},
		{name: "int16", typeName: "int16", want: true},
		{name: "int32", typeName: "int32", want: true},
		{name: "int64", typeName: "int64", want: true},

		{name: "uint", typeName: "uint", want: true},
		{name: "uint8", typeName: "uint8", want: true},
		{name: "uint16", typeName: "uint16", want: true},
		{name: "uint32", typeName: "uint32", want: true},
		{name: "uint64", typeName: "uint64", want: true},
		{name: "uintptr", typeName: "uintptr", want: true},

		{name: "float32", typeName: "float32", want: true},
		{name: "float64", typeName: "float64", want: true},

		{name: "byte", typeName: "byte", want: true},
		{name: "rune", typeName: "rune", want: true},

		{name: "string", typeName: "string", want: false},
		{name: "bool", typeName: "bool", want: false},
		{name: "complex64", typeName: "complex64", want: false},
		{name: "complex128", typeName: "complex128", want: false},

		{name: "empty ResolvedTypeInfo", typeName: "", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var typeInfo *ast_domain.ResolvedTypeInfo
			if tc.typeName != "" {
				typeInfo = &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent(tc.typeName),
				}
			}

			got := isNumeric(typeInfo)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsExpressionStringType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		want     bool
	}{
		{
			name:     "string type",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("string")},
			want:     true,
		},
		{
			name:     "int type",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: cachedIdent("int")},
			want:     false,
		},
		{
			name:     "nil typeInfo",
			typeInfo: nil,
			want:     false,
		},
		{
			name:     "nil typeExpr",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			want:     false,
		},
		{
			name:     "pointer to string",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: cachedIdent("string")}},
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isExpressionStringType(tc.typeInfo)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestShouldSkipEscaping(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		typeName      string
		stringability inspector_dto.StringabilityMethod
		want          bool
	}{

		{name: "int primitive", typeName: "int", stringability: inspector_dto.StringablePrimitive, want: true},
		{name: "int64 primitive", typeName: "int64", stringability: inspector_dto.StringablePrimitive, want: true},
		{name: "bool primitive", typeName: "bool", stringability: inspector_dto.StringablePrimitive, want: true},
		{name: "float64 primitive", typeName: "float64", stringability: inspector_dto.StringablePrimitive, want: true},

		{name: "string primitive NEEDS escaping", typeName: "string", stringability: inspector_dto.StringablePrimitive, want: false},
		{name: "rune primitive NEEDS escaping", typeName: "rune", stringability: inspector_dto.StringablePrimitive, want: false},

		{name: "Stringer interface", typeName: "MyType", stringability: inspector_dto.StringableViaStringer, want: false},
		{name: "TextMarshaler", typeName: "MyType", stringability: inspector_dto.StringableViaTextMarshaler, want: false},

		{name: "nil annotation", typeName: "", stringability: 0, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var ann *ast_domain.GoGeneratorAnnotation
			if tc.typeName != "" {
				ann = createMockAnnotation(tc.typeName, tc.stringability)
			}

			got := shouldSkipEscaping(ann)

			assert.Equal(t, tc.want, got, "Type %s with stringability %d: expected shouldSkipEscaping=%v",
				tc.typeName, tc.stringability, tc.want)
		})
	}
}

func TestIsNillableType_AllGoTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		category string
		cases    []struct {
			typeExpr goast.Expr
			name     string
			want     bool
		}
	}{
		{
			category: "Reference Types (Nillable)",
			cases: []struct {
				typeExpr goast.Expr
				name     string
				want     bool
			}{
				{name: "*int", typeExpr: &goast.StarExpr{X: cachedIdent("int")}, want: true},
				{name: "[]byte", typeExpr: &goast.ArrayType{Len: nil, Elt: cachedIdent("byte")}, want: true},
				{name: "map[string]int", typeExpr: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")}, want: true},
				{name: "chan int", typeExpr: &goast.ChanType{Value: cachedIdent("int")}, want: true},
				{name: "func()", typeExpr: &goast.FuncType{Params: &goast.FieldList{}}, want: true},
				{name: "interface{}", typeExpr: &goast.InterfaceType{Methods: &goast.FieldList{}}, want: true},
			},
		},
		{
			category: "Value Types (NOT Nillable)",
			cases: []struct {
				typeExpr goast.Expr
				name     string
				want     bool
			}{
				{name: "int", typeExpr: cachedIdent("int"), want: false},
				{name: "string", typeExpr: cachedIdent("string"), want: false},
				{name: "bool", typeExpr: cachedIdent("bool"), want: false},
				{name: "[5]int array", typeExpr: &goast.ArrayType{Len: &goast.BasicLit{Value: "5"}, Elt: cachedIdent("int")}, want: false},
				{name: "struct{}", typeExpr: &goast.StructType{Fields: &goast.FieldList{}}, want: false},
			},
		},
	}

	for _, category := range testCases {
		t.Run(category.category, func(t *testing.T) {
			for _, tc := range category.cases {
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					got := isNillableType(tc.typeExpr)
					assert.Equal(t, tc.want, got)
				})
			}
		})
	}
}

func TestIsNumeric_Comprehensive(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		typeName string
		want     bool
	}{

		{name: "int", typeName: "int", want: true},
		{name: "int8", typeName: "int8", want: true},
		{name: "int16", typeName: "int16", want: true},
		{name: "int32", typeName: "int32", want: true},
		{name: "int64", typeName: "int64", want: true},

		{name: "uint", typeName: "uint", want: true},
		{name: "uint8", typeName: "uint8", want: true},
		{name: "uint16", typeName: "uint16", want: true},
		{name: "uint32", typeName: "uint32", want: true},
		{name: "uint64", typeName: "uint64", want: true},
		{name: "uintptr", typeName: "uintptr", want: true},

		{name: "float32", typeName: "float32", want: true},
		{name: "float64", typeName: "float64", want: true},

		{name: "byte (alias for uint8)", typeName: "byte", want: true},
		{name: "rune (alias for int32)", typeName: "rune", want: true},

		{name: "string is NOT numeric", typeName: "string", want: false},
		{name: "bool is NOT numeric", typeName: "bool", want: false},
		{name: "complex64 is NOT numeric", typeName: "complex64", want: false},
		{name: "complex128 is NOT numeric", typeName: "complex128", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			typeInfo := &ast_domain.ResolvedTypeInfo{
				TypeExpression: cachedIdent(tc.typeName),
			}

			got := isNumeric(typeInfo)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsNumeric_EdgeCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		want     bool
	}{
		{
			name:     "nil typeInfo",
			typeInfo: nil,
			want:     false,
		},
		{
			name:     "nil typeExpr",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			want:     false,
		},
		{
			name:     "pointer to int (NOT numeric - pointer type)",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: cachedIdent("int")}},
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isNumeric(tc.typeInfo)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsBasicGoType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeExpr goast.Expr
		name     string
		want     bool
	}{
		{name: "string is basic", typeExpr: cachedIdent("string"), want: true},
		{name: "int is basic", typeExpr: cachedIdent("int"), want: true},
		{name: "bool is basic", typeExpr: cachedIdent("bool"), want: true},
		{name: "float64 is basic", typeExpr: cachedIdent("float64"), want: true},
		{name: "float32 is basic", typeExpr: cachedIdent("float32"), want: true},
		{name: "byte is basic", typeExpr: cachedIdent("byte"), want: true},
		{name: "rune is basic", typeExpr: cachedIdent("rune"), want: true},
		{name: "error is basic", typeExpr: cachedIdent("error"), want: true},
		{name: "any is basic", typeExpr: cachedIdent("any"), want: true},
		{name: "int8 is basic", typeExpr: cachedIdent("int8"), want: true},
		{name: "int16 is basic", typeExpr: cachedIdent("int16"), want: true},
		{name: "int32 is basic", typeExpr: cachedIdent("int32"), want: true},
		{name: "int64 is basic", typeExpr: cachedIdent("int64"), want: true},
		{name: "uint is basic", typeExpr: cachedIdent("uint"), want: true},
		{name: "uint8 is basic", typeExpr: cachedIdent("uint8"), want: true},
		{name: "uint16 is basic", typeExpr: cachedIdent("uint16"), want: true},
		{name: "uint32 is basic", typeExpr: cachedIdent("uint32"), want: true},
		{name: "uint64 is basic", typeExpr: cachedIdent("uint64"), want: true},
		{name: "uintptr is basic", typeExpr: cachedIdent("uintptr"), want: true},
		{name: "complex64 is basic", typeExpr: cachedIdent("complex64"), want: true},
		{name: "complex128 is basic", typeExpr: cachedIdent("complex128"), want: true},
		{name: "CustomType is not basic", typeExpr: cachedIdent("CustomType"), want: false},
		{name: "MyStruct is not basic", typeExpr: cachedIdent("MyStruct"), want: false},
		{name: "nil returns false", typeExpr: nil, want: false},
		{
			name:     "SelectorExpr is not basic",
			typeExpr: &goast.SelectorExpr{X: cachedIdent("pkg"), Sel: cachedIdent("Type")},
			want:     false,
		},
		{
			name:     "StarExpr is not basic",
			typeExpr: &goast.StarExpr{X: cachedIdent("int")},
			want:     false,
		},
		{
			name:     "MapType is not basic",
			typeExpr: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")},
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isBasicGoType(tc.typeExpr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestShouldSkipEscaping_SecurityCritical(t *testing.T) {
	t.Parallel()

	unsafeTypes := []struct {
		name          string
		typeName      string
		stringability inspector_dto.StringabilityMethod
	}{
		{name: "string (user input)", typeName: "string", stringability: inspector_dto.StringablePrimitive},
		{name: "rune (could be HTML char)", typeName: "rune", stringability: inspector_dto.StringablePrimitive},
		{name: "Stringer (unknown content)", typeName: "MyType", stringability: inspector_dto.StringableViaStringer},
		{name: "TextMarshaler (unknown content)", typeName: "MyType", stringability: inspector_dto.StringableViaTextMarshaler},
	}

	for _, tc := range unsafeTypes {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ann := createMockAnnotation(tc.typeName, tc.stringability)
			got := shouldSkipEscaping(ann)

			assert.False(t, got, "SECURITY: Type %s must not skip HTML escaping", tc.name)
		})
	}

	safeTypes := []string{"int", "int64", "bool", "float64", "uint32"}
	for _, typeName := range safeTypes {
		t.Run("safe_"+typeName, func(t *testing.T) {
			t.Parallel()

			ann := createMockAnnotation(typeName, inspector_dto.StringablePrimitive)
			got := shouldSkipEscaping(ann)

			assert.True(t, got, "Safe primitive %s can skip escaping", typeName)
		})
	}
}

func TestShouldSkipEscaping_NilHandling(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ann  *ast_domain.GoGeneratorAnnotation
		name string
	}{
		{name: "nil annotation", ann: nil},
		{name: "nil ResolvedType", ann: &ast_domain.GoGeneratorAnnotation{ResolvedType: nil}},
		{name: "nil TypeExpr", ann: &ast_domain.GoGeneratorAnnotation{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: nil}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldSkipEscaping(tc.ann)

			assert.False(t, got, "Nil cases should default to requiring escaping (safe)")
		})
	}
}

func BenchmarkIsBoolType(b *testing.B) {
	typeExpr := cachedIdent("bool")

	b.ResetTimer()
	for b.Loop() {
		_ = isBoolType(typeExpr)
	}
}

func BenchmarkIsNillableType(b *testing.B) {
	typeExpr := &goast.StarExpr{X: cachedIdent("int")}

	b.ResetTimer()
	for b.Loop() {
		_ = isNillableType(typeExpr)
	}
}

func BenchmarkIsNumeric(b *testing.B) {
	typeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression: cachedIdent("int64"),
	}

	b.ResetTimer()
	for b.Loop() {
		_ = isNumeric(typeInfo)
	}
}

func TestGetSyntheticTypeName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		typeInfo *ast_domain.ResolvedTypeInfo
		want     string
		contains string
	}{
		{
			name:     "nil typeInfo returns unknown",
			typeInfo: nil,
			want:     "unknown",
		},
		{
			name:     "nil TypeExpr returns unknown",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			want:     "unknown",
		},
		{
			name: "Ident returns its name",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.Ident{Name: "MyType"},
			},
			want: "MyType",
		},
		{
			name: "SelectorExpr falls through to ASTToTypeString",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.SelectorExpr{
					X:   cachedIdent("pkg"),
					Sel: cachedIdent("Type"),
				},
				PackageAlias: "pkg",
			},
			contains: "Type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := getSyntheticTypeName(tc.typeInfo)

			if tc.contains != "" {
				assert.NotEmpty(t, got)
				assert.True(t, strings.Contains(got, tc.contains),
					"expected result %q to contain %q", got, tc.contains)
			} else {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
