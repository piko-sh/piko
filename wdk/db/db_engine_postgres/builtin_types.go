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

const integerPromotionRankWidest = 3

var builtinTypeMap = map[string]querier_dto.SQLType{
	// Integer types
	"smallint":    {Category: querier_dto.TypeCategoryInteger, EngineName: "int2"},
	"int2":        {Category: querier_dto.TypeCategoryInteger, EngineName: "int2"},
	"integer":     {Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
	"int":         {Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
	"int4":        {Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
	"bigint":      {Category: querier_dto.TypeCategoryInteger, EngineName: "int8"},
	"int8":        {Category: querier_dto.TypeCategoryInteger, EngineName: "int8"},
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
	"float8":           {Category: querier_dto.TypeCategoryFloat, EngineName: "float8"},
	"float":            {Category: querier_dto.TypeCategoryFloat, EngineName: "float8"},

	// Decimal types
	"numeric": {Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"},
	"decimal": {Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"},

	// Boolean
	"boolean": {Category: querier_dto.TypeCategoryBoolean, EngineName: "bool"},
	"bool":    {Category: querier_dto.TypeCategoryBoolean, EngineName: "bool"},

	// Text types
	"text":              {Category: querier_dto.TypeCategoryText, EngineName: "text"},
	"character varying": {Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
	"varchar":           {Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
	"character":         {Category: querier_dto.TypeCategoryText, EngineName: "char"},
	"char":              {Category: querier_dto.TypeCategoryText, EngineName: "char"},
	"bpchar":            {Category: querier_dto.TypeCategoryText, EngineName: "char"},
	"name":              {Category: querier_dto.TypeCategoryText, EngineName: "name"},
	"citext":            {Category: querier_dto.TypeCategoryText, EngineName: "citext"},

	// Bytea
	"bytea": {Category: querier_dto.TypeCategoryBytea, EngineName: "bytea"},

	// Temporal types
	"timestamp without time zone": {Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp"},
	"timestamp":                   {Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamp"},
	"timestamp with time zone":    {Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamptz"},
	"timestamptz":                 {Category: querier_dto.TypeCategoryTemporal, EngineName: "timestamptz"},
	"date":                        {Category: querier_dto.TypeCategoryTemporal, EngineName: "date"},
	"time without time zone":      {Category: querier_dto.TypeCategoryTemporal, EngineName: "time"},
	"time":                        {Category: querier_dto.TypeCategoryTemporal, EngineName: "time"},
	"time with time zone":         {Category: querier_dto.TypeCategoryTemporal, EngineName: "timetz"},
	"timetz":                      {Category: querier_dto.TypeCategoryTemporal, EngineName: "timetz"},
	"interval":                    {Category: querier_dto.TypeCategoryTemporal, EngineName: "interval"},

	// JSON types
	"json":  {Category: querier_dto.TypeCategoryJSON, EngineName: "json"},
	"jsonb": {Category: querier_dto.TypeCategoryJSON, EngineName: "jsonb"},

	// UUID
	"uuid": {Category: querier_dto.TypeCategoryUUID, EngineName: "uuid"},

	// Network types
	"inet":     {Category: querier_dto.TypeCategoryNetwork, EngineName: "inet"},
	"cidr":     {Category: querier_dto.TypeCategoryNetwork, EngineName: "cidr"},
	"macaddr":  {Category: querier_dto.TypeCategoryNetwork, EngineName: "macaddr"},
	"macaddr8": {Category: querier_dto.TypeCategoryNetwork, EngineName: "macaddr8"},

	// Geometric types
	"point":   {Category: querier_dto.TypeCategoryGeometric, EngineName: "point"},
	"line":    {Category: querier_dto.TypeCategoryGeometric, EngineName: "line"},
	"lseg":    {Category: querier_dto.TypeCategoryGeometric, EngineName: "lseg"},
	"box":     {Category: querier_dto.TypeCategoryGeometric, EngineName: "box"},
	"path":    {Category: querier_dto.TypeCategoryGeometric, EngineName: "path"},
	"polygon": {Category: querier_dto.TypeCategoryGeometric, EngineName: "polygon"},
	"circle":  {Category: querier_dto.TypeCategoryGeometric, EngineName: "circle"},

	// Range types
	"int4range":      {Category: querier_dto.TypeCategoryRange, EngineName: "int4range"},
	"int8range":      {Category: querier_dto.TypeCategoryRange, EngineName: "int8range"},
	"numrange":       {Category: querier_dto.TypeCategoryRange, EngineName: "numrange"},
	"tsrange":        {Category: querier_dto.TypeCategoryRange, EngineName: "tsrange"},
	"tstzrange":      {Category: querier_dto.TypeCategoryRange, EngineName: "tstzrange"},
	"daterange":      {Category: querier_dto.TypeCategoryRange, EngineName: "daterange"},
	"int4multirange": {Category: querier_dto.TypeCategoryRange, EngineName: "int4multirange"},
	"int8multirange": {Category: querier_dto.TypeCategoryRange, EngineName: "int8multirange"},
	"nummultirange":  {Category: querier_dto.TypeCategoryRange, EngineName: "nummultirange"},
	"tsmultirange":   {Category: querier_dto.TypeCategoryRange, EngineName: "tsmultirange"},
	"tstzmultirange": {Category: querier_dto.TypeCategoryRange, EngineName: "tstzmultirange"},
	"datemultirange": {Category: querier_dto.TypeCategoryRange, EngineName: "datemultirange"},

	// Other system types
	"oid":      {Category: querier_dto.TypeCategoryInteger, EngineName: "oid"},
	"money":    {Category: querier_dto.TypeCategoryDecimal, EngineName: "money"},
	"xml":      {Category: querier_dto.TypeCategoryText, EngineName: "xml"},
	"tsvector": {Category: querier_dto.TypeCategoryText, EngineName: "tsvector"},
	"tsquery":  {Category: querier_dto.TypeCategoryText, EngineName: "tsquery"},
	"regtype":  {Category: querier_dto.TypeCategoryInteger, EngineName: "regtype"},
	"regclass": {Category: querier_dto.TypeCategoryInteger, EngineName: "regclass"},
	"pg_lsn":   {Category: querier_dto.TypeCategoryText, EngineName: "pg_lsn"},
	"void":     {Category: querier_dto.TypeCategoryUnknown, EngineName: "void"},
}

var multiWordTypes = map[string]string{
	"double precision":            "double precision",
	"character varying":           "character varying",
	"timestamp without time zone": "timestamp without time zone",
	"timestamp with time zone":    "timestamp with time zone",
	"time without time zone":      "time without time zone",
	"time with time zone":         "time with time zone",
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
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}
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
		return querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryArray,
			EngineName:  lowered,
			ElementType: new(normaliseTypeName(baseName, hook, modifiers...)),
		}
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
func floatPromotionRank(engineName string) int {
	switch engineName {
	case "float4":
		return 1
	default:
		return 2
	}
}
