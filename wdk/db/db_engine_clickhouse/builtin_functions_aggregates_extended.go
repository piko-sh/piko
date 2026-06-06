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

package db_engine_clickhouse

const (
	// sequenceNextNodeMinArgs is the minimum argument count for the sequenceNextNode
	// parametric aggregate: timestamp, event column and a base condition. Additional event
	// conditions repeat after the base triple.
	sequenceNextNodeMinArgs = 3

	// sequenceMatchEventsMinArgs is the minimum argument count for the variadic
	// sequenceMatchEvents parametric aggregate: pattern, timestamp and one base condition.
	// Additional event conditions repeat after the base triple.
	sequenceMatchEventsMinArgs = 3
)

// registerExtendedAggregateFunctions covers the wider arithmetic, quantile, time-series,
// parametric and structural aggregate set that sits outside the historic core
// registration block.
//
// It delegates to topical helpers so each one fits the function-length budget and so the
// topical anchors are easy to find when adding new entries.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedAggregateFunctions(b *FunctionCatalogueBuilder) {
	registerExtendedArgAndCorrMatrixAggregates(b)
	registerExtendedCovarStableAggregates(b)
	registerExtendedVarAndStddevStableAggregates(b)
	registerExtendedStructuralAggregates(b)
	registerExtendedHistogramAndQuantileAggregates(b)
	registerExtendedSumMapAggregates(b)
	registerExtendedTimeSeriesAggregates(b)
	registerExtendedParametricAggregates(b)
}

// registerExtendedArgAndCorrMatrixAggregates covers the argAndMin and argAndMax pair plus
// the correlation matrix and stable correlation aggregates.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedArgAndCorrMatrixAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("argAndMin", b.unknownType, b.unknownType, b.unknownType)
	b.RegisterAggregate("argAndMax", b.unknownType, b.unknownType, b.unknownType)
	b.RegisterVariadicAggregate("corrMatrix", arrayOf(arrayOf(b.float64Type)), 2, b.float64Type)
	b.RegisterAggregate("corrStable", b.float64Type, b.float64Type, b.float64Type)
}

// registerExtendedCovarStableAggregates covers the covariance-matrix and
// stable-covariance aggregates (both population and sample).
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedCovarStableAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterVariadicAggregate("covarPopMatrix", arrayOf(arrayOf(b.float64Type)), 2, b.float64Type)
	b.RegisterAggregate("covarPopStable", b.float64Type, b.float64Type, b.float64Type)
	b.RegisterVariadicAggregate("covarSampMatrix", arrayOf(arrayOf(b.float64Type)), 2, b.float64Type)
	b.RegisterAggregate("covarSampStable", b.float64Type, b.float64Type, b.float64Type)
}

// registerExtendedVarAndStddevStableAggregates covers the stable variance and stable
// standard-deviation aggregates (both population and sample variants).
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedVarAndStddevStableAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("varPopStable", b.float64Type, b.float64Type)
	b.RegisterAggregate("varSampStable", b.float64Type, b.float64Type)
	b.RegisterAggregate("stddevPopStable", b.float64Type, b.float64Type)
	b.RegisterAggregate("stddevSampStable", b.float64Type, b.float64Type)
}

// registerExtendedStructuralAggregates covers the structural and inspection aggregates.
//
// These are stochastic logistic regression, dynamic and JSON path distinct sets,
// compression-ratio estimation, the flame-graph builder, group-array concatenation, and
// the intersection counters.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedStructuralAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("stochasticLogisticRegression", arrayOf(b.float64Type), b.float64Type, b.float64Type)
	b.RegisterAggregate("distinctDynamicTypes", arrayOf(b.textType), b.unknownType)
	b.RegisterAggregate("distinctJSONPaths", arrayOf(b.textType), b.jsonType)
	b.RegisterAggregate("distinctJSONPathsAndTypes", arrayOf(b.textType), b.jsonType)
	b.RegisterAggregate("estimateCompressionRatio", b.float64Type, b.unknownType)
	b.RegisterAggregate("flameGraph", arrayOf(b.textType), arrayOf(b.uint64Type))
	b.RegisterAggregate("flameGraph", arrayOf(b.textType), arrayOf(b.uint64Type), b.uint64Type)
	b.RegisterAggregate("flameGraph", arrayOf(b.textType), arrayOf(b.uint64Type), b.uint64Type, b.uint64Type)
	b.RegisterAggregate("groupArrayArray", arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.RegisterAggregate("groupArrayIntersect", arrayOf(b.unknownType), arrayOf(b.unknownType))

	b.RegisterAggregate("maxIntersections", b.unknownType, b.unknownType, b.unknownType)
	b.RegisterAggregate("maxIntersectionsPosition", b.unknownType, b.unknownType, b.unknownType)
}

// registerExtendedHistogramAndQuantileAggregates covers the sparkbar visualisation
// aggregate plus the extended quantile family.
//
// The quantile family spans DD, exact-exclusive, inclusive, high, low,
// weighted-interpolated, Prometheus histogram, the GK plural form, and TDigest weighted.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedHistogramAndQuantileAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("sparkbar", b.textType, b.unknownType, b.unknownType)
	b.RegisterAggregate("quantileDD", b.float64Type, b.float64Type)
	b.RegisterAggregate("quantileExactExclusive", b.float64Type, b.float64Type)
	b.RegisterAggregate("quantileExactInclusive", b.float64Type, b.float64Type)
	b.RegisterAggregate("quantileExactHigh", b.float64Type, b.float64Type)
	b.RegisterAggregate("quantileExactLow", b.float64Type, b.float64Type)
	b.RegisterAggregate("quantileExactWeightedInterpolated", b.float64Type, b.float64Type, b.uint64Type)
	b.RegisterAggregate("quantilePrometheusHistogram", b.float64Type, b.float64Type, b.float64Type)
	b.RegisterAggregate("quantilesExactExclusive", arrayOf(b.float64Type), b.float64Type)
	b.RegisterAggregate("quantilesExactInclusive", arrayOf(b.float64Type), b.float64Type)
	b.RegisterAggregate("quantilesGK", arrayOf(b.float64Type), b.uint64Type, b.float64Type)
	b.RegisterAggregate("quantilesTimingWeighted", arrayOf(b.float64Type), b.float64Type, b.uint64Type)
	b.RegisterAggregate("quantileTDigestWeighted", b.float64Type, b.float64Type, b.uint64Type)
}

// registerExtendedSumMapAggregates covers the sumMap aggregate extensions.
//
// These are the overflow-aware variant, the filtered variant, and the combined overflow
// plus filtered variant.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedSumMapAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("sumMapWithOverflow", b.unknownType, arrayOf(b.unknownType), arrayOf(b.float64Type))
	b.RegisterAggregate("sumMapFiltered", b.unknownType, arrayOf(b.unknownType), arrayOf(b.float64Type))
	b.RegisterAggregate("sumMapFilteredWithOverflow", b.unknownType, arrayOf(b.unknownType), arrayOf(b.float64Type))
}

// registerExtendedTimeSeriesAggregates covers the time-series grid family which projects
// irregular samples onto a regular grid for downstream analytics.
//
// Every member returns Array(Float64) because the aggregation produces one bucket per
// grid position. ClickHouse exposes two arities: the short (timestamp, value) form that
// inherits grid parameters from the surrounding context, and the extended form that takes
// explicit (start, end, step, staleness) integer arguments before the (timestamp, value)
// columns.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedTimeSeriesAggregates(b *FunctionCatalogueBuilder) {
	timeSeriesNames := []string{
		"timeSeriesChangesToGrid",
		"timeSeriesDeltaToGrid",
		"timeSeriesDerivToGrid",
		"timeSeriesGroupArray",
		"timeSeriesInstantDeltaToGrid",
		"timeSeriesInstantRateToGrid",
		"timeSeriesLastTwoSamples",
		"timeSeriesPredictLinearToGrid",
		"timeSeriesRateToGrid",
		"timeSeriesResampleToGridWithStaleness",
		"timeSeriesResetsToGrid",
	}
	for _, name := range timeSeriesNames {
		b.RegisterAggregate(name, arrayOf(b.float64Type), b.dateTimeType, b.float64Type)

		b.RegisterAggregate(name, arrayOf(b.float64Type),
			b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type,
			b.dateTimeType, b.float64Type)
	}
}

// registerExtendedParametricAggregates covers the parametric aggregates, namely the
// pattern-matching sequence helpers plus the uniqUpTo truncated distinct counter.
//
// These calls take a literal parametric argument list (for example `uniqUpTo(N)(x)` or
// `sequenceMatchEvents('pattern')(t, c)`) that the parser handles before the catalogue is
// consulted, so the catalogue models only the data-argument signature. The parametric
// entries registered here are uniqUpTo, sequenceMatchEvents, and sequenceNextNode. Other
// parametric entries elsewhere in the catalogue are flameGraph, quantileGK, quantilesGK,
// quantileExactWeightedInterpolated, sparkbar, quantileDD, and
// quantilePrometheusHistogram.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedParametricAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterVariadicAggregate("sequenceMatchEvents", arrayOf(b.textType), sequenceMatchEventsMinArgs, b.textType, b.dateTimeType, b.boolType)
	b.RegisterVariadicAggregate("sequenceNextNode", b.unknownType, sequenceNextNodeMinArgs, b.dateTimeType, b.unknownType, b.boolType)
	b.RegisterAggregate("uniqUpTo", b.uint64Type, b.unknownType)
}
