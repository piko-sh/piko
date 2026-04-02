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

func applyDDL(t *testing.T, sql string) *querier_dto.CatalogueMutation {
	t.Helper()

	engine := NewPostgresEngine()
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
			name: "simple table with primary key and not null",
			sql:  "CREATE TABLE users (id integer PRIMARY KEY, name text NOT NULL)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "", m.SchemaName)
				require.Len(t, m.Columns, 2)

				assert.Equal(t, "id", m.Columns[0].Name)
				assert.Equal(t, querier_dto.TypeCategoryInteger, m.Columns[0].SQLType.Category)
				assert.Equal(t, "int4", m.Columns[0].SQLType.EngineName)
				assert.False(t, m.Columns[0].Nullable, "primary key column should not be nullable")
				assert.True(t, m.Columns[0].HasDefault, "primary key column should have default flag set")

				assert.Equal(t, "name", m.Columns[1].Name)
				assert.Equal(t, querier_dto.TypeCategoryText, m.Columns[1].SQLType.Category)
				assert.False(t, m.Columns[1].Nullable, "NOT NULL column should not be nullable")

				assert.Equal(t, []string{"id"}, m.PrimaryKey)
			},
		},
		{
			name: "schema-qualified table name",
			sql:  "CREATE TABLE myschema.users (id integer PRIMARY KEY)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "myschema", m.SchemaName)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "IF NOT EXISTS is accepted",
			sql:  "CREATE TABLE IF NOT EXISTS users (id integer PRIMARY KEY)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "users", m.TableName)
				require.Len(t, m.Columns, 1)
			},
		},
		{
			name: "column with DEFAULT expression",
			sql:  "CREATE TABLE t (x int DEFAULT 0)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.True(t, m.Columns[0].HasDefault, "column with DEFAULT should have HasDefault set")
			},
		},
		{
			name: "serial column normalises to int4",
			sql:  "CREATE TABLE t (id serial)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "int4", m.Columns[0].SQLType.EngineName)
				assert.Equal(t, querier_dto.TypeCategoryInteger, m.Columns[0].SQLType.Category)
			},
		},
		{
			name: "array column sets IsArray flag",
			sql:  "CREATE TABLE t (tags text[])",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.True(t, m.Columns[0].IsArray, "text[] column should have IsArray set")
				assert.Equal(t, 1, m.Columns[0].ArrayDimensions)
			},
		},
		{
			name: "INHERITS clause captures parent tables",
			sql:  "CREATE TABLE child_table (extra text) INHERITS (parent_one, parent_two)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "child_table", m.TableName)
				require.Len(t, m.InheritsTables, 2)
				assert.Equal(t, "parent_one", m.InheritsTables[0].Name)
				assert.Equal(t, "parent_two", m.InheritsTables[1].Name)
			},
		},
		{
			name: "table-level FOREIGN KEY constraint",
			sql:  "CREATE TABLE orders (id integer, user_id integer, CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintForeignKey, m.Constraints[0].Kind)
				assert.Equal(t, "fk_user", m.Constraints[0].Name)
				assert.Equal(t, []string{"user_id"}, m.Constraints[0].Columns)
				assert.Equal(t, "users", m.Constraints[0].ForeignTable)
				assert.Equal(t, []string{"id"}, m.Constraints[0].ForeignColumns)
			},
		},
		{
			name: "table-level CHECK constraint",
			sql:  "CREATE TABLE products (id integer, price numeric, CONSTRAINT chk_price CHECK (price > 0))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintCheck, m.Constraints[0].Kind)
				assert.Equal(t, "chk_price", m.Constraints[0].Name)
			},
		},
		{
			name: "table-level UNIQUE constraint",
			sql:  "CREATE TABLE users (id integer, email text, UNIQUE (email))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintUnique, m.Constraints[0].Kind)
				assert.Equal(t, []string{"email"}, m.Constraints[0].Columns)
			},
		},
		{
			name: "table-level composite PRIMARY KEY",
			sql:  "CREATE TABLE order_items (order_id integer, product_id integer, quantity integer, PRIMARY KEY (order_id, product_id))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 3)
				assert.Equal(t, []string{"order_id", "product_id"}, m.PrimaryKey)
			},
		},
		{
			name: "multiple column constraints on one column",
			sql:  "CREATE TABLE t (email text NOT NULL UNIQUE DEFAULT 'unknown')",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.False(t, m.Columns[0].Nullable)
				assert.True(t, m.Columns[0].HasDefault)
			},
		},
		{
			name: "column with REFERENCES constraint",
			sql:  "CREATE TABLE posts (id integer, author_id integer REFERENCES users (id))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
				assert.Equal(t, "author_id", m.Columns[1].Name)
			},
		},
		{
			name: "column with CHECK constraint",
			sql:  "CREATE TABLE t (age integer CHECK (age >= 0))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "age", m.Columns[0].Name)
			},
		},
		{
			name: "column with GENERATED ALWAYS AS IDENTITY",
			sql:  "CREATE TABLE t (id integer GENERATED ALWAYS AS IDENTITY)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.True(t, m.Columns[0].HasDefault)
			},
		},
		{
			name: "column with GENERATED BY DEFAULT AS IDENTITY",
			sql:  "CREATE TABLE t (id integer GENERATED BY DEFAULT AS IDENTITY)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.True(t, m.Columns[0].HasDefault)
			},
		},
		{
			name: "column with GENERATED ALWAYS AS (expression) STORED",
			sql:  "CREATE TABLE t (a integer, b integer, c integer GENERATED ALWAYS AS (a + b) STORED)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 3)
				assert.True(t, m.Columns[2].IsGenerated)
				assert.Equal(t, querier_dto.GeneratedKindStored, m.Columns[2].GeneratedKind)
			},
		},
		{
			name: "TEMP TABLE is accepted",
			sql:  "CREATE TEMP TABLE scratch (id integer)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "scratch", m.TableName)
			},
		},
		{
			name: "UNLOGGED TABLE is accepted",
			sql:  "CREATE UNLOGGED TABLE events (id integer, payload text)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "events", m.TableName)
			},
		},
		{
			name: "CREATE TABLE AS is accepted",
			sql:  "CREATE TABLE summary AS SELECT count(*) FROM users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				assert.Equal(t, "summary", m.TableName)
			},
		},
		{
			name: "timestamp with time zone column type",
			sql:  "CREATE TABLE t (created_at timestamp with time zone)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "timestamptz", m.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "character varying column type with length",
			sql:  "CREATE TABLE t (name character varying(100))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "varchar", m.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "double precision column type",
			sql:  "CREATE TABLE t (val double precision)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "float8", m.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "numeric with precision and scale",
			sql:  "CREATE TABLE t (price numeric(10,2))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "numeric", m.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "column with COLLATE",
			sql:  `CREATE TABLE t (name text COLLATE "en_GB")`,
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "name", m.Columns[0].Name)
			},
		},
		{
			name: "column explicitly marked NULL",
			sql:  "CREATE TABLE t (description text NULL)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.True(t, m.Columns[0].Nullable, "NULL column should be nullable")
			},
		},
		{
			name: "multi-dimensional array column",
			sql:  "CREATE TABLE t (matrix integer[][])",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.True(t, m.Columns[0].IsArray)
				assert.Equal(t, 2, m.Columns[0].ArrayDimensions)
			},
		},
		{
			name: "multiple table-level constraints in one table",
			sql: `CREATE TABLE inventory (
				id integer,
				warehouse_id integer,
				product_id integer,
				quantity integer,
				PRIMARY KEY (id),
				CONSTRAINT fk_warehouse FOREIGN KEY (warehouse_id) REFERENCES warehouses (id),
				CHECK (quantity >= 0)
			)`,
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 4)
				assert.Equal(t, []string{"id"}, m.PrimaryKey)

				require.Len(t, m.Constraints, 2)
				assert.Equal(t, querier_dto.ConstraintForeignKey, m.Constraints[0].Kind)
				assert.Equal(t, querier_dto.ConstraintCheck, m.Constraints[1].Kind)
			},
		},
		{
			name: "time without time zone column type",
			sql:  "CREATE TABLE t (event_time time without time zone)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "time", m.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "time with time zone column type",
			sql:  "CREATE TABLE t (event_time time with time zone)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "timetz", m.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "timestamp without time zone column type",
			sql:  "CREATE TABLE t (created_at timestamp without time zone)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "timestamp", m.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "column with DEFAULT expression involving function call",
			sql:  "CREATE TABLE t (created_at timestamp DEFAULT now())",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.True(t, m.Columns[0].HasDefault)
			},
		},
		{
			name: "EXCLUDE constraint is accepted",
			sql:  "CREATE TABLE t (id integer, period tsrange, EXCLUDE USING gist (period WITH &&))",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 2)
			},
		},
		{
			name: "bigserial column normalises to int8",
			sql:  "CREATE TABLE t (id bigserial)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "int8", m.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "boolean column type",
			sql:  "CREATE TABLE t (active boolean NOT NULL DEFAULT true)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, querier_dto.TypeCategoryBoolean, m.Columns[0].SQLType.Category)
				assert.False(t, m.Columns[0].Nullable)
				assert.True(t, m.Columns[0].HasDefault)
			},
		},
		{
			name: "uuid column type",
			sql:  "CREATE TABLE t (id uuid PRIMARY KEY)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "uuid", m.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "jsonb column type",
			sql:  "CREATE TABLE t (data jsonb)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "jsonb", m.Columns[0].SQLType.EngineName)
			},
		},
		{
			name: "schema-qualified type for column",
			sql:  "CREATE TABLE t (status myschema.status_type)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "myschema", m.Columns[0].SQLType.Schema)
			},
		},
		{
			name: "FOREIGN KEY with ON DELETE CASCADE action",
			sql:  "CREATE TABLE orders (id integer, user_id integer, CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintForeignKey, m.Constraints[0].Kind)
			},
		},
		{
			name: "INHERITS with schema-qualified parent",
			sql:  "CREATE TABLE child (extra text) INHERITS (myschema.parent)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTable, m.Kind)
				require.Len(t, m.InheritsTables, 1)
				assert.Equal(t, "myschema", m.InheritsTables[0].Schema)
				assert.Equal(t, "parent", m.InheritsTables[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
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
			sql:  "ALTER TABLE users ADD COLUMN email text",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddColumn, m.Kind)
				assert.Equal(t, "users", m.TableName)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "email", m.Columns[0].Name)
				assert.Equal(t, querier_dto.TypeCategoryText, m.Columns[0].SQLType.Category)
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
		{
			name: "ALTER COLUMN TYPE",
			sql:  "ALTER TABLE users ALTER COLUMN name TYPE varchar(255)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAlterColumn, m.Kind)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "name", m.ColumnName)
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
			name: "RENAME TO (table rename)",
			sql:  "ALTER TABLE users RENAME TO accounts",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableRenameTable, m.Kind)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "accounts", m.NewName)
			},
		},
		{
			name: "ADD CONSTRAINT",
			sql:  "ALTER TABLE users ADD CONSTRAINT uq_email UNIQUE (email)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddConstraint, m.Kind)
				assert.Equal(t, "users", m.TableName)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintUnique, m.Constraints[0].Kind)
				assert.Equal(t, "uq_email", m.Constraints[0].Name)
			},
		},
		{
			name: "DROP CONSTRAINT",
			sql:  "ALTER TABLE users DROP CONSTRAINT uq_email",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableDropConstraint, m.Kind)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "uq_email", m.ConstraintName)
			},
		},
		{
			name: "SET SCHEMA moves table to new schema",
			sql:  "ALTER TABLE public.users SET SCHEMA archive",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableSetSchema, m.Kind)
				assert.Equal(t, "public", m.SchemaName)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "archive", m.NewName)
			},
		},
		{
			name: "ADD COLUMN IF NOT EXISTS",
			sql:  "ALTER TABLE users ADD COLUMN IF NOT EXISTS nickname text",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddColumn, m.Kind)
				assert.Equal(t, "users", m.TableName)
				require.Len(t, m.Columns, 1)
				assert.Equal(t, "nickname", m.Columns[0].Name)
			},
		},
		{
			name: "DROP COLUMN IF EXISTS",
			sql:  "ALTER TABLE users DROP COLUMN IF EXISTS legacy_field",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableDropColumn, m.Kind)
				assert.Equal(t, "legacy_field", m.ColumnName)
			},
		},
		{
			name: "DROP CONSTRAINT IF EXISTS CASCADE",
			sql:  "ALTER TABLE users DROP CONSTRAINT IF EXISTS uq_email CASCADE",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableDropConstraint, m.Kind)
				assert.Equal(t, "uq_email", m.ConstraintName)
			},
		},
		{
			name: "schema-qualified ALTER TABLE",
			sql:  "ALTER TABLE myschema.users ADD COLUMN bio text",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddColumn, m.Kind)
				assert.Equal(t, "myschema", m.SchemaName)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "ALTER TABLE ONLY accepted",
			sql:  "ALTER TABLE ONLY users ADD COLUMN status text",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddColumn, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "ADD CONSTRAINT with FOREIGN KEY",
			sql:  "ALTER TABLE posts ADD CONSTRAINT fk_author FOREIGN KEY (user_id) REFERENCES users (id)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddConstraint, m.Kind)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintForeignKey, m.Constraints[0].Kind)
				assert.Equal(t, "fk_author", m.Constraints[0].Name)
				assert.Equal(t, "users", m.Constraints[0].ForeignTable)
			},
		},
		{
			name: "ADD CONSTRAINT with CHECK",
			sql:  "ALTER TABLE products ADD CONSTRAINT chk_positive CHECK (price > 0)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterTableAddConstraint, m.Kind)
				require.Len(t, m.Constraints, 1)
				assert.Equal(t, querier_dto.ConstraintCheck, m.Constraints[0].Kind)
				assert.Equal(t, "chk_positive", m.Constraints[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
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
			name: "simple drop",
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
		{
			name: "DROP TABLE CASCADE",
			sql:  "DROP TABLE users CASCADE",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTable, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "DROP TABLE schema-qualified",
			sql:  "DROP TABLE myschema.users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTable, m.Kind)
				assert.Equal(t, "myschema", m.SchemaName)
				assert.Equal(t, "users", m.TableName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
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
			sql:  "CREATE VIEW active_users AS SELECT id, name FROM users WHERE active = true",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, m.Kind)
				assert.Equal(t, "active_users", m.TableName)
				assert.NotNil(t, m.ViewDefinition, "view should have a parsed definition")
			},
		},
		{
			name: "CREATE OR REPLACE VIEW",
			sql:  "CREATE OR REPLACE VIEW active_users AS SELECT id FROM users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, m.Kind)
				assert.Equal(t, "active_users", m.TableName)
				assert.NotNil(t, m.ViewDefinition)
			},
		},
		{
			name: "CREATE MATERIALIZED VIEW",
			sql:  "CREATE MATERIALIZED VIEW user_stats AS SELECT count(*) FROM users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, m.Kind)
				assert.Equal(t, "user_stats", m.TableName)
				assert.NotNil(t, m.ViewDefinition)
			},
		},
		{
			name: "view with schema qualification",
			sql:  "CREATE VIEW reporting.monthly_totals AS SELECT 1",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, m.Kind)
				assert.Equal(t, "reporting", m.SchemaName)
				assert.Equal(t, "monthly_totals", m.TableName)
			},
		},
		{
			name: "view with explicit column list",
			sql:  "CREATE VIEW user_count (cnt) AS SELECT count(*) FROM users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, m.Kind)
				assert.Equal(t, "user_count", m.TableName)
				require.NotNil(t, m.ViewDefinition)
			},
		},
		{
			name: "CREATE MATERIALIZED VIEW IF NOT EXISTS",
			sql:  "CREATE MATERIALIZED VIEW IF NOT EXISTS user_stats AS SELECT count(*) FROM users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateView, m.Kind)
				assert.Equal(t, "user_stats", m.TableName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
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
			name: "simple drop view",
			sql:  "DROP VIEW active_users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropView, m.Kind)
				assert.Equal(t, "active_users", m.TableName)
			},
		},
		{
			name: "DROP MATERIALIZED VIEW",
			sql:  "DROP MATERIALIZED VIEW user_stats",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropView, m.Kind)
				assert.Equal(t, "user_stats", m.TableName)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
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
				assert.Equal(t, "idx_users_name", m.NewName)
			},
		},
		{
			name: "unique index",
			sql:  "CREATE UNIQUE INDEX idx_users_email ON users (email)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, m.Kind)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "idx_users_email", m.NewName)
			},
		},
		{
			name: "index with USING method",
			sql:  "CREATE INDEX idx_users_name ON users USING btree (name)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "index with CONCURRENTLY",
			sql:  "CREATE INDEX CONCURRENTLY idx_users_name ON users (name)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "index IF NOT EXISTS",
			sql:  "CREATE INDEX IF NOT EXISTS idx_users_name ON users (name)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "index on schema-qualified table",
			sql:  "CREATE INDEX idx_name ON myschema.users (name)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateIndex, m.Kind)
				assert.Equal(t, "myschema", m.SchemaName)
				assert.Equal(t, "users", m.TableName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
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
			name: "simple drop index",
			sql:  "DROP INDEX idx_users_name",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropIndex, m.Kind)
				assert.Equal(t, "idx_users_name", m.NewName)
			},
		},
		{
			name: "DROP INDEX CONCURRENTLY IF EXISTS",
			sql:  "DROP INDEX CONCURRENTLY IF EXISTS idx_users_name",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropIndex, m.Kind)
				assert.Equal(t, "idx_users_name", m.NewName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_CreateType(t *testing.T) {
	t.Parallel()

	t.Run("enum type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, "CREATE TYPE mood AS ENUM ('happy', 'sad', 'neutral')")

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateEnum, mutation.Kind)
		assert.Equal(t, "mood", mutation.EnumName)
		assert.Equal(t, []string{"happy", "sad", "neutral"}, mutation.EnumValues)
	})

	t.Run("composite type", func(t *testing.T) {
		t.Parallel()

		mutation := applyDDL(t, "CREATE TYPE address AS (street text, city text, postcode text)")

		require.NotNil(t, mutation)
		assert.Equal(t, querier_dto.MutationCreateCompositeType, mutation.Kind)
		assert.Equal(t, "address", mutation.EnumName)
		require.Len(t, mutation.Columns, 3)
		assert.Equal(t, "street", mutation.Columns[0].Name)
		assert.Equal(t, "city", mutation.Columns[1].Name)
		assert.Equal(t, "postcode", mutation.Columns[2].Name)
	})
}

func TestApplyDDL_AlterType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "ADD VALUE",
			sql:  "ALTER TYPE mood ADD VALUE 'anxious'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterEnumAddValue, m.Kind)
				assert.Equal(t, "mood", m.EnumName)
				assert.Equal(t, []string{"anxious"}, m.EnumValues)
			},
		},
		{
			name: "ADD VALUE IF NOT EXISTS",
			sql:  "ALTER TYPE mood ADD VALUE IF NOT EXISTS 'anxious'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterEnumAddValue, m.Kind)
				assert.Equal(t, "mood", m.EnumName)
				assert.Equal(t, []string{"anxious"}, m.EnumValues)
			},
		},
		{
			name: "ADD VALUE BEFORE",
			sql:  "ALTER TYPE mood ADD VALUE 'anxious' BEFORE 'sad'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterEnumAddValue, m.Kind)
				assert.Equal(t, "mood", m.EnumName)
				assert.Equal(t, []string{"anxious"}, m.EnumValues)
			},
		},
		{
			name: "ADD VALUE AFTER",
			sql:  "ALTER TYPE mood ADD VALUE 'anxious' AFTER 'happy'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterEnumAddValue, m.Kind)
				assert.Equal(t, "mood", m.EnumName)
				assert.Equal(t, []string{"anxious"}, m.EnumValues)
			},
		},
		{
			name: "RENAME VALUE",
			sql:  "ALTER TYPE mood RENAME VALUE 'sad' TO 'melancholy'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterEnumRenameValue, m.Kind)
				assert.Equal(t, "mood", m.EnumName)
				assert.Equal(t, []string{"sad", "melancholy"}, m.EnumValues)
			},
		},
		{
			name: "schema-qualified ALTER TYPE",
			sql:  "ALTER TYPE myschema.mood ADD VALUE 'excited'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationAlterEnumAddValue, m.Kind)
				assert.Equal(t, "myschema", m.SchemaName)
				assert.Equal(t, "mood", m.EnumName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_CreateFunction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple function with RETURNS and LANGUAGE",
			sql:  `CREATE FUNCTION add_numbers(a integer, b integer) RETURNS integer LANGUAGE sql AS 'SELECT a + b'`,
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "add_numbers", m.FunctionSignature.Name)
				require.Len(t, m.FunctionSignature.Arguments, 2)
				assert.Equal(t, "a", m.FunctionSignature.Arguments[0].Name)
				assert.Equal(t, "b", m.FunctionSignature.Arguments[1].Name)
				assert.Equal(t, querier_dto.TypeCategoryInteger, m.FunctionSignature.ReturnType.Category)
				assert.Equal(t, "sql", m.FunctionSignature.Language)
			},
		},
		{
			name: "function RETURNS TABLE",
			sql:  "CREATE FUNCTION get_users() RETURNS TABLE (id integer, name text) LANGUAGE sql AS 'SELECT id, name FROM users'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "get_users", m.FunctionSignature.Name)
				assert.True(t, m.FunctionSignature.ReturnsSet, "RETURNS TABLE should set ReturnsSet")
				assert.Equal(t, "sql", m.FunctionSignature.Language)
			},
		},
		{
			name: "function RETURNS SETOF",
			sql:  "CREATE FUNCTION all_users() RETURNS SETOF users LANGUAGE sql AS 'SELECT * FROM users'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.True(t, m.FunctionSignature.ReturnsSet, "RETURNS SETOF should set ReturnsSet")
			},
		},
		{
			name: "function with STRICT",
			sql:  "CREATE FUNCTION safe_div(a integer, b integer) RETURNS integer LANGUAGE sql STRICT AS 'SELECT a / b'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.True(t, m.FunctionSignature.IsStrict, "STRICT should set IsStrict")
			},
		},
		{
			name: "function with RETURNS NULL ON NULL INPUT",
			sql:  "CREATE FUNCTION safe_div(a integer, b integer) RETURNS integer LANGUAGE sql RETURNS NULL ON NULL INPUT AS 'SELECT a / b'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.True(t, m.FunctionSignature.IsStrict, "RETURNS NULL ON NULL INPUT should set IsStrict")
			},
		},
		{
			name: "function with CALLED ON NULL INPUT",
			sql:  "CREATE FUNCTION maybe_null(a integer) RETURNS integer LANGUAGE sql CALLED ON NULL INPUT AS 'SELECT a'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.False(t, m.FunctionSignature.IsStrict, "CALLED ON NULL INPUT should not set IsStrict")
			},
		},
		{
			name: "function with IMMUTABLE",
			sql:  "CREATE FUNCTION constant_value() RETURNS integer LANGUAGE sql IMMUTABLE AS 'SELECT 42'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, querier_dto.DataAccessReadOnly, m.FunctionSignature.DataAccess)
			},
		},
		{
			name: "function with STABLE",
			sql:  "CREATE FUNCTION current_count() RETURNS integer LANGUAGE sql STABLE AS 'SELECT count(*) FROM users'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, querier_dto.DataAccessReadOnly, m.FunctionSignature.DataAccess)
			},
		},
		{
			name: "function with VOLATILE",
			sql:  "CREATE FUNCTION do_something() RETURNS void LANGUAGE sql VOLATILE AS 'SELECT 1'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, querier_dto.DataAccessModifiesData, m.FunctionSignature.DataAccess)
			},
		},
		{
			name: "CREATE OR REPLACE FUNCTION",
			sql:  "CREATE OR REPLACE FUNCTION greet(name text) RETURNS text LANGUAGE sql AS 'SELECT name'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "greet", m.FunctionSignature.Name)
				require.Len(t, m.FunctionSignature.Arguments, 1)
				assert.Equal(t, "name", m.FunctionSignature.Arguments[0].Name)
			},
		},
		{
			name: "function with LANGUAGE plpgsql and dollar-quoted body",
			sql:  `CREATE FUNCTION increment(val integer) RETURNS integer LANGUAGE plpgsql AS $$ BEGIN RETURN val + 1; END; $$`,
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "plpgsql", m.FunctionSignature.Language)
				assert.NotEmpty(t, m.FunctionSignature.BodySQL)
			},
		},
		{
			name: "function with no arguments",
			sql:  "CREATE FUNCTION now_utc() RETURNS timestamp LANGUAGE sql AS 'SELECT now()'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "now_utc", m.FunctionSignature.Name)
				assert.Empty(t, m.FunctionSignature.Arguments)
			},
		},
		{
			name: "function with unnamed arguments",
			sql:  "CREATE FUNCTION add_ints(integer, integer) RETURNS integer LANGUAGE sql AS 'SELECT $1 + $2'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				require.Len(t, m.FunctionSignature.Arguments, 2)

				assert.Empty(t, m.FunctionSignature.Arguments[0].Name)
				assert.Empty(t, m.FunctionSignature.Arguments[1].Name)
			},
		},
		{
			name: "function with multiple clauses combined",
			sql:  "CREATE FUNCTION safe_add(a integer, b integer) RETURNS integer LANGUAGE sql IMMUTABLE STRICT AS 'SELECT a + b'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.True(t, m.FunctionSignature.IsStrict)
				assert.Equal(t, querier_dto.DataAccessReadOnly, m.FunctionSignature.DataAccess)
				assert.Equal(t, "sql", m.FunctionSignature.Language)
			},
		},
		{
			name: "schema-qualified function",
			sql:  "CREATE FUNCTION myschema.my_func() RETURNS void LANGUAGE sql AS 'SELECT 1'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "myschema", m.SchemaName)
				assert.Equal(t, "my_func", m.FunctionSignature.Name)
			},
		},
		{
			name: "CREATE PROCEDURE",
			sql:  "CREATE PROCEDURE do_cleanup() LANGUAGE sql AS 'DELETE FROM old_data'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "do_cleanup", m.FunctionSignature.Name)
			},
		},
		{
			name: "function with VARIADIC argument",
			sql:  "CREATE FUNCTION concat_all(VARIADIC items text[]) RETURNS text LANGUAGE sql AS 'SELECT array_to_string(items, '''')'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.True(t, m.FunctionSignature.IsVariadic)
			},
		},
		{
			name: "function with DEFAULT argument",
			sql:  "CREATE FUNCTION greet(name text DEFAULT 'world') RETURNS text LANGUAGE sql AS 'SELECT name'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				require.Len(t, m.FunctionSignature.Arguments, 1)
				assert.True(t, m.FunctionSignature.Arguments[0].IsOptional)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_DropFunction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "drop function with argument types",
			sql:  "DROP FUNCTION add_numbers(integer, integer)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "add_numbers", m.FunctionSignature.Name)
			},
		},
		{
			name: "drop function without argument types",
			sql:  "DROP FUNCTION simple_func",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "simple_func", m.FunctionSignature.Name)
			},
		},
		{
			name: "DROP FUNCTION IF EXISTS",
			sql:  "DROP FUNCTION IF EXISTS maybe_exists(text)",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "maybe_exists", m.FunctionSignature.Name)
			},
		},
		{
			name: "DROP FUNCTION CASCADE",
			sql:  "DROP FUNCTION my_func(integer) CASCADE",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "my_func", m.FunctionSignature.Name)
			},
		},
		{
			name: "DROP PROCEDURE",
			sql:  "DROP PROCEDURE cleanup_proc()",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropFunction, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "cleanup_proc", m.FunctionSignature.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_CreateSchema(t *testing.T) {
	t.Parallel()

	mutation := applyDDL(t, "CREATE SCHEMA analytics")

	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateSchema, mutation.Kind)
	assert.Equal(t, "analytics", mutation.SchemaName)
}

func TestApplyDDL_CreateExtension(t *testing.T) {
	t.Parallel()

	mutation := applyDDL(t, "CREATE EXTENSION IF NOT EXISTS pgcrypto")

	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateExtension, mutation.Kind)
	assert.Equal(t, "pgcrypto", mutation.NewName)
}

func TestApplyDDL_CreateSequence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple sequence",
			sql:  "CREATE SEQUENCE user_id_seq",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateSequence, m.Kind)
				assert.Equal(t, "user_id_seq", m.SequenceName)
			},
		},
		{
			name: "sequence with options",
			sql:  "CREATE SEQUENCE order_seq START 1000 INCREMENT 10 MINVALUE 1 MAXVALUE 99999",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateSequence, m.Kind)
				assert.Equal(t, "order_seq", m.SequenceName)
			},
		},
		{
			name: "sequence with OWNED BY",
			sql:  "CREATE SEQUENCE user_id_seq OWNED BY users.id",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateSequence, m.Kind)
				assert.Equal(t, "user_id_seq", m.SequenceName)
				assert.Equal(t, "users", m.OwnedByTable)
				assert.Equal(t, "id", m.OwnedByColumn)
			},
		},
		{
			name: "sequence IF NOT EXISTS",
			sql:  "CREATE SEQUENCE IF NOT EXISTS counter_seq",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateSequence, m.Kind)
				assert.Equal(t, "counter_seq", m.SequenceName)
			},
		},
		{
			name: "sequence with schema qualification",
			sql:  "CREATE SEQUENCE myschema.my_seq",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateSequence, m.Kind)
				assert.Equal(t, "myschema", m.SchemaName)
				assert.Equal(t, "my_seq", m.SequenceName)
			},
		},
		{
			name: "sequence OWNED BY NONE",
			sql:  "CREATE SEQUENCE detached_seq OWNED BY NONE",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateSequence, m.Kind)
				assert.Equal(t, "detached_seq", m.SequenceName)
				assert.Empty(t, m.OwnedByTable, "OWNED BY NONE should leave OwnedByTable empty")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
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
			name: "simple BEFORE INSERT trigger",
			sql:  "CREATE TRIGGER audit_insert BEFORE INSERT ON users FOR EACH ROW EXECUTE FUNCTION audit_func()",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, m.Kind)
				assert.Equal(t, "audit_insert", m.TriggerName)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "AFTER UPDATE trigger",
			sql:  "CREATE TRIGGER after_update AFTER UPDATE ON posts FOR EACH ROW EXECUTE FUNCTION notify()",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, m.Kind)
				assert.Equal(t, "after_update", m.TriggerName)
				assert.Equal(t, "posts", m.TableName)
			},
		},
		{
			name: "INSTEAD OF trigger",
			sql:  "CREATE TRIGGER view_insert INSTEAD OF INSERT ON active_users FOR EACH ROW EXECUTE FUNCTION insert_user()",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateTrigger, m.Kind)
				assert.Equal(t, "view_insert", m.TriggerName)
				assert.Equal(t, "active_users", m.TableName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
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
			name: "simple drop trigger",
			sql:  "DROP TRIGGER audit_insert ON users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTrigger, m.Kind)
				assert.Equal(t, "audit_insert", m.TriggerName)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "DROP TRIGGER IF EXISTS",
			sql:  "DROP TRIGGER IF EXISTS audit_insert ON users",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropTrigger, m.Kind)
				assert.Equal(t, "audit_insert", m.TriggerName)
				assert.Equal(t, "users", m.TableName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_DropType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple drop type",
			sql:  "DROP TYPE mood",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropType, m.Kind)
				assert.Equal(t, "mood", m.EnumName)
			},
		},
		{
			name: "DROP TYPE IF EXISTS CASCADE",
			sql:  "DROP TYPE IF EXISTS mood CASCADE",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropType, m.Kind)
				assert.Equal(t, "mood", m.EnumName)
			},
		},
		{
			name: "DROP TYPE schema-qualified",
			sql:  "DROP TYPE myschema.address",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropType, m.Kind)
				assert.Equal(t, "myschema", m.SchemaName)
				assert.Equal(t, "address", m.EnumName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_DropSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple drop schema",
			sql:  "DROP SCHEMA analytics",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropSchema, m.Kind)
				assert.Equal(t, "analytics", m.SchemaName)
			},
		},
		{
			name: "DROP SCHEMA IF EXISTS CASCADE",
			sql:  "DROP SCHEMA IF EXISTS old_data CASCADE",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropSchema, m.Kind)
				assert.Equal(t, "old_data", m.SchemaName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_DropSequence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "simple drop sequence",
			sql:  "DROP SEQUENCE user_id_seq",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropSequence, m.Kind)
				assert.Equal(t, "user_id_seq", m.SequenceName)
			},
		},
		{
			name: "DROP SEQUENCE IF EXISTS",
			sql:  "DROP SEQUENCE IF EXISTS user_id_seq",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropSequence, m.Kind)
				assert.Equal(t, "user_id_seq", m.SequenceName)
			},
		},
		{
			name: "DROP SEQUENCE CASCADE",
			sql:  "DROP SEQUENCE user_id_seq CASCADE",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationDropSequence, m.Kind)
				assert.Equal(t, "user_id_seq", m.SequenceName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_CreateExtension_Options(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "extension with SCHEMA",
			sql:  "CREATE EXTENSION hstore WITH SCHEMA public",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateExtension, m.Kind)
				assert.Equal(t, "hstore", m.NewName)
				assert.Equal(t, "public", m.SchemaName)
			},
		},
		{
			name: "extension without IF NOT EXISTS",
			sql:  "CREATE EXTENSION postgis",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateExtension, m.Kind)
				assert.Equal(t, "postgis", m.NewName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_DropExtension(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()
	stmts, err := engine.ParseStatements("DROP EXTENSION IF EXISTS pgcrypto")
	require.NoError(t, err)
	require.NotEmpty(t, stmts)

	mutation, err := engine.ApplyDDL(stmts[0])
	require.NoError(t, err)

	assert.Nil(t, mutation)
}

func TestApplyDDL_CreateSchema_Options(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "CREATE SCHEMA IF NOT EXISTS",
			sql:  "CREATE SCHEMA IF NOT EXISTS staging",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateSchema, m.Kind)
				assert.Equal(t, "staging", m.SchemaName)
			},
		},
		{
			name: "CREATE SCHEMA AUTHORIZATION",
			sql:  "CREATE SCHEMA AUTHORIZATION admin_role",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationCreateSchema, m.Kind)
				assert.Equal(t, "admin_role", m.SchemaName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_Comment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sql        string
		assertions func(t *testing.T, m *querier_dto.CatalogueMutation)
	}{
		{
			name: "COMMENT ON TABLE",
			sql:  "COMMENT ON TABLE users IS 'The main users table'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationComment, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "COMMENT ON COLUMN",
			sql:  "COMMENT ON COLUMN users.email IS 'Primary email address'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationComment, m.Kind)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "email", m.ColumnName)
			},
		},
		{
			name: "COMMENT ON TYPE",
			sql:  "COMMENT ON TYPE mood IS 'User mood enumeration'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationComment, m.Kind)
				assert.Equal(t, "mood", m.EnumName)
			},
		},
		{
			name: "COMMENT ON FUNCTION",
			sql:  "COMMENT ON FUNCTION add_numbers(integer, integer) IS 'Adds two numbers'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationComment, m.Kind)
				require.NotNil(t, m.FunctionSignature)
				assert.Equal(t, "add_numbers", m.FunctionSignature.Name)
			},
		},
		{
			name: "COMMENT ON SCHEMA",
			sql:  "COMMENT ON SCHEMA public IS 'Default schema'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationComment, m.Kind)
				assert.Equal(t, "public", m.SchemaName)
			},
		},
		{
			name: "COMMENT IS NULL removes comment",
			sql:  "COMMENT ON TABLE users IS NULL",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationComment, m.Kind)
				assert.Equal(t, "users", m.TableName)
			},
		},
		{
			name: "COMMENT ON COLUMN with schema-qualified name",
			sql:  "COMMENT ON COLUMN public.users.email IS 'User email address'",
			assertions: func(t *testing.T, m *querier_dto.CatalogueMutation) {
				assert.Equal(t, querier_dto.MutationComment, m.Kind)
				assert.Equal(t, "public", m.SchemaName)
				assert.Equal(t, "users", m.TableName)
				assert.Equal(t, "email", m.ColumnName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := applyDDL(t, tt.sql)
			require.NotNil(t, mutation)
			tt.assertions(t, mutation)
		})
	}
}

func TestApplyDDL_NonDDL(t *testing.T) {
	t.Parallel()

	mutation := applyDDL(t, "SELECT 1")

	assert.Nil(t, mutation, "non-DDL statement should return nil mutation")
}
