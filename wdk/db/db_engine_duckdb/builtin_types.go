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
		"tinyint":  querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int1"),
		"int1":     querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int1"),
		"smallint": querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int2"),
		"int2":     querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int2"),
		"integer":  querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4"),
		"int":      querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4"),
		"int4":     querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4"),
		"bigint":   querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int8"),
		"int8":     querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int8"),
		"hugeint":  querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "hugeint"),

		// Integer types (unsigned)
		"utinyint":  querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "utinyint"),
		"usmallint": querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "usmallint"),
		"uinteger":  querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "uinteger"),
		"ubigint":   querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "ubigint"),
		"uhugeint":  querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "uhugeint"),

		// Serial types (normalise to underlying integer)
		"smallserial": querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int2"),
		"serial2":     querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int2"),
		"serial":      querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4"),
		"serial4":     querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4"),
		"bigserial":   querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int8"),
		"serial8":     querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int8"),

		// Float types
		"real":             querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "float4"),
		"float4":           querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "float4"),
		"double precision": querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "float8"),
		"double":           querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "float8"),
		"float8":           querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "float8"),
		"float":            querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "float8"),

		// Decimal types
		"numeric": querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "numeric"),
		"decimal": querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "numeric"),

		// Boolean
		"boolean": querier_dto.NewSQLType(querier_dto.TypeCategoryBoolean, "bool"),
		"bool":    querier_dto.NewSQLType(querier_dto.TypeCategoryBoolean, "bool"),

		// Text types (DuckDB canonical text type is varchar)
		"text":              querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varchar"),
		"varchar":           querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varchar"),
		"character varying": querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varchar"),
		"character":         querier_dto.NewSQLType(querier_dto.TypeCategoryText, "char"),
		"char":              querier_dto.NewSQLType(querier_dto.TypeCategoryText, "char"),
		"bpchar":            querier_dto.NewSQLType(querier_dto.TypeCategoryText, "char"),
		"name":              querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varchar"),

		// Binary types
		"bytea": querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "blob"),
		"blob":  querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "blob"),

		// Temporal types
		"timestamp without time zone": querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamp"),
		"timestamp":                   querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamp"),
		"timestamp with time zone":    querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamptz"),
		"timestamptz":                 querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamptz"),
		"timestamp_s":                 querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamp_s"),
		"timestamp_ms":                querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamp_ms"),
		"timestamp_ns":                querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamp_ns"),
		"date":                        querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "date"),
		"time without time zone":      querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "time"),
		"time":                        querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "time"),
		"time with time zone":         querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timetz"),
		"timetz":                      querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timetz"),
		"interval":                    querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "interval"),

		// JSON types
		"json": querier_dto.NewSQLType(querier_dto.TypeCategoryJSON, "json"),

		// UUID
		"uuid": querier_dto.NewSQLType(querier_dto.TypeCategoryUUID, "uuid"),

		// Compound types (bare keywords - compound type parsing provides fields)
		"struct": querier_dto.NewSQLType(querier_dto.TypeCategoryStruct, "struct"),
		"map":    querier_dto.NewSQLType(querier_dto.TypeCategoryMap, "map"),
		"union":  querier_dto.NewSQLType(querier_dto.TypeCategoryUnion, "union"),

		// Void
		"void": querier_dto.NewSQLType(querier_dto.TypeCategoryUnknown, "void"),
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
		return querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varchar")
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
	arrayType := querier_dto.SQLType{}
	arrayType.Category = querier_dto.TypeCategoryArray
	arrayType.EngineName = lowered
	arrayType.ElementType = new(normaliseTypeName(baseName, hook, modifiers...))
	return arrayType, true
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
	return querier_dto.NewSQLType(querier_dto.TypeCategoryUnknown, lowered)
}

// applyModifiers writes numeric modifier values onto a SQLType according to its category.
// A precision or length must be at least one and a scale at least zero, and every value
// must be within maxTypeModifierValue; out-of-range modifiers are ignored rather than
// written into the catalogue.
//
// Takes sqlType (*querier_dto.SQLType) which is mutated in place.
// Takes modifiers ([]int) which holds the modifier values to apply.
func applyModifiers(sqlType *querier_dto.SQLType, modifiers []int) {
	if len(modifiers) == 0 {
		return
	}
	switch sqlType.Category {
	case querier_dto.TypeCategoryDecimal:
		assignPositiveModifier(&sqlType.Precision, modifiers, 0)
		assignScaleModifier(&sqlType.Scale, modifiers, 1)
	case querier_dto.TypeCategoryText:
		assignPositiveModifier(&sqlType.Length, modifiers, 0)
	case querier_dto.TypeCategoryTemporal:
		assignPositiveModifier(&sqlType.Precision, modifiers, 0)
	default:
	}
}

// assignPositiveModifier writes the modifier at index into target when it is present and
// is a positive value within the accepted ceiling.
//
// Takes target (**int) which receives a pointer to the validated value.
// Takes modifiers ([]int) which holds the candidate values.
// Takes index (int) which is the position of the candidate within modifiers.
func assignPositiveModifier(target **int, modifiers []int, index int) {
	if index >= len(modifiers) {
		return
	}
	value := modifiers[index]
	if value < 1 || value > maxTypeModifierValue {
		return
	}
	*target = new(value)
}

// assignScaleModifier writes the modifier at index into target when it is present and is
// a non-negative scale within the accepted ceiling.
//
// Takes target (**int) which receives a pointer to the validated value.
// Takes modifiers ([]int) which holds the candidate values.
// Takes index (int) which is the position of the candidate within modifiers.
func assignScaleModifier(target **int, modifiers []int, index int) {
	if index >= len(modifiers) {
		return
	}
	value := modifiers[index]
	if value < 0 || value > maxTypeModifierValue {
		return
	}
	*target = new(value)
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

// newArrayType returns the DuckDB list type whose elements are element, named with the
// element's name and a [] suffix.
//
// Takes element (querier_dto.SQLType) which is the element type.
//
// Returns querier_dto.SQLType which is the array type.
func newArrayType(element querier_dto.SQLType) querier_dto.SQLType {
	arrayType := querier_dto.NewSQLType(querier_dto.TypeCategoryArray, element.EngineName+arraySubscriptSuffix)
	arrayType.ElementType = &element
	return arrayType
}
