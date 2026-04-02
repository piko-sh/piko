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
	"testing"

	goast "go/ast"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestValueToString_Primitives(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		typeName     string
		wantCallTo   string
		wantIdentity bool
	}{

		{name: "string identity", typeName: "string", wantIdentity: true},

		{name: "int", typeName: "int", wantCallTo: "FormatInt"},
		{name: "int8", typeName: "int8", wantCallTo: "FormatInt"},
		{name: "int16", typeName: "int16", wantCallTo: "FormatInt"},
		{name: "int32", typeName: "int32", wantCallTo: "FormatInt"},
		{name: "int64", typeName: "int64", wantCallTo: "FormatInt"},

		{name: "uint", typeName: "uint", wantCallTo: "FormatUint"},
		{name: "uint8", typeName: "uint8", wantCallTo: "FormatUint"},
		{name: "uint16", typeName: "uint16", wantCallTo: "FormatUint"},
		{name: "uint32", typeName: "uint32", wantCallTo: "FormatUint"},
		{name: "uint64", typeName: "uint64", wantCallTo: "FormatUint"},
		{name: "uintptr", typeName: "uintptr", wantCallTo: "FormatUint"},
		{name: "byte", typeName: "byte", wantCallTo: "FormatUint"},

		{name: "float32", typeName: "float32", wantCallTo: "FormatFloat"},
		{name: "float64", typeName: "float64", wantCallTo: "FormatFloat"},

		{name: "bool", typeName: "bool", wantCallTo: "FormatBool"},

		{name: "rune", typeName: "rune", wantCallTo: "string"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sc := newStringConverter()
			ann := createMockAnnotation(tc.typeName, inspector_dto.StringablePrimitive)
			inputExpr := cachedIdent("x")

			result := sc.valueToString(inputExpr, ann)

			require.NotNil(t, result)

			if tc.wantIdentity {

				assert.Equal(t, inputExpr, result, "String type should be identity function")
			} else {

				callExpr, ok := result.(*goast.CallExpr)
				require.True(t, ok, "Expected CallExpr for %s conversion", tc.typeName)

				if tc.wantCallTo == "string" {

					funcIdent, ok := callExpr.Fun.(*goast.Ident)
					require.True(t, ok, "Type cast should use Ident")
					assert.Equal(t, "string", funcIdent.Name)
				} else {

					selector, ok := callExpr.Fun.(*goast.SelectorExpr)
					require.True(t, ok, "strconv calls should use SelectorExpr")
					assert.Equal(t, "strconv", selector.X.(*goast.Ident).Name)
					assert.Equal(t, tc.wantCallTo, selector.Sel.Name)
				}
			}
		})
	}
}

func TestValueToString_Stringer(t *testing.T) {
	t.Parallel()

	sc := newStringConverter()
	ann := createMockAnnotation("MyType", inspector_dto.StringableViaStringer)
	inputExpr := cachedIdent("obj")

	result := sc.valueToString(inputExpr, ann)

	callExpr, ok := result.(*goast.CallExpr)
	require.True(t, ok, "Expected method call")

	selector, ok := callExpr.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	assert.Equal(t, inputExpr, selector.X)
	assert.Equal(t, "String", selector.Sel.Name)
	assert.Empty(t, callExpr.Args, "String() takes no arguments")
}

func TestValueToString_TextMarshaler(t *testing.T) {
	t.Parallel()

	sc := newStringConverter()
	ann := createMockAnnotation("MyType", inspector_dto.StringableViaTextMarshaler)
	inputExpr := cachedIdent("obj")

	result := sc.valueToString(inputExpr, ann)

	callExpr := requireCallExpr(t, result, "IIFE CallExpr")

	funcLit := requireFuncLit(t, callExpr.Fun, "IIFE function literal")

	require.NotNil(t, funcLit.Type.Results)
	require.Len(t, funcLit.Type.Results.List, 1)
	returnType := requireIdent(t, funcLit.Type.Results.List[0].Type, "return type")
	assert.Equal(t, "string", returnType.Name)

	require.NotNil(t, funcLit.Body)
	assert.GreaterOrEqual(t, len(funcLit.Body.List), 2, "Should have MarshalText call and error handling")
}

func TestValueToString_PikoFormatter(t *testing.T) {
	t.Parallel()

	sc := newStringConverter()
	ann := createMockAnnotation("maths.Decimal", inspector_dto.StringableViaPikoFormatter)
	inputExpr := cachedIdent("price")

	result := sc.valueToString(inputExpr, ann)

	callExpr, ok := result.(*goast.CallExpr)
	require.True(t, ok, "Expected method call")

	selector, ok := callExpr.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	assert.Equal(t, inputExpr, selector.X)
	assert.Equal(t, "MustString", selector.Sel.Name)
	assert.Empty(t, callExpr.Args, "MustString() takes no arguments")
}

func TestValueToString_PointerToStringable(t *testing.T) {
	t.Parallel()

	sc := newStringConverter()

	ann := createMockAnnotationWithTypeExpr(
		&goast.StarExpr{X: cachedIdent("MyStringer")},
		inspector_dto.StringableViaStringer,
	)
	ann.IsPointerToStringable = true

	inputExpr := cachedIdent("ptr")

	result := sc.valueToString(inputExpr, ann)

	callExpr, ok := result.(*goast.CallExpr)
	require.True(t, ok, "Expected IIFE for nil-safe pointer handling")

	funcLit, ok := callExpr.Fun.(*goast.FuncLit)
	require.True(t, ok, "Expected FuncLit")

	require.GreaterOrEqual(t, len(funcLit.Body.List), 3, "Should have assign, nil check, and String() call")

	assignStmt, ok := funcLit.Body.List[0].(*goast.AssignStmt)
	require.True(t, ok, "First statement should be _ptr := goExpr")
	require.Len(t, assignStmt.Lhs, 1)
	assert.Equal(t, "_ptr", assignStmt.Lhs[0].(*goast.Ident).Name)

	ifStmt, ok := funcLit.Body.List[1].(*goast.IfStmt)
	require.True(t, ok, "Second statement should be nil check")

	returnStmt, ok := ifStmt.Body.List[0].(*goast.ReturnStmt)
	require.True(t, ok)
	require.Len(t, returnStmt.Results, 1)

	emptyStringLit, ok := returnStmt.Results[0].(*goast.BasicLit)
	require.True(t, ok)
	assert.Equal(t, `""`, emptyStringLit.Value)
}

func TestValueToString_Fallback(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ann  *ast_domain.GoGeneratorAnnotation
		name string
	}{
		{name: "nil annotation", ann: nil},
		{name: "nil ResolvedType", ann: &ast_domain.GoGeneratorAnnotation{ResolvedType: nil}},
		{name: "unknown stringability", ann: createMockAnnotation("UnknownType", 999)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sc := newStringConverter()
			inputExpr := cachedIdent("x")

			result := sc.valueToString(inputExpr, tc.ann)

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok, "Should fallback to runtime helper")

			selector, ok := callExpr.Fun.(*goast.SelectorExpr)
			require.True(t, ok)
			assert.Equal(t, "pikoruntime", selector.X.(*goast.Ident).Name)
			assert.Equal(t, "ValueToString", selector.Sel.Name)
			require.Len(t, callExpr.Args, 1)
			assert.Equal(t, inputExpr, callExpr.Args[0])
		})
	}
}

func TestConvertPrimitiveToString_DispatchTable(t *testing.T) {
	t.Parallel()

	sc := newStringConverter()

	primitiveTypes := []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte",
		"float32", "float64",
		"bool",
		"string",
		"rune",
	}

	for _, typeName := range primitiveTypes {
		t.Run(typeName, func(t *testing.T) {
			t.Parallel()

			ann := createMockAnnotation(typeName, inspector_dto.StringablePrimitive)
			result := sc.valueToString(cachedIdent("x"), ann)

			assert.NotNil(t, result, "Primitive type %s should have conversion", typeName)

			if typeName == "string" {
				assert.IsType(t, &goast.Ident{}, result, "String should be identity")
			} else {
				assert.IsType(t, &goast.CallExpr{}, result, "Non-string primitive should generate call")
			}
		})
	}
}

func BenchmarkValueToString_StringIdentity(b *testing.B) {
	sc := newStringConverter()
	ann := createMockAnnotation("string", inspector_dto.StringablePrimitive)
	expression := cachedIdent("x")

	b.ResetTimer()
	for b.Loop() {
		_ = sc.valueToString(expression, ann)
	}
}

func BenchmarkValueToString_Int64Conversion(b *testing.B) {
	sc := newStringConverter()
	ann := createMockAnnotation("int64", inspector_dto.StringablePrimitive)
	expression := cachedIdent("x")

	b.ResetTimer()
	for b.Loop() {
		_ = sc.valueToString(expression, ann)
	}
}

func BenchmarkValueToString_Stringer(b *testing.B) {
	sc := newStringConverter()
	ann := createMockAnnotation("MyType", inspector_dto.StringableViaStringer)
	expression := cachedIdent("obj")

	b.ResetTimer()
	for b.Loop() {
		_ = sc.valueToString(expression, ann)
	}
}

func TestValueToString_JSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		typeExpr     goast.Expr
		wantFallback string
	}{
		{
			name:         "map type returns {} fallback",
			typeExpr:     &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("string")},
			wantFallback: "{}",
		},
		{
			name:         "slice type returns [] fallback",
			typeExpr:     &goast.ArrayType{Elt: cachedIdent("int")},
			wantFallback: "[]",
		},
		{
			name:         "pointer to map returns {} fallback",
			typeExpr:     &goast.StarExpr{X: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")}},
			wantFallback: "{}",
		},
		{
			name:         "pointer to slice returns [] fallback",
			typeExpr:     &goast.StarExpr{X: &goast.ArrayType{Elt: cachedIdent("string")}},
			wantFallback: "[]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sc := newStringConverter()
			ann := createMockAnnotationWithTypeExpr(tc.typeExpr, inspector_dto.StringableViaJSON)
			inputExpr := cachedIdent("data")

			result := sc.valueToString(inputExpr, ann)

			callExpr := requireCallExpr(t, result, "IIFE CallExpr")

			funcLit := requireFuncLit(t, callExpr.Fun, "IIFE function literal")

			require.NotNil(t, funcLit.Type.Results)
			require.Len(t, funcLit.Type.Results.List, 1)
			returnType := requireIdent(t, funcLit.Type.Results.List[0].Type, "return type")
			assert.Equal(t, "string", returnType.Name)

			require.NotNil(t, funcLit.Body)
			require.GreaterOrEqual(t, len(funcLit.Body.List), 3, "Should have assign, if-error, and return")

			assignStmt, ok := funcLit.Body.List[0].(*goast.AssignStmt)
			require.True(t, ok, "First statement should be assignment")
			require.Len(t, assignStmt.Lhs, 2, "Should assign data and err")

			rhsCall, ok := assignStmt.Rhs[0].(*goast.CallExpr)
			require.True(t, ok)
			selector, ok := rhsCall.Fun.(*goast.SelectorExpr)
			require.True(t, ok)
			assert.Equal(t, "json", selector.X.(*goast.Ident).Name)
			assert.Equal(t, "Marshal", selector.Sel.Name)

			ifStmt, ok := funcLit.Body.List[1].(*goast.IfStmt)
			require.True(t, ok, "Second statement should be if err != nil")

			returnStmt, ok := ifStmt.Body.List[0].(*goast.ReturnStmt)
			require.True(t, ok)
			fallbackLit, ok := returnStmt.Results[0].(*goast.BasicLit)
			require.True(t, ok)
			assert.Equal(t, `"`+tc.wantFallback+`"`, fallbackLit.Value)
		})
	}
}

func TestDetermineJSONFallback(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		typeInfo *ast_domain.ResolvedTypeInfo
		want     string
	}{
		{
			name:     "nil typeInfo returns null",
			typeInfo: nil,
			want:     "null",
		},
		{
			name:     "nil typeExpr returns null",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			want:     "null",
		},
		{
			name: "map type returns {}",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")},
			},
			want: "{}",
		},
		{
			name: "slice type returns []",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Elt: cachedIdent("string")},
			},
			want: "[]",
		},
		{
			name: "array type returns []",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Len: &goast.BasicLit{Value: "10"}, Elt: cachedIdent("int")},
			},
			want: "[]",
		},
		{
			name: "pointer to map returns {}",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.StarExpr{X: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("bool")}},
			},
			want: "{}",
		},
		{
			name: "pointer to slice returns []",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.StarExpr{X: &goast.ArrayType{Elt: cachedIdent("float64")}},
			},
			want: "[]",
		},
		{
			name: "double pointer to map returns {}",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.StarExpr{X: &goast.StarExpr{X: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")}}},
			},
			want: "{}",
		},
		{
			name: "named type returns null (unknown JSON shape)",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: cachedIdent("time.Time"),
			},
			want: "null",
		},
		{
			name: "selector type returns null",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.SelectorExpr{X: cachedIdent("pkg"), Sel: cachedIdent("Type")},
			},
			want: "null",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := determineJSONFallback(tc.typeInfo)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetJSONFallbackForExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		typeExpr goast.Expr
		want     string
	}{
		{
			name:     "map returns {}",
			typeExpr: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")},
			want:     "{}",
		},
		{
			name:     "slice returns []",
			typeExpr: &goast.ArrayType{Elt: cachedIdent("string")},
			want:     "[]",
		},
		{
			name:     "nested pointer to map returns {}",
			typeExpr: &goast.StarExpr{X: &goast.StarExpr{X: &goast.StarExpr{X: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")}}}},
			want:     "{}",
		},
		{
			name:     "ident returns null",
			typeExpr: cachedIdent("MyType"),
			want:     "null",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := getJSONFallbackForExpr(tc.typeExpr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetPackageAliasFromType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		typeExpr goast.Expr
		fallback string
		want     string
	}{
		{
			name: "SelectorExpr pkg.Type returns pkg",
			typeExpr: &goast.SelectorExpr{
				X:   cachedIdent("mypkg"),
				Sel: cachedIdent("MyType"),
			},
			fallback: "fallback",
			want:     "mypkg",
		},
		{
			name: "StarExpr wrapping SelectorExpr returns pkg",
			typeExpr: &goast.StarExpr{
				X: &goast.SelectorExpr{
					X:   cachedIdent("otherpkg"),
					Sel: cachedIdent("OtherType"),
				},
			},
			fallback: "fallback",
			want:     "otherpkg",
		},
		{
			name: "double StarExpr wrapping SelectorExpr returns pkg",
			typeExpr: &goast.StarExpr{
				X: &goast.StarExpr{
					X: &goast.SelectorExpr{
						X:   cachedIdent("deep"),
						Sel: cachedIdent("Type"),
					},
				},
			},
			fallback: "fallback",
			want:     "deep",
		},
		{
			name:     "plain Ident returns fallback",
			typeExpr: cachedIdent("string"),
			fallback: "defaultPackage",
			want:     "defaultPackage",
		},
		{
			name:     "nil returns fallback",
			typeExpr: nil,
			fallback: "fb",
			want:     "fb",
		},
		{
			name:     "MapType returns fallback",
			typeExpr: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")},
			fallback: "mapFallback",
			want:     "mapFallback",
		},
		{
			name:     "empty fallback with plain Ident returns empty",
			typeExpr: cachedIdent("int"),
			fallback: "",
			want:     "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := getPackageAliasFromType(tc.typeExpr, tc.fallback)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestConvertPrimitiveToString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		typeName     string
		wantCallTo   string
		wantIdentity bool
	}{
		{
			name:         "string returns identity (no conversion)",
			typeName:     "string",
			wantIdentity: true,
		},
		{
			name:       "int uses FormatInt",
			typeName:   "int",
			wantCallTo: "FormatInt",
		},
		{
			name:       "int64 uses FormatInt",
			typeName:   "int64",
			wantCallTo: "FormatInt",
		},
		{
			name:       "float64 uses FormatFloat",
			typeName:   "float64",
			wantCallTo: "FormatFloat",
		},
		{
			name:       "float32 uses FormatFloat",
			typeName:   "float32",
			wantCallTo: "FormatFloat",
		},
		{
			name:       "bool uses FormatBool",
			typeName:   "bool",
			wantCallTo: "FormatBool",
		},
		{
			name:       "uint uses FormatUint",
			typeName:   "uint",
			wantCallTo: "FormatUint",
		},
		{
			name:       "byte uses FormatUint",
			typeName:   "byte",
			wantCallTo: "FormatUint",
		},
		{
			name:       "rune uses string cast",
			typeName:   "rune",
			wantCallTo: "string",
		},
		{
			name:       "unknown type falls back to ValueToString",
			typeName:   "CustomType",
			wantCallTo: "ValueToString",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sc := newStringConverter()
			typeInfo := &ast_domain.ResolvedTypeInfo{
				TypeExpression: cachedIdent(tc.typeName),
			}
			inputExpr := cachedIdent("val")

			result := sc.convertPrimitiveToString(inputExpr, typeInfo)
			require.NotNil(t, result)

			if tc.wantIdentity {
				assert.Equal(t, inputExpr, result, "String type should be identity")
				return
			}

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok, "Expected CallExpr for %s conversion, got %T", tc.typeName, result)

			switch tc.wantCallTo {
			case "string":
				funcIdent, ok := callExpr.Fun.(*goast.Ident)
				require.True(t, ok, "Type cast should use Ident")
				assert.Equal(t, "string", funcIdent.Name)
			case "ValueToString":
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "Fallback should use SelectorExpr")
				assert.Equal(t, "pikoruntime", selector.X.(*goast.Ident).Name)
				assert.Equal(t, "ValueToString", selector.Sel.Name)
			default:
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "strconv calls should use SelectorExpr")
				assert.Equal(t, "strconv", selector.X.(*goast.Ident).Name)
				assert.Equal(t, tc.wantCallTo, selector.Sel.Name)
			}
		})
	}
}

func BenchmarkValueToString_JSON(b *testing.B) {
	sc := newStringConverter()
	ann := createMockAnnotationWithTypeExpr(
		&goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")},
		inspector_dto.StringableViaJSON,
	)
	expression := cachedIdent("data")

	b.ResetTimer()
	for b.Loop() {
		_ = sc.valueToString(expression, ann)
	}
}
