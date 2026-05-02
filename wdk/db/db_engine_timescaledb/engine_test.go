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

package db_engine_timescaledb_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/wdk/db/db_engine_timescaledb"
)

func TestTimescaleDBEngine_SatisfiesEnginePort(t *testing.T) {
	t.Parallel()

	var _ querier_domain.EnginePort = db_engine_timescaledb.NewTimescaleDBEngine()
}

func TestTimescaleDBEngine_DialectName(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	assert.Equal(t, "timescaledb", engine.Dialect())
}

func TestTimescaleDBEngine_PostgresQueriesPassThrough(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("SELECT id FROM users")
	require.NoError(t, err)
	require.Len(t, statements, 1)
}

func TestTimescaleDB_FactoryReturnsConfig(t *testing.T) {
	t.Parallel()

	config := db_engine_timescaledb.TimescaleDB()
	assert.Equal(t, "postgres", config.DriverName)
	assert.NotNil(t, config.Engine)
	assert.NotNil(t, config.MigrationDialect)
}
