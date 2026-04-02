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

package db_engine_postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestResolveFunctionCall(t *testing.T) {
	t.Parallel()

	resolver := NewPostgresFunctionResolver()

	int2Type := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int2"}
	int4Type := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}
	int8Type := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"}
	float4Type := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float4"}
	textType := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}
	textArrayType := querier_dto.SQLType{
		Category:    querier_dto.TypeCategoryArray,
		EngineName:  "text[]",
		ElementType: &textType,
	}

	type testCase struct {
		description         string
		functionName        string
		argumentTypes       []querier_dto.SQLType
		expectNil           bool
		expectedEngineName  string
		expectedCategory    querier_dto.SQLTypeCategory
		expectedIsAggregate bool
		expectedReturnsSet  bool
		expectedElementType *querier_dto.SQLType
	}

	cases := []testCase{
		{
			description:         "array_agg with int4 returns int4 array and is aggregate",
			functionName:        "array_agg",
			argumentTypes:       []querier_dto.SQLType{int4Type},
			expectedEngineName:  "int4[]",
			expectedCategory:    querier_dto.TypeCategoryArray,
			expectedIsAggregate: true,
			expectedElementType: &int4Type,
		},
		{
			description:        "unnest with text array returns text element and returns set",
			functionName:       "unnest",
			argumentTypes:      []querier_dto.SQLType{textArrayType},
			expectedEngineName: "text",
			expectedCategory:   querier_dto.TypeCategoryText,
			expectedReturnsSet: true,
		},
		{
			description:         "sum with int2 promotes to int8",
			functionName:        "sum",
			argumentTypes:       []querier_dto.SQLType{int2Type},
			expectedEngineName:  "int8",
			expectedCategory:    querier_dto.TypeCategoryInteger,
			expectedIsAggregate: true,
		},
		{
			description:         "sum with int4 promotes to int8 (small integer promotion)",
			functionName:        "sum",
			argumentTypes:       []querier_dto.SQLType{int4Type},
			expectedEngineName:  "int8",
			expectedCategory:    querier_dto.TypeCategoryInteger,
			expectedIsAggregate: true,
		},
		{
			description:         "sum with int8 promotes to numeric (large integer)",
			functionName:        "sum",
			argumentTypes:       []querier_dto.SQLType{int8Type},
			expectedEngineName:  "numeric",
			expectedCategory:    querier_dto.TypeCategoryDecimal,
			expectedIsAggregate: true,
		},
		{
			description:         "sum with float4 promotes to float8",
			functionName:        "sum",
			argumentTypes:       []querier_dto.SQLType{float4Type},
			expectedEngineName:  "float8",
			expectedCategory:    querier_dto.TypeCategoryFloat,
			expectedIsAggregate: true,
		},
		{
			description:         "avg with int4 returns numeric",
			functionName:        "avg",
			argumentTypes:       []querier_dto.SQLType{int4Type},
			expectedEngineName:  "numeric",
			expectedCategory:    querier_dto.TypeCategoryDecimal,
			expectedIsAggregate: true,
		},
		{
			description:         "avg with float4 returns float8",
			functionName:        "avg",
			argumentTypes:       []querier_dto.SQLType{float4Type},
			expectedEngineName:  "float8",
			expectedCategory:    querier_dto.TypeCategoryFloat,
			expectedIsAggregate: true,
		},
		{
			description:         "min with int4 returns int4 (identity aggregate)",
			functionName:        "min",
			argumentTypes:       []querier_dto.SQLType{int4Type},
			expectedEngineName:  "int4",
			expectedCategory:    querier_dto.TypeCategoryInteger,
			expectedIsAggregate: true,
		},
		{
			description:         "max with text returns text (identity aggregate)",
			functionName:        "max",
			argumentTypes:       []querier_dto.SQLType{textType},
			expectedEngineName:  "text",
			expectedCategory:    querier_dto.TypeCategoryText,
			expectedIsAggregate: true,
		},
		{
			description:   "unknown function returns nil resolution",
			functionName:  "totally_unknown_function",
			argumentTypes: []querier_dto.SQLType{int4Type},
			expectNil:     true,
		},
		{
			description:        "pg_typeof returns text regardless of argument",
			functionName:       "pg_typeof",
			argumentTypes:      []querier_dto.SQLType{int4Type},
			expectedEngineName: "text",
			expectedCategory:   querier_dto.TypeCategoryText,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			resolution, err := resolver.ResolveFunctionCall(nil, tc.functionName, "", tc.argumentTypes)
			require.NoError(t, err)

			if tc.expectNil {
				assert.Nil(t, resolution, "expected nil resolution for unknown function")
				return
			}

			require.NotNil(t, resolution, "expected non-nil resolution")
			assert.Equal(t, tc.expectedEngineName, resolution.ReturnType.EngineName, "return type engine name")
			assert.Equal(t, tc.expectedCategory, resolution.ReturnType.Category, "return type category")
			assert.Equal(t, tc.expectedIsAggregate, resolution.IsAggregate, "is aggregate")
			assert.Equal(t, tc.expectedReturnsSet, resolution.ReturnsSet, "returns set")

			if tc.expectedElementType != nil {
				require.NotNil(t, resolution.ReturnType.ElementType, "expected element type")
				assert.Equal(t, tc.expectedElementType.EngineName, resolution.ReturnType.ElementType.EngineName)
				assert.Equal(t, tc.expectedElementType.Category, resolution.ReturnType.ElementType.Category)
			}
		})
	}
}
