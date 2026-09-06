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

package db_catalogue_sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	testIntrospectionTimeout = 30 * time.Second
	testSchema               = `
CREATE TABLE users (
	id INTEGER PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	name TEXT,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	name_upper TEXT GENERATED ALWAYS AS (upper(name)) VIRTUAL,
	name_length INTEGER GENERATED ALWAYS AS (length(name)) STORED
);
CREATE INDEX users_name_created ON users (name, created_at);
CREATE INDEX users_recent_name ON users (name) WHERE created_at IS NOT NULL;
CREATE INDEX users_name_lower_email ON users (name, lower(email));
CREATE UNIQUE INDEX users_lower_email ON users (lower(email));
CREATE TABLE memberships (
	user_id INTEGER NOT NULL,
	group_id INTEGER NOT NULL,
	PRIMARY KEY (group_id, user_id)
) WITHOUT ROWID;
CREATE TABLE sqliteapp (value TEXT);
CREATE TABLE audit (id INTEGER PRIMARY KEY AUTOINCREMENT, message TEXT);
CREATE VIEW user_emails AS SELECT id, email FROM users;
CREATE TRIGGER users_audit AFTER INSERT ON users BEGIN
	INSERT INTO audit (message) VALUES (NEW.email);
END;
`
)

type lowerCaseTypeNormaliser struct{}

func (lowerCaseTypeNormaliser) NormaliseTypeName(name string, _ ...int) querier_dto.SQLType {
	return querier_dto.NewSQLType(querier_dto.TypeCategoryUnknown, strings.ToLower(name))
}

func openSingleConnectionDatabase(t *testing.T, schema string) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite", "file::memory:")
	require.NoError(t, err)
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { assert.NoError(t, database.Close()) })

	if schema != "" {
		_, err = database.ExecContext(t.Context(), schema)
		require.NoError(t, err)
	}

	return database
}

func buildTestCatalogue(t *testing.T, database *sql.DB) *querier_dto.Schema {
	t.Helper()

	ctx, cancel := context.WithTimeoutCause(t.Context(), testIntrospectionTimeout,
		errors.New("catalogue introspection did not finish in time"))
	defer cancel()

	provider := NewPragmaIntrospectionProvider(database, lowerCaseTypeNormaliser{})
	catalogue, sourceErrors, err := provider.BuildCatalogue(ctx)
	require.NoError(t, err)
	assert.Nil(t, sourceErrors)
	require.NotNil(t, catalogue)
	assert.Equal(t, schemaMain, catalogue.DefaultSchema)

	schema := catalogue.Schemas[schemaMain]
	require.NotNil(t, schema)
	return schema
}

func indexesByName(indexes []querier_dto.Index) map[string]querier_dto.Index {
	byName := make(map[string]querier_dto.Index, len(indexes))
	for _, index := range indexes {
		byName[index.Name] = index
	}
	return byName
}

func TestBuildCatalogueListsUserTablesAndViews(t *testing.T) {
	schema := buildTestCatalogue(t, openSingleConnectionDatabase(t, testSchema))

	tableNames := make([]string, 0, len(schema.Tables))
	for name := range schema.Tables {
		tableNames = append(tableNames, name)
	}
	assert.ElementsMatch(t, []string{"audit", "memberships", "sqliteapp", "users"}, tableNames)

	require.Contains(t, schema.Views, "user_emails")
	view := schema.Views["user_emails"]
	assert.Equal(t, "user_emails", view.Name)
	require.Len(t, view.Columns, 2)
	assert.Equal(t, "id", view.Columns[0].Name)
	assert.Equal(t, "email", view.Columns[1].Name)
}

func TestBuildCatalogueDescribesColumns(t *testing.T) {
	schema := buildTestCatalogue(t, openSingleConnectionDatabase(t, testSchema))

	users := schema.Tables["users"]
	require.NotNil(t, users)
	assert.Equal(t, schemaMain, users.Schema)
	assert.Equal(t, []string{"id"}, users.PrimaryKey)

	testCases := []struct {
		name              string
		wantType          string
		wantGeneratedKind querier_dto.GeneratedKind
		wantNullable      bool
		wantHasDefault    bool
		wantGenerated     bool
	}{
		{name: "id", wantType: "integer", wantNullable: false, wantHasDefault: true},
		{name: "email", wantType: "text", wantNullable: false, wantHasDefault: false},
		{name: "name", wantType: "text", wantNullable: true, wantHasDefault: false},
		{name: "created_at", wantType: "text", wantNullable: false, wantHasDefault: true},
		{
			name:              "name_upper",
			wantType:          "text",
			wantNullable:      true,
			wantGenerated:     true,
			wantGeneratedKind: querier_dto.GeneratedKindVirtual,
		},
		{
			name:              "name_length",
			wantType:          "integer",
			wantNullable:      true,
			wantGenerated:     true,
			wantGeneratedKind: querier_dto.GeneratedKindStored,
		},
	}

	require.Len(t, users.Columns, len(testCases))
	for index, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			column := users.Columns[index]
			assert.Equal(t, testCase.name, column.Name)
			assert.Equal(t, testCase.wantType, column.SQLType.EngineName)
			assert.Equal(t, testCase.wantNullable, column.Nullable)
			assert.Equal(t, testCase.wantHasDefault, column.HasDefault)
			assert.Equal(t, testCase.wantGenerated, column.IsGenerated)
			assert.Equal(t, testCase.wantGeneratedKind, column.GeneratedKind)
		})
	}
}

func TestBuildCatalogueOrdersCompositePrimaryKey(t *testing.T) {
	schema := buildTestCatalogue(t, openSingleConnectionDatabase(t, testSchema))

	memberships := schema.Tables["memberships"]
	require.NotNil(t, memberships)
	assert.Equal(t, []string{"group_id", "user_id"}, memberships.PrimaryKey)

	sqliteApp := schema.Tables["sqliteapp"]
	require.NotNil(t, sqliteApp)
	assert.Nil(t, sqliteApp.PrimaryKey)
}

func TestBuildCatalogueDescribesIndexes(t *testing.T) {
	schema := buildTestCatalogue(t, openSingleConnectionDatabase(t, testSchema))

	testCases := []struct {
		name        string
		table       string
		index       string
		wantColumns []string
		wantUnique  bool
		wantPrimary bool
	}{
		{
			name:        "unique constraint creates an automatic unique index",
			table:       "users",
			index:       "sqlite_autoindex_users_1",
			wantColumns: []string{"email"},
			wantUnique:  true,
		},
		{
			name:        "multi-column index keeps declaration order",
			table:       "users",
			index:       "users_name_created",
			wantColumns: []string{"name", "created_at"},
		},
		{
			name:        "partial index reports its key columns",
			table:       "users",
			index:       "users_recent_name",
			wantColumns: []string{"name"},
		},
		{
			name:        "mixed index keeps only its named columns",
			table:       "users",
			index:       "users_name_lower_email",
			wantColumns: []string{"name"},
		},
		{
			name:        "primary key of a table without rowid is a primary index",
			table:       "memberships",
			index:       "sqlite_autoindex_memberships_1",
			wantColumns: []string{"group_id", "user_id"},
			wantUnique:  true,
			wantPrimary: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			table := schema.Tables[testCase.table]
			require.NotNil(t, table)

			index, found := indexesByName(table.Indexes)[testCase.index]
			require.Truef(t, found, "index %s missing from %v", testCase.index, table.Indexes)
			assert.Equal(t, testCase.wantColumns, index.Columns)
			assert.Equal(t, testCase.wantUnique, index.IsUnique)
			assert.Equal(t, testCase.wantPrimary, index.IsPrimary)
		})
	}

	assert.NotContains(t, indexesByName(schema.Tables["users"].Indexes), "users_lower_email",
		"an index keyed only by expressions has no column to describe")
	assert.Empty(t, schema.Tables["sqliteapp"].Indexes)
}

func TestBuildCatalogueEmptyDatabase(t *testing.T) {
	schema := buildTestCatalogue(t, openSingleConnectionDatabase(t, ""))

	assert.Empty(t, schema.Tables)
	assert.Empty(t, schema.Views)
}

func TestBuildCatalogueReturnsErrors(t *testing.T) {
	testCases := []struct {
		name      string
		prepare   func(t *testing.T) (*sql.DB, context.Context)
		wantError string
	}{
		{
			name: "cancelled context",
			prepare: func(t *testing.T) (*sql.DB, context.Context) {
				t.Helper()
				ctx, cancel := context.WithCancelCause(t.Context())
				cancel(errors.New("caller gave up"))
				return openSingleConnectionDatabase(t, testSchema), ctx
			},
			wantError: "listing tables",
		},
		{
			name: "closed database",
			prepare: func(t *testing.T) (*sql.DB, context.Context) {
				t.Helper()
				database, err := sql.Open("sqlite", "file::memory:")
				require.NoError(t, err)
				require.NoError(t, database.Close())
				return database, t.Context()
			},
			wantError: "listing tables",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			database, ctx := testCase.prepare(t)
			provider := NewPragmaIntrospectionProvider(database, lowerCaseTypeNormaliser{})

			catalogue, sourceErrors, err := provider.BuildCatalogue(ctx)
			require.Error(t, err)
			assert.ErrorContains(t, err, testCase.wantError)
			assert.Nil(t, catalogue)
			assert.Nil(t, sourceErrors)
		})
	}
}

func TestIntrospectionOfMissingRelationIsEmpty(t *testing.T) {
	database := openSingleConnectionDatabase(t, "")
	provider := NewPragmaIntrospectionProvider(database, lowerCaseTypeNormaliser{})

	table, err := provider.introspectTable(t.Context(), "missing")
	require.NoError(t, err)
	assert.Empty(t, table.Columns)
	assert.Empty(t, table.Indexes)
	assert.Nil(t, table.PrimaryKey)
}

func TestIntrospectionFailsOnClosedDatabase(t *testing.T) {
	database, err := sql.Open("sqlite", "file::memory:")
	require.NoError(t, err)
	require.NoError(t, database.Close())
	provider := NewPragmaIntrospectionProvider(database, lowerCaseTypeNormaliser{})

	testCases := []struct {
		introspect func() error
		name       string
	}{
		{
			name: "table",
			introspect: func() error {
				_, introspectError := provider.introspectTable(t.Context(), "users")
				return introspectError
			},
		},
		{
			name: "view",
			introspect: func() error {
				_, introspectError := provider.introspectView(t.Context(), "user_emails")
				return introspectError
			},
		},
		{
			name: "indexes",
			introspect: func() error {
				_, introspectError := provider.introspectIndexes(t.Context(), "users")
				return introspectError
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Error(t, testCase.introspect())
		})
	}
}

func TestClassifyGeneratedColumn(t *testing.T) {
	tests := []struct {
		name              string
		hidden            int
		wantIsGenerated   bool
		wantGeneratedKind querier_dto.GeneratedKind
	}{
		{
			name:              "ordinary visible column",
			hidden:            0,
			wantIsGenerated:   false,
			wantGeneratedKind: querier_dto.GeneratedKindNone,
		},
		{
			name:              "hidden column that is not generated",
			hidden:            1,
			wantIsGenerated:   false,
			wantGeneratedKind: querier_dto.GeneratedKindNone,
		},
		{
			name:              "virtual generated column",
			hidden:            hiddenVirtualColumn,
			wantIsGenerated:   true,
			wantGeneratedKind: querier_dto.GeneratedKindVirtual,
		},
		{
			name:              "stored generated column",
			hidden:            hiddenStoredColumn,
			wantIsGenerated:   true,
			wantGeneratedKind: querier_dto.GeneratedKindStored,
		},
		{
			name:              "unknown flag treated as ordinary",
			hidden:            42,
			wantIsGenerated:   false,
			wantGeneratedKind: querier_dto.GeneratedKindNone,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			isGenerated, generatedKind := classifyGeneratedColumn(testCase.hidden)
			if isGenerated != testCase.wantIsGenerated {
				t.Fatalf("classifyGeneratedColumn(%d) isGenerated = %t, want %t",
					testCase.hidden, isGenerated, testCase.wantIsGenerated)
			}
			if generatedKind != testCase.wantGeneratedKind {
				t.Fatalf("classifyGeneratedColumn(%d) generatedKind = %d, want %d",
					testCase.hidden, generatedKind, testCase.wantGeneratedKind)
			}
		})
	}
}
