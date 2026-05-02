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

func TestPolicy_AddCompressionPolicy(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT add_compression_policy('readings', INTERVAL '7 days')`)
	require.NoError(t, err)
	require.Len(t, statements, 1)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableAlterColumn, mutation.Kind)
	assert.Equal(t, "readings", mutation.TableName)
	assert.Equal(t, "add_compression_policy", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "readings", mutation.EngineSpecific["TIMESCALE_TABLE"])
	assert.Contains(t, mutation.EngineSpecific["TIMESCALE_POLICY_ARGS"], "INTERVAL")
}

func TestPolicy_AddColumnstorePolicy(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT add_columnstore_policy('readings', after => INTERVAL '7 days')`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "add_columnstore_policy", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "readings", mutation.TableName)
}

func TestPolicy_RemoveCompressionPolicy(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT remove_compression_policy('readings')`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "remove_compression_policy", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "readings", mutation.TableName)
}

func TestPolicy_AddRetentionPolicy(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT add_retention_policy('readings', INTERVAL '30 days')`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "add_retention_policy", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "readings", mutation.TableName)
}

func TestPolicy_RemoveRetentionPolicy(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT remove_retention_policy('readings')`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "remove_retention_policy", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
}

func TestPolicy_AddContinuousAggregatePolicy(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	sql := `SELECT add_continuous_aggregate_policy('hourly_temps', start_offset => INTERVAL '1 month', end_offset => INTERVAL '1 hour', schedule_interval => INTERVAL '1 hour')`
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "add_continuous_aggregate_policy", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "hourly_temps", mutation.TableName)
	assert.Contains(t, mutation.EngineSpecific["TIMESCALE_POLICY_ARGS"], "start_offset")
}

func TestPolicy_RemoveContinuousAggregatePolicy(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT remove_continuous_aggregate_policy('hourly_temps')`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "remove_continuous_aggregate_policy", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
}

func TestPolicy_AddReorderPolicy(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT add_reorder_policy('readings', 'readings_ts_device_idx')`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "add_reorder_policy", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "readings", mutation.TableName)
}

func TestPolicy_AddJob(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT add_job('my_proc', '1h', config => '{"a":1}'::jsonb)`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "add_job", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "my_proc", mutation.EngineSpecific["TIMESCALE_JOB_PROC"])
}

func TestPolicy_AlterJob(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT alter_job(1000, scheduled => false)`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "alter_job", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "1000", mutation.EngineSpecific["TIMESCALE_JOB_ID"])
}

func TestPolicy_DeleteJob(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT delete_job(1000)`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "delete_job", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "1000", mutation.EngineSpecific["TIMESCALE_JOB_ID"])
}

func TestPolicy_RunJob(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT run_job(1000)`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "run_job", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "1000", mutation.EngineSpecific["TIMESCALE_JOB_ID"])
}

func TestPolicy_AddDimensionLegacyColumn(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT add_dimension('readings', 'device_id', number_partitions => 4)`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "add_dimension", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "readings", mutation.TableName)
	assert.Equal(t, "device_id", mutation.EngineSpecific["TIMESCALE_TIME_COLUMN"])
	assert.Empty(t, mutation.EngineSpecific["TIMESCALE_DIMENSION_BUILDER"])
}

func TestPolicy_AddDimensionChunkTimeInterval(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT add_dimension('readings', 'ts', chunk_time_interval => INTERVAL '1 day')`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "add_dimension", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "readings", mutation.TableName)
	assert.Equal(t, "ts", mutation.EngineSpecific["TIMESCALE_TIME_COLUMN"])
	assert.Contains(t, mutation.EngineSpecific["TIMESCALE_POLICY_ARGS"], "chunk_time_interval")
}

func TestPolicy_AddDimensionByRange(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT add_dimension('readings', by_range('device_id', INTERVAL '1 day'))`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "by_range", mutation.EngineSpecific["TIMESCALE_DIMENSION_BUILDER"])
	assert.Equal(t, "device_id", mutation.EngineSpecific["TIMESCALE_TIME_COLUMN"])
}

func TestPolicy_AddDimensionByHash(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT add_dimension('readings', by_hash('device_id', 4))`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "by_hash", mutation.EngineSpecific["TIMESCALE_DIMENSION_BUILDER"])
	assert.Equal(t, "device_id", mutation.EngineSpecific["TIMESCALE_TIME_COLUMN"])
}

func TestPolicy_SetChunkTimeInterval(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT set_chunk_time_interval('readings', INTERVAL '12 hours')`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "set_chunk_time_interval", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "readings", mutation.TableName)
}

func TestPolicy_SetIntegerNowFunc(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT set_integer_now_func('readings', 'now_epoch')`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "set_integer_now_func", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
	assert.Equal(t, "readings", mutation.TableName)
}

func TestPolicy_EnableChunkSkipping(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT enable_chunk_skipping('readings', 'device_id')`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "enable_chunk_skipping", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
}

func TestPolicy_DisableChunkSkipping(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT disable_chunk_skipping('readings', 'device_id')`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "disable_chunk_skipping", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
}

func TestRefreshContinuousAggregate_Call(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`CALL refresh_continuous_aggregate('hourly_temps', '2020-01-01', '2021-01-01')`)
	require.NoError(t, err)
	require.Len(t, statements, 1)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationAlterTableAlterColumn, mutation.Kind)
	assert.Equal(t, "hourly_temps", mutation.TableName)
	assert.Equal(t, "hourly_temps", mutation.EngineSpecific["TIMESCALE_REFRESH_CONTINUOUS_AGGREGATE_TARGET"])
	assert.Equal(t, "'2020-01-01'", mutation.EngineSpecific["TIMESCALE_REFRESH_WINDOW_START"])
	assert.Equal(t, "'2021-01-01'", mutation.EngineSpecific["TIMESCALE_REFRESH_WINDOW_END"])
	assert.Equal(t, "refresh_continuous_aggregate", mutation.EngineSpecific["TIMESCALE_POLICY_OP"])
}

func TestRefreshContinuousAggregate_NullWindow(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`CALL refresh_continuous_aggregate('hourly_temps', NULL, NULL)`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "NULL", mutation.EngineSpecific["TIMESCALE_REFRESH_WINDOW_START"])
	assert.Equal(t, "NULL", mutation.EngineSpecific["TIMESCALE_REFRESH_WINDOW_END"])
}

func TestRefreshContinuousAggregate_DoesNotClaimBareCall(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`CALL some_other_proc('x')`)
	require.NoError(t, err)
	require.NotEmpty(t, statements)
}

func TestCreateHypertable_ByRangeBuilder(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT create_hypertable('readings', by_range('time', INTERVAL '1 day'))`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "by_range", mutation.EngineSpecific["TIMESCALE_DIMENSION_BUILDER"])
	assert.Equal(t, "time", mutation.EngineSpecific["TIMESCALE_TIME_COLUMN"])
	assert.Equal(t, "readings", mutation.TableName)
}

func TestCreateHypertable_ByHashBuilder(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT create_hypertable('readings', by_hash('device_id', 4))`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "by_hash", mutation.EngineSpecific["TIMESCALE_DIMENSION_BUILDER"])
	assert.Equal(t, "device_id", mutation.EngineSpecific["TIMESCALE_TIME_COLUMN"])
}

func TestCreateTable_PlainNoLift(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`CREATE TABLE widgets (id BIGINT, name TEXT)`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
	assert.Empty(t, mutation.EngineSpecific["TIMESCALE_HYPERTABLE"])
}

func TestCreateTable_WithTsdbHypertableBareActivator(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	sql := `CREATE TABLE readings (ts TIMESTAMPTZ NOT NULL, value DOUBLE PRECISION) WITH (tsdb.hypertable)`
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
	assert.Equal(t, "readings", mutation.TableName)
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_HYPERTABLE"])
}

func TestCreateTable_WithTsdbHypertableFullForm(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	sql := `CREATE TABLE readings (ts TIMESTAMPTZ NOT NULL, value DOUBLE PRECISION)
		WITH (tsdb.hypertable, tsdb.partition_column = 'ts', tsdb.chunk_interval = '1 day')`
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_HYPERTABLE"])

	assert.Equal(t, "ts", mutation.EngineSpecific["TIMESCALE_TIME_COLUMN"])
	assert.Equal(t, "1 day", mutation.EngineSpecific["TIMESCALE_CHUNK_TIME_INTERVAL"])
}

func TestCreateTable_WithTimescaledbHypertableForm(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	sql := `CREATE TABLE readings (ts TIMESTAMPTZ NOT NULL, value DOUBLE PRECISION)
		WITH (timescaledb.hypertable, timescaledb.partition_column = 'ts', timescaledb.segmentby = 'device_id', timescaledb.enable_columnstore = true)`
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_HYPERTABLE"])
	assert.Equal(t, "ts", mutation.EngineSpecific["TIMESCALE_TIME_COLUMN"])
	assert.Equal(t, "device_id", mutation.EngineSpecific["TIMESCALE_SEGMENTBY"])
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_COLUMNSTORE_ENABLED"])
}

func TestCreateTable_WithUnrelatedRelopt(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	sql := `CREATE TABLE widgets (id BIGINT, name TEXT) WITH (fillfactor = 80)`
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
	assert.Empty(t, mutation.EngineSpecific["TIMESCALE_HYPERTABLE"])
}

func TestCompression_ModernColumnstoreReloptions(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	sql := `ALTER TABLE readings SET (
		timescaledb.enable_columnstore = true,
		timescaledb.segmentby = 'device_id',
		timescaledb.orderby = 'ts DESC'
	)`
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_COLUMNSTORE_ENABLED"])
	assert.Equal(t, "device_id", mutation.EngineSpecific["TIMESCALE_SEGMENTBY"])
	assert.Equal(t, "ts DESC", mutation.EngineSpecific["TIMESCALE_ORDERBY"])
}

func TestContinuousAggregate_ModernColumnstoreReloptions(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	sql := `CREATE MATERIALIZED VIEW hourly_temps
		WITH (timescaledb.continuous = true, timescaledb.enable_columnstore = true, timescaledb.segmentby = 'device_id', timescaledb.orderby = 'ts DESC')
		AS SELECT time_bucket('1 hour', ts) AS bucket, avg(temperature) FROM readings GROUP BY bucket`
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_COLUMNSTORE_ENABLED"])
	assert.Equal(t, "device_id", mutation.EngineSpecific["TIMESCALE_SEGMENTBY"])
	assert.Equal(t, "ts DESC", mutation.EngineSpecific["TIMESCALE_ORDERBY"])
}
