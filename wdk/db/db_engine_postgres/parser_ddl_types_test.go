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

func TestApplyDDL_CreateFunctionParameterModes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		sql            string
		wantArguments  []string
		wantReturnType string
		wantColumns    []string
		wantVariadic   bool
		wantReturnsSet bool
	}{
		{
			name:           "OUT parameters are not callable arguments",
			sql:            "CREATE FUNCTION f(IN a int, OUT b text, OUT c int) LANGUAGE sql AS 'SELECT 1'",
			wantArguments:  []string{"a"},
			wantReturnType: "",
			wantColumns:    []string{"b", "c"},
		},
		{
			name:           "a single OUT parameter is the return type",
			sql:            "CREATE FUNCTION f(a int, OUT total int8) LANGUAGE sql AS 'SELECT 1'",
			wantArguments:  []string{"a"},
			wantReturnType: "int8",
		},
		{
			name:           "INOUT is both an argument and a result",
			sql:            "CREATE FUNCTION f(INOUT a int, b text) RETURNS record LANGUAGE sql AS 'SELECT 1'",
			wantArguments:  []string{"a", "b"},
			wantReturnType: "int4",
		},
		{
			name:           "set-returning OUT parameters become a row type",
			sql:            "CREATE FUNCTION f(a int, OUT b text, OUT c int) RETURNS SETOF record LANGUAGE sql AS 'SELECT 1'",
			wantArguments:  []string{"a"},
			wantReturnType: "",
			wantColumns:    []string{"b", "c"},
			wantReturnsSet: true,
		},
		{
			name:           "an explicit return type keeps precedence",
			sql:            "CREATE FUNCTION f(a int, OUT b text) RETURNS text LANGUAGE sql AS 'SELECT 1'",
			wantArguments:  []string{"a"},
			wantReturnType: "text",
		},
		{
			name:           "RETURNS TABLE keeps its own columns",
			sql:            "CREATE FUNCTION f(a int, OUT b text, OUT c int) RETURNS TABLE (d int) LANGUAGE sql AS 'SELECT 1'",
			wantArguments:  []string{"a"},
			wantReturnType: "",
			wantColumns:    []string{"d"},
			wantReturnsSet: true,
		},
		{
			name:           "VARIADIC is a callable argument",
			sql:            "CREATE FUNCTION f(a int, VARIADIC rest int[]) RETURNS int LANGUAGE sql AS 'SELECT 1'",
			wantArguments:  []string{"a", "rest"},
			wantReturnType: "int4",
			wantVariadic:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mutation := applyDDL(t, testCase.sql)

			require.NotNil(t, mutation.FunctionSignature)
			signature := mutation.FunctionSignature
			assert.Equal(t, testCase.wantArguments, argumentNames(signature.Arguments))
			assert.Equal(t, testCase.wantReturnType, signature.ReturnType.EngineName)
			assert.Equal(t, testCase.wantColumns, columnNames(mutation.Columns))
			assert.Equal(t, testCase.wantVariadic, signature.IsVariadic)
			assert.Equal(t, testCase.wantReturnsSet, signature.ReturnsSet)
		})
	}
}

func TestArrayTypeOf(t *testing.T) {
	t.Parallel()

	text := querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text")

	testCases := []struct {
		name       string
		wantEngine string
		dimensions int
		wantDepth  int
	}{
		{name: "zero dimensions keeps the element", dimensions: 0, wantEngine: "text", wantDepth: 0},
		{name: "one dimension", dimensions: 1, wantEngine: "text[]", wantDepth: 1},
		{name: "two dimensions", dimensions: 2, wantEngine: "text[][]", wantDepth: 2},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			wrapped := arrayTypeOf(text, testCase.dimensions)

			assert.Equal(t, testCase.wantEngine, wrapped.EngineName)
			depth := 0
			for current := wrapped; current.Category == querier_dto.TypeCategoryArray; current = *current.ElementType {
				depth++
			}
			assert.Equal(t, testCase.wantDepth, depth)
		})
	}
}

func argumentNames(arguments []querier_dto.FunctionArgument) []string {
	var names []string
	for _, argument := range arguments {
		names = append(names, argument.Name)
	}
	return names
}

func columnNames(columns []querier_dto.Column) []string {
	var names []string
	for _, column := range columns {
		names = append(names, column.Name)
	}
	return names
}
