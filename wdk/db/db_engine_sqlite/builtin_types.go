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

package db_engine_sqlite

import (
	"maps"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

var (
	// builtinTypeMap maps SQLite type spellings to normalised SQL types.
	//
	// This includes all SQLite type-affinity aliases. Integer spellings carry their declared
	// width as the canonical int2/int4/int8 engine name so the Go type matches the column's
	// declared range and stays consistent with the postgres family, rather than collapsing
	// every width to a single fallback. SQLite stores all integers as 64-bit, so a column
	// that needs more than its declared width (for example a key beyond the 32-bit range)
	// must be declared BIGINT to map to int64.
	builtinTypeMap = map[string]querier_dto.SQLType{
		"integer":   querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4"),
		"int":       querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4"),
		"mediumint": querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4"),
		"tinyint":   querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int2"),
		"smallint":  querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int2"),
		"int2":      querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int2"),
		"bigint":    querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int8"),
		"int8":      querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int8"),

		"text":      querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text"),
		"clob":      querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text"),
		"varchar":   querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text"),
		"nchar":     querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text"),
		"nvarchar":  querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text"),
		"char":      querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text"),
		"character": querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text"),

		"real":   querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "real"),
		"double": querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "real"),
		"float":  querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "real"),

		"blob": querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "blob"),

		"any": querier_dto.NewSQLType(querier_dto.TypeCategoryUnknown, "any"),

		"numeric": querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "numeric"),
		"decimal": querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "numeric"),

		"boolean": querier_dto.NewSQLType(querier_dto.TypeCategoryBoolean, "boolean"),
		"bool":    querier_dto.NewSQLType(querier_dto.TypeCategoryBoolean, "boolean"),

		"date":      querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "date"),
		"datetime":  querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "datetime"),
		"timestamp": querier_dto.NewSQLType(querier_dto.TypeCategoryTemporal, "timestamp"),

		"json": querier_dto.NewSQLType(querier_dto.TypeCategoryJSON, "json"),
	}
)

// buildTypeCatalogue constructs the SQLite type catalogue.
//
// Returns *querier_dto.TypeCatalogue which holds every built-in SQLite type spelling
// keyed by lowercase name.
func buildTypeCatalogue() *querier_dto.TypeCatalogue {
	catalogue := &querier_dto.TypeCatalogue{
		Types: make(map[string]querier_dto.SQLType, len(builtinTypeMap)),
	}
	maps.Copy(catalogue.Types, builtinTypeMap)
	return catalogue
}

// normaliseTypeName resolves a raw SQLite type spelling to a SQL type.
//
// Takes name (string) which is the raw type spelling from the source.
// Takes modifiers (...int) which holds optional precision and scale digits parsed from
// the type parentheses.
//
// Returns querier_dto.SQLType which is the normalised SQL type.
func normaliseTypeName(name string, modifiers ...int) querier_dto.SQLType {
	lowered := strings.ToLower(strings.TrimSpace(name))

	if lowered == "" {
		return querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "blob")
	}

	if sqlType, exists := builtinTypeMap[lowered]; exists {
		result := sqlType
		applyModifiers(&result, modifiers)
		return result
	}

	return normaliseByAffinity(lowered, modifiers)
}

// normaliseByAffinity falls back to the SQLite affinity rules.
//
// Takes lowered (string) which is the lowercase trimmed type name.
// Takes modifiers ([]int) which holds optional precision and scale digits.
//
// Returns querier_dto.SQLType which is the SQL type chosen by affinity match.
func normaliseByAffinity(lowered string, modifiers []int) querier_dto.SQLType {
	upper := strings.ToUpper(lowered)

	if strings.Contains(upper, "INT") {
		return querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4")
	}

	if strings.Contains(upper, "CHAR") || strings.Contains(upper, "CLOB") || strings.Contains(upper, "TEXT") {
		result := querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text")
		applyModifiers(&result, modifiers)
		return result
	}

	if strings.Contains(upper, "BLOB") {
		return querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "blob")
	}

	if strings.Contains(upper, "REAL") || strings.Contains(upper, "FLOA") || strings.Contains(upper, "DOUB") {
		return querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "real")
	}

	result := querier_dto.NewSQLType(querier_dto.TypeCategoryDecimal, "numeric")
	applyModifiers(&result, modifiers)
	return result
}

// applyModifiers attaches precision, length, and scale to a SQL type.
//
// Takes sqlType (*querier_dto.SQLType) which receives the modifier values.
// Takes modifiers ([]int) which holds precision and scale digits in order.
func applyModifiers(sqlType *querier_dto.SQLType, modifiers []int) {
	if sqlType.Category != querier_dto.TypeCategoryDecimal && sqlType.Category != querier_dto.TypeCategoryText {
		return
	}
	if len(modifiers) >= 1 {
		precision := modifiers[0]
		if sqlType.Category == querier_dto.TypeCategoryText {
			sqlType.Length = &precision
		} else {
			sqlType.Precision = &precision
		}
	}
	if len(modifiers) >= 2 && sqlType.Category == querier_dto.TypeCategoryDecimal {
		sqlType.Scale = new(modifiers[1])
	}
}
