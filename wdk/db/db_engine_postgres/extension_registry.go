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
	"piko.sh/piko/internal/querier/querier_dto"
)

var (
	// typeUUID is the SQL type descriptor for the uuid type.
	typeUUID = querier_dto.SQLType{Category: querier_dto.TypeCategoryUUID, EngineName: "uuid"}

	// typeText is the SQL type descriptor for the text type.
	typeText = querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}

	// typeBytea is the SQL type descriptor for the bytea type.
	typeBytea = querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "bytea"}

	// typeFloat8 is the SQL type descriptor for the float8 type.
	typeFloat8 = querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"}

	// typeJSON is the SQL type descriptor for the json type.
	typeJSON = querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"}

	// typeInteger is the SQL type descriptor for the int4 type.
	typeInteger = querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}
)

var (
	// extensionRegistry maps each supported Postgres extension to the function signatures it
	// adds when installed.
	extensionRegistry = map[string][]*querier_dto.FunctionSignature{
		"pgcrypto": {
			{Name: "gen_random_uuid", ReturnType: typeUUID, NullableBehaviour: querier_dto.FunctionNullableNeverNull},
			{Name: "crypt", ReturnType: typeText, NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments: []querier_dto.FunctionArgument{{Type: typeText}, {Type: typeText}}},
			{Name: "digest", ReturnType: typeBytea, NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments: []querier_dto.FunctionArgument{{Type: typeText}, {Type: typeText}}},
			{Name: "gen_salt", ReturnType: typeText, NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments: []querier_dto.FunctionArgument{{Type: typeText}}, MinArguments: 1, IsVariadic: false},
			{Name: "hmac", ReturnType: typeBytea, NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments: []querier_dto.FunctionArgument{{Type: typeText}, {Type: typeText}, {Type: typeText}}},
		},
		"uuid-ossp": {
			{Name: "uuid_generate_v1", ReturnType: typeUUID, NullableBehaviour: querier_dto.FunctionNullableNeverNull},
			{Name: "uuid_generate_v1mc", ReturnType: typeUUID, NullableBehaviour: querier_dto.FunctionNullableNeverNull},
			{Name: "uuid_generate_v3", ReturnType: typeUUID, NullableBehaviour: querier_dto.FunctionNullableNeverNull,
				Arguments: []querier_dto.FunctionArgument{{Type: typeUUID}, {Type: typeText}}},
			{Name: "uuid_generate_v4", ReturnType: typeUUID, NullableBehaviour: querier_dto.FunctionNullableNeverNull},
			{Name: "uuid_generate_v5", ReturnType: typeUUID, NullableBehaviour: querier_dto.FunctionNullableNeverNull,
				Arguments: []querier_dto.FunctionArgument{{Type: typeUUID}, {Type: typeText}}},
			{Name: "uuid_nil", ReturnType: typeUUID, NullableBehaviour: querier_dto.FunctionNullableNeverNull},
		},
		"pg_trgm": {
			{Name: "similarity", ReturnType: typeFloat8, NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments: []querier_dto.FunctionArgument{{Type: typeText}, {Type: typeText}}},
			{Name: "word_similarity", ReturnType: typeFloat8, NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments: []querier_dto.FunctionArgument{{Type: typeText}, {Type: typeText}}},
			{Name: "strict_word_similarity", ReturnType: typeFloat8, NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments: []querier_dto.FunctionArgument{{Type: typeText}, {Type: typeText}}},
			{Name: "show_trgm", ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text[]"},
				NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments:         []querier_dto.FunctionArgument{{Type: typeText}}},
		},
		"hstore": {
			{Name: "akeys", ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text[]"},
				NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments:         []querier_dto.FunctionArgument{{Type: typeText}}},
			{Name: "avals", ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text[]"},
				NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments:         []querier_dto.FunctionArgument{{Type: typeText}}},
			{Name: "hstore_to_json", ReturnType: typeJSON, NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments: []querier_dto.FunctionArgument{{Type: typeText}}},
		},
		"ltree": {
			{Name: "nlevel", ReturnType: typeInteger, NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments: []querier_dto.FunctionArgument{{Type: typeText}}},
			{Name: "lca", ReturnType: typeText, NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				Arguments: []querier_dto.FunctionArgument{{Type: typeText}, {Type: typeText}}, IsVariadic: true, MinArguments: 1},
		},
	}
)

// lookupExtensionFunctions returns the function signatures registered for an extension by
// name.
//
// Takes name (string) which is the extension name.
//
// Returns []*querier_dto.FunctionSignature which is the list of function signatures, or
// nil when no extension matches.
func lookupExtensionFunctions(name string) []*querier_dto.FunctionSignature {
	return extensionRegistry[name]
}
