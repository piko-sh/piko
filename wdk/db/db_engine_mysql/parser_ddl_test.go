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

func applyDDL(t *testing.T, sql string) *querier_dto.CatalogueMutation {
	t.Helper()
	engine := NewMySQLEngine()
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
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple table with columns",
			sql:  "CREATE TABLE users (id INT, name VARCHAR(255), email TEXT);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, "users", mutation.TableName)
				require.Len(t, mutation.Columns, 3)
				assert.Equal(t, "id", mutation.Columns[0].Name)
				assert.Equal(t, "name", mutation.Columns[1].Name)
				assert.Equal(t, "email", mutation.Columns[2].Name)
			},
		},
		{
			name: "auto increment column has default",
			sql:  "CREATE TABLE orders (id INT AUTO_INCREMENT, total DECIMAL(10,2));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, "orders", mutation.TableName)
				require.Len(t, mutation.Columns, 2)
				assert.True(t, mutation.Columns[0].HasDefault, "AUTO_INCREMENT column should have default")
			},
		},
		{
			name: "not null column",
			sql:  "CREATE TABLE items (id INT NOT NULL, label VARCHAR(100) NOT NULL);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				require.Len(t, mutation.Columns, 2)
				assert.False(t, mutation.Columns[0].Nullable, "NOT NULL column should not be nullable")
				assert.False(t, mutation.Columns[1].Nullable, "NOT NULL column should not be nullable")
			},
		},
		{
			name: "default value sets has default flag",
			sql:  "CREATE TABLE config (key_name VARCHAR(255), value TEXT DEFAULT 'empty');",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				require.Len(t, mutation.Columns, 2)
				assert.True(t, mutation.Columns[1].HasDefault, "column with DEFAULT should have HasDefault set")
			},
		},
		{
			name: "inline primary key",
			sql:  "CREATE TABLE accounts (id BIGINT PRIMARY KEY, owner VARCHAR(255));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, []string{"id"}, mutation.PrimaryKey)

				assert.False(t, mutation.Columns[0].Nullable)
				assert.True(t, mutation.Columns[0].HasDefault)
			},
		},
		{
			name: "table-level primary key constraint",
			sql:  "CREATE TABLE composite_pk (a INT, b INT, c TEXT, PRIMARY KEY (a, b));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, []string{"a", "b"}, mutation.PrimaryKey)
				require.Len(t, mutation.Columns, 3)
			},
		},
		{
			name: "engine option after closing parenthesis",
			sql:  "CREATE TABLE logs (id INT, message TEXT) ENGINE=InnoDB;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, "logs", mutation.TableName)
				require.Len(t, mutation.Columns, 2)
			},
		},
		{
			name: "if not exists is accepted",
			sql:  "CREATE TABLE IF NOT EXISTS sessions (id INT, token VARCHAR(64));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, "sessions", mutation.TableName)
				require.Len(t, mutation.Columns, 2)
			},
		},
		{
			name: "temporary table",
			sql:  "CREATE TEMPORARY TABLE tmp_data (id INT, payload BLOB);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, "tmp_data", mutation.TableName)
			},
		},
		{
			name: "unique constraint produces constraint entry",
			sql:  "CREATE TABLE emails (id INT, addr VARCHAR(255), UNIQUE KEY uk_addr (addr));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				require.NotEmpty(t, mutation.Constraints)
				assert.Equal(t, querier_dto.ConstraintUnique, mutation.Constraints[0].Kind)
				assert.Equal(t, []string{"addr"}, mutation.Constraints[0].Columns)
			},
		},
		{
			name: "foreign key constraint",
			sql: `CREATE TABLE orders (
				id INT,
				user_id INT,
				FOREIGN KEY (user_id) REFERENCES users(id)
			);`,
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				require.NotEmpty(t, mutation.Constraints)
				fk := mutation.Constraints[0]
				assert.Equal(t, querier_dto.ConstraintForeignKey, fk.Kind)
				assert.Equal(t, []string{"user_id"}, fk.Columns)
				assert.Equal(t, "users", fk.ForeignTable)
				assert.Equal(t, []string{"id"}, fk.ForeignColumns)
			},
		},
		{
			name: "check constraint",
			sql:  "CREATE TABLE products (id INT, price DECIMAL(10,2), CHECK (price > 0));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				require.NotEmpty(t, mutation.Constraints)
				assert.Equal(t, querier_dto.ConstraintCheck, mutation.Constraints[0].Kind)
			},
		},
		{
			name: "named constraint",
			sql: `CREATE TABLE invoices (
				id INT,
				amount DECIMAL(10,2),
				CONSTRAINT chk_amount CHECK (amount >= 0)
			);`,
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				require.NotEmpty(t, mutation.Constraints)
				assert.Equal(t, "chk_amount", mutation.Constraints[0].Name)
				assert.Equal(t, querier_dto.ConstraintCheck, mutation.Constraints[0].Kind)
			},
		},
		{
			name: "schema-qualified table name",
			sql:  "CREATE TABLE mydb.users (id INT);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, "mydb", mutation.SchemaName)
				assert.Equal(t, "users", mutation.TableName)
			},
		},
		{
			name: "create table as select",
			sql:  "CREATE TABLE archive AS SELECT * FROM orders;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, "archive", mutation.TableName)

				assert.Empty(t, mutation.Columns)
			},
		},
		{
			name: "nullable column explicitly marked NULL",
			sql:  "CREATE TABLE widgets (id INT, description TEXT NULL);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.True(t, mutation.Columns[1].Nullable, "explicitly NULL column should be nullable")
			},
		},
		{
			name: "multiple table options",
			sql:  "CREATE TABLE data (id INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, "data", mutation.TableName)
			},
		},

		{
			name: "tinyint column",
			sql:  "CREATE TABLE t (val TINYINT);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryInteger, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "tinyint", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "mediumint column",
			sql:  "CREATE TABLE t (val MEDIUMINT);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryInteger, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "mediumint", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "bigint column",
			sql:  "CREATE TABLE t (val BIGINT);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryInteger, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "bigint", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "float column",
			sql:  "CREATE TABLE t (val FLOAT);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryFloat, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "float", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "double column",
			sql:  "CREATE TABLE t (val DOUBLE);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryFloat, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "double", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "double precision column",
			sql:  "CREATE TABLE t (val DOUBLE PRECISION);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryFloat, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "double", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "decimal column with precision and scale",
			sql:  "CREATE TABLE t (val DECIMAL(10,2));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryDecimal, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "decimal", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "date column",
			sql:  "CREATE TABLE t (val DATE);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryTemporal, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "date", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "datetime column",
			sql:  "CREATE TABLE t (val DATETIME);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryTemporal, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "datetime", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "timestamp column",
			sql:  "CREATE TABLE t (val TIMESTAMP);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryTemporal, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "timestamp", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "char column",
			sql:  "CREATE TABLE t (val CHAR(36));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryText, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "char", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "varchar column",
			sql:  "CREATE TABLE t (val VARCHAR(255));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryText, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "varchar", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "text column",
			sql:  "CREATE TABLE t (val TEXT);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryText, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "text", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "blob column",
			sql:  "CREATE TABLE t (val BLOB);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryBytea, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "blob", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "enum column with values",
			sql:  "CREATE TABLE t (status ENUM('active', 'inactive', 'pending'));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryEnum, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "enum", mutation.Columns[0].SQLType.EngineName)
				assert.Equal(t, []string{"active", "inactive", "pending"}, mutation.Columns[0].SQLType.EnumValues)
			},
		},
		{
			name: "set column with values",
			sql:  "CREATE TABLE t (tags SET('a', 'b', 'c'));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, "set", mutation.Columns[0].SQLType.EngineName)
				assert.Equal(t, []string{"a", "b", "c"}, mutation.Columns[0].SQLType.EnumValues)
			},
		},
		{
			name: "json column",
			sql:  "CREATE TABLE t (data JSON);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryJSON, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "json", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "binary column",
			sql:  "CREATE TABLE t (val BINARY(16));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryBytea, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "binary", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "varbinary column",
			sql:  "CREATE TABLE t (val VARBINARY(256));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryBytea, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "varbinary", mutation.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "unsigned integer column",
			sql:  "CREATE TABLE t (val INT UNSIGNED);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryInteger, mutation.Columns[0].SQLType.Category)
				assert.Equal(t, "int unsigned", mutation.Columns[0].SQLType.EngineName)
			},
		},

		{
			name: "multiple keys: primary, unique, and index",
			sql: `CREATE TABLE articles (
				id INT AUTO_INCREMENT,
				slug VARCHAR(255),
				title VARCHAR(255),
				body TEXT,
				PRIMARY KEY (id),
				UNIQUE KEY uk_slug (slug),
				KEY idx_title (title)
			);`,
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, []string{"id"}, mutation.PrimaryKey)
				require.Len(t, mutation.Columns, 4)

				require.NotEmpty(t, mutation.Constraints)
				assert.Equal(t, querier_dto.ConstraintUnique, mutation.Constraints[0].Kind)
				assert.Equal(t, []string{"slug"}, mutation.Constraints[0].Columns)
			},
		},

		{
			name: "table options including AUTO_INCREMENT value",
			sql:  "CREATE TABLE counters (id INT) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, "counters", mutation.TableName)
				require.Len(t, mutation.Columns, 1)
			},
		},

		{
			name: "generated stored column",
			sql:  "CREATE TABLE products (price DECIMAL(10,2), tax DECIMAL(10,2), total DECIMAL(10,2) GENERATED ALWAYS AS (price + tax) STORED);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 3)
				assert.True(t, mutation.Columns[2].IsGenerated)
				assert.Equal(t, querier_dto.GeneratedKindStored, mutation.Columns[2].GeneratedKind)
			},
		},
		{
			name: "generated virtual column",
			sql:  "CREATE TABLE shapes (width INT, height INT, area INT GENERATED ALWAYS AS (width * height) VIRTUAL);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 3)
				assert.True(t, mutation.Columns[2].IsGenerated)
				assert.Equal(t, querier_dto.GeneratedKindVirtual, mutation.Columns[2].GeneratedKind)
			},
		},

		{
			name: "column with ON UPDATE CURRENT_TIMESTAMP",
			sql:  "CREATE TABLE events (id INT, updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.Equal(t, "updated_at", mutation.Columns[1].Name)
				assert.True(t, mutation.Columns[1].HasDefault)
			},
		},

		{
			name: "column with comment",
			sql:  `CREATE TABLE notes (id INT, body TEXT COMMENT 'main content');`,
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.Equal(t, "body", mutation.Columns[1].Name)
			},
		},

		{
			name: "column with COLLATE",
			sql:  "CREATE TABLE labels (id INT, name VARCHAR(100) COLLATE utf8mb4_bin);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.Equal(t, "name", mutation.Columns[1].Name)
			},
		},

		{
			name: "column with inline UNIQUE KEY",
			sql:  "CREATE TABLE logins (id INT, token VARCHAR(64) UNIQUE KEY);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.Equal(t, "token", mutation.Columns[1].Name)
			},
		},

		{
			name: "column with inline CHECK constraint",
			sql:  "CREATE TABLE scores (id INT, value INT CHECK (value >= 0));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.Equal(t, "value", mutation.Columns[1].Name)
			},
		},

		{
			name: "foreign key with ON DELETE CASCADE ON UPDATE SET NULL",
			sql: `CREATE TABLE comments (
				id INT,
				post_id INT,
				FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE ON UPDATE SET NULL
			);`,
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.NotEmpty(t, mutation.Constraints)
				fk := mutation.Constraints[0]
				assert.Equal(t, querier_dto.ConstraintForeignKey, fk.Kind)
				assert.Equal(t, "posts", fk.ForeignTable)
				assert.Equal(t, []string{"id"}, fk.ForeignColumns)
			},
		},

		{
			name: "column with inline REFERENCES",
			sql:  "CREATE TABLE orders (id INT, user_id INT REFERENCES users(id));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.Equal(t, "user_id", mutation.Columns[1].Name)
			},
		},

		{
			name: "column with DEFAULT function call",
			sql:  "CREATE TABLE events (id INT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.True(t, mutation.Columns[1].HasDefault)
			},
		},
		{
			name: "column with DEFAULT parenthesised expression",
			sql:  "CREATE TABLE config (id INT, data JSON DEFAULT (JSON_OBJECT()));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.True(t, mutation.Columns[1].HasDefault)
			},
		},

		{
			name: "named foreign key constraint",
			sql: `CREATE TABLE orders (
				id INT,
				user_id INT,
				CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id)
			);`,
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.NotEmpty(t, mutation.Constraints)
				assert.Equal(t, "fk_user", mutation.Constraints[0].Name)
				assert.Equal(t, querier_dto.ConstraintForeignKey, mutation.Constraints[0].Kind)
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
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "add column",
			sql:  "ALTER TABLE users ADD COLUMN age INT NOT NULL;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddColumn, mutation.Kind)
				assert.Equal(t, "users", mutation.TableName)
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, "age", mutation.Columns[0].Name)
				assert.False(t, mutation.Columns[0].Nullable)
			},
		},
		{
			name: "add column without COLUMN keyword",
			sql:  "ALTER TABLE users ADD email VARCHAR(255);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddColumn, mutation.Kind)
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, "email", mutation.Columns[0].Name)
			},
		},
		{
			name: "drop column",
			sql:  "ALTER TABLE users DROP COLUMN age;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableDropColumn, mutation.Kind)
				assert.Equal(t, "users", mutation.TableName)
				assert.Equal(t, "age", mutation.ColumnName)
			},
		},
		{
			name: "drop column without COLUMN keyword",
			sql:  "ALTER TABLE users DROP age;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableDropColumn, mutation.Kind)
				assert.Equal(t, "age", mutation.ColumnName)
			},
		},
		{
			name: "modify column",
			sql:  "ALTER TABLE users MODIFY COLUMN name VARCHAR(500) NOT NULL;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAlterColumn, mutation.Kind)
				assert.Equal(t, "users", mutation.TableName)
				assert.Equal(t, "name", mutation.ColumnName)
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, "name", mutation.Columns[0].Name)
				assert.False(t, mutation.Columns[0].Nullable)
			},
		},
		{
			name: "change column with same name acts as alter",
			sql:  "ALTER TABLE users CHANGE COLUMN name name TEXT;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAlterColumn, mutation.Kind)
				assert.Equal(t, "name", mutation.ColumnName)
				require.Len(t, mutation.Columns, 1)
			},
		},
		{
			name: "change column with different name produces rename",
			sql:  "ALTER TABLE users CHANGE COLUMN name full_name VARCHAR(255);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableRenameColumn, mutation.Kind)
				assert.Equal(t, "name", mutation.ColumnName)
				assert.Equal(t, "full_name", mutation.NewName)
			},
		},
		{
			name: "rename table with RENAME TO",
			sql:  "ALTER TABLE users RENAME TO customers;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableRenameTable, mutation.Kind)
				assert.Equal(t, "users", mutation.TableName)
				assert.Equal(t, "customers", mutation.NewName)
			},
		},
		{
			name: "rename column",
			sql:  "ALTER TABLE users RENAME COLUMN email TO email_address;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableRenameColumn, mutation.Kind)
				assert.Equal(t, "email", mutation.ColumnName)
				assert.Equal(t, "email_address", mutation.NewName)
			},
		},
		{
			name: "add constraint (unique key)",
			sql:  "ALTER TABLE users ADD UNIQUE KEY uk_email (email);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddConstraint, mutation.Kind)
				require.NotEmpty(t, mutation.Constraints)
				assert.Equal(t, querier_dto.ConstraintUnique, mutation.Constraints[0].Kind)
			},
		},
		{
			name: "add foreign key constraint",
			sql:  "ALTER TABLE orders ADD FOREIGN KEY (user_id) REFERENCES users(id);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddConstraint, mutation.Kind)
				require.NotEmpty(t, mutation.Constraints)
				fk := mutation.Constraints[0]
				assert.Equal(t, querier_dto.ConstraintForeignKey, fk.Kind)
				assert.Equal(t, "users", fk.ForeignTable)
			},
		},
		{
			name: "drop constraint",
			sql:  "ALTER TABLE orders DROP CONSTRAINT chk_amount;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableDropConstraint, mutation.Kind)
				assert.Equal(t, "chk_amount", mutation.ConstraintName)
			},
		},
		{
			name: "schema-qualified alter table",
			sql:  "ALTER TABLE mydb.users ADD COLUMN bio TEXT;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddColumn, mutation.Kind)
				assert.Equal(t, "mydb", mutation.SchemaName)
				assert.Equal(t, "users", mutation.TableName)
			},
		},
		{
			name: "modify column without COLUMN keyword",
			sql:  "ALTER TABLE users MODIFY name TEXT NOT NULL;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAlterColumn, mutation.Kind)
				assert.Equal(t, "name", mutation.ColumnName)
				require.Len(t, mutation.Columns, 1)
				assert.False(t, mutation.Columns[0].Nullable)
			},
		},
		{
			name: "add column with AFTER clause",
			sql:  "ALTER TABLE users ADD COLUMN bio TEXT AFTER name;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddColumn, mutation.Kind)
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, "bio", mutation.Columns[0].Name)
			},
		},
		{
			name: "add column with FIRST clause",
			sql:  "ALTER TABLE users ADD COLUMN id INT FIRST;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddColumn, mutation.Kind)
				require.Len(t, mutation.Columns, 1)
				assert.Equal(t, "id", mutation.Columns[0].Name)
			},
		},
		{
			name: "change column with AFTER clause",
			sql:  "ALTER TABLE users CHANGE COLUMN name full_name VARCHAR(500) AFTER id;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableRenameColumn, mutation.Kind)
				assert.Equal(t, "name", mutation.ColumnName)
				assert.Equal(t, "full_name", mutation.NewName)
			},
		},
		{
			name: "rename table without TO keyword",
			sql:  "ALTER TABLE users RENAME customers;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableRenameTable, mutation.Kind)
				assert.Equal(t, "users", mutation.TableName)
				assert.Equal(t, "customers", mutation.NewName)
			},
		},
		{
			name: "add check constraint via ALTER TABLE",
			sql:  "ALTER TABLE products ADD CHECK (price > 0);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddConstraint, mutation.Kind)
				require.NotEmpty(t, mutation.Constraints)
				assert.Equal(t, querier_dto.ConstraintCheck, mutation.Constraints[0].Kind)
			},
		},
		{
			name: "drop check constraint",
			sql:  "ALTER TABLE products DROP CHECK chk_price;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableDropConstraint, mutation.Kind)
				assert.Equal(t, "chk_price", mutation.ConstraintName)
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

func TestApplyDDL_AlterTable_NonColumnActions(t *testing.T) {
	t.Parallel()

	t.Run("misc action like ENGINE is consumed without error", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "ALTER TABLE users ENGINE=InnoDB;")
		assert.Nil(t, mutation, "misc ALTER TABLE action should return nil")
	})

	t.Run("add index produces add-constraint mutation", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "ALTER TABLE users ADD INDEX idx_name (name);")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAddConstraint, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
	})

	nilTests := []struct {
		name string
		sql  string
	}{
		{
			name: "drop index returns nil mutation",
			sql:  "ALTER TABLE users DROP INDEX idx_name;",
		},
		{
			name: "drop primary key returns nil mutation",
			sql:  "ALTER TABLE users DROP PRIMARY KEY;",
		},
		{
			name: "drop key returns nil mutation",
			sql:  "ALTER TABLE users DROP KEY idx_name;",
		},
		{
			name: "drop foreign key returns nil mutation",
			sql:  "ALTER TABLE orders DROP FOREIGN KEY fk_user;",
		},
	}

	for _, testCase := range nilTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			assert.Nil(t, mutation, "non-column-affecting ALTER TABLE action should return nil")
		})
	}
}

func TestApplyDDL_DropTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple drop table",
			sql:  "DROP TABLE users;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTable, mutation.Kind)
				assert.Equal(t, "users", mutation.TableName)
			},
		},
		{
			name: "drop table if exists",
			sql:  "DROP TABLE IF EXISTS users;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTable, mutation.Kind)
				assert.Equal(t, "users", mutation.TableName)
			},
		},
		{
			name: "drop table cascade keyword is tolerated",
			sql:  "DROP TABLE users CASCADE;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTable, mutation.Kind)
				assert.Equal(t, "users", mutation.TableName)
			},
		},
		{
			name: "schema-qualified drop table",
			sql:  "DROP TABLE mydb.users;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTable, mutation.Kind)
				assert.Equal(t, "mydb", mutation.SchemaName)
				assert.Equal(t, "users", mutation.TableName)
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
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple create view",
			sql:  "CREATE VIEW active_users AS SELECT * FROM users WHERE active = 1;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
				assert.Equal(t, "active_users", mutation.TableName)
			},
		},
		{
			name: "create or replace view",
			sql:  "CREATE OR REPLACE VIEW v_summary AS SELECT COUNT(*) FROM orders;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
				assert.Equal(t, "v_summary", mutation.TableName)
			},
		},
		{
			name: "create view if not exists",
			sql:  "CREATE VIEW IF NOT EXISTS v_active AS SELECT 1;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
				assert.Equal(t, "v_active", mutation.TableName)
			},
		},
		{
			name: "schema-qualified create view",
			sql:  "CREATE VIEW reporting.monthly_sales AS SELECT 1;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
				assert.Equal(t, "reporting", mutation.SchemaName)
				assert.Equal(t, "monthly_sales", mutation.TableName)
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
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple drop view",
			sql:  "DROP VIEW active_users;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropView, mutation.Kind)
				assert.Equal(t, "active_users", mutation.TableName)
			},
		},
		{
			name: "drop view if exists",
			sql:  "DROP VIEW IF EXISTS v_summary;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropView, mutation.Kind)
				assert.Equal(t, "v_summary", mutation.TableName)
			},
		},
		{
			name: "schema-qualified drop view",
			sql:  "DROP VIEW reporting.monthly_sales;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropView, mutation.Kind)
				assert.Equal(t, "reporting", mutation.SchemaName)
				assert.Equal(t, "monthly_sales", mutation.TableName)
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
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple create index",
			sql:  "CREATE INDEX idx_name ON users (name);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, mutation.Kind)
				assert.Equal(t, "users", mutation.TableName)
				assert.Equal(t, "idx_name", mutation.NewName)
			},
		},
		{
			name: "create unique index",
			sql:  "CREATE UNIQUE INDEX idx_email ON users (email);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, mutation.Kind)
				assert.Equal(t, "idx_email", mutation.NewName)
				assert.Equal(t, "users", mutation.TableName)
			},
		},
		{
			name: "create index with USING clause",
			sql:  "CREATE INDEX idx_hash ON users USING BTREE (email);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, mutation.Kind)
				assert.Equal(t, "idx_hash", mutation.NewName)
				assert.Equal(t, "users", mutation.TableName)
			},
		},
		{
			name: "schema-qualified create index",
			sql:  "CREATE INDEX idx_name ON mydb.users (name);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, mutation.Kind)
				assert.Equal(t, "mydb", mutation.SchemaName)
				assert.Equal(t, "users", mutation.TableName)
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
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "drop index on table",
			sql:  "DROP INDEX idx_name ON users;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropIndex, mutation.Kind)
				assert.Equal(t, "idx_name", mutation.NewName)
				assert.Equal(t, "users", mutation.TableName)
			},
		},
		{
			name: "drop index if exists",
			sql:  "DROP INDEX IF EXISTS idx_email ON users;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropIndex, mutation.Kind)
				assert.Equal(t, "idx_email", mutation.NewName)
				assert.Equal(t, "users", mutation.TableName)
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

func TestApplyDDL_CreateFunction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple function with return type",
			sql:  "CREATE FUNCTION add_numbers(a INT, b INT) RETURNS INT DETERMINISTIC RETURN a + b;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, "add_numbers", mutation.FunctionSignature.Name)
				assert.Len(t, mutation.FunctionSignature.Arguments, 2)
				assert.Equal(t, querier_dto.TypeCategoryInteger, mutation.FunctionSignature.ReturnType.Category)
				assert.Equal(t, querier_dto.DataAccessReadOnly, mutation.FunctionSignature.DataAccess,
					"DETERMINISTIC should imply read-only data access")
			},
		},
		{
			name: "function with READS SQL DATA attribute",
			sql:  "CREATE FUNCTION get_count() RETURNS INT READS SQL DATA RETURN 0;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, querier_dto.DataAccessReadOnly, mutation.FunctionSignature.DataAccess)
			},
		},
		{
			name: "function with MODIFIES SQL DATA attribute",
			sql:  "CREATE FUNCTION do_work() RETURNS INT MODIFIES SQL DATA RETURN 0;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, querier_dto.DataAccessModifiesData, mutation.FunctionSignature.DataAccess)
			},
		},
		{
			name: "schema-qualified function",
			sql:  "CREATE FUNCTION mydb.my_func() RETURNS INT DETERMINISTIC RETURN 1;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
				assert.Equal(t, "mydb", mutation.SchemaName)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, "my_func", mutation.FunctionSignature.Name)
				assert.Equal(t, "mydb", mutation.FunctionSignature.Schema)
			},
		},
		{
			name: "function with NO SQL attribute",
			sql:  "CREATE FUNCTION pure_calc(x INT) RETURNS INT NO SQL RETURN x * 2;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, querier_dto.DataAccessReadOnly, mutation.FunctionSignature.DataAccess)
			},
		},
		{
			name: "function with CONTAINS SQL attribute",
			sql:  "CREATE FUNCTION wrapper() RETURNS INT CONTAINS SQL RETURN 1;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, "wrapper", mutation.FunctionSignature.Name)
			},
		},
		{
			name: "function with LANGUAGE SQL and COMMENT",
			sql:  "CREATE FUNCTION documented() RETURNS INT LANGUAGE SQL COMMENT 'returns one' DETERMINISTIC RETURN 1;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, "sql", mutation.FunctionSignature.Language)
			},
		},
		{
			name: "function with NOT DETERMINISTIC",
			sql:  "CREATE FUNCTION rng() RETURNS INT NOT DETERMINISTIC RETURN FLOOR(RAND() * 100);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, "rng", mutation.FunctionSignature.Name)
			},
		},
		{
			name: "function with BEGIN/END body",
			sql: `CREATE FUNCTION complex_fn(x INT) RETURNS INT DETERMINISTIC
				BEGIN
					DECLARE result INT;
					SET result = x * 2;
					RETURN result;
				END;`,
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, "complex_fn", mutation.FunctionSignature.Name)
				assert.NotEmpty(t, mutation.FunctionSignature.BodySQL)
			},
		},
		{
			name: "function with SQL SECURITY DEFINER",
			sql:  "CREATE FUNCTION secure_fn() RETURNS INT SQL SECURITY DEFINER DETERMINISTIC RETURN 1;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, "secure_fn", mutation.FunctionSignature.Name)
			},
		},

		{
			name: "create procedure with IN/OUT parameters",
			sql: `CREATE PROCEDURE do_something(IN x INT, OUT result INT)
				BEGIN
					SET result = x + 1;
				END;`,
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, "do_something", mutation.FunctionSignature.Name)
				assert.Len(t, mutation.FunctionSignature.Arguments, 2)
			},
		},
		{
			name: "create procedure with MODIFIES SQL DATA",
			sql: `CREATE PROCEDURE insert_row(IN val INT) MODIFIES SQL DATA
				BEGIN
					INSERT INTO data (value) VALUES (val);
				END;`,
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, querier_dto.DataAccessModifiesData, mutation.FunctionSignature.DataAccess)
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

func TestApplyDDL_DropFunction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple drop function",
			sql:  "DROP FUNCTION add_numbers;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, "add_numbers", mutation.FunctionSignature.Name)
			},
		},
		{
			name: "drop function if exists",
			sql:  "DROP FUNCTION IF EXISTS my_func;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, "my_func", mutation.FunctionSignature.Name)
			},
		},
		{
			name: "drop procedure",
			sql:  "DROP PROCEDURE my_proc;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropFunction, mutation.Kind)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, "my_proc", mutation.FunctionSignature.Name)
			},
		},
		{
			name: "schema-qualified drop function",
			sql:  "DROP FUNCTION mydb.add_numbers;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropFunction, mutation.Kind)
				assert.Equal(t, "mydb", mutation.SchemaName)
				require.NotNil(t, mutation.FunctionSignature)
				assert.Equal(t, "add_numbers", mutation.FunctionSignature.Name)
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

func TestApplyDDL_CreateSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "create database",
			sql:  "CREATE DATABASE myapp;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateSchema, mutation.Kind)
				assert.Equal(t, "myapp", mutation.SchemaName)
			},
		},
		{
			name: "create schema",
			sql:  "CREATE SCHEMA analytics;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateSchema, mutation.Kind)
				assert.Equal(t, "analytics", mutation.SchemaName)
			},
		},
		{
			name: "create database if not exists",
			sql:  "CREATE DATABASE IF NOT EXISTS myapp;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateSchema, mutation.Kind)
				assert.Equal(t, "myapp", mutation.SchemaName)
			},
		},
		{
			name: "create database with character set and collation",
			sql:  "CREATE DATABASE myapp DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateSchema, mutation.Kind)
				assert.Equal(t, "myapp", mutation.SchemaName)
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

func TestApplyDDL_DropSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "drop database",
			sql:  "DROP DATABASE myapp;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropSchema, mutation.Kind)
				assert.Equal(t, "myapp", mutation.SchemaName)
			},
		},
		{
			name: "drop schema",
			sql:  "DROP SCHEMA analytics;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropSchema, mutation.Kind)
				assert.Equal(t, "analytics", mutation.SchemaName)
			},
		},
		{
			name: "drop database if exists",
			sql:  "DROP DATABASE IF EXISTS myapp;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropSchema, mutation.Kind)
				assert.Equal(t, "myapp", mutation.SchemaName)
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
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple trigger with BEGIN/END body",
			sql: `CREATE TRIGGER trg_audit
				BEFORE INSERT ON orders
				FOR EACH ROW
				BEGIN
					INSERT INTO audit_log(action) VALUES ('insert');
				END;`,
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, mutation.Kind)
				assert.Equal(t, "trg_audit", mutation.TriggerName)
				assert.Equal(t, "orders", mutation.TableName)
			},
		},
		{
			name: "trigger if not exists",
			sql: `CREATE TRIGGER IF NOT EXISTS trg_updated
				AFTER UPDATE ON users
				FOR EACH ROW
				BEGIN
					SET @x = 1;
				END;`,
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, mutation.Kind)
				assert.Equal(t, "trg_updated", mutation.TriggerName)
				assert.Equal(t, "users", mutation.TableName)
			},
		},
		{
			name: "AFTER DELETE trigger",
			sql: `CREATE TRIGGER trg_delete_audit
				AFTER DELETE ON orders
				FOR EACH ROW
				BEGIN
					INSERT INTO audit_log(action) VALUES ('delete');
				END;`,
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, mutation.Kind)
				assert.Equal(t, "trg_delete_audit", mutation.TriggerName)
				assert.Equal(t, "orders", mutation.TableName)
			},
		},
		{
			name: "BEFORE UPDATE trigger with single statement",
			sql:  "CREATE TRIGGER trg_before_upd BEFORE UPDATE ON accounts FOR EACH ROW SET NEW.updated_at = NOW();",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, mutation.Kind)
				assert.Equal(t, "trg_before_upd", mutation.TriggerName)
				assert.Equal(t, "accounts", mutation.TableName)
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
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple drop trigger",
			sql:  "DROP TRIGGER trg_audit;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTrigger, mutation.Kind)
				assert.Equal(t, "trg_audit", mutation.TriggerName)
			},
		},
		{
			name: "drop trigger if exists",
			sql:  "DROP TRIGGER IF EXISTS trg_audit;",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTrigger, mutation.Kind)
				assert.Equal(t, "trg_audit", mutation.TriggerName)
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

	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "SELECT returns nil mutation",
			sql:  "SELECT * FROM users;",
		},
		{
			name: "INSERT returns nil mutation",
			sql:  "INSERT INTO users (name) VALUES ('alice');",
		},
		{
			name: "UPDATE returns nil mutation",
			sql:  "UPDATE users SET name = 'bob' WHERE id = 1;",
		},
		{
			name: "DELETE returns nil mutation",
			sql:  "DELETE FROM users WHERE id = 1;",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, testCase.sql)
			assert.Nil(t, mutation, "non-DDL statement should produce nil mutation")
		})
	}
}

func TestApplyDDL_AlterTable_DialectSpecific(t *testing.T) {
	t.Parallel()

	t.Run("ALTER TABLE with parenthesised misc action", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "ALTER TABLE users CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;")
		assert.Nil(t, mutation, "CONVERT TO CHARACTER SET should return nil")
	})

	t.Run("ALTER TABLE RENAME AS", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "ALTER TABLE users RENAME AS new_users;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableRenameTable, mutation.Kind)
		assert.Equal(t, "new_users", mutation.NewName)
	})

	t.Run("ALTER TABLE with multiple comma-separated actions", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "ALTER TABLE users ADD COLUMN age INT, ADD COLUMN bio TEXT;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAddColumn, mutation.Kind)
		assert.Equal(t, "age", mutation.Columns[0].Name)
	})

	t.Run("ALTER TABLE MODIFY with FIRST", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "ALTER TABLE users MODIFY COLUMN name TEXT NOT NULL FIRST;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAlterColumn, mutation.Kind)
		assert.Equal(t, "name", mutation.ColumnName)
	})

	t.Run("ALTER TABLE MODIFY with AFTER", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "ALTER TABLE users MODIFY COLUMN name TEXT AFTER id;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAlterColumn, mutation.Kind)
	})

	t.Run("ALTER TABLE ADD with IF NOT EXISTS", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "ALTER TABLE users ADD COLUMN IF NOT EXISTS age INT;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAddColumn, mutation.Kind)
	})

	t.Run("ALTER TABLE DROP COLUMN IF EXISTS", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "ALTER TABLE users DROP COLUMN IF EXISTS age;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableDropColumn, mutation.Kind)
	})

	t.Run("ALTER TABLE ADD primary key constraint", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "ALTER TABLE users ADD PRIMARY KEY (id);")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAddConstraint, mutation.Kind)
	})

	t.Run("ALTER TABLE CHANGE with FIRST clause", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "ALTER TABLE users CHANGE COLUMN name new_name VARCHAR(255) FIRST;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableRenameColumn, mutation.Kind)
		assert.Equal(t, "name", mutation.ColumnName)
		assert.Equal(t, "new_name", mutation.NewName)
	})
}

func TestApplyDDL_CreateTable_IndexOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "index with USING BTREE option",
			sql:  "CREATE TABLE t (id INT, name VARCHAR(100), INDEX idx_name (name) USING BTREE);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				require.Len(t, mutation.Columns, 2)
			},
		},
		{
			name: "index with COMMENT option",
			sql:  "CREATE TABLE t (id INT, name VARCHAR(100), INDEX idx_name (name) COMMENT 'name index');",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				require.Len(t, mutation.Columns, 2)
			},
		},
		{
			name: "index with KEY_BLOCK_SIZE option",
			sql:  "CREATE TABLE t (id INT, name VARCHAR(100), INDEX idx_name (name) KEY_BLOCK_SIZE=1024);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				require.Len(t, mutation.Columns, 2)
			},
		},
		{
			name: "index with VISIBLE option",
			sql:  "CREATE TABLE t (id INT, name VARCHAR(100), INDEX idx_name (name) VISIBLE);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
			},
		},
		{
			name: "index with INVISIBLE option",
			sql:  "CREATE TABLE t (id INT, name VARCHAR(100), INDEX idx_name (name) INVISIBLE);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
			},
		},
		{
			name: "primary key with USING HASH",
			sql:  "CREATE TABLE t (id INT, PRIMARY KEY (id) USING HASH);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
				assert.Equal(t, []string{"id"}, mutation.PrimaryKey)
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

func TestApplyDDL_CreateTable_DefaultExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, mutation *querier_dto.CatalogueMutation)
	}{
		{
			name: "DEFAULT with function call containing arguments",
			sql:  "CREATE TABLE t (id INT, val DECIMAL(10,2) DEFAULT ROUND(1.5, 0));",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.True(t, mutation.Columns[1].HasDefault)
			},
		},
		{
			name: "DEFAULT NULL",
			sql:  "CREATE TABLE t (id INT, notes TEXT DEFAULT NULL);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.True(t, mutation.Columns[1].HasDefault)
			},
		},
		{
			name: "DEFAULT with numeric value",
			sql:  "CREATE TABLE t (id INT, score INT DEFAULT 0);",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.True(t, mutation.Columns[1].HasDefault)
			},
		},
		{
			name: "ON UPDATE with function call",
			sql:  "CREATE TABLE t (id INT, updated_at DATETIME DEFAULT NOW() ON UPDATE NOW());",
			assertions: func(t *testing.T, mutation *querier_dto.CatalogueMutation) {
				require.Len(t, mutation.Columns, 2)
				assert.True(t, mutation.Columns[1].HasDefault)
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

func TestApplyDDL_CreateView_OrReplace(t *testing.T) {
	t.Parallel()

	t.Run("create or replace view with schema-qualified name", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "CREATE OR REPLACE VIEW mydb.v_summary AS SELECT 1;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
		assert.Equal(t, "mydb", mutation.SchemaName)
		assert.Equal(t, "v_summary", mutation.TableName)
	})
}

func TestApplyDDL_DropDatabase_IfExists(t *testing.T) {
	t.Parallel()

	t.Run("drop schema if exists", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "DROP SCHEMA IF EXISTS analytics;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropSchema, mutation.Kind)
		assert.Equal(t, "analytics", mutation.SchemaName)
	})
}

func TestApplyDDL_DropTable_Restrict(t *testing.T) {
	t.Parallel()

	t.Run("drop table restrict", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "DROP TABLE users RESTRICT;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropTable, mutation.Kind)
	})
}

func TestApplyDDL_CreateIndex_IfNotExists(t *testing.T) {
	t.Parallel()

	t.Run("create index if not exists", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "CREATE INDEX IF NOT EXISTS idx_name ON users (name);")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateIndex, mutation.Kind)
		assert.Equal(t, "idx_name", mutation.NewName)
	})

	t.Run("schema-qualified drop index", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "DROP INDEX idx_email ON mydb.users;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropIndex, mutation.Kind)
		assert.Equal(t, "mydb", mutation.SchemaName)
	})
}

func TestApplyDDL_DropView_Cascade(t *testing.T) {
	t.Parallel()

	t.Run("drop view cascade", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "DROP VIEW IF EXISTS v_summary CASCADE;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropView, mutation.Kind)
	})
}

func TestApplyDDL_CreateFunction_Variants(t *testing.T) {
	t.Parallel()

	t.Run("function with no arguments", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "CREATE FUNCTION no_args() RETURNS VARCHAR(255) DETERMINISTIC RETURN 'hello';")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
		assert.Empty(t, mutation.FunctionSignature.Arguments)
		assert.Equal(t, querier_dto.TypeCategoryText, mutation.FunctionSignature.ReturnType.Category)
	})

	t.Run("drop procedure if exists", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, "DROP PROCEDURE IF EXISTS my_proc;")
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropFunction, mutation.Kind)
		assert.Equal(t, "my_proc", mutation.FunctionSignature.Name)
	})

	t.Run("function with nested BEGIN/END in body", func(t *testing.T) {
		t.Parallel()
		mutation := applyDDL(t, `CREATE FUNCTION nested_fn(x INT) RETURNS INT DETERMINISTIC
			BEGIN
				DECLARE result INT;
				IF x > 0 THEN
					BEGIN
						SET result = x * 2;
					END;
				END IF;
				RETURN result;
			END;`)
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
		assert.NotEmpty(t, mutation.FunctionSignature.BodySQL)
	})
}

func TestApplyDDL_CreateTable_ForeignKeySchemaQualified(t *testing.T) {
	t.Parallel()

	mutation := applyDDL(t, `CREATE TABLE orders (
		id INT,
		user_id INT,
		FOREIGN KEY (user_id) REFERENCES mydb.users(id)
	);`)
	require.NotNil(t, mutation)
	require.NotEmpty(t, mutation.Constraints)
	fk := mutation.Constraints[0]
	assert.Equal(t, querier_dto.ConstraintForeignKey, fk.Kind)
	assert.Equal(t, "users", fk.ForeignTable)
}

func TestClassifyStatement_Unknown(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()

	t.Run("SET statement is classified as unknown", func(t *testing.T) {
		t.Parallel()
		statements, err := engine.ParseStatements("SET @x = 1;")
		require.NoError(t, err)
		require.Len(t, statements, 1)

		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		assert.Nil(t, mutation)
	})

	t.Run("DROP with unknown object type", func(t *testing.T) {
		t.Parallel()
		statements, err := engine.ParseStatements("DROP EVENT my_event;")
		require.NoError(t, err)
		require.Len(t, statements, 1)
		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		assert.Nil(t, mutation)
	})

	t.Run("ALTER with non-TABLE object", func(t *testing.T) {
		t.Parallel()
		statements, err := engine.ParseStatements("ALTER DATABASE mydb CHARACTER SET utf8mb4;")
		require.NoError(t, err)
		require.Len(t, statements, 1)
		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		assert.Nil(t, mutation)
	})

	t.Run("CREATE with unknown object type", func(t *testing.T) {
		t.Parallel()
		statements, err := engine.ParseStatements("CREATE EVENT daily_cleanup ON SCHEDULE EVERY 1 DAY DO DELETE FROM tmp;")
		require.NoError(t, err)
		require.Len(t, statements, 1)
		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		assert.Nil(t, mutation)
	})
}

func TestParseStatements_MultipleStatements(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()
	sql := "CREATE TABLE a (id INT); CREATE TABLE b (id INT); DROP TABLE a;"
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, statements, 3, "expected three statements from semicolon-separated SQL")

	mutation1, err := engine.ApplyDDL(statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation1)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation1.Kind)
	assert.Equal(t, "a", mutation1.TableName)

	mutation2, err := engine.ApplyDDL(statements[1])
	require.NoError(t, err)
	require.NotNil(t, mutation2)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation2.Kind)
	assert.Equal(t, "b", mutation2.TableName)

	mutation3, err := engine.ApplyDDL(statements[2])
	require.NoError(t, err)
	require.NotNil(t, mutation3)
	assert.Equal(t, querier_dto.MutationDropTable, mutation3.Kind)
	assert.Equal(t, "a", mutation3.TableName)
}

func TestParseStatements_MixedDDLAndDML(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()
	sql := "CREATE TABLE users (id INT, name VARCHAR(100)); INSERT INTO users (id, name) VALUES (1, 'alice'); SELECT * FROM users;"
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, statements, 3)

	mutation, err := engine.ApplyDDL(statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)

	analysis, err := engine.AnalyseQuery(nil, statements[1])
	require.NoError(t, err)
	assert.Equal(t, "users", analysis.InsertTable)

	analysis, err = engine.AnalyseQuery(nil, statements[2])
	require.NoError(t, err)
	assert.True(t, analysis.ReadOnly)
}
