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

package db_engine_duckdb

import (
	"maps"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// integerPromotionRankInt1 ranks signed 1-byte integers.
	integerPromotionRankInt1 = 1

	// integerPromotionRankUtinyint ranks unsigned 1-byte integers.
	integerPromotionRankUtinyint = 2

	// integerPromotionRankInt2 ranks signed 2-byte integers.
	integerPromotionRankInt2 = 3

	// integerPromotionRankUsmall ranks unsigned 2-byte integers.
	integerPromotionRankUsmall = 4

	// integerPromotionRankInt4 ranks signed 4-byte integers and the default fallback.
	integerPromotionRankInt4 = 5

	// integerPromotionRankUint ranks unsigned 4-byte integers.
	integerPromotionRankUint = 6

	// integerPromotionRankInt8 ranks signed 8-byte integers.
	integerPromotionRankInt8 = 7

	// integerPromotionRankUbigint ranks unsigned 8-byte integers.
	integerPromotionRankUbigint = 8

	// integerPromotionRankHuge ranks signed 16-byte integers.
	integerPromotionRankHuge = 9

	// integerPromotionRankUhuge ranks unsigned 16-byte integers.
	integerPromotionRankUhuge = 10
)

var (
	// builtinTypeMap maps lower-case DuckDB type names (including Postgres aliases) to their
	// normalised SQLType representations.
	builtinTypeMap = map[string]querier_dto.SQLType{
		// Integer types (signed)
		"tinyint":  {Category: querier_dto.TypeCategoryInteger, EngineName: "int1"},
		"int1":     {Category: querier_dto.TypeCategoryInteger, EngineName: "int1"},
		"smallint": {Category: querier_dto.TypeCategoryInteger, EngineName: "int2"},
		"int2":     {Category: querier_dto.TypeCategoryInteger, EngineName: "int2"},
		"integer":  {Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
		"int":      {Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
		"int4":     {Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
		"bigint":   {Category: querier_dto.TypeCategoryInteger, EngineName: "int8"},
		"int8":     {Category: querier_dto.TypeCategoryInteger, EngineName: "int8"},
		"hugeint":  {Category: querier_dto.TypeCategoryInteger, EngineName: "hugeint"},

		// Integer types (unsigned)
		"utinyint":  {Category: querier_dto.TypeCategoryInteger, EngineName: "utinyint"},
		"usmallint": {Category: querier_dto.TypeCategoryInteger, EngineName: "usmallint"},
		"uinteger":  {Category: querier_dto.TypeCategoryInteger, EngineName: "uinteger"},
		"ubigint":   {Category: querier_dto.TypeCategoryInteger, EngineName: "ubigint"},
		"uhugeint":  {Category: querier_dto.TypeCategoryInteger, EngineName: "uhugeint"},

		// Serial types (normalise to underlying integer)
		"smallserial": {Category: querier_dto.TypeCategoryInteger, EngineName: "int2"},
		"serial2":     {Category: querier_dto.TypeCategoryInteger, EngineName: "int2"},
		"serial":      {Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
		"serial4":     {Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
		"bigserial":   {Category: querier_dto.TypeCategoryInteger, EngineName: "int8"},
		"serial8":     {Category: querier_dto.TypeCategoryInteger, EngineName: "int8"},

		// Float types
		"real":             {Category: querier_dto.TypeCategoryFloat, EngineName: "float4"},
		"float4":           {Category: querier_dto.TypeCategoryFloat, EngineName: "float4"},
		"double precision": {Category: querier_dto.TypeCategoryFloat, EngineName: "float8"},
		"double":           {Category: querier_dto.TypeCategoryFloat, EngineName: "float8"},
		"float8":           {Category: querier_dto.TypeCategoryFloat, EngineName: "float8"},
		"float":            {Category: querier_dto.TypeCategoryFloat, EngineName: "float8"},

		// Decimal types
		"numeric": {Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"},
		"decimal": {Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"},

		// Boolean
		"boolean": {Category: querier_dto.TypeCategoryBoolean, EngineName: "bool"},
		"bool":    {Category: querier_dto.TypeCategoryBoolean, EngineName: "bool"},

		// Text types (DuckDB canonical text type is varchar)
		"text":              {Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
		"varchar":           {Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
		"character varying": {Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
		"character":         {Category: querier_dto.TypeCategoryText, EngineName: "char"},
		"char":              {Category: querier_dto.TypeCategoryText, EngineName: "char"},
		"bpchar":            {Category: querier_dto.TypeCategoryText, EngineName: "char"},
		"name":              {Category: querier_dto.TypeCategoryText, EngineName: "varchar"},

		// Binary types
		"bytea": {Category: querier_dto.TypeCategoryBytea, EngineName: "blob"},
		"blob":  {Category: querier_dto.TypeCategoryBytea, EngineName: "blob"},

		// Temporal types
		"timestamp without time zone": {Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp"},
		"timestamp":                   {Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp"},
		"timestamp with time zone":    {Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamptz"},
		"timestamptz":                 {Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamptz"},
		"timestamp_s":                 {Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp_s"},
		"timestamp_ms":                {Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp_ms"},
		"timestamp_ns":                {Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp_ns"},
		"date":                        {Category: querier_dto.TypeCategoryTemporal, EngineName: "date"},
		"time without time zone":      {Category: querier_dto.TypeCategoryTemporal, EngineName: "time"},
		"time":                        {Category: querier_dto.TypeCategoryTemporal, EngineName: "time"},
		"time with time zone":         {Category: querier_dto.TypeCategoryTemporal, EngineName: "timetz"},
		"timetz":                      {Category: querier_dto.TypeCategoryTemporal, EngineName: "timetz"},
		"interval":                    {Category: querier_dto.TypeCategoryTemporal, EngineName: "interval"},

		// JSON types
		"json": {Category: querier_dto.TypeCategoryJSON, EngineName: "json"},

		// UUID
		"uuid": {Category: querier_dto.TypeCategoryUUID, EngineName: "uuid"},

		// Compound types (bare keywords - compound type parsing provides fields)
		"struct": {Category: querier_dto.TypeCategoryStruct, EngineName: "struct"},
		"map":    {Category: querier_dto.TypeCategoryMap, EngineName: "map"},
		"union":  {Category: querier_dto.TypeCategoryUnion, EngineName: "union"},

		// Void
		"void": {Category: querier_dto.TypeCategoryUnknown, EngineName: "void"},
	}
)

// buildTypeCatalogue combines the built-in DuckDB type set with any extension-supplied
// additions into a single TypeCatalogue.
//
// Takes extraTypes (map[string]querier_dto.SQLType) which holds additional named types to
// merge over the built-in set.
//
// Returns *querier_dto.TypeCatalogue which is the merged catalogue.
func buildTypeCatalogue(extraTypes map[string]querier_dto.SQLType) *querier_dto.TypeCatalogue {
	catalogue := &querier_dto.TypeCatalogue{
		Types: make(map[string]querier_dto.SQLType, len(builtinTypeMap)+len(extraTypes)),
	}
	maps.Copy(catalogue.Types, builtinTypeMap)
	maps.Copy(catalogue.Types, extraTypes)
	return catalogue
}

// normaliseTypeName resolves a raw type name into a structured SQLType, consulting an
// optional engine hook before falling back to array detection and the built-in lookup.
//
// Takes name (string) which is the raw type name as written.
// Takes hook (func(string, []int) *querier_dto.SQLType) which lets engines override
// resolution; may be nil.
// Takes modifiers (...int) which holds numeric modifiers such as precision and scale.
//
// Returns querier_dto.SQLType which is the normalised type.
func normaliseTypeName(
	name string,
	hook func(string, []int) *querier_dto.SQLType,
	modifiers ...int,
) querier_dto.SQLType {
	lowered := strings.ToLower(strings.TrimSpace(name))

	if hook != nil {
		if result := hook(lowered, modifiers); result != nil {
			return *result
		}
	}

	if lowered == "" {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}
	}

	if result, matched := normaliseArrayType(lowered, hook, modifiers); matched {
		return result
	}

	return lookupBuiltinType(lowered, modifiers)
}

// normaliseArrayType strips any trailing [] suffixes and resolves the element type
// recursively.
//
// Takes lowered (string) which is the lower-cased type name.
// Takes hook (func(string, []int) *querier_dto.SQLType) which is the optional engine
// override hook.
// Takes modifiers ([]int) which holds numeric modifiers to apply to the element type.
//
// Returns querier_dto.SQLType which is the array type when matched.
// Returns bool which is true when lowered carried at least one array suffix.
func normaliseArrayType(
	lowered string,
	hook func(string, []int) *querier_dto.SQLType,
	modifiers []int,
) (querier_dto.SQLType, bool) {
	baseName, found := strings.CutSuffix(lowered, arraySubscriptSuffix)
	if !found {
		return querier_dto.SQLType{}, false
	}
	for {
		trimmed, more := strings.CutSuffix(baseName, arraySubscriptSuffix)
		if !more {
			break
		}
		baseName = trimmed
	}
	return querier_dto.SQLType{
		Category:    querier_dto.TypeCategoryArray,
		EngineName:  lowered,
		ElementType: new(normaliseTypeName(baseName, hook, modifiers...)),
	}, true
}

// lookupBuiltinType resolves a known DuckDB type name and applies any numeric modifiers.
//
// Takes lowered (string) which is the lower-cased type name.
// Takes modifiers ([]int) which holds numeric modifiers.
//
// Returns querier_dto.SQLType which is the resolved type, or an unknown-category fallback
// when lowered is not in the map.
func lookupBuiltinType(lowered string, modifiers []int) querier_dto.SQLType {
	if sqlType, exists := builtinTypeMap[lowered]; exists {
		result := sqlType
		applyModifiers(&result, modifiers)
		return result
	}
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: lowered}
}

// applyModifiers writes numeric modifier values onto a SQLType according to its category.
//
// Takes sqlType (*querier_dto.SQLType) which is mutated in place.
// Takes modifiers ([]int) which holds the modifier values to apply.
func applyModifiers(sqlType *querier_dto.SQLType, modifiers []int) {
	if len(modifiers) == 0 {
		return
	}
	switch sqlType.Category {
	case querier_dto.TypeCategoryDecimal:
		if len(modifiers) >= 1 {
			sqlType.Precision = new(modifiers[0])
		}
		if len(modifiers) >= 2 {
			sqlType.Scale = new(modifiers[1])
		}
	case querier_dto.TypeCategoryText:
		if len(modifiers) >= 1 {
			sqlType.Length = new(modifiers[0])
		}
	case querier_dto.TypeCategoryTemporal:
		if len(modifiers) >= 1 {
			sqlType.Precision = new(modifiers[0])
		}
	default:
	}
}

// integerPromotionRank returns the width rank for an integer type.
//
// Unsigned variants are interleaved: utinyint < int2 < usmallint < int4 < uinteger < int8
// < ubigint < hugeint < uhugeint.
//
// Takes engineName (string) which is the canonical engine type name.
//
// Returns int which is the rank, defaulting to int4's rank for unknown names.
func integerPromotionRank(engineName string) int {
	switch engineName {
	case "int1":
		return integerPromotionRankInt1
	case "utinyint":
		return integerPromotionRankUtinyint
	case "int2":
		return integerPromotionRankInt2
	case "usmallint":
		return integerPromotionRankUsmall
	case "uinteger":
		return integerPromotionRankUint
	case "int8":
		return integerPromotionRankInt8
	case "ubigint":
		return integerPromotionRankUbigint
	case "hugeint":
		return integerPromotionRankHuge
	case "uhugeint":
		return integerPromotionRankUhuge
	default:
		return integerPromotionRankInt4
	}
}

// floatPromotionRank returns the width rank for a float type.
//
// Takes engineName (string) which is the canonical engine type name.
//
// Returns int which is 1 for float4 and 2 otherwise.
func floatPromotionRank(engineName string) int {
	switch engineName {
	case "float4":
		return 1
	default:
		return 2
	}
}
