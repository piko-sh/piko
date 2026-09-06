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

package db_engine_mysql

import (
	"maps"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// integerRankTinyint is the promotion rank assigned to TINYINT.
	integerRankTinyint = 1

	// integerRankSmallint is the promotion rank assigned to SMALLINT and unsigned TINYINT.
	integerRankSmallint = 2

	// integerRankMediumint is the promotion rank assigned to MEDIUMINT and unsigned
	// SMALLINT.
	integerRankMediumint = 3

	// integerRankInt is the promotion rank assigned to INT and unsigned MEDIUMINT.
	integerRankInt = 4

	// integerRankBigint is the promotion rank assigned to BIGINT and unsigned INT.
	integerRankBigint = 5

	// integerRankBigintUns is the promotion rank assigned to unsigned BIGINT.
	integerRankBigintUns = 6

	// integerRankDefault is the rank used for unrecognised integer engine names.
	integerRankDefault = integerRankInt

	// floatRankSingle is the promotion rank assigned to single-precision FLOAT.
	floatRankSingle = 1

	// floatRankDouble is the promotion rank assigned to DOUBLE and all other float names.
	floatRankDouble = 2
)

var (
	// builtinTypeMap maps lowercase MySQL type names to their structured SQLType
	// descriptors.
	builtinTypeMap = map[string]querier_dto.SQLType{
		// Signed integer types
		"tinyint":   querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "tinyint"),
		"smallint":  querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "smallint"),
		"mediumint": querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "mediumint"),
		"int":       querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int"),
		"integer":   querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int"),
		"bigint":    querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "bigint"),
		// Unsigned integer types
		"tinyint unsigned":   querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "tinyint unsigned"),
		"smallint unsigned":  querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "smallint unsigned"),
		"mediumint unsigned": querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "mediumint unsigned"),
		"int unsigned":       querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int unsigned"),
		"integer unsigned":   querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int unsigned"),
		"bigint unsigned":    querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "bigint unsigned"),
		// Float types
		"float":            querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "float"),
		"double":           querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "double"),
		"double precision": querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "double"),
		"real":             querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "double"),
		// Decimal types
		"decimal": querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "decimal"),
		"dec":     querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "decimal"),
		"numeric": querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "decimal"),
		"fixed":   querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "decimal"),
		// Boolean
		"boolean": querier_dto.NewSQLType(querier_dto.TypeCategoryBoolean, "tinyint"),
		"bool":    querier_dto.NewSQLType(querier_dto.TypeCategoryBoolean, "tinyint"),
		// Text types
		"char":       querier_dto.NewSQLType(querier_dto.TypeCategoryText, "char"),
		"varchar":    querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varchar"),
		"tinytext":   querier_dto.NewSQLType(querier_dto.TypeCategoryText, "tinytext"),
		"text":       querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text"),
		"mediumtext": querier_dto.NewSQLType(querier_dto.TypeCategoryText, "mediumtext"),
		"longtext":   querier_dto.NewSQLType(querier_dto.TypeCategoryText, "longtext"),
		// Binary types
		"binary":     querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "binary"),
		"varbinary":  querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "varbinary"),
		"tinyblob":   querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "tinyblob"),
		"blob":       querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "blob"),
		"mediumblob": querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "mediumblob"),
		"longblob":   querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "longblob"),
		// Temporal types
		"date":      querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "date"),
		"time":      querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "time"),
		"datetime":  querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "datetime"),
		"timestamp": querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamp"),
		"year":      querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "year"),
		// JSON
		"json": querier_dto.NewSQLType(querier_dto.TypeCategoryJSON, "json"),
		// Geometric types
		"geometry":           querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "geometry"),
		"point":              querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "point"),
		"linestring":         querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "linestring"),
		"polygon":            querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "polygon"),
		"multipoint":         querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "multipoint"),
		"multilinestring":    querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "multilinestring"),
		"multipolygon":       querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "multipolygon"),
		"geometrycollection": querier_dto.NewSQLType(querier_dto.TypeCategoryGeometric, "geometrycollection"),
		// Other types
		"enum": querier_dto.NewSQLType(querier_dto.TypeCategoryEnum, "enum"),
		"set":  querier_dto.NewSQLType(querier_dto.TypeCategoryText, "set"),
		"bit":  querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "bit"),
	}

	// multiWordTypes lists MySQL type names whose canonical form contains more than one
	// token, used by the lexer to keep them as a single name.
	multiWordTypes = map[string]string{
		"double precision": "double precision",
	}
)

// buildTypeCatalogue constructs a TypeCatalogue from the built-in MySQL types merged with
// any user-provided extra type mappings.
//
// Takes extraTypes (map[string]querier_dto.SQLType) which provides additional type
// mappings to merge in.
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

// normaliseTypeName resolves a raw SQL type name string to a structured SQLType.
//
// Consults the hook first, then multi-word types, then built-in types, falling back to
// Unknown for unrecognised names.
//
// Takes name (string) which is the raw type name to resolve.
// Takes hook (func(string, []int) *querier_dto.SQLType) which is an optional override
// consulted before the built-in tables.
// Takes modifiers (...int) which carry precision, scale, or length values to apply to the
// resolved type.
//
// Returns querier_dto.SQLType which is the resolved type, with modifiers applied where
// applicable.
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

	if base, found := strings.CutSuffix(lowered, " unsigned"); found {
		if sqlType, exists := builtinTypeMap[base]; exists {
			result := sqlType
			applyModifiers(&result, modifiers)
			return result
		}
	}

	return querier_dto.NewSQLType(querier_dto.TypeCategoryUnknown, lowered)
}

// applyModifiers sets precision, scale, or length on the given SQLType based on the type
// category and the provided modifier values.
//
// Takes sqlType (*querier_dto.SQLType) which is the type to mutate.
// Takes modifiers ([]int) which is the slice of modifier values to apply.
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

// integerPromotionRank returns the numeric width rank for MySQL integer types.
//
// Unsigned variants rank one step wider than their signed counterparts.
//
// Takes engineName (string) which is the engine-level integer type name.
//
// Returns int which is the promotion rank, defaulting to integerRankDefault for
// unrecognised names.
func integerPromotionRank(engineName string) int {
	rank, exists := integerRanks[engineName]
	if exists {
		return rank
	}
	return integerRankDefault
}

var (
	// integerRanks maps engine-level integer type names to their promotion rank used during
	// arithmetic type widening.
	integerRanks = map[string]int{
		"tinyint":            integerRankTinyint,
		"smallint":           integerRankSmallint,
		"tinyint unsigned":   integerRankSmallint,
		"mediumint":          integerRankMediumint,
		"smallint unsigned":  integerRankMediumint,
		"int":                integerRankInt,
		"mediumint unsigned": integerRankInt,
		"bigint":             integerRankBigint,
		"int unsigned":       integerRankBigint,
		"bigint unsigned":    integerRankBigintUns,
	}
)

// floatPromotionRank returns the numeric width rank for MySQL float types.
//
// Takes engineName (string) which is the engine-level float type name.
//
// Returns int which is the promotion rank, with FLOAT ranking 1 and all other names
// ranking 2.
func floatPromotionRank(engineName string) int {
	switch engineName {
	case "float":
		return floatRankSingle
	default:
		return floatRankDouble
	}
}
