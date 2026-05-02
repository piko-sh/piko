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

	"piko.sh/piko/wdk/db/db_engine_timescaledb"
)

func TestCompression_AlterTableEnable(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("ALTER TABLE readings SET (timescaledb.compress = true)")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "readings", mutation.TableName)
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_COMPRESSION_ENABLED"])
}

func TestCompression_AlterTableSegmentByAndOrderBy(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	sql := `ALTER TABLE readings SET (
		timescaledb.compress = true,
		timescaledb.compress_segmentby = 'device_id',
		timescaledb.compress_orderby = 'ts DESC'
	)`
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_COMPRESSION_ENABLED"])
	assert.Equal(t, "device_id", mutation.EngineSpecific["TIMESCALE_COMPRESSION_SEGMENTBY"])
	assert.Equal(t, "ts DESC", mutation.EngineSpecific["TIMESCALE_COMPRESSION_ORDERBY"])
}

func TestCompression_AlterMaterializedView(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	sql := `ALTER MATERIALIZED VIEW hourly_temps SET (timescaledb.compress = true, timescaledb.compress_segmentby = 'bucket')`
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "hourly_temps", mutation.TableName)
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_ALTER_MATERIALIZED"])
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_COMPRESSION_ENABLED"])
	assert.Equal(t, "bucket", mutation.EngineSpecific["TIMESCALE_COMPRESSION_SEGMENTBY"])
}

func TestCompression_AlterTableSetSchemaIsNotCompression(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("ALTER TABLE readings SET SCHEMA analytics")
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Empty(t, mutation.EngineSpecific["TIMESCALE_COMPRESSION_ENABLED"])
}
