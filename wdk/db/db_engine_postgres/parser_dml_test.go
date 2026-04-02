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

func newPostgresCatalogue() *querier_dto.Catalogue {
	return &querier_dto.Catalogue{
		DefaultSchema: "public",
		Schemas: map[string]*querier_dto.Schema{
			"public": {
				Name: "public",
				Tables: map[string]*querier_dto.Table{
					"users": {
						Name: "users",
						Columns: []querier_dto.Column{
							{
								Name:    "id",
								SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
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
								SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
							},
							{
								Name:    "user_id",
								SQLType: querier_dto.SQLType{EngineName: "int4", Category: querier_dto.TypeCategoryInteger},
							},
							{
								Name:    "title",
								SQLType: querier_dto.SQLType{EngineName: "text", Category: querier_dto.TypeCategoryText},
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

	engine := NewPostgresEngine()
	stmts, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.NotEmpty(t, stmts)

	analysis, err := engine.AnalyseQuery(catalogue, stmts[0])
	require.NoError(t, err)

	return analysis
}

func TestAnalyseQuery_Select(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()

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
			name: "WHERE with parameter reference",
			sql:  "SELECT id FROM users WHERE id = $1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "id", a.OutputColumns[0].Name)

				require.NotEmpty(t, a.ParameterReferences, "should have at least one parameter reference")
				assert.Equal(t, 1, a.ParameterReferences[0].Number)
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
			},
		},
		{
			name: "aggregate function",
			sql:  "SELECT COUNT(*) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)

				assert.Equal(t, "count", a.OutputColumns[0].Name)
				assert.True(t, a.ReadOnly, "SELECT with aggregate should be read-only")
			},
		},
		{
			name: "CTE with WITH clause",
			sql:  "WITH active AS (SELECT id, name FROM users WHERE id > 0) SELECT id, name FROM active",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CTEDefinitions, "should have CTE definitions")
				assert.Equal(t, "active", a.CTEDefinitions[0].Name)
				require.Len(t, a.OutputColumns, 2)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "recursive CTE",
			sql:  "WITH RECURSIVE nums AS (SELECT 1 AS n UNION ALL SELECT n + 1 FROM nums WHERE n < 10) SELECT n FROM nums",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CTEDefinitions)
				assert.Equal(t, "nums", a.CTEDefinitions[0].Name)
				assert.True(t, a.CTEDefinitions[0].IsRecursive)
			},
		},
		{
			name: "UNION of two selects",
			sql:  "SELECT id, name FROM users UNION SELECT id, title FROM posts",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CompoundBranches)
				assert.Equal(t, querier_dto.CompoundUnion, a.CompoundBranches[0].Operator)
			},
		},
		{
			name: "UNION ALL",
			sql:  "SELECT id FROM users UNION ALL SELECT id FROM posts",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CompoundBranches)
				assert.Equal(t, querier_dto.CompoundUnionAll, a.CompoundBranches[0].Operator)
			},
		},
		{
			name: "INTERSECT",
			sql:  "SELECT id FROM users INTERSECT SELECT user_id FROM posts",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CompoundBranches)
				assert.Equal(t, querier_dto.CompoundIntersect, a.CompoundBranches[0].Operator)
			},
		},
		{
			name: "EXCEPT",
			sql:  "SELECT id FROM users EXCEPT SELECT user_id FROM posts",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CompoundBranches)
				assert.Equal(t, querier_dto.CompoundExcept, a.CompoundBranches[0].Operator)
			},
		},
		{
			name: "subquery in FROM clause",
			sql:  "SELECT sub.cnt FROM (SELECT count(*) AS cnt FROM users) sub",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.RawDerivedTables, "should have derived tables")
				assert.Equal(t, "sub", a.RawDerivedTables[0].Alias)
			},
		},
		{
			name: "GROUP BY and HAVING",
			sql:  "SELECT user_id, count(*) FROM posts GROUP BY user_id HAVING count(*) > 5",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.GroupByColumns)
				assert.Equal(t, "user_id", a.GroupByColumns[0].ColumnName)
			},
		},
		{
			name: "DISTINCT",
			sql:  "SELECT DISTINCT name FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "name", a.OutputColumns[0].Name)
			},
		},
		{
			name: "DISTINCT ON",
			sql:  "SELECT DISTINCT ON (user_id) user_id, title FROM posts ORDER BY user_id, id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
				assert.Equal(t, "user_id", a.OutputColumns[0].Name)
			},
		},
		{
			name: "window function with OVER clause",
			sql:  "SELECT id, name, row_number() OVER (ORDER BY id) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 3)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "window function with PARTITION BY",
			sql:  "SELECT user_id, title, rank() OVER (PARTITION BY user_id ORDER BY id) FROM posts",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 3)
			},
		},
		{
			name: "FOR UPDATE marks as not read-only",
			sql:  "SELECT id, name FROM users WHERE id = $1 FOR UPDATE",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.False(t, a.ReadOnly, "SELECT FOR UPDATE should not be read-only")
				require.NotEmpty(t, a.ParameterReferences)
			},
		},
		{
			name: "LEFT JOIN",
			sql:  "SELECT u.id, p.title FROM users u LEFT JOIN posts p ON p.user_id = u.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.JoinClauses)
				assert.Equal(t, querier_dto.JoinLeft, a.JoinClauses[0].Kind)
				assert.Equal(t, "posts", a.JoinClauses[0].Table.Name)
			},
		},
		{
			name: "RIGHT JOIN",
			sql:  "SELECT u.id, p.title FROM users u RIGHT JOIN posts p ON p.user_id = u.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.JoinClauses)
				assert.Equal(t, querier_dto.JoinRight, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "FULL JOIN",
			sql:  "SELECT u.id, p.title FROM users u FULL JOIN posts p ON p.user_id = u.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.JoinClauses)
				assert.Equal(t, querier_dto.JoinFull, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "CROSS JOIN",
			sql:  "SELECT u.id, p.title FROM users u CROSS JOIN posts p",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.JoinClauses)
				assert.Equal(t, querier_dto.JoinCross, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "LEFT OUTER JOIN",
			sql:  "SELECT u.id FROM users u LEFT OUTER JOIN posts p ON p.user_id = u.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.JoinClauses)
				assert.Equal(t, querier_dto.JoinLeft, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "INNER JOIN",
			sql:  "SELECT u.id FROM users u INNER JOIN posts p ON p.user_id = u.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.JoinClauses)
				assert.Equal(t, querier_dto.JoinInner, a.JoinClauses[0].Kind)
			},
		},
		{
			name: "ORDER BY with multiple columns and directions",
			sql:  "SELECT id, name FROM users ORDER BY name ASC, id DESC",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "LIMIT and OFFSET with parameters",
			sql:  "SELECT id FROM users LIMIT $1 OFFSET $2",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)
				assert.Equal(t, 1, a.ParameterReferences[0].Number)
				assert.Equal(t, querier_dto.ParameterContextLimit, a.ParameterReferences[0].Context)
				assert.Equal(t, 2, a.ParameterReferences[1].Number)
				assert.Equal(t, querier_dto.ParameterContextOffset, a.ParameterReferences[1].Context)
			},
		},
		{
			name: "cast expression with ::type",
			sql:  "SELECT id FROM users WHERE id = $1::integer",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
				assert.Equal(t, 1, a.ParameterReferences[0].Number)
			},
		},
		{
			name: "CAST function syntax",
			sql:  "SELECT CAST($1 AS integer) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
				assert.Equal(t, querier_dto.ParameterContextCast, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].CastType)
			},
		},
		{
			name: "named parameter :name",
			sql:  "SELECT id FROM users WHERE name = :name",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
				assert.Equal(t, "name", a.ParameterReferences[0].Name)
			},
		},
		{
			name: "multiple parameters in complex WHERE",
			sql:  "SELECT id FROM users WHERE id > $1 AND name = $2 AND email = $3",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 3)
				assert.Equal(t, 1, a.ParameterReferences[0].Number)
				assert.Equal(t, 2, a.ParameterReferences[1].Number)
				assert.Equal(t, 3, a.ParameterReferences[2].Number)
			},
		},
		{
			name: "column alias with AS",
			sql:  "SELECT id AS user_id, name AS full_name FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
				assert.Equal(t, "user_id", a.OutputColumns[0].Name)
				assert.Equal(t, "full_name", a.OutputColumns[1].Name)
			},
		},
		{
			name: "table-qualified star",
			sql:  "SELECT u.* FROM users u",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.True(t, a.OutputColumns[0].IsStar)
				assert.Equal(t, "u", a.OutputColumns[0].TableAlias)
			},
		},
		{
			name: "comma-separated FROM tables",
			sql:  "SELECT u.id, p.title FROM users u, posts p WHERE p.user_id = u.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.FromTables, 2)
				assert.Equal(t, "users", a.FromTables[0].Name)
				assert.Equal(t, "posts", a.FromTables[1].Name)
			},
		},
		{
			name: "COALESCE expression",
			sql:  "SELECT COALESCE(email, 'unknown') FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "CASE WHEN expression",
			sql:  "SELECT CASE WHEN id > 10 THEN 'large' ELSE 'small' END FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "EXISTS subquery",
			sql:  "SELECT id FROM users WHERE EXISTS (SELECT 1 FROM posts WHERE posts.user_id = users.id)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "IN list subquery",
			sql:  "SELECT id FROM users WHERE id IN (SELECT user_id FROM posts)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "IN list with literal values",
			sql:  "SELECT id FROM users WHERE id IN (1, 2, 3)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "BETWEEN expression",
			sql:  "SELECT id FROM users WHERE id BETWEEN $1 AND $2",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)

				require.Len(t, a.ParameterReferences, 2)
				assert.Equal(t, 1, a.ParameterReferences[0].Number)
				assert.Equal(t, 2, a.ParameterReferences[1].Number)
			},
		},
		{
			name: "LIKE expression",
			sql:  "SELECT id FROM users WHERE name LIKE '%test%'",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "IS NULL expression",
			sql:  "SELECT id FROM users WHERE email IS NULL",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "IS NOT NULL expression",
			sql:  "SELECT id FROM users WHERE email IS NOT NULL",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "NOT IN expression",
			sql:  "SELECT id FROM users WHERE id NOT IN (1, 2, 3)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "OFFSET before LIMIT",
			sql:  "SELECT id FROM users OFFSET $1 LIMIT $2",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)
			},
		},
		{
			name: "FETCH FIRST syntax",
			sql:  "SELECT id FROM users FETCH FIRST 10 ROWS ONLY",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "scalar subquery in SELECT list",
			sql:  "SELECT id, (SELECT count(*) FROM posts WHERE posts.user_id = users.id) AS post_count FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "boolean literal in WHERE",
			sql:  "SELECT id FROM users WHERE true",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "arithmetic in SELECT",
			sql:  "SELECT id, id + 1 AS next_id FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "string concatenation",
			sql:  "SELECT name || ' <' || email || '>' AS display FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "table-valued function in FROM",
			sql:  "SELECT val FROM generate_series(1, 10) AS val",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.RawTableValuedFunctions)
				assert.Equal(t, "generate_series", a.RawTableValuedFunctions[0].FunctionName)
			},
		},
		{
			name: "multiple CTEs",
			sql:  "WITH a AS (SELECT 1 AS x), b AS (SELECT 2 AS y) SELECT a.x, b.y FROM a, b",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.CTEDefinitions, 2)
				assert.Equal(t, "a", a.CTEDefinitions[0].Name)
				assert.Equal(t, "b", a.CTEDefinitions[1].Name)
			},
		},
		{
			name: "JOIN with USING clause",
			sql:  "SELECT u.id FROM users u JOIN posts p USING (id)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.JoinClauses)
			},
		},
		{
			name: "expression with LIKE in SELECT list uses expression parser",
			sql:  "SELECT name LIKE '%test%' AS is_test FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "is_test", a.OutputColumns[0].Name)
			},
		},
		{
			name: "expression with BETWEEN in SELECT list",
			sql:  "SELECT id BETWEEN 1 AND 100 AS in_range FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "in_range", a.OutputColumns[0].Name)
			},
		},
		{
			name: "expression with IN list in SELECT list",
			sql:  "SELECT id IN (1, 2, 3) AS in_set FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "in_set", a.OutputColumns[0].Name)
			},
		},
		{
			name: "IS NULL in SELECT list",
			sql:  "SELECT email IS NULL AS has_no_email FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "has_no_email", a.OutputColumns[0].Name)
			},
		},
		{
			name: "IS NOT NULL in SELECT list",
			sql:  "SELECT email IS NOT NULL AS has_email FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "NOT expression in SELECT list",
			sql:  "SELECT NOT (id = 1) AS not_first FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "OR expression in SELECT list",
			sql:  "SELECT id = 1 OR id = 2 AS is_first_or_second FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "AND expression in SELECT list",
			sql:  "SELECT id > 0 AND name = 'test' AS both_true FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "multiplication and division in SELECT list",
			sql:  "SELECT id * 2 AS doubled, id / 2 AS halved FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "NOT LIKE in SELECT list",
			sql:  "SELECT name NOT LIKE '%admin%' AS is_regular FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "NOT BETWEEN in SELECT list",
			sql:  "SELECT id NOT BETWEEN 10 AND 20 AS outside_range FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "NOT IN in SELECT list",
			sql:  "SELECT id NOT IN (1, 2) AS not_first_two FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "INTERVAL expression in SELECT list",
			sql:  "SELECT INTERVAL '1 day' AS one_day FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "NULL literal in SELECT list",
			sql:  "SELECT NULL AS nothing FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "nothing", a.OutputColumns[0].Name)
			},
		},
		{
			name: "boolean literal in SELECT list",
			sql:  "SELECT true AS always_true, false AS always_false FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "ROW constructor in SELECT list",
			sql:  "SELECT ROW(id, name) AS user_row FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "function with arguments in SELECT list",
			sql:  "SELECT upper(name) AS upper_name FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "upper_name", a.OutputColumns[0].Name)
			},
		},
		{
			name: "JSON arrow operator in SELECT list",
			sql:  "SELECT '{}'::jsonb -> 'key' AS val FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "unary minus in SELECT list",
			sql:  "SELECT -id AS neg_id FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "ARRAY expression in SELECT list",
			sql:  "SELECT ARRAY[1, 2, 3] AS arr FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "window function with frame clause",
			sql:  "SELECT id, sum(id) OVER (ORDER BY id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS running_sum FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "CASE WHEN with searched expression",
			sql:  "SELECT CASE id WHEN 1 THEN 'one' WHEN 2 THEN 'two' ELSE 'other' END AS label FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "EXISTS subquery in SELECT list",
			sql:  "SELECT EXISTS (SELECT 1 FROM posts WHERE posts.user_id = users.id) AS has_posts FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "IN subquery in SELECT list",
			sql:  "SELECT id IN (SELECT user_id FROM posts) AS has_posts FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "float literal in SELECT list",
			sql:  "SELECT 3.14 AS pi FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "CURRENT_TIMESTAMP in SELECT list",
			sql:  "SELECT CURRENT_TIMESTAMP AS now FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "schema-qualified table in FROM",
			sql:  "SELECT id FROM public.users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "public", a.FromTables[0].Schema)
				assert.Equal(t, "users", a.FromTables[0].Name)
			},
		},
		{
			name: "schema-qualified column reference in SELECT",
			sql:  "SELECT public.users.id FROM public.users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "named parameter used twice deduplicates",
			sql:  "SELECT id FROM users WHERE name = :name OR email = :name",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)

				assert.Equal(t, a.ParameterReferences[0].Number, a.ParameterReferences[1].Number)
				assert.Equal(t, "name", a.ParameterReferences[0].Name)
			},
		},
		{
			name: "multiple different named parameters",
			sql:  "SELECT id FROM users WHERE name = :name AND email = :email",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)
				assert.Equal(t, "name", a.ParameterReferences[0].Name)
				assert.Equal(t, "email", a.ParameterReferences[1].Name)

				assert.NotEqual(t, a.ParameterReferences[0].Number, a.ParameterReferences[1].Number)
			},
		},
		{
			name: "FOR UPDATE OF table",
			sql:  "SELECT id FROM users FOR UPDATE OF users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.False(t, a.ReadOnly)
			},
		},
		{
			name: "GROUP BY with table-qualified column",
			sql:  "SELECT u.name, count(*) FROM users u GROUP BY u.name",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.GroupByColumns)
				assert.Equal(t, "u", a.GroupByColumns[0].TableAlias)
				assert.Equal(t, "name", a.GroupByColumns[0].ColumnName)
			},
		},
		{
			name: "LIMIT ALL",
			sql:  "SELECT id FROM users LIMIT ALL",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "parameter in function call in SELECT",
			sql:  "SELECT upper($1) AS upper_val FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
				assert.Equal(t, querier_dto.ParameterContextFunctionArgument, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "COALESCE with parameter gets comparison context",
			sql:  "SELECT COALESCE(email, $1) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)

				assert.Equal(t, querier_dto.ParameterContextComparison, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "email", a.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
		{
			name: "inline cast ::type on parameter in SELECT",
			sql:  "SELECT $1::text AS casted FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
				assert.Equal(t, querier_dto.ParameterContextCast, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].CastType)
			},
		},
		{
			name: "IS DISTINCT FROM in SELECT list",
			sql:  "SELECT id IS DISTINCT FROM 1 AS is_different FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "IS NOT DISTINCT FROM in SELECT list",
			sql:  "SELECT id IS NOT DISTINCT FROM 1 AS is_same FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "IS TRUE in SELECT list",
			sql:  "SELECT (id > 0) IS TRUE AS is_positive FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "SIMILAR TO in SELECT list",
			sql:  "SELECT name SIMILAR TO '%(test|admin)%' AS is_special FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "ILIKE in SELECT list",
			sql:  "SELECT name ILIKE '%test%' AS matches_ci FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "array subscript in SELECT list",
			sql:  "SELECT ARRAY[10, 20, 30][1] AS first_element FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "comparison with ANY subquery",
			sql:  "SELECT id = ANY(ARRAY[1, 2, 3]) AS in_array FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "modulo operator in SELECT list",
			sql:  "SELECT id % 2 AS remainder FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "BETWEEN SYMMETRIC in SELECT list",
			sql:  "SELECT id BETWEEN SYMMETRIC 10 AND 1 AS in_range FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "parameter in IN list in SELECT list",
			sql:  "SELECT id IN ($1, $2) AS in_set FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
				assert.Equal(t, querier_dto.ParameterContextInList, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "parameter in BETWEEN in SELECT list",
			sql:  "SELECT id BETWEEN $1 AND $2 AS in_range FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)
				assert.Equal(t, querier_dto.ParameterContextBetween, a.ParameterReferences[0].Context)
				assert.Equal(t, querier_dto.ParameterContextBetween, a.ParameterReferences[1].Context)
			},
		},
		{
			name: "LIKE with ESCAPE in SELECT list",
			sql:  "SELECT name LIKE '%\\%%' ESCAPE '\\' AS has_percent FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "double precision literal in SELECT",
			sql:  "SELECT 1e10 AS big_number FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "string literal in SELECT",
			sql:  "SELECT 'hello world' AS greeting FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "CTE with column names",
			sql:  "WITH named_cte (x, y) AS (SELECT id, name FROM users) SELECT x, y FROM named_cte",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CTEDefinitions)
				assert.Equal(t, "named_cte", a.CTEDefinitions[0].Name)
			},
		},
		{
			name: "CTE with VALUES body",
			sql:  "WITH data AS (VALUES (1, 'a'), (2, 'b')) SELECT * FROM data",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CTEDefinitions)
			},
		},
		{
			name: "window function with named window reference",
			sql:  "SELECT id, row_number() OVER w AS rn FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "bitwise AND operator in SELECT",
			sql:  "SELECT id & 1 AS bit_flag FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "aggregate with FILTER clause",
			sql:  "SELECT count(*) FILTER (WHERE id > 5) AS filtered_count FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "aggregate with WITHIN GROUP",
			sql:  "SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY id) AS median_id FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "CAST with array type",
			sql:  "SELECT CAST($1 AS integer[]) AS arr FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
				assert.Equal(t, querier_dto.ParameterContextCast, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "inline cast with timestamp with time zone",
			sql:  "SELECT $1::timestamp with time zone AS ts FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
				assert.Equal(t, querier_dto.ParameterContextCast, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "ARRAY subquery",
			sql:  "SELECT ARRAY(SELECT id FROM users) AS all_ids",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "parameter in WHERE with CAST via skip-tokens path",
			sql:  "SELECT id FROM users WHERE name = CAST($1 AS text)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
			},
		},
		{
			name: "schema.table.column function call",
			sql:  "SELECT pg_catalog.array_length(ARRAY[1,2,3], 1) AS len FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "FETCH FIRST with parameter",
			sql:  "SELECT id FROM users FETCH FIRST $1 ROWS ONLY",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
				assert.Equal(t, querier_dto.ParameterContextLimit, a.ParameterReferences[0].Context)
			},
		},
		{
			name: "OFFSET then FETCH",
			sql:  "SELECT id FROM users OFFSET 10 ROWS FETCH NEXT 5 ROWS ONLY",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "SELECT with implicit column alias (no AS) using expression",
			sql:  "SELECT id + 1 next_id FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "next_id", a.OutputColumns[0].Name)
			},
		},
		{
			name: "table.star in SELECT list with dot prefix",
			sql:  "SELECT users.id FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.Equal(t, "id", a.OutputColumns[0].ColumnName)
			},
		},
		{
			name: "parameter with inline cast in WHERE via skip-tokens",
			sql:  "SELECT id FROM users WHERE id = $1::integer",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
			},
		},
		{
			name: "NOT BETWEEN via expression parser",
			sql:  "SELECT NOT (id BETWEEN 1 AND 10) AS outside FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "parameter inside function in WHERE via skip-tokens",
			sql:  "SELECT id FROM users WHERE name = lower($1)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
			},
		},
		{
			name: "FETCH with OFFSET after it",
			sql:  "SELECT id FROM users FETCH FIRST 10 ROWS ONLY OFFSET 5",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "LIMIT with comma syntax for offset",
			sql:  "SELECT id FROM users LIMIT 10, 5",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly)
			},
		},
		{
			name: "TVF with column definitions",
			sql:  "SELECT x, y FROM generate_series(1, 10) AS gs(x integer)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.RawTableValuedFunctions)
			},
		},
		{
			name: "CTE with NOT MATERIALIZED hint",
			sql:  "WITH active AS NOT MATERIALIZED (SELECT id FROM users) SELECT id FROM active",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CTEDefinitions)
			},
		},
		{
			name: "window function with RANGE frame",
			sql:  "SELECT id, sum(id) OVER (ORDER BY id RANGE BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING) AS cumsum FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "window function with ROWS frame and numeric bounds",
			sql:  "SELECT id, sum(id) OVER (ORDER BY id ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING) AS moving_sum FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "window function with EXCLUDE clause",
			sql:  "SELECT id, sum(id) OVER (ORDER BY id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW EXCLUDE CURRENT ROW) FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 2)
			},
		},
		{
			name: "aggregate function with ORDER BY inside",
			sql:  "SELECT array_agg(name ORDER BY id) AS names FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "aggregate with DISTINCT",
			sql:  "SELECT count(DISTINCT name) AS unique_names FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "table-qualified dot.star in SELECT",
			sql:  "SELECT u.* FROM users u JOIN posts p ON p.user_id = u.id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
				assert.True(t, a.OutputColumns[0].IsStar)
				assert.Equal(t, "u", a.OutputColumns[0].TableAlias)
			},
		},
		{
			name: "array slice expression",
			sql:  "SELECT (ARRAY[1,2,3,4])[2:3] AS slice FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.OutputColumns, 1)
			},
		},
		{
			name: "inline cast on schema-qualified type",
			sql:  "SELECT $1::pg_catalog.int4 AS val FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
			},
		},
		{
			name: "WITH MATERIALIZED hint",
			sql:  "WITH cached AS MATERIALIZED (SELECT id FROM users) SELECT id FROM cached",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.CTEDefinitions)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, tt.sql)
			tt.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Insert(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple insert with parameter",
			sql:  "INSERT INTO users (name) VALUES ($1)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "users", a.InsertTable)
				assert.Equal(t, []string{"name"}, a.InsertColumns)

				require.NotEmpty(t, a.ParameterReferences)
				assert.Equal(t, 1, a.ParameterReferences[0].Number)

				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "name", a.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
		{
			name: "insert with RETURNING clause",
			sql:  "INSERT INTO users (name) VALUES ($1) RETURNING id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "users", a.InsertTable)
				assert.True(t, a.HasReturning, "should detect RETURNING clause")

				require.NotEmpty(t, a.OutputColumns)
				assert.Equal(t, "id", a.OutputColumns[0].Name)
			},
		},
		{
			name: "INSERT with ON CONFLICT DO NOTHING",
			sql:  "INSERT INTO users (name, email) VALUES ($1, $2) ON CONFLICT (email) DO NOTHING",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "users", a.InsertTable)
				assert.Equal(t, []string{"name", "email"}, a.InsertColumns)
				require.Len(t, a.ParameterReferences, 2)
			},
		},
		{
			name: "INSERT with ON CONFLICT DO UPDATE",
			sql:  "INSERT INTO users (name, email) VALUES ($1, $2) ON CONFLICT (email) DO UPDATE SET name = $3",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "users", a.InsertTable)
				require.Len(t, a.ParameterReferences, 3)
			},
		},
		{
			name: "INSERT ... SELECT",
			sql:  "INSERT INTO posts (user_id, title) SELECT id, name FROM users",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "posts", a.InsertTable)
				assert.Equal(t, []string{"user_id", "title"}, a.InsertColumns)
			},
		},
		{
			name: "INSERT with multiple rows",
			sql:  "INSERT INTO users (name) VALUES ($1), ($2), ($3)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "users", a.InsertTable)
				require.Len(t, a.ParameterReferences, 3)
			},
		},
		{
			name: "INSERT with multiple columns",
			sql:  "INSERT INTO users (name, email) VALUES ($1, $2)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "users", a.InsertTable)
				assert.Equal(t, []string{"name", "email"}, a.InsertColumns)
				require.Len(t, a.ParameterReferences, 2)

				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "name", a.ParameterReferences[0].ColumnReference.ColumnName)

				require.NotNil(t, a.ParameterReferences[1].ColumnReference)
				assert.Equal(t, "email", a.ParameterReferences[1].ColumnReference.ColumnName)
			},
		},
		{
			name: "INSERT with RETURNING star",
			sql:  "INSERT INTO users (name) VALUES ($1) RETURNING *",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.HasReturning)
				require.NotEmpty(t, a.OutputColumns)
				assert.True(t, a.OutputColumns[0].IsStar)
			},
		},
		{
			name: "INSERT with ON CONFLICT on constraint name",
			sql:  "INSERT INTO users (name) VALUES ($1) ON CONFLICT ON CONSTRAINT uq_name DO NOTHING",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "users", a.InsertTable)
			},
		},
		{
			name: "INSERT DEFAULT VALUES",
			sql:  "INSERT INTO users DEFAULT VALUES RETURNING id",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "users", a.InsertTable)
				assert.True(t, a.HasReturning)
			},
		},
		{
			name: "INSERT with OVERRIDING SYSTEM VALUE",
			sql:  "INSERT INTO users (id, name) OVERRIDING SYSTEM VALUE VALUES (1, 'test')",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "users", a.InsertTable)
			},
		},
		{
			name: "INSERT without column list",
			sql:  "INSERT INTO users VALUES ($1, $2, $3)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "users", a.InsertTable)
				assert.Empty(t, a.InsertColumns)
			},
		},
		{
			name: "INSERT with ON CONFLICT DO UPDATE SET with WHERE",
			sql:  "INSERT INTO users (name, email) VALUES ($1, $2) ON CONFLICT (email) DO UPDATE SET name = $3 WHERE users.id > 0",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "users", a.InsertTable)
				require.Len(t, a.ParameterReferences, 3)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, tt.sql)
			tt.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Update(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple update with parameters",
			sql:  "UPDATE users SET name = $1 WHERE id = $2",
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
			name: "UPDATE with FROM clause (join-style update)",
			sql:  "UPDATE posts SET title = $1 FROM users WHERE posts.user_id = users.id AND users.name = $2",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)

				require.True(t, len(a.FromTables) >= 2)
				assert.Equal(t, "posts", a.FromTables[0].Name)
				assert.Equal(t, "users", a.FromTables[1].Name)
				require.Len(t, a.ParameterReferences, 2)
			},
		},
		{
			name: "UPDATE with RETURNING",
			sql:  "UPDATE users SET name = $1 WHERE id = $2 RETURNING id, name",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.HasReturning)
				require.Len(t, a.OutputColumns, 2)
				assert.Equal(t, "id", a.OutputColumns[0].Name)
				assert.Equal(t, "name", a.OutputColumns[1].Name)
			},
		},
		{
			name: "UPDATE with multiple SET columns",
			sql:  "UPDATE users SET name = $1, email = $2 WHERE id = $3",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 3)
				assert.Equal(t, querier_dto.ParameterContextAssignment, a.ParameterReferences[0].Context)
				assert.Equal(t, querier_dto.ParameterContextAssignment, a.ParameterReferences[1].Context)
				assert.Equal(t, querier_dto.ParameterContextComparison, a.ParameterReferences[2].Context)
			},
		},
		{
			name: "UPDATE with alias",
			sql:  "UPDATE users u SET name = $1 WHERE u.id = $2",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)
				assert.Equal(t, "u", a.FromTables[0].Alias)
			},
		},
		{
			name: "UPDATE with multi-column SET",
			sql:  "UPDATE users SET (name, email) = ($1, $2) WHERE id = $3",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 3)
			},
		},
		{
			name: "UPDATE with AS alias",
			sql:  "UPDATE users AS u SET name = $1 WHERE u.id = $2",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)
				assert.Equal(t, "u", a.FromTables[0].Alias)
			},
		},
		{
			name: "UPDATE with expression in SET (not parameter)",
			sql:  "UPDATE users SET name = name || ' suffix' WHERE id = $1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.ParameterReferences)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, tt.sql)
			tt.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Delete(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple delete with parameter",
			sql:  "DELETE FROM users WHERE id = $1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)

				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)

				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, 1, a.ParameterReferences[0].Number)
			},
		},
		{
			name: "DELETE with USING clause",
			sql:  "DELETE FROM posts USING users WHERE posts.user_id = users.id AND users.name = $1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)

				require.True(t, len(a.FromTables) >= 2)
				assert.Equal(t, "posts", a.FromTables[0].Name)
				assert.Equal(t, "users", a.FromTables[1].Name)
				require.Len(t, a.ParameterReferences, 1)
			},
		},
		{
			name: "DELETE with RETURNING clause",
			sql:  "DELETE FROM users WHERE id = $1 RETURNING id, name",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.HasReturning)
				require.Len(t, a.OutputColumns, 2)
				assert.Equal(t, "id", a.OutputColumns[0].Name)
				assert.Equal(t, "name", a.OutputColumns[1].Name)
			},
		},
		{
			name: "DELETE with alias",
			sql:  "DELETE FROM users u WHERE u.id = $1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "users", a.FromTables[0].Name)
				assert.Equal(t, "u", a.FromTables[0].Alias)
			},
		},
		{
			name: "DELETE with multiple conditions",
			sql:  "DELETE FROM users WHERE id = $1 AND name = $2",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)
			},
		},
		{
			name: "DELETE with AS alias",
			sql:  "DELETE FROM users AS u WHERE u.id = $1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.NotEmpty(t, a.FromTables)
				assert.Equal(t, "u", a.FromTables[0].Alias)
			},
		},
		{
			name: "DELETE with USING and JOIN in USING",
			sql:  "DELETE FROM posts USING users JOIN posts p2 ON p2.user_id = users.id WHERE posts.id = p2.id AND users.name = $1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "posts", a.FromTables[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, tt.sql)
			tt.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Values(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple VALUES statement",
			sql:  "VALUES (1, 'hello'), (2, 'world')",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.True(t, a.ReadOnly, "VALUES should be read-only")
			},
		},
		{
			name: "VALUES with parameters",
			sql:  "VALUES ($1, $2)",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, tt.sql)
			tt.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_DataModifyingCTE(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "WITH INSERT CTE makes query non-read-only",
			sql:  "WITH ins AS (INSERT INTO posts (user_id, title) VALUES (1, 'test') RETURNING id) SELECT id FROM ins",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.False(t, a.ReadOnly, "data-modifying CTE should not be read-only")
			},
		},
		{
			name: "WITH DELETE CTE",
			sql:  "WITH del AS (DELETE FROM users WHERE id = $1 RETURNING id) SELECT id FROM del",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.False(t, a.ReadOnly)
				require.NotEmpty(t, a.ParameterReferences)
			},
		},
		{
			name: "WITH UPDATE CTE",
			sql:  "WITH upd AS (UPDATE users SET name = 'updated' WHERE id = $1 RETURNING id, name) SELECT id, name FROM upd",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.False(t, a.ReadOnly)
				require.NotEmpty(t, a.CTEDefinitions)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, tt.sql)
			tt.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_NonDML(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()
	engine := NewPostgresEngine()

	stmts, err := engine.ParseStatements("CREATE TABLE foo (id int)")
	require.NoError(t, err)
	require.NotEmpty(t, stmts)

	analysis, err := engine.AnalyseQuery(catalogue, stmts[0])
	require.NoError(t, err)

	require.NotNil(t, analysis, "non-DML should return a non-nil empty analysis")
	assert.Empty(t, analysis.OutputColumns, "non-DML should have no output columns")
	assert.Empty(t, analysis.FromTables, "non-DML should have no FROM tables")
	assert.Empty(t, analysis.ParameterReferences, "non-DML should have no parameter references")
}
