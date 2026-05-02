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

import (
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// aggregateNameSum is the name of the SQL sum() aggregate, used in multiple
	// type-overloaded registrations.
	aggregateNameSum = "sum"

	// md5DigestBytes is the byte length of an MD5 hash digest.
	md5DigestBytes = 16

	// sha1DigestBytes is the byte length of a SHA-1 hash digest.
	sha1DigestBytes = 20

	// sha224DigestBytes is the byte length of a SHA-224 hash digest.
	sha224DigestBytes = 28

	// sha256DigestBytes is the byte length of a SHA-256 hash digest.
	sha256DigestBytes = 32

	// sha384DigestBytes is the byte length of a SHA-384 hash digest.
	sha384DigestBytes = 48

	// sha512DigestBytes is the byte length of a SHA-512 hash digest.
	sha512DigestBytes = 64

	// multiIfMinArgs is the minimum argument count for multiIf: condition, value, default
	// (3).
	multiIfMinArgs = 3
)

var (
	// safetyConversionSuffixes is the canonical ordering of failure-mode suffixes for the
	// `toXxx` family. The catalogue records every base type x every suffix combination so
	// the analyser can match user spellings without an extra lookup table.
	safetyConversionSuffixes = []string{"OrNull", "OrZero", "OrDefault"}

	// nullZeroSafetyConversionSuffixes is the OrNull / OrZero subset for families that lack
	// an OrDefault spelling, such as the parseDateTime64 parsers.
	nullZeroSafetyConversionSuffixes = []string{"OrNull", "OrZero"}

	// defaultNullSafetyConversionSuffixes is the OrDefault / OrNull subset for families that
	// lack an OrZero spelling, such as the IP string-to-number conversions.
	defaultNullSafetyConversionSuffixes = []string{"OrDefault", "OrNull"}
)

// FunctionCatalogueBuilder accumulates ClickHouse function signatures before they are
// baked into the read-only catalogue returned from BuiltinFunctions. The builder exposes
// shorthand helpers (Register, RegisterVariadic, RegisterAggregate) so the registration
// blocks below stay readable.
type FunctionCatalogueBuilder struct {
	// functions maps a lower-cased function name to the signatures registered under it
	// during catalogue construction.
	functions map[string][]*querier_dto.FunctionSignature

	// uint64Type is the shared UInt64 SQLType shortcut. These type fields are computed once
	// at builder construction so each registration call is a short literal, and they carry
	// the *Type suffix so they do not shadow Go's built-in numeric type names.
	uint64Type querier_dto.SQLType

	// uint32Type is the shared UInt32 SQLType shortcut.
	uint32Type querier_dto.SQLType

	// int64Type is the shared Int64 SQLType shortcut.
	int64Type querier_dto.SQLType

	// int32Type is the shared Int32 SQLType shortcut.
	int32Type querier_dto.SQLType

	// float64Type is the shared Float64 SQLType shortcut.
	float64Type querier_dto.SQLType

	// float32Type is the shared Float32 SQLType shortcut.
	float32Type querier_dto.SQLType

	// textType is the shared String SQLType shortcut.
	textType querier_dto.SQLType

	// boolType is the shared Bool SQLType shortcut.
	boolType querier_dto.SQLType

	// dateType is the shared Date SQLType shortcut.
	dateType querier_dto.SQLType

	// dateTimeType is the shared DateTime SQLType shortcut.
	dateTimeType querier_dto.SQLType

	// dateTime64Type is the shared DateTime64 SQLType shortcut.
	dateTime64Type querier_dto.SQLType

	// uuidType is the shared UUID SQLType shortcut.
	uuidType querier_dto.SQLType

	// decimal128Type is the shared Decimal128 SQLType shortcut.
	decimal128Type querier_dto.SQLType

	// jsonType is the shared JSON SQLType shortcut.
	jsonType querier_dto.SQLType

	// unknownType is the shared unknown-category SQLType shortcut.
	unknownType querier_dto.SQLType
}

// newFunctionCatalogueBuilder returns an empty builder primed with the shared SQLType
// shortcuts.
//
// Returns *FunctionCatalogueBuilder which is the primed, empty builder.
func newFunctionCatalogueBuilder() *FunctionCatalogueBuilder {
	return &FunctionCatalogueBuilder{
		functions:      map[string][]*querier_dto.FunctionSignature{},
		uint64Type:     querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
		uint32Type:     querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt32"},
		int64Type:      querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"},
		int32Type:      querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int32"},
		float64Type:    querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "Float64"},
		float32Type:    querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "Float32"},
		textType:       querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"},
		boolType:       querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"},
		dateType:       querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "Date"},
		dateTimeType:   querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "DateTime"},
		dateTime64Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "DateTime64"},
		uuidType:       querier_dto.SQLType{Category: querier_dto.TypeCategoryUUID, EngineName: "UUID"},
		decimal128Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "Decimal128"},
		jsonType:       querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "JSON"},

		unknownType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
	}
}

// Register adds a non-aggregate function signature.
//
// Names are lower-cased before insertion so the catalogue lookup is case-insensitive,
// matching ClickHouse SQL semantics where COUNT, Count and count refer to the same
// function.
//
// Takes name (string) which is the function name to register under.
// Takes returnType (querier_dto.SQLType) which is the function's result type.
// Takes argTypes (...querier_dto.SQLType) which are the argument types in order.
func (builder *FunctionCatalogueBuilder) Register(name string, returnType querier_dto.SQLType, argTypes ...querier_dto.SQLType) {
	args := make([]querier_dto.FunctionArgument, len(argTypes))
	for i := range argTypes {
		args[i] = querier_dto.FunctionArgument{Type: argTypes[i]}
	}
	key := strings.ToLower(name)
	builder.functions[key] = append(builder.functions[key], &querier_dto.FunctionSignature{
		Name:       name,
		ReturnType: returnType,
		Arguments:  args,
		DataAccess: querier_dto.DataAccessReadOnly,
	})
}

// RegisterVariadic adds a function whose last argument may repeat.
//
// Names are lower-cased to match Register's case-insensitive lookup.
//
// Takes name (string) which is the function name to register under.
// Takes returnType (querier_dto.SQLType) which is the function's result type.
// Takes minArgs (int) which is the minimum number of arguments accepted.
// Takes argTypes (...querier_dto.SQLType) which are the argument types in order.
func (builder *FunctionCatalogueBuilder) RegisterVariadic(name string, returnType querier_dto.SQLType, minArgs int, argTypes ...querier_dto.SQLType) {
	args := make([]querier_dto.FunctionArgument, len(argTypes))
	for i := range argTypes {
		args[i] = querier_dto.FunctionArgument{Type: argTypes[i]}
	}
	key := strings.ToLower(name)
	builder.functions[key] = append(builder.functions[key], &querier_dto.FunctionSignature{
		Name:         name,
		ReturnType:   returnType,
		Arguments:    args,
		IsVariadic:   true,
		MinArguments: minArgs,
		DataAccess:   querier_dto.DataAccessReadOnly,
	})
}

// RegisterNullPropagating adds a scalar function whose result is NULL exactly when an
// argument is NULL - the standard cast/conversion behaviour (toInt64, toString, ...). It
// mirrors Register but stamps FunctionNullableReturnsNullOnNull, so converting a non-null
// value (for example toInt64(count())) is typed non-nullable rather than the default
// always-nullable.
//
// Takes name (string) which is the function name to register under.
// Takes returnType (querier_dto.SQLType) which is the function's result type.
// Takes argTypes (...querier_dto.SQLType) which are the argument types in order.
func (builder *FunctionCatalogueBuilder) RegisterNullPropagating(name string, returnType querier_dto.SQLType, argTypes ...querier_dto.SQLType) {
	args := make([]querier_dto.FunctionArgument, len(argTypes))
	for i := range argTypes {
		args[i] = querier_dto.FunctionArgument{Type: argTypes[i]}
	}
	key := strings.ToLower(name)
	builder.functions[key] = append(builder.functions[key], &querier_dto.FunctionSignature{
		Name:              name,
		ReturnType:        returnType,
		Arguments:         args,
		DataAccess:        querier_dto.DataAccessReadOnly,
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	})
}

// RegisterAggregate adds an aggregate function signature.
//
// Names are lower-cased to match Register's case-insensitive lookup.
//
// Takes name (string) which is the function name to register under.
// Takes returnType (querier_dto.SQLType) which is the function's result type.
// Takes argTypes (...querier_dto.SQLType) which are the argument types in order.
func (builder *FunctionCatalogueBuilder) RegisterAggregate(name string, returnType querier_dto.SQLType, argTypes ...querier_dto.SQLType) {
	args := make([]querier_dto.FunctionArgument, len(argTypes))
	for i := range argTypes {
		args[i] = querier_dto.FunctionArgument{Type: argTypes[i]}
	}
	key := strings.ToLower(name)
	builder.functions[key] = append(builder.functions[key], &querier_dto.FunctionSignature{
		Name:              name,
		ReturnType:        returnType,
		Arguments:         args,
		IsAggregate:       true,
		DataAccess:        querier_dto.DataAccessReadOnly,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

// RegisterAggregateNoArgs adds a no-argument aggregate such as count().
//
// Names are lower-cased to match the other Register* helpers.
//
// Takes name (string) which is the function name to register under.
// Takes returnType (querier_dto.SQLType) which is the function's result type.
func (builder *FunctionCatalogueBuilder) RegisterAggregateNoArgs(name string, returnType querier_dto.SQLType) {
	key := strings.ToLower(name)
	builder.functions[key] = append(builder.functions[key], &querier_dto.FunctionSignature{
		Name:              name,
		ReturnType:        returnType,
		IsAggregate:       true,
		DataAccess:        querier_dto.DataAccessReadOnly,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

// RegisterVariadicAggregate adds an aggregate function whose last argument may repeat.
//
// It is used by the categorical-information-value and matrix correlation aggregates which
// take a variable number of columns. Names are lower-cased to match the other Register*
// helpers.
//
// Takes name (string) which is the function name to register under.
// Takes returnType (querier_dto.SQLType) which is the function's result type.
// Takes minArgs (int) which is the minimum number of arguments accepted.
// Takes argTypes (...querier_dto.SQLType) which are the argument types in order.
func (builder *FunctionCatalogueBuilder) RegisterVariadicAggregate(name string, returnType querier_dto.SQLType, minArgs int, argTypes ...querier_dto.SQLType) {
	args := make([]querier_dto.FunctionArgument, len(argTypes))
	for index := range argTypes {
		args[index] = querier_dto.FunctionArgument{Type: argTypes[index]}
	}
	key := strings.ToLower(name)
	builder.functions[key] = append(builder.functions[key], &querier_dto.FunctionSignature{
		Name:              name,
		ReturnType:        returnType,
		Arguments:         args,
		IsVariadic:        true,
		MinArguments:      minArgs,
		IsAggregate:       true,
		DataAccess:        querier_dto.DataAccessReadOnly,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

// build returns a finalised FunctionCatalogue from the accumulated registrations.
//
// Returns *querier_dto.FunctionCatalogue which is the finalised catalogue.
func (builder *FunctionCatalogueBuilder) build() *querier_dto.FunctionCatalogue {
	return &querier_dto.FunctionCatalogue{
		Functions: builder.functions,
	}
}

// buildFunctionCatalogue constructs the ClickHouse built-in function catalogue.
//
// Registration is grouped by concern such as aggregate, date, string and array helpers,
// and the order is purely organisational. The helper functions cover the core families
// and the extended families, and splitting keeps each registration block within the
// linter's function-length budget.
//
// Takes extras (func(*FunctionCatalogueBuilder)) which registers extra signatures after
// the built-in families, or nil to register none.
//
// Returns *querier_dto.FunctionCatalogue which is the finalised catalogue.
func buildFunctionCatalogue(extras func(*FunctionCatalogueBuilder)) *querier_dto.FunctionCatalogue {
	builder := newFunctionCatalogueBuilder()
	registerCoreCatalogueFamilies(builder)
	registerExtendedCatalogueFamilies(builder)
	registerSpecialisedCatalogueFamilies(builder)
	if extras != nil {
		extras(builder)
	}
	return builder.build()
}

// registerCoreCatalogueFamilies registers the core aggregate, date-time, string, array,
// tuple, map, type-conversion, arithmetic, logical, hash, URL, JSON, conditional, system,
// window, bitwise, comparison, random and crypto family helpers.
//
// Takes builder (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerCoreCatalogueFamilies(builder *FunctionCatalogueBuilder) {
	registerAggregateFunctions(builder)
	registerDateTimeFunctions(builder)
	registerStringFunctions(builder)
	registerStringSearchFunctions(builder)
	registerArrayFunctions(builder)
	registerTupleAndMapFunctions(builder)
	registerTypeConversionFunctions(builder)
	registerMathFunctions(builder)
	registerLogicalFunctions(builder)
	registerHashingFunctions(builder)
	registerURLAndIPFunctions(builder)
	registerJSONFunctions(builder)
	registerConditionalFunctions(builder)
	registerSystemFunctions(builder)
	registerWindowFunctions(builder)
	registerBitwiseFunctions(builder)
	registerComparisonFunctions(builder)
	registerRandomFunctions(builder)
	registerCryptoFunctions(builder)
	registerGeoFunctions(builder)
	registerNullableHelpers(builder)
	registerEncodingFunctions(builder)
}

// registerExtendedCatalogueFamilies registers the array higher-order, visit and funnel,
// cloud-storage, checksum, introspection, dictionary, ML, reinterpret, extended
// conversion, IP validator, UUID and Snowflake, extended string-search, array
// set-operation, extended URL and the string similarity, HTML and miscellaneous helpers.
//
// Takes builder (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerExtendedCatalogueFamilies(builder *FunctionCatalogueBuilder) {
	registerArrayHigherOrderFunctions(builder)
	registerVisitFunnelFunctions(builder)
	registerS3AndCloudFunctions(builder)
	registerCheckSumFunctions(builder)
	registerIntrospectionFunctions(builder)
	registerDictionaryFunctions(builder)
	registerMLFunctions(builder)
	registerReinterpretFunctions(builder)
	registerExtendedConversionFunctions(builder)
	registerIPValidatorFunctions(builder)
	registerUUIDSnowflakeFunctions(builder)
	registerExtendedStringSearchFunctions(builder)
	registerArraySetOperationFunctions(builder)
	registerExtendedURLFunctions(builder)
	registerStringSimilarityFunctions(builder)
	registerStringHTMLFunctions(builder)
	registerStringMiscFunctions(builder)
}

// registerSpecialisedCatalogueFamilies registers the specialised function families.
//
// These cover the statistical and extended aggregates, bitmap helpers, vector distance
// and norm helpers, NLP, the s2, polygon and extended H3 geo families, the hyperbolic,
// rounding and bit-slice helpers, and the extended hash, JSON, tuple-map, encoding,
// random, IN and window families.
//
// Takes builder (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerSpecialisedCatalogueFamilies(builder *FunctionCatalogueBuilder) {
	registerStatisticalAggregateFunctions(builder)
	registerExtendedAggregateFunctions(builder)
	registerBitmapFunctions(builder)
	registerDistanceAndNormFunctions(builder)
	registerNLPFunctions(builder)
	registerGeoS2Functions(builder)
	registerGeoPolygonFunctions(builder)
	registerH3ExtendedFunctions(builder)
	registerExtendedMathFunctions(builder)
	registerExtendedRoundingFunctions(builder)
	registerExtendedBitFunctions(builder)
	registerExtendedHashFunctions(builder)
	registerExtendedJSONFunctions(builder)
	registerExtendedTupleAndMapFunctions(builder)
	registerExtendedEncodingFunctions(builder)
	registerExtendedRandomFunctions(builder)
	registerInOperatorFunctions(builder)
	registerExtendedWindowFunctions(builder)
}

// registerAggregateFunctions covers the aggregate function family.
//
// These include count, sum, avg, min, max, the any and uniq variants, the groupArray
// family, argMin and argMax, the quantile family, variance, stddev, covar and corr, topK
// and linear regression. The body delegates to several helpers to keep any single
// function within the linter budget and to give each topical group an explicit anchor
// that doc-style search picks up.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerAggregateFunctions(b *FunctionCatalogueBuilder) {
	registerCoreAggregates(b)
	registerUniqAggregates(b)
	registerGroupArrayAggregates(b)
	registerArgAndQuantileAggregates(b)
	registerVarianceAggregates(b)
	registerTopKAndRegressionAggregates(b)
	registerExtendedSumAggregates(b)
	registerExtendedQuantileAggregates(b)
	registerHigherMomentAggregates(b)
	registerExponentialAggregates(b)
	registerSamplingAggregates(b)
	registerIntervalAndDeltaAggregates(b)
}

// registerCoreAggregates handles count, sum, avg, min, max and the any* family.
//
// Each registration sticks to the wider promoted return type so signature lookup matches
// the typical user spelling.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerCoreAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregateNoArgs("count", b.uint64Type)
	b.RegisterAggregate("count", b.uint64Type, b.unknownType)
	b.RegisterAggregate(aggregateNameSum, b.float64Type, b.float64Type)
	b.RegisterAggregate(aggregateNameSum, b.int64Type, b.int64Type)
	b.RegisterAggregate(aggregateNameSum, b.uint64Type, b.uint64Type)
	b.RegisterAggregate(aggregateNameSum, b.decimal128Type, b.decimal128Type)
	b.RegisterAggregate("avg", b.float64Type, b.float64Type)
	b.RegisterAggregate("avg", b.float64Type, b.int64Type)
	b.RegisterAggregate("min", b.unknownType, b.unknownType)
	b.RegisterAggregate("max", b.unknownType, b.unknownType)
	b.RegisterAggregate("any", b.unknownType, b.unknownType)
	b.RegisterAggregate("anyLast", b.unknownType, b.unknownType)
	b.RegisterAggregate("anyHeavy", b.unknownType, b.unknownType)
}

// registerUniqAggregates covers the distinct-count family.
//
// Every implementation returns UInt64 regardless of the input column type.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerUniqAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("uniq", b.uint64Type, b.unknownType)
	b.RegisterAggregate("uniqExact", b.uint64Type, b.unknownType)
	b.RegisterAggregate("uniqHLL12", b.uint64Type, b.unknownType)
	b.RegisterAggregate("uniqCombined", b.uint64Type, b.unknownType)
	b.RegisterAggregate("uniqCombined64", b.uint64Type, b.unknownType)
	b.RegisterAggregate("uniqTheta", b.uint64Type, b.unknownType)
}

// registerGroupArrayAggregates covers the per-group collection aggregates that build
// arrays out of the rows in each group.
//
// The extended variants groupArrayLast, Sample, Sorted, InsertAt and Concat are
// registered alongside the canonical groupArray name to keep the topical grouping intact.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerGroupArrayAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("groupArray", arrayOf(b.unknownType), b.unknownType)
	b.RegisterAggregate("groupUniqArray", arrayOf(b.unknownType), b.unknownType)
	b.RegisterAggregate("groupArrayMovingSum", arrayOf(b.float64Type), b.float64Type)
	b.RegisterAggregate("groupArrayMovingAvg", arrayOf(b.float64Type), b.float64Type)
	b.RegisterAggregate("groupBitOr", b.uint64Type, b.uint64Type)
	b.RegisterAggregate("groupBitAnd", b.uint64Type, b.uint64Type)
	b.RegisterAggregate("groupBitXor", b.uint64Type, b.uint64Type)
	b.RegisterAggregate("groupArrayLast", arrayOf(b.unknownType), b.uint64Type, b.unknownType)
	b.RegisterAggregate("groupArrayInsertAt", arrayOf(b.unknownType), b.unknownType, b.uint64Type)
	b.RegisterAggregate("groupArraySample", arrayOf(b.unknownType), b.uint64Type, b.unknownType)
	b.RegisterAggregate("groupArraySorted", arrayOf(b.unknownType), b.uint64Type, b.unknownType)
	b.RegisterAggregate("groupConcat", b.textType, b.textType)
	b.RegisterAggregate("groupConcat", b.textType, b.textType, b.textType)
}

// registerArgAndQuantileAggregates covers argMin and argMax and the canonical quantile
// and median family.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerArgAndQuantileAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("argMin", b.unknownType, b.unknownType, b.unknownType)
	b.RegisterAggregate("argMax", b.unknownType, b.unknownType, b.unknownType)
	b.RegisterAggregate("quantile", b.float64Type, b.float64Type)
	b.RegisterAggregate("quantileExact", b.float64Type, b.float64Type)
	b.RegisterAggregate("quantileTDigest", b.float64Type, b.float64Type)
	b.RegisterAggregate("median", b.float64Type, b.float64Type)
	b.RegisterAggregate("quantiles", arrayOf(b.float64Type), b.float64Type)
}

// registerVarianceAggregates covers variance, standard deviation, covariance and
// correlation aggregates.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerVarianceAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("varPop", b.float64Type, b.float64Type)
	b.RegisterAggregate("varSamp", b.float64Type, b.float64Type)
	b.RegisterAggregate("stddevPop", b.float64Type, b.float64Type)
	b.RegisterAggregate("stddevSamp", b.float64Type, b.float64Type)
	b.RegisterAggregate("covarPop", b.float64Type, b.float64Type, b.float64Type)
	b.RegisterAggregate("covarSamp", b.float64Type, b.float64Type, b.float64Type)
	b.RegisterAggregate("corr", b.float64Type, b.float64Type, b.float64Type)
}

// registerTopKAndRegressionAggregates covers approximate ranking and regression
// aggregates.
//
// Both the snake_case and camelCase spellings are kept because ClickHouse exposes both.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerTopKAndRegressionAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("topK", arrayOf(b.unknownType), b.unknownType)
	b.RegisterAggregate("topKWeighted", arrayOf(b.unknownType), b.unknownType, b.uint64Type)
	b.RegisterAggregate("simpleLinearRegression", arrayOf(b.float64Type), b.float64Type, b.float64Type)
	b.RegisterAggregate("stochasticLinearRegression", arrayOf(b.float64Type), b.float64Type, b.float64Type)
	b.RegisterAggregate("approx_top_k", arrayOf(b.unknownType), b.unknownType)
	b.RegisterAggregate("approx_top_sum", arrayOf(b.unknownType), b.unknownType, b.uint64Type)
}

// registerExtendedSumAggregates covers weighted, overflow-aware, Kahan-summed and
// value-with-count sum aggregates.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerExtendedSumAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("avgWeighted", b.float64Type, b.float64Type, b.float64Type)
	b.RegisterAggregate("sumWithOverflow", b.unknownType, b.unknownType)
	b.RegisterAggregate("sumKahan", b.float64Type, b.float64Type)
	b.RegisterAggregate("sumCount", b.unknownType, b.unknownType)
}

// registerExtendedQuantileAggregates covers the timing-aware, weighted, interpolated,
// BFloat16, GK and deterministic quantile variants.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerExtendedQuantileAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("quantileTiming", b.float64Type, b.float64Type)
	b.RegisterAggregate("quantileTimingWeighted", b.float64Type, b.float64Type, b.uint64Type)
	b.RegisterAggregate("quantileExactWeighted", b.float64Type, b.float64Type, b.uint64Type)
	b.RegisterAggregate("quantileInterpolatedWeighted", b.float64Type, b.float64Type, b.uint64Type)
	b.RegisterAggregate("quantileBFloat16", b.float64Type, b.float64Type)
	b.RegisterAggregate("quantileGK", b.float64Type, b.uint64Type, b.float64Type)
	b.RegisterAggregate("quantileDeterministic", b.float64Type, b.float64Type, b.uint64Type)
}

// registerHigherMomentAggregates covers population and sample kurtosis and skewness.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerHigherMomentAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("kurtPop", b.float64Type, b.float64Type)
	b.RegisterAggregate("kurtSamp", b.float64Type, b.float64Type)
	b.RegisterAggregate("skewPop", b.float64Type, b.float64Type)
	b.RegisterAggregate("skewSamp", b.float64Type, b.float64Type)
}

// registerExponentialAggregates covers the exponential moving average and time-decayed
// variants.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerExponentialAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("exponentialMovingAverage", b.float64Type, b.float64Type, b.uint64Type)
	b.RegisterAggregate("exponentialTimeDecayedAvg", b.float64Type, b.float64Type, b.float64Type, b.uint64Type)
	b.RegisterAggregate("exponentialTimeDecayedCount", b.float64Type, b.float64Type, b.uint64Type)
	b.RegisterAggregate("exponentialTimeDecayedMax", b.float64Type, b.float64Type, b.float64Type, b.uint64Type)
	b.RegisterAggregate("exponentialTimeDecayedSum", b.float64Type, b.float64Type, b.float64Type, b.uint64Type)
}

// registerSamplingAggregates covers the spatial and sampling helpers plus histogram and
// the single-value and entropy aggregates.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerSamplingAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("largestTriangleThreeBuckets", arrayOf(b.unknownType), b.uint64Type, b.float64Type, b.float64Type)
	b.RegisterAggregate("boundingRatio", b.float64Type, b.float64Type, b.float64Type)
	histogramElement := querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryStruct,
		EngineName: "Tuple",
		StructFields: []querier_dto.StructField{
			{Name: "lower", SQLType: b.float64Type},
			{Name: "upper", SQLType: b.float64Type},
			{Name: "height", SQLType: b.float64Type},
		},
	}
	b.RegisterAggregate("histogram", arrayOf(histogramElement), b.uint64Type, b.float64Type)
	b.RegisterAggregate("singleValueOrNull", b.unknownType, b.unknownType)
	b.RegisterAggregate("entropy", b.float64Type, b.unknownType)
}

// registerIntervalAndDeltaAggregates covers interval length sums and delta sums,
// including the timestamp-aligned variant.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerIntervalAndDeltaAggregates(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("intervalLengthSum", b.float64Type, b.dateTimeType, b.dateTimeType)
	b.RegisterAggregate("deltaSum", b.float64Type, b.float64Type)
	b.RegisterAggregate("deltaSumTimestamp", b.float64Type, b.float64Type, b.dateTimeType)
}

// registerDateTimeFunctions covers the date and time function family.
//
// These include now, today, yesterday, the toDate and toDateTime variants, the extraction
// functions, formatDateTime, parseDateTimeBestEffort, date arithmetic and the Joda, ISO
// and sub-second extensions. It delegates to topical helpers to keep each function within
// the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerDateTimeFunctions(b *FunctionCatalogueBuilder) {
	registerCoreDateTime(b)
	registerDateNameAndParse(b)
	registerDateTimeTruncationAndISO(b)
	registerDateTimeIntervals(b)
	registerDateTimeConstructors(b)
}

// registerCoreDateTime covers the base set of date and time functions, namely clocks,
// calendar accessors, truncation, formatting, parsing and the day-grain arithmetic
// helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerCoreDateTime(b *FunctionCatalogueBuilder) {
	b.Register("now", b.dateTimeType)

	b.Register("now64", b.dateTime64Type)
	b.Register("now64", b.dateTime64Type, b.uint64Type)
	b.Register("today", b.dateType)
	b.Register("yesterday", b.dateType)
	b.Register("toDate", b.dateType, b.textType)
	b.Register("toDate", b.dateType, b.dateTimeType)
	b.Register("toDateTime", b.dateTimeType, b.textType)
	b.Register("toDateTime", b.dateTimeType, b.uint64Type)
	b.Register("toDateTime64", b.dateTime64Type, b.textType, b.uint64Type)
	b.Register("toYear", b.uint64Type, b.dateType)
	b.Register("toMonth", b.uint64Type, b.dateType)
	b.Register("toDayOfMonth", b.uint64Type, b.dateType)
	b.Register("toDayOfWeek", b.uint64Type, b.dateType)
	b.Register("toHour", b.uint64Type, b.dateTimeType)
	b.Register("toMinute", b.uint64Type, b.dateTimeType)
	b.Register("toSecond", b.uint64Type, b.dateTimeType)
	b.Register("toUnixTimestamp", b.uint64Type, b.dateTimeType)
	b.Register("toUnixTimestamp64Milli", b.int64Type, b.dateTime64Type)
	b.Register("toStartOfYear", b.dateType, b.dateType)
	b.Register("toStartOfMonth", b.dateType, b.dateType)
	b.Register("toStartOfWeek", b.dateType, b.dateType)
	b.Register("toStartOfDay", b.dateTimeType, b.dateTimeType)
	b.Register("toStartOfHour", b.dateTimeType, b.dateTimeType)
	b.Register("toStartOfMinute", b.dateTimeType, b.dateTimeType)
	b.Register("toStartOfFiveMinute", b.dateTimeType, b.dateTimeType)
	b.Register("toStartOfFifteenMinutes", b.dateTimeType, b.dateTimeType)
	b.Register("formatDateTime", b.textType, b.dateTimeType, b.textType)
	b.Register("parseDateTime", b.dateTimeType, b.textType, b.textType)
	b.Register("parseDateTimeBestEffort", b.dateTimeType, b.textType)
	b.Register("parseDateTime64BestEffort", b.dateTime64Type, b.textType, b.uint64Type)
	b.Register("dateAdd", b.dateTimeType, b.textType, b.int64Type, b.dateTimeType)
	b.Register("dateSub", b.dateTimeType, b.textType, b.int64Type, b.dateTimeType)
	b.Register("dateDiff", b.int64Type, b.textType, b.dateTimeType, b.dateTimeType)
	b.Register("addDays", b.dateTimeType, b.dateTimeType, b.int64Type)
	b.Register("addHours", b.dateTimeType, b.dateTimeType, b.int64Type)
	b.Register("addMinutes", b.dateTimeType, b.dateTimeType, b.int64Type)
	b.Register("addSeconds", b.dateTimeType, b.dateTimeType, b.int64Type)
	b.Register("subtractDays", b.dateTimeType, b.dateTimeType, b.int64Type)
	b.Register("subtractHours", b.dateTimeType, b.dateTimeType, b.int64Type)
	b.Register("toTimeZone", b.dateTimeType, b.dateTimeType, b.textType)
	b.Register("timezone", b.textType)
}

// registerDateNameAndParse covers the naming helpers dateName and monthName, the
// Joda-syntax format and parse functions, and the OrNull and OrZero safety variants of
// the best-effort parsers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerDateNameAndParse(b *FunctionCatalogueBuilder) {
	b.Register("dateName", b.textType, b.textType, b.dateTimeType)
	b.Register("age", b.int64Type, b.textType, b.dateTimeType, b.dateTimeType)
	b.Register("monthName", b.textType, b.dateType)
	b.Register("formatDateTimeInJodaSyntax", b.textType, b.dateTimeType, b.textType)
	b.Register("formatDateTimeInJodaSyntax", b.textType, b.dateTimeType, b.textType, b.textType)
	b.Register("parseDateTimeBestEffortOrNull", b.dateTimeType, b.textType)
	b.Register("parseDateTimeBestEffortOrZero", b.dateTimeType, b.textType)
	b.Register("parseDateTimeInJodaSyntax", b.dateTimeType, b.textType, b.textType)
	b.Register("parseDateTimeInJodaSyntaxOrNull", b.dateTimeType, b.textType, b.textType)
	b.Register("parseDateTimeInJodaSyntaxOrZero", b.dateTimeType, b.textType, b.textType)
	b.Register("parseDateTime32BestEffort", b.dateTimeType, b.textType)
	b.Register("parseDateTime32BestEffortOrNull", b.dateTimeType, b.textType)
	b.Register("parseDateTime32BestEffortOrZero", b.dateTimeType, b.textType)
	b.Register("parseDateTime64BestEffortOrNull", b.dateTime64Type, b.textType, b.uint64Type)
	b.Register("parseDateTime64BestEffortOrZero", b.dateTime64Type, b.textType, b.uint64Type)
}

// registerDateTimeTruncationAndISO covers the extended truncation helpers and the ISO
// calendar accessors.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerDateTimeTruncationAndISO(b *FunctionCatalogueBuilder) {
	b.Register("toStartOfQuarter", b.dateType, b.dateType)
	b.Register("toStartOfTenMinutes", b.dateTimeType, b.dateTimeType)
	b.Register("toStartOfInterval", b.dateTimeType, b.dateTimeType, b.textType)
	b.Register("toQuarter", b.uint64Type, b.dateType)
	b.Register("toWeek", b.uint64Type, b.dateType)
	b.Register("toWeek", b.uint64Type, b.dateType, b.uint64Type)
	b.Register("toISOWeek", b.uint64Type, b.dateType)
	b.Register("toISOYear", b.uint64Type, b.dateType)
	b.Register("toYearWeek", b.uint64Type, b.dateType)
	b.Register("toYearWeek", b.uint64Type, b.dateType, b.uint64Type)
	b.Register("toDayOfYear", b.uint64Type, b.dateType)
}

// registerDateTimeIntervals covers calendar and sub-second interval adders, subtractors
// and the field-change helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerDateTimeIntervals(b *FunctionCatalogueBuilder) {
	for _, name := range []string{
		"addMonths", "addQuarters", "addWeeks", "addYears",
		"addMilliseconds", "addMicroseconds", "addNanoseconds",
		"subtractMonths", "subtractQuarters", "subtractWeeks", "subtractYears",
		"subtractSeconds", "subtractMilliseconds", "subtractMicroseconds", "subtractNanoseconds",
	} {
		b.Register(name, b.dateTimeType, b.dateTimeType, b.int64Type)
	}
	for _, name := range []string{"changeYear", "changeMonth", "changeDay", "changeHour", "changeMinute", "changeSecond"} {
		b.Register(name, b.dateTimeType, b.dateTimeType, b.int64Type)
	}
}

// registerDateTimeConstructors covers the Date, DateTime and DateTime64 constructors and
// the sub-second Unix conversion helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerDateTimeConstructors(b *FunctionCatalogueBuilder) {
	b.Register("dateTrunc", b.dateTimeType, b.textType, b.dateTimeType)
	b.Register("dateTrunc", b.dateTimeType, b.textType, b.dateTimeType, b.textType)
	b.Register("makeDate", b.dateType, b.uint64Type, b.uint64Type, b.uint64Type)
	b.Register("makeDate32", b.dateType, b.uint64Type, b.uint64Type, b.uint64Type)
	b.Register("makeDateTime", b.dateTimeType, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type)
	b.Register("makeDateTime", b.dateTimeType, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type, b.textType)
	b.Register("makeDateTime64", b.dateTime64Type, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type, b.uint64Type)
	b.Register("toDate32", b.dateType, b.unknownType)
	b.Register("fromUnixTimestamp64Milli", b.dateTime64Type, b.int64Type)
	b.Register("fromUnixTimestamp64Micro", b.dateTime64Type, b.int64Type)
	b.Register("fromUnixTimestamp64Nano", b.dateTime64Type, b.int64Type)
	b.Register("toUnixTimestamp64Micro", b.int64Type, b.dateTime64Type)
	b.Register("toUnixTimestamp64Nano", b.int64Type, b.dateTime64Type)
}

// registerStringFunctions covers the string function family.
//
// These include length, lower and upper, substring, concat, position, the replace family,
// the splitBy helpers, format, trim, regex match and extract, the UTF-8 variants,
// padding, the regex helpers and the capitalisation and ASCII helpers. It delegates to
// topical helpers so each function fits the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerStringFunctions(b *FunctionCatalogueBuilder) {
	registerCoreStringFunctions(b)
	registerStringUTF8Functions(b)
	registerStringRegexHelpers(b)
}

// registerCoreStringFunctions covers the base string helpers, namely case, concat,
// substring, position, replace, split, format, trim, match and the LIKE, startsWith and
// endsWith family.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerCoreStringFunctions(b *FunctionCatalogueBuilder) {
	b.Register("length", b.uint64Type, b.textType)
	b.Register("empty", b.boolType, b.textType)
	b.Register("notEmpty", b.boolType, b.textType)
	b.Register("lower", b.textType, b.textType)
	b.Register("upper", b.textType, b.textType)
	b.Register("lowerUTF8", b.textType, b.textType)
	b.Register("upperUTF8", b.textType, b.textType)
	b.Register("reverse", b.textType, b.textType)
	b.RegisterVariadic("concat", b.textType, 1, b.textType)
	b.RegisterVariadic("concatWithSeparator", b.textType, 2, b.textType, b.textType)
	b.Register("substring", b.textType, b.textType, b.uint64Type, b.uint64Type)
	b.Register("substring", b.textType, b.textType, b.uint64Type)
	b.Register("substringUTF8", b.textType, b.textType, b.uint64Type, b.uint64Type)
	b.Register("position", b.uint64Type, b.textType, b.textType)
	b.Register("positionUTF8", b.uint64Type, b.textType, b.textType)
	b.Register("replaceAll", b.textType, b.textType, b.textType, b.textType)
	b.Register("replaceOne", b.textType, b.textType, b.textType, b.textType)
	b.Register("replaceRegexpAll", b.textType, b.textType, b.textType, b.textType)
	b.Register("replaceRegexpOne", b.textType, b.textType, b.textType, b.textType)
	b.Register("splitByChar", arrayOf(b.textType), b.textType, b.textType)
	b.Register("splitByString", arrayOf(b.textType), b.textType, b.textType)
	b.Register("splitByWhitespace", arrayOf(b.textType), b.textType)
	b.Register("arrayStringConcat", b.textType, arrayOf(b.textType))
	b.Register("arrayStringConcat", b.textType, arrayOf(b.textType), b.textType)
	b.Register("format", b.textType, b.textType, b.unknownType)
	b.Register("printf", b.textType, b.textType)
	b.Register("trimBoth", b.textType, b.textType)
	b.Register("trimLeft", b.textType, b.textType)
	b.Register("trimRight", b.textType, b.textType)
	b.Register("like", b.boolType, b.textType, b.textType)
	b.Register("notLike", b.boolType, b.textType, b.textType)
	b.Register("ilike", b.boolType, b.textType, b.textType)
	b.Register("match", b.boolType, b.textType, b.textType)
	b.Register("extract", b.textType, b.textType, b.textType)
	b.Register("extractAll", arrayOf(b.textType), b.textType, b.textType)
	b.Register("startsWith", b.boolType, b.textType, b.textType)
	b.Register("endsWith", b.boolType, b.textType, b.textType)
}

// registerStringUTF8Functions covers the UTF-8 length, slicing and padding helpers.
//
// ClickHouse keeps the UTF-8 variants distinct from the byte variants because UTF-8
// indexing is by codepoint not byte.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerStringUTF8Functions(b *FunctionCatalogueBuilder) {
	b.Register("lengthUTF8", b.uint64Type, b.textType)
	b.Register("leftUTF8", b.textType, b.textType, b.int64Type)
	b.Register("rightUTF8", b.textType, b.textType, b.int64Type)
	b.Register("left", b.textType, b.textType, b.int64Type)
	b.Register("right", b.textType, b.textType, b.int64Type)
	b.Register("leftPad", b.textType, b.textType, b.uint64Type)
	b.Register("leftPad", b.textType, b.textType, b.uint64Type, b.textType)
	b.Register("rightPad", b.textType, b.textType, b.uint64Type)
	b.Register("rightPad", b.textType, b.textType, b.uint64Type, b.textType)
	b.Register("leftPadUTF8", b.textType, b.textType, b.uint64Type)
	b.Register("leftPadUTF8", b.textType, b.textType, b.uint64Type, b.textType)
	b.Register("rightPadUTF8", b.textType, b.textType, b.uint64Type)
	b.Register("rightPadUTF8", b.textType, b.textType, b.uint64Type, b.textType)
	b.Register("repeat", b.textType, b.textType, b.uint64Type)
	b.Register("space", b.textType, b.uint64Type)
	b.Register("reverseUTF8", b.textType, b.textType)
}

// registerStringRegexHelpers covers the additional regex helpers, the capitalisation
// helpers and the ASCII and UTF-8 validity tests beyond the base extract and replace
// family.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerStringRegexHelpers(b *FunctionCatalogueBuilder) {
	b.Register("regexpExtract", b.textType, b.textType, b.textType)
	b.Register("regexpExtract", b.textType, b.textType, b.textType, b.uint64Type)
	b.Register("regexpQuoteMeta", b.textType, b.textType)
	b.Register("initcap", b.textType, b.textType)
	b.Register("initcapUTF8", b.textType, b.textType)
	b.Register("ascii", b.int64Type, b.textType)
	b.Register("isValidUTF8", b.boolType, b.textType)
	b.Register("notILike", b.boolType, b.textType, b.textType)
}

// registerStringSearchFunctions covers the token, multi-pattern and case-insensitive
// search families.
//
// It is kept separate from registerStringFunctions because the catalogue is large enough
// that readers benefit from a topical split.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerStringSearchFunctions(b *FunctionCatalogueBuilder) {
	b.Register("hasToken", b.boolType, b.textType, b.textType)
	b.Register("hasTokenCaseInsensitive", b.boolType, b.textType, b.textType)
	b.Register("hasTokenOrNull", b.boolType, b.textType, b.textType)

	b.Register("multiSearchAny", b.boolType, b.textType, arrayOf(b.textType))
	b.Register("multiSearchAnyCaseInsensitive", b.boolType, b.textType, arrayOf(b.textType))
	b.Register("multiMatchAny", b.boolType, b.textType, arrayOf(b.textType))
	b.Register("multiMatchAllIndices", arrayOf(b.uint64Type), b.textType, arrayOf(b.textType))

	b.Register("countSubstrings", b.uint64Type, b.textType, b.textType)
	b.Register("countMatches", b.uint64Type, b.textType, b.textType)

	b.Register("positionCaseInsensitive", b.uint64Type, b.textType, b.textType)
	b.Register("positionCaseInsensitiveUTF8", b.uint64Type, b.textType, b.textType)
	b.Register("locate", b.uint64Type, b.textType, b.textType)
}

// registerArrayFunctions covers the array library.
//
// These include arrayMap, arrayFilter, arrayReduce, arrayFold, arrayJoin, arrayEnumerate,
// arrayDistinct, arrayConcat, arraySlice, the push and pop helpers, resize, element
// access, has, hasAll and hasAny, indexOf, sort, uniq, difference, cumSum and compact.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerArrayFunctions(b *FunctionCatalogueBuilder) {
	b.Register("arrayMap", arrayOf(b.unknownType), b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayFilter", arrayOf(b.unknownType), b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayReduce", b.unknownType, b.textType, arrayOf(b.unknownType))
	b.Register("arrayFold", b.unknownType, b.unknownType, arrayOf(b.unknownType), b.unknownType)

	b.Register("arrayEnumerate", arrayOf(b.uint64Type), arrayOf(b.unknownType))
	b.Register("arrayDistinct", arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.RegisterVariadic("arrayConcat", arrayOf(b.unknownType), 1, arrayOf(b.unknownType))
	b.Register("arraySlice", arrayOf(b.unknownType), arrayOf(b.unknownType), b.int64Type, b.int64Type)
	b.Register("arraySlice", arrayOf(b.unknownType), arrayOf(b.unknownType), b.int64Type)
	b.Register("arrayPushBack", arrayOf(b.unknownType), arrayOf(b.unknownType), b.unknownType)
	b.Register("arrayPushFront", arrayOf(b.unknownType), arrayOf(b.unknownType), b.unknownType)
	b.Register("arrayPopBack", arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arrayPopFront", arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arrayResize", arrayOf(b.unknownType), arrayOf(b.unknownType), b.int64Type)
	b.Register("arrayElement", b.unknownType, arrayOf(b.unknownType), b.int64Type)
	b.Register("has", b.boolType, arrayOf(b.unknownType), b.unknownType)
	b.Register("hasAll", b.boolType, arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("hasAny", b.boolType, arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("indexOf", b.uint64Type, arrayOf(b.unknownType), b.unknownType)
	b.Register("arraySort", arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arrayReverseSort", arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arrayUniq", b.uint64Type, arrayOf(b.unknownType))
	b.Register("arrayDifference", arrayOf(b.float64Type), arrayOf(b.float64Type))
	b.Register("arrayCumSum", arrayOf(b.float64Type), arrayOf(b.float64Type))
	b.Register("arrayCompact", arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.RegisterVariadic("array", arrayOf(b.unknownType), 0, b.unknownType)
	b.Register("range", arrayOf(b.uint64Type), b.uint64Type)
	b.Register("range", arrayOf(b.uint64Type), b.uint64Type, b.uint64Type)
}

// registerTupleAndMapFunctions covers tuple constructors, element access, map
// constructors, key and value retrieval and the mapAdd, mapSubtract, mapFilter and
// mapApply helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerTupleAndMapFunctions(b *FunctionCatalogueBuilder) {
	b.RegisterVariadic("tuple", b.unknownType, 1, b.unknownType)
	b.Register("tupleElement", b.unknownType, b.unknownType, b.int64Type)
	b.Register("untuple", b.unknownType, b.unknownType)
	b.RegisterVariadic("map", b.unknownType, 0, b.unknownType)
	b.Register("mapKeys", arrayOf(b.unknownType), b.unknownType)
	b.Register("mapValues", arrayOf(b.unknownType), b.unknownType)
	b.Register("mapContains", b.boolType, b.unknownType, b.unknownType)
	b.Register("mapAdd", b.unknownType, b.unknownType, b.unknownType)
	b.Register("mapSubtract", b.unknownType, b.unknownType, b.unknownType)
	b.Register("mapFilter", b.unknownType, b.unknownType, b.unknownType)
	b.Register("mapApply", b.unknownType, b.unknownType, b.unknownType)
}

// registerTypeConversionFunctions covers CAST, accurateCast, the per-target-type toXxx
// helpers and the OrNull, OrZero and OrDefault safety variants for every integer, float,
// decimal and date target.
//
// It delegates to topical helpers so each function stays within the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerTypeConversionFunctions(b *FunctionCatalogueBuilder) {
	registerBaseConversions(b)
	registerIntegerSafetyConversions(b)
	registerFloatAndDateSafetyConversions(b)
	registerDecimalConversions(b)
}

// registerBaseConversions covers CAST, accurateCast, the integer, float, string and IP
// base conversions and the bool conversion.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerBaseConversions(b *FunctionCatalogueBuilder) {
	b.Register("CAST", b.unknownType, b.unknownType, b.textType)
	b.Register("accurateCast", b.unknownType, b.unknownType, b.textType)
	b.Register("accurateCastOrNull", b.unknownType, b.unknownType, b.textType)
	b.Register("accurateCastOrDefault", b.unknownType, b.unknownType, b.unknownType)
	for _, name := range signedIntegerConversionNames() {
		b.RegisterNullPropagating(name, b.int64Type, b.unknownType)
	}
	for _, name := range unsignedIntegerConversionNames() {
		b.RegisterNullPropagating(name, b.uint64Type, b.unknownType)
	}
	b.RegisterNullPropagating("toFloat32", b.float32Type, b.unknownType)
	b.RegisterNullPropagating("toFloat64", b.float64Type, b.unknownType)
	b.RegisterNullPropagating("toDecimal128", b.decimal128Type, b.unknownType, b.int64Type)
	b.RegisterNullPropagating("toString", b.textType, b.unknownType)
	b.RegisterNullPropagating("toFixedString", b.textType, b.textType, b.uint64Type)
	b.RegisterNullPropagating("toUUID", b.uuidType, b.textType)
	b.RegisterNullPropagating("toIPv4", ipv4Type(), b.textType)
	b.RegisterNullPropagating("toIPv6", ipv6Type(), b.textType)
	b.RegisterNullPropagating("toBool", b.boolType, b.unknownType)
}

// signedIntegerConversionNames returns the canonical ordering of signed-integer `toXxx`
// base names.
//
// It is pulled out so callers can reuse the same list when building the safety variants.
//
// Returns []string which is the ordered list of signed-integer conversion names.
func signedIntegerConversionNames() []string {
	return []string{"toInt8", "toInt16", "toInt32", "toInt64", "toInt128", "toInt256"}
}

// unsignedIntegerConversionNames returns the canonical ordering of unsigned-integer
// `toXxx` base names.
//
// Returns []string which is the ordered list of unsigned-integer conversion names.
func unsignedIntegerConversionNames() []string {
	return []string{"toUInt8", "toUInt16", "toUInt32", "toUInt64", "toUInt128", "toUInt256"}
}

// registerIntegerSafetyConversions registers OrNull, OrZero and OrDefault variants for
// every signed and unsigned integer width.
//
// Each returns the same numeric category as the corresponding `toXxx` but signals failure
// differently, where OrNull returns Nullable(T), OrZero returns the zero value and
// OrDefault returns the type's default.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerIntegerSafetyConversions(b *FunctionCatalogueBuilder) {
	for _, base := range signedIntegerConversionNames() {
		for _, suffix := range safetyConversionSuffixes {
			b.Register(base+suffix, b.int64Type, b.unknownType)
		}
	}
	for _, base := range unsignedIntegerConversionNames() {
		for _, suffix := range safetyConversionSuffixes {
			b.Register(base+suffix, b.uint64Type, b.unknownType)
		}
	}
}

// registerFloatAndDateSafetyConversions registers the OrNull, OrZero and OrDefault
// variants for float and date or datetime conversions, plus the US-locale best-effort
// parser safety variants.
//
// The other parseDateTimeBestEffort safety spellings live in registerDateNameAndParse to
// keep the topical grouping intact and to avoid emitting duplicate signatures for
// identical name and argument pairs.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerFloatAndDateSafetyConversions(b *FunctionCatalogueBuilder) {
	for _, suffix := range safetyConversionSuffixes {
		b.Register("toFloat32"+suffix, b.float32Type, b.unknownType)
		b.Register("toFloat64"+suffix, b.float64Type, b.unknownType)
	}
	for _, suffix := range safetyConversionSuffixes {
		b.Register("toDate"+suffix, b.dateType, b.unknownType)
		b.Register("toDateTime"+suffix, b.dateTimeType, b.unknownType)
		b.Register("toDateTime64"+suffix, b.dateTime64Type, b.unknownType, b.uint64Type)
	}
	b.Register("parseDateTimeBestEffortUSOrNull", b.dateTimeType, b.textType)
	b.Register("parseDateTimeBestEffortUSOrZero", b.dateTimeType, b.textType)
}

// registerDecimalConversions registers the toDecimalNN family across every supported
// width plus the safety variants.
//
// The list is indexed rather than ranged-by-value because each entry holds a sizeable
// SQLType that range-by-value would copy on every iteration.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerDecimalConversions(b *FunctionCatalogueBuilder) {
	decimalEntries := []struct {
		name        string
		decimalType querier_dto.SQLType
	}{
		{name: "toDecimal32", decimalType: querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "Decimal32"}},
		{name: "toDecimal64", decimalType: querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "Decimal64"}},
		{name: "toDecimal128", decimalType: b.decimal128Type},
		{name: "toDecimal256", decimalType: querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "Decimal256"}},
	}
	for index := range decimalEntries {
		entry := &decimalEntries[index]

		if entry.name != "toDecimal128" {
			b.Register(entry.name, entry.decimalType, b.unknownType, b.int64Type)
		}
		for _, suffix := range safetyConversionSuffixes {
			b.Register(entry.name+suffix, entry.decimalType, b.unknownType, b.int64Type)
		}
	}
}

// registerMathFunctions covers numeric helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerMathFunctions(b *FunctionCatalogueBuilder) {
	b.Register("abs", b.float64Type, b.float64Type)
	b.Register("abs", b.int64Type, b.int64Type)
	b.Register("round", b.float64Type, b.float64Type)
	b.Register("round", b.float64Type, b.float64Type, b.int64Type)
	b.Register("roundBankers", b.float64Type, b.float64Type)
	b.Register("floor", b.float64Type, b.float64Type)
	b.Register("ceil", b.float64Type, b.float64Type)
	b.Register("truncate", b.float64Type, b.float64Type)
	b.Register("mod", b.int64Type, b.int64Type, b.int64Type)
	b.Register("pow", b.float64Type, b.float64Type, b.float64Type)
	b.Register("sqrt", b.float64Type, b.float64Type)
	b.Register("exp", b.float64Type, b.float64Type)
	b.Register("log", b.float64Type, b.float64Type)
	b.Register("log10", b.float64Type, b.float64Type)
	b.Register("log2", b.float64Type, b.float64Type)
	b.Register("sin", b.float64Type, b.float64Type)
	b.Register("cos", b.float64Type, b.float64Type)
	b.Register("tan", b.float64Type, b.float64Type)
	b.Register("asin", b.float64Type, b.float64Type)
	b.Register("acos", b.float64Type, b.float64Type)
	b.Register("atan", b.float64Type, b.float64Type)
	b.Register("atan2", b.float64Type, b.float64Type, b.float64Type)
	b.Register("greatest", b.unknownType, b.unknownType, b.unknownType)
	b.Register("least", b.unknownType, b.unknownType, b.unknownType)
	b.Register("intDiv", b.int64Type, b.int64Type, b.int64Type)
	b.Register("intDivOrZero", b.int64Type, b.int64Type, b.int64Type)
	b.Register("modulo", b.int64Type, b.int64Type, b.int64Type)

	b.Register("pi", b.float64Type)
	b.Register("e", b.float64Type)

	b.Register("clamp", b.unknownType, b.unknownType, b.unknownType, b.unknownType)
	b.Register("sign", b.int64Type, b.float64Type)
	b.Register("factorial", b.uint64Type, b.uint64Type)
	b.Register("cbrt", b.float64Type, b.float64Type)
	b.Register("widthBucket", b.uint64Type, b.float64Type, b.float64Type, b.float64Type, b.uint64Type)
}

// registerLogicalFunctions covers boolean operators and case helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerLogicalFunctions(b *FunctionCatalogueBuilder) {
	b.RegisterVariadic("and", b.boolType, 2, b.boolType)
	b.RegisterVariadic("or", b.boolType, 2, b.boolType)
	b.Register("not", b.boolType, b.boolType)
	b.Register("xor", b.boolType, b.boolType, b.boolType)
}

// registerHashingFunctions covers the MD5, SHA, cityHash, xxHash and murmurHash variants.
//
// MD5 and the SHA family return FixedString(N) where N is the digest byte length.
// Modelling them as FixedString rather than String lets the emitter pick the precise Go
// type such as [16]byte and prevents truncation when the bytes contain NULs.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerHashingFunctions(b *FunctionCatalogueBuilder) {
	b.Register("MD5", fixedStringType(md5DigestBytes), b.textType)
	b.Register("SHA1", fixedStringType(sha1DigestBytes), b.textType)
	b.Register("SHA224", fixedStringType(sha224DigestBytes), b.textType)
	b.Register("SHA256", fixedStringType(sha256DigestBytes), b.textType)
	b.Register("SHA384", fixedStringType(sha384DigestBytes), b.textType)
	b.Register("SHA512", fixedStringType(sha512DigestBytes), b.textType)
	for _, name := range []string{"cityHash64", "farmHash64", "intHash64", "xxHash64", "murmurHash2_64", "murmurHash3_64"} {
		b.Register(name, b.uint64Type, b.unknownType)
	}
	b.Register("xxHash32", b.uint64Type, b.unknownType)
	b.Register("murmurHash3_32", b.uint64Type, b.unknownType)
	b.Register("halfMD5", b.uint64Type, b.unknownType)
	b.Register("sipHash64", b.uint64Type, b.unknownType)
}

// fixedStringType constructs a FixedString(N) SQLType.
//
// Takes length (int) which is the fixed byte length N.
//
// Returns querier_dto.SQLType which is the FixedString type carrying that length.
func fixedStringType(length int) querier_dto.SQLType {
	return querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryText,
		EngineName: "FixedString",
		Length:     new(length),
	}
}

// registerURLAndIPFunctions covers URL parsing helpers and IPv4 and IPv6 string
// conversion.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerURLAndIPFunctions(b *FunctionCatalogueBuilder) {
	for _, name := range []string{"protocol", "domain", "domainWithoutWWW", "topLevelDomain", "path", "pathFull", "queryString", "fragment"} {
		b.Register(name, b.textType, b.textType)
	}

	b.Register("extractURLParameter", b.textType, b.textType, b.textType)
	b.Register("IPv4ToString", b.textType, ipv4Type())
	b.Register("IPv6ToString", b.textType, ipv6Type())
	b.Register("IPv4StringToNum", b.uint32Type, b.textType)
	b.Register("IPv6StringToNum", b.textType, b.textType)
	b.Register("IPv4NumToString", b.textType, b.uint64Type)
}

// registerJSONFunctions covers the JSONExtract helpers, the JSON inspection helpers, the
// simpleJSON family for fast unindexed access and the SQL JSON path helpers JSON_VALUE,
// JSON_QUERY and JSON_EXISTS.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerJSONFunctions(b *FunctionCatalogueBuilder) {
	b.Register("JSONExtract", b.unknownType, b.textType, b.textType, b.textType)
	b.Register("JSONExtractString", b.textType, b.textType, b.textType)
	b.Register("JSONExtractInt", b.int64Type, b.textType, b.textType)
	b.Register("JSONExtractUInt", b.uint64Type, b.textType, b.textType)
	b.Register("JSONExtractFloat", b.float64Type, b.textType, b.textType)
	b.Register("JSONExtractBool", b.boolType, b.textType, b.textType)
	b.Register("JSONExtractRaw", b.textType, b.textType, b.textType)
	b.Register("JSONExtractArrayRaw", arrayOf(b.textType), b.textType, b.textType)
	b.Register("JSONExtractKeys", arrayOf(b.textType), b.textType, b.textType)
	b.Register("JSONHas", b.boolType, b.textType, b.textType)
	b.Register("JSONKey", b.textType, b.textType, b.uint64Type)
	b.Register("JSONLength", b.uint64Type, b.textType, b.textType)
	b.Register("JSONType", b.textType, b.textType, b.textType)
	b.Register("isValidJSON", b.boolType, b.textType)

	b.Register("simpleJSONExtractString", b.textType, b.textType, b.textType)
	b.Register("simpleJSONExtractInt", b.int64Type, b.textType, b.textType)
	b.Register("simpleJSONExtractUInt", b.uint64Type, b.textType, b.textType)
	b.Register("simpleJSONExtractFloat", b.float64Type, b.textType, b.textType)
	b.Register("simpleJSONExtractBool", b.boolType, b.textType, b.textType)
	b.Register("simpleJSONExtractRaw", b.textType, b.textType, b.textType)
	b.Register("simpleJSONHas", b.boolType, b.textType, b.textType)

	b.Register("JSON_VALUE", b.textType, b.textType, b.textType)
	b.Register("JSON_QUERY", b.textType, b.textType, b.textType)
	b.Register("JSON_EXISTS", b.boolType, b.textType, b.textType)
}

// registerConditionalFunctions covers if, multiIf, nullIf, assumeNotNull, isNull,
// isNotNull, coalesce and ifNull.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerConditionalFunctions(b *FunctionCatalogueBuilder) {
	b.Register("if", b.unknownType, b.boolType, b.unknownType, b.unknownType)
	b.RegisterVariadic("multiIf", b.unknownType, multiIfMinArgs, b.boolType)
	b.Register("nullIf", b.unknownType, b.unknownType, b.unknownType)
	b.Register("assumeNotNull", b.unknownType, b.unknownType)
	b.Register("isNull", b.boolType, b.unknownType)
	b.Register("isNotNull", b.boolType, b.unknownType)
	b.RegisterVariadic("coalesce", b.unknownType, 1, b.unknownType)
	b.Register("ifNull", b.unknownType, b.unknownType, b.unknownType)
}

// registerSystemFunctions covers version, hostName, currentDatabase and currentUser.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerSystemFunctions(b *FunctionCatalogueBuilder) {
	b.Register("version", b.textType)
	b.Register("hostName", b.textType)
	b.Register("currentDatabase", b.textType)
	b.Register("currentUser", b.textType)
	b.Register("user", b.textType)
	b.Register("uptime", b.uint64Type)
	b.Register("generateUUIDv4", b.uuidType)

	b.Register("generateUUIDv7", b.uuidType)
	b.Register("generateUUIDv7", b.uuidType, b.unknownType)

	b.Register("generateSnowflakeID", b.uint64Type)
	b.Register("generateSnowflakeID", b.uint64Type, b.uint64Type)
	b.Register("snowflakeIDToDateTime", b.dateTimeType, b.uint64Type)
	b.Register("snowflakeIDToDateTime64", b.dateTime64Type, b.uint64Type)
}

// registerWindowFunctions covers row_number, rank, dense_rank, lag, lead, first_value,
// last_value, nth_value, percent_rank and cume_dist.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerWindowFunctions(b *FunctionCatalogueBuilder) {
	b.Register("row_number", b.uint64Type)
	b.Register("rank", b.uint64Type)
	b.Register("dense_rank", b.uint64Type)
	b.Register("denseRank", b.uint64Type)
	b.Register("percent_rank", b.float64Type)
	b.Register("percentRank", b.float64Type)
	b.Register("cume_dist", b.float64Type)
	b.Register("ntile", b.uint64Type, b.uint64Type)

	b.Register("lag", b.unknownType, b.unknownType)
	b.Register("lag", b.unknownType, b.unknownType, b.uint64Type)
	b.Register("lag", b.unknownType, b.unknownType, b.uint64Type, b.unknownType)
	b.Register("lead", b.unknownType, b.unknownType)
	b.Register("lead", b.unknownType, b.unknownType, b.uint64Type)
	b.Register("lead", b.unknownType, b.unknownType, b.uint64Type, b.unknownType)

	b.Register("lagInFrame", b.unknownType, b.unknownType)
	b.Register("lagInFrame", b.unknownType, b.unknownType, b.uint64Type)
	b.Register("lagInFrame", b.unknownType, b.unknownType, b.uint64Type, b.unknownType)
	b.Register("leadInFrame", b.unknownType, b.unknownType)
	b.Register("leadInFrame", b.unknownType, b.unknownType, b.uint64Type)
	b.Register("leadInFrame", b.unknownType, b.unknownType, b.uint64Type, b.unknownType)
	b.Register("first_value", b.unknownType, b.unknownType)
	b.Register("last_value", b.unknownType, b.unknownType)
	b.Register("nth_value", b.unknownType, b.unknownType, b.uint64Type)
}

// arrayOf wraps the given element type in an Array(T) shape.
//
// Takes element (querier_dto.SQLType) which is the element type T to wrap.
//
// Returns querier_dto.SQLType which is the Array type carrying that element type.
func arrayOf(element querier_dto.SQLType) querier_dto.SQLType {
	return querier_dto.SQLType{
		Category:    querier_dto.TypeCategoryArray,
		EngineName:  "Array",
		ElementType: new(element),
	}
}

// registerBitwiseFunctions covers function-call equivalents of the bitwise operators.
//
// These are useful in expressions that combine bit manipulation with other helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerBitwiseFunctions(b *FunctionCatalogueBuilder) {
	for _, name := range []string{"bitAnd", "bitOr", "bitXor", "bitShiftLeft", "bitShiftRight", "bitRotateLeft", "bitRotateRight"} {
		b.Register(name, b.int64Type, b.int64Type, b.int64Type)
	}
	b.Register("bitNot", b.int64Type, b.int64Type)
	b.Register("bitTest", b.boolType, b.int64Type, b.int64Type)
	b.Register("bitTestAll", b.boolType, b.int64Type, b.int64Type)
	b.Register("bitTestAny", b.boolType, b.int64Type, b.int64Type)
	b.Register("bitCount", b.uint64Type, b.int64Type)
	b.Register("bitPositions", arrayOf(b.uint64Type), b.int64Type)
}

// registerComparisonFunctions exposes the function-call equivalents of the comparison
// operators.
//
// Some sites in user code use the function form for readability.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerComparisonFunctions(b *FunctionCatalogueBuilder) {
	for _, name := range []string{"equals", "notEquals", "less", "greater", "lessOrEquals", "greaterOrEquals"} {
		b.Register(name, b.boolType, b.unknownType, b.unknownType)
	}
	b.Register("isFinite", b.boolType, b.float64Type)
	b.Register("isInfinite", b.boolType, b.float64Type)
	b.Register("isNaN", b.boolType, b.float64Type)
	b.Register("isZeroOrNull", b.boolType, b.unknownType)
}

// registerRandomFunctions covers the random-number generator family.
//
// ClickHouse exposes many variants for different distributions.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerRandomFunctions(b *FunctionCatalogueBuilder) {
	b.Register("rand", b.uint64Type)
	b.Register("rand32", b.uint64Type)
	b.Register("rand64", b.uint64Type)
	b.Register("randConstant", b.uint64Type)
	b.Register("randCanonical", b.float64Type)
	b.Register("randUniform", b.float64Type, b.float64Type, b.float64Type)
	b.Register("randNormal", b.float64Type, b.float64Type, b.float64Type)
	b.Register("randLogNormal", b.float64Type, b.float64Type, b.float64Type)
	b.Register("randExponential", b.float64Type, b.float64Type)
	b.Register("randChiSquared", b.float64Type, b.float64Type)
	b.Register("randStudentT", b.float64Type, b.float64Type)
	b.Register("randFisherF", b.float64Type, b.float64Type, b.float64Type)
	b.Register("randPoisson", b.uint64Type, b.float64Type)
	b.Register("randBinomial", b.uint64Type, b.uint64Type, b.float64Type)
	b.Register("randBernoulli", b.uint64Type, b.float64Type)
}

// registerCryptoFunctions covers encryption and encoding helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerCryptoFunctions(b *FunctionCatalogueBuilder) {
	b.Register("encrypt", b.textType, b.textType, b.textType, b.textType)
	b.Register("decrypt", b.textType, b.textType, b.textType, b.textType)
	b.Register("aes_encrypt_mysql", b.textType, b.textType, b.textType, b.textType)
	b.Register("aes_decrypt_mysql", b.textType, b.textType, b.textType, b.textType)
	b.Register("base64Encode", b.textType, b.textType)
	b.Register("base64Decode", b.textType, b.textType)
	b.Register("tryBase64Decode", b.textType, b.textType)
	b.Register("hex", b.textType, b.unknownType)
	b.Register("unhex", b.textType, b.textType)
	b.Register("bin", b.textType, b.unknownType)
	b.Register("unbin", b.textType, b.textType)
	b.Register("HMAC", b.textType, b.textType, b.textType, b.textType)
}

// registerGeoFunctions covers geographical helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerGeoFunctions(b *FunctionCatalogueBuilder) {
	b.Register("greatCircleDistance", b.float64Type, b.float64Type, b.float64Type, b.float64Type, b.float64Type)
	b.Register("geoDistance", b.float64Type, b.float64Type, b.float64Type, b.float64Type, b.float64Type)
	b.Register("pointInEllipses", b.boolType, b.float64Type, b.float64Type, b.float64Type, b.float64Type, b.float64Type, b.float64Type)
	b.Register("pointInPolygon", b.boolType, b.unknownType, b.unknownType)
	b.Register("geoToH3", b.uint64Type, b.float64Type, b.float64Type, b.uint64Type)
	b.Register("h3ToGeo", b.unknownType, b.uint64Type)
	b.Register("h3ToParent", b.uint64Type, b.uint64Type, b.uint64Type)
	b.Register("h3GetResolution", b.uint64Type, b.uint64Type)
	b.Register("h3IsValid", b.boolType, b.uint64Type)
	b.Register("h3kRing", arrayOf(b.uint64Type), b.uint64Type, b.uint64Type)
	b.Register("geohashEncode", b.textType, b.float64Type, b.float64Type, b.uint64Type)
	b.Register("geohashDecode", b.unknownType, b.textType)
	b.Register("geohashesInBox", arrayOf(b.textType), b.float64Type, b.float64Type, b.float64Type, b.float64Type, b.uint64Type)
}

// registerNullableHelpers covers Nullable() helpers beyond the conditional family.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerNullableHelpers(b *FunctionCatalogueBuilder) {
	b.Register("toNullable", b.unknownType, b.unknownType)
	b.Register("ifNotFinite", b.float64Type, b.float64Type, b.float64Type)
	b.Register("validateNestedArraySizes", b.boolType, b.unknownType)
}

// registerEncodingFunctions covers character encoding helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerEncodingFunctions(b *FunctionCatalogueBuilder) {
	b.Register("convertCharset", b.textType, b.textType, b.textType, b.textType)
	b.Register("toValidUTF8", b.textType, b.textType)
	b.Register("normalizeUTF8NFC", b.textType, b.textType)
	b.Register("normalizeUTF8NFD", b.textType, b.textType)
	b.Register("normalizeUTF8NFKC", b.textType, b.textType)
	b.Register("normalizeUTF8NFKD", b.textType, b.textType)
}

// registerArrayHigherOrderFunctions covers arrayCount, arrayExists and arrayAll which
// take a lambda predicate.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerArrayHigherOrderFunctions(b *FunctionCatalogueBuilder) {
	b.Register("arrayCount", b.uint64Type, b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayExists", b.boolType, b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayAll", b.boolType, b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayFirstIndex", b.uint64Type, b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayLastIndex", b.uint64Type, b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayFirst", b.unknownType, b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayLast", b.unknownType, b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayFirstOrNull", b.unknownType, b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayLastOrNull", b.unknownType, b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayMin", b.unknownType, arrayOf(b.unknownType))
	b.Register("arrayMax", b.unknownType, arrayOf(b.unknownType))
	b.Register("arraySum", b.float64Type, arrayOf(b.float64Type))
	b.Register("arrayAvg", b.float64Type, arrayOf(b.float64Type))
	b.Register("arrayProduct", b.float64Type, arrayOf(b.float64Type))
	b.Register("arrayZip", arrayOf(b.unknownType), arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arrayIntersect", arrayOf(b.unknownType), arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arrayWithConstant", arrayOf(b.unknownType), b.uint64Type, b.unknownType)
	b.Register("arrayFlatten", arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("arrayReverse", arrayOf(b.unknownType), arrayOf(b.unknownType))
}

// registerVisitFunnelFunctions covers the visit, session and sequence analysis helpers
// commonly used for product analytics.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerVisitFunnelFunctions(b *FunctionCatalogueBuilder) {
	b.RegisterAggregate("sequenceMatch", b.boolType, b.textType, b.dateTimeType, b.boolType)
	b.RegisterAggregate("sequenceCount", b.uint64Type, b.textType, b.dateTimeType, b.boolType)
	b.RegisterAggregate("windowFunnel", b.uint64Type, b.uint64Type, b.dateTimeType, b.boolType)
	b.RegisterAggregate("retention", arrayOf(b.uint64Type), b.boolType)
	b.RegisterAggregate("sumMap", b.unknownType, arrayOf(b.unknownType), arrayOf(b.float64Type))
	b.RegisterAggregate("minMap", b.unknownType, arrayOf(b.unknownType), arrayOf(b.float64Type))
	b.RegisterAggregate("maxMap", b.unknownType, arrayOf(b.unknownType), arrayOf(b.float64Type))
}

// registerS3AndCloudFunctions covers cloud-storage helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerS3AndCloudFunctions(b *FunctionCatalogueBuilder) {
	b.Register("s3", b.unknownType, b.textType, b.textType)
	b.Register("s3Cluster", b.unknownType, b.textType, b.textType, b.textType)
	b.Register("url", b.unknownType, b.textType, b.textType)
	b.Register("file", b.unknownType, b.textType, b.textType)
	b.Register("hdfs", b.unknownType, b.textType, b.textType)
	b.Register("hudi", b.unknownType, b.textType, b.textType)
	b.Register("iceberg", b.unknownType, b.textType, b.textType)
	b.Register("deltaLake", b.unknownType, b.textType, b.textType)
}

// registerCheckSumFunctions covers checksum and fingerprint helpers commonly used for
// change detection.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerCheckSumFunctions(b *FunctionCatalogueBuilder) {
	b.Register("CRC32", b.uint64Type, b.textType)
	b.Register("CRC32IEEE", b.uint64Type, b.textType)
	b.Register("CRC64", b.uint64Type, b.textType)
	b.Register("adler32", b.uint64Type, b.textType)
}

// registerIntrospectionFunctions covers system and cluster introspection helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerIntrospectionFunctions(b *FunctionCatalogueBuilder) {
	b.Register("getMacro", b.textType, b.textType)
	b.Register("getSetting", b.unknownType, b.textType)
	b.Register("getServerPort", b.uint64Type, b.textType)
	b.Register("getCurrentRoles", arrayOf(b.textType))
	b.Register("hasColumnInTable", b.boolType, b.textType, b.textType, b.textType)
	b.Register("queryID", b.textType)
	b.Register("initialQueryID", b.textType)
	b.Register("shardCount", b.uint64Type)
	b.Register("shardNum", b.uint64Type)
	b.Register("nullable", b.boolType, b.unknownType)
}

// registerDictionaryFunctions covers the dictGet, dictHas and dictGetOrDefault family for
// dictionary table lookups.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerDictionaryFunctions(b *FunctionCatalogueBuilder) {
	b.Register("dictGet", b.unknownType, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetString", b.textType, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetUInt8", b.uint64Type, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetUInt16", b.uint64Type, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetUInt32", b.uint64Type, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetUInt64", b.uint64Type, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetInt8", b.int64Type, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetInt16", b.int64Type, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetInt32", b.int64Type, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetInt64", b.int64Type, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetFloat32", b.float32Type, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetFloat64", b.float64Type, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetUUID", b.uuidType, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetDate", b.dateType, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetDateTime", b.dateTimeType, b.textType, b.textType, b.uint64Type)
	b.Register("dictGetOrDefault", b.unknownType, b.textType, b.textType, b.uint64Type, b.unknownType)
	b.Register("dictGetOrNull", b.unknownType, b.textType, b.textType, b.uint64Type)
	b.Register("dictHas", b.boolType, b.textType, b.uint64Type)
	b.Register("dictGetHierarchy", arrayOf(b.uint64Type), b.textType, b.uint64Type)
	b.Register("dictGetDescendants", arrayOf(b.uint64Type), b.textType, b.uint64Type)
	b.Register("dictIsIn", b.boolType, b.textType, b.uint64Type, b.uint64Type)
}

// registerMLFunctions covers basic machine-learning evaluation helpers.
//
// Takes b (*FunctionCatalogueBuilder) which accumulates the registrations.
func registerMLFunctions(b *FunctionCatalogueBuilder) {
	b.Register("modelEvaluate", b.float64Type, b.textType, b.unknownType)
	b.Register("evalMLMethod", b.float64Type, b.unknownType, b.float64Type)
	b.Register("sigmoid", b.float64Type, b.float64Type)
	b.Register("softmax", arrayOf(b.float64Type), arrayOf(b.float64Type))
	b.Register("logTrace", b.float64Type, b.float64Type)
	b.Register("L1Distance", b.float64Type, arrayOf(b.float64Type), arrayOf(b.float64Type))
	b.Register("L2Distance", b.float64Type, arrayOf(b.float64Type), arrayOf(b.float64Type))
	b.Register("LinfDistance", b.float64Type, arrayOf(b.float64Type), arrayOf(b.float64Type))
	b.Register("cosineDistance", b.float64Type, arrayOf(b.float64Type), arrayOf(b.float64Type))
	b.Register("L1Norm", b.float64Type, arrayOf(b.float64Type))
	b.Register("L2Norm", b.float64Type, arrayOf(b.float64Type))
	b.Register("LinfNorm", b.float64Type, arrayOf(b.float64Type))
}
