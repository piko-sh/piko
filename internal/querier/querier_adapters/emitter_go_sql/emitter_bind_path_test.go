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

package emitter_go_sql

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestSQLStrategyMaxBindVariablesPerEngine(t *testing.T) {
	testCases := []struct {
		name     string
		strategy *sqlStrategy
		expected int
	}{
		{
			name:     "sqlite stays at the SQLITE_MAX_VARIABLE_NUMBER default",
			strategy: &sqlStrategy{},
			expected: maxSQLiteBindVariables,
		},
		{
			name:     "mysql uses the 16-bit wire-protocol ceiling",
			strategy: &sqlStrategy{plainPlaceholders: true},
			expected: maxMySQLBindVariables,
		},
		{
			name:     "clickhouse follows the same 16-bit ceiling",
			strategy: &sqlStrategy{wrapAsClickHouseNamed: true},
			expected: maxClickHouseBindVariables,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, testCase.strategy.MaxBindVariables())
		})
	}
}

func TestSQLStrategyPlaceholderDialectFlags(t *testing.T) {
	testCases := []struct {
		name                   string
		strategy               *sqlStrategy
		usesNumberedParams     bool
		preservesPlaceholderIx bool
	}{
		{
			name:                   "sqlite keeps numbered placeholder indices",
			strategy:               &sqlStrategy{},
			usesNumberedParams:     false,
			preservesPlaceholderIx: true,
		},
		{
			name:                   "mysql collapses to anonymous placeholders",
			strategy:               &sqlStrategy{plainPlaceholders: true},
			usesNumberedParams:     false,
			preservesPlaceholderIx: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.usesNumberedParams, testCase.strategy.UsesNumberedParams())
			assert.Equal(t, testCase.preservesPlaceholderIx, testCase.strategy.PreservesPlaceholderIndices())
		})
	}
}

func sliceManyQuery() *querier_dto.AnalysedQuery {
	return &querier_dto.AnalysedQuery{
		Name:     "FetchDueTasks",
		Command:  querier_dto.QueryCommandMany,
		SQL:      "-- piko.name: FetchDueTasks\n-- ?1 as piko.slice(statuses)\nSELECT id, status FROM tasks WHERE status IN (?1)",
		Filename: "tasks.sql",
		Parameters: []querier_dto.QueryParameter{
			{Number: 1, Name: "statuses", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}, IsSlice: true},
		},
		OutputColumns: []querier_dto.OutputColumn{
			{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
			{Name: "status", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
		},
	}
}

func TestEmitQueriesStaticSliceReturnsWrappedSentinelOnOverCap(t *testing.T) {
	emitter := NewSQLEmitter()

	files, err := emitter.EmitQueries("db", []*querier_dto.AnalysedQuery{sliceManyQuery()}, defaultMappings())
	require.NoError(t, err)

	bindLimits := findGeneratedFile(t, files, "bind_limits.go")
	sliceHelpers := findGeneratedFile(t, files, "slice_helpers.go")
	querySource := findGeneratedFile(t, files, "tasks.sql.go")

	for name, source := range map[string]string{
		"bind_limits.go":   bindLimits,
		"slice_helpers.go": sliceHelpers,
		"tasks.sql.go":     querySource,
	} {
		fileSet := token.NewFileSet()
		_, parseError := parser.ParseFile(fileSet, name, source, parser.AllErrors)
		require.NoError(t, parseError, "%s must be valid Go:\n%s", name, source)
	}

	assert.Contains(t, bindLimits, "const pikoMaxBindVariables = 999")
	assert.Contains(t, bindLimits, `var errPikoTooManyBindVariables = errors.New("piko: too many bind variables")`)

	assert.Contains(t, sliceHelpers, "func pikoExpandSlicePlaceholders(query string, specs []pikoSliceExpansionSpec) (string, error)")
	assert.Contains(t, sliceHelpers, "totalBindCount > pikoMaxBindVariables")
	assert.Contains(t, sliceHelpers, "%w")
	assert.Contains(t, sliceHelpers, "errPikoTooManyBindVariables")

	assert.Contains(t, querySource, "pikoExpandSlicePlaceholders(fetchduetasks")
	assert.Contains(t, querySource, "expansionError != nil")
}

func TestEmitQueriesStaticSliceForClickHouseUsesClickHouseCap(t *testing.T) {
	emitter := NewSQLEmitterForClickHouse()

	files, err := emitter.EmitQueries("db", []*querier_dto.AnalysedQuery{sliceManyQuery()}, defaultMappings())
	require.NoError(t, err)

	bindLimits := findGeneratedFile(t, files, "bind_limits.go")

	assert.Contains(t, bindLimits, "const pikoMaxBindVariables = 65535")
}

func TestEmitQueriesClickHouseWrapsParametersAndEmitsFormatHelper(t *testing.T) {
	emitter := NewSQLEmitterForClickHouse()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "GetByName",
			Command:  querier_dto.QueryCommandOne,
			SQL:      "SELECT id FROM users WHERE name = {name:String}",
			Filename: "users.sql",
			Parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "name", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
			},
			OutputColumns: []querier_dto.OutputColumn{
				{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
			},
		},
	}

	files, err := emitter.EmitQueries("db", queries, defaultMappings())
	require.NoError(t, err)

	querySource := findGeneratedFile(t, files, "users.sql.go")
	helperSource := findGeneratedFile(t, files, "clickhouse_format.go")

	fileSet := token.NewFileSet()
	_, queryParseError := parser.ParseFile(fileSet, "users.sql.go", querySource, parser.AllErrors)
	require.NoError(t, queryParseError, "generated query must be valid Go:\n%s", querySource)
	_, helperParseError := parser.ParseFile(fileSet, "clickhouse_format.go", helperSource, parser.AllErrors)
	require.NoError(t, helperParseError, "generated helper must be valid Go:\n%s", helperSource)

	assert.Contains(t, querySource, "{p_name:String}")
	assert.Contains(t, querySource, `clickhouse.Named("p_name", pikoClickHouseFormat(name))`)
	assert.Contains(t, querySource, `"github.com/ClickHouse/clickhouse-go/v2"`)

	assert.Contains(t, helperSource, "func pikoClickHouseFormat(value any) string")
	assert.Contains(t, helperSource, "func pikoClickHouseLiteral(value any) string")
}

func TestEmitQueriesClickHousePrefixesReservedKeywordParamName(t *testing.T) {
	emitter := NewSQLEmitterForClickHouse()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "TopEvents",
			Command:  querier_dto.QueryCommandMany,
			SQL:      "SELECT id FROM events ORDER BY id LIMIT {limit:Int32}",
			Filename: "events.sql",
			Parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "limit", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int32"}},
			},
			OutputColumns: []querier_dto.OutputColumn{
				{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
			},
		},
	}

	files, err := emitter.EmitQueries("db", queries, defaultMappings())
	require.NoError(t, err)

	querySource := findGeneratedFile(t, files, "events.sql.go")

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "events.sql.go", querySource, parser.AllErrors)
	require.NoError(t, parseError, "generated query must be valid Go:\n%s", querySource)

	assert.Contains(t, querySource, "{p_limit:Int32}")
	assert.Contains(t, querySource, `clickhouse.Named("p_limit", pikoClickHouseFormat(limit))`)
	assert.NotContains(t, querySource, "{limit:Int32}")
}

func TestSQLStrategyWrapParameterAccessReturnsAccessVerbatimWhenNotClickHouse(t *testing.T) {
	emitter := NewSQLEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:     "GetByName",
			Command:  querier_dto.QueryCommandOne,
			SQL:      "SELECT id FROM users WHERE name = $1",
			Filename: "users.sql",
			Parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "name", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
			},
			OutputColumns: []querier_dto.OutputColumn{
				{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
			},
		},
	}

	files, err := emitter.EmitQueries("db", queries, defaultMappings())
	require.NoError(t, err)

	querySource := findGeneratedFile(t, files, "users.sql.go")

	assert.NotContains(t, querySource, "clickhouse.Named")
	assert.NotContains(t, querySource, "pikoClickHouseFormat")
	for _, file := range files {
		assert.NotEqual(t, "clickhouse_format.go", file.Name, "no helper expected for the positional emitter")
	}
}

func TestEmitOTelMapsQueryConstantsToOperationNames(t *testing.T) {
	emitter := NewSQLEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:    "GetUserByID",
			Command: querier_dto.QueryCommandOne,
			SQL:     "SELECT id FROM users WHERE id = $1",
		},
	}

	file, err := emitter.EmitOTel("mydb", queries)
	require.NoError(t, err)
	assert.Equal(t, "otel.go", file.Name)

	source := string(file.Content)

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "otel.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated code must be valid Go:\n%s", source)

	assert.Contains(t, source, "package mydb")
	assert.Contains(t, source, "func QueryNameResolver(query string) string")

	assert.Contains(t, source, "var queryNameMap = map[string]string{")
	assert.Contains(t, source, "getuserbyid")
}
