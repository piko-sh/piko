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
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/sqldb/sqldb_dto"
	"piko.sh/piko/wdk/db"
)

func TestPostgresFamilyMigrationsRunOnTheTimescaleDBBackendOnlyForTheTimescaleDBDialect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		builtin  string
		dialect  sqldb_dto.DialectConfig
		expected postgresFamilyBackend
	}{
		{name: "postgres", builtin: "postgres", dialect: db.PostgresDialect(), expected: backendPostgres},
		{name: "postgres pgbouncer", builtin: "postgres", dialect: db.PostgresPgBouncerDialect(), expected: backendPostgresPgBouncer},
		{name: "custom dialect", builtin: "", dialect: db.PostgresDialect(), expected: backendPostgres},
		{name: "timescaledb", builtin: "timescaledb", dialect: db.PostgresDialect(), expected: backendTimescaleDB},
		{name: "timescaledb pgbouncer", builtin: "timescaledb", dialect: db.PostgresPgBouncerDialect(), expected: backendTimescaleDB},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testCase.expected, postgresBackendFor(testCase.dialect, testCase.builtin))
		})
	}
}

func TestNewBuiltinEngineSelectsTheDialectsOwnMigrationBackend(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		dialect string
		name    string
	}{
		{dialect: "timescaledb", name: ""},
		{dialect: "timescaledb", name: "metrics"},
		{dialect: "postgres", name: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.dialect+"/"+testCase.name, func(t *testing.T) {
			t.Parallel()

			engine := NewBuiltinEngine(DialectOptions{Dialect: testCase.dialect, Name: testCase.name})
			toolchain, ok := engine.Toolchain().(*Toolchain)
			require.True(t, ok)
			require.Equal(t, testCase.dialect, toolchain.builtin)
		})
	}
}
