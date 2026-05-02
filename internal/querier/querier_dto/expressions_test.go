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

func TestExpressionKind_ReturnsStableTagPerConcreteType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		expression Expression
		expected   string
	}{
		{name: "column ref", expression: &ColumnRefExpression{}, expected: "column_ref"},
		{name: "function call", expression: &FunctionCallExpression{}, expected: "function_call"},
		{name: "coalesce", expression: &CoalesceExpression{}, expected: "coalesce"},
		{name: "cast", expression: &CastExpression{}, expected: "cast"},
		{name: "literal", expression: &LiteralExpression{}, expected: "literal"},
		{name: "binary op", expression: &BinaryOpExpression{}, expected: "binary_op"},
		{name: "comparison", expression: &ComparisonExpression{}, expected: "comparison"},
		{name: "is null", expression: &IsNullExpression{}, expected: "is_null"},
		{name: "in list", expression: &InListExpression{}, expected: "in_list"},
		{name: "between", expression: &BetweenExpression{}, expected: "between"},
		{name: "logical op", expression: &LogicalOpExpression{}, expected: "logical_op"},
		{name: "unary op", expression: &UnaryOpExpression{}, expected: "unary_op"},
		{name: "case when", expression: &CaseWhenExpression{}, expected: "case_when"},
		{name: "exists", expression: &ExistsExpression{}, expected: "exists"},
		{name: "window function", expression: &WindowFunctionExpression{}, expected: "window_function"},
		{name: "scalar subquery", expression: &ScalarSubqueryExpression{}, expected: "scalar_subquery"},
		{name: "array subscript", expression: &ArraySubscriptExpression{}, expected: "array_subscript"},
		{name: "lambda", expression: &LambdaExpression{}, expected: "lambda"},
		{name: "struct field access", expression: &StructFieldAccessExpression{}, expected: "struct_field_access"},
		{name: "unknown", expression: &UnknownExpression{}, expected: "unknown"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, testCase.expression.expressionKind())
		})
	}
}
