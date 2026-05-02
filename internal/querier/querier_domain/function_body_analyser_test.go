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

func TestAnalyseFunctionBody_LiteralReturn(t *testing.T) {
	t.Parallel()

	engine := &mockEngine{
		normaliseTypeNameFn: func(name string, _ ...int) querier_dto.SQLType {
			if name == "Int64" {
				return querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"}
			}
			return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: name}
		},
	}
	resolver := newTestTypeResolverWithEngine(engine)

	signature := &querier_dto.FunctionSignature{
		Name: "fortytwo",
		BodyExpression: &querier_dto.LiteralExpression{
			TypeName: "Int64",
		},
	}

	err := AnalyseFunctionBody(signature, resolver)
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryInteger, signature.ReturnType.Category)
	assert.Equal(t, "Int64", signature.ReturnType.EngineName)
}

func TestAnalyseFunctionBody_FunctionCallReturn(t *testing.T) {
	t.Parallel()

	uint64Type := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"}
	builtins := &querier_dto.FunctionCatalogue{
		Functions: map[string][]*querier_dto.FunctionSignature{
			"arraylength": {
				{
					Name:       "arrayLength",
					ReturnType: uint64Type,
					Arguments: []querier_dto.FunctionArgument{
						{Name: "arr", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryArray}},
					},
					NullableBehaviour: querier_dto.FunctionNullableNeverNull,
					DataAccess:        querier_dto.DataAccessReadOnly,
				},
			},
		},
	}
	engine := &mockEngine{
		builtinFunctionsFn: func() *querier_dto.FunctionCatalogue { return builtins },
	}
	resolver := newTestTypeResolverWithEngine(engine)

	signature := &querier_dto.FunctionSignature{
		Name:           "lengthof",
		BodyParameters: []string{"x"},
		BodyExpression: &querier_dto.FunctionCallExpression{
			FunctionName: "arrayLength",
			Arguments: []querier_dto.Expression{
				&querier_dto.ColumnRefExpression{ColumnName: "x"},
			},
		},
	}

	err := AnalyseFunctionBody(signature, resolver)
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryInteger, signature.ReturnType.Category)
	assert.Equal(t, "UInt64", signature.ReturnType.EngineName)
}

func TestAnalyseFunctionBody_ArithmeticWithParameter(t *testing.T) {
	t.Parallel()

	engine := &mockEngine{
		normaliseTypeNameFn: func(name string, _ ...int) querier_dto.SQLType {
			if name == "Int64" {
				return querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"}
			}
			return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: name}
		},
	}
	resolver := newTestTypeResolverWithEngine(engine)

	signature := &querier_dto.FunctionSignature{
		Name:           "twice",
		BodyParameters: []string{"x"},
		BodyExpression: &querier_dto.BinaryOpExpression{
			Operator: "+",
			Left:     &querier_dto.ColumnRefExpression{ColumnName: "x"},
			Right:    &querier_dto.LiteralExpression{TypeName: "Int64"},
		},
	}

	err := AnalyseFunctionBody(signature, resolver)
	require.NoError(t, err)

	assert.Equal(t, querier_dto.TypeCategoryInteger, signature.ReturnType.Category)
	assert.Equal(t, "Int64", signature.ReturnType.EngineName)
}

func TestAnalyseFunctionBody_MultiParameter(t *testing.T) {
	t.Parallel()

	engine := &mockEngine{}
	resolver := newTestTypeResolverWithEngine(engine)

	signature := &querier_dto.FunctionSignature{
		Name:           "addxy",
		BodyParameters: []string{"x", "y"},
		BodyExpression: &querier_dto.BinaryOpExpression{
			Operator: "+",
			Left:     &querier_dto.ColumnRefExpression{ColumnName: "x"},
			Right:    &querier_dto.ColumnRefExpression{ColumnName: "y"},
		},
	}

	err := AnalyseFunctionBody(signature, resolver)
	require.NoError(t, err)

	assert.Equal(t, querier_dto.TypeCategoryUnknown, signature.ReturnType.Category)
}

func TestAnalyseFunctionBody_NilSignature(t *testing.T) {
	t.Parallel()

	resolver := newTestTypeResolver()

	err := AnalyseFunctionBody(nil, resolver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil signature")
}

func TestAnalyseFunctionBody_NilBodyExpression(t *testing.T) {
	t.Parallel()

	resolver := newTestTypeResolver()
	preset := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
	signature := &querier_dto.FunctionSignature{
		Name:       "noop",
		ReturnType: preset,
	}

	err := AnalyseFunctionBody(signature, resolver)
	require.NoError(t, err)

	assert.Equal(t, preset, signature.ReturnType)
}

func TestAnalyseFunctionBody_DoesNotOverwriteSetReturnType(t *testing.T) {
	t.Parallel()

	engine := &mockEngine{
		normaliseTypeNameFn: func(name string, _ ...int) querier_dto.SQLType {
			if name == "Int64" {
				return querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"}
			}
			return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: name}
		},
	}
	resolver := newTestTypeResolverWithEngine(engine)

	preset := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
	signature := &querier_dto.FunctionSignature{
		Name:       "preset",
		ReturnType: preset,
		BodyExpression: &querier_dto.LiteralExpression{
			TypeName: "Int64",
		},
	}

	err := AnalyseFunctionBody(signature, resolver)
	require.NoError(t, err)
	assert.Equal(t, preset, signature.ReturnType, "ReturnType must not be overwritten when set")
}

func TestAnalyseFunctionBody_NilResolver(t *testing.T) {
	t.Parallel()

	signature := &querier_dto.FunctionSignature{
		Name: "literal",
		BodyExpression: &querier_dto.LiteralExpression{
			TypeName: "Int64",
		},
	}

	err := AnalyseFunctionBody(signature, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil resolver")
}
