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
	integerRankTinyint = 1

	integerRankSmallint = 2

	integerRankMediumint = 3

	integerRankInt = 4

	integerRankBigint = 5

	integerRankBigintUns = 6

	integerRankDefault = integerRankInt
)

var builtinTypeMap = map[string]querier_dto.SQLType{
	// Signed integer types
	"tinyint":   {Category: querier_dto.TypeCategoryInteger, EngineName: "tinyint"},
	"smallint":  {Category: querier_dto.TypeCategoryInteger, EngineName: "smallint"},
	"mediumint": {Category: querier_dto.TypeCategoryInteger, EngineName: "mediumint"},
	"int":       {Category: querier_dto.TypeCategoryInteger, EngineName: "int"},
	"integer":   {Category: querier_dto.TypeCategoryInteger, EngineName: "int"},
	"bigint":    {Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},

	// Unsigned integer types
	"tinyint unsigned":   {Category: querier_dto.TypeCategoryInteger, EngineName: "tinyint unsigned"},
	"smallint unsigned":  {Category: querier_dto.TypeCategoryInteger, EngineName: "smallint unsigned"},
	"mediumint unsigned": {Category: querier_dto.TypeCategoryInteger, EngineName: "mediumint unsigned"},
	"int unsigned":       {Category: querier_dto.TypeCategoryInteger, EngineName: "int unsigned"},
	"integer unsigned":   {Category: querier_dto.TypeCategoryInteger, EngineName: "int unsigned"},
	"bigint unsigned":    {Category: querier_dto.TypeCategoryInteger, EngineName: "bigint unsigned"},

	// Float types
	"float":            {Category: querier_dto.TypeCategoryFloat, EngineName: "float"},
	"double":           {Category: querier_dto.TypeCategoryFloat, EngineName: "double"},
	"double precision": {Category: querier_dto.TypeCategoryFloat, EngineName: "double"},
	"real":             {Category: querier_dto.TypeCategoryFloat, EngineName: "double"},

	// Decimal types
	"decimal": {Category: querier_dto.TypeCategoryDecimal, EngineName: "decimal"},
	"dec":     {Category: querier_dto.TypeCategoryDecimal, EngineName: "decimal"},
	"numeric": {Category: querier_dto.TypeCategoryDecimal, EngineName: "decimal"},
	"fixed":   {Category: querier_dto.TypeCategoryDecimal, EngineName: "decimal"},

	// Boolean
	"boolean": {Category: querier_dto.TypeCategoryBoolean, EngineName: "tinyint"},
	"bool":    {Category: querier_dto.TypeCategoryBoolean, EngineName: "tinyint"},

	// Text types
	"char":       {Category: querier_dto.TypeCategoryText, EngineName: "char"},
	"varchar":    {Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
	"tinytext":   {Category: querier_dto.TypeCategoryText, EngineName: "tinytext"},
	"text":       {Category: querier_dto.TypeCategoryText, EngineName: "text"},
	"mediumtext": {Category: querier_dto.TypeCategoryText, EngineName: "mediumtext"},
	"longtext":   {Category: querier_dto.TypeCategoryText, EngineName: "longtext"},

	// Binary types
	"binary":     {Category: querier_dto.TypeCategoryBytea, EngineName: "binary"},
	"varbinary":  {Category: querier_dto.TypeCategoryBytea, EngineName: "varbinary"},
	"tinyblob":   {Category: querier_dto.TypeCategoryBytea, EngineName: "tinyblob"},
	"blob":       {Category: querier_dto.TypeCategoryBytea, EngineName: "blob"},
	"mediumblob": {Category: querier_dto.TypeCategoryBytea, EngineName: "mediumblob"},
	"longblob":   {Category: querier_dto.TypeCategoryBytea, EngineName: "longblob"},

	// Temporal types
	"date":      {Category: querier_dto.TypeCategoryTemporal, EngineName: "date"},
	"time":      {Category: querier_dto.TypeCategoryTemporal, EngineName: "time"},
	"datetime":  {Category: querier_dto.TypeCategoryTemporal, EngineName: "datetime"},
	"timestamp": {Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp"},
	"year":      {Category: querier_dto.TypeCategoryTemporal, EngineName: "year"},

	// JSON
	"json": {Category: querier_dto.TypeCategoryJSON, EngineName: "json"},

	// Geometric types
	"geometry":           {Category: querier_dto.TypeCategoryGeometric, EngineName: "geometry"},
	"point":              {Category: querier_dto.TypeCategoryGeometric, EngineName: "point"},
	"linestring":         {Category: querier_dto.TypeCategoryGeometric, EngineName: "linestring"},
	"polygon":            {Category: querier_dto.TypeCategoryGeometric, EngineName: "polygon"},
	"multipoint":         {Category: querier_dto.TypeCategoryGeometric, EngineName: "multipoint"},
	"multilinestring":    {Category: querier_dto.TypeCategoryGeometric, EngineName: "multilinestring"},
	"multipolygon":       {Category: querier_dto.TypeCategoryGeometric, EngineName: "multipolygon"},
	"geometrycollection": {Category: querier_dto.TypeCategoryGeometric, EngineName: "geometrycollection"},

	// Other types
	"enum": {Category: querier_dto.TypeCategoryEnum, EngineName: "enum"},
	"set":  {Category: querier_dto.TypeCategoryText, EngineName: "set"},
	"bit":  {Category: querier_dto.TypeCategoryInteger, EngineName: "bit"},
}

var multiWordTypes = map[string]string{
	"double precision": "double precision",
}

// buildTypeCatalogue constructs a TypeCatalogue from the built-in MySQL types
// merged with any user-provided extra type mappings.
func buildTypeCatalogue(extraTypes map[string]querier_dto.SQLType) *querier_dto.TypeCatalogue {
	catalogue := &querier_dto.TypeCatalogue{
		Types: make(map[string]querier_dto.SQLType, len(builtinTypeMap)+len(extraTypes)),
	}
	maps.Copy(catalogue.Types, builtinTypeMap)
	maps.Copy(catalogue.Types, extraTypes)
	return catalogue
}

// normaliseTypeName resolves a raw SQL type name string to a structured SQLType.
// It consults the hook first, then multi-word types, then built-in types,
// falling back to Unknown for unrecognised names.
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
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}
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

	return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: lowered}
}

// applyModifiers sets precision, scale, or length on the given SQLType based
// on the type category and the provided modifier values.
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
// Unsigned variants rank one step wider than their signed counterparts.
func integerPromotionRank(engineName string) int {
	rank, exists := integerRanks[engineName]
	if exists {
		return rank
	}
	return integerRankDefault
}

var integerRanks = map[string]int{
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

// floatPromotionRank returns the numeric width rank for MySQL float types.
func floatPromotionRank(engineName string) int {
	switch engineName {
	case "float":
		return 1
	default:
		return 2
	}
}
