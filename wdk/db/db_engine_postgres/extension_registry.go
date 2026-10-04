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
	typeUUID = querier_dto.NewSQLType(querier_dto.TypeCategoryUUID, "uuid")

	// typeText is the SQL type descriptor for the text type.
	typeText = querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text")

	// typeBytea is the SQL type descriptor for the bytea type.
	typeBytea = querier_dto.NewSQLType(querier_dto.TypeCategoryBytea, "bytea")

	// typeFloat8 is the SQL type descriptor for the float8 type.
	typeFloat8 = querier_dto.NewSQLType(querier_dto.TypeCategoryFloat, "float8")

	// typeJSON is the SQL type descriptor for the json type.
	typeJSON = querier_dto.NewSQLType(querier_dto.TypeCategoryJSON, "json")

	// typeInteger is the SQL type descriptor for the int4 type.
	typeInteger = querier_dto.NewSQLType(querier_dto.TypeCategoryInteger, "int4")

	// typeTextList is the SQL type descriptor the hstore and pg_trgm key and trigram
	// functions return, a text-category type spelt text[].
	typeTextList = querier_dto.NewSQLType(querier_dto.TypeCategoryText, "text[]")
)

var (
	// extensionRegistry maps each supported Postgres extension to the function signatures it
	// adds when installed.
	extensionRegistry = map[string][]*querier_dto.FunctionSignature{
		"pgcrypto": {
			extensionFunction("gen_random_uuid", nil, typeUUID, querier_dto.FunctionNullableNeverNull),
			extensionFunction("crypt", unnamedArguments(typeText, typeText), typeText, querier_dto.FunctionNullableReturnsNullOnNull),
			extensionFunction("digest", unnamedArguments(typeText, typeText), typeBytea, querier_dto.FunctionNullableReturnsNullOnNull),
			extensionFunction("gen_salt", unnamedArguments(typeText), typeText, querier_dto.FunctionNullableReturnsNullOnNull,
				querier_dto.WithMinArguments(1)),
			extensionFunction("hmac", unnamedArguments(typeText, typeText, typeText), typeBytea, querier_dto.FunctionNullableReturnsNullOnNull),
		},
		"uuid-ossp": {
			extensionFunction("uuid_generate_v1", nil, typeUUID, querier_dto.FunctionNullableNeverNull),
			extensionFunction("uuid_generate_v1mc", nil, typeUUID, querier_dto.FunctionNullableNeverNull),
			extensionFunction("uuid_generate_v3", unnamedArguments(typeUUID, typeText), typeUUID, querier_dto.FunctionNullableNeverNull),
			extensionFunction("uuid_generate_v4", nil, typeUUID, querier_dto.FunctionNullableNeverNull),
			extensionFunction("uuid_generate_v5", unnamedArguments(typeUUID, typeText), typeUUID, querier_dto.FunctionNullableNeverNull),
			extensionFunction("uuid_nil", nil, typeUUID, querier_dto.FunctionNullableNeverNull),
		},
		"pg_trgm": {
			extensionFunction("similarity", unnamedArguments(typeText, typeText), typeFloat8, querier_dto.FunctionNullableReturnsNullOnNull),
			extensionFunction("word_similarity", unnamedArguments(typeText, typeText), typeFloat8, querier_dto.FunctionNullableReturnsNullOnNull),
			extensionFunction("strict_word_similarity", unnamedArguments(typeText, typeText), typeFloat8,
				querier_dto.FunctionNullableReturnsNullOnNull),
			extensionFunction("show_trgm", unnamedArguments(typeText), typeTextList, querier_dto.FunctionNullableReturnsNullOnNull),
		},
		"hstore": {
			extensionFunction("akeys", unnamedArguments(typeText), typeTextList, querier_dto.FunctionNullableReturnsNullOnNull),
			extensionFunction("avals", unnamedArguments(typeText), typeTextList, querier_dto.FunctionNullableReturnsNullOnNull),
			extensionFunction("hstore_to_json", unnamedArguments(typeText), typeJSON, querier_dto.FunctionNullableReturnsNullOnNull),
		},
		"ltree": {
			extensionFunction("nlevel", unnamedArguments(typeText), typeInteger, querier_dto.FunctionNullableReturnsNullOnNull),
			extensionFunction("lca", unnamedArguments(typeText, typeText), typeText, querier_dto.FunctionNullableReturnsNullOnNull,
				querier_dto.WithVariadic(1)),
		},
	}
)

// Every registered extension function is marked read-only. They are all scalars that read
// but never modify table data, so a pure SELECT using one (similarity(),
// gen_random_uuid(), nlevel(), ...) must classify as read-only and route to a read
// replica. Without this they default to DataAccessUnknown, which the call-graph
// propagation treats as data-modifying.

func init() {
	for _, signatures := range extensionRegistry {
		for _, signature := range signatures {
			signature.DataAccess = querier_dto.DataAccessReadOnly
		}
	}
}

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

// extensionFunction builds a named extension function signature.
//
// Takes name (string) which is the function's name.
// Takes arguments ([]querier_dto.FunctionArgument) which are its arguments, or nil.
// Takes returnType (querier_dto.SQLType) which is the type it returns.
// Takes nullableBehaviour (querier_dto.FunctionNullableBehaviour) which says when the
// result is NULL.
// Takes options (...querier_dto.FunctionSignatureOption) which set further attributes.
//
// Returns *querier_dto.FunctionSignature which is the configured signature.
func extensionFunction(
	name string,
	arguments []querier_dto.FunctionArgument,
	returnType querier_dto.SQLType,
	nullableBehaviour querier_dto.FunctionNullableBehaviour,
	options ...querier_dto.FunctionSignatureOption,
) *querier_dto.FunctionSignature {
	options = append([]querier_dto.FunctionSignatureOption{querier_dto.WithFunctionName(name)}, options...)
	return querier_dto.NewFunctionSignature(arguments, returnType, nullableBehaviour, options...)
}

// unnamedArguments builds required, unnamed arguments of the given types, the shape every
// registered extension function declares.
//
// Takes types (...querier_dto.SQLType) which are the argument types in order.
//
// Returns []querier_dto.FunctionArgument which holds one required argument per type.
func unnamedArguments(types ...querier_dto.SQLType) []querier_dto.FunctionArgument {
	arguments := make([]querier_dto.FunctionArgument, len(types))
	for index := range types {
		arguments[index] = querier_dto.FunctionArgument{
			Type:       types[index],
			Name:       "",
			IsOptional: false,
		}
	}
	return arguments
}
