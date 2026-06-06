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

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_timescaledb"
)

const (
	engineNameCounterSummary  = "counter_summary"
	engineNameStatsSummary2D  = "statssummary2d"
	engineNameTimeWeightSum   = "time_weight_summary"
	engineNameHeartbeat       = "heartbeat"
	engineNameStateSummary    = "state_summary"
	engineNameCompactStateAgg = "compact_state_agg"
	engineNameCandlestick     = "candlestick"
	engineNameTDigest         = "tdigest"
	engineNameUDDSketch       = "uddsketch"
	engineNameTstzrange       = "tstzrange"
	engineNameFloat8          = "float8"
	engineNameBigint          = "int8"
	engineNameInteger         = "int4"
	engineNameSmallint        = "int2"
	engineNameTimestamptz     = "timestamptz"
	engineNameInterval        = "interval"
	engineNameText            = "text"
	engineNameBoolean         = "bool"
)

func findOverload(catalogue *querier_dto.FunctionCatalogue, name string, engineNames []string) *querier_dto.FunctionSignature {
	overloads, exists := catalogue.Functions[name]
	if !exists {
		return nil
	}
	for _, signature := range overloads {
		if len(signature.Arguments) != len(engineNames) {
			continue
		}
		matched := true
		for index, argument := range signature.Arguments {
			if argument.Type.EngineName != engineNames[index] {
				matched = false
				break
			}
		}
		if matched {
			return signature
		}
	}
	return nil
}

func TestTimeDeltaReturnsFloat8(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "time_delta", []string{engineNameCounterSummary})
	require.NotNil(t, signature, "time_delta(counter_summary) must be registered")
	assert.Equal(t, engineNameFloat8, signature.ReturnType.EngineName,
		"time_delta returns float8 seconds, not an interval")
	assert.Equal(t, querier_dto.TypeCategoryFloat, signature.ReturnType.Category)
}

func TestCorrIsRegistered(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "corr", []string{engineNameStatsSummary2D})
	require.NotNil(t, signature, "corr(statssummary2d) must be the canonical toolkit name")
	assert.Equal(t, engineNameFloat8, signature.ReturnType.EngineName)
}

func TestCorrelationAliasIsDropped(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "correlation", []string{engineNameStatsSummary2D})
	assert.Nil(t, signature, "correlation alias should be dropped; only corr is canonical")
}

func TestTimeBucketNgIsRemoved(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	_, exists := catalogue.Functions["time_bucket_ng"]
	assert.False(t, exists, "time_bucket_ng was removed upstream and must not resolve")
}

func TestTopkIsRemoved(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	_, exists := catalogue.Functions["topk"]
	assert.False(t, exists, "topk(k, value) does not exist in TimescaleDB and was removed")
}

func TestRateIntervalOverloadIsRemoved(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "rate", []string{engineNameCounterSummary, engineNameInterval})
	assert.Nil(t, signature, "rate(summary, interval) is not a real upstream overload")

	primary := findOverload(catalogue, "rate", []string{engineNameCounterSummary})
	require.NotNil(t, primary, "rate(summary) must remain registered")
	assert.Equal(t, engineNameFloat8, primary.ReturnType.EngineName)
}

func TestTimeBucketIntegerOverloads(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	cases := []struct {
		description string
		engineName  string
	}{
		{description: "smallint epoch column", engineName: engineNameSmallint},
		{description: "integer epoch column", engineName: engineNameInteger},
		{description: "bigint epoch column", engineName: engineNameBigint},
	}

	for _, testCase := range cases {
		t.Run(testCase.description, func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, "time_bucket", []string{testCase.engineName, testCase.engineName})
			require.NotNil(t, signature, "time_bucket(%s, %s) must be registered", testCase.engineName, testCase.engineName)
			assert.Equal(t, testCase.engineName, signature.ReturnType.EngineName)

			offsetSignature := findOverload(catalogue, "time_bucket",
				[]string{testCase.engineName, testCase.engineName, testCase.engineName})
			require.NotNil(t, offsetSignature,
				"time_bucket(%s, %s, %s) offset/origin overload must be registered",
				testCase.engineName, testCase.engineName, testCase.engineName)
			assert.Equal(t, testCase.engineName, offsetSignature.ReturnType.EngineName)
		})
	}
}

func TestTimeBucketGapfillTimezoneOverload(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "time_bucket_gapfill", []string{
		engineNameInterval, engineNameTimestamptz, engineNameText, engineNameTimestamptz, engineNameTimestamptz,
	})
	require.NotNil(t, signature, "time_bucket_gapfill 5-arg timezone form must be registered")
	assert.Equal(t, engineNameTimestamptz, signature.ReturnType.EngineName)
}

func TestCounterSummaryAccessors(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	cases := []struct {
		name       string
		returnName string
	}{
		{name: "num_changes", returnName: engineNameBigint},
		{name: "num_resets", returnName: engineNameBigint},
		{name: "num_elements", returnName: engineNameBigint},
		{name: "first_val", returnName: engineNameFloat8},
		{name: "last_val", returnName: engineNameFloat8},
		{name: "first_time", returnName: engineNameTimestamptz},
		{name: "last_time", returnName: engineNameTimestamptz},
		{name: "slope", returnName: engineNameFloat8},
		{name: "intercept", returnName: engineNameFloat8},
		{name: "corr", returnName: engineNameFloat8},
		{name: "counter_zero_time", returnName: engineNameTimestamptz},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, testCase.name, []string{engineNameCounterSummary})
			require.NotNil(t, signature, "%s(counter_summary) must be registered", testCase.name)
			assert.Equal(t, testCase.returnName, signature.ReturnType.EngineName)
		})
	}
}

func TestCounterExtrapolationMethodOverloads(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	cases := []string{"extrapolated_delta", "extrapolated_rate"}

	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, name, []string{engineNameCounterSummary, engineNameText})
			require.NotNil(t, signature, "%s(summary, method) must be registered", name)
			assert.Equal(t, engineNameFloat8, signature.ReturnType.EngineName)
		})
	}
}

func TestCounterWithBoundsAndThreeArgAgg(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	withBounds := findOverload(catalogue, "with_bounds", []string{engineNameCounterSummary, engineNameTstzrange})
	require.NotNil(t, withBounds, "with_bounds(summary, tstzrange) must be registered")
	assert.Equal(t, engineNameCounterSummary, withBounds.ReturnType.EngineName)

	threeArgAgg := findOverload(catalogue, "counter_agg", []string{engineNameTimestamptz, engineNameFloat8, engineNameTstzrange})
	require.NotNil(t, threeArgAgg, "counter_agg(timestamptz, float8, tstzrange) must be registered")
	assert.True(t, threeArgAgg.IsAggregate)
	assert.Equal(t, engineNameCounterSummary, threeArgAgg.ReturnType.EngineName)
}

func TestStats2DAxisAccessors(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	names := []string{
		"average_x", "average_y",
		"sum_x", "sum_y",
		"stddev_x", "stddev_y",
		"variance_x", "variance_y",
		"skewness_x", "skewness_y",
		"kurtosis_x", "kurtosis_y",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, name, []string{engineNameStatsSummary2D})
			require.NotNil(t, signature, "%s(statssummary2d) must be registered", name)
			assert.Equal(t, engineNameFloat8, signature.ReturnType.EngineName)
		})
	}
}

func TestStatsMethodTextOverloads(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	for _, name := range []string{"stddev", "variance", "skewness", "kurtosis"} {
		t.Run(name+" 1D method overload", func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, name, []string{"statssummary1d", engineNameText})
			require.NotNil(t, signature, "%s(statssummary1d, method) must be registered", name)
			assert.Equal(t, engineNameFloat8, signature.ReturnType.EngineName)
		})
		t.Run(name+" 2D method overload", func(t *testing.T) {
			t.Parallel()

			signature := findOverload(catalogue, name, []string{engineNameStatsSummary2D, engineNameText})
			require.NotNil(t, signature, "%s(statssummary2d, method) must be registered", name)
			assert.Equal(t, engineNameFloat8, signature.ReturnType.EngineName)
		})
	}

	covariance := findOverload(catalogue, "covariance", []string{engineNameStatsSummary2D, engineNameText})
	require.NotNil(t, covariance, "covariance(statssummary2d, method) must be registered")
	assert.Equal(t, engineNameFloat8, covariance.ReturnType.EngineName)
}

func TestPercentileAggReturnsUDDSketch(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	signature := findOverload(catalogue, "percentile_agg", []string{engineNameFloat8})
	require.NotNil(t, signature, "percentile_agg(float8) must be registered as an aggregate")
	assert.True(t, signature.IsAggregate)
	assert.Equal(t, engineNameUDDSketch, signature.ReturnType.EngineName)
}

func TestPercentileAccessors(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	tdigestRank := findOverload(catalogue, "approx_percentile_rank", []string{engineNameFloat8, engineNameTDigest})
	require.NotNil(t, tdigestRank)
	assert.Equal(t, engineNameFloat8, tdigestRank.ReturnType.EngineName)

	uddsketchRank := findOverload(catalogue, "approx_percentile_rank", []string{engineNameFloat8, engineNameUDDSketch})
	require.NotNil(t, uddsketchRank)
	assert.Equal(t, engineNameFloat8, uddsketchRank.ReturnType.EngineName)

	tdigestMean := findOverload(catalogue, "mean", []string{engineNameTDigest})
	require.NotNil(t, tdigestMean)
	assert.Equal(t, engineNameFloat8, tdigestMean.ReturnType.EngineName)

	uddsketchMean := findOverload(catalogue, "mean", []string{engineNameUDDSketch})
	require.NotNil(t, uddsketchMean)
	assert.Equal(t, engineNameFloat8, uddsketchMean.ReturnType.EngineName)

	tdigestNumVals := findOverload(catalogue, "num_vals", []string{engineNameTDigest})
	require.NotNil(t, tdigestNumVals)
	assert.Equal(t, engineNameBigint, tdigestNumVals.ReturnType.EngineName)

	uddsketchNumVals := findOverload(catalogue, "num_vals", []string{engineNameUDDSketch})
	require.NotNil(t, uddsketchNumVals)
	assert.Equal(t, engineNameBigint, uddsketchNumVals.ReturnType.EngineName)

	minVal := findOverload(catalogue, "min_val", []string{engineNameTDigest})
	require.NotNil(t, minVal)
	assert.Equal(t, engineNameFloat8, minVal.ReturnType.EngineName)

	maxVal := findOverload(catalogue, "max_val", []string{engineNameTDigest})
	require.NotNil(t, maxVal)
	assert.Equal(t, engineNameFloat8, maxVal.ReturnType.EngineName)
}

func TestCandlestickConstructorAndTimeAccessors(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	constructor := findOverload(catalogue, "candlestick", []string{
		engineNameTimestamptz,
		engineNameFloat8, engineNameFloat8, engineNameFloat8, engineNameFloat8, engineNameFloat8,
	})
	require.NotNil(t, constructor, "candlestick(timestamptz, float8x5) constructor must be registered")
	assert.Equal(t, engineNameCandlestick, constructor.ReturnType.EngineName)

	highTime := findOverload(catalogue, "high_time", []string{engineNameCandlestick})
	require.NotNil(t, highTime)
	assert.Equal(t, engineNameTimestamptz, highTime.ReturnType.EngineName)

	lowTime := findOverload(catalogue, "low_time", []string{engineNameCandlestick})
	require.NotNil(t, lowTime)
	assert.Equal(t, engineNameTimestamptz, lowTime.ReturnType.EngineName)
}

func TestTimeWeightFamily(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	constructor := findOverload(catalogue, "time_weight",
		[]string{engineNameText, engineNameTimestamptz, engineNameFloat8})
	require.NotNil(t, constructor, "time_weight(method, timestamptz, float8) aggregate must be registered")
	assert.True(t, constructor.IsAggregate)
	assert.Equal(t, engineNameTimeWeightSum, constructor.ReturnType.EngineName)

	avg := findOverload(catalogue, "average", []string{engineNameTimeWeightSum})
	require.NotNil(t, avg)
	assert.Equal(t, engineNameFloat8, avg.ReturnType.EngineName)

	integralOne := findOverload(catalogue, "integral", []string{engineNameTimeWeightSum})
	require.NotNil(t, integralOne)
	assert.Equal(t, engineNameFloat8, integralOne.ReturnType.EngineName)

	integralTwo := findOverload(catalogue, "integral", []string{engineNameTimeWeightSum, engineNameText})
	require.NotNil(t, integralTwo, "integral(summary, unit) overload must be registered")
	assert.Equal(t, engineNameFloat8, integralTwo.ReturnType.EngineName)

	firstVal := findOverload(catalogue, "first_val", []string{engineNameTimeWeightSum})
	require.NotNil(t, firstVal)
	assert.Equal(t, engineNameFloat8, firstVal.ReturnType.EngineName)

	lastTime := findOverload(catalogue, "last_time", []string{engineNameTimeWeightSum})
	require.NotNil(t, lastTime)
	assert.Equal(t, engineNameTimestamptz, lastTime.ReturnType.EngineName)

	interpolatedAvg := findOverload(catalogue, "interpolated_average", []string{
		engineNameTimeWeightSum, engineNameTimestamptz, engineNameInterval,
		engineNameTimeWeightSum, engineNameTimeWeightSum,
	})
	require.NotNil(t, interpolatedAvg, "interpolated_average must be registered with 5 args")
	assert.Equal(t, engineNameFloat8, interpolatedAvg.ReturnType.EngineName)

	rollup := findOverload(catalogue, "rollup", []string{engineNameTimeWeightSum})
	require.NotNil(t, rollup, "rollup(time_weight_summary) must be registered")
	assert.True(t, rollup.IsAggregate)
	assert.Equal(t, engineNameTimeWeightSum, rollup.ReturnType.EngineName)
}

func TestHeartbeatAccessorFamily(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	live := findOverload(catalogue, "live_ranges", []string{engineNameHeartbeat})
	require.NotNil(t, live, "live_ranges(heartbeat) must be registered as a set-returning function")
	assert.True(t, live.ReturnsSet)

	dead := findOverload(catalogue, "dead_ranges", []string{engineNameHeartbeat})
	require.NotNil(t, dead)
	assert.True(t, dead.ReturnsSet)

	uptime := findOverload(catalogue, "uptime", []string{engineNameHeartbeat})
	require.NotNil(t, uptime)
	assert.Equal(t, engineNameInterval, uptime.ReturnType.EngineName)

	downtime := findOverload(catalogue, "downtime", []string{engineNameHeartbeat})
	require.NotNil(t, downtime)
	assert.Equal(t, engineNameInterval, downtime.ReturnType.EngineName)

	liveAt := findOverload(catalogue, "live_at", []string{engineNameHeartbeat, engineNameTimestamptz})
	require.NotNil(t, liveAt)
	assert.Equal(t, engineNameBoolean, liveAt.ReturnType.EngineName)

	numLive := findOverload(catalogue, "num_live_ranges", []string{engineNameHeartbeat})
	require.NotNil(t, numLive)
	assert.Equal(t, engineNameInteger, numLive.ReturnType.EngineName)

	numGaps := findOverload(catalogue, "num_gaps", []string{engineNameHeartbeat})
	require.NotNil(t, numGaps)
	assert.Equal(t, engineNameInteger, numGaps.ReturnType.EngineName)

	interpolatedUp := findOverload(catalogue, "interpolated_uptime",
		[]string{engineNameHeartbeat, engineNameHeartbeat})
	require.NotNil(t, interpolatedUp)
	assert.Equal(t, engineNameInterval, interpolatedUp.ReturnType.EngineName)

	interpolatedDown := findOverload(catalogue, "interpolated_downtime",
		[]string{engineNameHeartbeat, engineNameHeartbeat})
	require.NotNil(t, interpolatedDown)
	assert.Equal(t, engineNameInterval, interpolatedDown.ReturnType.EngineName)

	trimTo := findOverload(catalogue, "trim_to",
		[]string{engineNameHeartbeat, engineNameTimestamptz, engineNameInterval})
	require.NotNil(t, trimTo)
	assert.Equal(t, engineNameHeartbeat, trimTo.ReturnType.EngineName)
}

func TestStateAggAdditions(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	bigintAgg := findOverload(catalogue, "state_agg", []string{engineNameTimestamptz, engineNameBigint})
	require.NotNil(t, bigintAgg, "state_agg(timestamptz, bigint) must be registered")
	assert.True(t, bigintAgg.IsAggregate)
	assert.Equal(t, engineNameStateSummary, bigintAgg.ReturnType.EngineName)

	compactText := findOverload(catalogue, "compact_state_agg", []string{engineNameTimestamptz, engineNameText})
	require.NotNil(t, compactText)
	assert.True(t, compactText.IsAggregate)
	assert.Equal(t, engineNameCompactStateAgg, compactText.ReturnType.EngineName)

	compactBigint := findOverload(catalogue, "compact_state_agg", []string{engineNameTimestamptz, engineNameBigint})
	require.NotNil(t, compactBigint)
	assert.Equal(t, engineNameCompactStateAgg, compactBigint.ReturnType.EngineName)

	durationSlice := findOverload(catalogue, "duration_in", []string{
		engineNameStateSummary, engineNameText, engineNameTimestamptz, engineNameInterval,
	})
	require.NotNil(t, durationSlice, "duration_in(summary, state, start, duration) must be registered")
	assert.Equal(t, engineNameInterval, durationSlice.ReturnType.EngineName)

	timeline := findOverload(catalogue, "state_timeline", []string{engineNameStateSummary})
	require.NotNil(t, timeline)
	assert.True(t, timeline.ReturnsSet)

	intTimeline := findOverload(catalogue, "state_int_timeline", []string{engineNameStateSummary})
	require.NotNil(t, intTimeline)
	assert.True(t, intTimeline.ReturnsSet)

	periods := findOverload(catalogue, "state_periods", []string{engineNameStateSummary, engineNameText})
	require.NotNil(t, periods)
	assert.True(t, periods.ReturnsSet)

	intPeriods := findOverload(catalogue, "state_int_periods", []string{engineNameStateSummary, engineNameBigint})
	require.NotNil(t, intPeriods)
	assert.True(t, intPeriods.ReturnsSet)

	stateAt := findOverload(catalogue, "state_at", []string{engineNameStateSummary, engineNameTimestamptz})
	require.NotNil(t, stateAt)
	assert.Equal(t, engineNameText, stateAt.ReturnType.EngineName)

	stateAtInt := findOverload(catalogue, "state_at_int", []string{engineNameStateSummary, engineNameTimestamptz})
	require.NotNil(t, stateAtInt)
	assert.Equal(t, engineNameBigint, stateAtInt.ReturnType.EngineName)

	interpolatedTimeline := findOverload(catalogue, "interpolated_state_timeline", []string{
		engineNameStateSummary, engineNameTimestamptz, engineNameInterval,
		engineNameStateSummary, engineNameStateSummary,
	})
	require.NotNil(t, interpolatedTimeline)
	assert.True(t, interpolatedTimeline.ReturnsSet)

	interpolatedPeriods := findOverload(catalogue, "interpolated_state_periods", []string{
		engineNameStateSummary, engineNameText, engineNameTimestamptz, engineNameInterval,
		engineNameStateSummary, engineNameStateSummary,
	})
	require.NotNil(t, interpolatedPeriods)
	assert.True(t, interpolatedPeriods.ReturnsSet)

	interpolatedDuration := findOverload(catalogue, "interpolated_duration_in", []string{
		engineNameStateSummary, engineNameText, engineNameTimestamptz, engineNameInterval,
		engineNameStateSummary, engineNameStateSummary,
	})
	require.NotNil(t, interpolatedDuration)
	assert.Equal(t, engineNameInterval, interpolatedDuration.ReturnType.EngineName)
}

func TestRollupHasTimeWeightAndCompactAndHeartbeat(t *testing.T) {
	t.Parallel()

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	timeWeight := findOverload(catalogue, "rollup", []string{engineNameTimeWeightSum})
	require.NotNil(t, timeWeight, "rollup(time_weight_summary) must be registered")

	compact := findOverload(catalogue, "rollup", []string{engineNameCompactStateAgg})
	require.NotNil(t, compact, "rollup(compact_state_agg) must be registered")

	heartbeat := findOverload(catalogue, "rollup", []string{engineNameHeartbeat})
	require.NotNil(t, heartbeat, "rollup(heartbeat) must be registered")
}

func TestTimeWeightSummaryTypeRegistered(t *testing.T) {
	t.Parallel()

	types := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinTypes()

	timeWeightType, exists := types.Types[engineNameTimeWeightSum]
	require.True(t, exists, "time_weight_summary opaque type must be registered")
	assert.Equal(t, querier_dto.TypeCategoryUnknown, timeWeightType.Category)
	assert.Equal(t, engineNameTimeWeightSum, timeWeightType.EngineName)

	compactType, exists := types.Types[engineNameCompactStateAgg]
	require.True(t, exists, "compact_state_agg opaque type must be registered")
	assert.Equal(t, querier_dto.TypeCategoryUnknown, compactType.Category)
}

func TestParseAndResolveTimeWeight(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(
		"SELECT average(time_weight('Linear', ts, value)) FROM readings",
	)
	require.NoError(t, err)
	require.Len(t, statements, 1)

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	timeWeight := findOverload(catalogue, "time_weight",
		[]string{engineNameText, engineNameTimestamptz, engineNameFloat8})
	require.NotNil(t, timeWeight)

	avg := findOverload(catalogue, "average", []string{engineNameTimeWeightSum})
	require.NotNil(t, avg)
}

func TestParseAndResolveHeartbeat(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(
		"SELECT num_live_ranges(heartbeat_agg(ts, $1, $2, $3)) FROM pings",
	)
	require.NoError(t, err)
	require.Len(t, statements, 1)

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	hb := findOverload(catalogue, "heartbeat_agg",
		[]string{engineNameTimestamptz, engineNameTimestamptz, engineNameInterval, engineNameInterval})
	require.NotNil(t, hb)
	assert.True(t, hb.IsAggregate)

	numLive := findOverload(catalogue, "num_live_ranges", []string{engineNameHeartbeat})
	require.NotNil(t, numLive)
}

func TestParseAndResolveStateAgg(t *testing.T) {
	t.Parallel()

	engine := db_engine_timescaledb.NewTimescaleDBEngine()
	statements, err := engine.ParseStatements(
		"SELECT duration_in(state_agg(ts, status), 'ok') FROM events",
	)
	require.NoError(t, err)
	require.Len(t, statements, 1)

	catalogue := db_engine_timescaledb.NewTimescaleDBEngine().BuiltinFunctions()

	textAgg := findOverload(catalogue, "state_agg", []string{engineNameTimestamptz, engineNameText})
	require.NotNil(t, textAgg)
	assert.True(t, textAgg.IsAggregate)

	bigintAgg := findOverload(catalogue, "state_agg", []string{engineNameTimestamptz, engineNameBigint})
	require.NotNil(t, bigintAgg, "bigint state_agg overload must resolve")
}
