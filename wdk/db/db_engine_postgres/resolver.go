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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// PostgresFunctionResolver implements FunctionResolverPort for polymorphic PostgreSQL
// functions whose return types depend on their argument types.
type PostgresFunctionResolver struct{}

// NewPostgresFunctionResolver creates a new PostgreSQL function resolver.
//
// Returns *PostgresFunctionResolver which is the new resolver.
func NewPostgresFunctionResolver() *PostgresFunctionResolver {
	return &PostgresFunctionResolver{}
}

// ResolveFunctionCall resolves a polymorphic PostgreSQL function call.
//
// Inspects argument types to compute the correct return type for array, JSON, aggregate,
// conditional, and type-introspection functions when the standard overload resolution
// could not match. Returns nil, nil for non-polymorphic functions so the caller falls
// back to the standard catalogue lookup.
//
// Takes name (string) which is the function name.
// Takes argumentTypes ([]querier_dto.SQLType) which is the list of argument types.
//
// Returns *querier_dto.FunctionResolution which is the resolved signature, or nil when no
// polymorphic rule applies.
// Returns error when resolution fails.
func (*PostgresFunctionResolver) ResolveFunctionCall(
	_ *querier_dto.Catalogue,
	name string,
	_ string,
	argumentTypes []querier_dto.SQLType,
) (*querier_dto.FunctionResolution, error) {
	switch strings.ToLower(name) {
	case "array_agg":
		return resolveArrayAgg(argumentTypes)
	case "unnest":
		return resolveUnnest(argumentTypes)
	case "array_append", "array_cat", "array_remove", "array_replace":
		return resolveArrayPassthrough(argumentTypes, 0)
	case "array_prepend":
		return resolveArrayPassthrough(argumentTypes, 1)
	case "jsonb_populate_record", "json_populate_record":
		return resolvePopulateRecord(argumentTypes, false)
	case "jsonb_populate_recordset", "json_populate_recordset":
		return resolvePopulateRecord(argumentTypes, true)
	case "jsonb_to_record", "json_to_record":
		return resolveToRecord(false)
	case "jsonb_to_recordset", "json_to_recordset":
		return resolveToRecord(true)
	case "min", "max":
		return resolveIdentityAggregate(argumentTypes)
	case "sum":
		return resolveSum(argumentTypes)
	case "avg":
		return resolveAvg(argumentTypes)
	case "coalesce":
		return resolveCoalesce(argumentTypes)
	case "pg_typeof":
		return resolvePgTypeof()
	default:
		return nil, nil
	}
}

// resolveArrayAgg returns the array_agg signature, producing an array whose element type
// matches the first argument's type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which is the call's argument types.
//
// Returns *querier_dto.FunctionResolution which is the resolved signature, or nil when no
// argument was provided.
// Returns error when resolution fails.
func resolveArrayAgg(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	elementType := argumentTypes[0]

	return &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryArray,
			EngineName:  elementType.EngineName + arraySubscriptSuffix,
			ElementType: &elementType,
		},
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsAggregate:       true,
	}, nil
}

// resolveUnnest returns the unnest signature whose return type is the element type of the
// first argument's array type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which is the call's argument types.
//
// Returns *querier_dto.FunctionResolution which is the resolved signature, or nil when no
// argument was provided.
// Returns error when resolution fails.
func resolveUnnest(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	if len(argumentTypes) > 1 {
		return &querier_dto.FunctionResolution{
			ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: "record"},
			NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
			ReturnsSet:        true,
		}, nil
	}

	arrayType := argumentTypes[0]
	if arrayType.Category == querier_dto.TypeCategoryArray && arrayType.ElementType != nil {
		return &querier_dto.FunctionResolution{
			ReturnType:        *arrayType.ElementType,
			NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
			ReturnsSet:        true,
		}, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		ReturnsSet:        true,
	}, nil
}

// resolveArrayPassthrough returns the signature for array functions whose return type
// matches the array argument at arrayArgumentIndex.
//
// Takes argumentTypes ([]querier_dto.SQLType) which is the call's argument types.
// Takes arrayArgumentIndex (int) which is the index of the array argument.
//
// Returns *querier_dto.FunctionResolution which is the resolved signature, or nil when
// not enough arguments were provided.
// Returns error when resolution fails.
func resolveArrayPassthrough(argumentTypes []querier_dto.SQLType, arrayArgumentIndex int) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) <= arrayArgumentIndex {
		return nil, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        argumentTypes[arrayArgumentIndex],
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

// resolvePopulateRecord returns the signature for json_populate_record and its
// set-returning variant, mapping the return type to the first argument's row type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which is the call's argument types.
// Takes returnsSet (bool) which signals the set-returning form.
//
// Returns *querier_dto.FunctionResolution which is the resolved signature, or nil when no
// argument was provided.
// Returns error when resolution fails.
func resolvePopulateRecord(argumentTypes []querier_dto.SQLType, returnsSet bool) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        argumentTypes[0],
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		ReturnsSet:        returnsSet,
	}, nil
}

// resolveToRecord returns the signature for json_to_record and json_to_recordset, both of
// which return a generic record type.
//
// Takes returnsSet (bool) which signals the set-returning form.
//
// Returns *querier_dto.FunctionResolution which is the resolved signature.
// Returns error when resolution fails.
func resolveToRecord(returnsSet bool) (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: "record"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		ReturnsSet:        returnsSet,
	}, nil
}

// resolveIdentityAggregate returns the signature for MIN and MAX whose return type
// matches the first argument's type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which is the call's argument types.
//
// Returns *querier_dto.FunctionResolution which is the resolved signature, or nil when no
// argument was provided.
// Returns error when resolution fails.
func resolveIdentityAggregate(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        argumentTypes[0],
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsAggregate:       true,
	}, nil
}

// resolveSum returns the SUM signature, widening small integers to int8 and other
// numerics to numeric or float8 per Postgres rules.
//
// Takes argumentTypes ([]querier_dto.SQLType) which is the call's argument types.
//
// Returns *querier_dto.FunctionResolution which is the resolved signature, or nil when no
// argument was provided.
// Returns error when resolution fails.
func resolveSum(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	argumentType := argumentTypes[0]
	var returnType querier_dto.SQLType

	switch argumentType.Category {
	case querier_dto.TypeCategoryInteger:
		if isSmallInteger(argumentType.EngineName) {
			returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"}
		} else {
			returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"}
		}
	case querier_dto.TypeCategoryFloat:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"}
	default:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"}
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsAggregate:       true,
	}, nil
}

// resolveAvg returns the AVG signature, producing float8 for floating inputs and numeric
// otherwise.
//
// Takes argumentTypes ([]querier_dto.SQLType) which is the call's argument types.
//
// Returns *querier_dto.FunctionResolution which is the resolved signature, or nil when no
// argument was provided.
// Returns error when resolution fails.
func resolveAvg(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	argumentType := argumentTypes[0]
	var returnType querier_dto.SQLType

	switch argumentType.Category {
	case querier_dto.TypeCategoryFloat:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"}
	default:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"}
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsAggregate:       true,
	}, nil
}

// resolveCoalesce returns the COALESCE signature, using the first known argument type as
// the result type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which is the call's argument types.
//
// Returns *querier_dto.FunctionResolution which is the resolved signature.
// Returns error when resolution fails.
func resolveCoalesce(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
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

// resolvePgTypeof returns the pg_typeof signature, which always reports text.
//
// Returns *querier_dto.FunctionResolution which is the resolved signature.
// Returns error when resolution fails.
func resolvePgTypeof() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

// isSmallInteger reports whether engineName names a small Postgres integer (int2 or int4)
// that SUM widens to int8.
//
// Takes engineName (string) which is the Postgres type's engine name.
//
// Returns bool which is true for int2 or int4.
func isSmallInteger(engineName string) bool {
	switch engineName {
	case "int2", "int4":
		return true
	default:
		return false
	}
}
