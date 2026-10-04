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
package db_catalogue_postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

var (
	columnQueryColumns = []string{
		"column_name", "udt_name", "is_nullable", "column_default", "character_maximum_length",
		"numeric_precision", "numeric_scale", "datetime_precision", "is_generated",
		"generation_expression", "is_identity", "identity_generation",
	}
	errFakePermissionDenied = errors.New("permission denied")
)

func columnRowValues(
	name string,
	udtName string,
	nullable string,
	columnDefault driver.Value,
	length driver.Value,
	precision driver.Value,
	scale driver.Value,
	datetimePrecision driver.Value,
	generated string,
	identity string,
) []driver.Value {
	return []driver.Value{
		name, udtName, nullable, columnDefault, length, precision, scale, datetimePrecision,
		generated, "", identity, "",
	}
}

func catalogueHandlers() []fakeQueryHandler {
	return []fakeQueryHandler{
		{
			marker:  "information_schema.schemata",
			respond: fixedResult([]string{"schema_name"}, []driver.Value{"app"}, []driver.Value{"public"}),
		},
		{
			marker: "information_schema.tables",
			respond: func(arguments []driver.NamedValue) fakeQueryResult {
				if argumentString(arguments, 0) == "app" {
					return fakeQueryResult{columns: []string{"table_name"}, rows: [][]driver.Value{{"users"}}}
				}
				return fakeQueryResult{columns: []string{"table_name"}}
			},
		},
		{
			marker: "information_schema.columns",
			respond: func(arguments []driver.NamedValue) fakeQueryResult {
				if argumentString(arguments, 1) == "user_emails" {
					return fakeQueryResult{columns: columnQueryColumns, rows: [][]driver.Value{
						columnRowValues("id", "int4", "YES", nil, nil, int64(32), int64(0), nil, "NEVER", "NO"),
						columnRowValues("email", "varchar", "YES", nil, int64(255), nil, nil, nil, "NEVER", "NO"),
					}}
				}
				return fakeQueryResult{columns: columnQueryColumns, rows: [][]driver.Value{
					columnRowValues("id", "int4", "NO", nil, nil, int64(32), int64(0), nil, "NEVER", "YES"),
					columnRowValues("email", "varchar", "NO", nil, int64(255), nil, nil, nil, "NEVER", "NO"),
					columnRowValues("price", "numeric", "YES", "0", nil, int64(10), int64(2), nil, "NEVER", "NO"),
					columnRowValues("seen", "timestamptz", "YES", nil, nil, nil, nil, int64(3), "NEVER", "NO"),
					columnRowValues("doubled", "int4", "YES", nil, nil, int64(32), int64(0), nil, "ALWAYS", "NO"),
				}}
			},
		},
		{
			marker: "FROM pg_constraint con",
			respond: fixedResult([]string{"conname", "contype", "attname", "ord"},
				[]driver.Value{"users_email_key", "u", "email", int64(1)},
				[]driver.Value{"users_pkey", "p", "id", int64(1)},
			),
		},
		{
			marker: "FROM pg_index ix",
			respond: fixedResult([]string{"index_name", "indisunique", "indisprimary", "attname", "column_position"},
				[]driver.Value{"users_email_key", true, false, "email", int64(1)},
				[]driver.Value{"users_pkey", true, true, "id", int64(1)},
				[]driver.Value{"users_seen_price", false, false, "seen", int64(1)},
				[]driver.Value{"users_seen_price", false, false, "price", int64(2)},
			),
		},
		{
			marker: "information_schema.views",
			respond: func(arguments []driver.NamedValue) fakeQueryResult {
				if argumentString(arguments, 0) != "app" {
					return fakeQueryResult{columns: []string{"table_name", "view_definition"}}
				}
				return fakeQueryResult{
					columns: []string{"table_name", "view_definition"},
					rows:    [][]driver.Value{{"user_emails", " SELECT id, email FROM app.users;"}},
				}
			},
		},
		{
			marker: "JOIN pg_enum e",
			respond: func(arguments []driver.NamedValue) fakeQueryResult {
				if argumentString(arguments, 0) != "app" {
					return fakeQueryResult{columns: []string{"typname", "enumlabel"}}
				}
				return fakeQueryResult{columns: []string{"typname", "enumlabel"}, rows: [][]driver.Value{
					{"mood", "sad"}, {"mood", "ok"}, {"mood", "happy"},
				}}
			},
		},
		{
			marker: "t.typtype = 'c'",
			respond: func(arguments []driver.NamedValue) fakeQueryResult {
				if argumentString(arguments, 0) != "app" {
					return fakeQueryResult{columns: []string{"typname", "attname", "type_name"}}
				}
				return fakeQueryResult{columns: []string{"typname", "attname", "type_name"}, rows: [][]driver.Value{
					{"address", "street", "text"},
					{"address", "postcode", "character varying(10)"},
				}}
			},
		},
		{
			marker: "FROM pg_proc p",
			respond: func(arguments []driver.NamedValue) fakeQueryResult {
				if argumentString(arguments, 0) != "app" {
					return fakeQueryResult{columns: functionParameterColumns}
				}
				return fakeQueryResult{columns: functionParameterColumns, rows: [][]driver.Value{
					{int64(10), "add", "f", true, false, false, int64(1), "integer", "i", "a", "integer"},
					{int64(10), "add", "f", true, false, false, int64(1), "integer", "i", "b", "integer"},
					{int64(11), "split_name", "f", false, false, false, int64(0), "record", "i", "full_name", "text"},
					{int64(11), "split_name", "f", false, false, false, int64(0), "record", "o", "first_name", "text"},
				}}
			},
		},
		{
			marker:  "FROM pg_extension",
			respond: fixedResult([]string{"extname"}, []driver.Value{"pgcrypto"}),
		},
	}
}

func buildFakeCatalogue(t *testing.T, handlers []fakeQueryHandler) (*querier_dto.Catalogue, error) {
	t.Helper()

	provider := NewPgIntrospectionProvider(openFakeDatabase(t, handlers...), modifierTypeNormaliser{})
	catalogue, sourceErrors, err := provider.BuildCatalogue(t.Context())
	assert.Nil(t, sourceErrors)
	return catalogue, err
}

func TestBuildCatalogueDescribesSchemas(t *testing.T) {
	catalogue, err := buildFakeCatalogue(t, catalogueHandlers())
	require.NoError(t, err)

	assert.Equal(t, "public", catalogue.DefaultSchema)
	assert.Contains(t, catalogue.Extensions, "pgcrypto")
	require.Contains(t, catalogue.Schemas, "public")
	assert.Empty(t, catalogue.Schemas["public"].Tables)

	app := catalogue.Schemas["app"]
	require.NotNil(t, app)

	users := app.Tables["users"]
	require.NotNil(t, users)
	assert.Equal(t, "app", users.Schema)
	assert.Equal(t, []string{"id"}, users.PrimaryKey)
	require.Len(t, users.Constraints, 1)
	assert.Equal(t, "users_email_key", users.Constraints[0].Name)
	assert.Equal(t, querier_dto.ConstraintUnique, users.Constraints[0].Kind)
	assert.Equal(t, []string{"email"}, users.Constraints[0].Columns)

	require.Len(t, users.Indexes, 3)
	assert.Equal(t, "users_pkey", users.Indexes[1].Name)
	assert.True(t, users.Indexes[1].IsPrimary)
	assert.Equal(t, []string{"seen", "price"}, users.Indexes[2].Columns)

	require.Contains(t, app.Views, "user_emails")
	assert.Equal(t, " SELECT id, email FROM app.users;", app.Views["user_emails"].Definition)
	assert.Len(t, app.Views["user_emails"].Columns, 2)

	require.Contains(t, app.Enums, "mood")
	assert.Equal(t, []string{"sad", "ok", "happy"}, app.Enums["mood"].Values)

	require.Contains(t, app.CompositeTypes, "address")
	fields := app.CompositeTypes["address"].Fields
	require.Len(t, fields, 2)
	assert.Equal(t, "postcode", fields[1].Name)
	assert.Equal(t, "character varying(10)", fields[1].SQLType.EngineName)

	require.Len(t, app.Functions["add"], 1)
	assert.Equal(t, 1, app.Functions["add"][0].MinArguments)
	require.Len(t, app.Functions["split_name"], 1)
	assert.Len(t, app.Functions["split_name"][0].Arguments, 1)
}

func TestBuildCatalogueDescribesColumns(t *testing.T) {
	catalogue, err := buildFakeCatalogue(t, catalogueHandlers())
	require.NoError(t, err)

	columns := catalogue.Schemas["app"].Tables["users"].Columns
	testCases := []struct {
		name              string
		wantType          string
		wantGeneratedKind querier_dto.GeneratedKind
		wantNullable      bool
		wantHasDefault    bool
		wantGenerated     bool
	}{
		{name: "id", wantType: "int4[32 0]", wantNullable: false, wantHasDefault: true},
		{name: "email", wantType: "varchar[255]", wantNullable: false},
		{name: "price", wantType: "numeric[10 2]", wantNullable: true, wantHasDefault: true},
		{name: "seen", wantType: "timestamptz[3]", wantNullable: true},
		{
			name:              "doubled",
			wantType:          "int4[32 0]",
			wantNullable:      true,
			wantGenerated:     true,
			wantGeneratedKind: querier_dto.GeneratedKindStored,
		},
	}

	require.Len(t, columns, len(testCases))
	for index, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			column := columns[index]
			assert.Equal(t, testCase.name, column.Name)
			assert.Equal(t, testCase.wantType, column.SQLType.EngineName)
			assert.Equal(t, testCase.wantNullable, column.Nullable)
			assert.Equal(t, testCase.wantHasDefault, column.HasDefault)
			assert.Equal(t, testCase.wantGenerated, column.IsGenerated)
			assert.Equal(t, testCase.wantGeneratedKind, column.GeneratedKind)
		})
	}
}

func TestBuildCatalogueReportsFailingQueries(t *testing.T) {
	testCases := []struct {
		name      string
		marker    string
		wantError string
	}{
		{name: "schemas", marker: "information_schema.schemata", wantError: "listing schemas"},
		{name: "tables", marker: "information_schema.tables", wantError: "introspecting tables in schema app"},
		{name: "columns", marker: "information_schema.columns", wantError: "introspecting table users"},
		{name: "constraints", marker: "FROM pg_constraint con", wantError: "introspecting table users"},
		{name: "indexes", marker: "FROM pg_index ix", wantError: "introspecting table users"},
		{name: "views", marker: "information_schema.views", wantError: "introspecting views in schema app"},
		{name: "enums", marker: "JOIN pg_enum e", wantError: "introspecting enums in schema app"},
		{name: "composite types", marker: "t.typtype = 'c'", wantError: "introspecting composite types in schema app"},
		{name: "functions", marker: "FROM pg_proc p", wantError: "introspecting functions in schema app"},
		{name: "extensions", marker: "FROM pg_extension", wantError: "introspecting extensions"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handlers := catalogueHandlers()
			for index := range handlers {
				if handlers[index].marker == testCase.marker {
					handlers[index].respond = func([]driver.NamedValue) fakeQueryResult {
						return fakeQueryResult{err: errFakePermissionDenied}
					}
				}
			}

			catalogue, err := buildFakeCatalogue(t, handlers)
			require.ErrorIs(t, err, errFakePermissionDenied)
			assert.ErrorContains(t, err, testCase.wantError)
			assert.Nil(t, catalogue)
		})
	}
}

func TestBuildCatalogueReportsRowFailures(t *testing.T) {
	closeFailure := errors.New("connection reset while closing")
	iterationFailure := errors.New("connection reset while reading")

	testCases := []struct {
		wantError error
		name      string
		marker    string
		columns   []string
		closeErr  bool
	}{
		{name: "schema iteration", marker: "information_schema.schemata", columns: []string{"schema_name"}, wantError: iterationFailure},
		{name: "column close", marker: "information_schema.columns", columns: columnQueryColumns, closeErr: true, wantError: closeFailure},
		{name: "constraint iteration", marker: "FROM pg_constraint con", columns: []string{"conname", "contype", "attname", "ord"}, wantError: iterationFailure},
		{name: "index close", marker: "FROM pg_index ix", columns: []string{"index_name", "indisunique", "indisprimary", "attname", "column_position"}, closeErr: true, wantError: closeFailure},
		{name: "view iteration", marker: "information_schema.views", columns: []string{"table_name", "view_definition"}, wantError: iterationFailure},
		{name: "enum close", marker: "JOIN pg_enum e", columns: []string{"typname", "enumlabel"}, closeErr: true, wantError: closeFailure},
		{name: "composite iteration", marker: "t.typtype = 'c'", columns: []string{"typname", "attname", "type_name"}, wantError: iterationFailure},
		{name: "extension close", marker: "FROM pg_extension", columns: []string{"extname"}, closeErr: true, wantError: closeFailure},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handlers := catalogueHandlers()
			for index := range handlers {
				if handlers[index].marker != testCase.marker {
					continue
				}
				result := fakeQueryResult{columns: testCase.columns, nextErr: iterationFailure}
				if testCase.closeErr {
					result = fakeQueryResult{columns: testCase.columns, closeErr: closeFailure}
				}
				handlers[index].respond = func([]driver.NamedValue) fakeQueryResult { return result }
			}

			catalogue, err := buildFakeCatalogue(t, handlers)
			require.ErrorIs(t, err, testCase.wantError)
			assert.Nil(t, catalogue)
		})
	}
}

func TestBuildCatalogueStopsWhenCancelled(t *testing.T) {
	provider := NewPgIntrospectionProvider(openFakeDatabase(t, catalogueHandlers()...), modifierTypeNormaliser{})
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(errors.New("caller gave up"))

	catalogue, _, err := provider.BuildCatalogue(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, catalogue)
}

func TestBuildCatalogueRejectsScanFailures(t *testing.T) {
	testCases := []struct {
		name    string
		marker  string
		columns []string
		row     []driver.Value
	}{
		{name: "schema", marker: "information_schema.schemata", columns: []string{"schema_name"}, row: []driver.Value{nil}},
		{name: "column", marker: "information_schema.columns", columns: columnQueryColumns, row: columnRowValues("id", "int4", "NO", nil, "wide", nil, nil, nil, "NEVER", "NO")},
		{name: "constraint", marker: "FROM pg_constraint con", columns: []string{"conname", "contype", "attname", "ord"}, row: []driver.Value{"c", "p", "id", "first"}},
		{name: "index", marker: "FROM pg_index ix", columns: []string{"index_name", "indisunique", "indisprimary", "attname", "column_position"}, row: []driver.Value{"i", "maybe", false, "id", int64(1)}},
		{name: "view", marker: "information_schema.views", columns: []string{"table_name", "view_definition"}, row: []driver.Value{nil, nil}},
		{name: "enum", marker: "JOIN pg_enum e", columns: []string{"typname", "enumlabel"}, row: []driver.Value{nil, "ok"}},
		{name: "composite", marker: "t.typtype = 'c'", columns: []string{"typname", "attname", "type_name"}, row: []driver.Value{nil, "a", "text"}},
		{name: "extension", marker: "FROM pg_extension", columns: []string{"extname"}, row: []driver.Value{nil}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handlers := catalogueHandlers()
			for index := range handlers {
				if handlers[index].marker == testCase.marker {
					handlers[index].respond = fixedResult(testCase.columns, testCase.row)
				}
			}

			catalogue, err := buildFakeCatalogue(t, handlers)
			require.Error(t, err)
			assert.ErrorContains(t, err, "Scan error")
			assert.Nil(t, catalogue)
		})
	}
}

func TestBuildTypeModifiers(t *testing.T) {
	testCases := []struct {
		name string
		want []int
		row  columnRow
	}{
		{name: "temporal precision", row: columnRow{datetimePrecision: sql.NullInt64{Int64: 3, Valid: true}}, want: []int{3}},
		{name: "text length", row: columnRow{characterMaximumLength: sql.NullInt64{Int64: 255, Valid: true}}, want: []int{255}},
		{
			name: "numeric precision and scale",
			row: columnRow{
				numericPrecision: sql.NullInt64{Int64: 10, Valid: true},
				numericScale:     sql.NullInt64{Int64: 2, Valid: true},
			},
			want: []int{10, 2},
		},
		{name: "no modifiers", row: columnRow{}, want: nil},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, buildTypeModifiers(testCase.row))
		})
	}
}

func TestBuildCatalogueNotConfigured(t *testing.T) {
	provider := NewPgIntrospectionProvider(nil, nil)

	catalogue, sourceErrors, buildError := provider.BuildCatalogue(context.Background())

	assert.Nil(t, catalogue)
	assert.Nil(t, sourceErrors)
	assert.ErrorIs(t, buildError, ErrProviderNotConfigured)
}

func TestAssembleIndexResultsSkipsEmptyColumns(t *testing.T) {
	indexMap := map[string]*indexEntry{
		"empty_expression_index": {},
		"name_index":             {columns: []string{"name"}},
	}
	indexOrder := []string{"empty_expression_index", "name_index"}

	indexes := assembleIndexResults(indexMap, indexOrder)

	require.Len(t, indexes, 1)
	assert.Equal(t, "name_index", indexes[0].Name)
}
