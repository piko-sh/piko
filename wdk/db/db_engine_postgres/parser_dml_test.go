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
	"errors"
	"strings"
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

func TestAnalyseQuery_CompoundLimitOffsetRegistersParameter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sql  string
	}{
		{name: "limit coalesce", sql: "SELECT id FROM users LIMIT COALESCE($1, 100)"},
		{name: "offset greatest", sql: "SELECT id FROM users OFFSET GREATEST($1, 0)"},
		{name: "fetch coalesce", sql: "SELECT id FROM users ORDER BY id FETCH FIRST COALESCE($1, 10) ROWS ONLY"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, nil, tt.sql)
			require.Len(t, analysis.ParameterReferences, 1,
				"a parameter nested in a compound LIMIT/OFFSET value must be registered")
			assert.Equal(t, 1, analysis.ParameterReferences[0].Number)
		})
	}
}

func TestAnalyseQuery_BareLimitOffsetKeepsContext(t *testing.T) {
	t.Parallel()

	analysis := analyseQuery(t, nil, "SELECT id FROM users LIMIT $1 OFFSET $2")
	require.Len(t, analysis.ParameterReferences, 2)
	assert.Equal(t, querier_dto.ParameterContextLimit, analysis.ParameterReferences[0].Context)
	assert.Equal(t, querier_dto.ParameterContextOffset, analysis.ParameterReferences[1].Context)
}

func TestAnalyseQuery_ExistsSubqueryExposesInnerQueryParameters(t *testing.T) {
	t.Parallel()

	analysis := analyseQuery(t, nil,
		`SELECT EXISTS(SELECT 1 FROM orchestrator_tasks WHERE workflow_id = $1 AND status = $2) AS has_incomplete`)

	require.Len(t, analysis.OutputColumns, 1)
	exists, ok := analysis.OutputColumns[0].Expression.(*querier_dto.ExistsExpression)
	require.Truef(t, ok, "EXISTS output column should carry an ExistsExpression, got %T", analysis.OutputColumns[0].Expression)
	require.NotNil(t, exists.InnerQuery)

	require.Len(t, exists.InnerQuery.FromTables, 1)
	assert.Equal(t, "orchestrator_tasks", exists.InnerQuery.FromTables[0].Name)

	require.NotEmpty(t, exists.InnerQuery.ParameterReferences)
	first := exists.InnerQuery.ParameterReferences[0]
	require.NotNil(t, first.ColumnReference)
	assert.Equal(t, "workflow_id", first.ColumnReference.ColumnName)
}

func TestAnalyseQuery_WherePredicateSubqueryCaptured(t *testing.T) {
	t.Parallel()

	analysis := analyseQuery(t, nil,
		`SELECT a.id FROM accounts a WHERE a.id = (SELECT MAX(av2.id) FROM account_versions av2 WHERE av2.id < $2)`)

	require.Len(t, analysis.PredicateSubqueries, 1)
	inner := analysis.PredicateSubqueries[0]
	require.NotNil(t, inner)
	require.Len(t, inner.FromTables, 1)
	assert.Equal(t, "account_versions", inner.FromTables[0].Name)
	assert.Equal(t, "av2", inner.FromTables[0].Alias)

	foundInFlat := false
	for _, parameter := range analysis.ParameterReferences {
		if parameter.ColumnReference != nil && parameter.ColumnReference.TableAlias == "av2" {
			foundInFlat = true
		}
	}
	assert.True(t, foundInFlat, "the subquery parameter must remain spliced into the flat list")
}

func TestAnalyseQuery_WhereParenthesisedExpressionNotCaptured(t *testing.T) {
	t.Parallel()

	analysis := analyseQuery(t, nil, `SELECT a.id FROM accounts a WHERE (a.id + 1) = $1`)
	assert.Empty(t, analysis.PredicateSubqueries)
}

func TestAnalyseQuery_SetCaseComparisonParameterTypedFromOperand(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()
	analysis := analyseQuery(t, catalogue,
		"UPDATE users SET name = CASE WHEN id >= $1 THEN 'flagged' ELSE name END WHERE email = $2")

	require.NotEmpty(t, analysis.ParameterReferences)
	caseParameter := analysis.ParameterReferences[0]
	assert.Equal(t, querier_dto.ParameterContextComparison, caseParameter.Context,
		"a parameter compared against a column must use comparison context, not assignment")
	require.NotNil(t, caseParameter.ColumnReference)
	assert.Equal(t, "id", caseParameter.ColumnReference.ColumnName,
		"the parameter must be typed from the compared column (id), not the SET target (name)")
}

func TestAnalyseQuery_Select(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()

	tests := []struct {
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
		name       string
		sql        string
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

func TestAnalyseQuery_LikeParameterContext(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()

	tests := []struct {
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
		name       string
		sql        string
	}{
		{
			name: "LIKE pattern with direct column LHS",
			sql:  "SELECT id FROM users WHERE name LIKE $1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextLike, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "name", a.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
		{
			name: "ILIKE classifies as Like context",
			sql:  "SELECT id FROM users WHERE email ILIKE $1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextLike, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "email", a.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
		{
			name: "NOT LIKE binds to LHS column",
			sql:  "SELECT id FROM users WHERE name NOT LIKE $1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextLike, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "name", a.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
		{
			name: "LIKE pattern wrapped in concat picks LHS column",
			sql:  "SELECT id FROM users WHERE name LIKE ('%' || $1 || '%')",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextLike, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "name", a.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
		{
			name: "LIKE with function-wrapped column picks the column",
			sql:  "SELECT id FROM users WHERE LOWER(name) LIKE $1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextLike, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "name", a.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
		{
			name: "LIKE with table-qualified LHS preserves the alias",
			sql:  "SELECT id FROM users u WHERE u.email LIKE $1",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextLike, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "u", a.ParameterReferences[0].ColumnReference.TableAlias)
				assert.Equal(t, "email", a.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
		{
			name: "two LIKE patterns on different columns each bind to their own LHS",
			sql:  "SELECT id FROM users WHERE name LIKE $1 OR email LIKE $2",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				require.Len(t, a.ParameterReferences, 2)
				assert.Equal(t, querier_dto.ParameterContextLike, a.ParameterReferences[0].Context)
				require.NotNil(t, a.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "name", a.ParameterReferences[0].ColumnReference.ColumnName)
				assert.Equal(t, querier_dto.ParameterContextLike, a.ParameterReferences[1].Context)
				require.NotNil(t, a.ParameterReferences[1].ColumnReference)
				assert.Equal(t, "email", a.ParameterReferences[1].ColumnReference.ColumnName)
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
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
		name       string
		sql        string
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

				require.NotNil(t, a.InsertSelect)
				require.NotEmpty(t, a.InsertSelect.FromTables)
				assert.Equal(t, "users", a.InsertSelect.FromTables[0].Name)
			},
		},
		{

			name: "INSERT ... SELECT with JOIN and WHERE parameters",
			sql: "INSERT INTO posts (user_id, title) " +
				"SELECT u.id, o.name FROM users u INNER JOIN orgs o ON o.id = u.org_id " +
				"WHERE u.id > $1 AND o.active = $2 " +
				"ON CONFLICT (user_id) DO NOTHING",
			assertions: func(t *testing.T, a *querier_dto.RawQueryAnalysis) {
				require.NotNil(t, a)
				assert.Equal(t, "posts", a.InsertTable)
				require.NotNil(t, a.InsertSelect)
				require.NotEmpty(t, a.InsertSelect.FromTables)
				assert.Equal(t, "users", a.InsertSelect.FromTables[0].Name)
				require.NotEmpty(t, a.InsertSelect.JoinClauses)
				assert.Equal(t, "orgs", a.InsertSelect.JoinClauses[0].Table.Name)
				require.Len(t, a.InsertSelect.ParameterReferences, 2)
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

func findParameterByNumber(references []querier_dto.RawParameterReference, number int) *querier_dto.RawParameterReference {
	for index := range references {
		if references[index].Number == number {
			return &references[index]
		}
	}
	return nil
}

func TestAnalyseQuery_FunctionArgumentMetadata(t *testing.T) {
	t.Parallel()

	t.Run("FROM-clause TVF argument ordinals follow the call-site slot", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, nil,
			"SELECT page_id FROM content.get_pages_with_latest_version($3, $1, $2)")

		require.NotEmpty(t, analysis.RawTableValuedFunctions)
		assert.Equal(t, "content.get_pages_with_latest_version",
			analysis.RawTableValuedFunctions[0].FunctionName)

		expectedOrdinals := map[int]int{3: 0, 1: 1, 2: 2}
		for number, expectedOrdinal := range expectedOrdinals {
			parameter := findParameterByNumber(analysis.ParameterReferences, number)
			require.NotNilf(t, parameter, "expected a reference for $%d", number)
			assert.Equal(t, querier_dto.ParameterContextFunctionArgument, parameter.Context)
			assert.Equal(t, "content.get_pages_with_latest_version", parameter.EnclosingFunctionName)
			assert.Equal(t, expectedOrdinal, parameter.ArgumentOrdinal)
		}
	})

	t.Run("scalar builtin function argument records the bare lower-cased name", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, nil, "SELECT STRING_TO_ARRAY($2, '/')")

		require.NotEmpty(t, analysis.ParameterReferences)
		parameter := findParameterByNumber(analysis.ParameterReferences, 2)
		require.NotNil(t, parameter)
		assert.Equal(t, querier_dto.ParameterContextFunctionArgument, parameter.Context)
		assert.Equal(t, "string_to_array", parameter.EnclosingFunctionName)
		assert.Equal(t, 0, parameter.ArgumentOrdinal)
	})

	t.Run("function argument in SELECT records name and ordinal", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, nil, "SELECT upper($1) AS upper_val FROM users")

		require.NotEmpty(t, analysis.ParameterReferences)
		assert.Equal(t, querier_dto.ParameterContextFunctionArgument, analysis.ParameterReferences[0].Context)
		assert.Equal(t, "upper", analysis.ParameterReferences[0].EnclosingFunctionName)
		assert.Equal(t, 0, analysis.ParameterReferences[0].ArgumentOrdinal)
	})

	t.Run("nested call ordinals are isolated to their own call", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, nil, "SELECT outer_fn($1, inner_fn($2, $3))")

		outerParameter := findParameterByNumber(analysis.ParameterReferences, 1)
		require.NotNil(t, outerParameter)
		assert.Equal(t, "outer_fn", outerParameter.EnclosingFunctionName)
		assert.Equal(t, 0, outerParameter.ArgumentOrdinal)

		innerFirst := findParameterByNumber(analysis.ParameterReferences, 2)
		require.NotNil(t, innerFirst)
		assert.Equal(t, "inner_fn", innerFirst.EnclosingFunctionName)
		assert.Equal(t, 0, innerFirst.ArgumentOrdinal)

		innerSecond := findParameterByNumber(analysis.ParameterReferences, 3)
		require.NotNil(t, innerSecond)
		assert.Equal(t, "inner_fn", innerSecond.EnclosingFunctionName)
		assert.Equal(t, 1, innerSecond.ArgumentOrdinal)
	})

	t.Run("non-placeholder TVF argument still consumes its ordinal", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, nil,
			"SELECT page_id FROM content.get_pages_with_latest_version('latest', $1, $2)")

		first := findParameterByNumber(analysis.ParameterReferences, 1)
		require.NotNil(t, first)
		assert.Equal(t, 1, first.ArgumentOrdinal, "a literal argument burns slot 0")

		second := findParameterByNumber(analysis.ParameterReferences, 2)
		require.NotNil(t, second)
		assert.Equal(t, 2, second.ArgumentOrdinal)
	})
}

func TestAnalyseQuery_InsertSelectProjectionParameter(t *testing.T) {
	t.Parallel()

	t.Run("projection placeholder gets the target column", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, nil,
			"INSERT INTO posts (user_id, title) SELECT $1, name FROM users WHERE id > $2")

		require.NotNil(t, analysis.InsertSelect)

		projection := findParameterByNumber(analysis.ParameterReferences, 1)
		require.NotNil(t, projection)
		assert.Equal(t, querier_dto.ParameterContextAssignment, projection.Context)
		require.NotNil(t, projection.ColumnReference)
		assert.Equal(t, "posts", projection.ColumnReference.TableAlias)
		assert.Equal(t, "user_id", projection.ColumnReference.ColumnName)

		whereParameter := findParameterByNumber(analysis.ParameterReferences, 2)
		require.NotNil(t, whereParameter)
		require.NotNil(t, whereParameter.ColumnReference)
		assert.Equal(t, "id", whereParameter.ColumnReference.ColumnName)
	})

	t.Run("literal projection items burn target-column slots", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, nil,
			"INSERT INTO session_versions (id, session_id, status, reason) "+
				"SELECT gen_random_uuid(), sv.session_id, 'deleted', $1 "+
				"FROM session_versions sv WHERE sv.id = $2")

		projection := findParameterByNumber(analysis.ParameterReferences, 1)
		require.NotNil(t, projection)
		assert.Equal(t, querier_dto.ParameterContextAssignment, projection.Context)
		require.NotNil(t, projection.ColumnReference)
		assert.Equal(t, "session_versions", projection.ColumnReference.TableAlias)
		assert.Equal(t, "reason", projection.ColumnReference.ColumnName)

		whereParameter := findParameterByNumber(analysis.ParameterReferences, 2)
		require.NotNil(t, whereParameter)
		require.NotNil(t, whereParameter.ColumnReference)
		assert.Equal(t, "id", whereParameter.ColumnReference.ColumnName)
	})

	t.Run("data-modifying CTE INSERT ... SELECT projection placeholders", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, nil,
			"WITH inserted AS ("+
				"INSERT INTO media_transformations (id, source_algorithm, priority) "+
				"SELECT $5, $1, $6 RETURNING id) SELECT id FROM inserted")

		idParameter := findParameterByNumber(analysis.ParameterReferences, 5)
		require.NotNil(t, idParameter)
		assert.Equal(t, querier_dto.ParameterContextAssignment, idParameter.Context)
		require.NotNil(t, idParameter.ColumnReference)
		assert.Equal(t, "media_transformations", idParameter.ColumnReference.TableAlias)
		assert.Equal(t, "id", idParameter.ColumnReference.ColumnName)

		priorityParameter := findParameterByNumber(analysis.ParameterReferences, 6)
		require.NotNil(t, priorityParameter)
		require.NotNil(t, priorityParameter.ColumnReference)
		assert.Equal(t, "priority", priorityParameter.ColumnReference.ColumnName)

		algorithmParameter := findParameterByNumber(analysis.ParameterReferences, 1)
		require.NotNil(t, algorithmParameter)
		require.NotNil(t, algorithmParameter.ColumnReference)
		assert.Equal(t, "source_algorithm", algorithmParameter.ColumnReference.ColumnName)
	})

	t.Run("placeholder nested in a projection subquery keeps its own scope", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, nil,
			"INSERT INTO posts (a, b) SELECT $1, (SELECT max(x) FROM t2 WHERE y = $2) FROM t1")

		outer := findParameterByNumber(analysis.ParameterReferences, 1)
		require.NotNil(t, outer)
		assert.Equal(t, querier_dto.ParameterContextAssignment, outer.Context)
		require.NotNil(t, outer.ColumnReference)
		assert.Equal(t, "a", outer.ColumnReference.ColumnName)

		nested := findParameterByNumber(analysis.ParameterReferences, 2)
		require.NotNil(t, nested)
		if nested.ColumnReference != nil {
			assert.NotEqual(t, "b", nested.ColumnReference.ColumnName,
				"a placeholder inside a projection subquery must not bind the INSERT target column")
		}
	})
}

func TestAnalyseQuery_Update(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()

	tests := []struct {
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
		name       string
		sql        string
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
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
		name       string
		sql        string
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
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
		name       string
		sql        string
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
		assertions func(t *testing.T, a *querier_dto.RawQueryAnalysis)
		name       string
		sql        string
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

func TestAnalyseQuery_HasWhereClause(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()

	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{name: "SELECT with WHERE", sql: "SELECT id FROM users WHERE id = $1", want: true},
		{name: "SELECT without WHERE", sql: "SELECT id FROM users", want: false},
		{name: "UPDATE with WHERE", sql: "UPDATE users SET name = $1 WHERE id = $2", want: true},
		{name: "UPDATE without WHERE", sql: "UPDATE users SET name = $1", want: false},
		{name: "DELETE with WHERE", sql: "DELETE FROM users WHERE id = $1", want: true},
		{name: "DELETE without WHERE", sql: "DELETE FROM users", want: false},
		{name: "UPDATE WHERE CURRENT OF", sql: "UPDATE users SET name = $1 WHERE CURRENT OF c1", want: true},
		{name: "DELETE WHERE CURRENT OF", sql: "DELETE FROM users WHERE CURRENT OF c1", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, catalogue, tt.sql)
			require.NotNil(t, analysis)
			assert.Equal(t, tt.want, analysis.HasWhereClause)
		})
	}
}

func TestAnalyseSelect_RejectsDeeplyNestedSubqueries(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()

	const depth = defaultMaxParseDepth + 4
	var sb strings.Builder
	sb.WriteString("SELECT * FROM users")
	for range depth {
		previous := sb.String()
		sb.Reset()
		sb.WriteString("SELECT * FROM (")
		sb.WriteString(previous)
		sb.WriteString(") sub")
	}

	engine := NewPostgresEngine()
	stmts, parseErr := engine.ParseStatements(sb.String())
	require.NoError(t, parseErr)
	require.NotEmpty(t, stmts)

	_, analyseErr := engine.AnalyseQuery(catalogue, stmts[0])
	require.Error(t, analyseErr)
	assert.True(t,
		errors.Is(analyseErr, errAnalysisDepthExceeded) || strings.Contains(analyseErr.Error(), "recursion depth exceeded"),
		"expected recursion depth error, got: %v", analyseErr)
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

func TestAnalyseQuery_InsertOnConflictTargetPredicateReturning(t *testing.T) {
	t.Parallel()

	catalogue := newPostgresCatalogue()

	tests := []struct {
		name           string
		sql            string
		expectedParams int
	}{
		{
			name:           "DO NOTHING RETURNING",
			sql:            "INSERT INTO users (name, email) VALUES ($1, $2) ON CONFLICT (email) WHERE email IS NOT NULL DO NOTHING RETURNING id",
			expectedParams: 2,
		},
		{
			name:           "DO SET RETURNING",
			sql:            "INSERT INTO users (name, email) VALUES ($1, $2) ON CONFLICT (email) WHERE email IS NOT NULL DO UPDATE SET name = $3 RETURNING id",
			expectedParams: 3,
		},
		{
			name:           "predicate over a parenthesised expression still reaches RETURNING",
			sql:            "INSERT INTO users (name, email) VALUES ($1, $2) ON CONFLICT (email) WHERE (email IS NOT NULL AND name <> '') DO NOTHING RETURNING id",
			expectedParams: 2,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			analysis := analyseQuery(t, catalogue, testCase.sql)

			require.NotNil(t, analysis)

			assert.True(t, analysis.HasReturning, "RETURNING has failed to be properly defined")
			require.Len(t, analysis.OutputColumns, 1)
			assert.Equal(t, "id", analysis.OutputColumns[0].Name)
			require.Len(t, analysis.ParameterReferences, testCase.expectedParams)
		})
	}
}
