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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSQLExpressionFeatureString_RendersEachSingleBit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		feature  SQLExpressionFeature
		expected string
	}{
		{name: "binary arithmetic", feature: SQLFeatureBinaryArithmetic, expected: "binary arithmetic (+, -, *, /, %)"},
		{name: "binary comparison", feature: SQLFeatureBinaryComparison, expected: "binary comparison (=, <>, <, >)"},
		{name: "string concat", feature: SQLFeatureStringConcat, expected: "string concatenation (||)"},
		{name: "case when", feature: SQLFeatureCaseWhen, expected: "CASE WHEN expression"},
		{name: "scalar subquery", feature: SQLFeatureScalarSubquery, expected: "scalar subquery"},
		{name: "exists", feature: SQLFeatureExists, expected: "EXISTS subquery"},
		{name: "is null", feature: SQLFeatureIsNull, expected: "IS NULL / IS NOT NULL"},
		{name: "in list", feature: SQLFeatureInList, expected: "IN list"},
		{name: "between", feature: SQLFeatureBetween, expected: "BETWEEN"},
		{name: "logical op", feature: SQLFeatureLogicalOp, expected: "logical operator (AND, OR)"},
		{name: "unary op", feature: SQLFeatureUnaryOp, expected: "unary operator (-, +, ~, NOT)"},
		{name: "window function", feature: SQLFeatureWindowFunction, expected: "window function (OVER)"},
		{name: "array subscript", feature: SQLFeatureArraySubscript, expected: "array subscript ([])"},
		{name: "json op", feature: SQLFeatureJSONOp, expected: "JSON operator (->, ->>)"},
		{name: "pattern match", feature: SQLFeaturePatternMatch, expected: "pattern match (LIKE, GLOB)"},
		{name: "bitwise op", feature: SQLFeatureBitwiseOp, expected: "bitwise operator (&, |, <<, >>)"},
		{name: "lambda", feature: SQLFeatureLambda, expected: "lambda expression (x -> expr)"},
		{name: "struct field access", feature: SQLFeatureStructFieldAccess, expected: "struct field access (s.field)"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, testCase.feature.String())
		})
	}
}

func TestSQLExpressionFeatureString_RendersFallbackForZeroCombinationAndAllBits(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		feature  SQLExpressionFeature
		expected string
	}{
		{
			name:     "zero value uses numeric fallback",
			feature:  SQLExpressionFeature(0),
			expected: "SQLExpressionFeature(0)",
		},
		{
			name:     "combination of two bits uses numeric fallback",
			feature:  SQLFeatureBinaryArithmetic | SQLFeatureBinaryComparison,
			expected: "SQLExpressionFeature(3)",
		},
		{
			name:     "all bits uses numeric fallback",
			feature:  SQLFeaturesAll,
			expected: "SQLExpressionFeature(18446744073709551615)",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, testCase.feature.String())
		})
	}
}

func TestSQLExpressionFeatureHas_ReportsMembership(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		set      SQLExpressionFeature
		flag     SQLExpressionFeature
		expected bool
	}{
		{name: "single set bit is present", set: SQLFeatureInList, flag: SQLFeatureInList, expected: true},
		{name: "absent bit reports false", set: SQLFeatureInList, flag: SQLFeatureBetween, expected: false},
		{name: "base set contains comparison", set: SQLFeaturesBase, flag: SQLFeatureBinaryComparison, expected: true},
		{name: "base set lacks window function", set: SQLFeaturesBase, flag: SQLFeatureWindowFunction, expected: false},
		{name: "base set lacks scalar subquery", set: SQLFeaturesBase, flag: SQLFeatureScalarSubquery, expected: false},
		{name: "all set contains every flag", set: SQLFeaturesAll, flag: SQLFeatureStructFieldAccess, expected: true},
		{name: "zero set contains nothing", set: SQLExpressionFeature(0), flag: SQLFeatureInList, expected: false},
		{name: "combined set contains either member", set: SQLFeatureInList | SQLFeatureBetween, flag: SQLFeatureBetween, expected: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, testCase.set.Has(testCase.flag))
		})
	}
}

func TestSQLFeaturesBase_CombinesExpectedBaselineFlags(t *testing.T) {
	t.Parallel()

	expected := SQLFeatureBinaryArithmetic |
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

	assert.Equal(t, expected, SQLFeaturesBase)
}

func TestSQLFeaturesAll_EnablesEveryDefinedFeature(t *testing.T) {
	t.Parallel()

	features := []SQLExpressionFeature{
		SQLFeatureBinaryArithmetic,
		SQLFeatureBinaryComparison,
		SQLFeatureStringConcat,
		SQLFeatureCaseWhen,
		SQLFeatureScalarSubquery,
		SQLFeatureExists,
		SQLFeatureIsNull,
		SQLFeatureInList,
		SQLFeatureBetween,
		SQLFeatureLogicalOp,
		SQLFeatureUnaryOp,
		SQLFeatureWindowFunction,
		SQLFeatureArraySubscript,
		SQLFeatureJSONOp,
		SQLFeaturePatternMatch,
		SQLFeatureBitwiseOp,
		SQLFeatureLambda,
		SQLFeatureStructFieldAccess,
	}

	for _, feature := range features {
		assert.True(t, SQLFeaturesAll.Has(feature), "all should contain %s", feature)
	}
}
