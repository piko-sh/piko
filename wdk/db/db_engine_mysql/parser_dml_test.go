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

package db_engine_mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func analyseQuery(t *testing.T, sql string) *querier_dto.RawQueryAnalysis {
	t.Helper()
	engine := NewMySQLEngine()
	stmts, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.NotEmpty(t, stmts)
	analysis, err := engine.AnalyseQuery(nil, stmts[0])
	require.NoError(t, err)
	require.NotNil(t, analysis)
	return analysis
}

func TestAnalyseQuery_Select(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, analysis *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple select with named columns",
			sql:  "SELECT id, name FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly, "SELECT should be read-only")
				require.Len(t, analysis.OutputColumns, 2)
				assert.Equal(t, "id", analysis.OutputColumns[0].ColumnName)
				assert.Equal(t, "name", analysis.OutputColumns[1].ColumnName)
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
			},
		},
		{
			name: "select star",
			sql:  "SELECT * FROM posts;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.OutputColumns)
				assert.True(t, analysis.OutputColumns[0].IsStar, "star column should have IsStar set")
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "posts", analysis.FromTables[0].Name)
			},
		},
		{
			name: "select with table-qualified star",
			sql:  "SELECT u.* FROM users u;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.OutputColumns)
				assert.True(t, analysis.OutputColumns[0].IsStar)
				assert.Equal(t, "u", analysis.OutputColumns[0].TableAlias)
			},
		},
		{
			name: "select with WHERE and question-mark parameter",
			sql:  "SELECT id, name FROM users WHERE id = ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 1)
				require.NotEmpty(t, analysis.ParameterReferences, "parameter reference should be recorded")
			},
		},
		{
			name: "inner join",
			sql:  "SELECT u.name, p.title FROM users u INNER JOIN posts p ON u.id = p.user_id;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				assert.Equal(t, "u", analysis.FromTables[0].Alias)
				require.Len(t, analysis.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinInner, analysis.JoinClauses[0].Kind)
				assert.Equal(t, "posts", analysis.JoinClauses[0].Table.Name)
				assert.Equal(t, "p", analysis.JoinClauses[0].Table.Alias)
			},
		},
		{
			name: "left join",
			sql:  "SELECT u.name, p.title FROM users u LEFT JOIN posts p ON u.id = p.user_id;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinLeft, analysis.JoinClauses[0].Kind)
			},
		},
		{
			name: "right join",
			sql:  "SELECT u.name, p.title FROM users u RIGHT JOIN posts p ON u.id = p.user_id;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinRight, analysis.JoinClauses[0].Kind)
			},
		},
		{
			name: "cross join",
			sql:  "SELECT a.id, b.id FROM users a CROSS JOIN posts b;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinCross, analysis.JoinClauses[0].Kind)
			},
		},
		{
			name: "select with alias on output column",
			sql:  "SELECT name AS user_name FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
				assert.Equal(t, "user_name", analysis.OutputColumns[0].Name)
			},
		},
		{
			name: "select with GROUP BY",
			sql:  "SELECT user_id, COUNT(*) FROM posts GROUP BY user_id;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.GroupByColumns)
				assert.Equal(t, "user_id", analysis.GroupByColumns[0].ColumnName)
			},
		},
		{
			name: "select with ORDER BY and LIMIT",
			sql:  "SELECT id FROM users ORDER BY id LIMIT 10;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 1)
			},
		},
		{
			name: "select with LIMIT and OFFSET parameters",
			sql:  "SELECT id FROM users LIMIT ? OFFSET ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.ParameterReferences, 2)

				assert.Equal(t, querier_dto.ParameterContextLimit, analysis.ParameterReferences[0].Context)
				assert.Equal(t, querier_dto.ParameterContextOffset, analysis.ParameterReferences[1].Context)
			},
		},
		{
			name: "for update marks query as non-read-only",
			sql:  "SELECT id FROM users FOR UPDATE;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.False(t, analysis.ReadOnly, "FOR UPDATE should mark query as non-read-only")
			},
		},
		{
			name: "select from multiple tables via comma join",
			sql:  "SELECT u.id, p.id FROM users u, posts p;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 2)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				assert.Equal(t, "posts", analysis.FromTables[1].Name)
			},
		},
		{
			name: "union compound query",
			sql:  "SELECT id FROM users UNION SELECT id FROM posts;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.CompoundBranches, 1)
				assert.Equal(t, querier_dto.CompoundUnion, analysis.CompoundBranches[0].Operator)
			},
		},
		{
			name: "union all compound query",
			sql:  "SELECT id FROM users UNION ALL SELECT id FROM posts;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.CompoundBranches, 1)
				assert.Equal(t, querier_dto.CompoundUnionAll, analysis.CompoundBranches[0].Operator)
			},
		},
		{
			name: "schema-qualified table in FROM",
			sql:  "SELECT id FROM mydb.users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "mydb", analysis.FromTables[0].Schema)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
			},
		},

		{
			name: "select with HAVING clause",
			sql:  "SELECT user_id, COUNT(*) AS cnt FROM posts GROUP BY user_id HAVING COUNT(*) > 5;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.GroupByColumns)
				assert.Equal(t, "user_id", analysis.GroupByColumns[0].ColumnName)
				require.Len(t, analysis.OutputColumns, 2)
			},
		},

		{
			name: "select with table-qualified GROUP BY column",
			sql:  "SELECT p.user_id, COUNT(*) FROM posts p GROUP BY p.user_id;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.GroupByColumns)
				assert.Equal(t, "p", analysis.GroupByColumns[0].TableAlias)
				assert.Equal(t, "user_id", analysis.GroupByColumns[0].ColumnName)
			},
		},

		{
			name: "select from derived table",
			sql:  "SELECT sub.id FROM (SELECT id FROM users) AS sub;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.RawDerivedTables)
				assert.Equal(t, "sub", analysis.RawDerivedTables[0].Alias)
			},
		},

		{
			name: "select with IN subquery",
			sql:  "SELECT name FROM users WHERE id IN (SELECT user_id FROM orders);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
			},
		},

		{
			name: "select with EXISTS subquery",
			sql:  "SELECT name FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
			},
		},

		{
			name: "select with scalar subquery in output",
			sql:  "SELECT id, (SELECT COUNT(*) FROM posts WHERE posts.user_id = users.id) AS post_count FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 2)
			},
		},

		{
			name: "select with AND/OR/NOT in WHERE",
			sql:  "SELECT id FROM users WHERE (active = 1 AND age > ?) OR NOT deleted;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.ParameterReferences)
			},
		},
		{
			name: "select with BETWEEN in WHERE",
			sql:  "SELECT id FROM users WHERE age BETWEEN ? AND ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.ParameterReferences, 2)
			},
		},
		{
			name: "select with IN list parameters",
			sql:  "SELECT id FROM users WHERE status IN (?, ?, ?);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.ParameterReferences, 3)
				for _, ref := range analysis.ParameterReferences {
					assert.Equal(t, querier_dto.ParameterContextInList, ref.Context)
				}
			},
		},
		{
			name: "select with LIKE in WHERE",
			sql:  "SELECT id FROM users WHERE name LIKE 'A%';",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 1)
			},
		},
		{
			name: "select with IS NULL in WHERE",
			sql:  "SELECT id FROM users WHERE deleted_at IS NULL;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 1)
			},
		},
		{
			name: "select with IS NOT NULL in WHERE",
			sql:  "SELECT id FROM users WHERE email IS NOT NULL;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 1)
			},
		},

		{
			name: "select with LIMIT parameter only",
			sql:  "SELECT id FROM users LIMIT ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextLimit, analysis.ParameterReferences[0].Context)
			},
		},

		{
			name: "select distinct",
			sql:  "SELECT DISTINCT status FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},

		{
			name: "select with function call in output",
			sql:  "SELECT COUNT(*), MAX(age), MIN(age), AVG(age) FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 4)
			},
		},

		{
			name: "select with CASE expression",
			sql:  "SELECT id, CASE WHEN active = 1 THEN 'yes' ELSE 'no' END AS status FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 2)
			},
		},

		{
			name: "select with COALESCE",
			sql:  "SELECT COALESCE(nickname, name) AS display_name FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},

		{
			name: "select with CAST",
			sql:  "SELECT CAST(id AS CHAR) FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},

		{
			name: "select with CTE",
			sql:  "WITH active_users AS (SELECT id, name FROM users WHERE active = 1) SELECT * FROM active_users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.CTEDefinitions)
				assert.Equal(t, "active_users", analysis.CTEDefinitions[0].Name)
			},
		},

		{
			name: "intersect compound query",
			sql:  "SELECT id FROM users INTERSECT SELECT id FROM admins;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.CompoundBranches, 1)
				assert.Equal(t, querier_dto.CompoundIntersect, analysis.CompoundBranches[0].Operator)
			},
		},
		{
			name: "except compound query",
			sql:  "SELECT id FROM users EXCEPT SELECT id FROM banned;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.CompoundBranches, 1)
				assert.Equal(t, querier_dto.CompoundExcept, analysis.CompoundBranches[0].Operator)
			},
		},

		{
			name: "lock in share mode is parsed without error",
			sql:  "SELECT id FROM users LOCK IN SHARE MODE;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {

				assert.True(t, analysis.ReadOnly)
			},
		},

		{
			name: "join with USING clause",
			sql:  "SELECT u.name, p.title FROM users u JOIN posts p USING (user_id);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinInner, analysis.JoinClauses[0].Kind)
			},
		},

		{
			name: "multiple union compound query",
			sql:  "SELECT id FROM users UNION SELECT id FROM admins UNION ALL SELECT id FROM guests;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)

				require.Len(t, analysis.CompoundBranches, 1)
				assert.Equal(t, querier_dto.CompoundUnion, analysis.CompoundBranches[0].Operator)
				require.NotNil(t, analysis.CompoundBranches[0].Query)
				require.Len(t, analysis.CompoundBranches[0].Query.CompoundBranches, 1)
				assert.Equal(t, querier_dto.CompoundUnionAll, analysis.CompoundBranches[0].Query.CompoundBranches[0].Operator)
			},
		},

		{
			name: "select with comma-style LIMIT offset,count",
			sql:  "SELECT id FROM users LIMIT 10, 20;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Insert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, analysis *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple insert with values",
			sql:  "INSERT INTO users (name, email) VALUES (?, ?);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.Equal(t, "users", analysis.InsertTable)
				assert.Equal(t, []string{"name", "email"}, analysis.InsertColumns)
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				require.Len(t, analysis.ParameterReferences, 2)
			},
		},
		{
			name: "insert without column list",
			sql:  "INSERT INTO users VALUES (?, ?, ?);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.Equal(t, "users", analysis.InsertTable)
				assert.Empty(t, analysis.InsertColumns, "no explicit column list provided")
				require.Len(t, analysis.ParameterReferences, 3)
			},
		},
		{
			name: "insert with ON DUPLICATE KEY UPDATE",
			sql:  "INSERT INTO users (name, email) VALUES (?, ?) ON DUPLICATE KEY UPDATE email = VALUES(email);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.Equal(t, "users", analysis.InsertTable)
				assert.Equal(t, []string{"name", "email"}, analysis.InsertColumns)
			},
		},
		{
			name: "insert ignore",
			sql:  "INSERT IGNORE INTO users (name) VALUES (?);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.Equal(t, "users", analysis.InsertTable)
				assert.Equal(t, []string{"name"}, analysis.InsertColumns)
			},
		},
		{
			name: "replace into",
			sql:  "REPLACE INTO users (id, name) VALUES (?, ?);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.Equal(t, "users", analysis.InsertTable)
				assert.Equal(t, []string{"id", "name"}, analysis.InsertColumns)
			},
		},
		{
			name: "insert into schema-qualified table",
			sql:  "INSERT INTO mydb.users (name) VALUES (?);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.Equal(t, "users", analysis.InsertTable)
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "mydb", analysis.FromTables[0].Schema)
			},
		},
		{
			name: "insert multiple rows",
			sql:  "INSERT INTO users (name) VALUES (?), (?), (?);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.Equal(t, "users", analysis.InsertTable)
				assert.Equal(t, []string{"name"}, analysis.InsertColumns)
				require.Len(t, analysis.ParameterReferences, 3)
			},
		},

		{
			name: "insert on duplicate key update with VALUES() expression",
			sql:  "INSERT INTO counters (id, count) VALUES (?, ?) ON DUPLICATE KEY UPDATE count = count + VALUES(count);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.Equal(t, "counters", analysis.InsertTable)
				assert.Equal(t, []string{"id", "count"}, analysis.InsertColumns)
				require.Len(t, analysis.ParameterReferences, 2)
			},
		},

		{
			name: "insert with SET syntax",
			sql:  "INSERT INTO users SET name = ?, email = ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.Equal(t, "users", analysis.InsertTable)
				require.Len(t, analysis.ParameterReferences, 2)
			},
		},

		{
			name: "insert select",
			sql:  "INSERT INTO archive (id, name) SELECT id, name FROM users WHERE active = 0;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.Equal(t, "archive", analysis.InsertTable)
				assert.Equal(t, []string{"id", "name"}, analysis.InsertColumns)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, analysis *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple update with WHERE",
			sql:  "UPDATE users SET name = ? WHERE id = ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				require.Len(t, analysis.ParameterReferences, 2)
			},
		},
		{
			name: "update with join",
			sql:  "UPDATE users u INNER JOIN profiles p ON u.id = p.user_id SET u.name = p.display_name WHERE u.id = ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				assert.Equal(t, "u", analysis.FromTables[0].Alias)
				require.Len(t, analysis.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinInner, analysis.JoinClauses[0].Kind)
				assert.Equal(t, "profiles", analysis.JoinClauses[0].Table.Name)
			},
		},
		{
			name: "update multiple columns",
			sql:  "UPDATE users SET name = ?, email = ? WHERE id = ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				require.Len(t, analysis.ParameterReferences, 3)
			},
		},
		{
			name: "update schema-qualified table",
			sql:  "UPDATE mydb.users SET name = ? WHERE id = ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "mydb", analysis.FromTables[0].Schema)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
			},
		},
		{
			name: "update with ORDER BY and LIMIT",
			sql:  "UPDATE users SET active = 0 ORDER BY last_login LIMIT 100;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
			},
		},
		{
			name: "update with left join",
			sql:  "UPDATE users u LEFT JOIN profiles p ON u.id = p.user_id SET u.bio = p.bio WHERE p.user_id IS NOT NULL;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				require.Len(t, analysis.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinLeft, analysis.JoinClauses[0].Kind)
			},
		},
		{
			name: "update with table-qualified SET column",
			sql:  "UPDATE users u INNER JOIN roles r ON u.role_id = r.id SET u.role_name = r.name WHERE u.id = ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				require.NotEmpty(t, analysis.ParameterReferences)
			},
		},
		{
			name: "update multiple tables via comma join",
			sql:  "UPDATE users u, profiles p SET u.name = p.display_name WHERE u.id = p.user_id;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 2)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				assert.Equal(t, "profiles", analysis.FromTables[1].Name)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, analysis *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple delete with WHERE",
			sql:  "DELETE FROM users WHERE id = ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				require.NotEmpty(t, analysis.ParameterReferences)
			},
		},
		{
			name: "delete all rows",
			sql:  "DELETE FROM sessions;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "sessions", analysis.FromTables[0].Name)
			},
		},
		{
			name: "delete with ORDER BY and LIMIT",
			sql:  "DELETE FROM logs ORDER BY created_at LIMIT 1000;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "logs", analysis.FromTables[0].Name)
			},
		},
		{
			name: "delete from schema-qualified table",
			sql:  "DELETE FROM mydb.users WHERE id = ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "mydb", analysis.FromTables[0].Schema)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
			},
		},
		{
			name: "delete with alias",
			sql:  "DELETE FROM users AS u WHERE u.id = ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				assert.Equal(t, "u", analysis.FromTables[0].Alias)
			},
		},
		{
			name: "delete with multiple parameters",
			sql:  "DELETE FROM users WHERE active = ? AND last_login < ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				require.Len(t, analysis.ParameterReferences, 2)
			},
		},

		{
			name: "multi-table delete",
			sql:  "DELETE u FROM users u INNER JOIN banned b ON u.id = b.user_id;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.NotEmpty(t, analysis.FromTables)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				require.Len(t, analysis.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinInner, analysis.JoinClauses[0].Kind)
			},
		},
		{
			name: "multi-table delete with WHERE",
			sql:  "DELETE u FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE o.id IS NULL;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.NotEmpty(t, analysis.FromTables)
				require.Len(t, analysis.JoinClauses, 1)
				assert.Equal(t, querier_dto.JoinLeft, analysis.JoinClauses[0].Kind)
			},
		},

		{
			name: "delete with implicit alias",
			sql:  "DELETE FROM users u WHERE u.id = ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
				assert.Equal(t, "u", analysis.FromTables[0].Alias)
				require.NotEmpty(t, analysis.ParameterReferences)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_NonDML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "CREATE TABLE returns empty analysis",
			sql:  "CREATE TABLE users (id INT);",
		},
		{
			name: "DROP TABLE returns empty analysis",
			sql:  "DROP TABLE users;",
		},
		{
			name: "ALTER TABLE returns empty analysis",
			sql:  "ALTER TABLE users ADD COLUMN age INT;",
		},
		{
			name: "CREATE INDEX returns empty analysis",
			sql:  "CREATE INDEX idx_name ON users (name);",
		},
		{
			name: "CREATE VIEW returns empty analysis",
			sql:  "CREATE VIEW v_active AS SELECT * FROM users WHERE active = 1;",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, testCase.sql)

			assert.Empty(t, analysis.FromTables, "DDL should produce no FROM tables")
			assert.Empty(t, analysis.OutputColumns, "DDL should produce no output columns")
			assert.Empty(t, analysis.InsertTable, "DDL should produce no insert table")
		})
	}
}

func TestAnalyseQuery_Values(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, analysis *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "simple VALUES row",
			sql:  "VALUES (1, 'alice');",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 2)
			},
		},
		{
			name: "VALUES with parameter",
			sql:  "VALUES (?);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.ParameterReferences)
			},
		},
		{
			name: "VALUES with multiple rows",
			sql:  "VALUES (1, 'a'), (2, 'b');",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 2)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_ParameterContexts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, analysis *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "parameter in comparison context",
			sql:  "SELECT id FROM users WHERE id = ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextComparison, analysis.ParameterReferences[0].Context)
				require.NotNil(t, analysis.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "id", analysis.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
		{
			name: "parameter in assignment context via INSERT",
			sql:  "INSERT INTO users (name) VALUES (?);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextAssignment, analysis.ParameterReferences[0].Context)
				require.NotNil(t, analysis.ParameterReferences[0].ColumnReference)
				assert.Equal(t, "name", analysis.ParameterReferences[0].ColumnReference.ColumnName)
			},
		},
		{
			name: "parameter in assignment context via UPDATE SET",
			sql:  "UPDATE users SET name = ? WHERE id = 1;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextAssignment, analysis.ParameterReferences[0].Context)
			},
		},
		{
			name: "parameter in function argument context",
			sql:  "SELECT CONCAT(?, name) FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextFunctionArgument, analysis.ParameterReferences[0].Context)
			},
		},
		{
			name: "parameter in CAST context",
			sql:  "SELECT CAST(? AS UNSIGNED) FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextCast, analysis.ParameterReferences[0].Context)
				require.NotNil(t, analysis.ParameterReferences[0].CastType)
			},
		},
		{
			name: "parameter in BETWEEN context via WHERE",
			sql:  "SELECT id FROM users WHERE created_at BETWEEN ? AND ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {

				require.Len(t, analysis.ParameterReferences, 2)
			},
		},
		{
			name: "parameter in IN list context",
			sql:  "SELECT id FROM users WHERE id IN (?);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextInList, analysis.ParameterReferences[0].Context)
			},
		},
		{
			name: "parameter in OFFSET context",
			sql:  "SELECT id FROM users OFFSET ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextOffset, analysis.ParameterReferences[0].Context)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_Expressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, analysis *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "arithmetic expressions in output",
			sql:  "SELECT price * quantity AS total, price + tax AS with_tax FROM orders;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 2)
			},
		},
		{
			name: "boolean TRUE and FALSE literals",
			sql:  "SELECT id FROM users WHERE active = TRUE AND deleted = FALSE;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "NOT BETWEEN expression",
			sql:  "SELECT id FROM users WHERE age NOT BETWEEN 18 AND 65;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "NOT IN expression",
			sql:  "SELECT id FROM users WHERE status NOT IN ('banned', 'suspended');",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "NOT LIKE expression",
			sql:  "SELECT id FROM users WHERE name NOT LIKE '%test%';",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "CONVERT expression",
			sql:  "SELECT CONVERT(price, DECIMAL(10,2)) FROM products;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "IF expression",
			sql:  "SELECT IF(active = 1, 'yes', 'no') AS status FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "INTERVAL literal",
			sql:  "SELECT id FROM events WHERE created_at > NOW() - INTERVAL 30 DAY;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "GROUP_CONCAT function",
			sql:  "SELECT user_id, GROUP_CONCAT(DISTINCT tag ORDER BY tag SEPARATOR ', ') FROM tags GROUP BY user_id;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.GroupByColumns)
			},
		},
		{
			name: "TRIM function",
			sql:  "SELECT TRIM(LEADING ' ' FROM name) FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "EXTRACT function",
			sql:  "SELECT EXTRACT(YEAR FROM created_at) FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "window function",
			sql:  "SELECT id, ROW_NUMBER() OVER (ORDER BY id) AS rn FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 2)
			},
		},
		{
			name: "window function with PARTITION BY",
			sql:  "SELECT dept_id, name, ROW_NUMBER() OVER (PARTITION BY dept_id ORDER BY name) AS rn FROM employees;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 3)
			},
		},
		{
			name: "JSON arrow operator",
			sql:  "SELECT data->'$.name' FROM documents;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "JSON double arrow operator",
			sql:  "SELECT data->>'$.name' FROM documents;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "unary negation in output",
			sql:  "SELECT -amount FROM transactions;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "hex string literal",
			sql:  "SELECT X'48454C4C4F' AS hex_val;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "string literal in output",
			sql:  "SELECT 'hello' AS greeting;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "numeric literals in output",
			sql:  "SELECT 42, 3.14, 1e10;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 3)
			},
		},
		{
			name: "COLLATE expression",
			sql:  "SELECT name COLLATE utf8mb4_bin FROM users ORDER BY name COLLATE utf8mb4_bin;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "implicit function identifiers",
			sql:  "SELECT CURRENT_TIMESTAMP, CURRENT_DATE, CURRENT_USER;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 3)
			},
		},

		{
			name: "IS NULL expression in output column",
			sql:  "SELECT (email IS NULL) AS missing_email FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "IS NOT NULL expression in output column",
			sql:  "SELECT (email IS NOT NULL) AS has_email FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "BETWEEN expression in output column",
			sql:  "SELECT (age BETWEEN 18 AND 65) AS working_age FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "IN list expression in output column",
			sql:  "SELECT (status IN ('active', 'pending')) AS is_valid FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "LIKE expression in output column",
			sql:  "SELECT (name LIKE 'A%') AS starts_with_a FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "NOT LIKE expression in output column",
			sql:  "SELECT (name NOT LIKE '%test%') AS not_test FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "NOT IN expression in output column",
			sql:  "SELECT (status NOT IN ('banned', 'deleted')) AS is_ok FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "NOT BETWEEN expression in output column",
			sql:  "SELECT (age NOT BETWEEN 0 AND 17) AS is_adult FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "OR expression in output column",
			sql:  "SELECT (active = 1 OR role = 'admin') AS has_access FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "AND expression in output column",
			sql:  "SELECT (active = 1 AND verified = 1) AS fully_active FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "NOT expression in output column",
			sql:  "SELECT NOT deleted AS is_active FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "NULL literal in output column",
			sql:  "SELECT NULL AS nothing;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "boolean TRUE/FALSE in output column",
			sql:  "SELECT TRUE AS yes, FALSE AS no;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 2)
			},
		},
		{
			name: "INTERVAL expression in output",
			sql:  "SELECT NOW() + INTERVAL 7 DAY AS next_week;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "REGEXP expression in output column",
			sql:  "SELECT (name REGEXP '^[A-Z]') AS starts_upper FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "comparison operators in output",
			sql:  "SELECT (a > b) AS gt, (a < b) AS lt, (a >= b) AS gte, (a <= b) AS lte, (a <> b) AS ne FROM t;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 5)
			},
		},
		{
			name: "bitwise operators in output",
			sql:  "SELECT (a & b) AS band, (a | b) AS bor, (a ^ b) AS bxor FROM t;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 3)
			},
		},
		{
			name: "parenthesised subquery in output column",
			sql:  "SELECT id, (SELECT MAX(score) FROM scores WHERE scores.user_id = users.id) AS max_score FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 2)
			},
		},
		{
			name: "schema-qualified function call",
			sql:  "SELECT mydb.my_func(1) AS result;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "LIKE pattern in output column",
			sql:  "SELECT (name LIKE '%test%') AS matches FROM users",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "IN subquery in output column",
			sql:  "SELECT (id IN (SELECT user_id FROM orders)) AS has_orders FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "EXISTS in output column",
			sql:  "SELECT EXISTS(SELECT 1 FROM orders WHERE orders.user_id = users.id) AS has_orders FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "CASE with searched WHEN clauses",
			sql:  "SELECT CASE status WHEN 'active' THEN 1 WHEN 'inactive' THEN 0 ELSE -1 END AS status_code FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "COALESCE with parameter",
			sql:  "SELECT COALESCE(nickname, ?) AS display FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.ParameterReferences)
			},
		},
		{
			name: "CONVERT with USING",
			sql:  "SELECT CONVERT(name USING utf8mb4) FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "IS NULL expression in output column",
			sql:  "SELECT (email IS NULL) AS missing_email FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "IS NOT NULL expression in output column",
			sql:  "SELECT (email IS NOT NULL) AS has_email FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "BETWEEN expression in output column",
			sql:  "SELECT (age BETWEEN 18 AND 65) AS working_age FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "IN list expression in output column",
			sql:  "SELECT (status IN ('active', 'pending')) AS is_valid FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "LIKE expression in output column",
			sql:  "SELECT (name LIKE 'A%') AS starts_with_a FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "NOT IN expression in output column",
			sql:  "SELECT (status NOT IN ('banned', 'deleted')) AS is_ok FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "OR expression in output column",
			sql:  "SELECT (active = 1 OR role = 'admin') AS has_access FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "AND expression in output column",
			sql:  "SELECT (active = 1 AND verified = 1) AS fully_active FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "REGEXP expression in output column",
			sql:  "SELECT (name REGEXP '^[A-Z]') AS starts_upper FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "SOUNDS LIKE in output column",
			sql:  "SELECT (name SOUNDS LIKE 'john') AS matches FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "ROW constructor in output",
			sql:  "SELECT ROW(1, 2, 3);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "BINARY prefix in output",
			sql:  "SELECT BINARY name FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "schema.table.column reference in output",
			sql:  "SELECT mydb.users.name FROM mydb.users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "IN subquery in output column",
			sql:  "SELECT (id IN (SELECT user_id FROM orders)) AS has_orders FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "EXISTS in output column",
			sql:  "SELECT EXISTS(SELECT 1 FROM orders WHERE orders.user_id = users.id) AS has_orders FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "BETWEEN with parameters in output column",
			sql:  "SELECT (price BETWEEN ? AND ?) AS in_range FROM products;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.ParameterReferences, 2)
				assert.Equal(t, querier_dto.ParameterContextBetween, analysis.ParameterReferences[0].Context)
				assert.Equal(t, querier_dto.ParameterContextBetween, analysis.ParameterReferences[1].Context)
			},
		},
		{
			name: "IN list with parameters in output column",
			sql:  "SELECT (status IN (?, ?)) AS matches FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.ParameterReferences, 2)
				assert.Equal(t, querier_dto.ParameterContextInList, analysis.ParameterReferences[0].Context)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_NamedParameters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, analysis *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "named parameter in WHERE",
			sql:  "SELECT id FROM users WHERE id = :user_id;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.ParameterReferences, 1)
				assert.Equal(t, "user_id", analysis.ParameterReferences[0].Name)
			},
		},
		{
			name: "named parameters in INSERT",
			sql:  "INSERT INTO users (name, email) VALUES (:name, :email);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.Equal(t, "users", analysis.InsertTable)
				require.Len(t, analysis.ParameterReferences, 2)
				assert.Equal(t, "name", analysis.ParameterReferences[0].Name)
				assert.Equal(t, "email", analysis.ParameterReferences[1].Name)
			},
		},
		{
			name: "repeated named parameter gets same number",
			sql:  "SELECT id FROM users WHERE name = :search OR email = :search;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.ParameterReferences, 2)
				assert.Equal(t, analysis.ParameterReferences[0].Number, analysis.ParameterReferences[1].Number,
					"repeated named parameter should share the same number")
			},
		},
		{
			name: "named parameter in UPDATE SET",
			sql:  "UPDATE users SET name = :name WHERE id = :id;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				require.Len(t, analysis.ParameterReferences, 2)
				assert.Equal(t, "name", analysis.ParameterReferences[0].Name)
				assert.Equal(t, "id", analysis.ParameterReferences[1].Name)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestParseStatements_Delimiter(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()

	t.Run("custom delimiter separates statements", func(t *testing.T) {
		t.Parallel()
		sql := "DELIMITER //\nCREATE TABLE a (id INT)//\nCREATE TABLE b (id INT)//\nDELIMITER ;"
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.Len(t, statements, 2)
	})

	t.Run("delimiter reset to semicolon", func(t *testing.T) {
		t.Parallel()
		sql := "DELIMITER //\nCREATE TABLE a (id INT)//\nDELIMITER ;\nCREATE TABLE b (id INT);"
		statements, err := engine.ParseStatements(sql)
		require.NoError(t, err)
		require.Len(t, statements, 2)
	})
}

func TestAnalyseQuery_Select_WindowsAndCTEs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, analysis *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "select with FOR UPDATE OF table",
			sql:  "SELECT id FROM users FOR UPDATE OF users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.False(t, analysis.ReadOnly, "FOR UPDATE should mark query as non-read-only")
			},
		},
		{
			name: "select with window ROWS BETWEEN frame",
			sql:  "SELECT id, SUM(amount) OVER (ORDER BY id ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING) AS running FROM orders;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 2)
			},
		},
		{
			name: "select with window RANGE UNBOUNDED",
			sql:  "SELECT id, SUM(amount) OVER (ORDER BY id RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS cumulative FROM orders;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "select with comma-style LIMIT using parameters",
			sql:  "SELECT id FROM users LIMIT ?, ?;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.ParameterReferences, 2)

				assert.Equal(t, querier_dto.ParameterContextOffset, analysis.ParameterReferences[0].Context)
				assert.Equal(t, querier_dto.ParameterContextLimit, analysis.ParameterReferences[1].Context)
			},
		},
		{
			name: "CTE with column names",
			sql:  "WITH numbered(id, rn) AS (SELECT id, ROW_NUMBER() OVER (ORDER BY id) FROM users) SELECT * FROM numbered;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.CTEDefinitions)
				assert.Equal(t, "numbered", analysis.CTEDefinitions[0].Name)
				require.Len(t, analysis.CTEDefinitions[0].OutputColumns, 2)
				assert.Equal(t, "id", analysis.CTEDefinitions[0].OutputColumns[0].Name)
				assert.Equal(t, "rn", analysis.CTEDefinitions[0].OutputColumns[1].Name)
			},
		},
		{
			name: "recursive CTE",
			sql: `WITH RECURSIVE cte AS (
				SELECT 1 AS n
				UNION ALL
				SELECT n + 1 FROM cte WHERE n < 10
			) SELECT n FROM cte;`,
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.CTEDefinitions)
				assert.True(t, analysis.CTEDefinitions[0].IsRecursive)
			},
		},
		{
			name: "select with expression alias without AS",
			sql:  "SELECT 1 + 2 total;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
				assert.Equal(t, "total", analysis.OutputColumns[0].Name)
			},
		},
		{
			name: "select with division and modulo operators",
			sql:  "SELECT (a / b) AS quotient, (a % b) AS remainder FROM t;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 2)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_SelectExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, analysis *querier_dto.RawQueryAnalysis)
	}{
		{
			name: "bang (!) negation operator",
			sql:  "SELECT !active AS inactive FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "comparison with ANY subquery",
			sql:  "SELECT id FROM users WHERE salary > ANY (SELECT salary FROM employees);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "comparison with ALL subquery",
			sql:  "SELECT id FROM users WHERE salary > ALL (SELECT salary FROM employees);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "comparison with SOME subquery",
			sql:  "SELECT id FROM users WHERE salary = SOME (SELECT salary FROM employees);",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "IS TRUE expression",
			sql:  "SELECT (active IS TRUE) AS definitely_active FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "IS FALSE expression",
			sql:  "SELECT (active IS FALSE) AS definitely_inactive FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "IS NOT TRUE expression",
			sql:  "SELECT (active IS NOT TRUE) AS not_active FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "IS UNKNOWN expression",
			sql:  "SELECT (active IS UNKNOWN) AS mystery FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "LIKE with ESCAPE clause",
			sql:  "SELECT id FROM users WHERE name LIKE '%!_%' ESCAPE '!';",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "tilde bitwise NOT",
			sql:  "SELECT ~flags AS inverted FROM t;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 1)
			},
		},
		{
			name: "unary plus",
			sql:  "SELECT +amount FROM orders;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "CAST with DOUBLE PRECISION type",
			sql:  "SELECT CAST(value AS DOUBLE PRECISION) FROM t;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "CAST with UNSIGNED type",
			sql:  "SELECT CAST(x AS UNSIGNED) FROM t;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "shift operators",
			sql:  "SELECT (a << 2) AS lshift, (a >> 1) AS rshift FROM t;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.OutputColumns, 2)
			},
		},
		{
			name: "null-safe comparison operator",
			sql:  "SELECT (a <=> b) AS safe_eq FROM t;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "not-equal operator !=",
			sql:  "SELECT (a != b) AS neq FROM t;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "bit string literal",
			sql:  "SELECT b'10101010' AS bits;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "window function with named window reference",
			sql:  "SELECT id, ROW_NUMBER() OVER w AS rn FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "window function with empty OVER()",
			sql:  "SELECT id, ROW_NUMBER() OVER () AS rn FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "table with USE INDEX hint",
			sql:  "SELECT id FROM users USE INDEX (idx_name) WHERE name = 'alice';",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 1)
				assert.Equal(t, "users", analysis.FromTables[0].Name)
			},
		},
		{
			name: "table with FORCE INDEX hint",
			sql:  "SELECT id FROM users FORCE INDEX (PRIMARY) WHERE id > 100;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 1)
			},
		},
		{
			name: "table with IGNORE INDEX hint",
			sql:  "SELECT id FROM users IGNORE INDEX (idx_name) WHERE name = 'bob';",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.FromTables, 1)
			},
		},
		{
			name: "CAST parameter in expression context",
			sql:  "SELECT CAST(? AS CHAR) FROM users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.ParameterReferences, 1)
				assert.Equal(t, querier_dto.ParameterContextCast, analysis.ParameterReferences[0].Context)
			},
		},
		{
			name: "schema-qualified column in schema.table.column form",
			sql:  "SELECT mydb.users.id FROM mydb.users;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "schema-qualified function call",
			sql:  "SELECT mydb.my_schema.custom_fn(1) AS result;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
			},
		},
		{
			name: "derived table without explicit AS keyword",
			sql:  "SELECT sub.id FROM (SELECT id FROM users) sub;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.NotEmpty(t, analysis.RawDerivedTables)
				assert.Equal(t, "sub", analysis.RawDerivedTables[0].Alias)
			},
		},
		{
			name: "NATURAL JOIN",
			sql:  "SELECT * FROM users NATURAL JOIN profiles;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.JoinClauses, 1)
			},
		},
		{
			name: "STRAIGHT_JOIN",
			sql:  "SELECT * FROM users STRAIGHT_JOIN posts ON users.id = posts.user_id;",
			assertions: func(t *testing.T, analysis *querier_dto.RawQueryAnalysis) {
				assert.True(t, analysis.ReadOnly)
				require.Len(t, analysis.JoinClauses, 1)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			analysis := analyseQuery(t, testCase.sql)
			testCase.assertions(t, analysis)
		})
	}
}

func TestAnalyseQuery_CTEWithDML(t *testing.T) {
	t.Parallel()

	t.Run("CTE with INSERT body marks query non-read-only", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "WITH inserted AS (INSERT INTO archive (id) VALUES (1)) SELECT * FROM inserted;")
		require.NotNil(t, analysis)
		assert.False(t, analysis.ReadOnly, "data-modifying CTE should mark query as non-read-only")
	})

	t.Run("CTE with UPDATE body marks query non-read-only", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "WITH updated AS (UPDATE users SET active = 0 WHERE id = 1) SELECT * FROM updated;")
		require.NotNil(t, analysis)
		assert.False(t, analysis.ReadOnly, "data-modifying CTE should mark query as non-read-only")
	})

	t.Run("CTE with DELETE body marks query non-read-only", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "WITH deleted AS (DELETE FROM users WHERE id = 1) SELECT * FROM deleted;")
		require.NotNil(t, analysis)
		assert.False(t, analysis.ReadOnly, "data-modifying CTE should mark query as non-read-only")
	})

	t.Run("CTE with VALUES body remains read-only", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "WITH data AS (VALUES (1, 'a'), (2, 'b')) SELECT * FROM data;")
		require.NotNil(t, analysis)
		assert.True(t, analysis.ReadOnly, "CTE with VALUES should remain read-only")
	})
}

func TestAnalyseQuery_Values_OrderByAndLimit(t *testing.T) {
	t.Parallel()

	t.Run("VALUES with ORDER BY", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "VALUES (3, 'c'), (1, 'a'), (2, 'b') ORDER BY 1;")
		require.NotNil(t, analysis)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("VALUES with LIMIT", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "VALUES (1), (2), (3) LIMIT 2;")
		require.NotNil(t, analysis)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("VALUES with OFFSET", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "VALUES (1), (2), (3) LIMIT 2 OFFSET 1;")
		require.NotNil(t, analysis)
		assert.True(t, analysis.ReadOnly)
	})
}

func TestAnalyseQuery_Delete_MultiTable(t *testing.T) {
	t.Parallel()

	t.Run("multi-table delete with schema-qualified target", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "DELETE u FROM mydb.users u WHERE u.active = 0;")
		require.NotNil(t, analysis)
		require.NotEmpty(t, analysis.FromTables)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
	})

	t.Run("multi-table delete with WHERE clause", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "DELETE u FROM users u INNER JOIN banned b ON u.id = b.user_id WHERE b.reason = 'spam';")
		require.NotNil(t, analysis)
		require.NotEmpty(t, analysis.FromTables)
	})

	t.Run("multi-table delete with multiple target tables", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "DELETE u, p FROM users u INNER JOIN profiles p ON u.id = p.user_id WHERE u.active = 0;")
		require.NotNil(t, analysis)
		require.NotEmpty(t, analysis.FromTables)
	})
}

func TestAnalyseQuery_InsertReturning(t *testing.T) {
	t.Parallel()

	t.Run("insert with RETURNING clause", func(t *testing.T) {
		t.Parallel()
		engine := NewMySQLEngine(WithReturningSupport(true))
		statements, err := engine.ParseStatements("INSERT INTO users (name) VALUES (?) RETURNING id, name;")
		require.NoError(t, err)
		require.NotEmpty(t, statements)
		analysis, err := engine.AnalyseQuery(nil, statements[0])
		require.NoError(t, err)
		require.NotNil(t, analysis)
		assert.Equal(t, "users", analysis.InsertTable)
	})
}

func TestAnalyseQuery_UpdateMultiColumnSet(t *testing.T) {
	t.Parallel()

	t.Run("update with tuple-style SET clause", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "UPDATE users SET (name, email) = (?, ?) WHERE id = 1;")
		require.NotNil(t, analysis)
		require.NotEmpty(t, analysis.FromTables)
		assert.Equal(t, "users", analysis.FromTables[0].Name)
		require.Len(t, analysis.ParameterReferences, 2)
	})
}

func TestAnalyseQuery_ForUpdateVariants(t *testing.T) {
	t.Parallel()

	t.Run("FOR UPDATE SKIP LOCKED", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "SELECT id FROM users FOR UPDATE SKIP LOCKED;")
		assert.False(t, analysis.ReadOnly)
	})

	t.Run("FOR UPDATE NOWAIT", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "SELECT id FROM users FOR UPDATE NOWAIT;")
		assert.False(t, analysis.ReadOnly)
	})

	t.Run("FOR SHARE also marks non-read-only", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "SELECT id FROM users FOR SHARE;")
		assert.False(t, analysis.ReadOnly, "FOR SHARE should mark query as non-read-only")
	})
}

func TestAnalyseQuery_WindowFrameVariants(t *testing.T) {
	t.Parallel()

	t.Run("ROWS UNBOUNDED PRECEDING", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "SELECT id, SUM(amount) OVER (ORDER BY id ROWS UNBOUNDED PRECEDING) AS running FROM orders;")
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("ROWS CURRENT ROW", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "SELECT id, SUM(amount) OVER (ORDER BY id ROWS CURRENT ROW) AS running FROM orders;")
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("window with ORDER BY DESC NULLS LAST", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "SELECT id, ROW_NUMBER() OVER (ORDER BY created_at DESC NULLS LAST) AS rn FROM users;")
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("window with multiple ORDER BY columns", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "SELECT id, RANK() OVER (ORDER BY dept ASC, salary DESC) AS ranked FROM employees;")
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("window with multiple PARTITION BY columns", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "SELECT id, ROW_NUMBER() OVER (PARTITION BY dept, team ORDER BY id) AS rn FROM employees;")
		assert.True(t, analysis.ReadOnly)
	})
}

func TestAnalyseQuery_GroupByVariants(t *testing.T) {
	t.Parallel()

	t.Run("GROUP BY with positional reference", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "SELECT status, COUNT(*) FROM users GROUP BY 1;")
		assert.True(t, analysis.ReadOnly)

		require.NotEmpty(t, analysis.OutputColumns)
	})

	t.Run("GROUP BY with function expression", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "SELECT YEAR(created_at), COUNT(*) FROM users GROUP BY YEAR(created_at);")
		assert.True(t, analysis.ReadOnly)
		require.NotEmpty(t, analysis.GroupByColumns)
	})
}

func TestAnalyseQuery_ConvertUsing(t *testing.T) {
	t.Parallel()

	t.Run("CONVERT with USING charset", func(t *testing.T) {
		t.Parallel()
		analysis := analyseQuery(t, "SELECT CONVERT(name USING utf8) FROM users;")
		assert.True(t, analysis.ReadOnly)
		require.Len(t, analysis.OutputColumns, 1)
	})
}

func TestAnalyseQuery_SubscriptExpression(t *testing.T) {
	t.Parallel()

	t.Run("JSON array subscript", func(t *testing.T) {
		t.Parallel()

		analysis := analyseQuery(t, "SELECT data->'$.items'[0] AS first_item FROM documents;")
		assert.True(t, analysis.ReadOnly)
		require.Len(t, analysis.OutputColumns, 1)
	})
}
