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
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

func createCoercionTestContext() *AnalysisContext {
	return &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/package",
		CurrentGoPackageName:     "test",
		CurrentGoSourcePath:      "/test/file.go",
		SFCSourcePath:            "/test/file.phtml",
		Logger:                   logger_domain.GetLogger("test"),
	}
}

func TestSuggestCoercionFunction(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		type1          string
		type2          string
		expectedSubstr string
	}{

		{
			name:           "string to int suggests string()",
			type1:          "string",
			type2:          "int",
			expectedSubstr: "string()",
		},
		{
			name:           "int to string suggests string()",
			type1:          "int",
			type2:          "string",
			expectedSubstr: "string()",
		},
		{
			name:           "string to float64 suggests string()",
			type1:          "string",
			type2:          "float64",
			expectedSubstr: "string()",
		},
		{
			name:           "string to maths.Decimal suggests string()",
			type1:          "string",
			type2:          "maths.Decimal",
			expectedSubstr: "string()",
		},
		{
			name:           "string to maths.BigInt suggests string()",
			type1:          "string",
			type2:          "maths.BigInt",
			expectedSubstr: "string()",
		},

		{
			name:           "bool to string suggests string()",
			type1:          "bool",
			type2:          "string",
			expectedSubstr: "string()",
		},
		{
			name:           "string to bool suggests string()",
			type1:          "string",
			type2:          "bool",
			expectedSubstr: "string()",
		},

		{
			name:           "int to float64 suggests explicit conversion",
			type1:          "int",
			type2:          "float64",
			expectedSubstr: "explicit type conversion",
		},
		{
			name:           "float32 to int suggests explicit conversion",
			type1:          "float32",
			type2:          "int",
			expectedSubstr: "explicit type conversion",
		},
		{
			name:           "int64 to int32 suggests explicit conversion",
			type1:          "int64",
			type2:          "int32",
			expectedSubstr: "explicit type conversion",
		},
		{
			name:           "maths.Decimal to maths.BigInt suggests explicit conversion",
			type1:          "maths.Decimal",
			type2:          "maths.BigInt",
			expectedSubstr: "explicit type conversion",
		},
		{
			name:           "byte to rune suggests explicit conversion",
			type1:          "byte",
			type2:          "rune",
			expectedSubstr: "explicit type conversion",
		},

		{
			name:           "int8 to string suggests string()",
			type1:          "int8",
			type2:          "string",
			expectedSubstr: "string()",
		},
		{
			name:           "int16 to string suggests string()",
			type1:          "int16",
			type2:          "string",
			expectedSubstr: "string()",
		},
		{
			name:           "int32 to string suggests string()",
			type1:          "int32",
			type2:          "string",
			expectedSubstr: "string()",
		},
		{
			name:           "int64 to string suggests string()",
			type1:          "int64",
			type2:          "string",
			expectedSubstr: "string()",
		},
		{
			name:           "uint to string suggests string()",
			type1:          "uint",
			type2:          "string",
			expectedSubstr: "string()",
		},
		{
			name:           "uint8 to string suggests string()",
			type1:          "uint8",
			type2:          "string",
			expectedSubstr: "string()",
		},
		{
			name:           "uint16 to string suggests string()",
			type1:          "uint16",
			type2:          "string",
			expectedSubstr: "string()",
		},
		{
			name:           "uint32 to string suggests string()",
			type1:          "uint32",
			type2:          "string",
			expectedSubstr: "string()",
		},
		{
			name:           "uint64 to string suggests string()",
			type1:          "uint64",
			type2:          "string",
			expectedSubstr: "string()",
		},

		{
			name:           "struct to chan no suggestion",
			type1:          "MyStruct",
			type2:          "chan int",
			expectedSubstr: "",
		},
		{
			name:           "map to slice no suggestion",
			type1:          "map[string]int",
			type2:          "[]int",
			expectedSubstr: "",
		},
		{
			name:           "same types no suggestion",
			type1:          "string",
			type2:          "string",
			expectedSubstr: "",
		},
		{
			name:           "bool to int no suggestion",
			type1:          "bool",
			type2:          "int",
			expectedSubstr: "",
		},
		{
			name:           "time.Time to int no suggestion",
			type1:          "time.Time",
			type2:          "int",
			expectedSubstr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := suggestCoercionFunction(tc.type1, tc.type2)
			if tc.expectedSubstr == "" {
				assert.Empty(t, result)
			} else {
				assert.Contains(t, result, tc.expectedSubstr)
			}
		})
	}
}

func TestCoercibleToStringMap(t *testing.T) {
	t.Parallel()

	expectedCoercible := []string{
		"string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "bool", "byte", "rune",
		"maths.Decimal", "maths.BigInt", "time.Time",
		"any", "interface{}",
	}

	for _, typeName := range expectedCoercible {
		t.Run(typeName+" is coercible", func(t *testing.T) {
			t.Parallel()
			assert.True(t, coercibleToString[typeName], "%s should be coercible to string", typeName)
		})
	}

	notCoercible := []string{
		"MyStruct", "chan int", "func()", "map[string]int", "[]byte",
	}

	for _, typeName := range notCoercible {
		t.Run(typeName+" is not coercible", func(t *testing.T) {
			t.Parallel()
			assert.False(t, coercibleToString[typeName], "%s should NOT be coercible to string", typeName)
		})
	}
}

func TestCoercibleToIntMap(t *testing.T) {
	t.Parallel()

	expectedCoercible := []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "bool", "byte", "rune",
		"string", "maths.Decimal", "maths.BigInt",
		"any", "interface{}",
	}

	for _, typeName := range expectedCoercible {
		t.Run(typeName+" is coercible", func(t *testing.T) {
			t.Parallel()
			assert.True(t, coercibleToInt[typeName], "%s should be coercible to int", typeName)
		})
	}

	assert.False(t, coercibleToInt["time.Time"], "time.Time should NOT be coercible to int")
}

func TestCoercibleToFloatMap(t *testing.T) {
	t.Parallel()

	expectedCoercible := []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "bool", "byte", "rune",
		"string", "maths.Decimal", "maths.BigInt",
		"any", "interface{}",
	}

	for _, typeName := range expectedCoercible {
		t.Run(typeName+" is coercible", func(t *testing.T) {
			t.Parallel()
			assert.True(t, coercibleToFloat[typeName], "%s should be coercible to float", typeName)
		})
	}

	assert.False(t, coercibleToFloat["time.Time"], "time.Time should NOT be coercible to float")
}

func TestCoercibleToBoolMap(t *testing.T) {
	t.Parallel()

	expectedCoercible := []string{
		"string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "bool", "byte", "rune",
		"maths.Decimal", "maths.BigInt", "time.Time",
		"any", "interface{}",
	}

	for _, typeName := range expectedCoercible {
		t.Run(typeName+" is coercible", func(t *testing.T) {
			t.Parallel()
			assert.True(t, coercibleToBool[typeName], "%s should be coercible to bool", typeName)
		})
	}
}

func TestCoercibleToDecimalMap(t *testing.T) {
	t.Parallel()

	expectedCoercible := []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "byte", "rune",
		"string", "maths.Decimal", "maths.BigInt",
		"any", "interface{}",
	}

	for _, typeName := range expectedCoercible {
		t.Run(typeName+" is coercible", func(t *testing.T) {
			t.Parallel()
			assert.True(t, coercibleToDecimal[typeName], "%s should be coercible to decimal", typeName)
		})
	}

	assert.False(t, coercibleToDecimal["bool"], "bool should NOT be coercible to decimal")
	assert.False(t, coercibleToDecimal["time.Time"], "time.Time should NOT be coercible to decimal")
}

func TestCoercibleToBigIntMap(t *testing.T) {
	t.Parallel()

	expectedCoercible := []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"byte", "rune", "string",
		"maths.Decimal", "maths.BigInt",
		"any", "interface{}",
	}

	for _, typeName := range expectedCoercible {
		t.Run(typeName+" is coercible", func(t *testing.T) {
			t.Parallel()
			assert.True(t, coercibleToBigInt[typeName], "%s should be coercible to bigint", typeName)
		})
	}

	notCoercible := []string{"float32", "float64", "bool", "time.Time"}
	for _, typeName := range notCoercible {
		t.Run(typeName+" is not coercible", func(t *testing.T) {
			t.Parallel()
			assert.False(t, coercibleToBigInt[typeName], "%s should NOT be coercible to bigint", typeName)
		})
	}
}

func TestGetStringReturnType(t *testing.T) {
	t.Parallel()

	result := getStringReturnType(nil, nil, nil, nil)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeExpression)

	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok, "TypeExpr should be an Ident")
	assert.Equal(t, "string", identifier.Name)
}

func TestGetIntReturnType(t *testing.T) {
	t.Parallel()

	result := getIntReturnType(nil, nil, nil, nil)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeExpression)

	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok, "TypeExpr should be an Ident")
	assert.Equal(t, "int", identifier.Name)
}

func TestGetInt64ReturnType(t *testing.T) {
	t.Parallel()

	result := getInt64ReturnType(nil, nil, nil, nil)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeExpression)

	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok, "TypeExpr should be an Ident")
	assert.Equal(t, "int64", identifier.Name)
}

func TestGetInt32ReturnType(t *testing.T) {
	t.Parallel()

	result := getInt32ReturnType(nil, nil, nil, nil)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeExpression)

	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok, "TypeExpr should be an Ident")
	assert.Equal(t, "int32", identifier.Name)
}

func TestGetInt16ReturnType(t *testing.T) {
	t.Parallel()

	result := getInt16ReturnType(nil, nil, nil, nil)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeExpression)

	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok, "TypeExpr should be an Ident")
	assert.Equal(t, "int16", identifier.Name)
}

func TestGetFloatReturnType(t *testing.T) {
	t.Parallel()

	result := getFloatReturnType(nil, nil, nil, nil)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeExpression)

	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok, "TypeExpr should be an Ident")
	assert.Equal(t, "float64", identifier.Name)
}

func TestGetFloat64ReturnType(t *testing.T) {
	t.Parallel()

	result := getFloat64ReturnType(nil, nil, nil, nil)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeExpression)

	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok, "TypeExpr should be an Ident")
	assert.Equal(t, "float64", identifier.Name)
}

func TestGetFloat32ReturnType(t *testing.T) {
	t.Parallel()

	result := getFloat32ReturnType(nil, nil, nil, nil)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeExpression)

	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok, "TypeExpr should be an Ident")
	assert.Equal(t, "float32", identifier.Name)
}

func TestGetBoolReturnType(t *testing.T) {
	t.Parallel()

	result := getBoolReturnType(nil, nil, nil, nil)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeExpression)

	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok, "TypeExpr should be an Ident")
	assert.Equal(t, "bool", identifier.Name)
}

func TestGetDecimalReturnType(t *testing.T) {
	t.Parallel()

	result := getDecimalReturnType(nil, nil, nil, nil)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeExpression)

	selExpr, ok := result.TypeExpression.(*goast.SelectorExpr)
	require.True(t, ok, "TypeExpr should be a SelectorExpr")

	xIdent, ok := selExpr.X.(*goast.Ident)
	require.True(t, ok, "X should be an Ident")
	assert.Equal(t, "maths", xIdent.Name)
	assert.Equal(t, "Decimal", selExpr.Sel.Name)

	assert.Equal(t, "maths", result.PackageAlias)
	assert.Equal(t, "piko.sh/piko/pkg/maths", result.CanonicalPackagePath)
}

func TestGetBigIntReturnType(t *testing.T) {
	t.Parallel()

	result := getBigIntReturnType(nil, nil, nil, nil)
	require.NotNil(t, result)
	require.NotNil(t, result.TypeExpression)

	selExpr, ok := result.TypeExpression.(*goast.SelectorExpr)
	require.True(t, ok, "TypeExpr should be a SelectorExpr")

	xIdent, ok := selExpr.X.(*goast.Ident)
	require.True(t, ok, "X should be an Ident")
	assert.Equal(t, "maths", xIdent.Name)
	assert.Equal(t, "BigInt", selExpr.Sel.Name)

	assert.Equal(t, "maths", result.PackageAlias)
	assert.Equal(t, "piko.sh/piko/pkg/maths", result.CanonicalPackagePath)
}

func TestReturnTypeFunctionSignatures(t *testing.T) {
	t.Parallel()

	returnTypeFuncs := []struct {
		function func(*TypeResolver, *AnalysisContext, *ast_domain.CallExpression, []*ast_domain.GoGeneratorAnnotation) *ast_domain.ResolvedTypeInfo
		name     string
	}{
		{name: "getStringReturnType", function: getStringReturnType},
		{name: "getIntReturnType", function: getIntReturnType},
		{name: "getInt64ReturnType", function: getInt64ReturnType},
		{name: "getInt32ReturnType", function: getInt32ReturnType},
		{name: "getInt16ReturnType", function: getInt16ReturnType},
		{name: "getFloatReturnType", function: getFloatReturnType},
		{name: "getFloat64ReturnType", function: getFloat64ReturnType},
		{name: "getFloat32ReturnType", function: getFloat32ReturnType},
		{name: "getBoolReturnType", function: getBoolReturnType},
		{name: "getDecimalReturnType", function: getDecimalReturnType},
		{name: "getBigIntReturnType", function: getBigIntReturnType},
	}

	for _, tc := range returnTypeFuncs {
		t.Run(tc.name+" returns non-nil with nil arguments", func(t *testing.T) {
			t.Parallel()

			result := tc.function(nil, nil, nil, nil)
			assert.NotNil(t, result, "%s should return non-nil result", tc.name)
			assert.NotNil(t, result.TypeExpression, "%s should return non-nil TypeExpr", tc.name)
		})
	}
}

func TestCoercibleMapsConsistency(t *testing.T) {
	t.Parallel()

	coercibleMaps := []struct {
		m    map[string]bool
		name string
	}{
		{name: "coercibleToString", m: coercibleToString},
		{name: "coercibleToInt", m: coercibleToInt},
		{name: "coercibleToFloat", m: coercibleToFloat},
		{name: "coercibleToBool", m: coercibleToBool},
		{name: "coercibleToDecimal", m: coercibleToDecimal},
		{name: "coercibleToBigInt", m: coercibleToBigInt},
	}

	for _, tc := range coercibleMaps {
		t.Run(tc.name+" includes any", func(t *testing.T) {
			t.Parallel()
			assert.True(t, tc.m["any"], "%s should include 'any'", tc.name)
		})
		t.Run(tc.name+" includes interface{}", func(t *testing.T) {
			t.Parallel()
			assert.True(t, tc.m["interface{}"], "%s should include 'interface{}'", tc.name)
		})
	}
}

func TestCoercibleMapsIncludeAllIntegerTypes(t *testing.T) {
	t.Parallel()

	integerTypes := []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"byte", "rune",
	}

	mapsToCheck := []struct {
		m    map[string]bool
		name string
	}{
		{name: "coercibleToString", m: coercibleToString},
		{name: "coercibleToInt", m: coercibleToInt},
		{name: "coercibleToFloat", m: coercibleToFloat},
		{name: "coercibleToBool", m: coercibleToBool},
	}

	for _, tc := range mapsToCheck {
		for _, intType := range integerTypes {
			t.Run(tc.name+" includes "+intType, func(t *testing.T) {
				t.Parallel()
				assert.True(t, tc.m[intType], "%s should include %s", tc.name, intType)
			})
		}
	}
}

func TestValidateCoercionArg_WrongArgCount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		argCount int
	}{
		{name: "zero arguments", argCount: 0},
		{name: "two arguments", argCount: 2},
		{name: "three arguments", argCount: 3},
		{name: "five arguments", argCount: 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createCoercionTestContext()
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "string"},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}

			argAnns := make([]*ast_domain.GoGeneratorAnnotation, tc.argCount)
			for i := range argAnns {
				argAnns[i] = &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("string"),
					},
				}
			}

			validateCoercionArg(ctx, callExpr, argAnns, baseLocation, "string", "string", coercibleToString)

			require.Len(t, *ctx.Diagnostics, 1, "should produce one diagnostic for wrong argument count")
			assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
			assert.Contains(t, (*ctx.Diagnostics)[0].Message, "expects exactly one argument")
		})
	}
}

func TestValidateCoercionArg_ValidTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		targetType     string
		coercibleTypes map[string]bool
		inputTypes     []string
	}{
		{
			name:           "valid string coercions",
			targetType:     "string",
			coercibleTypes: coercibleToString,
			inputTypes:     []string{"string", "int", "float64", "bool", "maths.Decimal", "any"},
		},
		{
			name:           "valid int coercions",
			targetType:     "int",
			coercibleTypes: coercibleToInt,
			inputTypes:     []string{"int", "int64", "uint", "float32", "string", "bool"},
		},
		{
			name:           "valid float coercions",
			targetType:     "float64",
			coercibleTypes: coercibleToFloat,
			inputTypes:     []string{"float32", "float64", "int", "string", "maths.Decimal"},
		},
		{
			name:           "valid bool coercions",
			targetType:     "bool",
			coercibleTypes: coercibleToBool,
			inputTypes:     []string{"bool", "string", "int", "float64", "time.Time"},
		},
		{
			name:           "valid decimal coercions",
			targetType:     "maths.Decimal",
			coercibleTypes: coercibleToDecimal,
			inputTypes:     []string{"int", "float64", "string", "maths.BigInt"},
		},
		{
			name:           "valid bigint coercions",
			targetType:     "maths.BigInt",
			coercibleTypes: coercibleToBigInt,
			inputTypes:     []string{"int", "int64", "string", "maths.Decimal"},
		},
	}

	for _, tc := range testCases {
		for _, inputType := range tc.inputTypes {
			t.Run(tc.name+"/"+inputType, func(t *testing.T) {
				t.Parallel()

				ctx := createCoercionTestContext()
				callExpr := &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: tc.targetType},
					Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "value"}},
				}
				baseLocation := ast_domain.Location{Line: 1, Column: 1}

				argAnns := []*ast_domain.GoGeneratorAnnotation{
					{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: goast.NewIdent(inputType),
						},
					},
				}

				validateCoercionArg(ctx, callExpr, argAnns, baseLocation, tc.targetType, tc.targetType, tc.coercibleTypes)

				assert.Empty(t, *ctx.Diagnostics, "should produce no diagnostics for valid type %s", inputType)
			})
		}
	}
}

func TestValidateCoercionArg_InvalidTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		functionName   string
		targetType     string
		coercibleTypes map[string]bool
		invalidTypes   []string
	}{
		{
			name:           "invalid string coercions",
			functionName:   "string",
			targetType:     "string",
			coercibleTypes: coercibleToString,
			invalidTypes:   []string{"MyStruct", "chan int", "func()", "map[string]int"},
		},
		{
			name:           "invalid int coercions",
			functionName:   "int",
			targetType:     "int",
			coercibleTypes: coercibleToInt,
			invalidTypes:   []string{"time.Time", "MyStruct", "[]byte"},
		},
		{
			name:           "invalid decimal coercions",
			functionName:   "decimal",
			targetType:     "maths.Decimal",
			coercibleTypes: coercibleToDecimal,
			invalidTypes:   []string{"bool", "time.Time", "MyStruct"},
		},
		{
			name:           "invalid bigint coercions",
			functionName:   "bigint",
			targetType:     "maths.BigInt",
			coercibleTypes: coercibleToBigInt,
			invalidTypes:   []string{"float32", "float64", "bool", "time.Time"},
		},
	}

	for _, tc := range testCases {
		for _, invalidType := range tc.invalidTypes {
			t.Run(tc.name+"/"+invalidType, func(t *testing.T) {
				t.Parallel()

				ctx := createCoercionTestContext()
				argument := &ast_domain.Identifier{Name: "value"}
				callExpr := &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: tc.functionName},
					Args:   []ast_domain.Expression{argument},
				}
				baseLocation := ast_domain.Location{Line: 10, Column: 5}

				argAnns := []*ast_domain.GoGeneratorAnnotation{
					{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: goast.NewIdent(invalidType),
						},
					},
				}

				validateCoercionArg(ctx, callExpr, argAnns, baseLocation, tc.functionName, tc.targetType, tc.coercibleTypes)

				require.Len(t, *ctx.Diagnostics, 1, "should produce one diagnostic for invalid type %s", invalidType)
				diagnostic := (*ctx.Diagnostics)[0]
				assert.Equal(t, ast_domain.Error, diagnostic.Severity)
				assert.Contains(t, diagnostic.Message, "Cannot coerce")
				assert.Contains(t, diagnostic.Message, invalidType)
				assert.Contains(t, diagnostic.Message, tc.targetType)
			})
		}
	}
}

func TestValidateCoercionArg_NilAnnotations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		argAnns []*ast_domain.GoGeneratorAnnotation
	}{
		{
			name:    "nil annotation in slice",
			argAnns: []*ast_domain.GoGeneratorAnnotation{nil},
		},
		{
			name: "annotation with nil ResolvedType",
			argAnns: []*ast_domain.GoGeneratorAnnotation{
				{ResolvedType: nil},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createCoercionTestContext()
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "string"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "value"}},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}

			validateCoercionArg(ctx, callExpr, tc.argAnns, baseLocation, "string", "string", coercibleToString)

			assert.Empty(t, *ctx.Diagnostics, "should produce no diagnostics when annotations are nil")
		})
	}
}

func TestValidateStringCoercionArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		inputType     string
		errorContains string
		expectError   bool
	}{
		{name: "valid int to string", inputType: "int", expectError: false},
		{name: "valid float64 to string", inputType: "float64", expectError: false},
		{name: "valid bool to string", inputType: "bool", expectError: false},
		{name: "valid time.Time to string", inputType: "time.Time", expectError: false},
		{name: "valid any to string", inputType: "any", expectError: false},
		{name: "invalid struct to string", inputType: "MyStruct", expectError: true, errorContains: "Cannot coerce"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createCoercionTestContext()
			resolver := &TypeResolver{}
			argument := &ast_domain.Identifier{Name: "value"}
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "string"},
				Args:   []ast_domain.Expression{argument},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}
			argAnns := []*ast_domain.GoGeneratorAnnotation{
				{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent(tc.inputType)}},
			}

			resolver.validateStringCoercionArgs(ctx, callExpr, argAnns, baseLocation)

			if tc.expectError {
				require.Len(t, *ctx.Diagnostics, 1)
				assert.Contains(t, (*ctx.Diagnostics)[0].Message, tc.errorContains)
			} else {
				assert.Empty(t, *ctx.Diagnostics)
			}
		})
	}
}

func TestValidateIntCoercionArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		inputType     string
		errorContains string
		expectError   bool
	}{
		{name: "valid string to int", inputType: "string", expectError: false},
		{name: "valid float64 to int", inputType: "float64", expectError: false},
		{name: "valid bool to int", inputType: "bool", expectError: false},
		{name: "invalid time.Time to int", inputType: "time.Time", expectError: true, errorContains: "Cannot coerce"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createCoercionTestContext()
			resolver := &TypeResolver{}
			argument := &ast_domain.Identifier{Name: "value"}
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "int"},
				Args:   []ast_domain.Expression{argument},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}
			argAnns := []*ast_domain.GoGeneratorAnnotation{
				{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent(tc.inputType)}},
			}

			resolver.validateIntCoercionArgs(ctx, callExpr, argAnns, baseLocation)

			if tc.expectError {
				require.Len(t, *ctx.Diagnostics, 1)
				assert.Contains(t, (*ctx.Diagnostics)[0].Message, tc.errorContains)
			} else {
				assert.Empty(t, *ctx.Diagnostics)
			}
		})
	}
}

func TestValidateFloatCoercionArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		inputType     string
		errorContains string
		expectError   bool
	}{
		{name: "valid int to float", inputType: "int", expectError: false},
		{name: "valid string to float", inputType: "string", expectError: false},
		{name: "valid maths.Decimal to float", inputType: "maths.Decimal", expectError: false},
		{name: "invalid time.Time to float", inputType: "time.Time", expectError: true, errorContains: "Cannot coerce"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createCoercionTestContext()
			resolver := &TypeResolver{}
			argument := &ast_domain.Identifier{Name: "value"}
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "float"},
				Args:   []ast_domain.Expression{argument},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}
			argAnns := []*ast_domain.GoGeneratorAnnotation{
				{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent(tc.inputType)}},
			}

			resolver.validateFloatCoercionArgs(ctx, callExpr, argAnns, baseLocation)

			if tc.expectError {
				require.Len(t, *ctx.Diagnostics, 1)
				assert.Contains(t, (*ctx.Diagnostics)[0].Message, tc.errorContains)
			} else {
				assert.Empty(t, *ctx.Diagnostics)
			}
		})
	}
}

func TestValidateBoolCoercionArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		inputType     string
		errorContains string
		expectError   bool
	}{
		{name: "valid string to bool", inputType: "string", expectError: false},
		{name: "valid int to bool", inputType: "int", expectError: false},
		{name: "valid time.Time to bool", inputType: "time.Time", expectError: false},
		{name: "invalid struct to bool", inputType: "MyStruct", expectError: true, errorContains: "Cannot coerce"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createCoercionTestContext()
			resolver := &TypeResolver{}
			argument := &ast_domain.Identifier{Name: "value"}
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "bool"},
				Args:   []ast_domain.Expression{argument},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}
			argAnns := []*ast_domain.GoGeneratorAnnotation{
				{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent(tc.inputType)}},
			}

			resolver.validateBoolCoercionArgs(ctx, callExpr, argAnns, baseLocation)

			if tc.expectError {
				require.Len(t, *ctx.Diagnostics, 1)
				assert.Contains(t, (*ctx.Diagnostics)[0].Message, tc.errorContains)
			} else {
				assert.Empty(t, *ctx.Diagnostics)
			}
		})
	}
}

func TestValidateDecimalCoercionArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		inputType     string
		errorContains string
		expectError   bool
	}{
		{name: "valid int to decimal", inputType: "int", expectError: false},
		{name: "valid float64 to decimal", inputType: "float64", expectError: false},
		{name: "valid string to decimal", inputType: "string", expectError: false},
		{name: "invalid bool to decimal", inputType: "bool", expectError: true, errorContains: "Cannot coerce"},
		{name: "invalid time.Time to decimal", inputType: "time.Time", expectError: true, errorContains: "Cannot coerce"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createCoercionTestContext()
			resolver := &TypeResolver{}
			argument := &ast_domain.Identifier{Name: "value"}
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "decimal"},
				Args:   []ast_domain.Expression{argument},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}
			argAnns := []*ast_domain.GoGeneratorAnnotation{
				{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent(tc.inputType)}},
			}

			resolver.validateDecimalCoercionArgs(ctx, callExpr, argAnns, baseLocation)

			if tc.expectError {
				require.Len(t, *ctx.Diagnostics, 1)
				assert.Contains(t, (*ctx.Diagnostics)[0].Message, tc.errorContains)
			} else {
				assert.Empty(t, *ctx.Diagnostics)
			}
		})
	}
}

func TestValidateBigIntCoercionArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		inputType     string
		errorContains string
		expectError   bool
	}{
		{name: "valid int to bigint", inputType: "int", expectError: false},
		{name: "valid int64 to bigint", inputType: "int64", expectError: false},
		{name: "valid string to bigint", inputType: "string", expectError: false},
		{name: "valid maths.Decimal to bigint", inputType: "maths.Decimal", expectError: false},
		{name: "invalid float32 to bigint", inputType: "float32", expectError: true, errorContains: "Cannot coerce"},
		{name: "invalid float64 to bigint", inputType: "float64", expectError: true, errorContains: "Cannot coerce"},
		{name: "invalid bool to bigint", inputType: "bool", expectError: true, errorContains: "Cannot coerce"},
		{name: "invalid time.Time to bigint", inputType: "time.Time", expectError: true, errorContains: "Cannot coerce"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createCoercionTestContext()
			resolver := &TypeResolver{}
			argument := &ast_domain.Identifier{Name: "value"}
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "bigint"},
				Args:   []ast_domain.Expression{argument},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}
			argAnns := []*ast_domain.GoGeneratorAnnotation{
				{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent(tc.inputType)}},
			}

			resolver.validateBigIntCoercionArgs(ctx, callExpr, argAnns, baseLocation)

			if tc.expectError {
				require.Len(t, *ctx.Diagnostics, 1)
				assert.Contains(t, (*ctx.Diagnostics)[0].Message, tc.errorContains)
			} else {
				assert.Empty(t, *ctx.Diagnostics)
			}
		})
	}
}

func TestValidateInt64CoercionArgs(t *testing.T) {
	t.Parallel()

	ctx := createCoercionTestContext()
	resolver := &TypeResolver{}
	argument := &ast_domain.Identifier{Name: "value"}
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "int64"},
		Args:   []ast_domain.Expression{argument},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")}},
	}

	resolver.validateInt64CoercionArgs(ctx, callExpr, argAnns, baseLocation)

	assert.Empty(t, *ctx.Diagnostics, "string should be coercible to int64")
}

func TestValidateInt32CoercionArgs(t *testing.T) {
	t.Parallel()

	ctx := createCoercionTestContext()
	resolver := &TypeResolver{}
	argument := &ast_domain.Identifier{Name: "value"}
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "int32"},
		Args:   []ast_domain.Expression{argument},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")}},
	}

	resolver.validateInt32CoercionArgs(ctx, callExpr, argAnns, baseLocation)

	assert.Empty(t, *ctx.Diagnostics, "float64 should be coercible to int32")
}

func TestValidateInt16CoercionArgs(t *testing.T) {
	t.Parallel()

	ctx := createCoercionTestContext()
	resolver := &TypeResolver{}
	argument := &ast_domain.Identifier{Name: "value"}
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "int16"},
		Args:   []ast_domain.Expression{argument},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
	}

	resolver.validateInt16CoercionArgs(ctx, callExpr, argAnns, baseLocation)

	assert.Empty(t, *ctx.Diagnostics, "int should be coercible to int16")
}

func TestValidateFloat64CoercionArgs(t *testing.T) {
	t.Parallel()

	ctx := createCoercionTestContext()
	resolver := &TypeResolver{}
	argument := &ast_domain.Identifier{Name: "value"}
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "float64"},
		Args:   []ast_domain.Expression{argument},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
	}

	resolver.validateFloat64CoercionArgs(ctx, callExpr, argAnns, baseLocation)

	assert.Empty(t, *ctx.Diagnostics, "int should be coercible to float64")
}

func TestValidateFloat32CoercionArgs(t *testing.T) {
	t.Parallel()

	ctx := createCoercionTestContext()
	resolver := &TypeResolver{}
	argument := &ast_domain.Identifier{Name: "value"}
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "float32"},
		Args:   []ast_domain.Expression{argument},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")}},
	}

	resolver.validateFloat32CoercionArgs(ctx, callExpr, argAnns, baseLocation)

	assert.Empty(t, *ctx.Diagnostics, "string should be coercible to float32")
}

func TestAllCoercionValidatorsSameSignature(t *testing.T) {
	t.Parallel()

	type validatorFunc func(*TypeResolver, *AnalysisContext, *ast_domain.CallExpression, []*ast_domain.GoGeneratorAnnotation, ast_domain.Location)

	validators := []struct {
		validatorFunction validatorFunc
		name              string
	}{
		{name: "validateStringCoercionArgs", validatorFunction: (*TypeResolver).validateStringCoercionArgs},
		{name: "validateIntCoercionArgs", validatorFunction: (*TypeResolver).validateIntCoercionArgs},
		{name: "validateInt64CoercionArgs", validatorFunction: (*TypeResolver).validateInt64CoercionArgs},
		{name: "validateInt32CoercionArgs", validatorFunction: (*TypeResolver).validateInt32CoercionArgs},
		{name: "validateInt16CoercionArgs", validatorFunction: (*TypeResolver).validateInt16CoercionArgs},
		{name: "validateFloatCoercionArgs", validatorFunction: (*TypeResolver).validateFloatCoercionArgs},
		{name: "validateFloat64CoercionArgs", validatorFunction: (*TypeResolver).validateFloat64CoercionArgs},
		{name: "validateFloat32CoercionArgs", validatorFunction: (*TypeResolver).validateFloat32CoercionArgs},
		{name: "validateBoolCoercionArgs", validatorFunction: (*TypeResolver).validateBoolCoercionArgs},
		{name: "validateDecimalCoercionArgs", validatorFunction: (*TypeResolver).validateDecimalCoercionArgs},
		{name: "validateBigIntCoercionArgs", validatorFunction: (*TypeResolver).validateBigIntCoercionArgs},
	}

	for _, v := range validators {
		t.Run(v.name+" callable with valid arguments", func(t *testing.T) {
			t.Parallel()

			ctx := createCoercionTestContext()
			resolver := &TypeResolver{}
			argument := &ast_domain.Identifier{Name: "value"}
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "coerce"},
				Args:   []ast_domain.Expression{argument},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}

			argAnns := []*ast_domain.GoGeneratorAnnotation{
				{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("any")}},
			}

			v.validatorFunction(resolver, ctx, callExpr, argAnns, baseLocation)

			assert.Empty(t, *ctx.Diagnostics, "%s should accept 'any' type", v.name)
		})
	}
}
