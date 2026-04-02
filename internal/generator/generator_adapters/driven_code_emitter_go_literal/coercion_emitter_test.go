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
	"go/token"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func setupCoercionEmitter() (*CoercionEmitter, *emitter) {
	em := &emitter{ctx: NewEmitterContext()}
	ee := &expressionEmitter{emitter: em}
	return newCoercionEmitter(ee), em
}

func TestNewCoercionEmitter(t *testing.T) {
	t.Parallel()

	ce, _ := setupCoercionEmitter()

	require.NotNil(t, ce, "newCoercionEmitter should return non-nil")
	require.NotNil(t, ce.ee, "expression emitter reference should be set")
}

func TestGetSourceType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ann  *ast_domain.GoGeneratorAnnotation
		want string
	}{
		{
			name: "nil annotation returns any",
			ann:  nil,
			want: "any",
		},
		{
			name: "nil ResolvedType returns any",
			ann:  &ast_domain.GoGeneratorAnnotation{ResolvedType: nil},
			want: "any",
		},
		{
			name: "nil TypeExpr returns any",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			},
			want: "any",
		},
		{
			name: "Ident type returns type name",
			ann:  createMockAnnotation("int64", inspector_dto.StringablePrimitive),
			want: "int64",
		},
		{
			name: "SelectorExpr type returns qualified name",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.SelectorExpr{
						X:   cachedIdent("maths"),
						Sel: cachedIdent("Decimal"),
					},
				},
			},
			want: "maths.Decimal",
		},
		{
			name: "SelectorExpr with non-Ident X returns any",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.SelectorExpr{
						X:   &goast.BasicLit{Kind: token.STRING, Value: `"pkg"`},
						Sel: cachedIdent("Type"),
					},
				},
			},
			want: "any",
		},
		{
			name: "unsupported type expression returns any",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.StarExpr{X: cachedIdent("int")},
				},
			},
			want: "any",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, _ := setupCoercionEmitter()
			got := ce.getSourceType(tc.ann)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEmitStringCoercion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		sourceType          string
		wantStrconvFunction string
		wantMethod          string
		wantPassthrough     bool
		wantRuntime         bool
	}{
		{
			name:            "string passthrough",
			sourceType:      StringTypeName,
			wantPassthrough: true,
		},
		{
			name:                "int uses Itoa",
			sourceType:          IntTypeName,
			wantStrconvFunction: strconvItoa,
		},
		{
			name:                "int64 uses FormatInt",
			sourceType:          Int64TypeName,
			wantStrconvFunction: strconvFormatInt,
		},
		{
			name:                "int32 uses FormatInt with cast",
			sourceType:          Int32TypeName,
			wantStrconvFunction: strconvFormatInt,
		},
		{
			name:                "int16 uses FormatInt with cast",
			sourceType:          Int16TypeName,
			wantStrconvFunction: strconvFormatInt,
		},
		{
			name:                "int8 uses FormatInt with cast",
			sourceType:          Int8TypeName,
			wantStrconvFunction: strconvFormatInt,
		},
		{
			name:                "uint uses FormatUint",
			sourceType:          UintTypeName,
			wantStrconvFunction: strconvFormatUint,
		},
		{
			name:                "uint64 uses FormatUint",
			sourceType:          Uint64TypeName,
			wantStrconvFunction: strconvFormatUint,
		},
		{
			name:                "uint32 uses FormatUint",
			sourceType:          Uint32TypeName,
			wantStrconvFunction: strconvFormatUint,
		},
		{
			name:                "uint16 uses FormatUint",
			sourceType:          Uint16TypeName,
			wantStrconvFunction: strconvFormatUint,
		},
		{
			name:                "uint8 uses FormatUint",
			sourceType:          Uint8TypeName,
			wantStrconvFunction: strconvFormatUint,
		},
		{
			name:                "byte uses FormatUint",
			sourceType:          ByteTypeName,
			wantStrconvFunction: strconvFormatUint,
		},
		{
			name:                "float64 uses FormatFloat",
			sourceType:          Float64TypeName,
			wantStrconvFunction: strconvFormatFloat,
		},
		{
			name:                "float32 uses FormatFloat",
			sourceType:          Float32TypeName,
			wantStrconvFunction: strconvFormatFloat,
		},
		{
			name:                "bool uses FormatBool",
			sourceType:          BoolTypeName,
			wantStrconvFunction: strconvFormatBool,
		},
		{
			name:       "maths.Decimal uses MustString method",
			sourceType: mathsDecimalTypeName,
			wantMethod: mathsMustString,
		},
		{
			name:       "maths.BigInt uses MustString method",
			sourceType: mathsBigIntTypeName,
			wantMethod: mathsMustString,
		},
		{
			name:        "unknown type uses runtime helper",
			sourceType:  "any",
			wantRuntime: true,
		},
		{
			name:        "custom type uses runtime helper",
			sourceType:  "MyCustomType",
			wantRuntime: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, _ := setupCoercionEmitter()
			inputExpr := cachedIdent("x")

			result := ce.emitStringCoercion(inputExpr, tc.sourceType)

			require.NotNil(t, result)

			if tc.wantPassthrough {
				assert.Equal(t, inputExpr, result, "string should be identity")
				return
			}

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok, "Expected CallExpr for %s conversion", tc.sourceType)

			if tc.wantStrconvFunction != "" {
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "strconv calls should use SelectorExpr")
				assert.Equal(t, pkgStrconv, selector.X.(*goast.Ident).Name)
				assert.Equal(t, tc.wantStrconvFunction, selector.Sel.Name)
			}

			if tc.wantMethod != "" {
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "method calls should use SelectorExpr")
				assert.Equal(t, tc.wantMethod, selector.Sel.Name)
			}

			if tc.wantRuntime {
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "runtime calls should use SelectorExpr")
				assert.Equal(t, runtimePackageName, selector.X.(*goast.Ident).Name)
				assert.Equal(t, "CoerceToString", selector.Sel.Name)
			}
		})
	}
}

func TestEmitIntCoercion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		sourceType      string
		wantPassthrough bool
		wantCast        bool
		wantIIFE        bool
		wantRuntime     bool
	}{
		{
			name:            "int passthrough",
			sourceType:      IntTypeName,
			wantPassthrough: true,
		},
		{
			name:       "int64 cast",
			sourceType: Int64TypeName,
			wantCast:   true,
		},
		{
			name:       "int32 cast",
			sourceType: Int32TypeName,
			wantCast:   true,
		},
		{
			name:       "float64 cast",
			sourceType: Float64TypeName,
			wantCast:   true,
		},
		{
			name:       "float32 cast",
			sourceType: Float32TypeName,
			wantCast:   true,
		},
		{
			name:       "uint64 cast",
			sourceType: Uint64TypeName,
			wantCast:   true,
		},
		{
			name:       "bool uses IIFE",
			sourceType: BoolTypeName,
			wantIIFE:   true,
		},
		{
			name:       "string uses IIFE",
			sourceType: StringTypeName,
			wantIIFE:   true,
		},
		{
			name:       "maths.Decimal uses method and cast",
			sourceType: mathsDecimalTypeName,
			wantCast:   true,
		},
		{
			name:        "any uses runtime helper",
			sourceType:  "any",
			wantRuntime: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, _ := setupCoercionEmitter()
			inputExpr := cachedIdent("x")

			result := ce.emitIntCoercion(inputExpr, tc.sourceType)

			require.NotNil(t, result)

			if tc.wantPassthrough {
				assert.Equal(t, inputExpr, result)
				return
			}

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok, "Expected CallExpr")

			if tc.wantCast {
				if identifier, ok := callExpr.Fun.(*goast.Ident); ok {
					assert.Equal(t, IntTypeName, identifier.Name)
				}
			}

			if tc.wantIIFE {
				_, ok := callExpr.Fun.(*goast.FuncLit)
				assert.True(t, ok, "Expected IIFE for %s", tc.sourceType)
			}

			if tc.wantRuntime {
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok)
				assert.Equal(t, runtimePackageName, selector.X.(*goast.Ident).Name)
				assert.Equal(t, "CoerceToInt", selector.Sel.Name)
			}
		})
	}
}

func TestEmitInt64Coercion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		sourceType      string
		wantPassthrough bool
		wantCast        bool
		wantMethod      bool
	}{
		{
			name:            "int64 passthrough",
			sourceType:      Int64TypeName,
			wantPassthrough: true,
		},
		{
			name:       "int cast",
			sourceType: IntTypeName,
			wantCast:   true,
		},
		{
			name:       "float64 cast",
			sourceType: Float64TypeName,
			wantCast:   true,
		},
		{
			name:       "maths.Decimal uses method",
			sourceType: mathsDecimalTypeName,
			wantMethod: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, _ := setupCoercionEmitter()
			inputExpr := cachedIdent("x")

			result := ce.emitInt64Coercion(inputExpr, tc.sourceType)

			require.NotNil(t, result)

			if tc.wantPassthrough {
				assert.Equal(t, inputExpr, result)
				return
			}

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok)

			if tc.wantCast {
				identifier, ok := callExpr.Fun.(*goast.Ident)
				require.True(t, ok)
				assert.Equal(t, Int64TypeName, identifier.Name)
			}

			if tc.wantMethod {
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok)
				assert.Equal(t, mathsMustInt64, selector.Sel.Name)
			}
		})
	}
}

func TestEmitInt64Coercion_Extended(t *testing.T) {
	t.Parallel()

	argExpr := cachedIdent("val")

	testCases := []struct {
		checkFunction func(t *testing.T, result goast.Expr)
		name          string
		sourceType    string
	}{
		{
			name:       "int64 passthrough",
			sourceType: Int64TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				assert.Equal(t, argExpr, result, "int64 should pass through unchanged")
			},
		},
		{
			name:       "int8 cast to int64",
			sourceType: Int8TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Int64TypeName, identifier.Name)
			},
		},
		{
			name:       "int16 cast to int64",
			sourceType: Int16TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Int64TypeName, identifier.Name)
			},
		},
		{
			name:       "int32 cast to int64",
			sourceType: Int32TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Int64TypeName, identifier.Name)
			},
		},
		{
			name:       "float32 cast to int64",
			sourceType: Float32TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Int64TypeName, identifier.Name)
			},
		},
		{
			name:       "float64 cast to int64",
			sourceType: Float64TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Int64TypeName, identifier.Name)
			},
		},
		{
			name:       "string produces IIFE",
			sourceType: StringTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr wrapping FuncLit")
				_, ok = call.Fun.(*goast.FuncLit)
				assert.True(t, ok, "expected FuncLit for string-to-int IIFE")
			},
		},
		{
			name:       "bool produces IIFE",
			sourceType: BoolTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr wrapping FuncLit")
				_, ok = call.Fun.(*goast.FuncLit)
				assert.True(t, ok, "expected FuncLit for bool-to-int IIFE")
			},
		},
		{
			name:       "maths.Decimal uses MustInt64 method",
			sourceType: mathsDecimalTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for method call")
				selector, ok := call.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "should be a SelectorExpr for method")
				assert.Equal(t, mathsMustInt64, selector.Sel.Name)
			},
		},
		{
			name:       "maths.BigInt uses MustInt64 method",
			sourceType: mathsBigIntTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for method call")
				selector, ok := call.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "should be a SelectorExpr for method")
				assert.Equal(t, mathsMustInt64, selector.Sel.Name)
			},
		},
		{
			name:       "unknown type uses runtime helper",
			sourceType: "any",
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr")
				selector, ok := call.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "should be a SelectorExpr")
				assert.Equal(t, runtimePackageName, selector.X.(*goast.Ident).Name)
				assert.Equal(t, "CoerceToInt64", selector.Sel.Name)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ce, _ := setupCoercionEmitter()
			result := ce.emitInt64Coercion(argExpr, tc.sourceType)
			require.NotNil(t, result)
			tc.checkFunction(t, result)
		})
	}
}

func TestEmitFloat64Coercion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		sourceType      string
		wantPassthrough bool
		wantCast        bool
		wantIIFE        bool
		wantMethod      bool
	}{
		{
			name:            "float64 passthrough",
			sourceType:      Float64TypeName,
			wantPassthrough: true,
		},
		{
			name:       "int cast",
			sourceType: IntTypeName,
			wantCast:   true,
		},
		{
			name:       "int64 cast",
			sourceType: Int64TypeName,
			wantCast:   true,
		},
		{
			name:       "float32 cast",
			sourceType: Float32TypeName,
			wantCast:   true,
		},
		{
			name:       "bool uses IIFE",
			sourceType: BoolTypeName,
			wantIIFE:   true,
		},
		{
			name:       "string uses IIFE",
			sourceType: StringTypeName,
			wantIIFE:   true,
		},
		{
			name:       "maths.Decimal uses method",
			sourceType: mathsDecimalTypeName,
			wantMethod: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, _ := setupCoercionEmitter()
			inputExpr := cachedIdent("x")

			result := ce.emitFloat64Coercion(inputExpr, tc.sourceType)

			require.NotNil(t, result)

			if tc.wantPassthrough {
				assert.Equal(t, inputExpr, result)
				return
			}

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok)

			if tc.wantCast {
				identifier, ok := callExpr.Fun.(*goast.Ident)
				require.True(t, ok)
				assert.Equal(t, Float64TypeName, identifier.Name)
			}

			if tc.wantIIFE {
				_, ok := callExpr.Fun.(*goast.FuncLit)
				assert.True(t, ok, "Expected IIFE")
			}

			if tc.wantMethod {
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok)
				assert.Equal(t, mathsMustFloat64, selector.Sel.Name)
			}
		})
	}
}

func TestEmitFloat64Coercion_Extended(t *testing.T) {
	t.Parallel()

	argExpr := cachedIdent("val")

	testCases := []struct {
		checkFunction func(t *testing.T, result goast.Expr)
		name          string
		sourceType    string
	}{
		{
			name:       "maths.BigInt casts MustInt64 result to float64",
			sourceType: mathsBigIntTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for outer cast")
				identifier := requireIdent(t, call.Fun, "outer cast function")
				assert.Equal(t, Float64TypeName, identifier.Name)

				require.Len(t, call.Args, 1)
				innerCall, ok := call.Args[0].(*goast.CallExpr)
				require.True(t, ok, "inner argument should be a CallExpr")
				selector, ok := innerCall.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "inner call should use SelectorExpr")
				assert.Equal(t, mathsMustInt64, selector.Sel.Name)
			},
		},
		{
			name:       "unknown type uses runtime helper",
			sourceType: "any",
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr")
				selector, ok := call.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "should be a SelectorExpr")
				assert.Equal(t, runtimePackageName, selector.X.(*goast.Ident).Name)
				assert.Equal(t, "CoerceToFloat64", selector.Sel.Name)
			},
		},
		{
			name:       "int32 cast to float64",
			sourceType: Int32TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Float64TypeName, identifier.Name)
			},
		},
		{
			name:       "uint8 cast to float64",
			sourceType: Uint8TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Float64TypeName, identifier.Name)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ce, _ := setupCoercionEmitter()
			result := ce.emitFloat64Coercion(argExpr, tc.sourceType)
			require.NotNil(t, result)
			tc.checkFunction(t, result)
		})
	}
}

func TestEmitDecimalCoercion_Extended(t *testing.T) {
	t.Parallel()

	argExpr := cachedIdent("val")

	testCases := []struct {
		checkFunction func(t *testing.T, result goast.Expr)
		name          string
		sourceType    string
	}{
		{
			name:       "float32 casts to float64 then NewDecimalFromFloat",
			sourceType: Float32TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr")
				selector, ok := call.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "should be a SelectorExpr for maths constructor")
				assert.Equal(t, pkgMaths, selector.X.(*goast.Ident).Name)
				assert.Equal(t, mathsNewDecimalFromFloat, selector.Sel.Name)

				require.Len(t, call.Args, 1)
				innerCall, ok := call.Args[0].(*goast.CallExpr)
				require.True(t, ok, "inner argument should be a CallExpr for float64 cast")
				innerIdent := requireIdent(t, innerCall.Fun, "inner cast function")
				assert.Equal(t, Float64TypeName, innerIdent.Name)
			},
		},
		{
			name:       "int32 casts to int64 then NewDecimalFromInt",
			sourceType: Int32TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr")
				selector, ok := call.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "should be a SelectorExpr for maths constructor")
				assert.Equal(t, pkgMaths, selector.X.(*goast.Ident).Name)
				assert.Equal(t, mathsNewDecimalFromInt, selector.Sel.Name)

				require.Len(t, call.Args, 1)
				innerCall, ok := call.Args[0].(*goast.CallExpr)
				require.True(t, ok, "inner argument should be a CallExpr for int64 cast")
				innerIdent := requireIdent(t, innerCall.Fun, "inner cast function")
				assert.Equal(t, Int64TypeName, innerIdent.Name)
			},
		},
		{
			name:       "unknown type uses runtime helper",
			sourceType: "any",
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr")
				selector, ok := call.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "should be a SelectorExpr")
				assert.Equal(t, runtimePackageName, selector.X.(*goast.Ident).Name)
				assert.Equal(t, "CoerceToDecimal", selector.Sel.Name)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ce, _ := setupCoercionEmitter()
			result := ce.emitDecimalCoercion(argExpr, tc.sourceType)
			require.NotNil(t, result)
			tc.checkFunction(t, result)
		})
	}
}

func TestEmitFloat32Coercion(t *testing.T) {
	t.Parallel()

	argExpr := cachedIdent("val")

	testCases := []struct {
		checkFunction func(t *testing.T, result goast.Expr)
		name          string
		sourceType    string
	}{
		{
			name:       "same type passthrough",
			sourceType: Float32TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				assert.Equal(t, argExpr, result, "float32 should pass through unchanged")
			},
		},
		{
			name:       "int family cast",
			sourceType: IntTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Float32TypeName, identifier.Name)
			},
		},
		{
			name:       "int64 cast",
			sourceType: Int64TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Float32TypeName, identifier.Name)
			},
		},
		{
			name:       "float64 cast",
			sourceType: Float64TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Float32TypeName, identifier.Name)
			},
		},
		{
			name:       "bool produces IIFE",
			sourceType: BoolTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr wrapping FuncLit")
				_, ok = call.Fun.(*goast.FuncLit)
				assert.True(t, ok, "expected FuncLit for bool-to-float IIFE")
			},
		},
		{
			name:       "string produces IIFE",
			sourceType: StringTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr wrapping FuncLit")
				_, ok = call.Fun.(*goast.FuncLit)
				assert.True(t, ok, "expected FuncLit for string-to-float IIFE")
			},
		},
		{
			name:       "maths decimal casts MustFloat64 result",
			sourceType: mathsDecimalTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for cast")
				identifier := requireIdent(t, call.Fun, "outer cast")
				assert.Equal(t, Float32TypeName, identifier.Name)
			},
		},
		{
			name:       "maths bigint casts MustInt64 result",
			sourceType: mathsBigIntTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for cast")
				identifier := requireIdent(t, call.Fun, "outer cast")
				assert.Equal(t, Float32TypeName, identifier.Name)
			},
		},
		{
			name:       "unknown type uses runtime helper",
			sourceType: "any",
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr")
				selector, ok := call.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "should be a SelectorExpr")
				assert.Equal(t, runtimePackageName, selector.X.(*goast.Ident).Name)
				assert.Equal(t, "CoerceToFloat32", selector.Sel.Name)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ce, _ := setupCoercionEmitter()
			result := ce.emitFloat32Coercion(argExpr, tc.sourceType)
			require.NotNil(t, result)
			tc.checkFunction(t, result)
		})
	}
}

func TestEmitInt32Coercion(t *testing.T) {
	t.Parallel()

	argExpr := cachedIdent("val")

	testCases := []struct {
		checkFunction func(t *testing.T, result goast.Expr)
		name          string
		sourceType    string
	}{
		{
			name:       "same type passthrough",
			sourceType: Int32TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				assert.Equal(t, argExpr, result, "int32 should pass through unchanged")
			},
		},
		{
			name:       "int family cast",
			sourceType: IntTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Int32TypeName, identifier.Name)
			},
		},
		{
			name:       "int64 cast",
			sourceType: Int64TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Int32TypeName, identifier.Name)
			},
		},
		{
			name:       "float64 cast",
			sourceType: Float64TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Int32TypeName, identifier.Name)
			},
		},
		{
			name:       "bool produces IIFE",
			sourceType: BoolTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr wrapping FuncLit")
				_, ok = call.Fun.(*goast.FuncLit)
				assert.True(t, ok, "expected FuncLit for bool-to-int IIFE")
			},
		},
		{
			name:       "string produces IIFE",
			sourceType: StringTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr wrapping FuncLit")
				_, ok = call.Fun.(*goast.FuncLit)
				assert.True(t, ok, "expected FuncLit for string-to-int IIFE")
			},
		},
		{
			name:       "maths decimal casts MustInt64 result",
			sourceType: mathsDecimalTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for cast")
				identifier := requireIdent(t, call.Fun, "outer cast")
				assert.Equal(t, Int32TypeName, identifier.Name)
			},
		},
		{
			name:       "maths bigint casts MustInt64 result",
			sourceType: mathsBigIntTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for cast")
				identifier := requireIdent(t, call.Fun, "outer cast")
				assert.Equal(t, Int32TypeName, identifier.Name)
			},
		},
		{
			name:       "unknown type uses runtime helper",
			sourceType: "any",
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr")
				selector, ok := call.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "should be a SelectorExpr")
				assert.Equal(t, runtimePackageName, selector.X.(*goast.Ident).Name)
				assert.Equal(t, "CoerceToInt32", selector.Sel.Name)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ce, _ := setupCoercionEmitter()
			result := ce.emitInt32Coercion(argExpr, tc.sourceType)
			require.NotNil(t, result)
			tc.checkFunction(t, result)
		})
	}
}

func TestEmitInt16Coercion(t *testing.T) {
	t.Parallel()

	argExpr := cachedIdent("val")

	testCases := []struct {
		checkFunction func(t *testing.T, result goast.Expr)
		name          string
		sourceType    string
	}{
		{
			name:       "same type passthrough",
			sourceType: Int16TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				assert.Equal(t, argExpr, result, "int16 should pass through unchanged")
			},
		},
		{
			name:       "int family cast",
			sourceType: IntTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Int16TypeName, identifier.Name)
			},
		},
		{
			name:       "int64 cast",
			sourceType: Int64TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Int16TypeName, identifier.Name)
			},
		},
		{
			name:       "float64 cast",
			sourceType: Float64TypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for type cast")
				identifier := requireIdent(t, call.Fun, "cast function")
				assert.Equal(t, Int16TypeName, identifier.Name)
			},
		},
		{
			name:       "bool produces IIFE",
			sourceType: BoolTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr wrapping FuncLit")
				_, ok = call.Fun.(*goast.FuncLit)
				assert.True(t, ok, "expected FuncLit for bool-to-int IIFE")
			},
		},
		{
			name:       "string produces IIFE",
			sourceType: StringTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr wrapping FuncLit")
				_, ok = call.Fun.(*goast.FuncLit)
				assert.True(t, ok, "expected FuncLit for string-to-int IIFE")
			},
		},
		{
			name:       "maths decimal casts MustInt64 result",
			sourceType: mathsDecimalTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for cast")
				identifier := requireIdent(t, call.Fun, "outer cast")
				assert.Equal(t, Int16TypeName, identifier.Name)
			},
		},
		{
			name:       "maths bigint casts MustInt64 result",
			sourceType: mathsBigIntTypeName,
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr for cast")
				identifier := requireIdent(t, call.Fun, "outer cast")
				assert.Equal(t, Int16TypeName, identifier.Name)
			},
		},
		{
			name:       "unknown type uses runtime helper",
			sourceType: "any",
			checkFunction: func(t *testing.T, result goast.Expr) {
				call, ok := result.(*goast.CallExpr)
				require.True(t, ok, "should be a CallExpr")
				selector, ok := call.Fun.(*goast.SelectorExpr)
				require.True(t, ok, "should be a SelectorExpr")
				assert.Equal(t, runtimePackageName, selector.X.(*goast.Ident).Name)
				assert.Equal(t, "CoerceToInt16", selector.Sel.Name)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ce, _ := setupCoercionEmitter()
			result := ce.emitInt16Coercion(argExpr, tc.sourceType)
			require.NotNil(t, result)
			tc.checkFunction(t, result)
		})
	}
}

func TestEmitBoolCoercion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		sourceType      string
		wantPassthrough bool
		wantIIFE        bool
		wantBinary      bool
		wantRuntime     bool
	}{
		{
			name:            "bool passthrough",
			sourceType:      BoolTypeName,
			wantPassthrough: true,
		},
		{
			name:       "string uses IIFE",
			sourceType: StringTypeName,
			wantIIFE:   true,
		},
		{
			name:       "int uses binary comparison",
			sourceType: IntTypeName,
			wantBinary: true,
		},
		{
			name:       "int64 uses binary comparison",
			sourceType: Int64TypeName,
			wantBinary: true,
		},
		{
			name:       "uint uses binary comparison",
			sourceType: UintTypeName,
			wantBinary: true,
		},
		{
			name:       "float64 uses binary comparison",
			sourceType: Float64TypeName,
			wantBinary: true,
		},
		{
			name:        "any uses runtime helper",
			sourceType:  "any",
			wantRuntime: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, _ := setupCoercionEmitter()
			inputExpr := cachedIdent("x")

			result := ce.emitBoolCoercion(inputExpr, tc.sourceType)

			require.NotNil(t, result)

			if tc.wantPassthrough {
				assert.Equal(t, inputExpr, result)
				return
			}

			if tc.wantIIFE {
				callExpr, ok := result.(*goast.CallExpr)
				require.True(t, ok)
				_, ok = callExpr.Fun.(*goast.FuncLit)
				assert.True(t, ok, "Expected IIFE for string->bool")
			}

			if tc.wantBinary {
				binaryExpr, ok := result.(*goast.BinaryExpr)
				require.True(t, ok, "Expected BinaryExpr for numeric->bool")
				assert.Equal(t, token.NEQ, binaryExpr.Op)
			}

			if tc.wantRuntime {
				callExpr, ok := result.(*goast.CallExpr)
				require.True(t, ok)
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok)
				assert.Equal(t, "CoerceToBool", selector.Sel.Name)
			}
		})
	}
}

func TestEmitDecimalCoercion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		sourceType      string
		wantConstructor string
		wantMethod      string
		wantPassthrough bool
	}{
		{
			name:            "maths.Decimal passthrough",
			sourceType:      mathsDecimalTypeName,
			wantPassthrough: true,
		},
		{
			name:            "int64 uses NewDecimalFromInt",
			sourceType:      Int64TypeName,
			wantConstructor: mathsNewDecimalFromInt,
		},
		{
			name:            "int uses NewDecimalFromInt with cast",
			sourceType:      IntTypeName,
			wantConstructor: mathsNewDecimalFromInt,
		},
		{
			name:            "float64 uses NewDecimalFromFloat",
			sourceType:      Float64TypeName,
			wantConstructor: mathsNewDecimalFromFloat,
		},
		{
			name:            "string uses NewDecimalFromString",
			sourceType:      StringTypeName,
			wantConstructor: mathsNewDecimalFromString,
		},
		{
			name:       "maths.BigInt uses ToDecimal method",
			sourceType: mathsBigIntTypeName,
			wantMethod: mathsToDecimal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, em := setupCoercionEmitter()
			inputExpr := cachedIdent("x")

			result := ce.emitDecimalCoercion(inputExpr, tc.sourceType)

			require.NotNil(t, result)

			if tc.wantPassthrough {
				assert.Equal(t, inputExpr, result)
				return
			}

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok)

			if tc.wantConstructor != "" {
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok)
				assert.Equal(t, pkgMaths, selector.X.(*goast.Ident).Name)
				assert.Equal(t, tc.wantConstructor, selector.Sel.Name)
			}

			if tc.wantMethod != "" {
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok)
				assert.Equal(t, tc.wantMethod, selector.Sel.Name)
			}

			_, hasMathsImport := em.ctx.requiredImports[mathsPackagePath]
			assert.True(t, hasMathsImport, "maths package should be imported")
		})
	}
}

func TestEmitBigIntCoercion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		sourceType      string
		wantConstructor string
		wantMethod      string
		wantPassthrough bool
	}{
		{
			name:            "maths.BigInt passthrough",
			sourceType:      mathsBigIntTypeName,
			wantPassthrough: true,
		},
		{
			name:            "int64 uses NewBigIntFromInt",
			sourceType:      Int64TypeName,
			wantConstructor: mathsNewBigIntFromInt,
		},
		{
			name:            "int uses NewBigIntFromInt with cast",
			sourceType:      IntTypeName,
			wantConstructor: mathsNewBigIntFromInt,
		},
		{
			name:            "string uses NewBigIntFromString",
			sourceType:      StringTypeName,
			wantConstructor: mathsNewBigIntFromString,
		},
		{
			name:       "maths.Decimal uses ToBigInt method",
			sourceType: mathsDecimalTypeName,
			wantMethod: mathsToBigInt,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, em := setupCoercionEmitter()
			inputExpr := cachedIdent("x")

			result := ce.emitBigIntCoercion(inputExpr, tc.sourceType)

			require.NotNil(t, result)

			if tc.wantPassthrough {
				assert.Equal(t, inputExpr, result)
				return
			}

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok)

			if tc.wantConstructor != "" {
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok)
				assert.Equal(t, pkgMaths, selector.X.(*goast.Ident).Name)
				assert.Equal(t, tc.wantConstructor, selector.Sel.Name)
			}

			if tc.wantMethod != "" {
				selector, ok := callExpr.Fun.(*goast.SelectorExpr)
				require.True(t, ok)
				assert.Equal(t, tc.wantMethod, selector.Sel.Name)
			}

			_, hasMathsImport := em.ctx.requiredImports[mathsPackagePath]
			assert.True(t, hasMathsImport, "maths package should be imported")
		})
	}
}

func TestEmitCoercionCall_Dispatch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		functionName string
		sourceType   string
		wantNil      bool
	}{
		{
			name:         "string coercion",
			functionName: StringTypeName,
			sourceType:   IntTypeName,
		},
		{
			name:         "int coercion",
			functionName: IntTypeName,
			sourceType:   Float64TypeName,
		},
		{
			name:         "int64 coercion",
			functionName: Int64TypeName,
			sourceType:   IntTypeName,
		},
		{
			name:         "int32 coercion",
			functionName: Int32TypeName,
			sourceType:   IntTypeName,
		},
		{
			name:         "int16 coercion",
			functionName: Int16TypeName,
			sourceType:   IntTypeName,
		},
		{
			name:         "float coercion",
			functionName: "float",
			sourceType:   IntTypeName,
		},
		{
			name:         "float64 coercion",
			functionName: Float64TypeName,
			sourceType:   IntTypeName,
		},
		{
			name:         "float32 coercion",
			functionName: Float32TypeName,
			sourceType:   IntTypeName,
		},
		{
			name:         "bool coercion",
			functionName: BoolTypeName,
			sourceType:   IntTypeName,
		},
		{
			name:         "decimal coercion",
			functionName: "decimal",
			sourceType:   Int64TypeName,
		},
		{
			name:         "bigint coercion",
			functionName: "bigint",
			sourceType:   Int64TypeName,
		},
		{
			name:         "unknown function returns input",
			functionName: "unknownFunc",
			sourceType:   IntTypeName,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, _ := setupCoercionEmitter()
			inputExpr := cachedIdent("x")
			ann := createMockAnnotation(tc.sourceType, inspector_dto.StringablePrimitive)

			result := ce.emitCoercionCall(tc.functionName, nil, inputExpr, ann)

			require.NotNil(t, result, "emitCoercionCall should never return nil")
		})
	}
}

func TestEmitBoolToIntIIFE(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		targetType string
	}{
		{name: "int", targetType: IntTypeName},
		{name: "int64", targetType: Int64TypeName},
		{name: "int32", targetType: Int32TypeName},
		{name: "int16", targetType: Int16TypeName},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, _ := setupCoercionEmitter()
			inputExpr := cachedIdent("b")

			result := ce.emitBoolToIntIIFE(inputExpr, tc.targetType)

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok, "Expected CallExpr for IIFE")

			funcLit, ok := callExpr.Fun.(*goast.FuncLit)
			require.True(t, ok, "Expected FuncLit")

			require.NotNil(t, funcLit.Type.Results)
			require.Len(t, funcLit.Type.Results.List, 1)
			returnType, ok := funcLit.Type.Results.List[0].Type.(*goast.Ident)
			require.True(t, ok, "expected *goast.Ident")
			assert.Equal(t, tc.targetType, returnType.Name)

			require.GreaterOrEqual(t, len(funcLit.Body.List), 2)

			ifStmt, ok := funcLit.Body.List[0].(*goast.IfStmt)
			require.True(t, ok)
			assert.Equal(t, inputExpr, ifStmt.Cond)

			returnStmt, ok := ifStmt.Body.List[0].(*goast.ReturnStmt)
			require.True(t, ok)
			lit, ok := returnStmt.Results[0].(*goast.BasicLit)
			require.True(t, ok)
			assert.Equal(t, "1", lit.Value)

			defaultReturn, ok := funcLit.Body.List[1].(*goast.ReturnStmt)
			require.True(t, ok)
			defaultLit, ok := defaultReturn.Results[0].(*goast.BasicLit)
			require.True(t, ok)
			assert.Equal(t, "0", defaultLit.Value)
		})
	}
}

func TestEmitBoolToFloatIIFE(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		targetType string
	}{
		{name: "float64", targetType: Float64TypeName},
		{name: "float32", targetType: Float32TypeName},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, _ := setupCoercionEmitter()
			inputExpr := cachedIdent("b")

			result := ce.emitBoolToFloatIIFE(inputExpr, tc.targetType)

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok)

			funcLit, ok := callExpr.Fun.(*goast.FuncLit)
			require.True(t, ok)

			returnType, ok := funcLit.Type.Results.List[0].Type.(*goast.Ident)
			require.True(t, ok, "expected *goast.Ident")
			assert.Equal(t, tc.targetType, returnType.Name)

			ifStmt, ok := funcLit.Body.List[0].(*goast.IfStmt)
			require.True(t, ok, "expected *goast.IfStmt")
			returnStmt, ok := ifStmt.Body.List[0].(*goast.ReturnStmt)
			require.True(t, ok, "expected *goast.ReturnStmt")
			lit, ok := returnStmt.Results[0].(*goast.BasicLit)
			require.True(t, ok, "expected *goast.BasicLit")
			assert.Equal(t, "1.0", lit.Value)

			defaultReturn, ok := funcLit.Body.List[1].(*goast.ReturnStmt)
			require.True(t, ok, "expected *goast.ReturnStmt")
			defaultLit, ok := defaultReturn.Results[0].(*goast.BasicLit)
			require.True(t, ok, "expected *goast.BasicLit")
			assert.Equal(t, "0.0", defaultLit.Value)
		})
	}
}

func TestEmitStringParseIIFE(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		targetType string
		bitSize    int
	}{
		{name: "int", targetType: IntTypeName, bitSize: bitSize64},
		{name: "int64", targetType: Int64TypeName, bitSize: bitSize64},
		{name: "int32", targetType: Int32TypeName, bitSize: bitSize32},
		{name: "int16", targetType: Int16TypeName, bitSize: bitSize16},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, em := setupCoercionEmitter()
			inputExpr := cachedIdent("s")

			result := ce.emitStringParseIIFE(inputExpr, tc.targetType, tc.bitSize)

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok)

			funcLit, ok := callExpr.Fun.(*goast.FuncLit)
			require.True(t, ok)

			returnType, ok := funcLit.Type.Results.List[0].Type.(*goast.Ident)
			require.True(t, ok, "expected *goast.Ident")
			assert.Equal(t, tc.targetType, returnType.Name)

			assignStmt, ok := funcLit.Body.List[0].(*goast.AssignStmt)
			require.True(t, ok)
			assert.Len(t, assignStmt.Lhs, 2)
			lhsIdent0, ok := assignStmt.Lhs[0].(*goast.Ident)
			require.True(t, ok, "expected *goast.Ident")
			assert.Equal(t, varNameV, lhsIdent0.Name)
			lhsIdent1, ok := assignStmt.Lhs[1].(*goast.Ident)
			require.True(t, ok, "expected *goast.Ident")
			assert.Equal(t, BlankIdentifier, lhsIdent1.Name)

			_, hasStrconvImport := em.ctx.requiredImports[pkgStrconv]
			assert.True(t, hasStrconvImport, "strconv should be imported")
		})
	}
}

func TestEmitStringParseToFloatIIFE(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		targetType string
		bitSize    int
		wantCast   bool
	}{
		{
			name:       "float64 no cast needed",
			targetType: Float64TypeName,
			bitSize:    bitSize64,
			wantCast:   false,
		},
		{
			name:       "float32 needs cast",
			targetType: Float32TypeName,
			bitSize:    bitSize32,
			wantCast:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, _ := setupCoercionEmitter()
			inputExpr := cachedIdent("s")

			result := ce.emitStringParseToFloatIIFE(inputExpr, tc.targetType, tc.bitSize)

			callExpr, ok := result.(*goast.CallExpr)
			require.True(t, ok)

			funcLit, ok := callExpr.Fun.(*goast.FuncLit)
			require.True(t, ok)

			returnType, ok := funcLit.Type.Results.List[0].Type.(*goast.Ident)
			require.True(t, ok, "expected *goast.Ident")
			assert.Equal(t, tc.targetType, returnType.Name)

			returnStmt, ok := funcLit.Body.List[1].(*goast.ReturnStmt)
			require.True(t, ok)

			if tc.wantCast {

				castCall, ok := returnStmt.Results[0].(*goast.CallExpr)
				require.True(t, ok)
				castIdent, ok := castCall.Fun.(*goast.Ident)
				require.True(t, ok)
				assert.Equal(t, Float32TypeName, castIdent.Name)
			} else {

				identifier, ok := returnStmt.Results[0].(*goast.Ident)
				require.True(t, ok)
				assert.Equal(t, varNameV, identifier.Name)
			}
		})
	}
}

func TestEmitStringParseToBoolIIFE(t *testing.T) {
	t.Parallel()

	ce, em := setupCoercionEmitter()
	inputExpr := cachedIdent("s")

	result := ce.emitStringParseToBoolIIFE(inputExpr)

	callExpr, ok := result.(*goast.CallExpr)
	require.True(t, ok)

	funcLit, ok := callExpr.Fun.(*goast.FuncLit)
	require.True(t, ok)

	returnType, ok := funcLit.Type.Results.List[0].Type.(*goast.Ident)
	require.True(t, ok, "expected *goast.Ident")
	assert.Equal(t, BoolTypeName, returnType.Name)

	assignStmt, ok := funcLit.Body.List[0].(*goast.AssignStmt)
	require.True(t, ok)

	parseCall, ok := assignStmt.Rhs[0].(*goast.CallExpr)
	require.True(t, ok)

	selector, ok := parseCall.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	selectorIdent, ok := selector.X.(*goast.Ident)
	require.True(t, ok, "expected *goast.Ident")
	assert.Equal(t, pkgStrconv, selectorIdent.Name)
	assert.Equal(t, strconvParseBool, selector.Sel.Name)

	_, hasStrconvImport := em.ctx.requiredImports[pkgStrconv]
	assert.True(t, hasStrconvImport, "strconv should be imported")
}

func TestCastTo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		targetType string
	}{
		{name: "int", targetType: IntTypeName},
		{name: "int64", targetType: Int64TypeName},
		{name: "float64", targetType: Float64TypeName},
		{name: "uint64", targetType: Uint64TypeName},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce, _ := setupCoercionEmitter()
			inputExpr := cachedIdent("x")

			result := ce.castTo(tc.targetType, inputExpr)

			require.NotNil(t, result)
			assert.IsType(t, &goast.CallExpr{}, result)

			funcIdent, ok := result.Fun.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, tc.targetType, funcIdent.Name)

			require.Len(t, result.Args, 1)
			assert.Equal(t, inputExpr, result.Args[0])
		})
	}
}

func TestMethodCall(t *testing.T) {
	t.Parallel()

	ce, _ := setupCoercionEmitter()
	receiver := cachedIdent("obj")
	method := "SomeMethod"

	result := ce.methodCall(receiver, method)

	require.NotNil(t, result)

	selector, ok := result.Fun.(*goast.SelectorExpr)
	require.True(t, ok)

	assert.Equal(t, receiver, selector.X)
	assert.Equal(t, method, selector.Sel.Name)
	assert.Empty(t, result.Args)
}

func TestStrconvCall(t *testing.T) {
	t.Parallel()

	ce, _ := setupCoercionEmitter()
	arg1 := cachedIdent("x")
	arg2 := intLit(10)

	result := ce.strconvCall("FormatInt", arg1, arg2)

	require.NotNil(t, result)

	selector, ok := result.Fun.(*goast.SelectorExpr)
	require.True(t, ok)

	assert.Equal(t, pkgStrconv, selector.X.(*goast.Ident).Name)
	assert.Equal(t, "FormatInt", selector.Sel.Name)
	require.Len(t, result.Args, 2)
	assert.Equal(t, arg1, result.Args[0])
	assert.Equal(t, arg2, result.Args[1])
}

func TestMathsConstructorCall(t *testing.T) {
	t.Parallel()

	ce, _ := setupCoercionEmitter()
	argument := cachedIdent("x")

	result := ce.mathsConstructorCall(mathsNewDecimalFromInt, argument)

	require.NotNil(t, result)

	selector, ok := result.Fun.(*goast.SelectorExpr)
	require.True(t, ok)

	assert.Equal(t, pkgMaths, selector.X.(*goast.Ident).Name)
	assert.Equal(t, mathsNewDecimalFromInt, selector.Sel.Name)
	require.Len(t, result.Args, 1)
	assert.Equal(t, argument, result.Args[0])
}

func TestEmitRuntimeCoercionCall(t *testing.T) {
	t.Parallel()

	ce, em := setupCoercionEmitter()
	argument := cachedIdent("x")
	helperName := "CoerceToString"

	result := ce.emitRuntimeCoercionCall(helperName, argument)

	require.NotNil(t, result)

	callExpr, ok := result.(*goast.CallExpr)
	require.True(t, ok)

	selector, ok := callExpr.Fun.(*goast.SelectorExpr)
	require.True(t, ok)

	assert.Equal(t, runtimePackageName, selector.X.(*goast.Ident).Name)
	assert.Equal(t, helperName, selector.Sel.Name)
	require.Len(t, callExpr.Args, 1)
	assert.Equal(t, argument, callExpr.Args[0])

	_, hasRuntimeImport := em.ctx.requiredImports[coercionRuntimePackagePath]
	assert.True(t, hasRuntimeImport, "runtime package should be imported")
}

func BenchmarkEmitStringCoercion_Passthrough(b *testing.B) {
	ce, _ := setupCoercionEmitter()
	expression := cachedIdent("x")

	b.ResetTimer()
	for b.Loop() {
		_ = ce.emitStringCoercion(expression, StringTypeName)
	}
}

func BenchmarkEmitStringCoercion_Int64(b *testing.B) {
	ce, _ := setupCoercionEmitter()
	expression := cachedIdent("x")

	b.ResetTimer()
	for b.Loop() {
		_ = ce.emitStringCoercion(expression, Int64TypeName)
	}
}

func BenchmarkEmitIntCoercion_FromFloat64(b *testing.B) {
	ce, _ := setupCoercionEmitter()
	expression := cachedIdent("x")

	b.ResetTimer()
	for b.Loop() {
		_ = ce.emitIntCoercion(expression, Float64TypeName)
	}
}

func BenchmarkEmitBoolToIntIIFE(b *testing.B) {
	ce, _ := setupCoercionEmitter()
	expression := cachedIdent("b")

	b.ResetTimer()
	for b.Loop() {
		_ = ce.emitBoolToIntIIFE(expression, IntTypeName)
	}
}
