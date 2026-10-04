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

package db_engine_postgres

import (
	"maps"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (

	// integerPromotionRankWidest is the rank assigned to the widest integer width (bigint /
	// int8) during numeric type promotion.
	integerPromotionRankWidest = 3
)

var (
	// builtinTypeMap holds the Postgres builtin SQL types keyed by their canonical lowercase
	// spelling.
	builtinTypeMap = map[string]querier_dto.SQLType{
		// Integer types
		"smallint":    querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int2"),
		"int2":        querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int2"),
		"integer":     querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4"),
		"int":         querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4"),
		"int4":        querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4"),
		"bigint":      querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int8"),
		"int8":        querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int8"),
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
		"float8":           querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "float8"),
		"float":            querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "float8"),
		// Decimal types
		"numeric": querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "numeric"),
		"decimal": querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "numeric"),
		// Boolean
		"boolean": querier_dto.NewSQLType(querier_dto.TypeCategoryBoolean, "bool"),
		"bool":    querier_dto.NewSQLType(querier_dto.TypeCategoryBoolean, "bool"),
		// Text types
		"text":              querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text"),
		"character varying": querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varchar"),
		"varchar":           querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varchar"),
		"character":         querier_dto.NewSQLType(querier_dto.TypeCategoryText, "char"),
		"char":              querier_dto.NewSQLType(querier_dto.TypeCategoryText, "char"),
		"bpchar":            querier_dto.NewSQLType(querier_dto.TypeCategoryText, "char"),
		"name":              querier_dto.NewSQLType(querier_dto.TypeCategoryText, "name"),
		"citext":            querier_dto.NewSQLType(querier_dto.TypeCategoryText, "citext"),
		// Bytea
		"bytea": querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "bytea"),
		// Bit string types (represented as Go strings of '0'/'1')
		"bit":         querier_dto.NewSQLType(querier_dto.TypeCategoryText, "bit"),
		"bit varying": querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varbit"),
		"varbit":      querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varbit"),
		// Temporal types
		"timestamp without time zone": querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamp"),
		"timestamp":                   querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamp"),
		"timestamp with time zone":    querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamptz"),
		"timestamptz":                 querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamptz"),
		"date":                        querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "date"),
		"time without time zone":      querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "time"),
		"time":                        querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "time"),
		"time with time zone":         querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timetz"),
		"timetz":                      querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timetz"),
		"interval":                    querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "interval"),
		// JSON types
		"json":     querier_dto.NewSQLType(querier_dto.TypeCategoryJSON, "json"),
		"jsonb":    querier_dto.NewSQLType(querier_dto.TypeCategoryJSON, "jsonb"),
		"jsonpath": querier_dto.NewSQLType(querier_dto.TypeCategoryText, "jsonpath"),
		// UUID
		"uuid": querier_dto.NewSQLType(querier_dto.TypeCategoryUUID, "uuid"),
		// Network types
		"inet":     querier_dto.NewSQLType(querier_dto.TypeCategoryNetwork, "inet"),
		"cidr":     querier_dto.NewSQLType(querier_dto.TypeCategoryNetwork, "cidr"),
		"macaddr":  querier_dto.NewSQLType(querier_dto.TypeCategoryNetwork, "macaddr"),
		"macaddr8": querier_dto.NewSQLType(querier_dto.TypeCategoryNetwork, "macaddr8"),
		// Geometric types
		"point":   querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "point"),
		"line":    querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "line"),
		"lseg":    querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "lseg"),
		"box":     querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "box"),
		"path":    querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "path"),
		"polygon": querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "polygon"),
		"circle":  querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "circle"),
		// Range types
		"int4range":      querier_dto.NewSQLType(querier_dto.TypeCategoryRange, "int4range"),
		"int8range":      querier_dto.NewSQLType(querier_dto.TypeCategoryRange, "int8range"),
		"numrange":       querier_dto.NewSQLType(querier_dto.TypeCategoryRange, "numrange"),
		"tsrange":        querier_dto.NewSQLType(querier_dto.TypeCategoryRange, "tsrange"),
		"tstzrange":      querier_dto.NewSQLType(querier_dto.TypeCategoryRange, "tstzrange"),
		"daterange":      querier_dto.NewSQLType(querier_dto.TypeCategoryRange, "daterange"),
		"int4multirange": querier_dto.NewSQLType(querier_dto.TypeCategoryRange, "int4multirange"),
		"int8multirange": querier_dto.NewSQLType(querier_dto.TypeCategoryRange, "int8multirange"),
		"nummultirange":  querier_dto.NewSQLType(querier_dto.TypeCategoryRange, "nummultirange"),
		"tsmultirange":   querier_dto.NewSQLType(querier_dto.TypeCategoryRange, "tsmultirange"),
		"tstzmultirange": querier_dto.NewSQLType(querier_dto.TypeCategoryRange, "tstzmultirange"),
		"datemultirange": querier_dto.NewSQLType(querier_dto.TypeCategoryRange, "datemultirange"),
		// Other system types
		"oid":      querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "oid"),
		"money":    querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "money"),
		"xml":      querier_dto.NewSQLType(querier_dto.TypeCategoryText, "xml"),
		"tsvector": querier_dto.NewSQLType(querier_dto.TypeCategoryText, "tsvector"),
		"tsquery":  querier_dto.NewSQLType(querier_dto.TypeCategoryText, "tsquery"),
		"regtype":  querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "regtype"),
		"regclass": querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "regclass"),
		"pg_lsn":   querier_dto.NewSQLType(querier_dto.TypeCategoryText, "pg_lsn"),
		"void":     querier_dto.NewSQLType(querier_dto.TypeCategoryUnknown, "void"),
	}

	// multiWordTypes maps lowercase multi-word Postgres type spellings to themselves so the
	// parser can recognise them as a single type token.
	multiWordTypes = map[string]string{
		"double precision":            "double precision",
		"character varying":           "character varying",
		"timestamp without time zone": "timestamp without time zone",
		"timestamp with time zone":    "timestamp with time zone",
		"time without time zone":      "time without time zone",
		"time with time zone":         "time with time zone",
		"bit varying":                 "bit varying",
	}
)

// buildTypeCatalogue assembles a type catalogue from builtin and extra types.
//
// Takes extraTypes (map[string]querier_dto.SQLType) which provides additional types
// layered over the builtins.
//
// Returns *querier_dto.TypeCatalogue which holds the merged type map.
func buildTypeCatalogue(extraTypes map[string]querier_dto.SQLType) *querier_dto.TypeCatalogue {
	catalogue := &querier_dto.TypeCatalogue{
		Types: make(map[string]querier_dto.SQLType, len(builtinTypeMap)+len(extraTypes)),
	}
	maps.Copy(catalogue.Types, builtinTypeMap)
	maps.Copy(catalogue.Types, extraTypes)
	return catalogue
}

// normaliseTypeName resolves a Postgres type name to its canonical SQLType.
//
// Takes name (string) which is the raw type spelling to resolve.
// Takes hook (func(string, []int) *querier_dto.SQLType) which optionally overrides the
// resolution for engine-specific types.
// Takes modifiers (...int) which provides precision, scale, or length modifiers applied
// to the resolved type.
//
// Returns querier_dto.SQLType which describes the resolved type.
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
		return querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text")
	}

	if baseName, found := strings.CutSuffix(lowered, arraySubscriptSuffix); found {
		dimensions := 1
		for {
			trimmed, more := strings.CutSuffix(baseName, arraySubscriptSuffix)
			if !more {
				break
			}
			baseName = trimmed
			dimensions++
		}
		arrayType := querier_dto.SQLType{}
		arrayType.Category = querier_dto.TypeCategoryArray
		arrayType.EngineName = lowered
		arrayType.ElementType = new(normaliseTypeName(baseName, hook, modifiers...))
		return arrayType
	}

	if _, exists := multiWordTypes[lowered]; exists {
		if sqlType, found := builtinTypeMap[lowered]; found {
			result := sqlType
			applyModifiers(&result, modifiers)
			return result
		}
	}

	if sqlType, exists := builtinTypeMap[lowered]; exists {
		result := sqlType
		applyModifiers(&result, modifiers)
		return result
	}

	return querier_dto.NewSQLType(querier_dto.TypeCategoryUnknown, lowered)
}

// applyModifiers writes precision, scale, or length modifiers onto sqlType.
//
// Takes sqlType (*querier_dto.SQLType) which is the target type to mutate.
// Takes modifiers ([]int) which provides the modifier values in declaration order.
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

// integerPromotionRank returns the numeric width rank for PG integer types.
//
// Takes engineName (string) which is the canonical engine type name.
//
// Returns int which is the width rank used for promotion ordering.
func integerPromotionRank(engineName string) int {
	switch engineName {
	case "int2":
		return 1
	case "int8":
		return integerPromotionRankWidest
	default:
		return 2
	}
}

// floatPromotionRank returns the numeric width rank for PG float types.
//
// Takes engineName (string) which is the canonical engine type name.
//
// Returns int which is the width rank used for promotion ordering.
func floatPromotionRank(engineName string) int {
	switch engineName {
	case "float4":
		return 1
	default:
		return 2
	}
}
