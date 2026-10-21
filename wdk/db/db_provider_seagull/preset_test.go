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

package db_provider_seagull

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"piko.sh/piko/internal/sqldb/sqldb_dto"
	"piko.sh/piko/wdk/db/dbjson"

	"seagull.sh/seagull"
	"seagull.sh/seagull/sdk/dialect/dialect_registry"
)

func TestPikoPresetKeepsPikoConstantNamesAndStringEnums(t *testing.T) {
	t.Parallel()

	preset := pikoPreset()

	require.Empty(t, preset.SQLConstantSuffix)
	require.True(t, preset.SQLConstantLowerCase)
	require.True(t, preset.NonNullElements)
	require.True(t, preset.KeepStars)
	require.Equal(t, "string", preset.Types.Enum.NotNull.Name)
	require.Equal(t, "*string", preset.Types.Enum.Nullable.Name)
	require.Empty(t, preset.Types.Enum.NotNull.Package)
}

func TestPresetTypeOverridesKeepSQLiteIntegersAsInt32(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		dialect  string
		expected []seagull.TypeOverride
	}{
		{dialect: "sqlite", expected: []seagull.TypeOverride{{SQLTypeName: "integer", GoName: "int32"}}},
		{dialect: "postgres", expected: nil},
		{dialect: "mysql", expected: nil},
		{dialect: "duckdb", expected: nil},
		{dialect: "clickhouse", expected: nil},
	}

	for _, testCase := range testCases {
		t.Run(testCase.dialect, func(t *testing.T) {
			t.Parallel()

			chosen, err := dialect_registry.New(testCase.dialect, dialect_registry.Options{})
			require.NoError(t, err)

			facts := chosen.Info()
			require.Equal(t, testCase.expected, presetTypeOverrides(&facts))
		})
	}
}

func TestMergeTypeOverridesLetsTheRequestWin(t *testing.T) {
	t.Parallel()

	presetInteger := seagull.TypeOverride{SQLTypeName: "integer", GoName: "int32"}
	requestedInteger := seagull.TypeOverride{SQLTypeName: "INTEGER", GoName: "int64"}
	requestedUUID := seagull.TypeOverride{SQLTypeName: "uuid", GoPackage: "example.com/ids", GoName: "ID"}
	requestedColumn := seagull.TypeOverride{Column: "users.id", GoName: "int64"}

	testCases := []struct {
		name      string
		preset    []seagull.TypeOverride
		requested []seagull.TypeOverride
		expected  []seagull.TypeOverride
	}{
		{name: "no overrides", preset: nil, requested: nil, expected: []seagull.TypeOverride{}},
		{name: "preset alone", preset: []seagull.TypeOverride{presetInteger}, requested: nil, expected: []seagull.TypeOverride{presetInteger}},
		{name: "request alone", preset: nil, requested: []seagull.TypeOverride{requestedUUID}, expected: []seagull.TypeOverride{requestedUUID}},
		{name: "other types keep the preset first", preset: []seagull.TypeOverride{presetInteger}, requested: []seagull.TypeOverride{requestedUUID}, expected: []seagull.TypeOverride{presetInteger, requestedUUID}},
		{name: "the same type in any case replaces the preset", preset: []seagull.TypeOverride{presetInteger}, requested: []seagull.TypeOverride{requestedInteger}, expected: []seagull.TypeOverride{requestedInteger}},
		{name: "a column override keeps the preset", preset: []seagull.TypeOverride{presetInteger}, requested: []seagull.TypeOverride{requestedColumn}, expected: []seagull.TypeOverride{presetInteger, requestedColumn}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testCase.expected, mergeTypeOverrides(testCase.preset, testCase.requested))
		})
	}
}

func TestGenerateReadsSQLiteIntegersAsInt32UnlessTheRequestOverrides(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"migrations/001_counters.up.sql": {Data: []byte("CREATE TABLE counters (id INTEGER PRIMARY KEY, hits INTEGER NOT NULL);\n")},
		"queries/counters.sql":           {Data: []byte("-- piko.query(GetCounter, one)\nSELECT id, hits FROM counters WHERE id = ?;\n")},
	}

	testCases := []struct {
		name      string
		overrides []sqldb_dto.TypeOverride
		expected  string
	}{
		{name: "preset", overrides: nil, expected: "Hits int32"},
		{name: "request override", overrides: []sqldb_dto.TypeOverride{{SQLTypeName: "integer", GoName: "int64"}}, expected: "Hits int64"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			chosen, err := dialect_registry.New("sqlite", dialect_registry.Options{})
			require.NoError(t, err)

			result, err := NewToolchain(chosen).Generate(context.Background(), sqldb_dto.GenerateRequest{
				Files:              files,
				MigrationDirectory: "migrations",
				QueryDirectory:     "queries",
				Driver:             sqldb_dto.DriverDatabaseSQL,
				TypeOverrides:      testCase.overrides,
			})
			require.NoError(t, err)

			queries := generatedFile(t, result, "counters.sql.go")
			require.Contains(t, strings.Join(strings.Fields(queries), " "), testCase.expected)
			require.Contains(t, queries, "const getcounter = ")
		})
	}
}

func TestPikoRuntimeTablesAreThePersistedTables(t *testing.T) {
	t.Parallel()

	require.Equal(t, []string{"piko_migrations", "piko_seeds", "piko_migration_lock", "piko_seed_lock"}, pikoRuntimeTables())
}

func TestPresetJSONTypeIsStoredAsTextInSQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	database, err := sql.Open("sqlite", "file:"+t.TempDir()+"/json.db")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	_, err = database.ExecContext(ctx, "CREATE TABLE events (id INTEGER PRIMARY KEY, payload JSON)")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "INSERT INTO events (id, payload) VALUES (?, ?), (?, ?)", 1, dbjson.JSON(`{"kind":"signup"}`), 2, dbjson.JSON(nil))
	require.NoError(t, err)

	var storage string
	require.NoError(t, database.QueryRowContext(ctx, "SELECT typeof(payload) FROM events WHERE id = 1").Scan(&storage))
	require.Equal(t, "text", storage)

	var found int
	require.NoError(t, database.QueryRowContext(ctx, `SELECT id FROM events WHERE payload = '{"kind":"signup"}'`).Scan(&found))
	require.Equal(t, 1, found)
	require.NoError(t, database.QueryRowContext(ctx, "SELECT id FROM events WHERE json_extract(payload, '$.kind') = ?", "signup").Scan(&found))
	require.Equal(t, 1, found)

	var missing dbjson.JSON
	require.NoError(t, database.QueryRowContext(ctx, "SELECT payload FROM events WHERE id = 2").Scan(&missing))
	require.Nil(t, missing)
	require.Equal(t, jsonPackage, pikoPreset().Types.JSON.NotNull.Package)
}

func generatedFile(t *testing.T, result *sqldb_dto.GenerationResult, name string) string {
	t.Helper()

	for _, file := range result.Files {
		if file.Name == name {
			return string(file.Content)
		}
	}
	require.Failf(t, "missing generated file", "no %s among the generated files", name)
	return ""
}
