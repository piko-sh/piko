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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// DuckDBFunctionResolver implements FunctionResolverPort for polymorphic DuckDB functions
// whose return types depend on their argument types.
type DuckDBFunctionResolver struct{}

// NewDuckDBFunctionResolver creates a DuckDB function resolver.
//
// Returns *DuckDBFunctionResolver which is the new resolver instance.
func NewDuckDBFunctionResolver() *DuckDBFunctionResolver {
	return &DuckDBFunctionResolver{}
}

// ResolveFunctionCall resolves a polymorphic DuckDB function call that the standard
// overload resolution could not match. It inspects the argument types to compute the
// correct return type for list, JSON, aggregate, conditional, and type-introspection
// functions.
//
// Takes name (string) which is the function name being called.
// Takes argumentTypes ([]querier_dto.SQLType) which describes call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which is the resolved overload, or nil when the
// function is not polymorphic so the caller falls back to the standard catalogue lookup.
// Returns error when resolution fails.
func (*DuckDBFunctionResolver) ResolveFunctionCall(
	_ *querier_dto.Catalogue,
	name string,
	_ string,
	argumentTypes []querier_dto.SQLType,
) (*querier_dto.FunctionResolution, error) {
	switch strings.ToLower(name) {
	case "array_agg", "list":
		return resolveArrayAgg(argumentTypes)
	case "unnest":
		return resolveUnnest(argumentTypes)
	case "array_append", "array_cat", "array_remove", "array_replace":
		return resolveArrayPassthrough(argumentTypes, 0)
	case "array_prepend":
		return resolveArrayPassthrough(argumentTypes, 1)
	case "min", "max":
		return resolveIdentityAggregate(argumentTypes)
	case "sum":
		return resolveSum(argumentTypes)
	case "avg":
		return resolveAvg(argumentTypes)
	case "coalesce":
		return resolveCoalesce(argumentTypes)
	case "typeof":
		return resolveTypeof()
	case "struct_pack":
		return resolveStructPack()
	case "struct_extract":
		return resolveStructExtract(argumentTypes)
	case "struct_insert":
		return resolveStructInsert(argumentTypes)
	case "map":
		return resolveMapConstruct()
	case "map_keys":
		return resolveMapKeys(argumentTypes)
	case "map_values":
		return resolveMapValues(argumentTypes)
	case "map_entries":
		return resolveMapEntries()
	case "element_at":
		return resolveElementAt(argumentTypes)
	case "list_transform", "list_filter", "list_reduce":
		return resolveListHigherOrder(argumentTypes)
	default:
		return nil, nil
	}
}

// resolveArrayAgg resolves array_agg and list aggregate return types.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which wraps the element type in an array, or
// nil when the call has no arguments.
// Returns error which is always nil.
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
		DataAccess:        querier_dto.DataAccessReadOnly,
		IsAggregate:       true,
	}, nil
}

// resolveUnnest resolves the unnest set-returning function.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which describes the set-returning element or
// record type, or nil when the call has no arguments.
// Returns error which is always nil.
func resolveUnnest(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	if len(argumentTypes) > 1 {
		return &querier_dto.FunctionResolution{
			ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: "record"},
			NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
			DataAccess:        querier_dto.DataAccessReadOnly,
			ReturnsSet:        true,
		}, nil
	}

	arrayType := argumentTypes[0]
	if arrayType.Category == querier_dto.TypeCategoryArray && arrayType.ElementType != nil {
		return &querier_dto.FunctionResolution{
			ReturnType:        *arrayType.ElementType,
			NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
			DataAccess:        querier_dto.DataAccessReadOnly,
			ReturnsSet:        true,
		}, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
		ReturnsSet:        true,
	}, nil
}

// resolveArrayPassthrough returns the array argument's type unchanged for array_append,
// array_cat, array_remove, array_replace, and array_prepend.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
// Takes arrayArgumentIndex (int) which is the position of the array argument to pass
// through.
//
// Returns *querier_dto.FunctionResolution which mirrors the array argument type, or nil
// when too few arguments are provided.
// Returns error which is always nil.
func resolveArrayPassthrough(argumentTypes []querier_dto.SQLType, arrayArgumentIndex int) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) <= arrayArgumentIndex {
		return nil, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        argumentTypes[arrayArgumentIndex],
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveIdentityAggregate returns the first argument's type for aggregates such as min
// and max where the result type matches the input type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which mirrors the first argument type, or nil
// when the call has no arguments.
// Returns error which is always nil.
func resolveIdentityAggregate(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        argumentTypes[0],
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
		IsAggregate:       true,
	}, nil
}

// resolveSum resolves the sum aggregate return type. DuckDB promotes every integer input
// (regardless of width, including BIGINT/HUGEINT) to HUGEINT; floating-point input stays
// double precision (float8); and DECIMAL input keeps the decimal category.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which describes the promoted return type, or
// nil when the call has no arguments.
// Returns error which is always nil.
func resolveSum(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	argumentType := argumentTypes[0]
	var returnType querier_dto.SQLType

	switch argumentType.Category {
	case querier_dto.TypeCategoryInteger:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "hugeint"}
	case querier_dto.TypeCategoryFloat:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"}
	default:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"}
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
		IsAggregate:       true,
	}, nil
}

// resolveAvg resolves the avg aggregate return type. DuckDB returns double precision
// (float8) for every numeric input, including integer, floating-point, and decimal, so
// avg over any numeric column resolves to float8.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which describes the return type, or nil when
// the call has no arguments.
// Returns error which is always nil.
func resolveAvg(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	argumentType := argumentTypes[0]
	var returnType querier_dto.SQLType

	switch argumentType.Category {
	case querier_dto.TypeCategoryInteger, querier_dto.TypeCategoryFloat, querier_dto.TypeCategoryDecimal:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"}
	default:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"}
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
		IsAggregate:       true,
	}, nil
}

// resolveCoalesce returns the type of the first argument with a known category, falling
// back to unknown when all arguments are unknown.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which describes the chosen return type.
// Returns error which is always nil.
func resolveCoalesce(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	for index := range argumentTypes {
		if argumentTypes[index].Category != querier_dto.TypeCategoryUnknown {
			return &querier_dto.FunctionResolution{
				ReturnType:        argumentTypes[index],
				NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
				DataAccess:        querier_dto.DataAccessReadOnly,
			}, nil
		}
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveTypeof resolves the typeof introspection function.
//
// Returns *querier_dto.FunctionResolution which describes the varchar return type.
// Returns error which is always nil.
func resolveTypeof() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveStructPack resolves the struct_pack constructor.
//
// Returns *querier_dto.FunctionResolution which describes the struct return type.
// Returns error which is always nil.
func resolveStructPack() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryStruct, EngineName: "struct"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveStructExtract resolves the struct_extract field accessor.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which describes the unknown field return type.
// Returns error which is always nil.
func resolveStructExtract(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) >= 1 && argumentTypes[0].Category == querier_dto.TypeCategoryStruct {
		return &querier_dto.FunctionResolution{
			ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
			NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
			DataAccess:        querier_dto.DataAccessReadOnly,
		}, nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveStructInsert resolves the struct_insert mutation.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which mirrors the input struct type, or returns
// a generic struct when no arguments are supplied.
// Returns error which is always nil.
func resolveStructInsert(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) >= 1 {
		return &querier_dto.FunctionResolution{
			ReturnType:        argumentTypes[0],
			NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
			DataAccess:        querier_dto.DataAccessReadOnly,
		}, nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryStruct, EngineName: "struct"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveMapConstruct resolves the map() constructor.
//
// Returns *querier_dto.FunctionResolution which describes the map return type.
// Returns error which is always nil.
func resolveMapConstruct() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryMap, EngineName: "map"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveMapKeys resolves map_keys to an array of the map's key type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which describes the array return type.
// Returns error which is always nil.
func resolveMapKeys(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) >= 1 {
		return resolveMapComponent(argumentTypes[0].KeyType)
	}
	return resolveMapComponent(nil)
}

// resolveMapValues resolves map_values to an array of the map's value type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which describes the array return type.
// Returns error which is always nil.
func resolveMapValues(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) >= 1 {
		return resolveMapComponent(argumentTypes[0].ElementType)
	}
	return resolveMapComponent(nil)
}

// resolveMapComponent wraps a map's key or value type in an array resolution, falling
// back to a generic list when no type is known.
//
// Takes extractedType (*querier_dto.SQLType) which is the key or value type pulled from
// the source map.
//
// Returns *querier_dto.FunctionResolution which describes the array return type.
// Returns error which is always nil.
func resolveMapComponent(extractedType *querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if extractedType != nil {
		componentType := *extractedType
		return &querier_dto.FunctionResolution{
			ReturnType: querier_dto.SQLType{
				Category:    querier_dto.TypeCategoryArray,
				EngineName:  componentType.EngineName + arraySubscriptSuffix,
				ElementType: &componentType,
			},
			NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
			DataAccess:        querier_dto.DataAccessReadOnly,
		}, nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryArray, EngineName: fallbackListEngineName},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveMapEntries resolves map_entries to a generic list of entries.
//
// Returns *querier_dto.FunctionResolution which describes the array return type.
// Returns error which is always nil.
func resolveMapEntries() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryArray, EngineName: fallbackListEngineName},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveElementAt resolves element_at on a map or array container.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which describes the element return type, or an
// unknown type when no container is matched.
// Returns error which is always nil.
func resolveElementAt(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) >= 1 {
		containerType := argumentTypes[0]
		if containerType.Category == querier_dto.TypeCategoryMap && containerType.ElementType != nil {
			return &querier_dto.FunctionResolution{
				ReturnType:        *containerType.ElementType,
				NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				DataAccess:        querier_dto.DataAccessReadOnly,
			}, nil
		}
		if containerType.Category == querier_dto.TypeCategoryArray && containerType.ElementType != nil {
			return &querier_dto.FunctionResolution{
				ReturnType:        *containerType.ElementType,
				NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
				DataAccess:        querier_dto.DataAccessReadOnly,
			}, nil
		}
	}
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveListHigherOrder resolves list_transform, list_filter, and list_reduce to the
// input list's type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which describes the call argument types in
// declared order.
//
// Returns *querier_dto.FunctionResolution which mirrors the input list type, falling back
// to a generic list when no input is supplied.
// Returns error which is always nil.
func resolveListHigherOrder(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) >= 1 {
		return &querier_dto.FunctionResolution{
			ReturnType:        argumentTypes[0],
			NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
			DataAccess:        querier_dto.DataAccessReadOnly,
		}, nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryArray, EngineName: fallbackListEngineName},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}
