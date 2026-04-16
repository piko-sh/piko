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

package querier_dto

import "fmt"

// SQLExpressionFeature is a bitmask that declares which SQL expression kinds
// an engine adapter supports. Each engine adapter returns its supported set via
// EnginePort.SupportedExpressions(), allowing the type resolver to validate
// expressions against dialect capabilities.
//
// This follows the same pattern as ast_domain.ExpressionFeature.
type SQLExpressionFeature uint64

const (
	// SQLFeatureBinaryArithmetic allows arithmetic operators (+, -, *, /, %).
	SQLFeatureBinaryArithmetic SQLExpressionFeature = 1 << iota

	// SQLFeatureBinaryComparison allows comparison operators (=, <>, <, >, <=, >=).
	SQLFeatureBinaryComparison

	// SQLFeatureStringConcat allows the string concatenation operator (||).
	SQLFeatureStringConcat

	// SQLFeatureCaseWhen allows CASE WHEN/THEN/ELSE/END expressions.
	SQLFeatureCaseWhen

	// SQLFeatureScalarSubquery allows scalar subqueries in expression position.
	SQLFeatureScalarSubquery

	// SQLFeatureExists allows EXISTS(...) subquery expressions.
	SQLFeatureExists

	// SQLFeatureIsNull allows IS NULL and IS NOT NULL expressions.
	SQLFeatureIsNull

	// SQLFeatureInList allows IN (...) expressions.
	SQLFeatureInList

	// SQLFeatureBetween allows BETWEEN x AND y expressions.
	SQLFeatureBetween

	// SQLFeatureLogicalOp allows AND, OR logical operators.
	SQLFeatureLogicalOp

	// SQLFeatureUnaryOp allows unary operators (-, +, ~, NOT).
	SQLFeatureUnaryOp

	// SQLFeatureWindowFunction allows window functions with OVER clauses.
	SQLFeatureWindowFunction

	// SQLFeatureArraySubscript allows array subscript access (arr[1]).
	SQLFeatureArraySubscript

	// SQLFeatureJSONOp allows JSON operators (->, ->>, #>, #>>).
	SQLFeatureJSONOp

	// SQLFeaturePatternMatch allows LIKE, GLOB, REGEXP, MATCH operators.
	SQLFeaturePatternMatch

	// SQLFeatureBitwiseOp allows bitwise operators (&, |, <<, >>).
	SQLFeatureBitwiseOp

	// SQLFeatureLambda allows lambda expressions (x -> x + 1) in function arguments.
	SQLFeatureLambda

	// SQLFeatureStructFieldAccess allows struct field access (s.field_name).
	SQLFeatureStructFieldAccess
)

const (
	// SQLFeaturesBase is the set of expression features supported by most
	// SQL dialects: arithmetic, comparisons, string concat, CASE, IS NULL,
	// IN, BETWEEN, logical ops, unary ops, and pattern matching. Engine
	// adapters can use this as a starting point and add or remove flags.
	SQLFeaturesBase = SQLFeatureBinaryArithmetic |
		SQLFeatureBinaryComparison |
		SQLFeatureStringConcat |
		SQLFeatureCaseWhen |
		SQLFeatureExists |
		SQLFeatureIsNull |
		SQLFeatureInList |
		SQLFeatureBetween |
		SQLFeatureLogicalOp |
		SQLFeatureUnaryOp |
		SQLFeaturePatternMatch

	// SQLFeaturesAll enables every expression feature. Used by the mock engine
	// in tests to exercise all code paths.
	SQLFeaturesAll SQLExpressionFeature = ^SQLExpressionFeature(0)
)

// Has reports whether the feature set includes the given feature.
//
// Takes flag ([SQLExpressionFeature]) which is the feature to test for.
//
// Returns bool which is true when the feature set includes the flag.
func (feature SQLExpressionFeature) Has(flag SQLExpressionFeature) bool {
	return feature&flag != 0
}

// String returns a human-readable name for a single expression feature flag.
//
// Returns string which is the feature's display name.
func (feature SQLExpressionFeature) String() string {
	switch feature {
	case SQLFeatureBinaryArithmetic:
		return "binary arithmetic (+, -, *, /, %)"
	case SQLFeatureBinaryComparison:
		return "binary comparison (=, <>, <, >)"
	case SQLFeatureStringConcat:
		return "string concatenation (||)"
	case SQLFeatureCaseWhen:
		return "CASE WHEN expression"
	case SQLFeatureScalarSubquery:
		return "scalar subquery"
	case SQLFeatureExists:
		return "EXISTS subquery"
	case SQLFeatureIsNull:
		return "IS NULL / IS NOT NULL"
	case SQLFeatureInList:
		return "IN list"
	case SQLFeatureBetween:
		return "BETWEEN"
	case SQLFeatureLogicalOp:
		return "logical operator (AND, OR)"
	case SQLFeatureUnaryOp:
		return "unary operator (-, +, ~, NOT)"
	case SQLFeatureWindowFunction:
		return "window function (OVER)"
	case SQLFeatureArraySubscript:
		return "array subscript ([])"
	case SQLFeatureJSONOp:
		return "JSON operator (->, ->>)"
	case SQLFeaturePatternMatch:
		return "pattern match (LIKE, GLOB)"
	case SQLFeatureBitwiseOp:
		return "bitwise operator (&, |, <<, >>)"
	case SQLFeatureLambda:
		return "lambda expression (x -> expr)"
	case SQLFeatureStructFieldAccess:
		return "struct field access (s.field)"
	default:
		return fmt.Sprintf("SQLExpressionFeature(%d)", uint64(feature))
	}
}
