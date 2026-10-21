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
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/sqldb/sqldb_dto"

	"seagull.sh/seagull/sdk/dialect/dialect_engine_sqlite"
)

func TestCustomFunctionDataAccessRoutesTheQuery(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"migrations/001_counters.up.sql": {Data: []byte("CREATE TABLE counters (id INTEGER PRIMARY KEY, hits INTEGER NOT NULL);\n")},
		"queries/counters.sql":           {Data: []byte("-- piko.query(BumpCounter, one)\nSELECT bump(id) AS hits FROM counters WHERE id = ?;\n")},
	}

	testCases := []struct {
		name       string
		dataAccess string
		expected   string
	}{
		{name: "default", dataAccess: "", expected: "queries.reader.QueryRowContext"},
		{name: "read only", dataAccess: "read_only", expected: "queries.reader.QueryRowContext"},
		{name: "modifies data", dataAccess: "modifies_data", expected: "queries.writer.QueryRowContext"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := NewToolchain(dialect_engine_sqlite.New()).Generate(context.Background(), sqldb_dto.GenerateRequest{
				Files:              files,
				MigrationDirectory: "migrations",
				QueryDirectory:     "queries",
				Driver:             sqldb_dto.DriverDatabaseSQL,
				CustomFunctions: []sqldb_dto.CustomFunction{{
					Name:       "bump",
					ReturnType: "integer",
					Nullable:   "never_null",
					DataAccess: testCase.dataAccess,
					Arguments:  []string{"integer"},
				}},
			})
			require.NoError(t, err)
			require.Empty(t, result.Diagnostics)

			require.Contains(t, generatedFile(t, result, "counters.sql.go"), testCase.expected)
		})
	}
}
