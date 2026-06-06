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

// functions_round2_test.go verifies the round-2 TimescaleDB signature
// extensions: reorder policies, extended add_job / alter_job,
// extended add_retention_policy / add_continuous_aggregate_policy,
// show_chunks / drop_chunks multi-arg overloads, hypercore procedures,
// hypertable management lifecycle helpers, the toolkit aggregate
// families (freq_agg / mcv_agg, min_n / max_n, lttb / asap_smooth /
// timevector, count_min_sketch), saturating math, time_bucket on UUID
// columns, and the timescaledb_pre/post_restore lifecycle helpers.
// Each test goes through findOverload (defined in functions_test.go)
// to verify the registered signature matches what upstream exposes.

package db_engine_timescaledb_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_timescaledb"
)

const (
	engineNameRegclass               = "regclass"
	engineNameRegproc                = "regproc"
	engineNameVoid                   = "void"
	engineNameUUID                   = "uuid"
	engineNameJSONB                  = "jsonb"
	engineNameAny                    = ""
	engineNameName                   = "name"
	engineNameSpaceSavingBigintAgg   = "space_saving_bigint_aggregate"
	engineNameSpaceSavingTextAgg     = "space_saving_text_aggregate"
	engineNameSpaceSavingAgg         = "space_saving_aggregate"
	engineNameMinNState              = "min_n_state"
	engineNameMinNByState            = "min_n_by_state"
	engineNameTimevectorTstzF64      = "timevector_tstz_f64"
	engineNameCountMinSketch         = "count_min_sketch"
	engineNameCompressionStatsRecord = "compression_stats_record"
	engineNameAlterJobRecord         = "alter_job_record"
	engineNameRegclassArray          = "regclass[]"
)

func TestReorderPolicyFunctionsRegistered(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	add := findOverload(catalogue, "add_reorder_policy", []string{
		engineNameRegclass, engineNameText, engineNameBoolean, engineNameTimestamptz, engineNameText,
	})
	require.NotNil(t, add, "add_reorder_policy 5-arg form must be registered")
	assert.Equal(t, engineNameInteger, add.ReturnType.EngineName)

	remove := findOverload(catalogue, "remove_reorder_policy", []string{
		engineNameRegclass, engineNameBoolean,
	})
	require.NotNil(t, remove, "remove_reorder_policy 2-arg form must be registered")
	assert.Equal(t, engineNameVoid, remove.ReturnType.EngineName)
}

func TestAddJobUsesRegproc(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "add_job", []string{
		engineNameRegproc,
		engineNameInterval,
		engineNameJSONB,
		engineNameTimestamptz,
		engineNameBoolean,
		engineNameRegproc,
		engineNameBoolean,
		engineNameText,
	})
	require.NotNil(t, signature, "add_job 8-arg signature must be registered with regproc proc")
	assert.Equal(t, engineNameInteger, signature.ReturnType.EngineName)
}

func TestAlterJobReturnsRecord(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "alter_job", []string{
		engineNameInteger,
		engineNameInterval,
		engineNameInterval,
		engineNameInteger,
		engineNameInterval,
		engineNameBoolean,
		engineNameJSONB,
		engineNameTimestamptz,
		engineNameBoolean,
		engineNameRegproc,
		engineNameBoolean,
		engineNameTimestamptz,
		engineNameText,
	})
	require.NotNil(t, signature, "alter_job 13-arg signature must be registered")
	assert.Equal(t, engineNameAlterJobRecord, signature.ReturnType.EngineName)
}

func TestExtendedRetentionPolicy(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "add_retention_policy", []string{
		engineNameRegclass,
		engineNameAny,
		engineNameInterval,
		engineNameTimestamptz,
		engineNameText,
		engineNameBoolean,
		engineNameInterval,
	})
	require.NotNil(t, signature, "add_retention_policy 7-arg form must be registered")
	assert.Equal(t, engineNameInteger, signature.ReturnType.EngineName)
}

func TestExtendedCompressionPolicy(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "add_compression_policy", []string{
		engineNameRegclass,
		engineNameAny,
		engineNameBoolean,
		engineNameInterval,
		engineNameTimestamptz,
		engineNameText,
		engineNameInterval,
	})
	require.NotNil(t, signature, "add_compression_policy full form must be registered")
	assert.Equal(t, engineNameInteger, signature.ReturnType.EngineName)
	assert.Equal(t, 2, signature.MinArguments,
		"add_compression_policy must require only hypertable and compress_after, like the sibling policies")
}

func TestExtendedContinuousAggregatePolicy(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "add_continuous_aggregate_policy", []string{
		engineNameRegclass,
		engineNameAny,
		engineNameAny,
		engineNameInterval,
		engineNameBoolean,
		engineNameTimestamptz,
		engineNameText,
		engineNameBoolean,
		engineNameInteger,
		engineNameInteger,
		engineNameBoolean,
	})
	require.NotNil(t, signature, "add_continuous_aggregate_policy 11-arg form must be registered")
	assert.Equal(t, engineNameInteger, signature.ReturnType.EngineName)
}

func TestShowChunksMultiArgOverload(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "show_chunks", []string{
		engineNameRegclass,
		engineNameAny,
		engineNameAny,
		engineNameAny,
		engineNameAny,
	})
	require.NotNil(t, signature, "show_chunks multi-arg form must be registered")
	assert.Equal(t, engineNameRegclass, signature.ReturnType.EngineName)
	assert.True(t, signature.ReturnsSet, "show_chunks must return a set")
}

func TestDropChunksMultiArgOverload(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "drop_chunks", []string{
		engineNameRegclass,
		engineNameAny,
		engineNameAny,
		engineNameBoolean,
		engineNameAny,
		engineNameAny,
	})
	require.NotNil(t, signature, "drop_chunks multi-arg form must be registered")
	assert.True(t, signature.ReturnsSet, "drop_chunks must return a set")
}

func TestHypercoreProcedures(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	cases := []struct {
		name    string
		argList []string
	}{
		{name: "convert_to_columnstore", argList: []string{engineNameRegclass, engineNameBoolean, engineNameBoolean}},
		{name: "convert_to_rowstore", argList: []string{engineNameRegclass, engineNameBoolean}},
		{name: "recompress_chunk", argList: []string{engineNameRegclass, engineNameBoolean}},
		{name: "rebuild_columnstore", argList: []string{engineNameRegclass}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, testCase.name, testCase.argList)
			require.NotNil(t, signature, "%s must be registered", testCase.name)
			assert.Equal(t, engineNameVoid, signature.ReturnType.EngineName)
		})
	}
}

func TestColumnstoreStatsAliases(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	for _, name := range []string{"hypertable_columnstore_stats", "chunk_columnstore_stats"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, name, []string{engineNameRegclass})
			require.NotNil(t, signature, "%s must be registered", name)
			assert.Equal(t, engineNameCompressionStatsRecord, signature.ReturnType.EngineName)
			assert.True(t, signature.ReturnsSet, "%s must return a set", name)
		})
	}
}

func TestMoveChunkSignature(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "move_chunk", []string{
		engineNameRegclass,
		engineNameName,
		engineNameName,
		engineNameRegclass,
		engineNameBoolean,
	})
	require.NotNil(t, signature, "move_chunk 5-arg form must be registered")
	assert.Equal(t, engineNameVoid, signature.ReturnType.EngineName)
}

func TestTablespaceAttachDetach(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	attach := findOverload(catalogue, "attach_tablespace", []string{
		engineNameText, engineNameRegclass, engineNameBoolean,
	})
	require.NotNil(t, attach, "attach_tablespace must be registered")
	assert.Equal(t, engineNameVoid, attach.ReturnType.EngineName)

	detach := findOverload(catalogue, "detach_tablespace", []string{
		engineNameText, engineNameRegclass, engineNameBoolean,
	})
	require.NotNil(t, detach, "detach_tablespace must be registered")
	assert.Equal(t, engineNameInteger, detach.ReturnType.EngineName)

	detachAll := findOverload(catalogue, "detach_tablespaces", []string{engineNameRegclass})
	require.NotNil(t, detachAll, "detach_tablespaces must be registered")
	assert.Equal(t, engineNameInteger, detachAll.ReturnType.EngineName)
}

func TestMergeAndSplitChunks(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	pair := findOverload(catalogue, "merge_chunks", []string{
		engineNameRegclass, engineNameRegclass, engineNameBoolean,
	})
	require.NotNil(t, pair, "merge_chunks(chunk1, chunk2, concurrently) must be registered")

	arrayForm := findOverload(catalogue, "merge_chunks", []string{
		engineNameRegclassArray, engineNameBoolean,
	})
	require.NotNil(t, arrayForm, "merge_chunks(chunks[], concurrently) must be registered")

	concurrently := findOverload(catalogue, "merge_chunks_concurrently", []string{engineNameRegclassArray})
	require.NotNil(t, concurrently, "merge_chunks_concurrently must be registered")
	assert.True(t, concurrently.IsVariadic, "merge_chunks_concurrently must be variadic")

	split := findOverload(catalogue, "split_chunk", []string{engineNameRegclass, engineNameTimestamptz})
	require.NotNil(t, split, "split_chunk must be registered")

	attach := findOverload(catalogue, "attach_chunk", []string{
		engineNameRegclass, engineNameRegclass, engineNameJSONB,
	})
	require.NotNil(t, attach, "attach_chunk must be registered")

	detach := findOverload(catalogue, "detach_chunk", []string{engineNameRegclass})
	require.NotNil(t, detach, "detach_chunk must be registered")

	create := findOverload(catalogue, "create_chunk", []string{
		engineNameRegclass, engineNameJSONB, engineNameText, engineNameText, engineNameRegclass,
	})
	require.NotNil(t, create, "create_chunk must be registered")

	drop := findOverload(catalogue, "drop_chunk", []string{engineNameRegclass})
	require.NotNil(t, drop, "drop_chunk must be registered")
	assert.Equal(t, engineNameBoolean, drop.ReturnType.EngineName)
}

func TestHypertableApproximateSize(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	scalar := findOverload(catalogue, "hypertable_approximate_size", []string{engineNameRegclass})
	require.NotNil(t, scalar, "hypertable_approximate_size must be registered")
	assert.Equal(t, engineNameBigint, scalar.ReturnType.EngineName)

	detailed := findOverload(catalogue, "hypertable_approximate_detailed_size", []string{engineNameRegclass})
	require.NotNil(t, detailed, "hypertable_approximate_detailed_size must be registered")
	assert.True(t, detailed.ReturnsSet, "hypertable_approximate_detailed_size must return a set")
}

func TestFreqAggregateConstructors(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	freqBigint := findOverload(catalogue, "freq_agg", []string{engineNameFloat8, engineNameBigint})
	require.NotNil(t, freqBigint, "freq_agg(frequency, bigint) must be registered")
	assert.True(t, freqBigint.IsAggregate)
	assert.Equal(t, engineNameSpaceSavingBigintAgg, freqBigint.ReturnType.EngineName)

	freqText := findOverload(catalogue, "freq_agg", []string{engineNameFloat8, engineNameText})
	require.NotNil(t, freqText, "freq_agg(frequency, text) must be registered")
	assert.Equal(t, engineNameSpaceSavingTextAgg, freqText.ReturnType.EngineName)

	mcvBigint := findOverload(catalogue, "mcv_agg", []string{engineNameInteger, engineNameBigint})
	require.NotNil(t, mcvBigint, "mcv_agg(count, bigint) must be registered")
	assert.Equal(t, engineNameSpaceSavingBigintAgg, mcvBigint.ReturnType.EngineName)

	mcvText := findOverload(catalogue, "mcv_agg", []string{engineNameInteger, engineNameText})
	require.NotNil(t, mcvText, "mcv_agg(count, text) must be registered")
	assert.Equal(t, engineNameSpaceSavingTextAgg, mcvText.ReturnType.EngineName)

	mcvSkewBigint := findOverload(catalogue, "mcv_agg", []string{
		engineNameInteger, engineNameFloat8, engineNameBigint,
	})
	require.NotNil(t, mcvSkewBigint, "mcv_agg(count, skew, bigint) must be registered")

	rawFreq := findOverload(catalogue, "raw_freq_agg", []string{engineNameFloat8, engineNameAny})
	require.NotNil(t, rawFreq, "raw_freq_agg must be registered")
	assert.Equal(t, engineNameSpaceSavingAgg, rawFreq.ReturnType.EngineName)
}

func TestFreqAggregateAccessors(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	topn := findOverload(catalogue, "topn", []string{engineNameSpaceSavingBigintAgg, engineNameInteger})
	require.NotNil(t, topn, "topn(bigint_agg, n) must be registered")
	assert.True(t, topn.ReturnsSet)

	intoValues := findOverload(catalogue, "into_values", []string{engineNameSpaceSavingTextAgg})
	require.NotNil(t, intoValues, "into_values(text_agg) must be registered")
	assert.True(t, intoValues.ReturnsSet)

	maxFreq := findOverload(catalogue, "max_frequency", []string{engineNameSpaceSavingAgg, engineNameAny})
	require.NotNil(t, maxFreq, "max_frequency(agg, value) must be registered")
	assert.Equal(t, engineNameFloat8, maxFreq.ReturnType.EngineName)

	minFreq := findOverload(catalogue, "min_frequency", []string{engineNameSpaceSavingAgg, engineNameAny})
	require.NotNil(t, minFreq, "min_frequency(agg, value) must be registered")
	assert.Equal(t, engineNameFloat8, minFreq.ReturnType.EngineName)
}

func TestFrequencyRollup(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	for _, stateName := range []string{
		engineNameSpaceSavingBigintAgg,
		engineNameSpaceSavingTextAgg,
		engineNameSpaceSavingAgg,
	} {
		t.Run(stateName, func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, "rollup", []string{stateName})
			require.NotNil(t, signature, "rollup(%s) must be registered", stateName)
			assert.True(t, signature.IsAggregate)
			assert.Equal(t, stateName, signature.ReturnType.EngineName)
		})
	}
}

func TestMinMaxNAggregates(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	for _, valueType := range []string{engineNameFloat8, engineNameBigint, engineNameText, engineNameTimestamptz} {
		t.Run("min_n/"+valueType, func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, "min_n", []string{valueType, engineNameInteger})
			require.NotNil(t, signature, "min_n(%s, n) must be registered", valueType)
			assert.True(t, signature.IsAggregate)
			assert.Equal(t, engineNameMinNState, signature.ReturnType.EngineName)
		})
		t.Run("max_n/"+valueType, func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, "max_n", []string{valueType, engineNameInteger})
			require.NotNil(t, signature, "max_n(%s, n) must be registered", valueType)
		})
		t.Run("min_n_by/"+valueType, func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, "min_n_by", []string{valueType, engineNameAny, engineNameInteger})
			require.NotNil(t, signature, "min_n_by(%s, by, n) must be registered", valueType)
			assert.Equal(t, engineNameMinNByState, signature.ReturnType.EngineName)
		})
		t.Run("max_n_by/"+valueType, func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, "max_n_by", []string{valueType, engineNameAny, engineNameInteger})
			require.NotNil(t, signature, "max_n_by(%s, by, n) must be registered", valueType)
		})
	}
}

func TestMinNAccessors(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	intoArray := findOverload(catalogue, "into_array", []string{engineNameMinNState})
	require.NotNil(t, intoArray, "into_array(min_n_state) must be registered")
	assert.Equal(t, querier_dto.TypeCategoryArray, intoArray.ReturnType.Category)

	intoValues := findOverload(catalogue, "into_values", []string{engineNameMinNState})
	require.NotNil(t, intoValues, "into_values(min_n_state) must be registered")
	assert.True(t, intoValues.ReturnsSet)
}

func TestTimevectorFamily(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	lttbAgg := findOverload(catalogue, "lttb", []string{
		engineNameTimestamptz, engineNameFloat8, engineNameInteger,
	})
	require.NotNil(t, lttbAgg, "lttb aggregate must be registered")
	assert.True(t, lttbAgg.IsAggregate)
	assert.Equal(t, engineNameTimevectorTstzF64, lttbAgg.ReturnType.EngineName)

	lttbState := findOverload(catalogue, "lttb", []string{engineNameTimevectorTstzF64, engineNameInteger})
	require.NotNil(t, lttbState, "lttb state-to-state form must be registered")
	assert.False(t, lttbState.IsAggregate)

	asapAgg := findOverload(catalogue, "asap_smooth", []string{
		engineNameTimestamptz, engineNameFloat8, engineNameInteger,
	})
	require.NotNil(t, asapAgg, "asap_smooth aggregate must be registered")
	assert.True(t, asapAgg.IsAggregate)

	tv := findOverload(catalogue, "timevector", []string{engineNameTimestamptz, engineNameFloat8})
	require.NotNil(t, tv, "timevector aggregate must be registered")

	rollup := findOverload(catalogue, "rollup", []string{engineNameTimevectorTstzF64})
	require.NotNil(t, rollup, "rollup(timevector_tstz_f64) must be registered")

	unnest := findOverload(catalogue, "unnest", []string{engineNameTimevectorTstzF64})
	require.NotNil(t, unnest, "unnest(timevector_tstz_f64) must be registered")
	assert.True(t, unnest.ReturnsSet)
}

func TestCountMinSketch(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	agg := findOverload(catalogue, "count_min_sketch", []string{
		engineNameText, engineNameFloat8, engineNameFloat8,
	})
	require.NotNil(t, agg, "count_min_sketch aggregate must be registered")
	assert.True(t, agg.IsAggregate)
	assert.Equal(t, engineNameCountMinSketch, agg.ReturnType.EngineName)

	approx := findOverload(catalogue, "approx_count", []string{engineNameText, engineNameCountMinSketch})
	require.NotNil(t, approx, "approx_count(item, sketch) must be registered")
	assert.Equal(t, engineNameBigint, approx.ReturnType.EngineName)
}

func TestSaturatingMath(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	for _, name := range []string{
		"saturating_add",
		"saturating_sub",
		"saturating_mul",
		"saturating_add_pos",
		"saturating_sub_pos",
		"saturating_mul_pos",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, name, []string{engineNameInteger, engineNameInteger})
			require.NotNil(t, signature, "%s must be registered", name)
			assert.Equal(t, engineNameInteger, signature.ReturnType.EngineName)
		})
	}
}

func TestTimeBucketUUIDOverloads(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	bare := findOverload(catalogue, "time_bucket", []string{engineNameInterval, engineNameUUID})
	require.NotNil(t, bare, "time_bucket(interval, uuid) must be registered")
	assert.Equal(t, engineNameTimestamptz, bare.ReturnType.EngineName)

	tz := findOverload(catalogue, "time_bucket", []string{engineNameInterval, engineNameUUID, engineNameText})
	require.NotNil(t, tz, "time_bucket(interval, uuid, timezone) must be registered")

	origin := findOverload(catalogue, "time_bucket", []string{
		engineNameInterval, engineNameUUID, engineNameTimestamptz,
	})
	require.NotNil(t, origin, "time_bucket(interval, uuid, origin) must be registered")

	offset := findOverload(catalogue, "time_bucket", []string{
		engineNameInterval, engineNameUUID, engineNameInterval,
	})
	require.NotNil(t, offset, "time_bucket(interval, uuid, offset) must be registered")
}

func TestExtensionLifecycleFunctions(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	pre := findOverload(catalogue, "timescaledb_pre_restore", nil)
	require.NotNil(t, pre, "timescaledb_pre_restore must be registered")
	assert.Equal(t, engineNameVoid, pre.ReturnType.EngineName)

	post := findOverload(catalogue, "timescaledb_post_restore", nil)
	require.NotNil(t, post, "timescaledb_post_restore must be registered")
	assert.Equal(t, engineNameVoid, post.ReturnType.EngineName)
}
