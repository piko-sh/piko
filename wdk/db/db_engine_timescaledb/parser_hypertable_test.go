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

func TestParseCreateHypertable_Keyword_Basic(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("CREATE HYPERTABLE readings (ts TIMESTAMPTZ NOT NULL, device_id BIGINT, temperature DOUBLE PRECISION)")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
	assert.Equal(t, "readings", mutation.TableName)
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_HYPERTABLE"])

	require.Len(t, mutation.Columns, 3)
	assert.Equal(t, "ts", mutation.Columns[0].Name)
	assert.False(t, mutation.Columns[0].Nullable, "ts is declared NOT NULL")
	assert.Equal(t, "device_id", mutation.Columns[1].Name)
	assert.Equal(t, "temperature", mutation.Columns[2].Name)
}

func TestParseCreateHypertable_Keyword_IfNotExists(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("CREATE HYPERTABLE IF NOT EXISTS metrics (ts TIMESTAMPTZ, value DOUBLE PRECISION)")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "metrics", mutation.TableName)
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_IF_NOT_EXISTS"])
}

func TestParseCreateHypertable_Keyword_SchemaQualified(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("CREATE HYPERTABLE analytics.events (ts TIMESTAMPTZ, kind TEXT)")
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "analytics", mutation.SchemaName)
	assert.Equal(t, "events", mutation.TableName)
}

func TestParseCreateHypertable_FunctionCallForm(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("SELECT create_hypertable('readings', 'ts')")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)

	assert.Equal(t, querier_dto.MutationAlterTableAlterColumn, mutation.Kind)
	assert.Equal(t, "readings", mutation.TableName)
	assert.Equal(t, "ts", mutation.EngineSpecific["TIMESCALE_TIME_COLUMN"])
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_ANNOTATE_ONLY"])
}

func TestParseCreateHypertable_FunctionCallForm_WithExtras(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(`SELECT create_hypertable('readings', 'ts', chunk_time_interval => INTERVAL '1 day')`)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "readings", mutation.TableName)
	assert.NotEmpty(t, mutation.EngineSpecific["TIMESCALE_CALL_EXTRAS"], "extras should capture chunk_time_interval clause")
}

func TestParseCreateHypertable_CallExtrasRoundTripsNonPlainLiterals(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		extras      string
		mustContain string
	}{
		{
			description: "bit-string literal keeps its B prefix and quotes",
			extras:      `'readings', 'ts', flags => B'1010'`,
			mustContain: "B'1010'",
		},
		{
			description: "dollar-quoted literal keeps its delimiters",
			extras:      `'readings', 'ts', note => $tag$body$tag$`,
			mustContain: "$$body$$",
		},
		{
			description: "quoted identifier keeps its double quotes",
			extras:      `'readings', 'ts', "Weird Param" => 1`,
			mustContain: `"Weird Param"`,
		},
	}

	engine := db_engine_timescaledb.NewTimescaleDBEngine()

	for _, testCase := range cases {
		t.Run(testCase.description, func(t *testing.T) {
			t.Parallel()

			statements, err := engine.ParseStatements("SELECT create_hypertable(" + testCase.extras + ")")
			require.NoError(t, err)
			require.Len(t, statements, 1)

			mutation, err := engine.ApplyDDL(context.Background(), statements[0])
			require.NoError(t, err)
			require.NotNil(t, mutation)

			extras := mutation.EngineSpecific["TIMESCALE_CALL_EXTRAS"]
			assert.Contains(t, extras, testCase.mustContain,
				"captured call extras must re-wrap non-plain literals with their delimiters, got %q", extras)
		})
	}
}

func TestParseCreateHypertable_InlinePrimaryKey(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("CREATE HYPERTABLE readings (id BIGINT PRIMARY KEY, ts TIMESTAMPTZ NOT NULL)")
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, []string{"id"}, mutation.PrimaryKey)
	require.Len(t, mutation.Columns, 2)
	assert.False(t, mutation.Columns[0].Nullable, "PRIMARY KEY implies NOT NULL")
}

func TestParseCreateHypertable_InlineConstraintsAfterDefaultAndReferences(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()

	statements, err := engine.ParseStatements(
		"CREATE HYPERTABLE readings (" +
			"id BIGINT DEFAULT 1 PRIMARY KEY, " +
			"ts TIMESTAMPTZ DEFAULT now() NOT NULL, " +
			"device_id BIGINT REFERENCES devices(id) ON DELETE SET NULL NOT NULL, " +
			"score INT CHECK (score >= 0) NOT NULL)",
	)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	require.Len(t, mutation.Columns, 4)

	assert.Equal(t, "id", mutation.Columns[0].Name)
	assert.Equal(t, []string{"id"}, mutation.PrimaryKey, "DEFAULT 1 PRIMARY KEY must keep the PK")
	assert.False(t, mutation.Columns[0].Nullable, "PRIMARY KEY implies NOT NULL")

	assert.Equal(t, "ts", mutation.Columns[1].Name)
	assert.True(t, mutation.Columns[1].HasDefault)
	assert.False(t, mutation.Columns[1].Nullable, "DEFAULT now() NOT NULL must record NOT NULL")

	assert.Equal(t, "device_id", mutation.Columns[2].Name)
	assert.False(t, mutation.Columns[2].Nullable, "trailing NOT NULL after ON DELETE SET NULL must survive")

	assert.Equal(t, "score", mutation.Columns[3].Name)
	assert.False(t, mutation.Columns[3].Nullable, "trailing NOT NULL after CHECK(...) must survive")
}

func TestParseCreateHypertable_TablePrimaryKeyMultiColumn(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("CREATE HYPERTABLE readings (ts TIMESTAMPTZ, device_id BIGINT, PRIMARY KEY (ts, device_id))")
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, []string{"ts", "device_id"}, mutation.PrimaryKey)
}

func TestParseCreateHypertable_NamedPrimaryKeyConstraint(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("CREATE HYPERTABLE readings (ts TIMESTAMPTZ, device_id BIGINT, CONSTRAINT readings_pk PRIMARY KEY (ts, device_id))")
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, []string{"ts", "device_id"}, mutation.PrimaryKey)
}

func TestParseCreateHypertable_TablePKOverridesInlinePK(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("CREATE HYPERTABLE readings (id BIGINT PRIMARY KEY, ts TIMESTAMPTZ, PRIMARY KEY (ts))")
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, []string{"ts"}, mutation.PrimaryKey)
}

func TestParseCreateHypertable_TrailingPartitionBy(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("CREATE HYPERTABLE readings (ts TIMESTAMPTZ, value DOUBLE PRECISION) PARTITION BY RANGE (ts)")
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Contains(t, mutation.EngineSpecific["TIMESCALE_TRAILING"], "PARTITION")
}

func TestParseCreateHypertable_FunctionCallForm_SchemaQualified(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("SELECT create_hypertable('analytics.readings', 'ts')")
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "analytics", mutation.SchemaName)
	assert.Equal(t, "readings", mutation.TableName)
}

func TestParseCreateHypertable_FunctionCallForm_DoesNotClaimBareIdentifier(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("SELECT create_hypertable FROM hypertable_registry")
	require.NoError(t, err)
	require.NotEmpty(t, statements)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])

	require.NoError(t, err)
	if mutation != nil {
		assert.Empty(t, mutation.EngineSpecific["TIMESCALE_HYPERTABLE"])
	}
}

func TestParseCreateHypertable_DoesNotAffectRegularCreateTable(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements("CREATE TABLE widgets (id BIGINT, name TEXT)")
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateTable, mutation.Kind)
	assert.Equal(t, "widgets", mutation.TableName)
	assert.Empty(t, mutation.EngineSpecific["TIMESCALE_HYPERTABLE"], "regular CREATE TABLE should not carry hypertable marker")
}
