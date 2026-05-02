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
	// extractKeyValuePairsName is the canonical lower-case spelling for the basic key=value
	// extraction helper. Pulled out so the variadic registrations for every arity match
	// against a single identifier and stay under the add-constant lint threshold.
	extractKeyValuePairsName = "extractKeyValuePairs"

	// extractKeyValuePairsWithEscapingName is the spelling for the escaping-aware variant of
	// extractKeyValuePairs.
	extractKeyValuePairsWithEscapingName = "extractKeyValuePairsWithEscaping"
)

// registerExtendedTupleAndMapFunctions covers the long tail of tuple arithmetic, naming,
// and similarity helpers, plus the broader map completeness family.
//
// The map family spans mapAll, mapExists, mapConcat, mapContains, mapExtract,
// mapFromArrays, the sort variants, mapPopulateSeries, mapUpdate, and key/value
// extraction. Registration delegates to topical helpers so each function stays within the
// linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedTupleAndMapFunctions(b *FunctionCatalogueBuilder) {
	registerExtendedTupleFunctions(b)
	registerTupleArithmeticFunctions(b)
	registerExtendedMapFunctions(b)
	registerMapHigherOrderFunctions(b)
	registerMapMembershipFunctions(b)
	registerMapTransformFunctions(b)
	registerKeyValueExtractionFunctions(b)
}

// registerExtendedTupleFunctions covers concatenation, naming, and flattening helpers
// beyond the base tuple, tupleElement, and untuple registrations in builtin_functions.go.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedTupleFunctions(b *FunctionCatalogueBuilder) {
	b.Register("tupleConcat", b.unknownType, b.unknownType, b.unknownType)
	b.Register("tupleNames", arrayOf(b.textType), b.unknownType)
	b.Register("flattenTuple", b.unknownType, b.unknownType)
	keyValueTupleType := jsonKeyValueTupleType(b)
	b.Register("tupleToNameValuePairs", arrayOf(keyValueTupleType), b.unknownType)
}

// registerTupleArithmeticFunctions covers element-wise tuple arithmetic, integer-division
// variants, modulo and positive-modulo variants, plus the Hamming distance and dot
// product helpers.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerTupleArithmeticFunctions(b *FunctionCatalogueBuilder) {
	b.Register("tupleNegate", b.unknownType, b.unknownType)
	for _, name := range []string{"tuplePlus", "tupleMinus", "tupleMultiply", "tupleDivide"} {
		b.Register(name, b.unknownType, b.unknownType, b.unknownType)
	}
	b.Register("tupleMultiplyByNumber", b.unknownType, b.unknownType, b.float64Type)
	b.Register("tupleDivideByNumber", b.unknownType, b.unknownType, b.float64Type)
	for _, name := range []string{"tupleIntDiv", "tupleIntDivOrZero", "tupleModulo"} {
		b.Register(name, b.unknownType, b.unknownType, b.unknownType)
	}
	for _, name := range []string{
		"tupleIntDivByNumber", "tupleIntDivOrZeroByNumber",
		"tupleModuloByNumber", "tuplePositiveModuloByNumber",
	} {
		b.Register(name, b.unknownType, b.unknownType, b.float64Type)
	}

	b.Register("dotProduct", b.float64Type, b.unknownType, b.unknownType)
}

// registerExtendedMapFunctions covers the construction helpers and the conversion and
// population helpers that produce or transform a Map without taking a higher-order
// function.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedMapFunctions(b *FunctionCatalogueBuilder) {
	b.RegisterVariadic("mapConcat", b.unknownType, 1, b.unknownType)
	b.Register("mapFromArrays", b.unknownType, arrayOf(b.unknownType), arrayOf(b.unknownType))
	b.Register("mapUpdate", b.unknownType, b.unknownType, b.unknownType)
	b.Register("mapPopulateSeries", b.unknownType, b.unknownType)
	b.Register("mapPopulateSeries", b.unknownType, b.unknownType, b.unknownType)
}

// registerMapHigherOrderFunctions covers the higher-order predicates (mapAll and
// mapExists) and the sort family with optional comparator.
//
// The sort family accepts an optional lambda argument; both arities are registered for
// each spelling.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerMapHigherOrderFunctions(b *FunctionCatalogueBuilder) {
	b.RegisterVariadic("mapAll", b.boolType, 1, b.unknownType)
	b.RegisterVariadic("mapExists", b.boolType, 1, b.unknownType)
	b.Register("mapSort", b.unknownType, b.unknownType)
	b.Register("mapSort", b.unknownType, b.unknownType, b.unknownType)
	b.Register("mapReverseSort", b.unknownType, b.unknownType)
	b.Register("mapReverseSort", b.unknownType, b.unknownType, b.unknownType)
	b.Register("mapPartialSort", b.unknownType, b.uint64Type, b.unknownType)
	b.Register("mapPartialSort", b.unknownType, b.unknownType, b.uint64Type, b.unknownType)
	b.Register("mapPartialReverseSort", b.unknownType, b.uint64Type, b.unknownType)
	b.Register("mapPartialReverseSort", b.unknownType, b.unknownType, b.uint64Type, b.unknownType)
}

// registerMapMembershipFunctions covers the membership predicates that match by key,
// value, or LIKE pattern.
//
// ClickHouse exposes these as distinct functions from mapContains so each spelling needs
// its own catalogue entry.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerMapMembershipFunctions(b *FunctionCatalogueBuilder) {
	b.Register("mapContainsKey", b.boolType, b.unknownType, b.unknownType)
	b.Register("mapContainsKeyLike", b.boolType, b.unknownType, b.textType)
	b.Register("mapContainsValue", b.boolType, b.unknownType, b.unknownType)
	b.Register("mapContainsValueLike", b.boolType, b.unknownType, b.textType)
}

// registerMapTransformFunctions covers the LIKE-pattern extraction helpers that filter a
// map by key or value pattern.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerMapTransformFunctions(b *FunctionCatalogueBuilder) {
	b.Register("mapExtractKeyLike", b.unknownType, b.unknownType, b.textType)
	b.Register("mapExtractValueLike", b.unknownType, b.unknownType, b.textType)
}

// registerKeyValueExtractionFunctions covers the string-to-map extraction helpers used to
// parse delimited key=value pairs.
//
// Each arity covers (s), (s, pair_delim), (s, pair_delim, kv_delim), and (s, pair_delim,
// kv_delim, quote).
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerKeyValueExtractionFunctions(b *FunctionCatalogueBuilder) {
	b.Register(extractKeyValuePairsName, b.unknownType, b.textType)
	b.Register(extractKeyValuePairsName, b.unknownType, b.textType, b.textType)
	b.Register(extractKeyValuePairsName, b.unknownType, b.textType, b.textType, b.textType)
	b.Register(extractKeyValuePairsName, b.unknownType, b.textType, b.textType, b.textType, b.textType)
	b.Register(extractKeyValuePairsWithEscapingName, b.unknownType, b.textType)
	b.Register(extractKeyValuePairsWithEscapingName, b.unknownType, b.textType, b.textType)
	b.Register(extractKeyValuePairsWithEscapingName, b.unknownType, b.textType, b.textType, b.textType)
	b.Register(extractKeyValuePairsWithEscapingName, b.unknownType, b.textType, b.textType, b.textType, b.textType)
}
