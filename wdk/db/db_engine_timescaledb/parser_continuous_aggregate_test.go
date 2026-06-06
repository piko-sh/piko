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

func TestContinuousAggregate_Basic(t *testing.T) {
	t.Parallel()

	sql := `CREATE MATERIALIZED VIEW hourly_temps
		WITH (timescaledb.continuous = true)
		AS SELECT time_bucket('1 hour', ts) AS bucket, avg(temperature) FROM readings GROUP BY bucket`

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, statements, 1)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, querier_dto.MutationCreateView, mutation.Kind)
	assert.Equal(t, "hourly_temps", mutation.TableName)
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_CONTINUOUS_AGGREGATE"])
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_CONTINUOUS_FLAG"])
	assert.NotEmpty(t, mutation.EngineSpecific["TIMESCALE_VIEW_BODY"])
}

func TestContinuousAggregate_ViewDefinitionTypesColumns(t *testing.T) {
	t.Parallel()

	sql := `CREATE MATERIALIZED VIEW hourly_temps
		WITH (timescaledb.continuous = true)
		AS SELECT time_bucket('1 hour', ts) AS bucket, avg(temperature) AS avg_temp FROM readings GROUP BY bucket`

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, statements, 1)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	require.NotNil(t, mutation.ViewDefinition, "continuous aggregate must carry a parsed view definition for column typing")
	require.Len(t, mutation.ViewDefinition.OutputColumns, 2)
	assert.Equal(t, "bucket", mutation.ViewDefinition.OutputColumns[0].Name)
	assert.Equal(t, "avg_temp", mutation.ViewDefinition.OutputColumns[1].Name)
}

func TestContinuousAggregate_MaterializedOnly(t *testing.T) {
	t.Parallel()

	sql := `CREATE MATERIALIZED VIEW daily_stats
		WITH (timescaledb.continuous = true, timescaledb.materialized_only = true)
		AS SELECT time_bucket('1 day', ts) AS bucket, count(*) FROM events GROUP BY bucket`

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "true", mutation.EngineSpecific["TIMESCALE_MATERIALIZED_ONLY"])
}

func TestContinuousAggregate_WithNoData(t *testing.T) {
	t.Parallel()

	sql := `CREATE MATERIALIZED VIEW hourly_temps
		WITH (timescaledb.continuous = true)
		AS SELECT time_bucket('1 hour', ts) FROM readings WITH NO DATA`

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "false", mutation.EngineSpecific["TIMESCALE_WITH_DATA"])
}

func TestContinuousAggregate_SchemaQualified(t *testing.T) {
	t.Parallel()

	sql := `CREATE MATERIALIZED VIEW analytics.hourly_temps
		WITH (timescaledb.continuous = true)
		AS SELECT 1`

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "analytics", mutation.SchemaName)
	assert.Equal(t, "hourly_temps", mutation.TableName)
}

func TestContinuousAggregate_CTEBody(t *testing.T) {
	t.Parallel()

	sql := `CREATE MATERIALIZED VIEW daily_summary
		WITH (timescaledb.continuous = true)
		AS WITH source AS (SELECT * FROM readings)
		SELECT time_bucket('1 day', ts) AS bucket FROM source GROUP BY bucket`

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	body := mutation.EngineSpecific["TIMESCALE_VIEW_BODY"]
	assert.Contains(t, body, "WITH source")
	assert.Contains(t, body, "GROUP BY bucket")
}

func TestContinuousAggregate_PreservesUnknownReloption(t *testing.T) {
	t.Parallel()

	sql := `CREATE MATERIALIZED VIEW hourly_temps
		WITH (timescaledb.continuous = true, timescaledb.future_option = 'value')
		AS SELECT 1`

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Equal(t, "value", mutation.EngineSpecific["TIMESCALE_RELOPTION_TIMESCALEDB.FUTURE_OPTION"],
		"unknown timescaledb.* reloptions must be preserved verbatim")
}

func TestContinuousAggregate_ViewBodyRoundTripsStringLiterals(t *testing.T) {
	t.Parallel()

	sql := `CREATE MATERIALIZED VIEW hourly_temps
		WITH (timescaledb.continuous = true)
		AS SELECT time_bucket('1 hour', ts) FROM readings WHERE kind = 'sensor'`

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	body := mutation.EngineSpecific["TIMESCALE_VIEW_BODY"]
	assert.Contains(t, body, "'1 hour'", "string literals must round-trip with surrounding quotes")
	assert.Contains(t, body, "'sensor'")
}

func TestContinuousAggregate_ViewBodyRoundTripsNonPlainLiterals(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		body        string
		mustContain string
	}{
		{
			description: "bit-string literal keeps its B prefix and quotes",
			body:        `SELECT id FROM readings WHERE flags = B'1010'`,
			mustContain: "B'1010'",
		},
		{
			description: "dollar-quoted literal keeps its delimiters",
			body:        `SELECT id FROM readings WHERE note = $tag$body$tag$`,
			mustContain: "$$body$$",
		},
		{
			description: "quoted identifier keeps its double quotes",
			body:        `SELECT "Weird Col" FROM readings`,
			mustContain: `"Weird Col"`,
		},
	}

	engine := db_engine_timescaledb.NewTimescaleDBEngine()

	for _, testCase := range cases {
		t.Run(testCase.description, func(t *testing.T) {
			t.Parallel()

			sql := "CREATE MATERIALIZED VIEW v\n" +
				"WITH (timescaledb.continuous = true)\n" +
				"AS " + testCase.body

			statements, err := engine.ParseStatements(sql)
			require.NoError(t, err)
			require.Len(t, statements, 1)

			mutation, err := engine.ApplyDDL(context.Background(), statements[0])
			require.NoError(t, err)
			require.NotNil(t, mutation)

			body := mutation.EngineSpecific["TIMESCALE_VIEW_BODY"]
			assert.Contains(t, body, testCase.mustContain,
				"captured view body must re-wrap non-plain literals with their delimiters, got %q", body)
		})
	}
}

func TestContinuousAggregate_DoesNotClaimWhenContinuousMentionedInBody(t *testing.T) {
	t.Parallel()

	sql := `CREATE MATERIALIZED VIEW summary
		AS SELECT timescaledb.continuous FROM tsconfig`

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Empty(t, mutation.EngineSpecific["TIMESCALE_CONTINUOUS_AGGREGATE"])
}

func TestContinuousAggregate_RegularMaterializedViewUnaffected(t *testing.T) {
	t.Parallel()

	sql := `CREATE MATERIALIZED VIEW summary AS SELECT 1`

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)

	mutation, err := engine.ApplyDDL(context.Background(), statements[0])
	require.NoError(t, err)
	require.NotNil(t, mutation)
	assert.Empty(t, mutation.EngineSpecific["TIMESCALE_CONTINUOUS_AGGREGATE"])
}
