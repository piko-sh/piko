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
	"piko.sh/piko/internal/querier/querier_dto"
)

// registerExtendedJSONFunctions covers the long tail of JSON path helpers,
// case-insensitive extractor variants, key/value pair extraction, JSON formatting
// helpers, and the Dynamic column element and type interrogation helpers.
//
// Registration delegates to topical helpers so each function stays within the linter
// budget.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedJSONFunctions(b *FunctionCatalogueBuilder) {
	registerJSONKeyValueExtractors(b)
	registerJSONPathInspectionFunctions(b)
	registerJSONCaseInsensitiveExtractors(b)
	registerJSONFormattingFunctions(b)
	registerDynamicTypeFunctions(b)
}

// registerJSONKeyValueExtractors covers the JSONArrayLength, JSONExtractKeysAndValues,
// and JSONExtractKeysAndValuesRaw helpers.
//
// Both arities (with and without path) are registered for the raw and typed variants.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerJSONKeyValueExtractors(b *FunctionCatalogueBuilder) {
	b.Register("JSONArrayLength", b.uint64Type, b.textType)
	keyValueTupleType := jsonKeyValueTupleType(b)
	b.Register("JSONExtractKeysAndValues", arrayOf(keyValueTupleType), b.textType, b.textType)
	b.Register("JSONExtractKeysAndValues", arrayOf(keyValueTupleType), b.textType, b.textType, b.textType)
	b.Register("JSONExtractKeysAndValuesRaw", arrayOf(keyValueTupleType), b.textType)
	b.Register("JSONExtractKeysAndValuesRaw", arrayOf(keyValueTupleType), b.textType, b.textType)
}

// registerJSONPathInspectionFunctions covers the path and type enumeration helpers
// exposed by the JSON column type.
//
// Each returns Array(String) listing dotted paths or path-with-type pairs.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerJSONPathInspectionFunctions(b *FunctionCatalogueBuilder) {
	for _, name := range []string{
		"JSONAllPaths", "JSONAllPathsWithTypes", "JSONAllValues",
		"JSONDynamicPaths", "JSONDynamicPathsWithTypes",
		"JSONSharedDataPaths", "JSONSharedDataPathsWithTypes",
	} {
		b.Register(name, arrayOf(b.textType), b.jsonType)
	}
}

// registerJSONCaseInsensitiveExtractors covers the case-insensitive variants of the
// JSONExtract* family.
//
// ClickHouse resolves them as distinct functions even though the return type matches the
// base form, so each spelling needs its own catalogue entry.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerJSONCaseInsensitiveExtractors(b *FunctionCatalogueBuilder) {
	b.Register("JSONExtractCaseInsensitive", b.unknownType, b.textType, b.textType, b.textType)
	b.Register("JSONExtractStringCaseInsensitive", b.textType, b.textType, b.textType)
	b.Register("JSONExtractIntCaseInsensitive", b.int64Type, b.textType, b.textType)
	b.Register("JSONExtractUIntCaseInsensitive", b.uint64Type, b.textType, b.textType)
	b.Register("JSONExtractFloatCaseInsensitive", b.float64Type, b.textType, b.textType)
	b.Register("JSONExtractBoolCaseInsensitive", b.boolType, b.textType, b.textType)
	b.Register("JSONExtractRawCaseInsensitive", b.textType, b.textType, b.textType)
	b.Register("JSONExtractArrayRawCaseInsensitive", arrayOf(b.textType), b.textType, b.textType)
	b.Register("JSONExtractKeysCaseInsensitive", arrayOf(b.textType), b.textType, b.textType)
	keyValueTupleType := jsonKeyValueTupleType(b)
	b.Register("JSONExtractKeysAndValuesCaseInsensitive", arrayOf(keyValueTupleType), b.textType, b.textType)
	b.Register("JSONExtractKeysAndValuesRawCaseInsensitive", arrayOf(keyValueTupleType), b.textType)
	b.Register("JSONExtractKeysAndValuesRawCaseInsensitive", arrayOf(keyValueTupleType), b.textType, b.textType)
}

// registerJSONFormattingFunctions covers helpers that serialise or re-format JSON values
// and the JSON merge-patch operator.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerJSONFormattingFunctions(b *FunctionCatalogueBuilder) {
	b.Register("prettyPrintJSON", b.textType, b.textType)
	b.Register("toJSONString", b.textType, b.unknownType)
	b.RegisterVariadic("JSONMergePatch", b.jsonType, 1, b.textType)
}

// registerDynamicTypeFunctions covers the helpers that interrogate a Dynamic column's
// runtime element type, extract a typed view, and report whether the value currently
// lives in the shared subcolumn.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerDynamicTypeFunctions(b *FunctionCatalogueBuilder) {
	b.Register("dynamicElement", b.unknownType, b.unknownType, b.textType)
	b.Register("dynamicType", b.textType, b.unknownType)
	b.Register("isDynamicElementInSharedData", b.boolType, b.unknownType)
}

// jsonKeyValueTupleType constructs the Tuple(String, String) shape returned by the JSON
// key/value extraction helpers.
//
// Used by both the typed and case-insensitive extractor sites, and reused from
// builtin_functions_tuple_map_extended.go via the tupleToNameValuePairs registration
// which produces the same key/value tuple shape.
//
// Takes b (*FunctionCatalogueBuilder) which supplies the shared text type descriptor.
//
// Returns querier_dto.SQLType which is the Tuple(String, String) key/value shape.
func jsonKeyValueTupleType(b *FunctionCatalogueBuilder) querier_dto.SQLType {
	return querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryStruct,
		EngineName: "Tuple",
		StructFields: []querier_dto.StructField{
			{Name: "key", SQLType: b.textType},
			{Name: "value", SQLType: b.textType},
		},
	}
}
