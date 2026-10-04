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
package db_catalogue_postgres

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

var (
	functionParameterColumns = []string{
		"oid", "proname", "prokind", "proisstrict", "proretset", "provariadic",
		"pronargdefaults", "return_type", "mode", "name", "type_name",
	}
)

func parameter(mode string, name string, typeName string) functionParameter {
	return functionParameter{mode: mode, name: name, typeName: typeName}
}

func typeNames(arguments []querier_dto.FunctionArgument) []string {
	names := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		names = append(names, argument.Type.EngineName)
	}
	return names
}

func argumentNames(arguments []querier_dto.FunctionArgument) []string {
	names := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		names = append(names, argument.Name)
	}
	return names
}

func optionalFlags(arguments []querier_dto.FunctionArgument) []bool {
	flags := make([]bool, 0, len(arguments))
	for _, argument := range arguments {
		flags = append(flags, argument.IsOptional)
	}
	return flags
}

func TestBuildFunctionSignature(t *testing.T) {
	testCases := []struct {
		name              string
		wantReturnType    string
		wantArgumentNames []string
		wantArgumentTypes []string
		wantOptional      []bool
		function          introspectedFunction
		wantMinArguments  int
		wantNullable      querier_dto.FunctionNullableBehaviour
		wantStrict        bool
		wantVariadic      bool
		wantAggregate     bool
		wantReturnsSet    bool
	}{
		{
			name: "trailing default makes the last argument optional",
			function: introspectedFunction{
				name: "add", procKind: "f", returnType: "integer", defaultCount: 1, isStrict: true,
				parameters: []functionParameter{
					parameter("i", "a", "integer"),
					parameter("i", "b", "integer"),
				},
			},
			wantReturnType:    "integer",
			wantArgumentNames: []string{"a", "b"},
			wantArgumentTypes: []string{"integer", "integer"},
			wantOptional:      []bool{false, true},
			wantMinArguments:  1,
			wantNullable:      querier_dto.FunctionNullableReturnsNullOnNull,
			wantStrict:        true,
		},
		{
			name: "OUT parameters describe the result rather than the call",
			function: introspectedFunction{
				name: "split_name", procKind: "f", returnType: "record",
				parameters: []functionParameter{
					parameter("i", "full_name", "text"),
					parameter("o", "first_name", "text"),
					parameter("o", "last_name", "text"),
				},
			},
			wantReturnType:    "record",
			wantArgumentNames: []string{"full_name"},
			wantArgumentTypes: []string{"text"},
			wantOptional:      []bool{false},
			wantMinArguments:  1,
			wantNullable:      querier_dto.FunctionNullableCalledOnNull,
		},
		{
			name: "RETURNS TABLE columns are not arguments",
			function: introspectedFunction{
				name: "users_since", procKind: "f", returnType: "record", returnsSet: true,
				parameters: []functionParameter{
					parameter("i", "since", "timestamp with time zone"),
					parameter("t", "id", "integer"),
					parameter("t", "email", "character varying"),
				},
			},
			wantReturnType:    "record",
			wantArgumentNames: []string{"since"},
			wantArgumentTypes: []string{"timestamp with time zone"},
			wantOptional:      []bool{false},
			wantMinArguments:  1,
			wantNullable:      querier_dto.FunctionNullableCalledOnNull,
			wantReturnsSet:    true,
		},
		{
			name: "INOUT parameters are arguments",
			function: introspectedFunction{
				name: "swap", procKind: "f", returnType: "record",
				parameters: []functionParameter{
					parameter("b", "a", "integer"),
					parameter("b", "b", "integer"),
				},
			},
			wantReturnType:    "record",
			wantArgumentNames: []string{"a", "b"},
			wantArgumentTypes: []string{"integer", "integer"},
			wantOptional:      []bool{false, false},
			wantMinArguments:  2,
			wantNullable:      querier_dto.FunctionNullableCalledOnNull,
		},
		{
			name: "VARIADIC parameter without a default needs one value",
			function: introspectedFunction{
				name: "total", procKind: "f", returnType: "numeric", isVariadic: true,
				parameters: []functionParameter{parameter("v", "amounts", "numeric[]")},
			},
			wantReturnType:    "numeric",
			wantArgumentNames: []string{"amounts"},
			wantArgumentTypes: []string{"numeric[]"},
			wantOptional:      []bool{false},
			wantMinArguments:  1,
			wantNullable:      querier_dto.FunctionNullableCalledOnNull,
			wantVariadic:      true,
		},
		{
			name: "VARIADIC parameter with a default accepts no values",
			function: introspectedFunction{
				name: "join_parts", procKind: "f", returnType: "text", isVariadic: true, defaultCount: 1,
				parameters: []functionParameter{
					parameter("i", "separator", "text"),
					parameter("v", "parts", "text[]"),
				},
			},
			wantReturnType:    "text",
			wantArgumentNames: []string{"separator", "parts"},
			wantArgumentTypes: []string{"text", "text[]"},
			wantOptional:      []bool{false, true},
			wantMinArguments:  1,
			wantNullable:      querier_dto.FunctionNullableCalledOnNull,
			wantVariadic:      true,
		},
		{
			name: "quoted names containing commas and quotes stay whole",
			function: introspectedFunction{
				name: "weird, name", procKind: "f", returnType: "text", defaultCount: 2,
				parameters: []functionParameter{
					parameter("i", "quoted, arg", "text"),
					parameter("i", "x\"y", "numeric"),
				},
			},
			wantReturnType:    "text",
			wantArgumentNames: []string{"quoted, arg", "x\"y"},
			wantArgumentTypes: []string{"text", "numeric"},
			wantOptional:      []bool{true, true},
			wantMinArguments:  0,
			wantNullable:      querier_dto.FunctionNullableCalledOnNull,
		},
		{
			name: "aggregate with an unnamed argument",
			function: introspectedFunction{
				name: "my_sum", procKind: "a", returnType: "numeric",
				parameters: []functionParameter{parameter("i", "", "numeric")},
			},
			wantReturnType:    "numeric",
			wantArgumentNames: []string{""},
			wantArgumentTypes: []string{"numeric"},
			wantOptional:      []bool{false},
			wantMinArguments:  1,
			wantNullable:      querier_dto.FunctionNullableCalledOnNull,
			wantAggregate:     true,
		},
		{
			name: "window function counts as an aggregate",
			function: introspectedFunction{
				name: "my_rank", procKind: "w", returnType: "bigint",
			},
			wantReturnType:    "bigint",
			wantArgumentNames: []string{},
			wantArgumentTypes: []string{},
			wantOptional:      []bool{},
			wantNullable:      querier_dto.FunctionNullableCalledOnNull,
			wantAggregate:     true,
		},
		{
			name: "set-returning function without parameters",
			function: introspectedFunction{
				name: "noargs", procKind: "f", returnType: "integer", returnsSet: true,
			},
			wantReturnType:    "integer",
			wantArgumentNames: []string{},
			wantArgumentTypes: []string{},
			wantOptional:      []bool{},
			wantNullable:      querier_dto.FunctionNullableCalledOnNull,
			wantReturnsSet:    true,
		},
		{
			name: "more defaults than inputs never makes the minimum negative",
			function: introspectedFunction{
				name: "odd", procKind: "f", returnType: "void", defaultCount: 3,
				parameters: []functionParameter{parameter("i", "only", "text")},
			},
			wantReturnType:    "void",
			wantArgumentNames: []string{"only"},
			wantArgumentTypes: []string{"text"},
			wantOptional:      []bool{true},
			wantMinArguments:  0,
			wantNullable:      querier_dto.FunctionNullableCalledOnNull,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			signature := buildFunctionSignature(testCase.function, "app", modifierTypeNormaliser{})

			assert.Equal(t, testCase.function.name, signature.Name)
			assert.Equal(t, "app", signature.Schema)
			assert.Equal(t, testCase.wantReturnType, signature.ReturnType.EngineName)
			assert.Equal(t, testCase.wantArgumentNames, argumentNames(signature.Arguments))
			assert.Equal(t, testCase.wantArgumentTypes, typeNames(signature.Arguments))
			assert.Equal(t, testCase.wantOptional, optionalFlags(signature.Arguments))
			assert.Equal(t, testCase.wantMinArguments, signature.MinArguments)
			assert.Equal(t, testCase.wantNullable, signature.NullableBehaviour)
			assert.Equal(t, testCase.wantStrict, signature.IsStrict)
			assert.Equal(t, testCase.wantVariadic, signature.IsVariadic)
			assert.Equal(t, testCase.wantAggregate, signature.IsAggregate)
			assert.Equal(t, testCase.wantReturnsSet, signature.ReturnsSet)
		})
	}
}

func TestIsInputParameterMode(t *testing.T) {
	testCases := []struct {
		mode string
		want bool
	}{
		{mode: "i", want: true},
		{mode: "b", want: true},
		{mode: "v", want: true},
		{mode: "o", want: false},
		{mode: "t", want: false},
		{mode: "", want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.mode, func(t *testing.T) {
			assert.Equal(t, testCase.want, isInputParameterMode(testCase.mode))
		})
	}
}

func parameterRow(functionID int64, name string, mode string, parameterName string, typeName string) functionParameterRow {
	row := functionParameterRow{}
	row.functionID = functionID
	row.functionName = name
	row.procKind = "f"
	row.returnType = "integer"
	if typeName != "" {
		row.parameterMode = sql.NullString{String: mode, Valid: true}
		row.parameterName = sql.NullString{String: parameterName, Valid: true}
		row.parameterType = sql.NullString{String: typeName, Valid: true}
	}
	return row
}

func TestGroupFunctionParameterRows(t *testing.T) {
	testCases := []struct {
		name           string
		rows           []functionParameterRow
		wantNames      []string
		wantParameters [][]functionParameter
	}{
		{
			name:           "no rows",
			rows:           nil,
			wantNames:      nil,
			wantParameters: nil,
		},
		{
			name: "overloads with one name stay separate",
			rows: []functionParameterRow{
				parameterRow(1, "area", "i", "side", "integer"),
				parameterRow(2, "area", "i", "width", "numeric"),
				parameterRow(2, "area", "i", "height", "numeric"),
			},
			wantNames: []string{"area", "area"},
			wantParameters: [][]functionParameter{
				{parameter("i", "side", "integer")},
				{parameter("i", "width", "numeric"), parameter("i", "height", "numeric")},
			},
		},
		{
			name: "function without parameters has an empty list",
			rows: []functionParameterRow{
				parameterRow(7, "now_utc", "", "", ""),
				parameterRow(8, "split_name", "i", "full_name", "text"),
				parameterRow(8, "split_name", "o", "first_name", "text"),
			},
			wantNames: []string{"now_utc", "split_name"},
			wantParameters: [][]functionParameter{
				nil,
				{parameter("i", "full_name", "text"), parameter("o", "first_name", "text")},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			functions := groupFunctionParameterRows(testCase.rows)

			var names []string
			var parameters [][]functionParameter
			for _, function := range functions {
				names = append(names, function.name)
				parameters = append(parameters, function.parameters)
			}
			assert.Equal(t, testCase.wantNames, names)
			assert.Equal(t, testCase.wantParameters, parameters)
		})
	}
}

func TestIntrospectFunctionsReadsStructuredParameters(t *testing.T) {
	database := openFakeDatabase(t, fakeQueryHandler{
		marker: "FROM pg_proc p",
		respond: fixedResult(functionParameterColumns,
			[]driver.Value{int64(10), "add", "f", true, false, false, int64(1), "integer", "i", "a", "integer"},
			[]driver.Value{int64(10), "add", "f", true, false, false, int64(1), "integer", "i", "b", "integer"},
			[]driver.Value{int64(11), "noargs", "f", false, true, false, int64(0), "integer", nil, nil, nil},
			[]driver.Value{int64(12), "total", "f", false, false, true, int64(0), "numeric", "v", "amounts", "numeric[]"},
		),
	})
	provider := NewPgIntrospectionProvider(database, modifierTypeNormaliser{})
	schema := newTestSchema("app")

	require.NoError(t, provider.introspectFunctions(t.Context(), schema))

	require.Len(t, schema.Functions["add"], 1)
	add := schema.Functions["add"][0]
	assert.Equal(t, []string{"a", "b"}, argumentNames(add.Arguments))
	assert.Equal(t, 1, add.MinArguments)
	assert.True(t, add.IsStrict)

	require.Len(t, schema.Functions["noargs"], 1)
	assert.Empty(t, schema.Functions["noargs"][0].Arguments)
	assert.True(t, schema.Functions["noargs"][0].ReturnsSet)

	require.Len(t, schema.Functions["total"], 1)
	assert.True(t, schema.Functions["total"][0].IsVariadic)
	assert.Equal(t, 1, schema.Functions["total"][0].MinArguments)
}

func TestIntrospectFunctionsReportsQueryFailures(t *testing.T) {
	queryFailure := errors.New("permission denied for pg_proc")
	closeFailure := errors.New("connection reset while closing")
	iterationFailure := errors.New("connection reset while reading")

	testCases := []struct {
		wantError error
		name      string
		result    fakeQueryResult
	}{
		{
			name:      "query fails",
			result:    fakeQueryResult{err: queryFailure},
			wantError: queryFailure,
		},
		{
			name:      "iteration fails",
			result:    fakeQueryResult{columns: functionParameterColumns, nextErr: iterationFailure},
			wantError: iterationFailure,
		},
		{
			name:      "closing the rows fails",
			result:    fakeQueryResult{columns: functionParameterColumns, closeErr: closeFailure},
			wantError: closeFailure,
		},
		{
			name: "row cannot be scanned",
			result: fakeQueryResult{
				columns: functionParameterColumns,
				rows:    [][]driver.Value{{"not a number", "f", "f", true, false, false, int64(0), "integer", nil, nil, nil}},
			},
			wantError: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			database := openFakeDatabase(t, fakeQueryHandler{
				marker:  "FROM pg_proc p",
				respond: func([]driver.NamedValue) fakeQueryResult { return testCase.result },
			})
			provider := NewPgIntrospectionProvider(database, modifierTypeNormaliser{})

			err := provider.introspectFunctions(t.Context(), newTestSchema("app"))
			require.Error(t, err)
			if testCase.wantError != nil {
				assert.ErrorIs(t, err, testCase.wantError)
			}
		})
	}
}
