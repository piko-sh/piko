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

package db_engine_duckdb

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestParseFunctionArgumentListRejectsMalformedArguments(t *testing.T) {
	t.Parallel()

	tokens, tokeniseError := tokenise("(123)")
	require.NoError(t, tokeniseError)

	parser := newParser(tokens)
	_, argumentsError := parser.parseFunctionArgumentList(NewDuckDBEngine())
	require.ErrorIs(t, argumentsError, errMalformedFunctionArguments)
}

func TestApplyDDL_CreateMacroVariants(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		sql          string
		wantName     string
		wantOptional []bool
	}{
		{name: "temporary macro", sql: `CREATE TEMP MACRO add_one(x) AS x + 1`, wantName: "add_one", wantOptional: []bool{false}},
		{name: "temporary macro spelt out", sql: `CREATE TEMPORARY MACRO add_one(x) AS x + 1`, wantName: "add_one", wantOptional: []bool{false}},
		{name: "replaced temporary macro", sql: `CREATE OR REPLACE TEMP MACRO add_one(x) AS x + 1`, wantName: "add_one", wantOptional: []bool{false}},
		{name: "parameter default", sql: `CREATE MACRO add_default(a, b := 5) AS a + b`, wantName: "add_default", wantOptional: []bool{false, true}},
		{name: "parameter default without spaces", sql: `CREATE MACRO greet(name:='world') AS 'Hello, ' || name`, wantName: "greet", wantOptional: []bool{true}},
		{name: "parenthesised default", sql: `CREATE MACRO scaled(a, factor := (2 * 3)) AS a * factor`, wantName: "scaled", wantOptional: []bool{false, true}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mutation := applyDDL(t, testCase.sql)

			require.NotNil(t, mutation)
			assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
			require.NotNil(t, mutation.FunctionSignature)
			assert.Equal(t, testCase.wantName, mutation.FunctionSignature.Name)
			require.Len(t, mutation.FunctionSignature.Arguments, len(testCase.wantOptional))
			for index, wantOptional := range testCase.wantOptional {
				assert.Equalf(t, wantOptional, mutation.FunctionSignature.Arguments[index].IsOptional, "argument %d", index)
			}
		})
	}
}

func TestApplyDDL_DefaultedMacroParameterIsNamedLikeOtherParameters(t *testing.T) {
	t.Parallel()

	mutation := applyDDL(t, `CREATE MACRO add_default(a, b := 5) AS a + b`)

	require.Len(t, mutation.FunctionSignature.Arguments, 2)
	assert.Equal(t, "a", mutation.FunctionSignature.Arguments[0].Type.EngineName)
	assert.Equal(t, "b", mutation.FunctionSignature.Arguments[1].Type.EngineName)
}

func TestFunctionArgumentArrayType(t *testing.T) {
	t.Parallel()

	element := querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varchar")

	testCases := []struct {
		name       string
		wantName   string
		dimensions int
	}{
		{name: "scalar", dimensions: 0, wantName: "varchar"},
		{name: "one dimension", dimensions: 1, wantName: "varchar[]"},
		{name: "two dimensions", dimensions: 2, wantName: "varchar[][]"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			wrapped := functionArgumentArrayType(element, testCase.dimensions)

			assert.Equal(t, testCase.wantName, wrapped.EngineName)
			depth := 0
			for current := &wrapped; current.ElementType != nil; current = current.ElementType {
				assert.Equal(t, querier_dto.TypeCategoryArray, current.Category)
				depth++
			}
			assert.Equal(t, testCase.dimensions, depth)
		})
	}
}

func TestApplyDDL_MacroArgumentArrayDimensionsAreBounded(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()
	statements, err := engine.ParseStatements(`CREATE MACRO f(x VARCHAR` + strings.Repeat("[]", 10_000) + `) AS 1`)
	require.NoError(t, err)
	require.Len(t, statements, 1)

	_, err = engine.ApplyDDL(context.Background(), statements[0])

	require.ErrorIs(t, err, errTooManyArrayDimensions)
}
