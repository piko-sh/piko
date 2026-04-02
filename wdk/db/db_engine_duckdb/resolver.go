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

// DuckDBFunctionResolver implements FunctionResolverPort for polymorphic
// DuckDB functions whose return types depend on their argument types.
type DuckDBFunctionResolver struct{}

// NewDuckDBFunctionResolver creates a new DuckDB function resolver.
func NewDuckDBFunctionResolver() *DuckDBFunctionResolver {
	return &DuckDBFunctionResolver{}
}

// ResolveFunctionCall resolves a polymorphic DuckDB function call that the
// standard overload resolution could not match. It inspects the argument types
// to compute the correct return type for list, JSON, aggregate, conditional,
// and type-introspection functions.
//
// Returns nil, nil for non-polymorphic functions so the caller falls back to
// the standard catalogue lookup.
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

func resolveArrayPassthrough(argumentTypes []querier_dto.SQLType, arrayArgumentIndex int) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) <= arrayArgumentIndex {
		return nil, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        argumentTypes[arrayArgumentIndex],
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

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

func resolveSum(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < 1 {
		return nil, nil
	}

	argumentType := argumentTypes[0]
	var returnType querier_dto.SQLType

	switch argumentType.Category {
	case querier_dto.TypeCategoryInteger:
		if isSmallInteger(argumentType.EngineName) {
			returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "hugeint"}
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

func resolveTypeof() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

func resolveStructPack() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryStruct, EngineName: "struct"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

func resolveStructExtract(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) >= 1 && argumentTypes[0].Category == querier_dto.TypeCategoryStruct {
		return &querier_dto.FunctionResolution{
			ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
			NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		}, nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

func resolveStructInsert(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) >= 1 {
		return &querier_dto.FunctionResolution{
			ReturnType:        argumentTypes[0],
			NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		}, nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryStruct, EngineName: "struct"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

func resolveMapConstruct() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryMap, EngineName: "map"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

func resolveMapKeys(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) >= 1 {
		return resolveMapComponent(argumentTypes[0].KeyType)
	}
	return resolveMapComponent(nil)
}

func resolveMapValues(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) >= 1 {
		return resolveMapComponent(argumentTypes[0].ElementType)
	}
	return resolveMapComponent(nil)
}

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
		}, nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryArray, EngineName: fallbackListEngineName},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

func resolveMapEntries() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryArray, EngineName: fallbackListEngineName},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

func resolveElementAt(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) >= 1 {
		containerType := argumentTypes[0]
		if containerType.Category == querier_dto.TypeCategoryMap && containerType.ElementType != nil {
			return &querier_dto.FunctionResolution{
				ReturnType:        *containerType.ElementType,
				NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
			}, nil
		}
		if containerType.Category == querier_dto.TypeCategoryArray && containerType.ElementType != nil {
			return &querier_dto.FunctionResolution{
				ReturnType:        *containerType.ElementType,
				NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
			}, nil
		}
	}
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

func resolveListHigherOrder(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) >= 1 {
		return &querier_dto.FunctionResolution{
			ReturnType:        argumentTypes[0],
			NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		}, nil
	}
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryArray, EngineName: fallbackListEngineName},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

func isSmallInteger(engineName string) bool {
	switch engineName {
	case "int1", "int2", "int4", "utinyint", "usmallint":
		return true
	default:
		return false
	}
}
