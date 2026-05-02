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
	// minArgumentsIf is the minimum argument count for the IF function.
	minArgumentsIf = 3

	// minArgumentsIfNull is the minimum argument count for IFNULL.
	minArgumentsIfNull = 2

	// minArgumentsSingleArgFunction caps single-argument resolver entry.
	minArgumentsSingleArgFunction = 1

	// promotionRankText ranks text categories highest during promotion.
	promotionRankText = 4

	// promotionRankDecimal ranks decimal below text but above float.
	promotionRankDecimal = 3

	// promotionRankFloat ranks float above integer in promotion order.
	promotionRankFloat = 2

	// promotionRankInteger is the lowest numeric promotion rank.
	promotionRankInteger = 1

	// promotionRankDefault applies to unrecognised type categories.
	promotionRankDefault = 0
)

// MySQLFunctionResolver implements FunctionResolverPort for polymorphic MySQL functions
// whose return types depend on their argument types.
type MySQLFunctionResolver struct{}

// NewMySQLFunctionResolver creates a new MySQL function resolver.
//
// Returns *MySQLFunctionResolver which resolves polymorphic call sites.
func NewMySQLFunctionResolver() *MySQLFunctionResolver {
	return &MySQLFunctionResolver{}
}

// ResolveFunctionCall resolves a polymorphic MySQL function call.
//
// Inspects argument types to compute return types for conditional, aggregate, JSON, and
// string functions. Returns nil for non-polymorphic functions so the caller falls back to
// the standard catalogue lookup.
//
// Takes _ (*querier_dto.Catalogue) which is the catalogue context (unused).
// Takes name (string) which is the function name to resolve.
// Takes _ (string) which is the schema scope (unused).
// Takes argumentTypes ([]querier_dto.SQLType) which carries the actual argument types
// provided at the call site.
//
// Returns *querier_dto.FunctionResolution which describes the chosen signature, or nil
// when no polymorphic rule applies.
// Returns error when resolution fails.
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
	case "coalesce":
		return resolveCoalesce(argumentTypes)
	case "greatest", "least":
		return resolveGreatestLeast(argumentTypes)
	case "group_concat":
		return resolveGroupConcat()
	case "json_extract":
		return resolveJSONExtract()
	case "json_unquote", "concat", "concat_ws":
		return resolveTextReturn()
	case "floor", "ceil", "ceiling":
		return resolveFloorCeil(argumentTypes)
	case "mod":
		return resolveMod(argumentTypes)
	case "sum":
		return resolveSum(argumentTypes)
	case "avg":
		return resolveAvg()
	case "min", "max", "bit_and", "bit_or", "bit_xor":
		return resolveIdentityAggregate(argumentTypes)
	case "count":
		return resolveCount()
	case "nullif", "first_value", "last_value", "lag", "lead":
		return resolveIdentityFunction(argumentTypes)
	default:
		return nil, nil
	}
}

// resolveIf computes the return type for the IF conditional function.
//
// Takes argumentTypes ([]querier_dto.SQLType) which carries the actual argument types at
// the call site.
//
// Returns *querier_dto.FunctionResolution which describes the promoted branch type, or
// nil when too few arguments are present.
// Returns error when resolution fails.
func resolveIf(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < minArgumentsIf {
		return nil, nil
	}

	returnType := promoteTypes(argumentTypes[1], argumentTypes[2])

	return &querier_dto.FunctionResolution{
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveIfNull computes the return type for IFNULL.
//
// Takes argumentTypes ([]querier_dto.SQLType) which carries the actual argument types at
// the call site.
//
// Returns *querier_dto.FunctionResolution which describes the promoted type, or nil when
// too few arguments are present.
// Returns error when resolution fails.
func resolveIfNull(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < minArgumentsIfNull {
		return nil, nil
	}

	returnType := promoteTypes(argumentTypes[0], argumentTypes[1])

	return &querier_dto.FunctionResolution{
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// promoteArgumentTypes promotes all non-unknown argument types to a single common type.
//
// Takes argumentTypes ([]querier_dto.SQLType) which carries the actual argument types at
// the call site.
//
// Returns querier_dto.SQLType which is the promoted type, or an unknown type when every
// argument is unknown.
func promoteArgumentTypes(argumentTypes []querier_dto.SQLType) querier_dto.SQLType {
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
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""}
	}

	return result
}

// resolveCoalesce computes the return type for COALESCE.
//
// COALESCE returns its first non-NULL argument, so it is called even when arguments are
// NULL.
//
// Takes argumentTypes ([]querier_dto.SQLType) which carries the actual argument types at
// the call site.
//
// Returns *querier_dto.FunctionResolution which describes the promoted type across all
// non-unknown arguments.
// Returns error when resolution fails.
func resolveCoalesce(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        promoteArgumentTypes(argumentTypes),
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveGreatestLeast computes the return type for GREATEST/LEAST.
//
// Unlike COALESCE, GREATEST and LEAST return NULL whenever any argument is NULL, so the
// resolution is modelled as returns-null-on-null.
//
// Takes argumentTypes ([]querier_dto.SQLType) which carries the actual argument types at
// the call site.
//
// Returns *querier_dto.FunctionResolution which describes the promoted type across all
// non-unknown arguments.
// Returns error when resolution fails.
func resolveGreatestLeast(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        promoteArgumentTypes(argumentTypes),
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveGroupConcat computes the return type for GROUP_CONCAT.
//
// Returns *querier_dto.FunctionResolution which describes the text-typed aggregate
// result.
// Returns error when resolution fails.
func resolveGroupConcat() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
		IsAggregate:       true,
	}, nil
}

// resolveJSONExtract computes the return type for JSON_EXTRACT.
//
// Returns *querier_dto.FunctionResolution which describes the JSON-typed result.
// Returns error when resolution fails.
func resolveJSONExtract() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveTextReturn yields a text-typed resolution used by string builders.
//
// Returns *querier_dto.FunctionResolution which describes the text-typed result.
// Returns error when resolution fails.
func resolveTextReturn() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveFloorCeil computes the return type for FLOOR / CEIL / CEILING.
//
// The static catalogue declares these as INT, but MySQL returns a value with the same
// numeric scale as the argument for floating-point input: FLOOR(double) and CEIL(double)
// yield a DOUBLE, not an INT. Narrowing a double to int32 silently overflows for values
// beyond the 32-bit range, so when the argument is a float the result echoes the float
// type. Integer (and any other) arguments fall through to nil so the catalogue's INT
// signature still applies; the decimal case is out of scope (MySQL errors on an implicit
// Decimal-to-Int cast there).
//
// Takes argumentTypes ([]querier_dto.SQLType) which carries the actual argument types at
// the call site.
//
// Returns *querier_dto.FunctionResolution which echoes a float argument type, or nil to
// defer to the catalogue signature.
// Returns error when resolution fails.
func resolveFloorCeil(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < minArgumentsSingleArgFunction {
		return nil, nil
	}

	if argumentTypes[0].Category != querier_dto.TypeCategoryFloat {
		return nil, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        argumentTypes[0],
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveMod computes the return type for MOD.
//
// The static catalogue declares MOD as INT, but MySQL preserves the fractional part when
// either operand is floating-point: MOD(double, double) yields a DOUBLE, so narrowing to
// INT would drop the fraction. When either argument is a float the result promotes to the
// wider float type; otherwise the call defers to the catalogue's INT signature.
//
// Takes argumentTypes ([]querier_dto.SQLType) which carries the actual argument types at
// the call site.
//
// Returns *querier_dto.FunctionResolution which echoes the promoted float type, or nil to
// defer to the catalogue signature.
// Returns error when resolution fails.
func resolveMod(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < minArgumentsIfNull {
		return nil, nil
	}

	leftIsFloat := argumentTypes[0].Category == querier_dto.TypeCategoryFloat
	rightIsFloat := argumentTypes[1].Category == querier_dto.TypeCategoryFloat
	if !leftIsFloat && !rightIsFloat {
		return nil, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        widerFloat(argumentTypes[0], argumentTypes[1]),
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// widerFloat returns the wider float result for a MOD whose operands include at least one
// float. Any non-float operand contributes the DOUBLE width MySQL uses when mixing a
// float with an integer, so the result is the wider of the two float widths (DOUBLE
// unless both floats are FLOAT).
//
// Takes left (querier_dto.SQLType) which is the left operand type.
// Takes right (querier_dto.SQLType) which is the right operand type.
//
// Returns querier_dto.SQLType which is the float type to use for the result.
func widerFloat(left querier_dto.SQLType, right querier_dto.SQLType) querier_dto.SQLType {
	bothFloat := left.Category == querier_dto.TypeCategoryFloat &&
		right.Category == querier_dto.TypeCategoryFloat
	if bothFloat && left.EngineName == "float" && right.EngineName == "float" {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float"}
	}
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "double"}
}

// resolveSum computes the return type for the SUM aggregate.
//
// Takes argumentTypes ([]querier_dto.SQLType) which carries the actual argument types at
// the call site.
//
// Returns *querier_dto.FunctionResolution which describes the aggregate result type, or
// nil when no argument is present.
// Returns error when resolution fails.
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
		DataAccess:        querier_dto.DataAccessReadOnly,
		IsAggregate:       true,
	}, nil
}

// resolveAvg computes the return type for the AVG aggregate.
//
// Returns *querier_dto.FunctionResolution which describes the double-typed aggregate
// result.
// Returns error when resolution fails.
func resolveAvg() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "double"},
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
		IsAggregate:       true,
	}, nil
}

// resolveIdentityAggregate returns the argument type for MIN/MAX.
//
// Takes argumentTypes ([]querier_dto.SQLType) which carries the actual argument types at
// the call site.
//
// Returns *querier_dto.FunctionResolution which echoes the first argument type as the
// aggregate result, or nil when no argument is present.
// Returns error when resolution fails.
func resolveIdentityAggregate(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < minArgumentsSingleArgFunction {
		return nil, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        argumentTypes[0],
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
		IsAggregate:       true,
	}, nil
}

// resolveIdentityFunction returns the first argument type for non-aggregate identity
// functions such as NULLIF and the value window functions FIRST_VALUE, LAST_VALUE, LAG,
// and LEAD.
//
// Takes argumentTypes ([]querier_dto.SQLType) which carries the actual argument types at
// the call site.
//
// Returns *querier_dto.FunctionResolution which echoes the first argument type as the
// result, or nil when no argument is present.
// Returns error when resolution fails.
func resolveIdentityFunction(argumentTypes []querier_dto.SQLType) (*querier_dto.FunctionResolution, error) {
	if len(argumentTypes) < minArgumentsSingleArgFunction {
		return nil, nil
	}

	return &querier_dto.FunctionResolution{
		ReturnType:        argumentTypes[0],
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
	}, nil
}

// resolveCount returns the BIGINT non-null resolution used by COUNT.
//
// Returns *querier_dto.FunctionResolution which describes the bigint aggregate result.
// Returns error when resolution fails.
func resolveCount() (*querier_dto.FunctionResolution, error) {
	return &querier_dto.FunctionResolution{
		ReturnType:        querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
		DataAccess:        querier_dto.DataAccessReadOnly,
		IsAggregate:       true,
	}, nil
}

// promoteTypes returns the wider of two SQL types under MySQL rules.
//
// When both types share the same category the first is returned. Otherwise the category
// hierarchy (text > decimal > float > integer) determines the result.
//
// Takes left (querier_dto.SQLType) which is the left operand type.
// Takes right (querier_dto.SQLType) which is the right operand type.
//
// Returns querier_dto.SQLType which is the promoted SQL type.
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

// typePromotionRank returns the comparison rank for a type category.
//
// Takes category (querier_dto.SQLTypeCategory) which is the category to rank.
//
// Returns int which is the rank used for promotion comparisons.
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
