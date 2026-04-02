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

func applyDDL(t *testing.T, sql string) *querier_dto.CatalogueMutation {
	t.Helper()
	engine := NewDuckDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.NotEmpty(t, statements)
	mutation, err := engine.ApplyDDL(statements[0])
	require.NoError(t, err)
	return mutation
}

func TestApplyDDL_CreateTable(t *testing.T) {
	t.Parallel()

	t.Run("simple table with typed columns", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name VARCHAR NOT NULL,
			email TEXT
		)`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
		require.Len(t, mutation.Columns, 3)

		assert.Equal(t, "id", mutation.Columns[0].Name)
		assert.Equal(t, querier_dto.TypeCategoryInteger, mutation.Columns[0].SQLType.Category)
		assert.False(t, mutation.Columns[0].Nullable, "primary key column should not be nullable")

		assert.Equal(t, "name", mutation.Columns[1].Name)
		assert.Equal(t, querier_dto.TypeCategoryText, mutation.Columns[1].SQLType.Category)
		assert.False(t, mutation.Columns[1].Nullable, "NOT NULL column should not be nullable")

		assert.Equal(t, "email", mutation.Columns[2].Name)
		assert.Equal(t, querier_dto.TypeCategoryText, mutation.Columns[2].SQLType.Category)
		assert.True(t, mutation.Columns[2].Nullable, "column without NOT NULL should be nullable")

		assert.Equal(t, []string{"id"}, mutation.PrimaryKey)
	})

	t.Run("IF NOT EXISTS is tolerated", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE IF NOT EXISTS products (
			id BIGINT PRIMARY KEY,
			title VARCHAR
		)`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
		assert.Equal(t, "products", mutation.TableName)
		require.Len(t, mutation.Columns, 2)
	})

	t.Run("column constraints are parsed", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE items (
			id INTEGER PRIMARY KEY,
			name VARCHAR NOT NULL UNIQUE,
			quantity INTEGER DEFAULT 0,
			price NUMERIC CHECK (price >= 0)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 4)

		assert.True(t, mutation.Columns[2].HasDefault, "column with DEFAULT should have HasDefault set")
	})

	t.Run("schema-qualified table name", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE myschema.orders (id INTEGER PRIMARY KEY)`)

		require.NotNil(t, mutation)
		assert.Equal(t, "myschema", mutation.SchemaName)
		assert.Equal(t, "orders", mutation.TableName)
	})

	t.Run("composite primary key via table constraint", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE order_items (
			order_id INTEGER,
			product_id INTEGER,
			quantity INTEGER,
			PRIMARY KEY (order_id, product_id)
		)`)

		require.NotNil(t, mutation)
		assert.Equal(t, []string{"order_id", "product_id"}, mutation.PrimaryKey)
	})

	t.Run("CREATE TABLE AS SELECT produces empty columns", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE snapshot AS SELECT * FROM users`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
		assert.Equal(t, "snapshot", mutation.TableName)
		assert.Empty(t, mutation.Columns, "CTAS should not produce column definitions")
	})

	t.Run("generated always as identity column", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE accounts (
			id INTEGER GENERATED ALWAYS AS IDENTITY,
			name VARCHAR NOT NULL
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.True(t, mutation.Columns[0].HasDefault, "identity column should have HasDefault set")
	})

	t.Run("foreign key table constraint", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE reviews (
			id INTEGER PRIMARY KEY,
			user_id INTEGER,
			FOREIGN KEY (user_id) REFERENCES users (id)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Constraints, 1)
		assert.Equal(t, querier_dto.ConstraintForeignKey, mutation.Constraints[0].Kind)
		assert.Equal(t, "users", mutation.Constraints[0].ForeignTable)
		assert.Equal(t, []string{"user_id"}, mutation.Constraints[0].Columns)
		assert.Equal(t, []string{"id"}, mutation.Constraints[0].ForeignColumns)
	})

	t.Run("unique table constraint", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE emails (
			id INTEGER PRIMARY KEY,
			address VARCHAR,
			UNIQUE (address)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Constraints, 1)
		assert.Equal(t, querier_dto.ConstraintUnique, mutation.Constraints[0].Kind)
		assert.Equal(t, []string{"address"}, mutation.Constraints[0].Columns)
	})

	t.Run("DuckDB struct column type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE events (
			id INTEGER PRIMARY KEY,
			metadata STRUCT(key VARCHAR, value INTEGER)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.Equal(t, querier_dto.TypeCategoryStruct, mutation.Columns[1].SQLType.Category)
		require.Len(t, mutation.Columns[1].SQLType.StructFields, 2)
		assert.Equal(t, "key", mutation.Columns[1].SQLType.StructFields[0].Name)
		assert.Equal(t, "value", mutation.Columns[1].SQLType.StructFields[1].Name)
	})

	t.Run("array column type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE tags (
			id INTEGER PRIMARY KEY,
			labels VARCHAR[]
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.True(t, mutation.Columns[1].IsArray, "column with [] should be an array")
		assert.Equal(t, 1, mutation.Columns[1].ArrayDimensions)
	})

	t.Run("map column type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE config (
			id INTEGER PRIMARY KEY,
			settings MAP(VARCHAR, INTEGER)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.Equal(t, querier_dto.TypeCategoryMap, mutation.Columns[1].SQLType.Category)
	})

	t.Run("union column type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE data (
			id INTEGER PRIMARY KEY,
			payload UNION(i INTEGER, s VARCHAR)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.Equal(t, querier_dto.TypeCategoryUnion, mutation.Columns[1].SQLType.Category)
	})

	t.Run("list column type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE lists (
			id INTEGER PRIMARY KEY,
			items LIST(INTEGER)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.Equal(t, querier_dto.TypeCategoryArray, mutation.Columns[1].SQLType.Category)
	})

	t.Run("GENERATED ALWAYS AS stored column", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE products (
			price NUMERIC,
			tax NUMERIC,
			total NUMERIC GENERATED ALWAYS AS (price + tax) STORED
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 3)
		assert.True(t, mutation.Columns[2].IsGenerated, "generated column should be marked generated")
		assert.Equal(t, querier_dto.GeneratedKindStored, mutation.Columns[2].GeneratedKind)
	})

	t.Run("CHECK constraint on table", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE amounts (
			id INTEGER PRIMARY KEY,
			value NUMERIC,
			CHECK (value >= 0)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Constraints, 1)
		assert.Equal(t, querier_dto.ConstraintCheck, mutation.Constraints[0].Kind)
	})

	t.Run("timestamp with time zone column", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE logs (
			id INTEGER PRIMARY KEY,
			created_at TIMESTAMP WITH TIME ZONE
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.Equal(t, querier_dto.TypeCategoryTemporal, mutation.Columns[1].SQLType.Category)
	})

	t.Run("double precision column type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE measurements (
			id INTEGER PRIMARY KEY,
			value DOUBLE PRECISION
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.Equal(t, querier_dto.TypeCategoryFloat, mutation.Columns[1].SQLType.Category)
	})

	t.Run("column with DEFAULT value", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE flags (
			id INTEGER PRIMARY KEY,
			active BOOLEAN DEFAULT true
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.True(t, mutation.Columns[1].HasDefault, "column with DEFAULT should have HasDefault set")
	})

	t.Run("named CONSTRAINT on table", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE t (
			x INTEGER,
			CONSTRAINT pk_x PRIMARY KEY (x)
		)`)

		require.NotNil(t, mutation)
		assert.Equal(t, []string{"x"}, mutation.PrimaryKey)
	})

	t.Run("column with REFERENCES constraint", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE comments (
			id INTEGER PRIMARY KEY,
			user_id INTEGER REFERENCES users (id)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.Equal(t, "user_id", mutation.Columns[1].Name)
	})

	t.Run("column with COLLATE constraint", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE names (
			id INTEGER PRIMARY KEY,
			name VARCHAR COLLATE NOCASE
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.Equal(t, "name", mutation.Columns[1].Name)
	})

	t.Run("column with GENERATED BY DEFAULT AS IDENTITY", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE tickets (
			id INTEGER GENERATED BY DEFAULT AS IDENTITY,
			title VARCHAR
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.True(t, mutation.Columns[0].HasDefault)
	})

	t.Run("character varying column type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE texts (
			id INTEGER PRIMARY KEY,
			content CHARACTER VARYING(500)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.Equal(t, querier_dto.TypeCategoryText, mutation.Columns[1].SQLType.Category)
	})

	t.Run("timestamp without time zone", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE events (
			id INTEGER PRIMARY KEY,
			created_at TIMESTAMP WITHOUT TIME ZONE
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
		assert.Equal(t, querier_dto.TypeCategoryTemporal, mutation.Columns[1].SQLType.Category)
	})

	t.Run("DEFAULT with complex expression", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE t (
			id INTEGER PRIMARY KEY,
			created_at TIMESTAMP DEFAULT (now()),
			status VARCHAR DEFAULT 'active'
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 3)
		assert.True(t, mutation.Columns[1].HasDefault)
		assert.True(t, mutation.Columns[2].HasDefault)
	})

	t.Run("multiple FOREIGN KEY constraints", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE reviews (
			id INTEGER PRIMARY KEY,
			user_id INTEGER,
			product_id INTEGER,
			FOREIGN KEY (user_id) REFERENCES users (id),
			FOREIGN KEY (product_id) REFERENCES products (id)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Constraints, 2)
		assert.Equal(t, querier_dto.ConstraintForeignKey, mutation.Constraints[0].Kind)
		assert.Equal(t, querier_dto.ConstraintForeignKey, mutation.Constraints[1].Kind)
	})
}

func TestApplyDDL_AlterTable(t *testing.T) {
	t.Parallel()

	t.Run("ADD COLUMN", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE users ADD COLUMN age INTEGER`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAddColumn, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
		require.Len(t, mutation.Columns, 1)
		assert.Equal(t, "age", mutation.Columns[0].Name)
		assert.Equal(t, querier_dto.TypeCategoryInteger, mutation.Columns[0].SQLType.Category)
	})

	t.Run("ADD COLUMN without COLUMN keyword", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE users ADD bio TEXT`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAddColumn, mutation.Kind)
		require.Len(t, mutation.Columns, 1)
		assert.Equal(t, "bio", mutation.Columns[0].Name)
	})

	t.Run("DROP COLUMN", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE users DROP COLUMN email`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableDropColumn, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
		assert.Equal(t, "email", mutation.ColumnName)
	})

	t.Run("DROP COLUMN without COLUMN keyword", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE users DROP email`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableDropColumn, mutation.Kind)
		assert.Equal(t, "email", mutation.ColumnName)
	})

	t.Run("RENAME COLUMN", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE users RENAME COLUMN name TO full_name`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableRenameColumn, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
		assert.Equal(t, "name", mutation.ColumnName)
		assert.Equal(t, "full_name", mutation.NewName)
	})

	t.Run("RENAME TABLE", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE users RENAME TO accounts`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableRenameTable, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
		assert.Equal(t, "accounts", mutation.NewName)
	})

	t.Run("SET SCHEMA", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE users SET SCHEMA archive`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableSetSchema, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
		assert.Equal(t, "archive", mutation.NewName)
	})

	t.Run("ADD CONSTRAINT", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE orders ADD UNIQUE (order_number)`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAddConstraint, mutation.Kind)
		assert.Equal(t, "orders", mutation.TableName)
		require.Len(t, mutation.Constraints, 1)
		assert.Equal(t, querier_dto.ConstraintUnique, mutation.Constraints[0].Kind)
	})

	t.Run("DROP CONSTRAINT", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE orders DROP CONSTRAINT orders_number_unique`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableDropConstraint, mutation.Kind)
		assert.Equal(t, "orders", mutation.TableName)
		assert.Equal(t, "orders_number_unique", mutation.ConstraintName)
	})

	t.Run("schema-qualified ALTER TABLE", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE myschema.users ADD COLUMN age INTEGER`)

		require.NotNil(t, mutation)
		assert.Equal(t, "myschema", mutation.SchemaName)
		assert.Equal(t, "users", mutation.TableName)
	})

	t.Run("ALTER COLUMN", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE users ALTER COLUMN email SET NOT NULL`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAlterColumn, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
		assert.Equal(t, "email", mutation.ColumnName)
	})
}

func TestApplyDDL_DropTable(t *testing.T) {
	t.Parallel()

	t.Run("simple drop", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP TABLE users`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropTable, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
	})

	t.Run("IF EXISTS", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP TABLE IF EXISTS users`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropTable, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
	})

	t.Run("CASCADE", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP TABLE users CASCADE`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropTable, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
	})

	t.Run("schema-qualified drop", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP TABLE myschema.users`)

		require.NotNil(t, mutation)
		assert.Equal(t, "myschema", mutation.SchemaName)
		assert.Equal(t, "users", mutation.TableName)
	})
}

func TestApplyDDL_CreateView(t *testing.T) {
	t.Parallel()

	t.Run("simple view", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE VIEW active_users AS SELECT id, name FROM users WHERE active = true`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
		assert.Equal(t, "active_users", mutation.TableName)
		assert.NotNil(t, mutation.ViewDefinition, "view body should be analysed")
	})

	t.Run("view with explicit column names", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE VIEW user_summary (user_id, user_name) AS SELECT id, name FROM users`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
		require.NotNil(t, mutation.ViewDefinition)
		require.Len(t, mutation.ViewDefinition.OutputColumns, 2)
		assert.Equal(t, "user_id", mutation.ViewDefinition.OutputColumns[0].Name)
		assert.Equal(t, "user_name", mutation.ViewDefinition.OutputColumns[1].Name)
	})

	t.Run("OR REPLACE view", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE OR REPLACE VIEW active_users AS SELECT id FROM users`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
		assert.Equal(t, "active_users", mutation.TableName)
	})

	t.Run("IF NOT EXISTS view", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE VIEW IF NOT EXISTS active_users AS SELECT id FROM users`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
	})

	t.Run("schema-qualified view", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE VIEW reporting.summary AS SELECT count(*) FROM users`)

		require.NotNil(t, mutation)
		assert.Equal(t, "reporting", mutation.SchemaName)
		assert.Equal(t, "summary", mutation.TableName)
	})
}

func TestApplyDDL_DropView(t *testing.T) {
	t.Parallel()

	t.Run("simple drop view", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP VIEW active_users`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropView, mutation.Kind)
		assert.Equal(t, "active_users", mutation.TableName)
	})

	t.Run("IF EXISTS drop view", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP VIEW IF EXISTS active_users`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropView, mutation.Kind)
		assert.Equal(t, "active_users", mutation.TableName)
	})
}

func TestApplyDDL_CreateIndex(t *testing.T) {
	t.Parallel()

	t.Run("simple index", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE INDEX idx_users_email ON users (email)`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateIndex, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
		assert.Equal(t, "idx_users_email", mutation.NewName)
	})

	t.Run("unique index", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE UNIQUE INDEX idx_users_email_unique ON users (email)`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateIndex, mutation.Kind)
		assert.Equal(t, "idx_users_email_unique", mutation.NewName)
	})

	t.Run("IF NOT EXISTS index", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE INDEX IF NOT EXISTS idx_users_email ON users (email)`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateIndex, mutation.Kind)
		assert.Equal(t, "idx_users_email", mutation.NewName)
	})

	t.Run("index on schema-qualified table", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE INDEX idx_orders_date ON myschema.orders (order_date)`)

		require.NotNil(t, mutation)
		assert.Equal(t, "myschema", mutation.SchemaName)
		assert.Equal(t, "orders", mutation.TableName)
	})
}

func TestApplyDDL_DropIndex(t *testing.T) {
	t.Parallel()

	t.Run("simple drop index", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP INDEX idx_users_email`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropIndex, mutation.Kind)
		assert.Equal(t, "idx_users_email", mutation.NewName)
	})

	t.Run("IF EXISTS drop index", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP INDEX IF EXISTS idx_users_email`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropIndex, mutation.Kind)
		assert.Equal(t, "idx_users_email", mutation.NewName)
	})
}

func TestApplyDDL_CreateType(t *testing.T) {
	t.Parallel()

	t.Run("enum type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TYPE mood AS ENUM ('happy', 'sad', 'neutral')`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateEnum, mutation.Kind)
		assert.Equal(t, "mood", mutation.EnumName)
		assert.Equal(t, []string{"happy", "sad", "neutral"}, mutation.EnumValues)
	})

	t.Run("schema-qualified enum type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TYPE myschema.status AS ENUM ('active', 'inactive')`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateEnum, mutation.Kind)
		assert.Equal(t, "myschema", mutation.SchemaName)
		assert.Equal(t, "status", mutation.EnumName)
	})

	t.Run("composite type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TYPE address AS (
			street VARCHAR,
			city VARCHAR,
			postcode VARCHAR
		)`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateCompositeType, mutation.Kind)
		assert.Equal(t, "address", mutation.EnumName, "composite type name stored in EnumName field")
		require.Len(t, mutation.Columns, 3)
		assert.Equal(t, "street", mutation.Columns[0].Name)
		assert.Equal(t, "city", mutation.Columns[1].Name)
		assert.Equal(t, "postcode", mutation.Columns[2].Name)
	})

	t.Run("composite type with mixed types", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TYPE measurement AS (
			value DOUBLE,
			unit VARCHAR,
			timestamp TIMESTAMP
		)`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateCompositeType, mutation.Kind)
		require.Len(t, mutation.Columns, 3)
		assert.Equal(t, querier_dto.TypeCategoryFloat, mutation.Columns[0].SQLType.Category)
		assert.Equal(t, querier_dto.TypeCategoryText, mutation.Columns[1].SQLType.Category)
		assert.Equal(t, querier_dto.TypeCategoryTemporal, mutation.Columns[2].SQLType.Category)
	})

	t.Run("enum type with many values", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TYPE colour AS ENUM ('red', 'green', 'blue', 'yellow', 'purple')`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateEnum, mutation.Kind)
		assert.Equal(t, "colour", mutation.EnumName)
		require.Len(t, mutation.EnumValues, 5)
		assert.Equal(t, "red", mutation.EnumValues[0])
		assert.Equal(t, "purple", mutation.EnumValues[4])
	})
}

func TestApplyDDL_DropType(t *testing.T) {
	t.Parallel()

	t.Run("simple drop type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP TYPE mood`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropType, mutation.Kind)
		assert.Equal(t, "mood", mutation.EnumName)
	})

	t.Run("IF EXISTS drop type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP TYPE IF EXISTS mood`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropType, mutation.Kind)
		assert.Equal(t, "mood", mutation.EnumName)
	})

	t.Run("CASCADE drop type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP TYPE mood CASCADE`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropType, mutation.Kind)
	})

	t.Run("schema-qualified DROP TYPE", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP TYPE myschema.mood`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropType, mutation.Kind)
		assert.Equal(t, "myschema", mutation.SchemaName)
		assert.Equal(t, "mood", mutation.EnumName)
	})
}

func TestApplyDDL_AlterType(t *testing.T) {
	t.Parallel()

	t.Run("ALTER TYPE ADD VALUE", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TYPE mood ADD VALUE 'excited'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterEnumAddValue, mutation.Kind)
		assert.Equal(t, "mood", mutation.EnumName)
		require.Len(t, mutation.EnumValues, 1)
		assert.Equal(t, "excited", mutation.EnumValues[0])
	})

	t.Run("ALTER TYPE ADD VALUE IF NOT EXISTS", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TYPE mood ADD VALUE IF NOT EXISTS 'calm'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterEnumAddValue, mutation.Kind)
		assert.Equal(t, "mood", mutation.EnumName)
		require.Len(t, mutation.EnumValues, 1)
		assert.Equal(t, "calm", mutation.EnumValues[0])
	})

	t.Run("ALTER TYPE ADD VALUE BEFORE", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TYPE mood ADD VALUE 'anxious' BEFORE 'sad'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterEnumAddValue, mutation.Kind)
		require.Len(t, mutation.EnumValues, 1)
		assert.Equal(t, "anxious", mutation.EnumValues[0])
	})

	t.Run("ALTER TYPE ADD VALUE AFTER", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TYPE mood ADD VALUE 'ecstatic' AFTER 'happy'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterEnumAddValue, mutation.Kind)
		require.Len(t, mutation.EnumValues, 1)
		assert.Equal(t, "ecstatic", mutation.EnumValues[0])
	})

	t.Run("ALTER TYPE RENAME VALUE", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TYPE mood RENAME VALUE 'sad' TO 'melancholy'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterEnumRenameValue, mutation.Kind)
		assert.Equal(t, "mood", mutation.EnumName)
		require.Len(t, mutation.EnumValues, 2)
		assert.Equal(t, "sad", mutation.EnumValues[0])
		assert.Equal(t, "melancholy", mutation.EnumValues[1])
	})

	t.Run("schema-qualified ALTER TYPE ADD VALUE", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TYPE myschema.mood ADD VALUE 'relaxed'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterEnumAddValue, mutation.Kind)
		assert.Equal(t, "myschema", mutation.SchemaName)
		assert.Equal(t, "mood", mutation.EnumName)
	})
}

func TestApplyDDL_CreateMacro(t *testing.T) {
	t.Parallel()

	t.Run("scalar macro", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE MACRO add_one(x) AS x + 1`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
		require.NotNil(t, mutation.FunctionSignature)
		assert.Equal(t, "add_one", mutation.FunctionSignature.Name)
		require.Len(t, mutation.FunctionSignature.Arguments, 1)

		assert.Equal(t, "x", mutation.FunctionSignature.Arguments[0].Type.EngineName)
	})

	t.Run("table macro", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE MACRO recent_users() AS TABLE SELECT * FROM users WHERE created_at > now() - INTERVAL '1 day'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
		require.NotNil(t, mutation.FunctionSignature)
		assert.Equal(t, "recent_users", mutation.FunctionSignature.Name)
		assert.True(t, mutation.FunctionSignature.ReturnsSet, "table macro should have ReturnsSet")
	})

	t.Run("OR REPLACE macro", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE OR REPLACE MACRO double_it(x) AS x * 2`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
		assert.Equal(t, "double_it", mutation.FunctionSignature.Name)
	})

	t.Run("CREATE FUNCTION alias for macro", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE FUNCTION triple(x) AS x * 3`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
		assert.Equal(t, "triple", mutation.FunctionSignature.Name)
	})

	t.Run("schema-qualified macro", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE MACRO myschema.greet(n) AS 'Hello, ' || n`)

		require.NotNil(t, mutation)
		assert.Equal(t, "myschema", mutation.SchemaName)
		assert.Equal(t, "greet", mutation.FunctionSignature.Name)
	})

	t.Run("macro with multiple arguments", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE MACRO add(a, b) AS a + b`)

		require.NotNil(t, mutation)
		require.NotNil(t, mutation.FunctionSignature)
		require.Len(t, mutation.FunctionSignature.Arguments, 2)

		assert.Equal(t, "a", mutation.FunctionSignature.Arguments[0].Type.EngineName)
		assert.Equal(t, "b", mutation.FunctionSignature.Arguments[1].Type.EngineName)
	})

	t.Run("table macro with arguments", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE MACRO user_orders(uid) AS TABLE SELECT * FROM orders WHERE user_id = uid`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateFunction, mutation.Kind)
		require.NotNil(t, mutation.FunctionSignature)
		assert.Equal(t, "user_orders", mutation.FunctionSignature.Name)
		assert.True(t, mutation.FunctionSignature.ReturnsSet, "table macro should have ReturnsSet")
		require.Len(t, mutation.FunctionSignature.Arguments, 1)
	})

	t.Run("macro with string concatenation body", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE MACRO greeting(name) AS 'Hello, ' || name || '!'`)

		require.NotNil(t, mutation)
		require.NotNil(t, mutation.FunctionSignature)
		assert.Equal(t, "greeting", mutation.FunctionSignature.Name)
	})
}

func TestApplyDDL_DropMacro(t *testing.T) {
	t.Parallel()

	t.Run("DROP MACRO", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP MACRO add_one`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropFunction, mutation.Kind)
		require.NotNil(t, mutation.FunctionSignature)
		assert.Equal(t, "add_one", mutation.FunctionSignature.Name)
	})

	t.Run("DROP FUNCTION", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP FUNCTION my_func`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropFunction, mutation.Kind)
		require.NotNil(t, mutation.FunctionSignature)
		assert.Equal(t, "my_func", mutation.FunctionSignature.Name)
	})

	t.Run("IF EXISTS drop macro", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP MACRO IF EXISTS add_one`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropFunction, mutation.Kind)
	})
}

func TestApplyDDL_CreateSchema(t *testing.T) {
	t.Parallel()

	t.Run("simple schema", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE SCHEMA analytics`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateSchema, mutation.Kind)
		assert.Equal(t, "analytics", mutation.SchemaName)
	})

	t.Run("IF NOT EXISTS schema", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE SCHEMA IF NOT EXISTS analytics`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateSchema, mutation.Kind)
		assert.Equal(t, "analytics", mutation.SchemaName)
	})
}

func TestApplyDDL_DropSchema(t *testing.T) {
	t.Parallel()

	t.Run("simple drop schema", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP SCHEMA analytics`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropSchema, mutation.Kind)
		assert.Equal(t, "analytics", mutation.SchemaName)
	})

	t.Run("IF EXISTS drop schema", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP SCHEMA IF EXISTS analytics`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropSchema, mutation.Kind)
		assert.Equal(t, "analytics", mutation.SchemaName)
	})

	t.Run("CASCADE drop schema", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP SCHEMA analytics CASCADE`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropSchema, mutation.Kind)
	})
}

func TestApplyDDL_CreateSequence(t *testing.T) {
	t.Parallel()

	t.Run("simple sequence", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE SEQUENCE user_id_seq`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateSequence, mutation.Kind)
		assert.Equal(t, "user_id_seq", mutation.SequenceName)
	})

	t.Run("IF NOT EXISTS sequence", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE SEQUENCE IF NOT EXISTS user_id_seq`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateSequence, mutation.Kind)
		assert.Equal(t, "user_id_seq", mutation.SequenceName)
	})

	t.Run("schema-qualified sequence", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE SEQUENCE myschema.order_id_seq`)

		require.NotNil(t, mutation)
		assert.Equal(t, "myschema", mutation.SchemaName)
		assert.Equal(t, "order_id_seq", mutation.SequenceName)
	})

	t.Run("sequence with OWNED BY", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE SEQUENCE user_id_seq OWNED BY users.id`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateSequence, mutation.Kind)
		assert.Equal(t, "users", mutation.OwnedByTable)
		assert.Equal(t, "id", mutation.OwnedByColumn)
	})

	t.Run("sequence with all options", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE SEQUENCE order_id_seq START WITH 1000 INCREMENT BY 1 MINVALUE 1 MAXVALUE 999999 OWNED BY orders.id`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateSequence, mutation.Kind)
		assert.Equal(t, "order_id_seq", mutation.SequenceName)
		assert.Equal(t, "orders", mutation.OwnedByTable)
		assert.Equal(t, "id", mutation.OwnedByColumn)
	})

	t.Run("sequence with OWNED BY NONE", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE SEQUENCE standalone_seq OWNED BY NONE`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateSequence, mutation.Kind)
		assert.Equal(t, "standalone_seq", mutation.SequenceName)
		assert.Empty(t, mutation.OwnedByTable, "OWNED BY NONE should not set table")
		assert.Empty(t, mutation.OwnedByColumn, "OWNED BY NONE should not set column")
	})
}

func TestApplyDDL_DropSequence(t *testing.T) {
	t.Parallel()

	t.Run("simple drop sequence", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP SEQUENCE user_id_seq`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropSequence, mutation.Kind)
		assert.Equal(t, "user_id_seq", mutation.SequenceName)
	})

	t.Run("IF EXISTS drop sequence", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP SEQUENCE IF EXISTS user_id_seq`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropSequence, mutation.Kind)
		assert.Equal(t, "user_id_seq", mutation.SequenceName)
	})

	t.Run("schema-qualified drop sequence", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP SEQUENCE myschema.order_id_seq`)

		require.NotNil(t, mutation)
		assert.Equal(t, "myschema", mutation.SchemaName)
		assert.Equal(t, "order_id_seq", mutation.SequenceName)
	})

	t.Run("CASCADE drop sequence", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `DROP SEQUENCE user_id_seq CASCADE`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropSequence, mutation.Kind)
		assert.Equal(t, "user_id_seq", mutation.SequenceName)
	})
}

func TestApplyDDL_Comment(t *testing.T) {
	t.Parallel()

	t.Run("comment on table", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `COMMENT ON TABLE users IS 'User accounts'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationComment, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
	})

	t.Run("comment on column", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `COMMENT ON COLUMN users.email IS 'Primary email address'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationComment, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
		assert.Equal(t, "email", mutation.ColumnName)
	})

	t.Run("comment on schema", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `COMMENT ON SCHEMA analytics IS 'Analytics data'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationComment, mutation.Kind)
		assert.Equal(t, "analytics", mutation.SchemaName)
	})

	t.Run("comment on function", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `COMMENT ON FUNCTION add_one IS 'Adds one'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationComment, mutation.Kind)
		require.NotNil(t, mutation.FunctionSignature)
		assert.Equal(t, "add_one", mutation.FunctionSignature.Name)
	})

	t.Run("comment on type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `COMMENT ON TYPE mood IS 'Mood enumeration'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationComment, mutation.Kind)
		assert.Equal(t, "mood", mutation.EnumName)
	})

	t.Run("remove comment with NULL", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `COMMENT ON TABLE users IS NULL`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationComment, mutation.Kind)
		assert.Equal(t, "users", mutation.TableName)
	})
}

func TestApplyDDL_NonDDL(t *testing.T) {
	t.Parallel()

	t.Run("SELECT returns nil mutation", func(t *testing.T) {
		t.Parallel()

		engine := NewDuckDBEngine()
		statements, err := engine.ParseStatements(`SELECT * FROM users`)
		require.NoError(t, err)
		require.NotEmpty(t, statements)

		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		assert.Nil(t, mutation, "SELECT should not produce a DDL mutation")
	})

	t.Run("INSERT returns nil mutation", func(t *testing.T) {
		t.Parallel()

		engine := NewDuckDBEngine()
		statements, err := engine.ParseStatements(`INSERT INTO users (name) VALUES ('alice')`)
		require.NoError(t, err)
		require.NotEmpty(t, statements)

		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		assert.Nil(t, mutation, "INSERT should not produce a DDL mutation")
	})

	t.Run("UPDATE returns nil mutation", func(t *testing.T) {
		t.Parallel()

		engine := NewDuckDBEngine()
		statements, err := engine.ParseStatements(`UPDATE users SET name = 'bob' WHERE id = 1`)
		require.NoError(t, err)
		require.NotEmpty(t, statements)

		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		assert.Nil(t, mutation, "UPDATE should not produce a DDL mutation")
	})

	t.Run("DELETE returns nil mutation", func(t *testing.T) {
		t.Parallel()

		engine := NewDuckDBEngine()
		statements, err := engine.ParseStatements(`DELETE FROM users WHERE id = 1`)
		require.NoError(t, err)
		require.NotEmpty(t, statements)

		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		assert.Nil(t, mutation, "DELETE should not produce a DDL mutation")
	})
}

func TestApplyDDL_CreateTable_ConstraintsAndDefaults(t *testing.T) {
	t.Parallel()

	t.Run("column with DEFAULT function call expression", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE t (
			id INTEGER PRIMARY KEY,
			uid UUID DEFAULT (gen_random_uuid()),
			ts TIMESTAMP DEFAULT (now() + INTERVAL '1 day')
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 3)
		assert.True(t, mutation.Columns[1].HasDefault)
		assert.True(t, mutation.Columns[2].HasDefault)
	})

	t.Run("column with REFERENCES ON DELETE CASCADE ON UPDATE SET NULL", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE child (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER REFERENCES parent (id) ON DELETE CASCADE ON UPDATE SET NULL
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 2)
	})

	t.Run("table FOREIGN KEY with ON DELETE and ON UPDATE", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE child (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER,
			FOREIGN KEY (parent_id) REFERENCES parent (id) ON DELETE SET DEFAULT ON UPDATE NO ACTION
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Constraints, 1)
	})

	t.Run("named UNIQUE constraint", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE t (
			a INTEGER,
			b INTEGER,
			CONSTRAINT uq_ab UNIQUE (a, b)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Constraints, 1)
		assert.Equal(t, querier_dto.ConstraintUnique, mutation.Constraints[0].Kind)
	})

	t.Run("PRIMARY KEY with named constraint", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE t (
			a INTEGER,
			b INTEGER,
			CONSTRAINT pk_ab PRIMARY KEY (a, b)
		)`)

		require.NotNil(t, mutation)
		assert.Equal(t, []string{"a", "b"}, mutation.PrimaryKey)
	})

	t.Run("multi-word type: CHARACTER VARYING", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE t (
			name CHARACTER VARYING(100)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 1)
		assert.Equal(t, querier_dto.TypeCategoryText, mutation.Columns[0].SQLType.Category)
	})

	t.Run("GENERATED ALWAYS AS virtual column", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE products (
			price NUMERIC,
			tax NUMERIC,
			total NUMERIC GENERATED ALWAYS AS (price + tax) VIRTUAL
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 3)
		assert.True(t, mutation.Columns[2].IsGenerated)

		assert.Equal(t, querier_dto.GeneratedKindStored, mutation.Columns[2].GeneratedKind)
	})

	t.Run("nested struct type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE t (
			data STRUCT(inner_struct STRUCT(x INTEGER, y INTEGER), label VARCHAR)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 1)
		assert.Equal(t, querier_dto.TypeCategoryStruct, mutation.Columns[0].SQLType.Category)
	})

	t.Run("multi-dimensional array type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE t (
			matrix INTEGER[][]
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 1)
		assert.True(t, mutation.Columns[0].IsArray)
	})

	t.Run("numeric with precision and scale", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE TABLE t (
			amount NUMERIC(10, 2)
		)`)

		require.NotNil(t, mutation)
		require.Len(t, mutation.Columns, 1)
		assert.Equal(t, querier_dto.TypeCategoryDecimal, mutation.Columns[0].SQLType.Category)
	})
}

func TestApplyDDL_AlterTable_ConditionalAndConstraints(t *testing.T) {
	t.Parallel()

	t.Run("ADD COLUMN IF NOT EXISTS", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE users ADD COLUMN IF NOT EXISTS age INTEGER`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAddColumn, mutation.Kind)
		require.Len(t, mutation.Columns, 1)
		assert.Equal(t, "age", mutation.Columns[0].Name)
	})

	t.Run("DROP COLUMN IF EXISTS", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE users DROP COLUMN IF EXISTS email`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableDropColumn, mutation.Kind)
		assert.Equal(t, "email", mutation.ColumnName)
	})

	t.Run("ADD CONSTRAINT CHECK", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE users ADD CHECK (age >= 0)`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAddConstraint, mutation.Kind)
	})

	t.Run("ADD CONSTRAINT FOREIGN KEY", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `ALTER TABLE orders ADD FOREIGN KEY (user_id) REFERENCES users (id)`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterTableAddConstraint, mutation.Kind)
		require.Len(t, mutation.Constraints, 1)
		assert.Equal(t, querier_dto.ConstraintForeignKey, mutation.Constraints[0].Kind)
	})
}

func TestApplyDDL_CreateMacro_ArgumentVariants(t *testing.T) {
	t.Parallel()

	t.Run("macro with typed arguments", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE FUNCTION add_nums(a INTEGER, b INTEGER) AS a + b`)

		require.NotNil(t, mutation)
		require.NotNil(t, mutation.FunctionSignature)
		assert.Equal(t, "add_nums", mutation.FunctionSignature.Name)
		require.Len(t, mutation.FunctionSignature.Arguments, 2)
	})

	t.Run("macro with default argument value", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `CREATE MACRO greet(name, greeting DEFAULT 'Hello') AS greeting || ', ' || name`)

		require.NotNil(t, mutation)
		require.NotNil(t, mutation.FunctionSignature)
		assert.Equal(t, "greet", mutation.FunctionSignature.Name)
	})
}

func TestApplyDDL_Comment_SchemaQualified(t *testing.T) {
	t.Parallel()

	t.Run("comment on schema-qualified column", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `COMMENT ON COLUMN myschema.users.email IS 'Email address'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationComment, mutation.Kind)
	})

	t.Run("comment on schema-qualified table", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `COMMENT ON TABLE myschema.users IS 'Users table'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationComment, mutation.Kind)
	})

	t.Run("comment on function with parentheses", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, `COMMENT ON FUNCTION add_one(integer) IS 'Adds one to an integer'`)

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationComment, mutation.Kind)
	})
}

func TestParseStatements_Classification(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()

	t.Run("CREATE TEMP TABLE classified correctly", func(t *testing.T) {
		t.Parallel()

		statements, err := engine.ParseStatements(`CREATE TEMP TABLE t (id INTEGER)`)
		require.NoError(t, err)
		require.Len(t, statements, 1)

		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
	})

	t.Run("CREATE TEMPORARY TABLE classified correctly", func(t *testing.T) {
		t.Parallel()

		statements, err := engine.ParseStatements(`CREATE TEMPORARY TABLE t (id INTEGER)`)
		require.NoError(t, err)
		require.Len(t, statements, 1)

		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
	})

	t.Run("ALTER TYPE classified correctly", func(t *testing.T) {
		t.Parallel()

		statements, err := engine.ParseStatements(`ALTER TYPE mood ADD VALUE 'calm'`)
		require.NoError(t, err)
		require.Len(t, statements, 1)

		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationAlterEnumAddValue, mutation.Kind)
	})

	t.Run("INSTALL classified as DDL returning nil", func(t *testing.T) {
		t.Parallel()

		statements, err := engine.ParseStatements(`INSTALL httpfs`)
		require.NoError(t, err)
		require.Len(t, statements, 1)

		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		assert.Nil(t, mutation, "INSTALL should return nil mutation")
	})

	t.Run("LOAD classified as DDL returning nil", func(t *testing.T) {
		t.Parallel()

		statements, err := engine.ParseStatements(`LOAD httpfs`)
		require.NoError(t, err)
		require.Len(t, statements, 1)

		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		assert.Nil(t, mutation, "LOAD should return nil mutation")
	})

	t.Run("VALUES statement is classified as DML", func(t *testing.T) {
		t.Parallel()

		statements, err := engine.ParseStatements(`VALUES (1, 2), (3, 4)`)
		require.NoError(t, err)
		require.Len(t, statements, 1)

		catalogue := buildTestCatalogue()
		analysis, err := engine.AnalyseQuery(catalogue, statements[0])
		require.NoError(t, err)
		assert.True(t, analysis.ReadOnly)
	})

	t.Run("DROP TABLE TABLE classified correctly", func(t *testing.T) {
		t.Parallel()

		statements, err := engine.ParseStatements(`DROP TABLE t`)
		require.NoError(t, err)
		require.Len(t, statements, 1)

		mutation, err := engine.ApplyDDL(statements[0])
		require.NoError(t, err)
		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationDropTable, mutation.Kind)
	})
}

func TestEngine_Accessors(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()

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
		columns := engine.TableValuedFunctionColumns("generate_series")
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

	t.Run("DefaultSchema", func(t *testing.T) {
		t.Parallel()
		schema := engine.DefaultSchema()
		assert.Equal(t, "main", schema)
	})

	t.Run("Dialect", func(t *testing.T) {
		t.Parallel()
		dialect := engine.Dialect()
		assert.Equal(t, "duckdb", dialect)
	})

	t.Run("SupportsReturning", func(t *testing.T) {
		t.Parallel()
		assert.True(t, engine.SupportsReturning())
	})

	t.Run("ParameterStyle", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, querier_dto.ParameterStyleDollar, engine.ParameterStyle())
	})
}
