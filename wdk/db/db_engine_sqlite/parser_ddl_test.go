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

func applyDDL(t *testing.T, sql string) *querier_dto.CatalogueMutation {
	t.Helper()

	engine := NewSQLiteEngine()
	stmts, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.NotEmpty(t, stmts)

	mutation, err := engine.ApplyDDL(stmts[0])
	require.NoError(t, err)

	return mutation
}

func TestApplyDDL_CreateTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple table with columns",
			sql:  "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "users", m.TableName)
				require.Len(t, m.Columns, 3)

				assert.Equal(t, "id", m.Columns[0].Name)
				assert.Equal(t, querier_dto.TypeCategoryInteger, m.Columns[0].SQLType.Category)
				assert.False(t, m.Columns[0].Nullable, "primary key column should not be nullable")
				assert.True(t, m.Columns[0].HasDefault, "primary key column should have default flag set")

				assert.Equal(t, "name", m.Columns[1].Name)
				assert.Equal(t, querier_dto.TypeCategoryText, m.Columns[1].SQLType.Category)
				assert.False(t, m.Columns[1].Nullable, "NOT NULL column should not be nullable")

				assert.Equal(t, "email", m.Columns[2].Name)
				assert.True(t, m.Columns[2].Nullable, "column without NOT NULL should be nullable")

				assert.Equal(t, []string{"id"}, m.PrimaryKey)
			},
		},
		{
			name: "WITHOUT ROWID table",
			sql:  "CREATE TABLE kv (key TEXT PRIMARY KEY, value BLOB) WITHOUT ROWID",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "kv", m.TableName)
				assert.True(t, m.IsWithoutRowID, "should detect WITHOUT ROWID")
				require.Len(t, m.Columns, 2)
				assert.Equal(t, "key", m.Columns[0].Name)
				assert.Equal(t, querier_dto.TypeCategoryText, m.Columns[0].SQLType.Category)
				assert.Equal(t, []string{"key"}, m.PrimaryKey)
			},
		},
		{
			name: "IF NOT EXISTS is accepted",
			sql:  "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "users", m.TableName)
				require.Len(t, m.Columns, 1)
			},
		},
		{
			name: "type affinity columns",
			sql:  "CREATE TABLE data (a VARCHAR(255), b BIGINT, c DOUBLE, d BOOLEAN, e BLOB)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 5)

				assert.Equal(t, querier_dto.TypeCategoryText, m.Columns[0].SQLType.Category,
					"VARCHAR should have text affinity")

				assert.Equal(t, querier_dto.TypeCategoryInteger, m.Columns[1].SQLType.Category,
					"BIGINT should have integer affinity")

				assert.Equal(t, querier_dto.TypeCategoryFloat, m.Columns[2].SQLType.Category,
					"DOUBLE should have real affinity")

				assert.Equal(t, querier_dto.TypeCategoryBoolean, m.Columns[3].SQLType.Category,
					"BOOLEAN should have boolean category")

				assert.Equal(t, querier_dto.TypeCategoryBytea, m.Columns[4].SQLType.Category,
					"BLOB should have bytea category")
			},
		},
		{
			name: "composite PRIMARY KEY constraint",
			sql:  "CREATE TABLE membership (user_id INTEGER, group_id INTEGER, PRIMARY KEY (user_id, group_id))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
				assert.Equal(t, []string{"user_id", "group_id"}, m.PrimaryKey)
			},
		},
		{
			name: "AUTOINCREMENT column",
			sql:  "CREATE TABLE events (id INTEGER PRIMARY KEY AUTOINCREMENT, payload TEXT)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
				assert.Equal(t, "id", m.Columns[0].Name)
				assert.False(t, m.Columns[0].Nullable, "AUTOINCREMENT column should not be nullable")
				assert.True(t, m.Columns[0].HasDefault, "AUTOINCREMENT column should have default flag set")
				assert.Equal(t, []string{"id"}, m.PrimaryKey)
			},
		},
		{
			name: "CREATE TABLE AS returns mutation without columns",
			sql:  "CREATE TABLE archive AS SELECT id, name FROM users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "archive", m.TableName)

				assert.Empty(t, m.Columns, "CREATE TABLE AS should have no explicit columns")
			},
		},
		{
			name: "column with DEFAULT expression",
			sql:  "CREATE TABLE t (x INTEGER DEFAULT 0, y TEXT DEFAULT 'hello')",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
				assert.True(t, m.Columns[0].HasDefault, "column with DEFAULT should have HasDefault set")
				assert.True(t, m.Columns[1].HasDefault, "column with DEFAULT should have HasDefault set")
			},
		},
		{
			name: "GENERATED ALWAYS AS stored column",
			sql:  "CREATE TABLE products (price REAL, tax REAL, total REAL GENERATED ALWAYS AS (price + tax) STORED)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 3)
				assert.True(t, m.Columns[2].IsGenerated, "generated column should be marked generated")
				assert.Equal(t, querier_dto.GeneratedKindStored, m.Columns[2].GeneratedKind)
			},
		},
		{
			name: "GENERATED ALWAYS AS virtual column",
			sql:  "CREATE TABLE items (a INTEGER, b INTEGER, c AS (a * b) VIRTUAL)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 3)
				assert.True(t, m.Columns[2].IsGenerated, "virtual generated column should be marked generated")
				assert.Equal(t, querier_dto.GeneratedKindVirtual, m.Columns[2].GeneratedKind)
			},
		},
		{
			name: "CHECK constraint on table",
			sql:  "CREATE TABLE orders (id INTEGER PRIMARY KEY, amount REAL, CHECK (amount > 0))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintCheck, m.Constraints[0].Kind)
			},
		},
		{
			name: "multiple UNIQUE constraints",
			sql:  "CREATE TABLE accounts (id INTEGER PRIMARY KEY, email TEXT, username TEXT, UNIQUE (email), UNIQUE (username))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Constraints, 2)
				assert.Equal(t, querier_dto.ConstraintUnique, m.Constraints[0].Kind)
				assert.Equal(t, []string{"email"}, m.Constraints[0].Columns)
				assert.Equal(t, querier_dto.ConstraintUnique, m.Constraints[1].Kind)
				assert.Equal(t, []string{"username"}, m.Constraints[1].Columns)
			},
		},
		{
			name: "FOREIGN KEY table constraint",
			sql:  "CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER, FOREIGN KEY (user_id) REFERENCES users (id))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintForeignKey, m.Constraints[0].Kind)
				assert.Equal(t, "users", m.Constraints[0].ForeignTable)
				assert.Equal(t, []string{"user_id"}, m.Constraints[0].Columns)
				assert.Equal(t, []string{"id"}, m.Constraints[0].ForeignColumns)
			},
		},
		{
			name: "column with COLLATE constraint",
			sql:  "CREATE TABLE names (id INTEGER PRIMARY KEY, name TEXT COLLATE NOCASE)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
				assert.Equal(t, "name", m.Columns[1].Name)
			},
		},
		{
			name: "column with REFERENCES constraint",
			sql:  "CREATE TABLE comments (id INTEGER PRIMARY KEY, post_id INTEGER REFERENCES posts (id))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
				assert.Equal(t, "post_id", m.Columns[1].Name)
			},
		},
		{
			name: "STRICT table mode",
			sql:  "CREATE TABLE strict_tbl (id INTEGER PRIMARY KEY, val TEXT) STRICT",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "strict_tbl", m.TableName)
				require.Len(t, m.Columns, 2)
			},
		},
		{
			name: "named CONSTRAINT on table",
			sql:  "CREATE TABLE t (x INTEGER, CONSTRAINT pk_x PRIMARY KEY (x))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, []string{"x"}, m.PrimaryKey)
			},
		},
		{
			name: "DEFAULT with parenthesised expression",
			sql:  "CREATE TABLE t (created_at TEXT DEFAULT (datetime('now')))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.True(t, m.Columns[0].HasDefault, "DEFAULT with parenthesised expression should set HasDefault")
			},
		},
		{
			name: "column with UNIQUE keyword constraint",
			sql:  "CREATE TABLE t (code TEXT UNIQUE NOT NULL)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.False(t, m.Columns[0].Nullable, "NOT NULL after UNIQUE should set Nullable=false")
			},
		},
		{
			name: "column with CHECK constraint",
			sql:  "CREATE TABLE t (age INTEGER CHECK (age >= 0 AND age <= 200))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "age", m.Columns[0].Name)
			},
		},
		{
			name: "DEFAULT with negative numeric literal",
			sql:  "CREATE TABLE t (balance REAL DEFAULT -0.01)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.True(t, m.Columns[0].HasDefault, "DEFAULT with negative literal should set HasDefault")
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_CreateTableSchemaQualified(t *testing.T) {
	t.Parallel()

	mutation := applyDDL(t, "CREATE TABLE main.events (id INTEGER PRIMARY KEY, name TEXT)")
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
	assert.Equal(t, "events", mutation.TableName)
	require.Len(t, mutation.Columns, 2)
}

func TestApplyDDL_CreateVirtualTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "FTS5 virtual table extracts content columns and rank",
			sql:  "CREATE VIRTUAL TABLE documents USING fts5(title, body, tokenize='porter')",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "documents", m.TableName)
				assert.True(t, m.IsVirtual, "should be marked as virtual")
				assert.Equal(t, "fts5", m.VirtualModuleName)

				require.Len(t, m.Columns, 3)
				assert.Equal(t, "title", m.Columns[0].Name)
				assert.Equal(t, querier_dto.TypeCategoryText, m.Columns[0].SQLType.Category)
				assert.Equal(t, "body", m.Columns[1].Name)
				assert.Equal(t, querier_dto.TypeCategoryText, m.Columns[1].SQLType.Category)
				assert.Equal(t, "rank", m.Columns[2].Name)
				assert.True(t, m.Columns[2].IsGenerated, "rank column should be generated")
				assert.Equal(t, querier_dto.GeneratedKindVirtual, m.Columns[2].GeneratedKind)
			},
		},
		{
			name: "FTS5 with IF NOT EXISTS",
			sql:  "CREATE VIRTUAL TABLE IF NOT EXISTS search USING fts5(content)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "search", m.TableName)
				assert.True(t, m.IsVirtual)
				assert.Equal(t, "fts5", m.VirtualModuleName)

				require.Len(t, m.Columns, 2)
				assert.Equal(t, "content", m.Columns[0].Name)
			},
		},
		{
			name: "RTree virtual table",
			sql:  "CREATE VIRTUAL TABLE spatial USING rtree(id, min_x, max_x, min_y, max_y)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "spatial", m.TableName)
				assert.True(t, m.IsVirtual)
				assert.Equal(t, "rtree", m.VirtualModuleName)
				require.Len(t, m.Columns, 5)
				assert.Equal(t, "id", m.Columns[0].Name)
				assert.False(t, m.Columns[0].Nullable, "rtree id column should not be nullable")
				assert.Equal(t, []string{"id"}, m.PrimaryKey)
			},
		},
		{
			name: "generic virtual table module",
			sql:  "CREATE VIRTUAL TABLE my_table USING custom_module(col1, col2)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "my_table", m.TableName)
				assert.True(t, m.IsVirtual)
				assert.Equal(t, "custom_module", m.VirtualModuleName)
				require.Len(t, m.Columns, 2)
				assert.Equal(t, "col1", m.Columns[0].Name)
				assert.Equal(t, "col2", m.Columns[1].Name)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_AlterTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "ADD COLUMN",
			sql:  "ALTER TABLE users ADD COLUMN email TEXT",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddColumn, m.Kind)
				assert.Equal(t, "users", m.TableName)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "email", m.Columns[0].Name)
				assert.Equal(t, querier_dto.TypeCategoryText, m.Columns[0].SQLType.Category)
			},
		},
		{
			name: "ADD COLUMN without COLUMN keyword",
			sql:  "ALTER TABLE users ADD age INTEGER",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddColumn, m.Kind)
				assert.Equal(t, "users", m.TableName)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "age", m.Columns[0].Name)
				assert.Equal(t, querier_dto.TypeCategoryInteger, m.Columns[0].SQLType.Category)
			},
		},
		{
			name: "RENAME TO",
			sql:  "ALTER TABLE users RENAME TO accounts",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableRenameTable, m.Kind)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "accounts", m.NewName)
			},
		},
		{
			name: "RENAME COLUMN",
			sql:  "ALTER TABLE users RENAME COLUMN name TO full_name",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableRenameColumn, m.Kind)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "name", m.ColumnName)
				assert.Equal(t, "full_name", m.NewName)
			},
		},
		{
			name: "DROP COLUMN",
			sql:  "ALTER TABLE users DROP COLUMN email",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableDropColumn, m.Kind)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "email", m.ColumnName)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_DropTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple DROP TABLE",
			sql:  "DROP TABLE users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTable, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "DROP TABLE IF EXISTS",
			sql:  "DROP TABLE IF EXISTS users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTable, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_CreateView(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple view",
			sql:  "CREATE VIEW active_users AS SELECT id, name FROM users WHERE active = 1",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, m.Kind)
				assert.Equal(t, "active_users", m.TableName)
			},
		},
		{
			name: "view with explicit column list",
			sql:  "CREATE VIEW user_names (user_id, user_name) AS SELECT id, name FROM users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, m.Kind)
				assert.Equal(t, "user_names", m.TableName)
				require.Len(t, m.Columns, 2)
				assert.Equal(t, "user_id", m.Columns[0].Name)
				assert.Equal(t, "user_name", m.Columns[1].Name)
			},
		},
		{
			name: "temporary view with IF NOT EXISTS",
			sql:  "CREATE TEMP VIEW IF NOT EXISTS tmp_view AS SELECT 1",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, m.Kind)
				assert.Equal(t, "tmp_view", m.TableName)
			},
		},
		{
			name: "view with complex SELECT including JOIN",
			sql:  "CREATE VIEW user_posts AS SELECT u.name, p.title FROM users u JOIN posts p ON p.user_id = u.id",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, m.Kind)
				assert.Equal(t, "user_posts", m.TableName)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_DropView(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple DROP VIEW",
			sql:  "DROP VIEW active_users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropView, m.Kind)
				assert.Equal(t, "active_users", m.TableName)
			},
		},
		{
			name: "DROP VIEW IF EXISTS",
			sql:  "DROP VIEW IF EXISTS active_users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropView, m.Kind)
				assert.Equal(t, "active_users", m.TableName)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_CreateIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple index",
			sql:  "CREATE INDEX idx_users_name ON users (name)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "unique index",
			sql:  "CREATE UNIQUE INDEX idx_users_email ON users (email)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "index with IF NOT EXISTS",
			sql:  "CREATE INDEX IF NOT EXISTS idx_users_name ON users (name)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "index on multiple columns",
			sql:  "CREATE INDEX idx_users_name_email ON users (name, email)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_DropIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple DROP INDEX",
			sql:  "DROP INDEX idx_users_name",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropIndex, m.Kind)
			},
		},
		{
			name: "DROP INDEX IF EXISTS",
			sql:  "DROP INDEX IF EXISTS idx_users_name",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropIndex, m.Kind)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_CreateTrigger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "BEFORE INSERT trigger",
			sql:  "CREATE TRIGGER trg_before_insert BEFORE INSERT ON users BEGIN SELECT 1; END",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, m.Kind)
				assert.Equal(t, "trg_before_insert", m.TriggerName)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "AFTER UPDATE trigger",
			sql:  "CREATE TRIGGER trg_after_update AFTER UPDATE ON posts BEGIN SELECT 1; END",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, m.Kind)
				assert.Equal(t, "trg_after_update", m.TriggerName)
				assert.Equal(t, "posts", m.TableName)
			},
		},
		{
			name: "trigger with IF NOT EXISTS",
			sql:  "CREATE TRIGGER IF NOT EXISTS trg_audit AFTER DELETE ON users BEGIN SELECT 1; END",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, m.Kind)
				assert.Equal(t, "trg_audit", m.TriggerName)
				assert.Equal(t, "users", m.TableName)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_DropTrigger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple DROP TRIGGER",
			sql:  "DROP TRIGGER trg_before_insert",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTrigger, m.Kind)
				assert.Equal(t, "trg_before_insert", m.TriggerName)
			},
		},
		{
			name: "DROP TRIGGER IF EXISTS",
			sql:  "DROP TRIGGER IF EXISTS trg_before_insert",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTrigger, m.Kind)
				assert.Equal(t, "trg_before_insert", m.TriggerName)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_CreateTrigger_Variants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "INSTEAD OF trigger on view",
			sql:  "CREATE TRIGGER trg_instead INSTEAD OF INSERT ON users BEGIN SELECT 1; END",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, m.Kind)
				assert.Equal(t, "trg_instead", m.TriggerName)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "UPDATE OF columns trigger",
			sql:  "CREATE TRIGGER trg_update_of AFTER UPDATE OF name, email ON users BEGIN SELECT 1; END",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, m.Kind)
				assert.Equal(t, "trg_update_of", m.TriggerName)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "TEMP TRIGGER",
			sql:  "CREATE TEMP TRIGGER trg_temp BEFORE DELETE ON users BEGIN SELECT 1; END",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, m.Kind)
				assert.Equal(t, "trg_temp", m.TriggerName)
				assert.Equal(t, "users", m.TableName)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_CreateTable_ColumnConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "column with ON CONFLICT on NOT NULL",
			sql:  "CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT NOT NULL ON CONFLICT REPLACE)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
				assert.False(t, m.Columns[1].Nullable)
			},
		},
		{
			name: "column with ON CONFLICT on UNIQUE",
			sql:  "CREATE TABLE t (id INTEGER PRIMARY KEY, code TEXT UNIQUE ON CONFLICT IGNORE)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
			},
		},
		{
			name: "column with CONSTRAINT name prefix",
			sql:  "CREATE TABLE t (id INTEGER, CONSTRAINT pk PRIMARY KEY (id))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, []string{"id"}, m.PrimaryKey)
			},
		},
		{
			name: "column with inline REFERENCES with ON DELETE CASCADE",
			sql:  "CREATE TABLE t (user_id INTEGER REFERENCES users (id) ON DELETE CASCADE)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "user_id", m.Columns[0].Name)
			},
		},
		{
			name: "FOREIGN KEY with ON UPDATE and ON DELETE",
			sql:  "CREATE TABLE t (id INTEGER, uid INTEGER, FOREIGN KEY (uid) REFERENCES users (id) ON DELETE SET NULL ON UPDATE CASCADE)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintForeignKey, m.Constraints[0].Kind)
				assert.Equal(t, "users", m.Constraints[0].ForeignTable)
				assert.Equal(t, []string{"id"}, m.Constraints[0].ForeignColumns)
			},
		},
		{
			name: "column with type modifiers applied to text",
			sql:  "CREATE TABLE t (name VARCHAR(100))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryText, m.Columns[0].SQLType.Category)
				require.NotNil(t, m.Columns[0].SQLType.Length, "text type with modifier should set Length")
				assert.Equal(t, 100, *m.Columns[0].SQLType.Length)
			},
		},
		{
			name: "column with DECIMAL precision and scale modifiers",
			sql:  "CREATE TABLE t (amount DECIMAL(10, 2))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryDecimal, m.Columns[0].SQLType.Category)
				require.NotNil(t, m.Columns[0].SQLType.Precision, "DECIMAL should set Precision")
				assert.Equal(t, 10, *m.Columns[0].SQLType.Precision)
				require.NotNil(t, m.Columns[0].SQLType.Scale, "DECIMAL(p,s) should set Scale")
				assert.Equal(t, 2, *m.Columns[0].SQLType.Scale)
			},
		},
		{
			name: "column with no explicit type defaults to blob",
			sql:  "CREATE TABLE t (id INTEGER PRIMARY KEY, data)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)

				assert.Equal(t, querier_dto.TypeCategoryBytea, m.Columns[1].SQLType.Category)
			},
		},
		{
			name: "named CHECK constraint on table",
			sql:  "CREATE TABLE t (x INTEGER, CONSTRAINT chk_positive CHECK (x > 0))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintCheck, m.Constraints[0].Kind)
				assert.Equal(t, "chk_positive", m.Constraints[0].Name)
			},
		},
		{
			name: "named UNIQUE constraint on table",
			sql:  "CREATE TABLE t (email TEXT, CONSTRAINT uq_email UNIQUE (email))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintUnique, m.Constraints[0].Kind)
				assert.Equal(t, "uq_email", m.Constraints[0].Name)
				assert.Equal(t, []string{"email"}, m.Constraints[0].Columns)
			},
		},
		{
			name: "named FOREIGN KEY constraint",
			sql:  "CREATE TABLE t (uid INTEGER, CONSTRAINT fk_user FOREIGN KEY (uid) REFERENCES users (id))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintForeignKey, m.Constraints[0].Kind)
				assert.Equal(t, "fk_user", m.Constraints[0].Name)
			},
		},
		{
			name: "multi-word type name",
			sql:  "CREATE TABLE t (x DOUBLE PRECISION)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)

				assert.Equal(t, querier_dto.TypeCategoryFloat, m.Columns[0].SQLType.Category)
			},
		},
		{
			name: "PRIMARY KEY ASC column constraint",
			sql:  "CREATE TABLE t (id INTEGER PRIMARY KEY ASC)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, []string{"id"}, m.PrimaryKey)
			},
		},
		{
			name: "PRIMARY KEY DESC column constraint",
			sql:  "CREATE TABLE t (id INTEGER PRIMARY KEY DESC)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, []string{"id"}, m.PrimaryKey)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_AlterTable_ImplicitColumnKeyword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "RENAME COLUMN without COLUMN keyword",
			sql:  "ALTER TABLE users RENAME name TO full_name",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableRenameColumn, m.Kind)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "name", m.ColumnName)
				assert.Equal(t, "full_name", m.NewName)
			},
		},
		{
			name: "DROP COLUMN without COLUMN keyword",
			sql:  "ALTER TABLE users DROP email",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableDropColumn, m.Kind)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "email", m.ColumnName)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_CreateVirtualTable_ModuleArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "FTS5 with content option using equals syntax",
			sql:  "CREATE VIRTUAL TABLE search USING fts5(title, body, content=posts, content_rowid=id)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.True(t, m.IsVirtual)
				assert.Equal(t, "fts5", m.VirtualModuleName)

				require.Len(t, m.Columns, 3)
				assert.Equal(t, "title", m.Columns[0].Name)
				assert.Equal(t, "body", m.Columns[1].Name)
				assert.Equal(t, "rank", m.Columns[2].Name)
			},
		},
		{
			name: "generic virtual table with key=value options skipped",
			sql:  "CREATE VIRTUAL TABLE vt USING mymod(col1, opt1=val1, col2)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.True(t, m.IsVirtual)

				require.Len(t, m.Columns, 2)
				assert.Equal(t, "col1", m.Columns[0].Name)
				assert.Equal(t, "col2", m.Columns[1].Name)
			},
		},
		{
			name: "virtual table without arguments",
			sql:  "CREATE VIRTUAL TABLE vt USING mymod",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.True(t, m.IsVirtual)
				assert.Equal(t, "mymod", m.VirtualModuleName)
				assert.Empty(t, m.Columns)
			},
		},
		{
			name: "rtree_i32 virtual table",
			sql:  "CREATE VIRTUAL TABLE coords USING rtree_i32(id, x0, x1, y0, y1)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.True(t, m.IsVirtual)
				assert.Equal(t, "rtree_i32", m.VirtualModuleName)
				require.Len(t, m.Columns, 5)
				assert.Equal(t, "id", m.Columns[0].Name)
				assert.Equal(t, []string{"id"}, m.PrimaryKey)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			require.NotNil(t, mutation)
			testCase.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_NonDDL(t *testing.T) {
	t.Parallel()

	mutation := applyDDL(t, "SELECT 1")

	assert.Nil(t, mutation, "non-DDL statement should return nil mutation")
}

func TestEngine_Accessors(t *testing.T) {
	t.Parallel()

	engine := NewSQLiteEngine()

	t.Run("SupportedDirectivePrefixes", func(t *testing.T) {
		t.Parallel()
		prefixes := engine.SupportedDirectivePrefixes()
		require.NotEmpty(t, prefixes)
	})

	t.Run("SupportedExpressions", func(t *testing.T) {
		t.Parallel()
		features := engine.SupportedExpressions()
		assert.NotZero(t, features)
	})

	t.Run("TableValuedFunctionColumns known", func(t *testing.T) {
		t.Parallel()
		columns := engine.TableValuedFunctionColumns("json_each")
		require.NotEmpty(t, columns)
	})

	t.Run("TableValuedFunctionColumns unknown", func(t *testing.T) {
		t.Parallel()
		columns := engine.TableValuedFunctionColumns("nonexistent_function")
		assert.Nil(t, columns)
	})

	t.Run("CommentStyle", func(t *testing.T) {
		t.Parallel()
		style := engine.CommentStyle()
		assert.NotZero(t, style)
	})
}
