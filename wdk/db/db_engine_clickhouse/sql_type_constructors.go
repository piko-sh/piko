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

// arrayOf wraps the given element type in an Array(T) shape.
//
// Takes element (querier_dto.SQLType) which is the element type T to wrap.
//
// Returns querier_dto.SQLType which is the Array type carrying that element type.
func arrayOf(element querier_dto.SQLType) querier_dto.SQLType {
	return elementWrapperOf(querier_dto.TypeCategoryArray, engineNameArray, element)
}

// aggregateStateOf wraps the given value type in an aggregate-state shape such as
// AggregateFunction(name, T) or SimpleAggregateFunction(name, T).
//
// Takes engineName (string) which is the aggregate-state engine name.
// Takes element (querier_dto.SQLType) which is the value type T held by the state.
//
// Returns querier_dto.SQLType which is the aggregate-state type carrying that value type.
func aggregateStateOf(engineName string, element querier_dto.SQLType) querier_dto.SQLType {
	return elementWrapperOf(querier_dto.TypeCategoryAggregateState, engineName, element)
}

// elementWrapperOf builds a type of the given category and engine name that wraps a
// single element type.
//
// Takes category (querier_dto.SQLTypeCategory) which classifies the wrapping type.
// Takes engineName (string) which is the engine's name for the wrapping type.
// Takes element (querier_dto.SQLType) which is the wrapped type, copied so the result
// owns it.
//
// Returns querier_dto.SQLType which is the wrapping type pointing at its element.
func elementWrapperOf(
	category querier_dto.SQLTypeCategory,
	engineName string,
	element querier_dto.SQLType,
) querier_dto.SQLType {
	wrapper := querier_dto.NewSQLType(category, engineName)
	wrapper.ElementType = &element
	return wrapper
}

// mapOf builds a Map(K, V) type from its key and value types.
//
// Takes key (querier_dto.SQLType) which is the key type K.
// Takes value (querier_dto.SQLType) which is the value type V.
//
// Returns querier_dto.SQLType which is the Map type carrying both types.
func mapOf(key querier_dto.SQLType, value querier_dto.SQLType) querier_dto.SQLType {
	mapType := querier_dto.NewSQLType(querier_dto.TypeCategoryMap, engineNameMap)
	mapType.KeyType = &key
	mapType.ElementType = &value
	return mapType
}

// tupleOf builds a Tuple(...) type from its fields.
//
// Takes fields ([]querier_dto.StructField) which are the tuple's fields in order.
//
// Returns querier_dto.SQLType which is the Tuple type carrying those fields.
func tupleOf(fields []querier_dto.StructField) querier_dto.SQLType {
	tupleType := querier_dto.NewSQLType(querier_dto.TypeCategoryStruct, engineNameTuple)
	tupleType.StructFields = fields
	return tupleType
}

// fixedStringType constructs a FixedString(N) SQLType.
//
// Takes length (int) which is the fixed byte length N.
//
// Returns querier_dto.SQLType which is the FixedString type carrying that length.
func fixedStringType(length int) querier_dto.SQLType {
	fixedString := querier_dto.NewSQLType(querier_dto.TypeCategoryText, "FixedString")
	fixedString.Length = &length
	return fixedString
}
