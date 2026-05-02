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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_timescaledb"
)

func applyTimescaleDDL(t *testing.T, sql string) (*querier_dto.CatalogueMutation, error) {
	t.Helper()
	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, statements, 1)
	return engine.ApplyDDL(context.Background(), statements[0])
}

func TestCreateHypertable_RegclassCast(t *testing.T) {
	t.Parallel()

	mutation, err := applyTimescaleDDL(t, `SELECT create_hypertable('readings'::regclass, 'ts'::name)`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "readings", mutation.TableName)
	assert.Equal(t, "ts", mutation.EngineSpecific["TIMESCALE_TIME_COLUMN"])
}

func TestRetentionPolicy_RegclassCast(t *testing.T) {
	t.Parallel()

	mutation, err := applyTimescaleDDL(t, `SELECT add_retention_policy('conditions'::regclass, drop_after => INTERVAL '24 hours')`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
}

func TestCompressionPolicy_RegclassCast(t *testing.T) {
	t.Parallel()

	mutation, err := applyTimescaleDDL(t, `SELECT add_compression_policy('conditions'::regclass, INTERVAL '7 days')`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
}

func TestCreateHypertable_InlineNamedConstraint(t *testing.T) {
	t.Parallel()

	mutation, err := applyTimescaleDDL(t, `CREATE HYPERTABLE t (val INTEGER CONSTRAINT pos CHECK (val > 0), ts TIMESTAMPTZ NOT NULL)`)
	require.NoError(t, err)
	require.NotNil(t, mutation)
	require.GreaterOrEqual(t, len(mutation.Columns), 2)
	assert.Equal(t, "val", mutation.Columns[0].Name)

	assert.Equal(t, "int4", mutation.Columns[0].SQLType.EngineName)
	assert.Equal(t, querier_dto.TypeCategoryInteger, mutation.Columns[0].SQLType.Category)
	assert.NotContains(t, mutation.Columns[0].SQLType.EngineName, "CONSTRAINT")

	assert.Equal(t, "ts", mutation.Columns[1].Name)
	assert.Equal(t, "timestamptz", mutation.Columns[1].SQLType.EngineName)
	assert.Equal(t, querier_dto.TypeCategoryTemporal, mutation.Columns[1].SQLType.Category)
}
