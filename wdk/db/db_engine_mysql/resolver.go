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

package db_engine_mysql

import (
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	minArgumentsIf = 3

	minArgumentsIfNull = 2

	minArgumentsSingleArgFunction = 1

	promotionRankText = 4

	promotionRankDecimal = 3

	promotionRankFloat = 2

	promotionRankInteger = 1

	promotionRankDefault = 0
)

// MySQLFunctionResolver implements FunctionResolverPort for polymorphic MySQL
// functions whose return types depend on their argument types.
type MySQLFunctionResolver struct{}

// NewMySQLFunctionResolver creates a new MySQL function resolver.
func NewMySQLFunctionResolver() *MySQLFunctionResolver {
	return &MySQLFunctionResolver{}
}

// ResolveFunctionCall resolves a polymorphic MySQL function call that the
// standard overload resolution could not match. It inspects the argument types
// to compute the correct return type for conditional, aggregate, JSON, and
// string functions.
//
// Returns nil, nil for non-polymorphic functions so the caller falls back to
// the standard catalogue lookup.
func (*MySQLFunctionResolver) ResolveFunctionCall(
	_ *querier_dto.Catalogue,
	name string,
	_ string,
	argumentTypes []querier_dto.SQLType,
) (*querier_dto.FunctionResolution, error) {
	switch strings.ToLower(name) {
	case "if":
		return resolveIf(argumentTypes)
	case "ifnull":
		return resolveIfNull(argumentTypes)
	case "coalesce", "greatest", "least":
		return resolveCoalesce(argumentTypes)
	case "group_concat":
		return resolveGroupConcat()
	case "json_extract":
		return resolveJSONExtract()
	case "json_unquote", "concat", "concat_ws":
		return resolveTextReturn()
	case "sum":
		return resolveSum(argumentTypes)
	case "avg":
		return resolveAvg()
	case "min", "max":
		return resolveIdentityAggregate(argumentTypes)
	case "count":
		return resolveCount()
	default:
		return nil, nil
	}
}

func resolveIf(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < minArgumentsIf {
		return nil, nil
	}

	returnType := promoteTypes(argumentTypes[1], argumentTypes[2])

	return &querier_dto.FunctionResolution{
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	}, nil
}

func resolveIfNull(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < minArgumentsIfNull {
		return nil, nil
	}

	returnType := promoteTypes(argumentTypes[0], argumentTypes[1])

	return &querier_dto.FunctionResolution{
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	}, nil
}

func resolveCoalesce(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	var result querier_dto.SQLType
	initialised := false

	for index := range argumentTypes {
		if argumentTypes[index].Category == querier_dto.TypeCategoryUnknown {
			continue
		}

		if !initialised {
			result = argumentTypes[index]
			initialised = true

			continue
		}

		result = promoteTypes(result, argumentTypes[index])
	}

	if !initialised {
		result = querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""}
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        result,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	}, nil
}

func resolveGroupConcat() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsAggregate:       true,
	}, nil
}

func resolveJSONExtract() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

func resolveTextReturn() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	}, nil
}

func resolveSum(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < minArgumentsSingleArgFunction {
		return nil, nil
	}

	argumentType := argumentTypes[0]
	var returnType querier_dto.SQLType

	switch argumentType.Category {
	case querier_dto.TypeCategoryFloat:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "double"}
	default:
		returnType = querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "decimal"}
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsAggregate:       true,
	}, nil
}

func resolveAvg() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "double"},
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsAggregate:       true,
	}, nil
}

func resolveIdentityAggregate(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < minArgumentsSingleArgFunction {
		return nil, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        argumentTypes[0],
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		IsAggregate:       true,
	}, nil
}

func resolveCount() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		IsAggregate:       true,
	}, nil
}

// promoteTypes returns the wider of two SQL types following MySQL's implicit
// type promotion rules. When both types share the same category, the first is
// returned. Otherwise, the category hierarchy (text > decimal > float >
// integer) determines the result.
func promoteTypes(left querier_dto.SQLType, right querier_dto.SQLType) querier_dto.SQLType {
	if left.Category == querier_dto.TypeCategoryUnknown {
		return right
	}

	if right.Category == querier_dto.TypeCategoryUnknown {
		return left
	}

	if left.Category == right.Category {
		return left
	}

	leftRank := typePromotionRank(left.Category)
	rightRank := typePromotionRank(right.Category)

	if leftRank >= rightRank {
		return left
	}

	return right
}

func typePromotionRank(category querier_dto.SQLTypeCategory) int {
	switch category {
	case querier_dto.TypeCategoryText:
		return promotionRankText
	case querier_dto.TypeCategoryDecimal:
		return promotionRankDecimal
	case querier_dto.TypeCategoryFloat:
		return promotionRankFloat
	case querier_dto.TypeCategoryInteger:
		return promotionRankInteger
	default:
		return promotionRankDefault
	}
}
