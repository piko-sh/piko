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

func newSQLiteCatalogue() *querier_dto.Catalogue {
	return &querier_dto.Catalogue{
		DefaultSchema: "main",
		Schemas: map[string]*querier_dto.Schema{
			"main": {
				Name: "main",
				Tables: map[string]*querier_dto.Table{
					"users": {
						Name: "users",
						Columns: []querier_dto.Column{
							{
								Name:    "id",
								SQLType: querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger},
							},
							{
								Name:    "name",
								SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
							},
							{
								Name:     "email",
								SQLType:  querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
								Nullable: true,
							},
						},
					},
					"posts": {
						Name: "posts",
						Columns: []querier_dto.Column{
							{
								Name:    "id",
								SQLType: querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger},
							},
							{
								Name:    "user_id",
								SQLType: querier_dto.SQLType{EngineName: "integer", Category: querier_dto.TypeCategoryInteger},
							},
							{
								Name:    "title",
								SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
							},
							{
								Name:     "body",
								SQLType:  querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
								Nullable: true,
							},
						},
					},
				},
				Views:          map[string]*querier_dto.View{},
				Functions:      map[string][]*querier_dto.FunctionSignature{},
				Enums:          map[string]*querier_dto.Enum{},
				CompositeTypes: map[string]*querier_dto.CompositeType{},
				Sequences:      map[string]*querier_dto.Sequence{},
			},
		},
		Extensions: map[string]struct{}{},
	}
}

func analyseQuery(t *testing.T, catalogue *querier_dto.Catalogue, sql string) *querier_dto.RawQueryAnalysis {
	t.Helper()

	engine := NewSQLiteEngine()
	stmts, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.NotEmpty(t, stmts)

	analysis, err := engine.AnalyseQuery(catalogue, stmts[0])
	require.NoError(t, err)

	return analysis
}

func TestAnalyseQuery_Select(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple column list",
			sql:  "SELECT id, name FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)

				assert.Equal(t, "id", a.OutputColumns[0].Name)
				assert.Equal(t, "id", a.OutputColumns[0].ColumnName)
				assert.Equal(t, "name", a.OutputColumns[1].Name)
				assert.Equal(t, "name", a.OutputColumns[1].ColumnName)

				require.Len(t, a.FromTables, 1)
				assert.Equal(t, "users", a.FromTables[0].Name)
				assert.True(t, a.ReadOnly, "SELECT should be read-only")
			},
		},
		{
			name: "star expansion",
			sql:  "SELECT * FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.True(t, a.OutputColumns[0].IsStar, "should be a star column")
			},
		},
		{
			name: "WHERE with numbered parameter ?1",
			sql:  "SELECT id FROM users WHERE id = ?1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "id", a.OutputColumns[0].Name)

				require.NotEmpty(t, a.ParameterReferences, "should have at least one parameter reference")
				assert.Equal(t, 1, a.ParameterReferences[0].Number)
			},
		},
		{
			name: "WHERE with named parameter :email",
			sql:  "SELECT id FROM users WHERE email = :email",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences, "should have at least one parameter reference")
				assert.Equal(t, "email", a.ParameterReferences[0].Name)
			},
		},
		{
			name: "JOIN across two tables",
			sql:  "SELECT u.id, p.title FROM users u JOIN posts p ON p.user_id = u.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)

				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)
				assert.Equal(t, "u", a.FromTables[0].Alias)

				require.NotEmpty(t, a.JoinClauses)
				assert.Equal(t, "posts", a.JoinClauses[0].Table.Name)
				assert.Equal(t, "p", a.JoinClauses[0].Table.Alias)
				assert.Equal(t, querier_dto.JoinInner, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "LEFT JOIN",
			sql:  "SELECT u.id, p.title FROM users u LEFT JOIN posts p ON p.user_id = u.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.JoinClauses)
				assert.Equal(t, querier_dto.JoinLeft, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "aggregate function COUNT",
			sql:  "SELECT COUNT(*) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "count", a.OutputColumns[0].Name)
				assert.True(t, a.ReadOnly, "SELECT with aggregate should be read-only")
			},
		},
		{
			name: "ORDER BY and LIMIT",
			sql:  "SELECT id, name FROM users ORDER BY name LIMIT 10",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
				require.Len(t, a.FromTables, 1)
				assert.Equal(t, "users", a.FromTables[0].Name)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "qualified star expansion",
			sql:  "SELECT u.* FROM users u",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.True(t, a.OutputColumns[0].IsStar, "should be a star column")
				assert.Equal(t, "u", a.OutputColumns[0].TableAlias, "star should be qualified with table alias")
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Select_CTEsAndCompoundQueries(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "CTE with SELECT",
			sql:  "WITH active AS (SELECT id, name FROM users WHERE name IS NOT NULL) SELECT id, name FROM active",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.CTEDefinitions, 1)
				assert.Equal(t, "active", a.CTEDefinitions[0].Name)
				require.Len(t, a.CTEDefinitions[0].OutputColumns, 2)
				assert.Equal(t, "id", a.CTEDefinitions[0].OutputColumns[0].Name)
				assert.Equal(t, "name", a.CTEDefinitions[0].OutputColumns[1].Name)
			},
		},
		{
			name: "CTE with explicit column names",
			sql:  "WITH cte (user_id, user_name) AS (SELECT id, name FROM users) SELECT user_id, user_name FROM cte",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.CTEDefinitions, 1)
				assert.Equal(t, "cte", a.CTEDefinitions[0].Name)
				require.Len(t, a.CTEDefinitions[0].OutputColumns, 2)
				assert.Equal(t, "user_id", a.CTEDefinitions[0].OutputColumns[0].Name)
				assert.Equal(t, "user_name", a.CTEDefinitions[0].OutputColumns[1].Name)
			},
		},
		{
			name: "multiple CTEs",
			sql:  "WITH a AS (SELECT id FROM users), b AS (SELECT user_id FROM posts) SELECT a.id FROM a JOIN b ON a.id = b.user_id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.CTEDefinitions, 2)
				assert.Equal(t, "a", a.CTEDefinitions[0].Name)
				assert.Equal(t, "b", a.CTEDefinitions[1].Name)
			},
		},
		{
			name: "UNION compound query",
			sql:  "SELECT id, name FROM users WHERE id < 5 UNION SELECT id, name FROM users WHERE id >= 5",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.CompoundBranches, 1)
				assert.Equal(t, querier_dto.CompoundUnion, a.CompoundBranches[0].Operator)
				require.NotNil(t, a.CompoundBranches[0].Query)
			},
		},
		{
			name: "INTERSECT compound query",
			sql:  "SELECT id FROM users INTERSECT SELECT user_id FROM posts",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.CompoundBranches, 1)
				assert.Equal(t, querier_dto.CompoundIntersect, a.CompoundBranches[0].Operator)
			},
		},
		{
			name: "EXCEPT compound query",
			sql:  "SELECT id FROM users EXCEPT SELECT user_id FROM posts",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.CompoundBranches, 1)
				assert.Equal(t, querier_dto.CompoundExcept, a.CompoundBranches[0].Operator)
			},
		},
		{
			name: "subquery in FROM",
			sql:  "SELECT s.id FROM (SELECT id FROM users) s",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.RawDerivedTables)
				assert.Equal(t, "s", a.RawDerivedTables[0].Alias)
			},
		},
		{
			name: "GROUP BY and HAVING",
			sql:  "SELECT user_id, COUNT(*) FROM posts GROUP BY user_id HAVING COUNT(*) > 1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.GroupByColumns, 1)
				assert.Equal(t, "user_id", a.GroupByColumns[0].ColumnName)
			},
		},
		{
			name: "GROUP BY multiple columns",
			sql:  "SELECT user_id, name FROM users GROUP BY user_id, name",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.GroupByColumns, 2)
				assert.Equal(t, "user_id", a.GroupByColumns[0].ColumnName)
				assert.Equal(t, "name", a.GroupByColumns[1].ColumnName)
			},
		},
		{
			name: "DISTINCT modifier",
			sql:  "SELECT DISTINCT name FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "name", a.OutputColumns[0].Name)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "window function with OVER",
			sql:  "SELECT id, ROW_NUMBER() OVER (ORDER BY id) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "window function with PARTITION BY",
			sql:  "SELECT user_id, title, ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY title) FROM posts",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 3)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "CASE expression in SELECT",
			sql:  "SELECT id, CASE WHEN name IS NOT NULL THEN name ELSE 'unknown' END FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "COALESCE expression",
			sql:  "SELECT COALESCE(email, 'none') FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "CAST expression",
			sql:  "SELECT CAST(id AS TEXT) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "IS NULL condition",
			sql:  "SELECT id FROM users WHERE email IS NULL",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "IS NOT NULL condition",
			sql:  "SELECT id FROM users WHERE email IS NOT NULL",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "BETWEEN with parameters in WHERE",
			sql:  "SELECT id FROM users WHERE id BETWEEN ? AND ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)

				assert.Equal(t, 1, a.ParameterReferences[0].Number)
				assert.Equal(t, 2, a.ParameterReferences[1].Number)
			},
		},
		{
			name: "IN list with parameters",
			sql:  "SELECT id FROM users WHERE id IN (?1, ?2, ?3)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 3)
				assert.Equal(t, querier_dto.ParameterContextInList, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "LIKE expression",
			sql:  "SELECT id, name FROM users WHERE name LIKE '%test%'",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "ORDER BY with DESC",
			sql:  "SELECT id, name FROM users ORDER BY name DESC",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "LIMIT with OFFSET parameter",
			sql:  "SELECT id FROM users LIMIT 10 OFFSET ?1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextOffset, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "LIMIT with parameter",
			sql:  "SELECT id FROM users LIMIT ?1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextLimit, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "multiple JOINs",
			sql:  "SELECT u.name, p.title FROM users u JOIN posts p ON p.user_id = u.id LEFT JOIN posts p2 ON p2.user_id = u.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.JoinClauses)
				assert.Equal(t, querier_dto.JoinInner, a.JoinClauses[0].Kind)
				assert.Equal(t, "posts", a.JoinClauses[0].Table.Name)
			},
		},
		{
			name: "CROSS JOIN",
			sql:  "SELECT u.id, p.id FROM users u CROSS JOIN posts p",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinCross, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "correlated scalar subquery in SELECT",
			sql:  "SELECT id, (SELECT COUNT(*) FROM posts p WHERE p.user_id = u.id) FROM users u",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "EXISTS subquery in WHERE",
			sql:  "SELECT id FROM users u WHERE EXISTS (SELECT 1 FROM posts p WHERE p.user_id = u.id)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "comma-joined tables",
			sql:  "SELECT u.id, p.id FROM users u, posts p WHERE u.id = p.user_id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.FromTables, 2)
				assert.Equal(t, "users", a.FromTables[0].Name)
				assert.Equal(t, "posts", a.FromTables[1].Name)
			},
		},
		{
			name: "column alias with AS",
			sql:  "SELECT name AS user_name FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "user_name", a.OutputColumns[0].Name)
			},
		},
		{
			name: "function call in output",
			sql:  "SELECT MAX(id) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "max", a.OutputColumns[0].Name)
			},
		},
		{
			name: "CAST parameter context detection",
			sql:  "SELECT CAST(? AS INTEGER) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextCast, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].CastType)
			},
		},
		{
			name: "parameter in function argument",
			sql:  "SELECT UPPER(?) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextFunctionArgument, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "comparison context for parameter",
			sql:  "SELECT id FROM users WHERE id = ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextComparison, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "id", a.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
		{
			name: "UNION ALL compound query",
			sql:  "SELECT id FROM users UNION ALL SELECT id FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.CompoundBranches, 1)
				assert.Equal(t, querier_dto.CompoundUnionAll, a.CompoundBranches[0].Operator)
			},
		},
		{
			name: "RIGHT JOIN",
			sql:  "SELECT u.id, p.title FROM users u RIGHT JOIN posts p ON p.user_id = u.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinRight, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "FULL OUTER JOIN",
			sql:  "SELECT u.id, p.title FROM users u FULL OUTER JOIN posts p ON p.user_id = u.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinFull, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "LIMIT comma OFFSET syntax",
			sql:  "SELECT id FROM users LIMIT 10, 20",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "CASE with simple form",
			sql:  "SELECT CASE id WHEN 1 THEN 'one' WHEN 2 THEN 'two' ELSE 'other' END FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "boolean literals in expression",
			sql:  "SELECT id FROM users WHERE TRUE AND NOT FALSE",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "GROUP BY with table-qualified column",
			sql:  "SELECT u.name FROM users u GROUP BY u.name",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.GroupByColumns, 1)
				assert.Equal(t, "u", a.GroupByColumns[0].TableAlias)
				assert.Equal(t, "name", a.GroupByColumns[0].ColumnName)
			},
		},
		{
			name: "window function with ROWS BETWEEN frame",
			sql:  "SELECT id, SUM(id) OVER (ORDER BY id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "string concatenation in output",
			sql:  "SELECT name || ' <' || email || '>' FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "arithmetic operations in output",
			sql:  "SELECT id, id * 2, id + 1, id - 1, id / 2, id % 3 FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 6)
			},
		},
		{
			name: "unary minus in expression",
			sql:  "SELECT -id FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "nested function calls",
			sql:  "SELECT LOWER(UPPER(name)) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "table-valued function in FROM",
			sql:  "SELECT * FROM json_each('[1,2,3]') AS j",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.RawTableValuedFunctions)
				assert.Equal(t, "json_each", a.RawTableValuedFunctions[0].FunctionName)
			},
		},
		{
			name: "OR logical expression in WHERE",
			sql:  "SELECT id FROM users WHERE name = 'alice' OR name = 'bob'",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "NOT expression in WHERE",
			sql:  "SELECT id FROM users WHERE NOT email IS NULL",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "parenthesised expression",
			sql:  "SELECT id FROM users WHERE (id > 5) AND (name IS NOT NULL)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "number literal types",
			sql:  "SELECT 42, 3.14, 1e10 FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 3)
			},
		},
		{
			name: "GLOB expression",
			sql:  "SELECT id FROM users WHERE name GLOB 'A*'",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "IN list with static values",
			sql:  "SELECT id FROM users WHERE id IN (1, 2, 3)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "BETWEEN with static values",
			sql:  "SELECT id FROM users WHERE id BETWEEN 1 AND 10",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "FILTER clause on aggregate",
			sql:  "SELECT COUNT(*) FILTER (WHERE name IS NOT NULL) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "multiple aggregate functions",
			sql:  "SELECT COUNT(*), MAX(id), MIN(id), SUM(id) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 4)
			},
		},
		{
			name: "subquery in WHERE with IN",
			sql:  "SELECT id FROM users WHERE id IN (SELECT user_id FROM posts)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "column with table alias in output",
			sql:  "SELECT u.id, u.name FROM users u WHERE u.id > 1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
				assert.Equal(t, "u", a.OutputColumns[0].TableAlias)
			},
		},
		{
			name: "USING clause in JOIN",
			sql:  "SELECT u.name FROM users u JOIN posts p USING (id)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.JoinClauses)
			},
		},
		{
			name: "NULL expression",
			sql:  "SELECT COALESCE(email, NULL, 'none') FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "repeated named parameter",
			sql:  "SELECT id FROM users WHERE name = :name OR email = :name",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)

				assert.Equal(t, a.ParameterReferences[0].Number, a.ParameterReferences[1].Number)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Insert(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple insert with anonymous parameters",
			sql:  "INSERT INTO users (name, email) VALUES (?, ?)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)

				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)

				require.Len(t, a.ParameterReferences, 2)
				assert.Equal(t, 1, a.ParameterReferences[0].Number)
				assert.Equal(t, 2, a.ParameterReferences[1].Number)
				assert.Equal(t, querier_dto.ParameterContextAssignment, a.ParameterReferences[0].Context)
				assert.Equal(t, querier_dto.ParameterContextAssignment, a.ParameterReferences[1].Context)
			},
		},
		{
			name: "insert with RETURNING clause",
			sql:  "INSERT INTO users (name) VALUES (?) RETURNING id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.HasReturning, "should detect RETURNING clause")

				require.NotEmpty(t, a.OutputColumns)
				assert.Equal(t, "id", a.OutputColumns[0].Name)
			},
		},
		{
			name: "INSERT OR REPLACE",
			sql:  "INSERT OR REPLACE INTO users (id, name) VALUES (?1, ?2)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)

				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)

				require.Len(t, a.ParameterReferences, 2)
				assert.Equal(t, 1, a.ParameterReferences[0].Number)
				assert.Equal(t, 2, a.ParameterReferences[1].Number)
			},
		},
		{
			name: "INSERT with SELECT source",
			sql:  "INSERT INTO posts (user_id, title) SELECT id, name FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "posts", a.FromTables[0].Name)
			},
		},
		{
			name: "INSERT with ON CONFLICT DO NOTHING",
			sql:  "INSERT INTO users (id, name) VALUES (?, ?) ON CONFLICT (id) DO NOTHING",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)
			},
		},
		{
			name: "INSERT with ON CONFLICT DO UPDATE",
			sql:  "INSERT INTO users (id, name) VALUES (?, ?) ON CONFLICT (id) DO UPDATE SET name = ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 3)
			},
		},
		{
			name: "INSERT with CTE",
			sql:  "WITH new_users AS (SELECT id, name FROM users WHERE id > ?) INSERT INTO posts (user_id, title) SELECT id, name FROM new_users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.CTEDefinitions, 1)
				assert.Equal(t, "new_users", a.CTEDefinitions[0].Name)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Update(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple update with SET and WHERE",
			sql:  "UPDATE users SET name = ? WHERE id = ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)

				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)

				require.Len(t, a.ParameterReferences, 2)
				assert.Equal(t, 1, a.ParameterReferences[0].Number)
				assert.Equal(t, 2, a.ParameterReferences[1].Number)

				assert.Equal(t, querier_dto.ParameterContextAssignment, a.ParameterReferences[0].Context)

				assert.Equal(t, querier_dto.ParameterContextComparison, a.ParameterReferences[1].Context)
			},
		},
		{
			name: "update with named parameters",
			sql:  "UPDATE users SET email = :email WHERE id = :id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)

				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)

				require.Len(t, a.ParameterReferences, 2)
				assert.Equal(t, "email", a.ParameterReferences[0].Name)
				assert.Equal(t, "id", a.ParameterReferences[1].Name)
			},
		},
		{
			name: "update with RETURNING",
			sql:  "UPDATE posts SET title = ? WHERE id = ? RETURNING id, title",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)

				assert.True(t, a.HasReturning, "should detect RETURNING clause")
				require.Len(t, a.OutputColumns, 2)
				assert.Equal(t, "id", a.OutputColumns[0].Name)
				assert.Equal(t, "title", a.OutputColumns[1].Name)
			},
		},
		{
			name: "update with complex SET expression",
			sql:  "UPDATE users SET name = UPPER(name) WHERE id = ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)
				require.Len(t, a.ParameterReferences, 1)
			},
		},
		{
			name: "update with FROM clause",
			sql:  "UPDATE posts SET title = ? FROM users WHERE posts.user_id = users.id AND users.name = ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.FromTables, 2)
				assert.Equal(t, "posts", a.FromTables[0].Name)
				assert.Equal(t, "users", a.FromTables[1].Name)
			},
		},
		{
			name: "update multiple columns with SET",
			sql:  "UPDATE users SET name = ?, email = ? WHERE id = ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 3)
				assert.Equal(t, querier_dto.ParameterContextAssignment, a.ParameterReferences[0].Context)
				assert.Equal(t, querier_dto.ParameterContextAssignment, a.ParameterReferences[1].Context)
			},
		},
		{
			name: "update with CTE",
			sql:  "WITH old AS (SELECT id FROM users WHERE name = ?) UPDATE users SET name = ? WHERE id IN (SELECT id FROM old)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.CTEDefinitions, 1)
				assert.Equal(t, "old", a.CTEDefinitions[0].Name)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Delete(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple delete with WHERE parameter",
			sql:  "DELETE FROM users WHERE id = ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)

				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)

				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, 1, a.ParameterReferences[0].Number)
			},
		},
		{
			name: "delete with RETURNING clause",
			sql:  "DELETE FROM posts WHERE user_id = ? RETURNING id, title",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)

				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "posts", a.FromTables[0].Name)

				assert.True(t, a.HasReturning, "should detect RETURNING clause")
				require.Len(t, a.OutputColumns, 2)
				assert.Equal(t, "id", a.OutputColumns[0].Name)
				assert.Equal(t, "title", a.OutputColumns[1].Name)

				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, 1, a.ParameterReferences[0].Number)
			},
		},
		{
			name: "delete with named parameter",
			sql:  "DELETE FROM users WHERE email = :email",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)

				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)

				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, "email", a.ParameterReferences[0].Name)
			},
		},
		{
			name: "delete with ORDER BY and LIMIT",
			sql:  "DELETE FROM posts WHERE user_id = ? ORDER BY id LIMIT 10",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "posts", a.FromTables[0].Name)
				require.Len(t, a.ParameterReferences, 1)
			},
		},
		{
			name: "delete with CTE",
			sql:  "WITH old_posts AS (SELECT id FROM posts WHERE user_id = ?) DELETE FROM posts WHERE id IN (SELECT id FROM old_posts)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.CTEDefinitions, 1)
				assert.Equal(t, "old_posts", a.CTEDefinitions[0].Name)
			},
		},
		{
			name: "delete with multiple conditions",
			sql:  "DELETE FROM users WHERE name = ? AND email IS NULL",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Values(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	t.Run("simple VALUES statement", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, catalogue, "VALUES (1, 'hello'), (2, 'world')")
		require.NotNil(t, analysis)
		assert.True(t, analysis.ReadOnly, "VALUES should be read-only")
		require.Len(t, analysis.OutputColumns, 2)
	})

	t.Run("VALUES with parameters", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, catalogue, "VALUES (?1, ?2)")
		require.NotNil(t, analysis)
		require.Len(t, analysis.ParameterReferences, 2)
	})
}

func TestAnalyseQuery_SelectExpressions(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "IN list with column context propagated to parameters",
			sql:  "SELECT id FROM users WHERE id IN (:a, :b)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)
				assert.Equal(t, querier_dto.ParameterContextInList, a.ParameterReferences[0].Context)
				assert.Equal(t, querier_dto.ParameterContextInList, a.ParameterReferences[1].Context)
				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "id", a.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
		{
			name: "BETWEEN expression in SELECT list with column context",
			sql:  "SELECT id BETWEEN 1 AND 10 FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "IN list expression in SELECT list",
			sql:  "SELECT id IN (1, 2, 3) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "EXISTS subquery parsed as expression",
			sql:  "SELECT EXISTS (SELECT 1 FROM users WHERE id = 1)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "comparison operators in expression tree",
			sql:  "SELECT id FROM users WHERE id > 1 AND id < 100 AND name <> 'test' AND id >= 5 AND id <= 50 AND id != 42",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "bitwise operators in expression",
			sql:  "SELECT id & 0xFF, id | 0x01, id << 2, id >> 1 FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 4)
			},
		},
		{
			name: "JSON arrow operators",
			sql:  "SELECT id, '{}' -> 'key', '{}' ->> 'nested' FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 3)
			},
		},
		{
			name: "NOT expression wrapping comparison",
			sql:  "SELECT id FROM users WHERE NOT id > 10",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "OR expression joining AND expressions",
			sql:  "SELECT id FROM users WHERE (id = 1 AND name = 'a') OR (id = 2 AND name = 'b')",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "COLLATE suffix on column",
			sql:  "SELECT name COLLATE NOCASE FROM users ORDER BY name COLLATE NOCASE",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "blob literal in expression",
			sql:  "SELECT X'DEADBEEF' FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "REGEXP expression",
			sql:  "SELECT id FROM users WHERE name REGEXP '^[A-Z]'",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "MATCH expression",
			sql:  "SELECT id FROM users WHERE name MATCH 'pattern'",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "window function with RANGE BETWEEN frame",
			sql:  "SELECT id, SUM(id) OVER (ORDER BY id RANGE BETWEEN 1 PRECEDING AND 1 FOLLOWING) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "window function with GROUPS frame",
			sql:  "SELECT id, SUM(id) OVER (ORDER BY id GROUPS BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "window function without parenthesised spec",
			sql:  "SELECT id, ROW_NUMBER() OVER w FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "function with ORDER BY in arguments",
			sql:  "SELECT GROUP_CONCAT(name ORDER BY name ASC) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "function with DISTINCT keyword",
			sql:  "SELECT COUNT(DISTINCT name) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "unary tilde operator",
			sql:  "SELECT ~id FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "unary plus operator",
			sql:  "SELECT +id FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "qualified column with alias in qualified select item",
			sql:  "SELECT u.name AS user_name, u.id user_id FROM users u",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
				assert.Equal(t, "user_name", a.OutputColumns[0].Name)
				assert.Equal(t, "user_id", a.OutputColumns[1].Name)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_OrderByLimitOffset(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "ORDER BY with parameters in expressions",
			sql:  "SELECT id FROM users ORDER BY id LIMIT ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextLimit, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "LIMIT with comma-separated OFFSET parameter",
			sql:  "SELECT id FROM users LIMIT ?, ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)
				assert.Equal(t, querier_dto.ParameterContextLimit, a.ParameterReferences[0].Context)
				assert.Equal(t, querier_dto.ParameterContextOffset, a.ParameterReferences[1].Context)
			},
		},
		{
			name: "ORDER BY terminates on LIMIT keyword",
			sql:  "SELECT id FROM users ORDER BY name, id LIMIT 10",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "ORDER BY terminates on RETURNING keyword",
			sql:  "DELETE FROM users WHERE id = 1 ORDER BY id LIMIT 5",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
			},
		},
		{
			name: "ORDER BY with parameter in expression",
			sql:  "SELECT id FROM users ORDER BY id LIMIT ? OFFSET ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)
				assert.Equal(t, querier_dto.ParameterContextLimit, a.ParameterReferences[0].Context)
				assert.Equal(t, querier_dto.ParameterContextOffset, a.ParameterReferences[1].Context)
			},
		},
		{
			name: "ORDER BY with parenthesised sub-expression",
			sql:  "SELECT id FROM users ORDER BY (id + 1) DESC LIMIT 5",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Insert_ConflictAndSources(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "INSERT DEFAULT VALUES",
			sql:  "INSERT INTO users DEFAULT VALUES",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)
			},
		},
		{
			name: "INSERT with ON CONFLICT DO UPDATE SET with WHERE",
			sql:  "INSERT INTO users (id, name) VALUES (?, ?) ON CONFLICT (id) DO UPDATE SET name = ? WHERE name <> ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 4)
			},
		},
		{
			name: "INSERT with SELECT source containing parameter",
			sql:  "INSERT INTO posts (user_id, title) SELECT id, name FROM users WHERE id > ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "posts", a.FromTables[0].Name)
				require.Len(t, a.ParameterReferences, 1)
			},
		},
		{
			name: "INSERT with multiple value rows",
			sql:  "INSERT INTO users (name, email) VALUES (?, ?), (?, ?)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 4)
			},
		},
		{
			name: "REPLACE INTO",
			sql:  "REPLACE INTO users (id, name) VALUES (?, ?)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)
			},
		},
		{
			name: "INSERT with RETURNING star",
			sql:  "INSERT INTO users (name) VALUES (?) RETURNING *",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.HasReturning)
				require.Len(t, a.OutputColumns, 1)
				assert.True(t, a.OutputColumns[0].IsStar)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Update_ConflictAndSubquery(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "UPDATE OR IGNORE",
			sql:  "UPDATE OR IGNORE users SET name = ? WHERE id = ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)
				require.Len(t, a.ParameterReferences, 2)
			},
		},
		{
			name: "UPDATE with subquery in SET expression",
			sql:  "UPDATE users SET name = (SELECT 'updated') WHERE id = ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Delete_LimitAndReturning(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "DELETE with ORDER BY, LIMIT, and parameter",
			sql:  "DELETE FROM users ORDER BY id LIMIT ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextLimit, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "DELETE with RETURNING star",
			sql:  "DELETE FROM users WHERE id = ? RETURNING *",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.HasReturning)
				require.Len(t, a.OutputColumns, 1)
				assert.True(t, a.OutputColumns[0].IsStar)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Values_OrderByAndLimit(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "VALUES with ORDER BY and LIMIT",
			sql:  "VALUES (1, 'a'), (2, 'b') ORDER BY 1 LIMIT 1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "VALUES with trailing row parameters",
			sql:  "VALUES (1, 'first'), (?, ?)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly)
				require.Len(t, a.ParameterReferences, 2)
			},
		},
		{
			name: "VALUES statement without opening parenthesis",
			sql:  "VALUES",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly)
				assert.Empty(t, a.OutputColumns)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_FromClauseVariants(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "derived table without explicit alias",
			sql:  "SELECT * FROM (SELECT id FROM users) sub",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.RawDerivedTables)
				assert.Equal(t, "sub", a.RawDerivedTables[0].Alias)
			},
		},
		{
			name: "table-valued function without explicit alias",
			sql:  "SELECT * FROM json_each('[1,2,3]')",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.RawTableValuedFunctions)
				assert.Equal(t, "json_each", a.RawTableValuedFunctions[0].FunctionName)
			},
		},
		{
			name: "schema-qualified table reference",
			sql:  "SELECT id FROM main.users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.FromTables, 1)
				assert.Equal(t, "users", a.FromTables[0].Name)
			},
		},
		{
			name: "NATURAL JOIN",
			sql:  "SELECT * FROM users NATURAL JOIN posts",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinInner, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "INNER JOIN explicit keyword",
			sql:  "SELECT * FROM users INNER JOIN posts ON posts.user_id = users.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinInner, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "LEFT OUTER JOIN with OUTER keyword",
			sql:  "SELECT * FROM users LEFT OUTER JOIN posts ON posts.user_id = users.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinLeft, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "join with table-valued function target",
			sql:  "SELECT * FROM users JOIN json_each('[1]') ON 1=1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.RawTableValuedFunctions)
			},
		},
		{
			name: "join with derived table target",
			sql:  "SELECT * FROM users JOIN (SELECT id FROM users) d ON d.id = users.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.RawDerivedTables)
			},
		},
		{
			name: "table alias without AS keyword",
			sql:  "SELECT t.id FROM users t WHERE t.id > 0",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.FromTables, 1)
				assert.Equal(t, "t", a.FromTables[0].Alias)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_ParameterContextDetection(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "parameter in CAST via detectParameterContext",
			sql:  "SELECT id FROM users WHERE id IN (CAST(? AS INTEGER))",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)

				assert.Equal(t, querier_dto.ParameterContextCast, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "parameter in function argument via detectParameterContext path",
			sql:  "SELECT id FROM users WHERE LENGTH(?) > 0",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextFunctionArgument, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "parameter in IN list via detectParameterContext",
			sql:  "SELECT id FROM users WHERE id IN (?, ?, ?)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 3)
				for _, ref := range a.ParameterReferences {
					assert.Equal(t, querier_dto.ParameterContextInList, ref.Context)
				}
			},
		},
		{
			name: "comparison with table-qualified column",
			sql:  "SELECT id FROM users u WHERE u.id = ?",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextComparison, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "u", a.ParameterReferences[0].ColumnReference.TableAlias)
				assert.Equal(t, "id", a.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_WithClassification(t *testing.T) {
	t.Parallel()

	catalogue := newSQLiteCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "WITH INSERT classifies as insert",
			sql:  "WITH src AS (SELECT id, name FROM users) INSERT INTO posts (user_id, title) SELECT id, name FROM src",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CTEDefinitions)
			},
		},
		{
			name: "WITH UPDATE classifies as update",
			sql:  "WITH src AS (SELECT id FROM users) UPDATE users SET name = 'x' WHERE id IN (SELECT id FROM src)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CTEDefinitions)
			},
		},
		{
			name: "WITH DELETE classifies as delete",
			sql:  "WITH old AS (SELECT id FROM users WHERE id < 5) DELETE FROM users WHERE id IN (SELECT id FROM old)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CTEDefinitions)
			},
		},
		{
			name: "RECURSIVE CTE",
			sql:  "WITH RECURSIVE cnt(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM cnt WHERE x < 10) SELECT x FROM cnt",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.CTEDefinitions, 1)
				assert.True(t, a.CTEDefinitions[0].IsRecursive)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestParseStatements_Multiple(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()
	stmts, err := engine.ParseStatements("SELECT 1; SELECT 2")
	require.NoError(t, err)
	require.Len(t, stmts, 2, "should split multiple statements on semicolons")
}

func TestIsParsedStatement(t *testing.T) {
	t.Parallel()

	stmt := &parsedStatement{
		kind: statementKindSelect,
	}

	stmt.IsParsedStatement()
}

func TestClassifyStatement(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()

	tests := []struct {
		name     string
		sql      string
		wantKind statementKind
	}{
		{
			name:     "CREATE TEMP TABLE classifies correctly",
			sql:      "CREATE TEMP TABLE t (id INTEGER)",
			wantKind: statementKindCreateTable,
		},
		{
			name:     "CREATE TEMPORARY VIEW classifies correctly",
			sql:      "CREATE TEMPORARY VIEW v AS SELECT 1",
			wantKind: statementKindCreateView,
		},
		{
			name:     "CREATE TEMP TRIGGER classifies correctly",
			sql:      "CREATE TEMP TRIGGER tr BEFORE INSERT ON t BEGIN SELECT 1; END",
			wantKind: statementKindCreateTrigger,
		},
		{
			name:     "CREATE UNIQUE INDEX classifies correctly",
			sql:      "CREATE UNIQUE INDEX idx ON t (c)",
			wantKind: statementKindCreateIndex,
		},
		{
			name:     "DROP VIEW classifies correctly",
			sql:      "DROP VIEW v",
			wantKind: statementKindDropView,
		},
		{
			name:     "DROP TRIGGER classifies correctly",
			sql:      "DROP TRIGGER tr",
			wantKind: statementKindDropTrigger,
		},
		{
			name:     "DROP INDEX classifies correctly",
			sql:      "DROP INDEX idx",
			wantKind: statementKindDropIndex,
		},
		{
			name:     "REPLACE classifies as insert",
			sql:      "REPLACE INTO t (id) VALUES (1)",
			wantKind: statementKindInsert,
		},
		{
			name:     "unknown statement classifies as unknown",
			sql:      "PRAGMA table_info(users)",
			wantKind: statementKindUnknown,
		},
		{
			name:     "CREATE with unknown second token classifies as unknown",
			sql:      "CREATE SOMETHING t",
			wantKind: statementKindUnknown,
		},
		{
			name:     "DROP with unknown second token classifies as unknown",
			sql:      "DROP SOMETHING t",
			wantKind: statementKindUnknown,
		},
		{
			name:     "WITH VALUES classifies as values",
			sql:      "WITH cte AS (SELECT 1) VALUES (1)",
			wantKind: statementKindValues,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			stmts, err := engine.ParseStatements(testCase.sql)
			require.NoError(t, err)
			require.NotEmpty(t, stmts)

			parsed, ok := stmts[0].Raw.(*parsedStatement)
			require.True(t, ok)
			assert.Equal(t, testCase.wantKind, parsed.kind)
		})
	}
}
