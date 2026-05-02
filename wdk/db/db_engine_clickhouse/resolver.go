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
	"errors"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// engineNameArray is the canonical ClickHouse Array(T) engine name.
	engineNameArray = "Array"

	// engineNameMap is the canonical ClickHouse Map(K, V) engine name.
	engineNameMap = "Map"

	// engineNameFloat64 is the canonical ClickHouse Float64 engine name.
	engineNameFloat64 = "Float64"

	// ifFunctionArgumentCount is the number of arguments the if() form takes: (condition,
	// then, else).
	ifFunctionArgumentCount = 3

	// maxStackedCombinators caps how many combinator suffixes the resolver will strip from a
	// single function name.
	//
	// Real ClickHouse names stack at most a handful (for example quantileExactIfOrNull uses
	// two), so this bound sits comfortably above any genuine case and only pathological
	// inputs are refused.
	maxStackedCombinators = 8

	// maxResolvableFunctionNameLength caps the length of a function name the resolver will
	// attempt to split. Names longer than this are returned unsplit, since no real
	// ClickHouse function name approaches it and an unbounded identifier would otherwise
	// drive quadratic strip work.
	maxResolvableFunctionNameLength = 256

	// integerRank8Bit is the rank for 8-bit integer engine names.
	integerRank8Bit = 1

	// integerRank16Bit is the rank for 16-bit integer engine names.
	integerRank16Bit = 2

	// integerRank32Bit is the rank for 32-bit integer engine names.
	integerRank32Bit = 3

	// integerRank64Bit is the rank for 64-bit integer engine names.
	integerRank64Bit = 4

	// integerRank128Bit is the rank for 128-bit integer engine names.
	integerRank128Bit = 5

	// integerRank256Bit is the rank for 256-bit integer engine names.
	integerRank256Bit = 6
)

var (
	// errResolverNoOpinion signals that the resolver has no answer for a call where the
	// absence of an answer is not an error.
	//
	// Callers wanting to distinguish "no opinion" from "no error and no result" can
	// errors.Is the returned error against this sentinel.
	errResolverNoOpinion = errors.New("clickhouse: resolver has no opinion")
)

// ClickHouseFunctionResolver resolves function calls whose return type cannot be
// expressed by a single static signature.
//
// ClickHouse's function library is heavy on polymorphic and combinator-based shapes, so
// static catalogue lookups need a fallback. The resolver covers arrayMap(lambda, arr),
// whose return type is Array of the lambda result; arrayFilter(lambda, arr), whose return
// type matches the input array; arrayReduce('agg', arr), whose return type is the named
// aggregate's return shape; arrayJoin(arr), whose return type is the array's element
// type; tupleElement(t, n), whose return type is the n-th field of the tuple; if(cond, a,
// b), whose return type is the wider of a and b; coalesce(a, b, ...), whose return type
// is the first non-NULL type; the passthrough aggregates any, anyLast, argMin, argMax,
// max, and min, whose return type matches the first input argument; and the combinator
// suffixes If, Array, OrNull, OrDefault, and Distinct, which apply a function transformer
// to the base function's return.
//
// The resolver is invoked by the engine after the static catalogue fails to find a
// signature. It returns a nil resolution with errResolverNoOpinion when it has no answer,
// prompting the analyser to fall back to TypeCategoryUnknown.
type ClickHouseFunctionResolver struct{}

// NewClickHouseFunctionResolver constructs a resolver with no configuration.
//
// The resolver is stateless, so the same instance can be shared across all queries.
//
// Returns *ClickHouseFunctionResolver which is the ready-to-use resolver.
func NewClickHouseFunctionResolver() *ClickHouseFunctionResolver {
	return &ClickHouseFunctionResolver{}
}

// ResolveFunctionCall produces a FunctionResolution for a call when the static catalogue
// cannot.
//
// Combinator suffixes are stripped and the base name with its transformer pair handles
// every shape this resolver knows. Callers can errors.Is the returned error against
// errResolverNoOpinion to distinguish "no opinion" from a real failure, after which the
// analyser falls through to its own default.
//
// Takes catalogue (*querier_dto.Catalogue) which is the schema catalogue, currently
// unused.
// Takes name (string) which is the function name being resolved.
// Takes schema (string) which is the schema the call appears in, currently unused.
// Takes argumentTypes ([]querier_dto.SQLType) which are the resolved argument types.
//
// Returns *querier_dto.FunctionResolution which is the resolved shape, or nil when the
// resolver has no opinion.
// Returns error when the resolver has no answer, set to errResolverNoOpinion.
func (r *ClickHouseFunctionResolver) ResolveFunctionCall(
	catalogue *querier_dto.Catalogue,
	name string,
	schema string,
	argumentTypes []querier_dto.SQLType,
) (*querier_dto.FunctionResolution, error) {
	_ = catalogue
	_ = schema

	baseName, combinators := splitCombinatorSuffixes(name)

	resolution := r.resolveBase(baseName, argumentTypes)
	if resolution == nil {
		return nil, errResolverNoOpinion
	}

	for _, combinator := range combinators {
		resolution = applyCombinator(resolution, combinator)
	}
	return resolution, nil
}

// splitCombinatorSuffixes strips known ClickHouse aggregate-function combinator suffixes
// from the function name.
//
// The recognised combinators are If for a filtered aggregate (countIf, sumIf); Array for
// an aggregate over array elements (groupArrayArray); OrNull, which returns Nullable(T)
// when the input is empty; OrDefault, which returns the type's default value when the
// input is empty; Resample for a bucketed aggregate; Distinct, which deduplicates before
// aggregating; Map for an aggregate over Map values, returning Map(K, T); ForEach for an
// aggregate at each array position, returning Array(T); State for an
// AggregateFunction(name, T) intermediate state; SimpleState for a
// SimpleAggregateFunction(name, T) intermediate state; MergeState, which combines states
// and returns AggregateFunction(name, T); Merge, which combines AggregateFunction states
// and returns base T; and ArgMin and ArgMax, which run argMin or argMax over the
// aggregate by an extra column and return base T.
//
// Suffixes can stack, so quantileExactIfOrNull strips to quantileExact with combinators
// If and OrNull. Suffixes are accumulated in strip order, outermost first, then reversed
// once at the end to match the bottom-up application order, which avoids the quadratic
// prepend pattern that the obvious loop body would produce.
//
// The known list is ordered so longer suffixes that share a tail with shorter ones come
// first. Without this, MergeState would strip State first leaving an orphan Merge tail,
// and the same concern applies to SimpleState.
//
// The strip loop is bounded so a pathological identifier cannot drive quadratic work at
// resolve time: names longer than maxResolvableFunctionNameLength are never split, and no
// more than maxStackedCombinators suffixes are stripped. Both bounds sit well above any
// real ClickHouse name, so legitimate resolution is unaffected.
//
// Takes name (string) which is the function name to split.
//
// Returns string which is the base function name after suffix stripping.
// Returns []string which is the ordered list of stripped suffixes for bottom-up
// application.
func splitCombinatorSuffixes(name string) (string, []string) {
	if len(name) > maxResolvableFunctionNameLength {
		return name, nil
	}
	if isStandaloneFunctionName(name) {
		return name, nil
	}
	known := combinatorSuffixOrder()
	var stripped []string
	current := name
	for len(stripped) < maxStackedCombinators && !isStandaloneFunctionName(current) {
		matched := false
		for _, suffix := range known {
			if strings.HasSuffix(current, suffix) && len(current) > len(suffix) {
				current = current[:len(current)-len(suffix)]
				stripped = append(stripped, suffix)
				matched = true
				break
			}
		}
		if !matched {
			break
		}
	}
	slices.Reverse(stripped)
	return current, stripped
}

// combinatorSuffixOrder returns the ordered list of recognised combinator suffixes, with
// longer suffixes that overlap shorter ones placed first.
//
// The order matters because the strip loop tries suffixes in order and takes the first
// match; without this order, MergeState would strip as State first and leave a Merge
// tail.
//
// Returns []string which is the suffix list in strip-priority order.
func combinatorSuffixOrder() []string {
	return []string{
		"MergeState",
		"SimpleState",
		"OrNull",
		"OrDefault",
		"Resample",
		"Distinct",
		"ForEach",
		"ArgMin",
		"ArgMax",
		"State",
		"Merge",
		engineNameArray,
		engineNameMap,
		"If",
	}
}

// isStandaloneFunctionName reports whether the function name must not be split into a
// base and combinator pair.
//
// Some ClickHouse functions have names that end in tokens resembling combinator suffixes
// (for example arrayMap ends with "Map") but are not derived from a combinator. The
// resolver consults this list before attempting suffix stripping, and matching names
// short-circuit the strip loop.
//
// Takes name (string) which is the function name to test.
//
// Returns bool which is true when the name must not be split.
func isStandaloneFunctionName(name string) bool {
	switch strings.ToLower(name) {
	case "arraymap", "arrayfilter", "arrayreduce", "arrayreducebackward",
		"arrayfold", "arrayjoin", "arrayflatten", "arraysort",
		"arrayreversesort", "arraydistinct", "arraycompact",
		"arrayfirst", "arraylast", "arrayfirstornull", "arraylastornull",
		"arraymin", "arraymax",
		"arrayfirstindex", "arraylastindex", "arrayelement",
		"arrayenumerate", "arrayconcat", "arrayslice", "arraypushback",
		"arraypushfront", "arraypopback", "arraypopfront", "arrayresize",
		"has", "hasall", "hasany", "indexof", "tupleelement",
		"untuple", "tuple", "mapkeys", "mapvalues", "mapcontains",
		"mapadd", "mapsubtract", "mapfilter", "mapapply",
		"variantelement", "varianttype", "dynamicelement", "dynamictype",
		"finalizeaggregation", "initializeaggregation", "finalizearray",
		"laginframe", "leadinframe", "nth_value",
		"percent_rank", "cume_dist", "denserank", "percentrank",
		"length", "lower", "upper", "reverse", "concat",

		"grouparray", "groupuniqarray", "multiif":
		return true
	}
	return false
}

// applyCombinator transforms a base resolution according to the combinator's semantic.
//
// The combinator order matters because the analyser applies suffixes outer to inner. The
// cases merge the identity-return combinators If, Array, OrDefault, Distinct, ArgMin, and
// ArgMax because they all leave the resolution unchanged. Merge finalises an aggregate
// state: it consumes an AggregateFunction state and returns the underlying base value, so
// it unwraps ElementType when the base resolved to an AggregateState, the same unwrap
// that finalizeAggregation performs. Resample and ForEach wrap the result in Array(T).
// Map wraps in Map(String, T) because the key type is not available from the base
// resolution. State and MergeState set the engine name to AggregateFunction, SimpleState
// sets it to SimpleAggregateFunction, and OrNull marks the result nullable.
//
// Takes base (*querier_dto.FunctionResolution) which is the resolution to transform.
// Takes combinator (string) which is the combinator suffix to apply.
//
// Returns *querier_dto.FunctionResolution which is the transformed resolution.
func applyCombinator(base *querier_dto.FunctionResolution, combinator string) *querier_dto.FunctionResolution {
	clone := *base
	switch combinator {
	case "If", engineNameArray, "OrDefault", "Distinct", "ArgMin", "ArgMax":

		return &clone
	case "Merge":

		if clone.ReturnType.Category == querier_dto.TypeCategoryAggregateState &&
			clone.ReturnType.ElementType != nil {
			clone.ReturnType = *clone.ReturnType.ElementType
		}
		return &clone
	case "OrNull":

		clone.ReturnType.Nullable = true
		return &clone
	case "Resample", "ForEach":

		clone.ReturnType = querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryArray,
			EngineName:  engineNameArray,
			ElementType: new(clone.ReturnType),
		}
		return &clone
	case engineNameMap:

		key := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
		clone.ReturnType = querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryMap,
			EngineName:  engineNameMap,
			KeyType:     &key,
			ElementType: new(clone.ReturnType),
		}
		return &clone
	case "State", "MergeState":
		clone.ReturnType = querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryAggregateState,
			EngineName:  "AggregateFunction",
			ElementType: new(clone.ReturnType),
		}
		return &clone
	case "SimpleState":
		clone.ReturnType = querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryAggregateState,
			EngineName:  "SimpleAggregateFunction",
			ElementType: new(clone.ReturnType),
		}
		return &clone
	}
	return &clone
}

// resolveBase handles base functions whose return type depends on argument types.
//
// The lookup runs through baseResolvers, the canonical dispatch table, so the cyclomatic
// complexity stays within the linter budget; the small wrapper functions retain the
// per-function semantics in their own godoc.
//
// Takes name (string) which is the base function name to resolve.
// Takes argumentTypes ([]querier_dto.SQLType) which are the resolved argument types.
//
// Returns *querier_dto.FunctionResolution which is the resolved shape, or nil when the
// function is not one the resolver covers.
func (*ClickHouseFunctionResolver) resolveBase(name string, argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	lower := strings.ToLower(name)
	if resolver, found := baseResolvers[lower]; found {
		return resolver(argumentTypes, lower)
	}
	return nil
}

// baseResolverFunc is the call signature of every resolveBase handler. Most handlers
// ignore the lowercased name; sum / avg use it to pick the appropriate promotion target.
type baseResolverFunc func(argumentTypes []querier_dto.SQLType, loweredName string) *querier_dto.FunctionResolution

var (
	// baseResolvers maps each known function spelling to its handler.
	//
	// Multiple spellings such as arraySort and arrayReverseSort share a handler when they
	// have the same resolution rules.
	baseResolvers = map[string]baseResolverFunc{
		"arraymap":              ignoreName(resolveArrayMap),
		"arrayfilter":           ignoreName(resolveArrayFilter),
		"arrayreduce":           ignoreName(resolveArrayReduce),
		"arrayreducebackward":   ignoreName(resolveArrayReduce),
		"arrayjoin":             ignoreName(resolveArrayJoin),
		"arrayflatten":          ignoreName(resolveArrayFlatten),
		"arraysort":             ignoreName(resolvePassthroughArray),
		"arrayreversesort":      ignoreName(resolvePassthroughArray),
		"arraydistinct":         ignoreName(resolvePassthroughArray),
		"arraycompact":          ignoreName(resolvePassthroughArray),
		"arrayfirst":            ignoreName(resolveArrayScalarReduce),
		"arraylast":             ignoreName(resolveArrayScalarReduce),
		"arraymin":              ignoreName(resolveArrayScalarReduce),
		"arraymax":              ignoreName(resolveArrayScalarReduce),
		"arrayelement":          ignoreName(resolveArrayElement),
		"tupleelement":          ignoreName(resolveTupleElement),
		"if":                    ignoreName(resolveIf),
		"multiif":               ignoreName(resolveIf),
		"coalesce":              ignoreName(resolveCoalesce),
		"ifnull":                ignoreName(resolveCoalesce),
		"nullif":                ignoreName(resolveNullIf),
		"any":                   ignoreName(resolveAggregateIdentity),
		"anylast":               ignoreName(resolveAggregateIdentity),
		"anyheavy":              ignoreName(resolveAggregateIdentity),
		"min":                   ignoreName(resolveAggregateIdentity),
		"max":                   ignoreName(resolveAggregateIdentity),
		"argmin":                ignoreName(resolveArgMinMax),
		"argmax":                ignoreName(resolveArgMinMax),
		"sum":                   resolveSumAvg,
		"avg":                   resolveSumAvg,
		"count":                 ignoreName(resolveCount),
		"abs":                   ignoreName(resolveScalarIdentity),
		"greatest":              ignoreName(resolveScalarIdentity),
		"least":                 ignoreName(resolveScalarIdentity),
		"untuple":               ignoreName(resolveUntuple),
		"tuple":                 ignoreName(resolveTuple),
		"map":                   ignoreName(resolveMap),
		"mapkeys":               ignoreName(resolveMapKeys),
		"mapvalues":             ignoreName(resolveMapValues),
		"mapcontains":           ignoreName(resolveMapContains),
		"mapadd":                ignoreName(resolveMapAdd),
		"mapsubtract":           ignoreName(resolveMapAdd),
		"mapfilter":             ignoreName(resolveMapPassthrough),
		"mapapply":              ignoreName(resolveMapPassthrough),
		"variantelement":        ignoreName(resolveVariantElement),
		"varianttype":           ignoreName(resolveVariantType),
		"dynamicelement":        ignoreName(resolveVariantElement),
		"dynamictype":           ignoreName(resolveVariantType),
		"finalizeaggregation":   ignoreName(resolveFinalizeAggregation),
		"initializeaggregation": ignoreName(resolveInitializeAggregation),
		"finalizearray":         ignoreName(resolveFinalizeArray),
	}
)

// ignoreName adapts a single-argument resolver to the baseResolverFunc shape, discarding
// the lowered-name argument that only sum and avg need.
//
// Takes fn (func([]querier_dto.SQLType) *querier_dto.FunctionResolution) which is the
// single-argument resolver to adapt.
//
// Returns baseResolverFunc which calls fn and ignores the lowered name.
func ignoreName(fn func(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution) baseResolverFunc {
	return func(argumentTypes []querier_dto.SQLType, _ string) *querier_dto.FunctionResolution {
		return fn(argumentTypes)
	}
}

// resolveArrayMap returns Array of the lambda's return type.
//
// Since the resolver does not inspect the lambda body, the result type defaults to the
// array's element type when no better information is available.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// array being the last one.
//
// Returns *querier_dto.FunctionResolution which is the array resolution, or nil when the
// input is not a usable array.
func resolveArrayMap(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) < 2 {
		return nil
	}
	arrayArg := argumentTypes[len(argumentTypes)-1]
	if arrayArg.Category != querier_dto.TypeCategoryArray || arrayArg.ElementType == nil {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryArray,
			EngineName:  engineNameArray,
			ElementType: new(*arrayArg.ElementType),
		},
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}
}

// resolveArrayFilter returns the input array type unchanged because filtering preserves
// shape and only removes elements that fail the predicate.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// array being the last one.
//
// Returns *querier_dto.FunctionResolution which is the array resolution, or nil when the
// input is not an array.
func resolveArrayFilter(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) < 2 {
		return nil
	}
	arrayArg := argumentTypes[len(argumentTypes)-1]
	if arrayArg.Category != querier_dto.TypeCategoryArray {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType:        arrayArg,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}
}

// resolveArrayReduce returns the named aggregate's return type.
//
// The aggregate name is the first argument's literal value. The resolver returns the
// array element type as the safest approximation when the aggregate is not known.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// array being the last one.
//
// Returns *querier_dto.FunctionResolution which is the element-type resolution, or nil
// when the input is not a usable array.
func resolveArrayReduce(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) < 2 {
		return nil
	}
	arrayArg := argumentTypes[len(argumentTypes)-1]
	if arrayArg.Category != querier_dto.TypeCategoryArray || arrayArg.ElementType == nil {
		return nil
	}
	element := *arrayArg.ElementType
	return &querier_dto.FunctionResolution{
		ReturnType:        element,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}
}

// resolveArrayJoin returns the array's element type, which is what the projection sees
// after the array is unrolled.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// array being the first one.
//
// Returns *querier_dto.FunctionResolution which is the element-type resolution, or nil
// when the input is not a usable array.
func resolveArrayJoin(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[0]
	if arg.Category != querier_dto.TypeCategoryArray || arg.ElementType == nil {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType:        *arg.ElementType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		ReturnsSet:        true,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}
}

// resolveArrayFlatten returns Array(T) where T is the inner element of a nested
// Array(Array(T)).
//
// When the input is not nested, the input array type is returned unchanged.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// array being the first one.
//
// Returns *querier_dto.FunctionResolution which is the flattened array resolution, or nil
// when the input is not a usable array.
func resolveArrayFlatten(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[0]
	if arg.Category != querier_dto.TypeCategoryArray || arg.ElementType == nil {
		return nil
	}
	if arg.ElementType.Category == querier_dto.TypeCategoryArray && arg.ElementType.ElementType != nil {
		return &querier_dto.FunctionResolution{
			ReturnType: querier_dto.SQLType{
				Category:    querier_dto.TypeCategoryArray,
				EngineName:  engineNameArray,
				ElementType: new(*arg.ElementType.ElementType),
			},
			DataAccess: querier_dto.DataAccessReadOnly,
		}
	}
	return &querier_dto.FunctionResolution{
		ReturnType: arg,
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolvePassthroughArray returns the input array type unchanged for functions that
// reorder, deduplicate, or compact elements without changing the shape.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// array being the first one.
//
// Returns *querier_dto.FunctionResolution which is the unchanged array resolution, or nil
// when the input is not an array.
func resolvePassthroughArray(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[0]
	if arg.Category != querier_dto.TypeCategoryArray {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: arg,
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveArrayScalarReduce returns the array's element type for scalar-reduction
// functions such as arrayFirst, arrayLast, arrayMin, and arrayMax.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// array being the last one.
//
// Returns *querier_dto.FunctionResolution which is the element-type resolution, or nil
// when the input is not a usable array.
func resolveArrayScalarReduce(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[len(argumentTypes)-1]
	if arg.Category != querier_dto.TypeCategoryArray || arg.ElementType == nil {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: *arg.ElementType,
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveArrayElement returns the array's element type.
//
// ClickHouse also accepts arr[idx] subscript syntax, which the parser lowers to
// arrayElement().
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// array being the first one.
//
// Returns *querier_dto.FunctionResolution which is the element-type resolution, or nil
// when the input is not a usable array.
func resolveArrayElement(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[0]
	if arg.Category != querier_dto.TypeCategoryArray || arg.ElementType == nil {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: *arg.ElementType,
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveTupleElement returns the n-th field of a tuple.
//
// When the tuple has named fields, the index can also be a string literal, and the
// resolver returns the catch-all Unknown when the index value is not statically known.
// The resolver receives only the argument types, not the literal index value, so for a
// tuple with more than one field it cannot tell which field tupleElement(t, n) selects
// and falls back to Unknown. The single-field case is unambiguous and resolved exactly.
// Callers that need the precise field type for a multi-field tuple use the dotted
// accessor form t.n, which the parser lowers to a statically-known field selection.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// tuple being the first one.
//
// Returns *querier_dto.FunctionResolution which is the field-type resolution, or nil when
// the input is not a tuple.
func resolveTupleElement(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	tupleArg := argumentTypes[0]
	if tupleArg.Category != querier_dto.TypeCategoryStruct || len(tupleArg.StructFields) == 0 {
		return nil
	}

	returnType := querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
	if len(tupleArg.StructFields) == 1 {
		returnType = tupleArg.StructFields[0].SQLType
	}
	return &querier_dto.FunctionResolution{
		ReturnType: returnType,
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveIf produces the return type for if(cond, then, else) and multiIf(cond1, then1,
// cond2, then2, ..., else).
//
// The result type is the unification of the then and else branches. For if, the arguments
// are (cond, then, else) and the result is the wider of then and else. For multiIf, the
// arguments are alternating (cond, then) pairs followed by an else value, and the result
// is the wider of every then branch together with the final else value.
//
// Type unification follows the ClickHouse rules approximated by unifyBranchTypes:
// identical types collapse to that type, one Unknown yields the other, mixed integers
// widen to the wider integer, mixed integer and float widen to Float64, mixed text yields
// String, and otherwise the first non-Unknown wins.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types.
//
// Returns *querier_dto.FunctionResolution which is the unified branch resolution, or nil
// when fewer than two arguments are supplied.
func resolveIf(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) < 2 {
		return nil
	}
	branchTypes := collectBranchTypes(argumentTypes)
	unified := unifyBranchTypes(branchTypes)
	return &querier_dto.FunctionResolution{
		ReturnType: unified,
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// collectBranchTypes extracts the result branches from an if or multiIf argument list.
//
// For if(cond, then, else) the result is the then and else types. For multiIf(c1, t1, c2,
// t2, ..., else) the result is each then type followed by the trailing else type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types.
//
// Returns []querier_dto.SQLType which is the ordered list of branch result types.
func collectBranchTypes(argumentTypes []querier_dto.SQLType) []querier_dto.SQLType {
	if len(argumentTypes) == ifFunctionArgumentCount {
		return []querier_dto.SQLType{argumentTypes[1], argumentTypes[2]}
	}

	branches := []querier_dto.SQLType{}
	for index := 1; index+1 < len(argumentTypes); index += 2 {
		branches = append(branches, argumentTypes[index])
	}
	if len(argumentTypes)%2 == 1 {
		branches = append(branches, argumentTypes[len(argumentTypes)-1])
	}
	return branches
}

// resolveCoalesce returns the union of every argument's type.
//
// The return type is non-nullable when at least one argument is known to be non-nullable,
// which the analyser handles downstream. Type unification follows the same rules as
// resolveIf, so coalesce(int32_col, int64_col) widens to Int64 and coalesce(int_col,
// float_col) widens to Float64.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types.
//
// Returns *querier_dto.FunctionResolution which is the unified resolution, or nil when no
// arguments are supplied.
func resolveCoalesce(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	unified := unifyBranchTypes(argumentTypes)
	return &querier_dto.FunctionResolution{
		ReturnType: unified,
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// unifyBranchTypes returns a single SQLType that all of the input branches can be safely
// promoted to.
//
// The rule chain is described in resolveIf's doc comment. Unknown is returned when no
// rule matches.
//
// Takes branches ([]querier_dto.SQLType) which are the branch types to unify.
//
// Returns querier_dto.SQLType which is the unified type, or Unknown when none applies.
func unifyBranchTypes(branches []querier_dto.SQLType) querier_dto.SQLType {
	if len(branches) == 0 {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
	}

	candidates := []querier_dto.SQLType{}
	for index := range branches {
		if branches[index].Category != querier_dto.TypeCategoryUnknown {
			candidates = append(candidates, branches[index])
		}
	}
	if len(candidates) == 0 {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
	}
	current := candidates[0]
	for index := 1; index < len(candidates); index++ {
		current = unifyPair(current, candidates[index])
	}
	return current
}

// unifyPair returns the wider of two types per ClickHouse's permissive promotion rules,
// mirroring the union-compatibility rules used by set operations.
//
// Takes left (querier_dto.SQLType) which is the first operand type.
// Takes right (querier_dto.SQLType) which is the second operand type.
//
// Returns querier_dto.SQLType which is the wider of the two types.
func unifyPair(left, right querier_dto.SQLType) querier_dto.SQLType {
	if left.Category == right.Category && left.EngineName == right.EngineName {
		return left
	}
	if left.Category == right.Category {
		switch left.Category {
		case querier_dto.TypeCategoryInteger:
			return widerInteger(left, right)
		case querier_dto.TypeCategoryFloat:
			return querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: engineNameFloat64}
		case querier_dto.TypeCategoryText:
			return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
		default:

			return left
		}
	}
	if left.Category == querier_dto.TypeCategoryInteger && right.Category == querier_dto.TypeCategoryFloat {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: engineNameFloat64}
	}
	if left.Category == querier_dto.TypeCategoryFloat && right.Category == querier_dto.TypeCategoryInteger {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: engineNameFloat64}
	}
	return left
}

// widerInteger returns the wider of two ClickHouse integer types using the same rank
// table as set-operation unification.
//
// The ranking maps bit width to a numeric rank so the comparison is a single integer
// compare regardless of signed and unsigned mixing. When the two operands share a width
// rank but differ in sign (for example Int32 and UInt32) the least-common-type is the
// next wider signed type, because the signed range cannot hold the unsigned operand's
// high-bit values at the same width. The 64-bit pair (Int64 and UInt64) is excluded
// because stock ClickHouse throws rather than promoting to Int128, so the resolver leaves
// the left operand in place for that pair.
//
// Takes left (querier_dto.SQLType) which is the first integer operand.
// Takes right (querier_dto.SQLType) which is the second integer operand.
//
// Returns querier_dto.SQLType which is the wider integer type.
func widerInteger(left, right querier_dto.SQLType) querier_dto.SQLType {
	leftRank := integerRank(left.EngineName)
	rightRank := integerRank(right.EngineName)
	if leftRank == rightRank && leftRank != 0 &&
		isUnsignedInteger(left.EngineName) != isUnsignedInteger(right.EngineName) {
		if widened, found := nextWiderSignedInteger(leftRank); found {
			return widened
		}
	}
	if rightRank > leftRank {
		return right
	}
	return left
}

// isUnsignedInteger reports whether a ClickHouse integer engine name denotes an unsigned
// type in the UInt8 to UInt256 range.
//
// Takes engineName (string) which is the integer engine name to test.
//
// Returns bool which is true when the name denotes an unsigned integer.
func isUnsignedInteger(engineName string) bool {
	return strings.HasPrefix(strings.ToUpper(engineName), "UINT")
}

// nextWiderSignedInteger returns the signed integer type one width rank above the
// supplied rank.
//
// It is used when mixed-sign operands of equal width must widen to a signed type. No
// wider signed type exists for the 64-bit rank because stock ClickHouse throws rather
// than widening to Int128.
//
// Takes rank (int) which is the current integer width rank.
//
// Returns querier_dto.SQLType which is the next wider signed integer type.
// Returns bool which is true when a wider signed type exists.
func nextWiderSignedInteger(rank int) (querier_dto.SQLType, bool) {
	switch rank {
	case integerRank8Bit:
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int16"}, true
	case integerRank16Bit:
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int32"}, true
	case integerRank32Bit:
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"}, true
	default:
		return querier_dto.SQLType{}, false
	}
}

// integerRank assigns a numeric rank to ClickHouse integer type names, where a higher
// rank means a wider type.
//
// Takes engineName (string) which is the integer engine name to rank.
//
// Returns int which is the width rank, or zero when the name is not a known integer type.
func integerRank(engineName string) int {
	switch strings.ToUpper(engineName) {
	case "UINT8", "INT8":
		return integerRank8Bit
	case "UINT16", "INT16":
		return integerRank16Bit
	case "UINT32", "INT32":
		return integerRank32Bit
	case "UINT64", "INT64":
		return integerRank64Bit
	case "UINT128", "INT128":
		return integerRank128Bit
	case "UINT256", "INT256":
		return integerRank256Bit
	}
	return 0
}

// resolveNullIf returns the first argument's type.
//
// The result is Nullable; the analyser sets the flag on the consumer.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types.
//
// Returns *querier_dto.FunctionResolution which is the first-argument resolution, or nil
// when fewer than two arguments are supplied.
func resolveNullIf(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) < 2 {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: argumentTypes[0],
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveCount resolves the count aggregate and every combinator form that strips to it.
//
// The covered forms include countIf, countDistinct, and countIfDistinct. count always
// returns a non-null UInt64 row count regardless of its arguments, so the argument types
// are ignored. A dedicated base resolver keeps countIf and related forms from falling
// through to an Unknown type, a common telemetry pattern that would otherwise generate an
// any column.
//
// Returns *querier_dto.FunctionResolution which is the non-null UInt64 count resolution.
func resolveCount(_ []querier_dto.SQLType) *querier_dto.FunctionResolution {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
		IsAggregate:       true,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}
}

// resolveAggregateIdentity returns the first argument's type with the IsAggregate flag
// set.
//
// It is used by passthrough aggregates such as any, anyLast, min, max, and anyHeavy,
// where the aggregate yields one row of the same shape as one input row.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types.
//
// Returns *querier_dto.FunctionResolution which is the first-argument resolution flagged
// as an aggregate, or nil when no arguments are supplied.
func resolveAggregateIdentity(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType:  argumentTypes[0],
		IsAggregate: true,
		DataAccess:  querier_dto.DataAccessReadOnly,
	}
}

// resolveScalarIdentity returns the first argument's type without the IsAggregate flag.
//
// It is used by scalar functions such as abs, greatest, and least, where the result shape
// matches an input but the call does not collapse rows.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types.
//
// Returns *querier_dto.FunctionResolution which is the first-argument resolution, or nil
// when no arguments are supplied.
func resolveScalarIdentity(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: argumentTypes[0],
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveArgMinMax returns the first argument's type because argMin(x, y) returns the x
// that minimises y, so the result has x's shape.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types.
//
// Returns *querier_dto.FunctionResolution which is the first-argument resolution flagged
// as an aggregate, or nil when fewer than two arguments are supplied.
func resolveArgMinMax(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) < 2 {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType:  argumentTypes[0],
		IsAggregate: true,
		DataAccess:  querier_dto.DataAccessReadOnly,
	}
}

// resolveSumAvg promotes integer inputs to a wider integer (sum) or to Float64 (avg). The
// exact promotion rules vary by input bit width; the simplified mapping below covers the
// common cases.
//
// Takes argumentTypes ([]querier_dto.SQLType), the argument list.
// Takes name (string), the lowercase function spelling ("sum" / "avg").
//
// Returns the resolution with widened return type, or nil for empty input.
func resolveSumAvg(argumentTypes []querier_dto.SQLType, name string) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[0]
	if name == "avg" {
		return &querier_dto.FunctionResolution{
			ReturnType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: engineNameFloat64},
			IsAggregate: true,
			DataAccess:  querier_dto.DataAccessReadOnly,
		}
	}
	if arg.Category == querier_dto.TypeCategoryInteger {
		if strings.HasPrefix(strings.ToUpper(arg.EngineName), "UINT") {
			return &querier_dto.FunctionResolution{
				ReturnType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
				IsAggregate: true,
				DataAccess:  querier_dto.DataAccessReadOnly,
			}
		}
		return &querier_dto.FunctionResolution{
			ReturnType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"},
			IsAggregate: true,
			DataAccess:  querier_dto.DataAccessReadOnly,
		}
	}

	return &querier_dto.FunctionResolution{
		ReturnType:  arg,
		IsAggregate: true,
		DataAccess:  querier_dto.DataAccessReadOnly,
	}
}

// resolveUntuple returns the first tuple field's type as a best-effort approximation.
//
// untuple expands to all fields, but only the projection level can model that, so
// downstream codegen treats untuple(t) as a struct expansion site.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// tuple being the first one.
//
// Returns *querier_dto.FunctionResolution which is the first field's resolution, or nil
// when the input is not a tuple.
func resolveUntuple(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[0]
	if arg.Category != querier_dto.TypeCategoryStruct || len(arg.StructFields) == 0 {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: arg.StructFields[0].SQLType,
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveTuple constructs a Tuple(...) from the argument types.
//
// The result's StructFields are anonymous, synthesised as _1, _2, and so on.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the tuple field types.
//
// Returns *querier_dto.FunctionResolution which is the constructed Tuple resolution.
func resolveTuple(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	fields := make([]querier_dto.StructField, len(argumentTypes))
	for index := range argumentTypes {
		fields[index] = querier_dto.StructField{
			Name:    synthesiseAnonymousFieldName(index + 1),
			SQLType: argumentTypes[index],
		}
	}
	return &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{
			Category:     querier_dto.TypeCategoryStruct,
			EngineName:   "Tuple",
			StructFields: fields,
		},
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveMap constructs a Map(K, V) from alternating key and value argument pairs.
//
// ClickHouse's map(k1, v1, k2, v2, ...) calls require even arity. The resolver uses the
// first key and value pair to derive types and trusts the analyser to flag arity
// mistakes.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the alternating key and value
// types.
//
// Returns *querier_dto.FunctionResolution which is the constructed Map resolution, or nil
// when fewer than two arguments are supplied.
func resolveMap(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) < 2 {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryMap,
			EngineName:  engineNameMap,
			KeyType:     new(argumentTypes[0]),
			ElementType: new(argumentTypes[1]),
		},
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveMapKeys returns Array(K) where K is the map's key type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the map
// being the first one.
//
// Returns *querier_dto.FunctionResolution which is the Array(K) resolution, or nil when
// the input is not a usable map.
func resolveMapKeys(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[0]
	if arg.Category != querier_dto.TypeCategoryMap || arg.KeyType == nil {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryArray,
			EngineName:  engineNameArray,
			ElementType: new(*arg.KeyType),
		},
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveMapValues returns Array(V) where V is the map's value type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the map
// being the first one.
//
// Returns *querier_dto.FunctionResolution which is the Array(V) resolution, or nil when
// the input is not a usable map.
func resolveMapValues(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[0]
	if arg.Category != querier_dto.TypeCategoryMap || arg.ElementType == nil {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryArray,
			EngineName:  engineNameArray,
			ElementType: new(*arg.ElementType),
		},
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveMapContains returns Bool because mapContains(map, key) reports membership.
//
// nil is returned only when no map argument is supplied, so the analyser can fall through
// to its Unknown default.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types.
//
// Returns *querier_dto.FunctionResolution which is the Bool resolution, or nil when no
// arguments are supplied.
func resolveMapContains(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"},
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveMapAdd returns the map's type with the union of both operands' key and value
// pairs.
//
// ClickHouse's mapAdd and mapSubtract take two equal-shape maps and elementwise add or
// subtract the values, and the resulting shape matches the first argument.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the map
// being the first one.
//
// Returns *querier_dto.FunctionResolution which is the map resolution, or nil when the
// input is not a map.
func resolveMapAdd(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[0]
	if arg.Category != querier_dto.TypeCategoryMap {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: arg,
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveMapPassthrough returns the map's type unchanged.
//
// It is used by mapFilter and mapApply, both of which preserve the input map's shape.
// mapApply may transform values, but the type-level shape is the same as the input until
// the lambda body is type-checked.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the map
// being the last one.
//
// Returns *querier_dto.FunctionResolution which is the unchanged map resolution, or nil
// when the input is not a map.
func resolveMapPassthrough(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[len(argumentTypes)-1]
	if arg.Category != querier_dto.TypeCategoryMap {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: arg,
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveVariantElement returns the type identified by the literal TypeName second
// argument.
//
// Because the resolver interface sees only argument types rather than literal values, the
// second argument's value is not statically known, so the resolver falls back to the
// variant's first member type and the analyser still receives a concrete SQLType. The
// result is nullable because variantElement returns NULL when the requested type is not
// the active variant member.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// variant being the first one.
//
// Returns *querier_dto.FunctionResolution which is the nullable member-type resolution,
// or nil when no arguments are supplied.
func resolveVariantElement(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	variant := argumentTypes[0]
	if len(variant.UnionMembers) == 0 {
		return &querier_dto.FunctionResolution{
			ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, Nullable: true},
			DataAccess: querier_dto.DataAccessReadOnly,
		}
	}
	member := variant.UnionMembers[0].SQLType
	member.Nullable = true
	return &querier_dto.FunctionResolution{
		ReturnType: member,
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveVariantType returns String because ClickHouse's variantType and dynamicType
// accessors yield the textual name of the active variant member.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types.
//
// Returns *querier_dto.FunctionResolution which is the String resolution, or nil when no
// arguments are supplied.
func resolveVariantType(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"},
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveFinalizeAggregation strips the AggregateFunction wrapper from an aggregate-state
// argument and returns the underlying value type.
//
// The wrapped value type is held on the ElementType pointer of the AggregateState
// SQLType. When it is absent, the resolver returns Unknown so downstream consumers can
// refine via the catalogue.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// state being the first one.
//
// Returns *querier_dto.FunctionResolution which is the underlying value resolution, or
// nil when no arguments are supplied.
func resolveFinalizeAggregation(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[0]
	if arg.ElementType != nil {
		return &querier_dto.FunctionResolution{
			ReturnType: *arg.ElementType,
			DataAccess: querier_dto.DataAccessReadOnly,
		}
	}
	return &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveInitializeAggregation wraps a value type into an AggregateFunction state.
//
// ClickHouse uses this to build aggregate states from raw inputs for incremental
// materialised view paths. The argument list is (aggregateName, value...), and the
// resolver returns the AggregateFunction shape holding the first value's type so codegen
// sees a concrete state type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// first value being the second one.
//
// Returns *querier_dto.FunctionResolution which is the AggregateFunction state
// resolution, or nil when fewer than two arguments are supplied.
func resolveInitializeAggregation(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) < 2 {
		return nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryAggregateState,
			EngineName:  "AggregateFunction",
			ElementType: new(argumentTypes[1]),
		},
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}

// resolveFinalizeArray finalises an Array(AggregateFunction(...)) into an Array of the
// underlying value type.
//
// The input shape comes from arrayMap over a column of aggregate states, and the result
// is Array(T) where T is the finalised value type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which are the call argument types, the
// array of states being the first one.
//
// Returns *querier_dto.FunctionResolution which is the finalised Array(T) resolution, or
// nil when the input is not a usable array.
func resolveFinalizeArray(argumentTypes []querier_dto.SQLType) *querier_dto.FunctionResolution {
	if len(argumentTypes) == 0 {
		return nil
	}
	arg := argumentTypes[0]
	if arg.Category != querier_dto.TypeCategoryArray || arg.ElementType == nil {
		return nil
	}
	inner := *arg.ElementType
	finalised := querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
	if inner.ElementType != nil {
		finalised = *inner.ElementType
	}
	return &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryArray,
			EngineName:  engineNameArray,
			ElementType: &finalised,
		},
		DataAccess: querier_dto.DataAccessReadOnly,
	}
}
