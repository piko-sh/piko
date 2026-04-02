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

package driven_code_emitter_go_literal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestVisitExpression(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression    ast_domain.Expression
		name          string
		expectedCalls int
		stopEarly     bool
	}{
		{
			name: "visits MemberExpr and children",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "user"},
				Property: &ast_domain.Identifier{Name: "name"},
			},
			stopEarly:     false,
			expectedCalls: 3,
		},
		{
			name: "visits IndexExpr and children",
			expression: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "items"},
				Index: &ast_domain.IntegerLiteral{Value: 0},
			},
			stopEarly:     false,
			expectedCalls: 3,
		},
		{
			name: "visits UnaryExpr and operand",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNeg,
				Right:    &ast_domain.IntegerLiteral{Value: 42},
			},
			stopEarly:     false,
			expectedCalls: 2,
		},
		{
			name: "visits BinaryExpr and both operands",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.IntegerLiteral{Value: 1},
				Operator: ast_domain.OpPlus,
				Right:    &ast_domain.IntegerLiteral{Value: 2},
			},
			stopEarly:     false,
			expectedCalls: 3,
		},
		{
			name: "visits CallExpr, callee, and arguments",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "fn"},
				Args: []ast_domain.Expression{
					&ast_domain.Identifier{Name: "arg1"},
					&ast_domain.Identifier{Name: "arg2"},
				},
			},
			stopEarly:     false,
			expectedCalls: 4,
		},
		{
			name: "visits TemplateLiteral and dynamic parts",
			expression: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "prefix"},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "var"}},
					{IsLiteral: true, Literal: "suffix"},
				},
			},
			stopEarly:     false,
			expectedCalls: 2,
		},
		{
			name: "visits ObjectLiteral and values",
			expression: &ast_domain.ObjectLiteral{
				Pairs: map[string]ast_domain.Expression{
					"key1": &ast_domain.Identifier{Name: "val1"},
					"key2": &ast_domain.Identifier{Name: "val2"},
				},
			},
			stopEarly:     false,
			expectedCalls: 3,
		},
		{
			name: "visits ArrayLiteral and elements",
			expression: &ast_domain.ArrayLiteral{
				Elements: []ast_domain.Expression{
					&ast_domain.IntegerLiteral{Value: 1},
					&ast_domain.IntegerLiteral{Value: 2},
				},
			},
			stopEarly:     false,
			expectedCalls: 3,
		},
		{
			name: "visits TernaryExpr and all branches",
			expression: &ast_domain.TernaryExpression{
				Condition:  &ast_domain.BooleanLiteral{Value: true},
				Consequent: &ast_domain.IntegerLiteral{Value: 1},
				Alternate:  &ast_domain.IntegerLiteral{Value: 2},
			},
			stopEarly:     false,
			expectedCalls: 4,
		},
		{
			name:          "stops when visitor returns false",
			expression:    &ast_domain.Identifier{Name: "test"},
			stopEarly:     true,
			expectedCalls: 1,
		},
		{
			name:          "handles nil expression",
			expression:    nil,
			stopEarly:     false,
			expectedCalls: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			callCount := 0
			visitor := func(expression ast_domain.Expression) bool {
				callCount++
				return !tc.stopEarly
			}

			visitExpression(tc.expression, visitor)

			assert.Equal(t, tc.expectedCalls, callCount, "Should visit expected number of nodes")
		})
	}
}

func TestVisitMemberExpression(t *testing.T) {
	t.Parallel()

	callCount := 0
	visitor := func(expression ast_domain.Expression) bool {
		callCount++
		return true
	}

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "obj"},
		Property: &ast_domain.Identifier{Name: "field"},
	}

	visitMemberExpression(expression, visitor)

	assert.Equal(t, 2, callCount, "Should visit both base and property")
}

func TestVisitIndexExpression(t *testing.T) {
	t.Parallel()

	callCount := 0
	visitor := func(expression ast_domain.Expression) bool {
		callCount++
		return true
	}

	expression := &ast_domain.IndexExpression{
		Base:  &ast_domain.Identifier{Name: "arr"},
		Index: &ast_domain.IntegerLiteral{Value: 0},
	}

	visitIndexExpression(expression, visitor)

	assert.Equal(t, 2, callCount, "Should visit both base and index")
}

func TestVisitBinaryExpression(t *testing.T) {
	t.Parallel()

	callCount := 0
	visitor := func(expression ast_domain.Expression) bool {
		callCount++
		return true
	}

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Operator: ast_domain.OpPlus,
		Right:    &ast_domain.Identifier{Name: "b"},
	}

	visitBinaryExpression(expression, visitor)

	assert.Equal(t, 2, callCount, "Should visit both left and right")
}

func TestVisitCallExpression(t *testing.T) {
	t.Parallel()

	callCount := 0
	visitor := func(expression ast_domain.Expression) bool {
		callCount++
		return true
	}

	expression := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "fn"},
		Args: []ast_domain.Expression{
			&ast_domain.Identifier{Name: "arg1"},
			&ast_domain.Identifier{Name: "arg2"},
		},
	}

	visitCallExpression(expression, visitor)

	assert.Equal(t, 3, callCount, "Should visit callee and 2 arguments")
}

func TestVisitTemplateLiteral(t *testing.T) {
	t.Parallel()

	callCount := 0
	visitor := func(expression ast_domain.Expression) bool {
		callCount++
		return true
	}

	expression := &ast_domain.TemplateLiteral{
		Parts: []ast_domain.TemplateLiteralPart{
			{IsLiteral: true, Literal: "static"},
			{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "var1"}},
			{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "var2"}},
		},
	}

	visitTemplateLiteral(expression, visitor)

	assert.Equal(t, 2, callCount, "Should visit only dynamic parts")
}

func TestVisitObjectLiteral(t *testing.T) {
	t.Parallel()

	callCount := 0
	visitor := func(expression ast_domain.Expression) bool {
		callCount++
		return true
	}

	expression := &ast_domain.ObjectLiteral{
		Pairs: map[string]ast_domain.Expression{
			"key1": &ast_domain.Identifier{Name: "val1"},
			"key2": &ast_domain.Identifier{Name: "val2"},
			"key3": &ast_domain.Identifier{Name: "val3"},
		},
	}

	visitObjectLiteral(expression, visitor)

	assert.Equal(t, 3, callCount, "Should visit all 3 values")
}

func TestVisitArrayLiteral(t *testing.T) {
	t.Parallel()

	callCount := 0
	visitor := func(expression ast_domain.Expression) bool {
		callCount++
		return true
	}

	expression := &ast_domain.ArrayLiteral{
		Elements: []ast_domain.Expression{
			&ast_domain.IntegerLiteral{Value: 1},
			&ast_domain.IntegerLiteral{Value: 2},
			&ast_domain.IntegerLiteral{Value: 3},
		},
	}

	visitArrayLiteral(expression, visitor)

	assert.Equal(t, 3, callCount, "Should visit all 3 elements")
}

func TestVisitTernaryExpression(t *testing.T) {
	t.Parallel()

	callCount := 0
	visitor := func(expression ast_domain.Expression) bool {
		callCount++
		return true
	}

	expression := &ast_domain.TernaryExpression{
		Condition:  &ast_domain.BooleanLiteral{Value: true},
		Consequent: &ast_domain.IntegerLiteral{Value: 1},
		Alternate:  &ast_domain.IntegerLiteral{Value: 2},
	}

	visitTernaryExpression(expression, visitor)

	assert.Equal(t, 3, callCount, "Should visit all 3 parts")
}

func TestVisitExpression_NestedStructures(t *testing.T) {
	t.Parallel()

	callCount := 0
	visitor := func(expression ast_domain.Expression) bool {
		callCount++
		return true
	}

	expression := &ast_domain.BinaryExpression{
		Left: &ast_domain.BinaryExpression{
			Left: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "user"},
				Property: &ast_domain.Identifier{Name: "name"},
			},
			Operator: ast_domain.OpPlus,
			Right:    &ast_domain.StringLiteral{Value: "-"},
		},
		Operator: ast_domain.OpPlus,
		Right: &ast_domain.CallExpression{
			Callee: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "strconv"},
				Property: &ast_domain.Identifier{Name: "Itoa"},
			},
			Args: []ast_domain.Expression{
				&ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "user"},
					Property: &ast_domain.Identifier{Name: "age"},
				},
			},
		},
	}

	visitExpression(expression, visitor)

	assert.GreaterOrEqual(t, callCount, 10, "Should visit many nodes in nested structure")
}

func TestVisitExpression_StopsOnFalse(t *testing.T) {
	t.Parallel()

	callCount := 0
	visitor := func(expression ast_domain.Expression) bool {
		callCount++

		return false
	}

	expression := &ast_domain.BinaryExpression{
		Left:     &ast_domain.Identifier{Name: "a"},
		Operator: ast_domain.OpPlus,
		Right:    &ast_domain.Identifier{Name: "b"},
	}

	visitExpression(expression, visitor)

	assert.Equal(t, 1, callCount, "Should stop traversal when visitor returns false")
}
