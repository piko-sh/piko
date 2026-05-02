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

// functions_round2.go covers the toolkit aggregate families that
// extend TimescaleDB beyond the core hypertable surface: frequency
// analysis (space-saving), min_n / max_n / *_by, timevector (LTTB /
// ASAP), count_min_sketch, saturating math, and the
// timescaledb_pre/post_restore extension lifecycle helpers. The split
// keeps functions.go close to the canonical statement-management
// surface and groups the long-tail toolkit aggregates together so
// adding a new opaque-state family is a localised edit.

package db_engine_timescaledb

import (
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

var (
	// spaceSavingStateNames lists the engine names of the three space-saving aggregate
	// states; the topn, into_values, and scalar accessor registrations share the same
	// per-state loop so collecting the names in one slice keeps them aligned.
	spaceSavingStateNames = []string{
		typeNameSpaceSavingBigintAggregate,
		typeNameSpaceSavingTextAggregate,
		typeNameSpaceSavingAggregate,
	}
)

// registerFrequencyAnalysisFamily registers the space-saving family of aggregates.
//
// These are freq_agg / mcv_agg for bigint and text observation columns, the polymorphic
// raw_freq_agg / raw_mcv_agg constructors, and the topn / into_values / *_frequency
// accessors. The aggregate states are exposed as opaque types so downstream consumers can
// pass them between continuous aggregates without forcing a numeric materialisation.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerFrequencyAnalysisFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	registerFrequencyAggregateConstructors(b)
	registerFrequencyAggregateAccessors(b)
	registerFrequencyAggregateRollup(b)
}

// registerFrequencyAggregateConstructors registers the freq_agg / mcv_agg / raw_freq_agg
// / raw_mcv_agg aggregate constructors.
//
// The concrete bigint and text overloads return the matching typed state so downstream
// rollups and accessors do not need to widen through the polymorphic raw state.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerFrequencyAggregateConstructors(b *db_engine_postgres.FunctionCatalogueBuilder) {
	addAggregate(b, "freq_agg",
		b.Args(db_engine_postgres.Arg{Name: "frequency", Type: b.Float8}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Bigint}),
		opaqueType(typeNameSpaceSavingBigintAggregate),
	)
	addAggregate(b, "freq_agg",
		b.Args(db_engine_postgres.Arg{Name: "frequency", Type: b.Float8}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Text}),
		opaqueType(typeNameSpaceSavingTextAggregate),
	)
	addAggregate(b, funcNameMCVAgg,
		b.Args(db_engine_postgres.Arg{Name: paramNameCount, Type: b.Integer}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Bigint}),
		opaqueType(typeNameSpaceSavingBigintAggregate),
	)
	addAggregate(b, funcNameMCVAgg,
		b.Args(db_engine_postgres.Arg{Name: paramNameCount, Type: b.Integer}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Text}),
		opaqueType(typeNameSpaceSavingTextAggregate),
	)
	addAggregate(b, funcNameMCVAgg,
		b.Args(db_engine_postgres.Arg{Name: paramNameCount, Type: b.Integer}, db_engine_postgres.Arg{Name: "skew", Type: b.Float8}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Bigint}),
		opaqueType(typeNameSpaceSavingBigintAggregate),
	)
	addAggregate(b, funcNameMCVAgg,
		b.Args(db_engine_postgres.Arg{Name: paramNameCount, Type: b.Integer}, db_engine_postgres.Arg{Name: "skew", Type: b.Float8}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Text}),
		opaqueType(typeNameSpaceSavingTextAggregate),
	)
	addAggregate(b, "raw_freq_agg",
		b.Args(db_engine_postgres.Arg{Name: "frequency", Type: b.Float8}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any}),
		opaqueType(typeNameSpaceSavingAggregate),
	)
	addAggregate(b, "raw_mcv_agg",
		b.Args(db_engine_postgres.Arg{Name: paramNameCount, Type: b.Integer}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any}),
		opaqueType(typeNameSpaceSavingAggregate),
	)
	addAggregate(b, "raw_mcv_agg",
		b.Args(db_engine_postgres.Arg{Name: paramNameCount, Type: b.Integer}, db_engine_postgres.Arg{Name: "skew", Type: b.Float8}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any}),
		opaqueType(typeNameSpaceSavingAggregate),
	)
}

// registerFrequencyAggregateAccessors registers topn, into_values, and min_frequency /
// max_frequency.
//
// topn returns a SETOF (value, frequency) tuple; into_values returns a SETOF (value,
// min_freq, max_freq) tuple. Both the typed and polymorphic state surfaces are
// registered.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerFrequencyAggregateAccessors(b *db_engine_postgres.FunctionCatalogueBuilder) {
	registerFrequencyTopnAccessors(b)
	registerFrequencyIntoValuesAccessors(b)
	registerFrequencyScalarAccessors(b)
}

// registerFrequencyTopnAccessors registers topn(agg) and topn(agg, n) across the bigint,
// text, and polymorphic space-saving states.
//
// topn returns a SETOF record so the catalogue records the row shape as an opaque
// per-type record name.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerFrequencyTopnAccessors(b *db_engine_postgres.FunctionCatalogueBuilder) {
	for _, stateName := range spaceSavingStateNames {
		addReturnsSet(b, "topn",
			b.Args(db_engine_postgres.Arg{Name: paramNameAgg, Type: opaqueType(stateName)}),
			opaqueType(stateName+"_topn_record"),
		)
		addReturnsSet(b, "topn",
			b.Args(db_engine_postgres.Arg{Name: paramNameAgg, Type: opaqueType(stateName)}, db_engine_postgres.Arg{Name: paramNameN, Type: b.Integer}),
			opaqueType(stateName+"_topn_record"),
		)
	}
}

// registerFrequencyIntoValuesAccessors registers into_values on the space-saving states.
//
// The 1-arg form returns the captured values; the 2-arg form takes a witness value to
// type-hint the return for the polymorphic state.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerFrequencyIntoValuesAccessors(b *db_engine_postgres.FunctionCatalogueBuilder) {
	for _, stateName := range spaceSavingStateNames {
		addReturnsSet(b, funcNameIntoValues,
			b.Args(db_engine_postgres.Arg{Name: paramNameAgg, Type: opaqueType(stateName)}),
			opaqueType(stateName+"_into_values_record"),
		)
		addReturnsSet(b, funcNameIntoValues,
			b.Args(db_engine_postgres.Arg{Name: paramNameAgg, Type: opaqueType(stateName)}, db_engine_postgres.Arg{Name: "witness", Type: b.Any}),
			opaqueType(stateName+"_into_values_record"),
		)
	}
}

// registerFrequencyScalarAccessors registers max_frequency and min_frequency on the
// space-saving state surface.
//
// Each returns a float8 frequency bound for the requested value.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerFrequencyScalarAccessors(b *db_engine_postgres.FunctionCatalogueBuilder) {
	for _, stateName := range spaceSavingStateNames {
		b.NullOnNull("max_frequency",
			b.Args(db_engine_postgres.Arg{Name: paramNameAgg, Type: opaqueType(stateName)}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any}),
			b.Float8,
		)
		b.NullOnNull("min_frequency",
			b.Args(db_engine_postgres.Arg{Name: paramNameAgg, Type: opaqueType(stateName)}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Any}),
			b.Float8,
		)
	}
}

// registerFrequencyAggregateRollup registers rollup over the three space-saving state
// shapes.
//
// Each merges per-bucket states into a matching aggregate state, preserving the
// underlying value type.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerFrequencyAggregateRollup(b *db_engine_postgres.FunctionCatalogueBuilder) {
	for _, stateName := range spaceSavingStateNames {
		addAggregate(b, funcNameRollup,
			b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(stateName)}),
			opaqueType(stateName),
		)
	}
}

// registerMinMaxNFamily registers min_n / max_n and the by-clause min_n_by / max_n_by
// aggregates.
//
// The non-by aggregates take a value column and a top-N count and retain the N smallest
// or largest values; the *_by aggregates additionally retain a paired secondary value
// which the into_values accessor unpacks alongside the principal value. The polymorphic
// Any registration matches the upstream catalogue where the value type is inferred from
// the call site.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerMinMaxNFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	registerMinMaxNConstructors(b)
	registerMinMaxNAccessors(b)
	registerMinMaxNRollup(b)
}

// registerMinMaxNConstructors registers the min_n / max_n / min_n_by / max_n_by aggregate
// constructors.
//
// The constructors cover the float8, bigint, text, and timestamptz value types plus the
// polymorphic Any surface. The loop iterates by index and reads each element through the
// indexed expression so the value-type struct is not copied into a loop-local binding on
// each iteration.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerMinMaxNConstructors(b *db_engine_postgres.FunctionCatalogueBuilder) {
	valueTypes := []querier_dto.SQLType{b.Float8, b.Bigint, b.Text, b.Timestamptz, b.Any}
	for index := range valueTypes {
		addAggregate(b, "min_n",
			b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: valueTypes[index]}, db_engine_postgres.Arg{Name: paramNameN, Type: b.Integer}),
			opaqueType(typeNameMinNState),
		)
		addAggregate(b, "max_n",
			b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: valueTypes[index]}, db_engine_postgres.Arg{Name: paramNameN, Type: b.Integer}),
			opaqueType(typeNameMinNState),
		)
		addAggregate(b, "min_n_by",
			b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: valueTypes[index]}, db_engine_postgres.Arg{Name: "by", Type: b.Any}, db_engine_postgres.Arg{Name: paramNameN, Type: b.Integer}),
			opaqueType(typeNameMinNByState),
		)
		addAggregate(b, "max_n_by",
			b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: valueTypes[index]}, db_engine_postgres.Arg{Name: "by", Type: b.Any}, db_engine_postgres.Arg{Name: paramNameN, Type: b.Integer}),
			opaqueType(typeNameMinNByState),
		)
	}
}

// registerMinMaxNAccessors registers into_array and into_values on the min_n / min_n_by
// states.
//
// into_array projects the captured values back into an array; into_values returns the
// captured values as a SETOF row (plus the secondary value for the by-clause state).
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerMinMaxNAccessors(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NullOnNull("into_array",
		b.Args(db_engine_postgres.Arg{Name: paramNameState, Type: opaqueType(typeNameMinNState)}),
		arrayOf(b.Any),
	)
	b.NullOnNull("into_array",
		b.Args(db_engine_postgres.Arg{Name: paramNameState, Type: opaqueType(typeNameMinNByState)}),
		arrayOf(b.Any),
	)
	addReturnsSet(b, funcNameIntoValues,
		b.Args(db_engine_postgres.Arg{Name: paramNameState, Type: opaqueType(typeNameMinNState)}),
		b.Any,
	)
	addReturnsSet(b, funcNameIntoValues,
		b.Args(db_engine_postgres.Arg{Name: paramNameState, Type: opaqueType(typeNameMinNByState)}),
		opaqueType("min_n_by_record"),
	)
}

// registerMinMaxNRollup registers rollup over min_n / min_n_by state shapes.
//
// Both aggregates project back to their input shape so continuous aggregates can carry
// the captured top-N values across bucket boundaries.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerMinMaxNRollup(b *db_engine_postgres.FunctionCatalogueBuilder) {
	addAggregate(b, funcNameRollup,
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameMinNState)}),
		opaqueType(typeNameMinNState),
	)
	addAggregate(b, funcNameRollup,
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameMinNByState)}),
		opaqueType(typeNameMinNByState),
	)
}

// registerTimevectorFamily registers the lttb / asap_smooth / timevector aggregates and
// the unnest accessor that projects a timevector back into per-row tuples.
//
// lttb downsamples to a fixed resolution; asap_smooth applies an adaptive smoothing pass;
// timevector retains the raw paired observations. Each constructor accepts a timestamptz
// column and a float8 value column; each has both the aggregate form (which builds a new
// state from raw observations) and a state-to-state form (which applies the algorithm to
// an existing timevector).
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerTimevectorFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	addAggregate(b, "lttb",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8},
			db_engine_postgres.Arg{Name: "resolution", Type: b.Integer},
		),
		opaqueType(typeNameTimevectorTstzF64),
	)
	b.NullOnNull("lttb",
		b.Args(db_engine_postgres.Arg{Name: "series", Type: opaqueType(typeNameTimevectorTstzF64)}, db_engine_postgres.Arg{Name: "threshold", Type: b.Integer}),
		opaqueType(typeNameTimevectorTstzF64),
	)
	addAggregate(b, "asap_smooth",
		b.Args(
			db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz},
			db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8},
			db_engine_postgres.Arg{Name: "resolution", Type: b.Integer},
		),
		opaqueType(typeNameTimevectorTstzF64),
	)
	b.NullOnNull("asap_smooth",
		b.Args(db_engine_postgres.Arg{Name: "series", Type: opaqueType(typeNameTimevectorTstzF64)}, db_engine_postgres.Arg{Name: "resolution", Type: b.Integer}),
		opaqueType(typeNameTimevectorTstzF64),
	)
	addAggregate(b, "timevector",
		b.Args(db_engine_postgres.Arg{Name: paramNameTS, Type: b.Timestamptz}, db_engine_postgres.Arg{Name: paramNameValue, Type: b.Float8}),
		opaqueType(typeNameTimevectorTstzF64),
	)
	addAggregate(b, funcNameRollup,
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTimevectorTstzF64)}),
		opaqueType(typeNameTimevectorTstzF64),
	)
	addReturnsSet(b, "unnest",
		b.Args(db_engine_postgres.Arg{Name: "series", Type: opaqueType(typeNameTimevectorTstzF64)}),
		opaqueType("timevector_point_record"),
	)
}

// registerCountMinSketchFamily registers the count_min_sketch aggregate and the
// approx_count accessor.
//
// The aggregate accepts an observed text value plus error and probability bounds that
// control the sketch's accuracy guarantees; approx_count returns the estimated frequency
// of a probe value.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerCountMinSketchFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	addAggregate(b, "count_min_sketch",
		b.Args(db_engine_postgres.Arg{Name: paramNameValue, Type: b.Text}, db_engine_postgres.Arg{Name: "error", Type: b.Float8}, db_engine_postgres.Arg{Name: "probability", Type: b.Float8}),
		opaqueType(typeNameCountMinSketch),
	)
	b.NullOnNull("approx_count",
		b.Args(db_engine_postgres.Arg{Name: "item", Type: b.Text}, db_engine_postgres.Arg{Name: paramNameAgg, Type: opaqueType(typeNameCountMinSketch)}),
		b.Bigint,
	)
	addAggregate(b, funcNameRollup,
		b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCountMinSketch)}),
		opaqueType(typeNameCountMinSketch),
	)
}

// registerSaturatingMathFamily registers the saturating_* integer math helpers from the
// TimescaleDB toolkit.
//
// Each clamps to the integer bounds rather than wrapping or raising; the _pos variants
// clamp at zero rather than at the negative integer bound. All variants take and return
// the canonical int4 type.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerSaturatingMathFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	for _, name := range []string{
		"saturating_add",
		"saturating_sub",
		"saturating_mul",
		"saturating_add_pos",
		"saturating_sub_pos",
		"saturating_mul_pos",
	} {
		b.NullOnNull(name,
			b.Args(db_engine_postgres.Arg{Name: "a", Type: b.Integer}, db_engine_postgres.Arg{Name: "b", Type: b.Integer}),
			b.Integer,
		)
	}
}

// registerExtensionLifecycleFamily registers timescaledb_pre_restore and
// timescaledb_post_restore.
//
// Both are void-returning system helpers the dump/restore tooling uses to disable and
// re-enable the background workers around a pg_restore run; they take no arguments.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerExtensionLifecycleFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	b.NeverNull("timescaledb_pre_restore", nil, voidType())
	b.NeverNull("timescaledb_post_restore", nil, voidType())
}

// registerRollupFamily registers the toolkit rollup functions used to merge aggregate
// states across continuous-aggregate buckets.
//
// The time_weight_summary rollup is registered alongside the time_weight family because
// it lives in the same accessor surface; this helper covers every other opaque state.
//
// Takes b (*db_engine_postgres.FunctionCatalogueBuilder) which receives the signatures.
func registerRollupFamily(b *db_engine_postgres.FunctionCatalogueBuilder) {
	addAggregate(b, funcNameRollup, b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary1D)}), opaqueType(typeNameStatsSummary1D))
	addAggregate(b, funcNameRollup, b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStatsSummary2D)}), opaqueType(typeNameStatsSummary2D))
	addAggregate(b, funcNameRollup, b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCounterSummary)}), opaqueType(typeNameCounterSummary))
	addAggregate(b, funcNameRollup, b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameGaugeSummary)}), opaqueType(typeNameGaugeSummary))
	addAggregate(b, funcNameRollup, b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCandlestick)}), opaqueType(typeNameCandlestick))
	addAggregate(b, funcNameRollup, b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameStateSummary)}), opaqueType(typeNameStateSummary))
	addAggregate(b, funcNameRollup, b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameCompactStateSummary)}), opaqueType(typeNameCompactStateSummary))
	addAggregate(b, funcNameRollup, b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameHeartbeat)}), opaqueType(typeNameHeartbeat))
	addAggregate(b, funcNameRollup, b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameHyperloglog)}), opaqueType(typeNameHyperloglog))
	addAggregate(b, funcNameRollup, b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameTDigest)}), opaqueType(typeNameTDigest))
	addAggregate(b, funcNameRollup, b.Args(db_engine_postgres.Arg{Name: paramNameSummary, Type: opaqueType(typeNameUDDSketch)}), opaqueType(typeNameUDDSketch))
}
