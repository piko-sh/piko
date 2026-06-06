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

package db_engine_timescaledb

import (
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

const (
	// funcNameTimeBucket is the canonical name of the time_bucket family of bucketing
	// aggregates.
	funcNameTimeBucket = "time_bucket"

	// funcNameTimeBucketGapfill is the canonical name of the time_bucket_gapfill family; the
	// helper has four overloads sharing the same name.
	funcNameTimeBucketGapfill = "time_bucket_gapfill"

	// funcNameInterpolate is the locf/interpolate gap-fill helper name.
	funcNameInterpolate = "interpolate"

	// funcNameRollup is the toolkit aggregate that merges per-bucket summary states across
	// continuous-aggregate buckets.
	funcNameRollup = "rollup"
)

const (
	// paramNameValue is the canonical parameter name for an aggregate or gapfill input
	// value.
	paramNameValue = "value"

	// paramNamePrev is the canonical parameter name for the previous value in the gap-fill
	// family.
	paramNamePrev = "prev"

	// paramNameTime is the canonical parameter name for a timestamptz time column in
	// first/last and other time-series helpers.
	paramNameTime = "time"

	// paramNameTS is the canonical parameter name for a timestamp argument in bucketing and
	// aggregate calls.
	paramNameTS = "ts"

	// paramNameBucketWidth is the canonical parameter name for the bucket interval in the
	// time_bucket family.
	paramNameBucketWidth = "bucket_width"

	// paramNameSummary is the canonical parameter name for an opaque summary-state argument.
	paramNameSummary = "summary"

	// paramNameHypertable is the canonical parameter name for the hypertable target argument
	// in TimescaleDB management functions.
	paramNameHypertable = "hypertable"

	// paramNameChunk is the canonical parameter name for a chunk argument in size and
	// reorder helpers.
	paramNameChunk = "chunk"

	// paramNameStart is the canonical parameter name for the inclusive start of a window in
	// time-series helpers.
	paramNameStart = "start"

	// paramNameMethod is the canonical parameter name for the method selector text accepted
	// by stats and extrapolation overloads.
	paramNameMethod = "method"

	// paramNameState is the canonical parameter name for the state argument of state_agg and
	// its accessors.
	paramNameState = "state"

	// paramNameSketch is the canonical parameter name for a sketch (tdigest or uddsketch)
	// argument in percentile-rank accessors.
	paramNameSketch = "sketch"

	// paramNameNext is the canonical parameter name for the next bucket's state or value in
	// continuous-aggregate interpolation accessors and the 3-arg interpolate gap-fill
	// helper.
	paramNameNext = "next"

	// paramNameInterval is the canonical parameter name for the bucket width interval
	// supplied to interpolation accessors.
	paramNameInterval = "interval"

	// paramNameTimezone is the canonical parameter name for the timezone text argument
	// shared by the time_bucket family, retention/continuous-aggregate policy options, and
	// add_job.
	paramNameTimezone = "timezone"

	// paramNameInitialStart is the canonical parameter name for the timestamptz
	// initial_start argument shared by add_job, add_retention_policy,
	// add_continuous_aggregate_policy, and add_reorder_policy.
	paramNameInitialStart = "initial_start"

	// paramNameScheduleInterval is the canonical parameter name for the interval
	// schedule_interval argument shared by the policy family (add_compression_policy,
	// add_retention_policy, add_continuous_aggregate_policy, add_job).
	paramNameScheduleInterval = "schedule_interval"

	// paramNameCount is the canonical parameter name for the integer count argument shared
	// by the frequency analysis aggregates (mcv_agg).
	paramNameCount = "count"

	// paramNameAgg is the canonical parameter name for an aggregate state passed to an
	// accessor (topn, into_values, approx_count).
	paramNameAgg = "agg"

	// paramNameN is the canonical parameter name for the integer top-N count shared by topn
	// and the min_n / max_n family.
	paramNameN = "n"
)

const (
	// funcNameMCVAgg is the canonical name of the most-common-value aggregate exposed by the
	// frequency analysis family.
	funcNameMCVAgg = "mcv_agg"

	// funcNameIntoValues is the canonical name of the projection accessor that returns the
	// captured values from a space-saving or min-n aggregate state.
	funcNameIntoValues = "into_values"
)

const (
	// typeNameCompressionStatsRecord is the engine name of the composite record returned by
	// hypertable_compression_stats, chunk_compression_stats, hypertable_columnstore_stats,
	// and chunk_columnstore_stats.
	typeNameCompressionStatsRecord = "compression_stats_record"
)

const (
	// typeNameStatsSummary1D is the engine name of the 1-D stats summary aggregate state. It
	// is also registered with the engine type catalogue (see types.go) so the analyser
	// recognises it as a known result type.
	typeNameStatsSummary1D = "statssummary1d"

	// typeNameStatsSummary2D is the engine name of the 2-D stats summary aggregate state.
	typeNameStatsSummary2D = "statssummary2d"

	// typeNameCounterSummary is the engine name of the counter_agg aggregate state.
	typeNameCounterSummary = "counter_summary"

	// typeNameGaugeSummary is the engine name of the gauge_agg aggregate state.
	typeNameGaugeSummary = "gauge_summary"

	// typeNameCandlestick is the engine name of the candlestick_agg aggregate state.
	typeNameCandlestick = "candlestick"

	// typeNameStateSummary is the engine name of the state_agg aggregate state.
	typeNameStateSummary = "state_summary"

	// typeNameCompactStateSummary is the engine name of the compact_state_agg aggregate
	// state, the run-length-compressed counterpart of state_agg.
	typeNameCompactStateSummary = "compact_state_agg"

	// typeNameHyperloglog is the engine name of the hyperloglog aggregate state.
	typeNameHyperloglog = "hyperloglog"

	// typeNameTDigest is the engine name of the tdigest aggregate state.
	typeNameTDigest = "tdigest"

	// typeNameUDDSketch is the engine name of the uddsketch aggregate state.
	typeNameUDDSketch = "uddsketch"

	// typeNameTimeWeightSummary is the engine name of the time_weight aggregate state, used
	// by the time-weighted family of accessors.
	typeNameTimeWeightSummary = "time_weight_summary"

	// typeNameHeartbeat is the engine name of the heartbeat_agg aggregate state used by the
	// live/dead range accessors.
	typeNameHeartbeat = "heartbeat"

	// typeNameTstzRange is the engine name of the timestamptz range type accepted by the
	// with_bounds counter-summary helper.
	typeNameTstzRange = "tstzrange"

	// typeNameSpaceSavingAggregate is the engine name of the polymorphic
	// space_saving_aggregate state used by raw_freq_agg and raw_mcv_agg.
	typeNameSpaceSavingAggregate = "space_saving_aggregate"

	// typeNameSpaceSavingBigintAggregate is the engine name of the bigint-valued
	// space-saving state produced by freq_agg and mcv_agg over a bigint value column.
	typeNameSpaceSavingBigintAggregate = "space_saving_bigint_aggregate"

	// typeNameSpaceSavingTextAggregate is the engine name of the text-valued space-saving
	// state produced by freq_agg and mcv_agg over a text value column.
	typeNameSpaceSavingTextAggregate = "space_saving_text_aggregate"

	// typeNameMinNState is the engine name of the polymorphic min_n/max_n aggregate state
	// used by the corresponding constructors and accessors.
	typeNameMinNState = "min_n_state"

	// typeNameMinNByState is the engine name of the by-clause min_n_by/max_n_by aggregate
	// state that retains a paired secondary value for each captured observation.
	typeNameMinNByState = "min_n_by_state"

	// typeNameTimevectorTstzF64 is the engine name of the timestamptz/float8 timevector
	// summary used by lttb, asap_smooth, and the raw timevector aggregate.
	typeNameTimevectorTstzF64 = "timevector_tstz_f64"

	// typeNameCountMinSketch is the engine name of the experimental count_min_sketch
	// aggregate state.
	typeNameCountMinSketch = "count_min_sketch"
)

var (
	// stateSummaryStateNames lists the engine names of the state aggregate states.
	// compact_state_agg shares the same accessor names as state_agg but uses its own opaque
	// value type, so collecting both names in one slice keeps the per-state accessor
	// registrations aligned.
	stateSummaryStateNames = []string{
		typeNameStateSummary,
		typeNameCompactStateSummary,
	}
)

// registerTimescaleDBFunctions registers TimescaleDB-specific functions with the postgres
// function-catalogue builder. Categories are split into helper functions for readability;
// each helper is a thin sequence of builder calls.
//
// Takes builder (*db_engine_postgres.FunctionCatalogueBuilder) which receives the
// registered TimescaleDB signatures.
func registerTimescaleDBFunctions(builder *db_engine_postgres.FunctionCatalogueBuilder) {
	registerTimeBucketFamily(builder)
	registerTimeBucketIntegerFamily(builder)
	registerTimeBucketUUIDFamily(builder)
	registerGapfillFamily(builder)
	registerFirstLastFamily(builder)
	registerStatsFamily(builder)
	registerStatsAccessorFamily(builder)
	registerStats2DAccessorFamily(builder)
	registerStatsMethodOverloadFamily(builder)
	registerCounterGaugeAccessorFamily(builder)
	registerHyperfunctionsFamily(builder)
	registerCandlestickAccessorFamily(builder)
	registerPercentileAccessorFamily(builder)
	registerTimeWeightFamily(builder)
	registerHeartbeatAccessorFamily(builder)
	registerStateAccessorFamily(builder)
	registerPolicyFamily(builder)
	registerHypertableManagementFamily(builder)
	registerHypercoreFamily(builder)
	registerMetadataFamily(builder)
	registerRollupFamily(builder)
	registerFrequencyAnalysisFamily(builder)
	registerMinMaxNFamily(builder)
	registerTimevectorFamily(builder)
	registerCountMinSketchFamily(builder)
	registerSaturatingMathFamily(builder)
	registerExtensionLifecycleFamily(builder)
	markDataModifyingFunctions(builder)
}

// opaqueType returns an unknown-category SQLType with the given engine name. Used for
// TimescaleDB aggregate-state types that do not have a direct Go representation;
// downstream codegen emits these as `interface{}` unless a piko.column override is
// supplied.
//
// Takes name (string) which is the engine type name to attach.
//
// Returns querier_dto.SQLType which is the opaque unknown-category type.
func opaqueType(name string) querier_dto.SQLType {
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: name}
}

// tstzRangeType returns the SQLType for PostgreSQL's tstzrange. Unlike the opaque
// aggregate-state types it is a real built-in range type, so it is modelled with
// TypeCategoryRange rather than the unknown category, preserving range semantics for
// downstream resolution.
//
// Returns querier_dto.SQLType which is the tstzrange range type.
func tstzRangeType() querier_dto.SQLType {
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryRange, EngineName: typeNameTstzRange}
}

// arrayOf wraps a base type in an Array(T) shape. Used by histogram which returns an
// integer array of bucket counts.
//
// Takes element (querier_dto.SQLType) which is the array element type.
//
// Returns querier_dto.SQLType which is the array type over element.
func arrayOf(element querier_dto.SQLType) querier_dto.SQLType {
	return querier_dto.SQLType{
		Category:    querier_dto.TypeCategoryArray,
		EngineName:  element.EngineName + "[]",
		ElementType: new(element),
	}
}

// regclassType is the regclass SQLType used for arguments that reference a relation by
// name in TimescaleDB management functions. The canonical call form passes a string
// literal (hypertable_size('conditions')), so the category is Text (keeping the regclass
// EngineName); an Integer category would fail overload resolution because Text->Integer
// is not an implicit cast.
//
// Returns querier_dto.SQLType which is the regclass type with the Text category.
func regclassType() querier_dto.SQLType {
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "regclass"}
}

// regprocType is the regproc SQLType used for arguments that reference a procedure or
// function by name (add_job's proc and check_config arguments).
//
// Like regclass it is an oid alias whose canonical call form is a string literal, so it
// carries the Text category (keeping the regproc EngineName) to resolve from those call
// sites.
//
// Returns querier_dto.SQLType which is the regproc type with the Text category.
func regprocType() querier_dto.SQLType {
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "regproc"}
}

// nameType is the name SQLType used for the tablespace and chunk destination arguments on
// move_chunk and attach/detach_tablespace. Postgres `name` is a fixed-length identifier
// string distinct from the variable-length text type, so we register it with its
// dedicated engine name rather than collapsing into Text.
//
// Returns querier_dto.SQLType which is the name type with the Text category.
func nameType() querier_dto.SQLType {
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "name"}
}

// voidType is the void SQLType used for procedures that return no value
// (refresh_continuous_aggregate, reorder_chunk, etc.).
//
// Returns querier_dto.SQLType which is the void type.
func voidType() querier_dto.SQLType {
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: "void"}
}

// addAggregate registers an aggregate function. NullableBehaviour is CalledOnNull because
// aggregates run their state-transition function for every input row including those with
// NULL inputs.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signature.
// Takes name (string) which is the aggregate function name.
// Takes args ([]querier_dto.FunctionArgument) which are the argument definitions.
// Takes returnType (querier_dto.SQLType) which is the aggregate result type.
func addAggregate(b *db_engine_postgres.FunctionCatalogueBuilder, name string, args []querier_dto.FunctionArgument, returnType querier_dto.SQLType) {
	b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         args,
		ReturnType:        returnType,
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// addReturnsSet registers a function whose return type is a set of the declared element
// type (used for show_chunks, drop_chunks).
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// signature.
// Takes name (string) which is the function name.
// Takes args ([]querier_dto.FunctionArgument) which are the argument definitions.
// Takes returnType (querier_dto.SQLType) which is the set element type.
//
// Returns *querier_dto.FunctionSignature which is the registered signature.
func addReturnsSet(b *db_engine_postgres.FunctionCatalogueBuilder, name string, args []querier_dto.FunctionArgument, returnType querier_dto.SQLType) *querier_dto.FunctionSignature {
	return b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         args,
		ReturnType:        returnType,
		ReturnsSet:        true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

// registerTimeBucketFamily covers time_bucket and its variants across the common
// (interval, timestamptz), (interval, timestamptz, ...) overloads including timezone and
// origin shapes.
//
// The deprecated time_bucket_ng helper is deliberately omitted: it was removed from
// TimescaleDB in 2.18 and its presence in the catalogue only encouraged the resolver to
// bind a no-longer-extant identifier.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// time_bucket overloads.
func registerTimeBucketFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull(funcNameTimeBucket, b.Args(db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval}, db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz}), b.Timestamptz)
	b.NullOnNull(funcNameTimeBucket, b.Args(db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval}, db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamp}), b.Timestamp)
	b.NullOnNull(funcNameTimeBucket, b.Args(db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval}, db_engine_postgres.Arg{Name: paramNameTS, Type: b.Date}), b.Date)
	b.NullOnNull(funcNameTimeBucket,
		b.Args(
			db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: "offset", Type: b.Interval},
		),
		b.Timestamptz,
	)
	b.NullOnNull(funcNameTimeBucket,
		b.Args(
			db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: "origin", Type: b.Timestamptz},
		),
		b.Timestamptz,
	)
	b.NullOnNull(funcNameTimeBucket,
		b.Args(
			db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameTimezone, Type: b.Text},
		),
		b.Timestamptz,
	)
}

// registerTimeBucketIntegerFamily covers time_bucket overloads where the timestamp column
// is an integer (epoch seconds, milliseconds, or nanoseconds).
//
// Each overload mirrors the upstream signature with a base (width, ts) form and
// offset/origin extensions on the same integer type. Smallint, integer and bigint widths
// are all registered so the resolver can bind metrics-table queries that store time as
// int2, int4, or int8 epoch values.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// integer-width overloads.
func registerTimeBucketIntegerFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	registerTimeBucketIntegerOverloads(b, b.Smallint)
	registerTimeBucketIntegerOverloads(b, b.Integer)
	registerTimeBucketIntegerOverloads(b, b.Bigint)
}

// registerTimeBucketIntegerOverloads registers the four integer-width time_bucket
// variants for a single integer type: the base form, the offset form, and the origin
// form. The bucket_width parameter uses the same integer type as the timestamp column
// because TimescaleDB requires the bucket width to share its integer width with the input
// column.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// overloads.
// Takes integerType (querier_dto.SQLType) which is the shared integer width.
func registerTimeBucketIntegerOverloads(b *db_engine_postgres.FunctionCatalogueBuilder, integerType querier_dto.SQLType) {
	b.NullOnNull(funcNameTimeBucket,
		b.Args(db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: integerType}, db_engine_postgres.Arg{Name: paramNameTS, Type: integerType}),
		integerType,
	)
	b.NullOnNull(funcNameTimeBucket,
		b.Args(
			db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: integerType},
			db_engine_postgres.Arg{Name: paramNameTS, Type: integerType},
			db_engine_postgres.Arg{Name: "offset", Type: integerType},
		),
		integerType,
	)
	b.NullOnNull(funcNameTimeBucket,
		b.Args(
			db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: integerType},
			db_engine_postgres.Arg{Name: paramNameTS, Type: integerType},
			db_engine_postgres.Arg{Name: "origin", Type: integerType},
		),
		integerType,
	)
}

// registerTimeBucketUUIDFamily registers time_bucket overloads that accept a UUID v7
// column as the timestamp source.
//
// UUIDv7 embeds a millisecond-resolution timestamp in its prefix; TimescaleDB exposes
// time_bucket(interval, uuid) to bucket on that prefix without the caller needing to
// extract the timestamp explicitly. Each overload mirrors the timestamptz family so
// timezone, origin, and offset shapes are all bindable against a uuid time column.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// UUID overloads.
func registerTimeBucketUUIDFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull(funcNameTimeBucket,
		b.Args(db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval}, db_engine_postgres.Arg{Name: paramNameTS, Type: b.UUID}),
		b.Timestamptz,
	)
	b.NullOnNull(funcNameTimeBucket,
		b.Args(
			db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.UUID},
			db_engine_postgres.Arg{Name: paramNameTimezone, Type: b.Text},
		),
		b.Timestamptz,
	)
	b.NullOnNull(funcNameTimeBucket,
		b.Args(
			db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.UUID},
			db_engine_postgres.Arg{Name: "origin", Type: b.Timestamptz},
		),
		b.Timestamptz,
	)
	b.NullOnNull(funcNameTimeBucket,
		b.Args(db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval}, db_engine_postgres.Arg{Name: paramNameTS, Type: b.UUID}, db_engine_postgres.Arg{Name: "offset", Type: b.Interval}),
		b.Timestamptz,
	)
}

// registerGapfillFamily covers time_bucket_gapfill (2-, 3-, 4-, and 5-arg) and the locf /
// interpolate / heartbeat_agg helpers. The 5-arg form accepts a text timezone between the
// bucket and the start of the window.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// gapfill signatures.
func registerGapfillFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull(funcNameTimeBucketGapfill,
		b.Args(db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval}, db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz}),
		b.Timestamptz,
	)
	b.NullOnNull(funcNameTimeBucketGapfill,
		b.Args(
			db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameStart, Type: b.Timestamptz},
		),
		b.Timestamptz,
	)
	b.NullOnNull(funcNameTimeBucketGapfill,
		b.Args(
			db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: "finish", Type: b.Timestamptz},
		),
		b.Timestamptz,
	)
	b.NullOnNull(funcNameTimeBucketGapfill,
		b.Args(
			db_engine_postgres.Arg{Name: paramNameBucketWidth, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameTimezone, Type: b.Text},
			db_engine_postgres.Arg{Name: paramNameStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: "finish", Type: b.Timestamptz},
		),
		b.Timestamptz,
	)
	b.CalledOnNull("locf", b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any}), b.Any)
	b.CalledOnNull("locf", b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any}, db_engine_postgres.Arg{Name: paramNamePrev, Type: b.Any}), b.Any)
	b.CalledOnNull("locf", b.Args(
		db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any},
		db_engine_postgres.Arg{Name: paramNamePrev, Type: b.Any},
		db_engine_postgres.Arg{Name: "treat_null_as_missing", Type: b.Boolean},
	), b.Any)
	b.CalledOnNull(funcNameInterpolate, b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any}), b.Any)
	b.CalledOnNull(funcNameInterpolate, b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any}, db_engine_postgres.Arg{Name: paramNamePrev, Type: b.Any}), b.Any)
	b.CalledOnNull(funcNameInterpolate, b.Args(
		db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any},
		db_engine_postgres.Arg{Name: paramNamePrev, Type: b.Any},
		db_engine_postgres.Arg{Name: paramNameNext, Type: b.Any},
	), b.Any)
	addAggregate(b, "heartbeat_agg",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: "agg_interval", Type: b.Interval},
			db_engine_postgres.Arg{Name: "heartbeat_interval", Type: b.Interval},
		),
		opaqueType(typeNameHeartbeat),
	)
}

// registerFirstLastFamily registers the polymorphic first/last aggregates plus a couple
// of concrete overloads to help the catalogue resolver pick a return type quickly for
// common cases.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// first/last signatures.
func registerFirstLastFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	addAggregate(b, "first", b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any}, db_engine_postgres.Arg{Name: paramNameTime, Type: b.Timestamptz}), b.Any)
	addAggregate(b, "last", b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any}, db_engine_postgres.Arg{Name: paramNameTime, Type: b.Timestamptz}), b.Any)
	addAggregate(b, "first", b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Numeric}, db_engine_postgres.Arg{Name: paramNameTime, Type: b.Timestamptz}), b.Numeric)
	addAggregate(b, "last", b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Numeric}, db_engine_postgres.Arg{Name: paramNameTime, Type: b.Timestamptz}), b.Numeric)
	addAggregate(b, "first", b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Text}, db_engine_postgres.Arg{Name: paramNameTime, Type: b.Timestamptz}), b.Text)
	addAggregate(b, "last", b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Text}, db_engine_postgres.Arg{Name: paramNameTime, Type: b.Timestamptz}), b.Text)
}

// registerStatsFamily registers histogram and the stats/counter/gauge aggregate state
// constructors.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// stats constructors.
func registerStatsFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	addAggregate(b, "histogram",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8},
			db_engine_postgres.Arg{Name: "min", Type: b.Float8},
			db_engine_postgres.Arg{Name: "max", Type: b.Float8},
			db_engine_postgres.Arg{Name: "nbuckets", Type: b.Integer},
		),
		arrayOf(b.Integer),
	)
	addAggregate(b, "stats_agg", b.Args(db_engine_postgres.Arg{Name: "y", Type: b.Float8}), opaqueType(typeNameStatsSummary1D))
	addAggregate(b, "stats_agg", b.Args(db_engine_postgres.Arg{Name: "y", Type: b.Float8}, db_engine_postgres.Arg{Name: "x", Type: b.Float8}), opaqueType(typeNameStatsSummary2D))
	addAggregate(b, "counter_agg", b.Args(
		db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
		db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8},
	), opaqueType(typeNameCounterSummary))
	addAggregate(b, "gauge_agg", b.Args(db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8}), opaqueType(typeNameGaugeSummary))
}

// registerStatsAccessorFamily registers the scalar accessors that extract concrete
// numeric values from statssummary states.
//
// The 2-D correlation is registered as `corr` (the toolkit canonical name); the older
// `correlation` alias was dropped because TimescaleDB 2.x renamed it and keeping a
// phantom registration only confused the resolver into binding queries that PostgreSQL
// would reject.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// scalar accessors.
func registerStatsAccessorFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull("average", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary1D)}), b.Float8)
	b.NullOnNull("mean", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary1D)}), b.Float8)
	b.NullOnNull("sum", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary1D)}), b.Float8)
	b.NullOnNull("num_vals", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary1D)}), b.Bigint)
	b.NullOnNull("stddev", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary1D)}), b.Float8)
	b.NullOnNull("variance", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary1D)}), b.Float8)
	b.NullOnNull("skewness", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary1D)}), b.Float8)
	b.NullOnNull("kurtosis", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary1D)}), b.Float8)
	b.NullOnNull("corr", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("slope", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("intercept", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("x_intercept", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("determination_coeff", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
}

// registerStats2DAccessorFamily registers the per-axis scalar accessors on the 2-D stats
// summary. Each name has both an X and Y variant covering the projected moments and
// totals.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// per-axis accessors.
func registerStats2DAccessorFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull("average_x", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("average_y", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("sum_x", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("sum_y", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("stddev_x", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("stddev_y", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("variance_x", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("variance_y", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("skewness_x", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("skewness_y", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("kurtosis_x", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
	b.NullOnNull("kurtosis_y", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), b.Float8)
}

// registerStatsMethodOverloadFamily registers the method-text variants of the stats
// accessors.
//
// TimescaleDB exposes a parallel surface where the caller picks between population and
// sample estimators by passing a text method argument. Both 1-D and 2-D states are
// accepted.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// method-text accessors.
func registerStatsMethodOverloadFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	registerStatsMethodOverload(b, "stddev")
	registerStatsMethodOverload(b, "variance")
	registerStatsMethodOverload(b, "skewness")
	registerStatsMethodOverload(b, "kurtosis")
	b.NullOnNull("covariance",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}, db_engine_postgres.Arg{Name: paramNameMethod, Type: b.Text}),
		b.Float8,
	)
}

// registerStatsMethodOverload registers a single (summary, method) accessor for both the
// 1-D and 2-D stats summary states.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// accessors.
// Takes name (string) which is the accessor function name.
func registerStatsMethodOverload(b *db_engine_postgres.FunctionCatalogueBuilder, name string) {
	b.NullOnNull(name,
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary1D)}, db_engine_postgres.Arg{Name: paramNameMethod, Type: b.Text}),
		b.Float8,
	)
	b.NullOnNull(name,
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}, db_engine_postgres.Arg{Name: paramNameMethod, Type: b.Text}),
		b.Float8,
	)
}

// registerCounterGaugeAccessorFamily registers the accessor functions for counter and
// gauge summary states.
//
// time_delta returns float8 seconds because the upstream toolkit exposes the elapsed time
// as a numeric quantity rather than an interval; the catalogue previously over-promised
// this as interval, which caused codegen to emit a time.Duration where a float was
// expected.
//
// rate has a single 1-arg signature: the 2-arg (summary, interval) form does not exist
// upstream and was confusing the resolver.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// counter and gauge accessors.
func registerCounterGaugeAccessorFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull("delta", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Float8)
	b.NullOnNull("delta", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameGaugeSummary)}), b.Float8)
	b.NullOnNull("rate", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Float8)
	b.NullOnNull("idelta_left", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Float8)
	b.NullOnNull("idelta_right", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Float8)
	b.NullOnNull("irate_left", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Float8)
	b.NullOnNull("irate_right", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Float8)
	b.NullOnNull("time_delta", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Float8)

	registerCounterChangeAccessors(b)
	registerCounterEndpointAccessors(b)
	registerCounterRegressionAccessors(b)
	registerCounterExtrapolationAccessors(b)

	b.NullOnNull("with_bounds",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}, db_engine_postgres.Arg{Name: "bounds", Type: tstzRangeType()}),
		opaqueType(typeNameCounterSummary),
	)

	addAggregate(b, "counter_agg",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8},
			db_engine_postgres.Arg{Name: "bounds", Type: tstzRangeType()},
		),
		opaqueType(typeNameCounterSummary),
	)
}

// registerCounterChangeAccessors registers the counter-summary scalar accessors that
// count internal events: changes, resets, and the number of observed elements.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// change-count accessors.
func registerCounterChangeAccessors(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull("num_changes", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Bigint)
	b.NullOnNull("num_resets", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Bigint)
	b.NullOnNull("num_elements", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Bigint)
}

// registerCounterEndpointAccessors registers the first/last value and first/last time
// accessors on a counter summary. These differ from the polymorphic first/last aggregates
// because they read directly from the captured state.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// endpoint accessors.
//
//nolint:dupl // structurally similar counter-summary accessors
func registerCounterEndpointAccessors(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull("first_val", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Float8)
	b.NullOnNull("last_val", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Float8)
	b.NullOnNull("first_time", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Timestamptz)
	b.NullOnNull("last_time", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Timestamptz)
}

// registerCounterRegressionAccessors registers the linear-regression scalar accessors on
// a counter summary: slope, intercept, corr, and counter_zero_time.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// regression accessors.
//
//nolint:dupl // structurally similar counter-summary accessors
func registerCounterRegressionAccessors(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull("slope", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Float8)
	b.NullOnNull("intercept", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Float8)
	b.NullOnNull("corr", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Float8)
	b.NullOnNull("counter_zero_time", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), b.Timestamptz)
}

// registerCounterExtrapolationAccessors registers the extrapolated delta and rate
// accessors that take a text method argument to choose between the prometheus and
// last-value extrapolation strategies.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// extrapolation accessors.
func registerCounterExtrapolationAccessors(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull("extrapolated_delta",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}, db_engine_postgres.Arg{Name: paramNameMethod, Type: b.Text}),
		b.Float8,
	)
	b.NullOnNull("extrapolated_rate",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}, db_engine_postgres.Arg{Name: paramNameMethod, Type: b.Text}),
		b.Float8,
	)
}

// registerHyperfunctionsFamily registers the broader hyperfunctions toolkit. Each
// aggregate produces an opaque state.
//
// The phantom topk(k, value) entry was removed: TimescaleDB does not expose a `topk`
// aggregate, and keeping the registration only made the resolver bind a name that the
// database would reject.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// hyperfunction signatures.
func registerHyperfunctionsFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	addAggregate(b, "candlestick_agg",
		b.Args(db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz}, db_engine_postgres.Arg{Name: "price", Type: b.Float8}, db_engine_postgres.Arg{Name: "volume", Type: b.Float8}),
		opaqueType(typeNameCandlestick),
	)

	b.NullOnNull("candlestick",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: "open", Type: b.Float8},
			db_engine_postgres.Arg{Name: "high", Type: b.Float8},
			db_engine_postgres.Arg{Name: "low", Type: b.Float8},
			db_engine_postgres.Arg{Name: "close", Type: b.Float8},
			db_engine_postgres.Arg{Name: "volume", Type: b.Float8},
		),
		opaqueType(typeNameCandlestick),
	)

	addAggregate(b, "state_agg", b.Args(db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz}, db_engine_postgres.Arg{Name: paramNameState, Type: b.Text}), opaqueType(typeNameStateSummary))
	addAggregate(b, "state_agg", b.Args(db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz}, db_engine_postgres.Arg{Name: paramNameState, Type: b.Bigint}), opaqueType(typeNameStateSummary))
	addAggregate(b, "compact_state_agg",
		b.Args(db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz}, db_engine_postgres.Arg{Name: paramNameState, Type: b.Text}),
		opaqueType(typeNameCompactStateSummary),
	)
	addAggregate(b, "compact_state_agg",
		b.Args(db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz}, db_engine_postgres.Arg{Name: paramNameState, Type: b.Bigint}),
		opaqueType(typeNameCompactStateSummary),
	)
	addAggregate(b, typeNameHyperloglog, b.Args(db_engine_postgres.Arg{Name: "precision", Type: b.Integer}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any}), opaqueType(typeNameHyperloglog))
	b.NullOnNull("approx_count_distinct", b.Args(db_engine_postgres.Arg{Name: "hll", Type: opaqueType(typeNameHyperloglog)}), b.Bigint)
	addAggregate(b, typeNameTDigest, b.Args(db_engine_postgres.Arg{Name: "buckets", Type: b.Integer}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8}), opaqueType(typeNameTDigest))
	b.NullOnNull("approx_percentile", b.Args(db_engine_postgres.Arg{Name: "percentile", Type: b.Float8}, db_engine_postgres.Arg{Name: "digest", Type: opaqueType(typeNameTDigest)}), b.Float8)
	addAggregate(b, typeNameUDDSketch, b.Args(
		db_engine_postgres.Arg{Name: "buckets", Type: b.Integer},
		db_engine_postgres.Arg{Name: "alpha", Type: b.Float8},
		db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8},
	), opaqueType(typeNameUDDSketch))
	b.NullOnNull("approx_percentile", b.Args(db_engine_postgres.Arg{Name: "percentile", Type: b.Float8}, db_engine_postgres.Arg{Name: paramNameSketch, Type: opaqueType(typeNameUDDSketch)}), b.Float8)
	b.NullOnNull("error", b.Args(db_engine_postgres.Arg{Name: paramNameSketch, Type: opaqueType(typeNameUDDSketch)}), b.Float8)

	addAggregate(b, "percentile_agg",
		b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8}),
		opaqueType(typeNameUDDSketch),
	)
}

// registerCandlestickAccessorFamily registers accessor functions on the candlestick
// aggregate state. These produce the conventional open / high / low / close / volume /
// vwap scalar values, plus the timestamps of the open, close, high, and low observations.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// candlestick accessors.
func registerCandlestickAccessorFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull("open", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCandlestick)}), b.Float8)
	b.NullOnNull("high", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCandlestick)}), b.Float8)
	b.NullOnNull("low", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCandlestick)}), b.Float8)
	b.NullOnNull("close", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCandlestick)}), b.Float8)
	b.NullOnNull("open_time", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCandlestick)}), b.Timestamptz)
	b.NullOnNull("close_time", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCandlestick)}), b.Timestamptz)
	b.NullOnNull("high_time", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCandlestick)}), b.Timestamptz)
	b.NullOnNull("low_time", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCandlestick)}), b.Timestamptz)
	b.NullOnNull("volume", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCandlestick)}), b.Float8)
	b.NullOnNull("vwap", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCandlestick)}), b.Float8)
}

// registerPercentileAccessorFamily registers the scalar accessors that extract concrete
// values from tdigest and uddsketch states. approx_percentile_rank gives the
// cumulative-distribution value for a given observation; mean, num_vals, min_val, and
// max_val pull directly recorded fields from the sketch state.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// percentile accessors.
func registerPercentileAccessorFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull("approx_percentile_rank",
		b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8}, db_engine_postgres.Arg{Name: paramNameSketch, Type: opaqueType(typeNameTDigest)}),
		b.Float8,
	)
	b.NullOnNull("approx_percentile_rank",
		b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8}, db_engine_postgres.Arg{Name: paramNameSketch, Type: opaqueType(typeNameUDDSketch)}),
		b.Float8,
	)

	b.NullOnNull("mean", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTDigest)}), b.Float8)
	b.NullOnNull("mean", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameUDDSketch)}), b.Float8)

	b.NullOnNull("num_vals", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTDigest)}), b.Bigint)
	b.NullOnNull("num_vals", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameUDDSketch)}), b.Bigint)

	b.NullOnNull("min_val", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTDigest)}), b.Float8)
	b.NullOnNull("max_val", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTDigest)}), b.Float8)
}

// registerTimeWeightFamily registers the time_weight constructor and its full accessor
// surface. Time-weighted aggregates compute the average and integral of a series where
// each observation contributes proportional to its dwell time, optionally interpolated
// across continuous-aggregate buckets.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// time_weight signatures.
func registerTimeWeightFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	addAggregate(b, "time_weight",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameMethod, Type: b.Text},
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8},
		),
		opaqueType(typeNameTimeWeightSummary),
	)

	b.NullOnNull("average", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTimeWeightSummary)}), b.Float8)
	b.NullOnNull("integral",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTimeWeightSummary)}),
		b.Float8,
	)
	b.NullOnNull("integral",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTimeWeightSummary)}, db_engine_postgres.Arg{Name: "unit", Type: b.Text}),
		b.Float8,
	)

	b.NullOnNull("first_val", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTimeWeightSummary)}), b.Float8)
	b.NullOnNull("last_val", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTimeWeightSummary)}), b.Float8)
	b.NullOnNull("first_time", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTimeWeightSummary)}), b.Timestamptz)
	b.NullOnNull("last_time", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTimeWeightSummary)}), b.Timestamptz)

	registerTimeWeightInterpolation(b)

	addAggregate(b, funcNameRollup,
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTimeWeightSummary)}),
		opaqueType(typeNameTimeWeightSummary),
	)
}

// registerTimeWeightInterpolation registers the continuous-aggregate interpolation
// helpers that take the surrounding bucket boundaries and prior/next states to smooth the
// time-weighted estimate across bucket joins.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// interpolation helpers.
func registerTimeWeightInterpolation(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull("interpolated_average",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTimeWeightSummary)},
			db_engine_postgres.Arg{Name: paramNameStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameInterval, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNamePrev, Type: opaqueType(typeNameTimeWeightSummary)},
			db_engine_postgres.Arg{Name: paramNameNext, Type: opaqueType(typeNameTimeWeightSummary)},
		),
		b.Float8,
	)
	b.NullOnNull("interpolated_integral",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTimeWeightSummary)},
			db_engine_postgres.Arg{Name: paramNameStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameInterval, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNamePrev, Type: opaqueType(typeNameTimeWeightSummary)},
			db_engine_postgres.Arg{Name: paramNameNext, Type: opaqueType(typeNameTimeWeightSummary)},
			db_engine_postgres.Arg{Name: "unit", Type: b.Text},
		),
		b.Float8,
	)
}

// registerHeartbeatAccessorFamily registers the accessor surface on the heartbeat_agg
// state. Heartbeats record the periods during which a sender was alive; the accessors
// expose live or dead ranges, total uptime and downtime, point-in-time liveness, range
// counts, and continuous-aggregate interpolated variants.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// heartbeat accessors.
func registerHeartbeatAccessorFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	addReturnsSet(b, "live_ranges",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameHeartbeat)}),
		opaqueType("heartbeat_range_record"),
	)
	addReturnsSet(b, "dead_ranges",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameHeartbeat)}),
		opaqueType("heartbeat_range_record"),
	)

	b.NullOnNull("uptime", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameHeartbeat)}), b.Interval)
	b.NullOnNull("downtime", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameHeartbeat)}), b.Interval)
	b.NullOnNull("live_at",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameHeartbeat)}, db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz}),
		b.Boolean,
	)
	b.NullOnNull("num_live_ranges", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameHeartbeat)}), b.Integer)
	b.NullOnNull("num_gaps", b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameHeartbeat)}), b.Integer)

	b.NullOnNull("interpolated_uptime",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameHeartbeat)}, db_engine_postgres.Arg{Name: paramNamePrev, Type: opaqueType(typeNameHeartbeat)}),
		b.Interval,
	)
	b.NullOnNull("interpolated_downtime",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameHeartbeat)}, db_engine_postgres.Arg{Name: paramNamePrev, Type: opaqueType(typeNameHeartbeat)}),
		b.Interval,
	)

	b.NullOnNull("trim_to",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameHeartbeat)},
			db_engine_postgres.Arg{Name: paramNameStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: "duration", Type: b.Interval},
		),
		opaqueType(typeNameHeartbeat),
	)
}

// registerStateAccessorFamily registers accessor functions on the state aggregate state,
// used to inspect how long a series spent in each state. Both text- and bigint-valued
// state representations are covered, as is the compact_state_agg state surface which
// shares the same accessor names but uses its own opaque value type.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// state accessors.
func registerStateAccessorFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	for _, stateName := range stateSummaryStateNames {
		addReturnsSet(b, funcNameIntoValues, b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)}), b.Text)

		b.NullOnNull("duration_in",
			b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)}, db_engine_postgres.Arg{Name: paramNameState, Type: b.Text}),
			b.Interval,
		)
		b.NullOnNull("duration_in",
			b.Args(
				db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)},
				db_engine_postgres.Arg{Name: paramNameState, Type: b.Text},
				db_engine_postgres.Arg{Name: paramNameStart, Type: b.Timestamptz},
				db_engine_postgres.Arg{Name: "duration", Type: b.Interval},
			),
			b.Interval,
		)

		registerStateTimelineAccessors(b, stateName)
		registerStatePeriodAccessors(b, stateName)
		registerStatePointAccessors(b, stateName)
		registerStateInterpolatedAccessors(b, stateName)
	}
}

// registerStateTimelineAccessors registers the timeline accessors that project the
// state-summary state into a set of (state, range) tuples covering the captured window.
// The text variant returns an opaque state_timeline_record per row; the int variant uses
// the bigint state surface and returns state_int_timeline_record.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// timeline accessors.
// Takes stateName (string) which is the opaque state engine name.
func registerStateTimelineAccessors(b *db_engine_postgres.FunctionCatalogueBuilder, stateName string) {
	addReturnsSet(b, "state_timeline",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)}),
		opaqueType("state_timeline_record"),
	)
	addReturnsSet(b, "state_int_timeline",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)}),
		opaqueType("state_int_timeline_record"),
	)
}

// registerStatePeriodAccessors registers the period accessors that return the (start,
// end) intervals during which the series was in a caller-supplied state. The text and int
// variants mirror the timeline accessors.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// period accessors.
// Takes stateName (string) which is the opaque state engine name.
func registerStatePeriodAccessors(b *db_engine_postgres.FunctionCatalogueBuilder, stateName string) {
	addReturnsSet(b, "state_periods",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)}, db_engine_postgres.Arg{Name: paramNameState, Type: b.Text}),
		opaqueType("state_period_record"),
	)
	addReturnsSet(b, "state_int_periods",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)}, db_engine_postgres.Arg{Name: paramNameState, Type: b.Bigint}),
		opaqueType("state_int_period_record"),
	)
}

// registerStatePointAccessors registers the point-in-time accessors that report which
// state the series was in at a given timestamp.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// point-in-time accessors.
// Takes stateName (string) which is the opaque state engine name.
func registerStatePointAccessors(b *db_engine_postgres.FunctionCatalogueBuilder, stateName string) {
	b.NullOnNull("state_at",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)}, db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz}),
		b.Text,
	)
	b.NullOnNull("state_at_int",
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)}, db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz}),
		b.Bigint,
	)
}

// registerStateInterpolatedAccessors registers the continuous-aggregate interpolation
// accessors that smooth the state timeline and duration across bucket boundaries using
// prior and next states.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// interpolation accessors.
// Takes stateName (string) which is the opaque state engine name.
func registerStateInterpolatedAccessors(b *db_engine_postgres.FunctionCatalogueBuilder, stateName string) {
	addReturnsSet(b, "interpolated_state_timeline",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)},
			db_engine_postgres.Arg{Name: paramNameStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameInterval, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNamePrev, Type: opaqueType(stateName)},
			db_engine_postgres.Arg{Name: paramNameNext, Type: opaqueType(stateName)},
		),
		opaqueType("state_timeline_record"),
	)
	addReturnsSet(b, "interpolated_state_periods",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)},
			db_engine_postgres.Arg{Name: paramNameState, Type: b.Text},
			db_engine_postgres.Arg{Name: paramNameStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameInterval, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNamePrev, Type: opaqueType(stateName)},
			db_engine_postgres.Arg{Name: paramNameNext, Type: opaqueType(stateName)},
		),
		opaqueType("state_period_record"),
	)
	b.NullOnNull("interpolated_duration_in",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)},
			db_engine_postgres.Arg{Name: paramNameState, Type: b.Text},
			db_engine_postgres.Arg{Name: paramNameStart, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameInterval, Type: b.Interval},
			db_engine_postgres.Arg{Name: paramNamePrev, Type: opaqueType(stateName)},
			db_engine_postgres.Arg{Name: paramNameNext, Type: opaqueType(stateName)},
		),
		b.Interval,
	)
}

// registerMetadataFamily registers approximate_row_count and the size-introspection
// functions. Size functions report bytes and the approximate counter reports a bigint row
// count.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the registered
// metadata signatures.
func registerMetadataFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NeverNull("approximate_row_count", b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}), b.Bigint)
	b.NullOnNull("hypertable_size", b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}), b.Bigint)
	b.NullOnNull("chunk_size", b.Args(db_engine_postgres.Arg{Name: paramNameChunk, Type: regclassType()}), b.Bigint)
	b.NullOnNull("hypertable_index_size", b.Args(db_engine_postgres.Arg{Name: "index", Type: regclassType()}), b.Bigint)
	addReturnsSet(b, "hypertable_detailed_size", b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}), opaqueType("hypertable_detailed_size_record"))
	addReturnsSet(b, "chunks_detailed_size", b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}), opaqueType("chunk_detailed_size_record"))
	addReturnsSet(b, "hypertable_compression_stats", b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}), opaqueType(typeNameCompressionStatsRecord))
	addReturnsSet(b, "chunk_compression_stats", b.Args(db_engine_postgres.Arg{Name: paramNameHypertable, Type: regclassType()}), opaqueType(typeNameCompressionStatsRecord))
}
