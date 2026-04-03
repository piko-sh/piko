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

package db_engine_sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestResolveFunctionCall(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()

	integerType := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "integer"}
	realType := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "real"}
	textType := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}
	blobType := querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "blob"}
	unknownType := querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""}

	type testCase struct {
		description         string
		functionName        string
		argumentTypes       []querier_dto.SQLType
		expectNil           bool
		expectedEngineName  string
		expectedCategory    querier_dto.SQLTypeCategory
		expectedIsAggregate bool
	}

	cases := []testCase{
		{
			description:         "min with integer returns integer",
			functionName:        "min",
			argumentTypes:       []querier_dto.SQLType{integerType},
			expectedEngineName:  "integer",
			expectedCategory:    querier_dto.TypeCategoryInteger,
			expectedIsAggregate: true,
		},
		{
			description:         "min with text returns text",
			functionName:        "min",
			argumentTypes:       []querier_dto.SQLType{textType},
			expectedEngineName:  "text",
			expectedCategory:    querier_dto.TypeCategoryText,
			expectedIsAggregate: true,
		},
		{
			description:         "min with real returns real",
			functionName:        "min",
			argumentTypes:       []querier_dto.SQLType{realType},
			expectedEngineName:  "real",
			expectedCategory:    querier_dto.TypeCategoryFloat,
			expectedIsAggregate: true,
		},
		{
			description:         "max with integer returns integer",
			functionName:        "max",
			argumentTypes:       []querier_dto.SQLType{integerType},
			expectedEngineName:  "integer",
			expectedCategory:    querier_dto.TypeCategoryInteger,
			expectedIsAggregate: true,
		},
		{
			description:         "max with text returns text",
			functionName:        "max",
			argumentTypes:       []querier_dto.SQLType{textType},
			expectedEngineName:  "text",
			expectedCategory:    querier_dto.TypeCategoryText,
			expectedIsAggregate: true,
		},
		{
			description:         "max with blob returns blob",
			functionName:        "max",
			argumentTypes:       []querier_dto.SQLType{blobType},
			expectedEngineName:  "blob",
			expectedCategory:    querier_dto.TypeCategoryBytea,
			expectedIsAggregate: true,
		},
		{
			description:         "sum with integer returns integer",
			functionName:        "sum",
			argumentTypes:       []querier_dto.SQLType{integerType},
			expectedEngineName:  "integer",
			expectedCategory:    querier_dto.TypeCategoryInteger,
			expectedIsAggregate: true,
		},
		{
			description:         "sum with real returns real",
			functionName:        "sum",
			argumentTypes:       []querier_dto.SQLType{realType},
			expectedEngineName:  "real",
			expectedCategory:    querier_dto.TypeCategoryFloat,
			expectedIsAggregate: true,
		},
		{
			description:         "sum with text falls back to real",
			functionName:        "sum",
			argumentTypes:       []querier_dto.SQLType{textType},
			expectedEngineName:  "real",
			expectedCategory:    querier_dto.TypeCategoryFloat,
			expectedIsAggregate: true,
		},
		{
			description:        "coalesce with integer and text returns integer",
			functionName:       "coalesce",
			argumentTypes:      []querier_dto.SQLType{integerType, textType},
			expectedEngineName: "integer",
			expectedCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			description:        "coalesce with unknown then text returns text",
			functionName:       "coalesce",
			argumentTypes:      []querier_dto.SQLType{unknownType, textType},
			expectedEngineName: "text",
			expectedCategory:   querier_dto.TypeCategoryText,
		},
		{
			description:        "coalesce with all unknown returns unknown",
			functionName:       "coalesce",
			argumentTypes:      []querier_dto.SQLType{unknownType, unknownType},
			expectedEngineName: "",
			expectedCategory:   querier_dto.TypeCategoryUnknown,
		},
		{
			description:   "min with no arguments returns nil",
			functionName:  "min",
			argumentTypes: []querier_dto.SQLType{},
			expectNil:     true,
		},
		{
			description:   "sum with no arguments returns nil",
			functionName:  "sum",
			argumentTypes: []querier_dto.SQLType{},
			expectNil:     true,
		},
		{
			description:   "unknown function returns nil",
			functionName:  "totally_unknown_function",
			argumentTypes: []querier_dto.SQLType{integerType},
			expectNil:     true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			resolution, err := engine.ResolveFunctionCall(nil, tc.functionName, "", tc.argumentTypes)
			require.NoError(t, err)

			if tc.expectNil {
				assert.Nil(t, resolution, "expected nil resolution")
				return
			}

			require.NotNil(t, resolution, "expected non-nil resolution")
			assert.Equal(t, tc.expectedEngineName, resolution.ReturnType.EngineName, "return type engine name")
			assert.Equal(t, tc.expectedCategory, resolution.ReturnType.Category, "return type category")
			assert.Equal(t, tc.expectedIsAggregate, resolution.IsAggregate, "is aggregate")
		})
	}
}
