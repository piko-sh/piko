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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// ResolveFunctionCall resolves a polymorphic SQLite
// function call whose return type depends on its
// argument types. This handles aggregate functions like
// SUM, MIN, and MAX where the builtin catalogue declares
// the return type as "any" but the actual return type
// should propagate from the argument.
//
// Takes catalogue (*querier_dto.Catalogue) which holds
// the schema context (unused for SQLite resolution).
// Takes name (string) which holds the function name to
// resolve.
// Takes schema (string) which holds the schema qualifier
// (unused for SQLite resolution).
// Takes argumentTypes ([]querier_dto.SQLType) which
// holds the resolved types of the call-site arguments.
//
// Returns *querier_dto.FunctionResolution which holds
// the resolved return type and metadata, or nil if the
// function does not need polymorphic resolution.
// Returns error which is always nil for this
// implementation.
func (*SQLiteEngine) ResolveFunctionCall(
	_ *querier_dto.Catalogue,
	name string,
	_ string,
	argumentTypes []querier_dto.SQLType,
) (*querier_dto.FunctionResolution, error) {
	switch strings.ToLower(name) {
	case "min", "max":
		return resolveSQLiteIdentityAggregate(argumentTypes)
	case "sum":
		return resolveSQLiteSum(argumentTypes)
	case "coalesce":
		return resolveSQLiteCoalesce(argumentTypes)
	default:
		return nil, nil
	}
}

// resolveSQLiteIdentityAggregate resolves MIN and MAX
// which return the same type as their argument.
//
// Takes argumentTypes ([]querier_dto.SQLType) which
// holds the resolved types of the call-site arguments.
//
// Returns *querier_dto.FunctionResolution which holds
// the argument type as the return type, or nil if no
// arguments were provided.
// Returns error which is always nil.
func resolveSQLiteIdentityAggregate(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        argumentTypes[0],
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsAggregate:       true,
	}, nil
}

// resolveSQLiteSum resolves SUM which in SQLite returns
// integer for integer arguments and real for everything
// else.
//
// Takes argumentTypes ([]querier_dto.SQLType) which
// holds the resolved types of the call-site arguments.
//
// Returns *querier_dto.FunctionResolution which holds
// the promoted return type, or nil if no arguments were
// provided.
// Returns error which is always nil.
func resolveSQLiteSum(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	argumentType := argumentTypes[0]
	var returnType querier_dto.SQLType

	switch argumentType.Category {
	case querier_dto.TypeCategoryInteger:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "integer"}
	default:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "real"}
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsAggregate:       true,
	}, nil
}

// resolveSQLiteCoalesce resolves COALESCE which returns
// the type of the first non-unknown argument.
//
// Takes argumentTypes ([]querier_dto.SQLType) which
// holds the resolved types of the call-site arguments.
//
// Returns *querier_dto.FunctionResolution which holds
// the type of the first argument with a known category,
// or unknown if all arguments have unknown types.
// Returns error which is always nil.
func resolveSQLiteCoalesce(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	for index := range argumentTypes {
		if argumentTypes[index].Category != querier_dto.TypeCategoryUnknown {
			return &querier_dto.FunctionResolution{
				ReturnType:        argumentTypes[index],
				NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
			}, nil
		}
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	}, nil
}
