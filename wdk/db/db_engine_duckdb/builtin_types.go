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
	integerPromotionRankInt1 = 1

	integerPromotionRankUtinyint = 2

	integerPromotionRankInt2 = 3

	integerPromotionRankUsmall = 4

	integerPromotionRankInt4 = 5

	integerPromotionRankUint = 6

	integerPromotionRankInt8 = 7

	integerPromotionRankUbigint = 8

	integerPromotionRankHuge = 9

	integerPromotionRankUhuge = 10
)

var builtinTypeMap = map[string]querier_dto.SQLType{
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

func buildTypeCatalogue(extraTypes map[string]querier_dto.SQLType) *querier_dto.TypeCatalogue {
	catalogue := &querier_dto.TypeCatalogue{
		Types: make(map[string]querier_dto.SQLType, len(builtinTypeMap)+len(extraTypes)),
	}
	maps.Copy(catalogue.Types, builtinTypeMap)
	maps.Copy(catalogue.Types, extraTypes)
	return catalogue
}

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

func lookupBuiltinType(lowered string, modifiers []int) querier_dto.SQLType {
	if sqlType, exists := builtinTypeMap[lowered]; exists {
		result := sqlType
		applyModifiers(&result, modifiers)
		return result
	}
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: lowered}
}

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

// integerPromotionRank returns the numeric width rank for DuckDB integer types.
// Unsigned variants are interleaved: utinyint < int2 < usmallint < int4 <
// uinteger < int8 < ubigint < hugeint < uhugeint.
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

// floatPromotionRank returns the numeric width rank for DuckDB float types.
func floatPromotionRank(engineName string) int {
	switch engineName {
	case "float4":
		return 1
	default:
		return 2
	}
}
