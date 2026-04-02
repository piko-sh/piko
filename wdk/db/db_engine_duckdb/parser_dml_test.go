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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func buildTestCatalogue() *querier_dto.Catalogue {
	return &querier_dto.Catalogue{
		DefaultSchema: "main",
		Schemas: map[string]*querier_dto.Schema{
			"main": {
				Name: "main",
				Tables: map[string]*querier_dto.Table{
					"users": {
						Name:   "users",
						Schema: "main",
						Columns: []querier_dto.Column{
							{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}, Nullable: false},
							{Name: "name", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}, Nullable: false},
							{Name: "email", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}, Nullable: true},
							{Name: "age", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}, Nullable: true},
							{Name: "active", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: "bool"}, Nullable: false},
						},
						PrimaryKey: []string{"id"},
					},
					"orders": {
						Name:   "orders",
						Schema: "main",
						Columns: []querier_dto.Column{
							{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}, Nullable: false},
							{Name: "user_id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}, Nullable: false},
							{Name: "total", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"}, Nullable: true},
							{Name: "created_at", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp"}, Nullable: false},
						},
						PrimaryKey: []string{"id"},
					},
				},
				Views:          map[string]*querier_dto.View{},
				Enums:          map[string]*querier_dto.Enum{},
				Functions:      map[string][]*querier_dto.FunctionSignature{},
				CompositeTypes: map[string]*querier_dto.CompositeType{},
				Sequences:      map[string]*querier_dto.Sequence{},
			},
		},
	}
}

func analyseQuery(t *testing.T, catalogue *querier_dto.Catalogue, sql string) *querier_dto.RawQueryAnalysis {
	t.Helper()
	engine := NewDuckDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.NotEmpty(t, statements)
	analysis, err := engine.AnalyseQuery(catalogue, statements[0])
	require.NoError(t, err)
	require.NotNil(t, analysis)
	return analysis
}

func TestAnalyseQuery_Select(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("simple column selection", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, name FROM users`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.Equal(t, "id", analysis.OutputColumns[0].Name)
		assert.Equal(t, "name", analysis.OutputColumns[1].Name)
		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
		assert.True(t, analysis.ReadOnly, "SELECT should be read-only")
	})

	t.Run("star expansion", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT * FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.OutputColumns[0].IsStar, "should detect star expansion")
		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
	})

	t.Run("table-qualified star", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT u.* FROM users u`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.OutputColumns[0].IsStar)
		assert.Equal(t, "u", analysis.OutputColumns[0].TableAlias)
	})

	t.Run("WHERE with dollar parameter", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, name FROM users WHERE id = $1`)

		require.Len(t, analysis.OutputColumns, 2)
		require.Len(t, analysis.ParameterReferences, 1)
		assert.Equal(t, 1, analysis.ParameterReferences[0].Number)
	})

	t.Run("multiple parameters", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE name = $1 AND age > $2`)

		require.Len(t, analysis.ParameterReferences, 2)
		assert.Equal(t, 1, analysis.ParameterReferences[0].Number)
		assert.Equal(t, 2, analysis.ParameterReferences[1].Number)
	})

	t.Run("INNER JOIN", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
		assert.Equal(t, "u", analysis.FromTables[0].Alias)
		require.Len(t, analysis.JoinClauses, 1)
		assert.Equal(t, querier_dto.JoinInner, analysis.JoinClauses[0].Kind)
		assert.Equal(t, "orders", analysis.JoinClauses[0].Table.Name)
		assert.Equal(t, "o", analysis.JoinClauses[0].Table.Alias)
	})

	t.Run("LEFT JOIN", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT u.name, o.total FROM users u LEFT JOIN orders o ON u.id = o.user_id`)

		require.Len(t, analysis.JoinClauses, 1)
		assert.Equal(t, querier_dto.JoinLeft, analysis.JoinClauses[0].Kind)
	})

	t.Run("CROSS JOIN", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT u.name, o.total FROM users u CROSS JOIN orders o`)

		require.Len(t, analysis.JoinClauses, 1)
		assert.Equal(t, querier_dto.JoinCross, analysis.JoinClauses[0].Kind)
		assert.Equal(t, "orders", analysis.JoinClauses[0].Table.Name)
	})

	t.Run("aggregate function in output", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT count(*) FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.Equal(t, "count", analysis.OutputColumns[0].Name)
	})

	t.Run("column alias", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT name AS user_name FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.Equal(t, "user_name", analysis.OutputColumns[0].Name)
	})

	t.Run("CTE with SELECT", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `WITH active AS (
			SELECT id, name FROM users WHERE active = true
		)
		SELECT id, name FROM active`)

		require.Len(t, analysis.CTEDefinitions, 1)
		assert.Equal(t, "active", analysis.CTEDefinitions[0].Name)
		require.Len(t, analysis.CTEDefinitions[0].OutputColumns, 2)
		assert.Equal(t, "id", analysis.CTEDefinitions[0].OutputColumns[0].Name)
		assert.Equal(t, "name", analysis.CTEDefinitions[0].OutputColumns[1].Name)
	})

	t.Run("ORDER BY and LIMIT", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, name FROM users ORDER BY name ASC LIMIT 10`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("GROUP BY", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT active, count(*) FROM users GROUP BY active`)

		require.Len(t, analysis.GroupByColumns, 1)
		assert.Equal(t, "active", analysis.GroupByColumns[0].ColumnName)
	})

	t.Run("comma-joined tables", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT u.name, o.total FROM users u, orders o WHERE u.id = o.user_id`)

		require.Len(t, analysis.FromTables, 2)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
		assert.Equal(t, "orders", analysis.FromTables[1].Name)
	})

	t.Run("UNION ALL compound query", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, name FROM users WHERE active = true
			UNION ALL
			SELECT id, name FROM users WHERE active = false`)

		require.Len(t, analysis.CompoundBranches, 1)
		assert.Equal(t, querier_dto.CompoundUnionAll, analysis.CompoundBranches[0].Operator)
		require.NotNil(t, analysis.CompoundBranches[0].Query)
	})

	t.Run("LIMIT with parameter", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users LIMIT $1`)

		require.Len(t, analysis.ParameterReferences, 1)
		assert.Equal(t, querier_dto.ParameterContextLimit, analysis.ParameterReferences[0].Context)
	})

	t.Run("OFFSET with parameter", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users LIMIT 10 OFFSET $1`)

		require.Len(t, analysis.ParameterReferences, 1)
		assert.Equal(t, querier_dto.ParameterContextOffset, analysis.ParameterReferences[0].Context)
	})

	t.Run("schema-qualified table in FROM", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM main.users`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "main", analysis.FromTables[0].Schema)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
	})
}

func TestAnalyseQuery_Select_ClausesAndExpressions(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("DISTINCT modifier", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT DISTINCT name FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.Equal(t, "name", analysis.OutputColumns[0].Name)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("HAVING clause", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT user_id, COUNT(*) FROM orders GROUP BY user_id HAVING COUNT(*) > 1`)

		require.Len(t, analysis.GroupByColumns, 1)
		assert.Equal(t, "user_id", analysis.GroupByColumns[0].ColumnName)
	})

	t.Run("GROUP BY multiple columns", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT active, name, COUNT(*) FROM users GROUP BY active, name`)

		require.Len(t, analysis.GroupByColumns, 2)
		assert.Equal(t, "active", analysis.GroupByColumns[0].ColumnName)
		assert.Equal(t, "name", analysis.GroupByColumns[1].ColumnName)
	})

	t.Run("window function with OVER", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, ROW_NUMBER() OVER (ORDER BY id) FROM users`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("window function with PARTITION BY", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT user_id, total, ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY total DESC) FROM orders`)

		require.Len(t, analysis.OutputColumns, 3)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("CASE expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, CASE WHEN active = true THEN 'yes' ELSE 'no' END FROM users`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("COALESCE expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT COALESCE(email, 'none') FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("CAST expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT CAST(id AS VARCHAR) FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("IS NULL condition", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE email IS NULL`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("IS NOT NULL condition", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE email IS NOT NULL`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("BETWEEN with parameters in WHERE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE id BETWEEN $1 AND $2`)

		require.Len(t, analysis.ParameterReferences, 2)

		assert.Equal(t, 1, analysis.ParameterReferences[0].Number)
		assert.Equal(t, 2, analysis.ParameterReferences[1].Number)
	})

	t.Run("IN list with parameters", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE id IN ($1, $2, $3)`)

		require.Len(t, analysis.ParameterReferences, 3)
		assert.Equal(t, querier_dto.ParameterContextInList, analysis.ParameterReferences[0].Context)
	})

	t.Run("LIKE expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, name FROM users WHERE name LIKE '%test%'`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("ILIKE expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE name ILIKE '%test%'`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("UNION compound query", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, name FROM users WHERE active = true
			UNION
			SELECT id, name FROM users WHERE active = false`)

		require.Len(t, analysis.CompoundBranches, 1)
		assert.Equal(t, querier_dto.CompoundUnion, analysis.CompoundBranches[0].Operator)
	})

	t.Run("INTERSECT compound query", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users INTERSECT SELECT user_id FROM orders`)

		require.Len(t, analysis.CompoundBranches, 1)
		assert.Equal(t, querier_dto.CompoundIntersect, analysis.CompoundBranches[0].Operator)
	})

	t.Run("EXCEPT compound query", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users EXCEPT SELECT user_id FROM orders`)

		require.Len(t, analysis.CompoundBranches, 1)
		assert.Equal(t, querier_dto.CompoundExcept, analysis.CompoundBranches[0].Operator)
	})

	t.Run("subquery in FROM clause", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT s.id FROM (SELECT id FROM users) s`)

		require.NotEmpty(t, analysis.RawDerivedTables)
		assert.Equal(t, "s", analysis.RawDerivedTables[0].Alias)
	})

	t.Run("correlated scalar subquery", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) FROM users u`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("EXISTS subquery in WHERE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("RIGHT JOIN", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT u.id, o.total FROM users u RIGHT JOIN orders o ON u.id = o.user_id`)

		require.Len(t, analysis.JoinClauses, 1)
		assert.Equal(t, querier_dto.JoinRight, analysis.JoinClauses[0].Kind)
	})

	t.Run("FULL OUTER JOIN", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT u.id, o.total FROM users u FULL OUTER JOIN orders o ON u.id = o.user_id`)

		require.Len(t, analysis.JoinClauses, 1)
		assert.Equal(t, querier_dto.JoinFull, analysis.JoinClauses[0].Kind)
	})

	t.Run("multiple JOINs", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id LEFT JOIN orders o2 ON u.id = o2.user_id`)

		require.NotEmpty(t, analysis.JoinClauses)
		assert.Equal(t, querier_dto.JoinInner, analysis.JoinClauses[0].Kind)
		assert.Equal(t, "orders", analysis.JoinClauses[0].Table.Name)
	})

	t.Run("ORDER BY with NULLS FIRST", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, email FROM users ORDER BY email ASC NULLS FIRST`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("ORDER BY with NULLS LAST", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, email FROM users ORDER BY email DESC NULLS LAST`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("CAST parameter context detection", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT CAST($1 AS INTEGER) FROM users`)

		require.Len(t, analysis.ParameterReferences, 1)
		assert.Equal(t, querier_dto.ParameterContextCast, analysis.ParameterReferences[0].Context)
		require.NotNil(t, analysis.ParameterReferences[0].CastType)
	})

	t.Run("parameter in function argument", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT UPPER($1) FROM users`)

		require.Len(t, analysis.ParameterReferences, 1)
		assert.Equal(t, querier_dto.ParameterContextFunctionArgument, analysis.ParameterReferences[0].Context)
	})

	t.Run("comparison context for parameter", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE id = $1`)

		require.Len(t, analysis.ParameterReferences, 1)
		assert.Equal(t, querier_dto.ParameterContextComparison, analysis.ParameterReferences[0].Context)
		require.NotNil(t, analysis.ParameterReferences[0].ColumnReference)
		assert.Equal(t, "id", analysis.ParameterReferences[0].ColumnReference.ColumnName)
	})

	t.Run("boolean literal in expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE active = true`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("GROUP BY with table-qualified column", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT u.name FROM users u GROUP BY u.name`)

		require.Len(t, analysis.GroupByColumns, 1)
		assert.Equal(t, "u", analysis.GroupByColumns[0].TableAlias)
		assert.Equal(t, "name", analysis.GroupByColumns[0].ColumnName)
	})

	t.Run("CASE with simple form", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT CASE id WHEN 1 THEN 'one' WHEN 2 THEN 'two' ELSE 'other' END FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("window function with ROWS BETWEEN frame", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, SUM(id) OVER (ORDER BY id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) FROM users`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("OFFSET before LIMIT", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users OFFSET $1 ROWS LIMIT $2`)

		require.Len(t, analysis.ParameterReferences, 2)
		assert.Equal(t, querier_dto.ParameterContextOffset, analysis.ParameterReferences[0].Context)
		assert.Equal(t, querier_dto.ParameterContextLimit, analysis.ParameterReferences[1].Context)
	})

	t.Run("FETCH FIRST syntax", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users FETCH FIRST 5 ROWS ONLY`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("COALESCE with parameter inherits column reference", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT COALESCE(email, $1) FROM users`)

		require.Len(t, analysis.ParameterReferences, 1)
		require.NotNil(t, analysis.ParameterReferences[0].ColumnReference)
		assert.Equal(t, "email", analysis.ParameterReferences[0].ColumnReference.ColumnName)
	})

	t.Run("multiple CTEs", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `
			WITH active AS (SELECT id, name FROM users WHERE active = true),
			     totals AS (SELECT user_id, SUM(total) AS total FROM orders GROUP BY user_id)
			SELECT a.name, t.total FROM active a JOIN totals t ON a.id = t.user_id`)

		require.Len(t, analysis.CTEDefinitions, 2)
		assert.Equal(t, "active", analysis.CTEDefinitions[0].Name)
		assert.Equal(t, "totals", analysis.CTEDefinitions[1].Name)
	})

	t.Run("NOT IN subquery", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE id NOT IN (SELECT user_id FROM orders)`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("DISTINCT ON clause", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT DISTINCT ON (user_id) user_id, total FROM orders ORDER BY user_id, total DESC`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("interval literal in expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM orders WHERE created_at > created_at - INTERVAL '30' DAY`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("ARRAY expression in SELECT", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT ARRAY[1, 2, 3]`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("NOT LIKE expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE name NOT LIKE '%test%'`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("NOT BETWEEN expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE age NOT BETWEEN 10 AND 20`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("IS DISTINCT FROM", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE email IS DISTINCT FROM 'test@example.com'`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("concatenation operator", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT name || ' <' || email || '>' FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("nested function calls", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT LOWER(TRIM(name)) FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("arithmetic in output", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, total * 1.1 AS with_tax FROM orders`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.Equal(t, "with_tax", analysis.OutputColumns[1].Name)
	})

	t.Run("unary minus in expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT -total FROM orders`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("subquery in WHERE with IN", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE id IN (SELECT user_id FROM orders WHERE total > 100)`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("multiple aggregate functions", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT COUNT(*), SUM(total), AVG(total), MIN(total), MAX(total) FROM orders`)

		require.Len(t, analysis.OutputColumns, 5)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("table-valued function in FROM", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT * FROM generate_series(1, 10) AS gs`)

		require.NotEmpty(t, analysis.RawTableValuedFunctions)
		assert.Equal(t, "generate_series", analysis.RawTableValuedFunctions[0].FunctionName)
	})

	t.Run("USING clause in JOIN", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT u.name FROM users u JOIN orders o USING (id)`)

		require.NotEmpty(t, analysis.JoinClauses)
	})

	t.Run("NULL literal in expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT COALESCE(email, NULL, 'none') FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("CURRENT_TIMESTAMP implicit function", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT CURRENT_TIMESTAMP`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("inline cast with :: syntax", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id::VARCHAR FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})
}

func TestAnalyseQuery_Insert(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("simple INSERT with VALUES", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users (name, email) VALUES ($1, $2)`)

		assert.Equal(t, "users", analysis.InsertTable)
		require.Len(t, analysis.InsertColumns, 2)
		assert.Equal(t, "name", analysis.InsertColumns[0])
		assert.Equal(t, "email", analysis.InsertColumns[1])
		require.Len(t, analysis.ParameterReferences, 2)
		assert.Equal(t, querier_dto.ParameterContextAssignment, analysis.ParameterReferences[0].Context)
	})

	t.Run("INSERT without column list", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users VALUES ($1, $2, $3, $4, $5)`)

		assert.Equal(t, "users", analysis.InsertTable)
		assert.Empty(t, analysis.InsertColumns, "no explicit column list should be empty")
	})

	t.Run("INSERT with RETURNING", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name`)

		assert.True(t, analysis.HasReturning)
		require.Len(t, analysis.OutputColumns, 2)
		assert.Equal(t, "id", analysis.OutputColumns[0].Name)
		assert.Equal(t, "name", analysis.OutputColumns[1].Name)
	})

	t.Run("INSERT with RETURNING star", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users (name) VALUES ($1) RETURNING *`)

		assert.True(t, analysis.HasReturning)
		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.OutputColumns[0].IsStar)
	})

	t.Run("INSERT INTO schema-qualified table", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO main.users (name) VALUES ($1)`)

		assert.Equal(t, "users", analysis.InsertTable)
		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "main", analysis.FromTables[0].Schema)
	})

	t.Run("INSERT with ON CONFLICT DO NOTHING", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users (name, email) VALUES ($1, $2) ON CONFLICT (email) DO NOTHING`)

		assert.Equal(t, "users", analysis.InsertTable)
		require.Len(t, analysis.ParameterReferences, 2)
	})

	t.Run("INSERT with ON CONFLICT DO UPDATE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users (name, email) VALUES ($1, $2) ON CONFLICT (email) DO UPDATE SET name = $3`)

		assert.Equal(t, "users", analysis.InsertTable)
		require.Len(t, analysis.ParameterReferences, 3)
	})

	t.Run("INSERT with SELECT source", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO orders (user_id, total) SELECT id, 0 FROM users WHERE active = true`)

		assert.Equal(t, "orders", analysis.InsertTable)
		require.Len(t, analysis.InsertColumns, 2)
		assert.Equal(t, "user_id", analysis.InsertColumns[0])
		assert.Equal(t, "total", analysis.InsertColumns[1])
	})

	t.Run("INSERT with CTE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `WITH new_data AS (SELECT id FROM users WHERE active = true) INSERT INTO orders (user_id) SELECT id FROM new_data`)

		assert.Equal(t, "orders", analysis.InsertTable)
		require.Len(t, analysis.CTEDefinitions, 1)
		assert.Equal(t, "new_data", analysis.CTEDefinitions[0].Name)
	})

	t.Run("INSERT with DEFAULT VALUES", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users DEFAULT VALUES RETURNING id`)

		assert.Equal(t, "users", analysis.InsertTable)
		assert.True(t, analysis.HasReturning)
	})
}

func TestAnalyseQuery_Update(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("simple UPDATE SET WHERE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `UPDATE users SET name = $1 WHERE id = $2`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
		require.Len(t, analysis.ParameterReferences, 2)
		assert.Equal(t, querier_dto.ParameterContextAssignment, analysis.ParameterReferences[0].Context)
	})

	t.Run("UPDATE multiple columns", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `UPDATE users SET name = $1, email = $2 WHERE id = $3`)

		require.Len(t, analysis.ParameterReferences, 3)
	})

	t.Run("UPDATE with RETURNING", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `UPDATE users SET name = $1 WHERE id = $2 RETURNING id, name`)

		assert.True(t, analysis.HasReturning)
		require.Len(t, analysis.OutputColumns, 2)
		assert.Equal(t, "id", analysis.OutputColumns[0].Name)
		assert.Equal(t, "name", analysis.OutputColumns[1].Name)
	})

	t.Run("UPDATE with alias", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `UPDATE users AS u SET name = $1 WHERE u.id = $2`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
		assert.Equal(t, "u", analysis.FromTables[0].Alias)
	})

	t.Run("UPDATE with FROM clause", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `UPDATE orders SET total = $1 FROM users WHERE orders.user_id = users.id AND users.name = $2`)

		require.Len(t, analysis.FromTables, 2)
		assert.Equal(t, "orders", analysis.FromTables[0].Name)
		assert.Equal(t, "users", analysis.FromTables[1].Name)
	})

	t.Run("UPDATE schema-qualified table", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `UPDATE main.users SET active = true WHERE id = $1`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "main", analysis.FromTables[0].Schema)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
	})

	t.Run("UPDATE with complex SET expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `UPDATE users SET name = UPPER(name) WHERE id = $1`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
		require.Len(t, analysis.ParameterReferences, 1)
	})

	t.Run("UPDATE with CTE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `WITH old AS (SELECT id FROM users WHERE name = $1) UPDATE users SET name = $2 WHERE id IN (SELECT id FROM old)`)

		require.Len(t, analysis.CTEDefinitions, 1)
		assert.Equal(t, "old", analysis.CTEDefinitions[0].Name)
	})
}

func TestAnalyseQuery_Delete(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("simple DELETE WHERE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `DELETE FROM users WHERE id = $1`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
		require.Len(t, analysis.ParameterReferences, 1)
		assert.Equal(t, 1, analysis.ParameterReferences[0].Number)
	})

	t.Run("DELETE with RETURNING", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `DELETE FROM users WHERE id = $1 RETURNING id, name`)

		assert.True(t, analysis.HasReturning)
		require.Len(t, analysis.OutputColumns, 2)
		assert.Equal(t, "id", analysis.OutputColumns[0].Name)
		assert.Equal(t, "name", analysis.OutputColumns[1].Name)
	})

	t.Run("DELETE with RETURNING star", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `DELETE FROM users WHERE id = $1 RETURNING *`)

		assert.True(t, analysis.HasReturning)
		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.OutputColumns[0].IsStar)
	})

	t.Run("DELETE with alias", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `DELETE FROM users AS u WHERE u.id = $1`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
		assert.Equal(t, "u", analysis.FromTables[0].Alias)
	})

	t.Run("DELETE with USING clause", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `DELETE FROM orders USING users WHERE orders.user_id = users.id AND users.active = false`)

		require.Len(t, analysis.FromTables, 2)
		assert.Equal(t, "orders", analysis.FromTables[0].Name)
		assert.Equal(t, "users", analysis.FromTables[1].Name)
	})

	t.Run("DELETE from schema-qualified table", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `DELETE FROM main.users WHERE id = $1`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "main", analysis.FromTables[0].Schema)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
	})

	t.Run("DELETE with multiple parameters", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `DELETE FROM users WHERE name = $1 AND age < $2`)

		require.Len(t, analysis.ParameterReferences, 2)
		assert.Equal(t, 1, analysis.ParameterReferences[0].Number)
		assert.Equal(t, 2, analysis.ParameterReferences[1].Number)
	})

	t.Run("DELETE with CTE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `WITH inactive AS (SELECT id FROM users WHERE active = false) DELETE FROM orders WHERE user_id IN (SELECT id FROM inactive)`)

		require.Len(t, analysis.CTEDefinitions, 1)
		assert.Equal(t, "inactive", analysis.CTEDefinitions[0].Name)
	})

	t.Run("DELETE with complex WHERE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `DELETE FROM users WHERE email IS NULL AND age < $1`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
		require.Len(t, analysis.ParameterReferences, 1)
	})
}

func TestAnalyseQuery_Values(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("simple VALUES statement", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `VALUES (1, 'hello'), (2, 'world')`)

		assert.True(t, analysis.ReadOnly, "VALUES should be read-only")
		require.Len(t, analysis.OutputColumns, 2)
	})

	t.Run("VALUES with parameters", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `VALUES ($1, $2)`)

		require.Len(t, analysis.ParameterReferences, 2)
	})
}

func TestAnalyseQuery_SelectExpressions(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("SIMILAR TO expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE name SIMILAR TO '%pattern%'`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("SIMILAR TO with ESCAPE clause", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE name SIMILAR TO '%\%%' ESCAPE '\'`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("LIKE with ESCAPE clause", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE name LIKE '%\%%' ESCAPE '\'`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("bitwise operators", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id & 0xFF, id | 0x01, id << 2, id >> 1 FROM users`)

		require.Len(t, analysis.OutputColumns, 4)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("JSON arrow operators", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT '{"a": 1}'::json -> 'a', '{"b": 2}'::json ->> 'b'`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("ROW constructor expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT ROW(1, 'hello', true)`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("subscript expression with array access", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT ARRAY[1, 2, 3][1]`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("subscript slice expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT ARRAY[1, 2, 3][1:2]`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("comparison with ANY subquery", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE id = ANY (SELECT user_id FROM orders)`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("OR expression in WHERE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE name = 'alice' OR name = 'bob'`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("NOT expression in WHERE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE NOT active`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("inline cast with parameter", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE id = $1::int`)

		require.Len(t, analysis.ParameterReferences, 1)

		require.NotNil(t, analysis.ParameterReferences[0].CastType)
	})

	t.Run("IS TRUE and IS FALSE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE active IS TRUE`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("IS NOT DISTINCT FROM", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE email IS NOT DISTINCT FROM 'test@example.com'`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("schema-qualified function call", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT main.my_func(id) FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("schema-qualified column reference", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT main.users.id FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("NOT EXISTS subquery", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users u WHERE NOT EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("ARRAY subquery expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT ARRAY(SELECT id FROM users)`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("BETWEEN with parameters and column reference", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE age BETWEEN $1 AND $2`)

		require.Len(t, analysis.ParameterReferences, 2)
		assert.Equal(t, 1, analysis.ParameterReferences[0].Number)
		assert.Equal(t, 2, analysis.ParameterReferences[1].Number)
	})

	t.Run("IN list with literal values", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE id IN (1, 2, 3)`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("QUALIFY clause", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, ROW_NUMBER() OVER (ORDER BY id) AS rn FROM users QUALIFY rn = 1`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})
}

func TestAnalyseQuery_SelectFunctions(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("lambda expression single parameter", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT list_transform([1, 2, 3], x -> x + 1)`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("multi-parameter lambda expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT list_reduce([1, 2, 3], (x, y) -> x + y)`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("WITHIN GROUP aggregate", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT mode() WITHIN GROUP (ORDER BY age) FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("FILTER clause on aggregate", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT COUNT(*) FILTER (WHERE active) FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("function with ORDER BY in arguments", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT string_agg(name, ', ' ORDER BY name ASC) FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("window function with named window reference", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, SUM(id) OVER w FROM users WINDOW w AS (ORDER BY id)`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("window function with RANGE frame", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, AVG(age) OVER (ORDER BY id RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) FROM users`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("window function with GROUPS frame", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, SUM(id) OVER (ORDER BY id GROUPS BETWEEN 1 PRECEDING AND 1 FOLLOWING) FROM users`)

		require.Len(t, analysis.OutputColumns, 2)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("CAST with array type", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT CAST($1 AS INTEGER[]) FROM users`)

		require.Len(t, analysis.ParameterReferences, 1)
		assert.Equal(t, querier_dto.ParameterContextCast, analysis.ParameterReferences[0].Context)
	})
}

func TestAnalyseQuery_SelectClauses(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("FOR UPDATE clause", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users FOR UPDATE`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.False(t, analysis.ReadOnly, "FOR UPDATE should make query non-read-only")
	})

	t.Run("FOR UPDATE OF table NOWAIT", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users FOR UPDATE OF users NOWAIT`)

		assert.False(t, analysis.ReadOnly)
	})

	t.Run("FOR NO KEY UPDATE SKIP LOCKED", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users FOR NO KEY UPDATE SKIP LOCKED`)

		assert.False(t, analysis.ReadOnly)
	})

	t.Run("FOR SHARE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users FOR SHARE`)

		assert.False(t, analysis.ReadOnly)
	})

	t.Run("FETCH NEXT syntax", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users FETCH NEXT 5 ROWS ONLY`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("FETCH FIRST with TIES", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users ORDER BY id FETCH FIRST 5 ROWS WITH TIES`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("OFFSET after FETCH", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users OFFSET 5 ROWS FETCH FIRST 10 ROWS ONLY`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("GROUP BY with expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT LOWER(name), count(*) FROM users GROUP BY LOWER(name)`)

		require.Len(t, analysis.GroupByColumns, 1)
	})

	t.Run("CTE with column names", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `WITH active (uid, username) AS (
			SELECT id, name FROM users WHERE active = true
		)
		SELECT uid, username FROM active`)

		require.Len(t, analysis.CTEDefinitions, 1)
		assert.Equal(t, "active", analysis.CTEDefinitions[0].Name)
		require.Len(t, analysis.CTEDefinitions[0].OutputColumns, 2)
		assert.Equal(t, "uid", analysis.CTEDefinitions[0].OutputColumns[0].Name)
		assert.Equal(t, "username", analysis.CTEDefinitions[0].OutputColumns[1].Name)
	})

	t.Run("RECURSIVE CTE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `WITH RECURSIVE nums AS (
			SELECT 1 AS n
			UNION ALL
			SELECT n + 1 FROM nums WHERE n < 10
		)
		SELECT n FROM nums`)

		require.Len(t, analysis.CTEDefinitions, 1)
		assert.True(t, analysis.CTEDefinitions[0].IsRecursive)
	})

	t.Run("MATERIALIZED CTE hint", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `WITH active AS MATERIALIZED (
			SELECT id FROM users WHERE active = true
		)
		SELECT id FROM active`)

		require.Len(t, analysis.CTEDefinitions, 1)
	})

	t.Run("NOT MATERIALIZED CTE hint", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `WITH active AS NOT MATERIALIZED (
			SELECT id FROM users WHERE active = true
		)
		SELECT id FROM active`)

		require.Len(t, analysis.CTEDefinitions, 1)
	})

	t.Run("TVF with column definitions", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT * FROM read_csv('data.csv') AS t(id INTEGER, name VARCHAR)`)

		require.NotEmpty(t, analysis.RawTableValuedFunctions)
		assert.Equal(t, "read_csv", analysis.RawTableValuedFunctions[0].FunctionName)
		assert.Equal(t, "t", analysis.RawTableValuedFunctions[0].Alias)
		require.NotEmpty(t, analysis.RawTableValuedFunctions[0].ColumnDefinitions)
	})

	t.Run("PIVOT clause", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT p.* FROM (SELECT id, active FROM users) PIVOT (COUNT(id) FOR active IN (true, false)) AS p`)

		require.NotEmpty(t, analysis.RawDerivedTables)
	})

	t.Run("LATERAL join", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT u.name, gs.val FROM users u, LATERAL generate_series(1, u.id) AS gs(val)`)

		require.NotEmpty(t, analysis.RawTableValuedFunctions)
	})
}

func TestAnalyseQuery_Update_AliasAndMultiColumnSet(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("UPDATE with multi-column SET", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `UPDATE users SET (name, email) = ($1, $2) WHERE id = $3`)

		require.Len(t, analysis.ParameterReferences, 3)
	})

	t.Run("UPDATE with alias without AS keyword", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `UPDATE users u SET name = $1 WHERE u.id = $2`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "u", analysis.FromTables[0].Alias)
	})

	t.Run("UPDATE with WHERE CURRENT OF", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `UPDATE users SET name = 'test' WHERE CURRENT OF cursor_name`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
	})

	t.Run("DELETE with alias without AS keyword", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `DELETE FROM users u WHERE u.id = $1`)

		require.Len(t, analysis.FromTables, 1)
		assert.Equal(t, "u", analysis.FromTables[0].Alias)
	})
}

func TestAnalyseQuery_Insert_ConflictAndOverriding(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("INSERT with OVERRIDING SYSTEM VALUE", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users (id, name) OVERRIDING SYSTEM VALUE VALUES ($1, $2)`)

		assert.Equal(t, "users", analysis.InsertTable)
		require.Len(t, analysis.ParameterReferences, 2)
	})

	t.Run("INSERT with alias AS", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users AS u (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name`)

		assert.Equal(t, "users", analysis.InsertTable)
	})

	t.Run("INSERT with ON CONFLICT ON CONSTRAINT", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users (name) VALUES ($1) ON CONFLICT ON CONSTRAINT users_pkey DO NOTHING`)

		assert.Equal(t, "users", analysis.InsertTable)
	})

	t.Run("INSERT VALUES with non-parameter literal expressions", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users (name, email) VALUES ('alice', 'alice@example.com')`)

		assert.Equal(t, "users", analysis.InsertTable)
		require.Len(t, analysis.InsertColumns, 2)
	})

	t.Run("INSERT VALUES with subexpression in parentheses", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users (name) VALUES ((SELECT 'alice'))`)

		assert.Equal(t, "users", analysis.InsertTable)
	})

	t.Run("INSERT with multiple value rows", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `INSERT INTO users (name, email) VALUES ($1, $2), ($3, $4)`)

		assert.Equal(t, "users", analysis.InsertTable)
		require.Len(t, analysis.ParameterReferences, 4)
	})
}

func TestAnalyseQuery_Values_OrderByAndLimit(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("VALUES with ORDER BY and LIMIT", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `VALUES (1, 'a'), (2, 'b') ORDER BY 1 LIMIT 1`)

		assert.True(t, analysis.ReadOnly)
		require.Len(t, analysis.OutputColumns, 2)
	})
}

func TestAnalyseQuery_NamedParameters(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("named parameter with colon prefix", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE name = :name`)

		require.Len(t, analysis.ParameterReferences, 1)
		assert.Equal(t, "name", analysis.ParameterReferences[0].Name)
		assert.Equal(t, 1, analysis.ParameterReferences[0].Number)
	})

	t.Run("repeated named parameter", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users WHERE name = :name OR email = :name`)

		require.Len(t, analysis.ParameterReferences, 2)
		assert.Equal(t, analysis.ParameterReferences[0].Number, analysis.ParameterReferences[1].Number,
			"repeated named parameter should share the same number")
	})
}

func TestAnalyseQuery_CTEWithDML(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("CTE with INSERT body", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `WITH inserted AS (
			INSERT INTO orders (user_id, total) VALUES ($1, $2) RETURNING id
		)
		SELECT id FROM inserted`)

		require.Len(t, analysis.CTEDefinitions, 1)
		assert.False(t, analysis.ReadOnly, "data-modifying CTE should not be read-only")
	})

	t.Run("CTE with UPDATE body", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `WITH updated AS (
			UPDATE users SET name = 'new' WHERE id = $1 RETURNING id
		)
		SELECT id FROM updated`)

		require.Len(t, analysis.CTEDefinitions, 1)
		assert.False(t, analysis.ReadOnly)
	})

	t.Run("CTE with DELETE body", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `WITH deleted AS (
			DELETE FROM users WHERE active = false RETURNING id
		)
		SELECT id FROM deleted`)

		require.Len(t, analysis.CTEDefinitions, 1)
		assert.False(t, analysis.ReadOnly)
	})
}

func TestAnalyseQuery_Select_OutputExpressionOperators(t *testing.T) {
	t.Parallel()

	catalogue := buildTestCatalogue()

	t.Run("LIKE in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT name LIKE '%test%' AS is_test FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.Equal(t, "is_test", analysis.OutputColumns[0].Name)
	})

	t.Run("ILIKE in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT name ILIKE '%test%' AS matches FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("SIMILAR TO in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT name SIMILAR TO '%(test|demo)%' AS matches FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("NOT LIKE in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT name NOT LIKE '%test%' AS not_test FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("NOT BETWEEN in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT age NOT BETWEEN 10 AND 20 AS outside_range FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("BETWEEN in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT age BETWEEN 10 AND 20 AS in_range FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("IN list in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id IN (1, 2, 3) AS in_list FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("IS NULL in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT email IS NULL AS missing_email FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("IS NOT NULL in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT email IS NOT NULL AS has_email FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("IS DISTINCT FROM in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT email IS DISTINCT FROM 'test@example.com' AS different FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("comparison operators in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id > 5 AS big, id <= 10 AS small, id <> 0 AS nonzero FROM users`)

		require.Len(t, analysis.OutputColumns, 3)
	})

	t.Run("OR and AND in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT (active AND age > 18) OR (name = 'admin') AS eligible FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("NOT in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT NOT active AS inactive FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("INTERVAL literal in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT INTERVAL '30' DAY AS one_month`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("INTERVAL with YEAR TO MONTH", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT INTERVAL '1' YEAR TO MONTH AS period`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("EXISTS subquery in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id) AS has_orders FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("parenthesised expression in output", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT (id + 1) * 2 AS doubled FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("NOT unary in cast expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT NOT true AS result`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("tilde unary operator", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT ~id AS inverted FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("inline cast in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id::varchar AS id_text FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("cast with type modifiers", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT CAST(name AS VARCHAR(100)) FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("schema-qualified function call in output", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT main.lower(name) FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("three-part schema.table.column reference", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT main.users.name FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("schema-qualified function with three parts", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT main.pg_catalog.now()`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("BETWEEN SYMMETRIC in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT age BETWEEN SYMMETRIC 20 AND 10 AS in_range FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("IN list with subquery in output", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id IN (SELECT user_id FROM orders) AS has_orders FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("NOT IN list in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id NOT IN (1, 2, 3) AS not_in_list FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("LIKE with ESCAPE in output", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT name LIKE '%\_%' ESCAPE '\' AS has_underscore FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("SIMILAR TO with ESCAPE in output", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT name SIMILAR TO '%\_%' ESCAPE '\' AS matches FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("IS TRUE in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT active IS TRUE AS is_active FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("IS NOT TRUE in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT active IS NOT TRUE AS not_active FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("IS UNKNOWN in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT active IS UNKNOWN AS is_unknown FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("comparison with ALL subquery", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id > ALL (SELECT user_id FROM orders) AS bigger_than_all FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("CAST target type with multi-word type", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT CAST(id AS DOUBLE PRECISION) FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("CAST target type with array brackets", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT CAST(ARRAY[1,2] AS INTEGER[])`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("LIMIT ALL syntax", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users LIMIT ALL`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("OFFSET ROWS FETCH", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id FROM users OFFSET 10 ROWS FETCH FIRST 5 ROWS ONLY`)

		require.Len(t, analysis.OutputColumns, 1)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("window function with EXCLUDE CURRENT ROW", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, SUM(age) OVER (ORDER BY id ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING EXCLUDE CURRENT ROW) FROM users`)

		require.Len(t, analysis.OutputColumns, 2)
	})

	t.Run("window function with numeric frame bound", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT id, AVG(age) OVER (ORDER BY id ROWS BETWEEN 2 PRECEDING AND 2 FOLLOWING) FROM users`)

		require.Len(t, analysis.OutputColumns, 2)
	})

	t.Run("function with DISTINCT keyword", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT COUNT(DISTINCT name) FROM users`)

		require.Len(t, analysis.OutputColumns, 1)
	})

	t.Run("parameter in CAST in output expression", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT CAST($1 AS VARCHAR) AS value`)

		require.Len(t, analysis.ParameterReferences, 1)
		assert.Equal(t, querier_dto.ParameterContextCast, analysis.ParameterReferences[0].Context)
	})

	t.Run("parameter in inline cast in output", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, catalogue, `SELECT $1::INTEGER AS value`)

		require.Len(t, analysis.ParameterReferences, 1)
		assert.Equal(t, querier_dto.ParameterContextCast, analysis.ParameterReferences[0].Context)
	})
}

func TestParseStatements_Multiple(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()
	stmts, err := engine.ParseStatements(`SELECT 1; SELECT 2`)
	require.NoError(t, err)
	require.Len(t, stmts, 2, "should split multiple statements on semicolons")
}
