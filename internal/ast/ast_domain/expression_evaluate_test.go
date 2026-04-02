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

package ast_domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/maths"
)

func TestEvaluateExpression(t *testing.T) {
	t.Parallel()

	userScope := map[string]any{
		"name":     "Alice",
		"age":      30.0,
		"isActive": true,
		"roles":    []string{"admin", "editor"},
		"profile": map[string]any{
			"email": "alice@example.com",
			"prefs": map[string]any{
				"theme": "dark",
			},
		},
	}

	tests := []struct {
		expected         any
		scope            map[string]any
		name             string
		expressionString string
	}{
		{name: "String Literal", expressionString: `'hello world'`, scope: nil, expected: "hello world"},
		{name: "Integer Literal", expressionString: `123`, scope: nil, expected: 123.0},
		{name: "Float Literal", expressionString: `45.67`, scope: nil, expected: 45.67},
		{name: "Boolean Literal (true)", expressionString: `true`, scope: nil, expected: true},
		{name: "Boolean Literal (false)", expressionString: `false`, scope: nil, expected: false},
		{name: "Nil Literal", expressionString: `nil`, scope: nil, expected: nil},
		{
			name:             "Decimal Literal",
			expressionString: "19.99d",
			scope:            nil,
			expected:         maths.NewDecimalFromString("19.99"),
		},
		{
			name:             "DateTime Literal",
			expressionString: "dt'2025-08-30T10:20:30Z'",
			scope:            nil,
			expected:         time.Date(2025, 8, 30, 10, 20, 30, 0, time.UTC),
		},
		{
			name:             "Date Literal",
			expressionString: "d'2025-08-30'",
			scope:            nil,
			expected:         time.Date(2025, 8, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			name:             "Time Literal",
			expressionString: "t'15:04:05'",
			scope:            nil,

			expected: time.Date(0, 1, 1, 15, 4, 5, 0, time.UTC),
		},
		{name: "Simple Identifier", expressionString: `name`, scope: map[string]any{"name": "Bob"}, expected: "Bob"},
		{name: "Dotted Identifier", expressionString: `user.age`, scope: map[string]any{"user": userScope}, expected: 30.0},
		{name: "Deeply Dotted Identifier", expressionString: `user.profile.prefs.theme`, scope: map[string]any{"user": userScope}, expected: "dark"},
		{name: "Missing Identifier", expressionString: `address`, scope: map[string]any{"name": "Bob"}, expected: nil},
		{name: "Missing Nested Identifier", expressionString: `user.profile.phone`, scope: map[string]any{"user": userScope}, expected: nil},
		{name: "Logical NOT (true)", expressionString: `!false`, scope: nil, expected: true},
		{name: "Logical NOT (false)", expressionString: `!true`, scope: nil, expected: false},
		{name: "Logical NOT on truthy value", expressionString: `!user.isActive`, scope: map[string]any{"user": userScope}, expected: false},
		{name: "Logical NOT on falsy value (empty string)", expressionString: `!""`, scope: nil, expected: true},
		{name: "Logical NOT on falsy value (zero)", expressionString: `!0`, scope: nil, expected: true},
		{name: "Numeric Negation", expressionString: `-10`, scope: nil, expected: -10.0},
		{name: "Numeric Negation on Identifier", expressionString: `-user.age`, scope: map[string]any{"user": userScope}, expected: -30.0},
		{
			name:             "Unary Negation on Decimal",
			expressionString: "-10.5d",
			scope:            nil,
			expected:         maths.NewDecimalFromString("-10.5"),
		},
		{name: "AND (true && true)", expressionString: `true && true`, scope: nil, expected: true},
		{name: "AND (true && false)", expressionString: `true && false`, scope: nil, expected: false},
		{name: "OR (false || true)", expressionString: `false || true`, scope: nil, expected: true},
		{name: "OR (false || false)", expressionString: `false || false`, scope: nil, expected: false},
		{name: "AND with truthiness", expressionString: `'hello' && 1`, scope: nil, expected: true},
		{name: "Equal (numbers)", expressionString: `5 == 5.0`, scope: nil, expected: true},
		{name: "Equal (strings)", expressionString: `'a' == 'a'`, scope: nil, expected: true},
		{name: "Equal (booleans)", expressionString: `true == (1 > 0)`, scope: nil, expected: true},
		{name: "Not Equal", expressionString: `5 != 6`, scope: nil, expected: true},
		{name: "Greater Than", expressionString: `10 > 5`, scope: nil, expected: true},
		{name: "Less Than", expressionString: `5 < 10`, scope: nil, expected: true},
		{name: "Greater Than or Equal", expressionString: `10 >= 10`, scope: nil, expected: true},
		{name: "Less Than or Equal", expressionString: `5 <= 10`, scope: nil, expected: true},
		{name: "Cross-type comparison (numeric)", expressionString: `5 > '3'`, scope: nil, expected: true},
		{name: "Addition (numbers)", expressionString: `10.5 + 2`, scope: nil, expected: 12.5},
		{name: "Subtraction", expressionString: `10 - 2.5`, scope: nil, expected: 7.5},
		{name: "Multiplication", expressionString: `5 * 4`, scope: nil, expected: 20.0},
		{name: "Division", expressionString: `20 / 8`, scope: nil, expected: 2.5},
		{name: "Division by zero", expressionString: `10 / 0`, scope: nil, expected: 0.0},
		{name: "Modulo", expressionString: `10 % 3`, scope: nil, expected: 1.0},
		{name: "Modulo by zero", expressionString: `10 % 0`, scope: nil, expected: 0.0},
		{name: "String Concatenation", expressionString: `'hello' + ' ' + 'world'`, scope: nil, expected: "hello world"},
		{name: "String + Number", expressionString: `'age: ' + 30`, scope: nil, expected: "age: 30"},
		{name: "Number + String", expressionString: `30 + ' years'`, scope: nil, expected: "30 years"},
		{name: "String + Identifier", expressionString: `'name: ' + user.name`, scope: map[string]any{"user": userScope}, expected: "name: Alice"},
		{
			name:             "Decimal + Decimal",
			expressionString: "10.5d + 1.2d",
			scope:            nil,
			expected:         maths.NewDecimalFromString("11.7"),
		},
		{
			name:             "Decimal + Float (Promotion)",
			expressionString: "10.5d + 1.2",
			scope:            nil,
			expected:         maths.NewDecimalFromString("11.7"),
		},
		{
			name:             "Decimal > Decimal",
			expressionString: "10d > 5d",
			scope:            nil,
			expected:         true,
		},
		{
			name:             "Date > Date",
			expressionString: "d'2025-02-01' > d'2025-01-01'",
			scope:            nil,
			expected:         true,
		},
		{
			name:             "Date comparison with scope variable",
			expressionString: "releaseDate < d'2025-01-01'",
			scope:            map[string]any{"releaseDate": time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)},
			expected:         true,
		},
		{
			name:             "DateTime - DateTime (returns duration)",
			expressionString: "dt'2025-01-01T01:00:00Z' - dt'2025-01-01T00:00:00Z'",
			scope:            nil,
			expected:         time.Hour,
		},
		{name: "Precedence (add/mul)", expressionString: `2 + 3 * 4`, scope: nil, expected: 14.0},
		{name: "Precedence (mul/add)", expressionString: `3 * 4 + 2`, scope: nil, expected: 14.0},
		{name: "Grouping with parentheses", expressionString: `(2 + 3) * 4`, scope: nil, expected: 20.0},
		{name: "Complex precedence", expressionString: `user.age > 18 && (user.profile.prefs.theme == 'dark' || !user.isActive)`, scope: map[string]any{"user": userScope}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.expressionString)
			result := EvaluateExpression(expression, tt.scope)

			if expectedDecimal, ok := tt.expected.(maths.Decimal); ok {
				resultDecimal, ok := result.(maths.Decimal)
				require.True(t, ok, "Expected result to be maths.Decimal")
				eq, err := expectedDecimal.Equals(resultDecimal)
				require.NoError(t, err)
				assert.True(t, eq, "Expected decimal %s, got %s", expectedDecimal.MustString(), resultDecimal.MustString())
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEvaluateExpression_ObjectLiteral(t *testing.T) {
	t.Parallel()

	scope := map[string]any{
		"user": map[string]any{
			"isActive": true,
			"name":     "Alice",
		},
		"theme": "dark",
	}

	tests := []struct {
		expected         map[string]any
		name             string
		expressionString string
	}{
		{
			name:             "Object with literals",
			expressionString: `{ name: 'Bob', age: 30, active: true }`,
			expected:         map[string]any{"name": "Bob", "age": 30.0, "active": true},
		},
		{
			name:             "Object with simple identifiers",
			expressionString: `{ theme: theme, 'user-active': user.isActive }`,
			expected:         map[string]any{"theme": "dark", "user-active": true},
		},
		{
			name:             "Object with complex expressions",
			expressionString: `{ fullName: user.name + ' Smith', status: !user.isActive, nextAge: 30 + 1 }`,
			expected:         map[string]any{"fullName": "Alice Smith", "status": false, "nextAge": 31.0},
		},
		{
			name:             "Empty object",
			expressionString: `{}`,
			expected:         map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.expressionString)
			result := EvaluateExpression(expression, scope)

			resultMap, ok := result.(map[string]any)
			require.True(t, ok, "Evaluator should return a map[string]any")

			assert.Equal(t, tt.expected, resultMap)
		})
	}
}

func TestEvaluateExpression_ArrayLiteral(t *testing.T) {
	t.Parallel()

	scope := map[string]any{
		"user": map[string]any{
			"isActive": true,
			"name":     "Alice",
		},
		"count": 10.0,
	}

	tests := []struct {
		name             string
		expressionString string
		expected         []any
	}{
		{
			name:             "Array with literals",
			expressionString: `['a', 1, true, nil]`,
			expected:         []any{"a", 1.0, true, nil},
		},
		{
			name:             "Array with identifiers",
			expressionString: `[user.name, count, user.isActive]`,
			expected:         []any{"Alice", 10.0, true},
		},
		{
			name:             "Array with complex expressions",
			expressionString: `[1 + 2, count * 2, user.name + '!']`,
			expected:         []any{3.0, 20.0, "Alice!"},
		},
		{
			name:             "Empty array",
			expressionString: `[]`,
			expected:         []any{},
		},
		{
			name:             "Nested array",
			expressionString: `[1, [2, 3], 4]`,
			expected:         []any{1.0, []any{2.0, 3.0}, 4.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.expressionString)
			result := EvaluateExpression(expression, scope)

			resultSlice, ok := result.([]any)
			require.True(t, ok, "Evaluator should return a []any")

			assert.Equal(t, tt.expected, resultSlice)
		})
	}
}

func TestEvaluateExpression_NullishCoalescing(t *testing.T) {
	t.Parallel()

	scope := map[string]any{
		"nilValue":    nil,
		"falseValue":  false,
		"zeroValue":   0.0,
		"emptyString": "",
		"realValue":   "hello",
		"realNum":     42.0,
	}

	tests := []struct {
		expected         any
		name             string
		expressionString string
	}{
		{
			name:             "Should return right-hand side for nil",
			expressionString: "nilValue ?? 'default'",
			expected:         "default",
		},
		{
			name:             "Should return left-hand side for false",
			expressionString: "falseValue ?? 'default'",
			expected:         false,
		},
		{
			name:             "Should return left-hand side for zero",
			expressionString: "zeroValue ?? 'default'",
			expected:         0.0,
		},
		{
			name:             "Should return left-hand side for empty string",
			expressionString: "emptyString ?? 'default'",
			expected:         "",
		},
		{
			name:             "Should return left-hand side for non-nil value",
			expressionString: "realValue ?? 'default'",
			expected:         "hello",
		},
		{
			name:             "Should return left-hand side for non-nil number",
			expressionString: "realNum ?? 100.0",
			expected:         42.0,
		},
		{
			name:             "Should chain correctly, picking first non-nil",
			expressionString: "nilValue ?? nilValue ?? 'default'",
			expected:         "default",
		},
		{
			name:             "Should chain correctly, stopping at first non-nil",
			expressionString: "nilValue ?? falseValue ?? 'default'",
			expected:         false,
		},
		{
			name:             "Works with member access",
			expressionString: "user.profile?.nonExistent ?? 'fallback'",
			expected:         "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.expressionString)
			result := EvaluateExpression(expression, scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertArgument_FloatToInt(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expected         any
		scope            map[string]any
		name             string
		expressionString string
	}{
		{
			name:             "float64 argument is converted to int64 parameter",
			expressionString: "doubleInt(5)",
			scope: map[string]any{
				"doubleInt": func(n int64) int64 { return n * 2 },
			},
			expected: int64(10),
		},
		{
			name:             "float64 argument with decimal part truncates to int64",
			expressionString: "doubleInt(5.7)",
			scope: map[string]any{
				"doubleInt": func(n int64) int64 { return n * 2 },
			},
			expected: int64(10),
		},
		{
			name:             "negative float to int",
			expressionString: "abs(-42)",
			scope: map[string]any{
				"abs": func(n int64) int64 {
					if n < 0 {
						return -n
					}
					return n
				},
			},
			expected: int64(42),
		},
		{
			name:             "zero float to int",
			expressionString: "identity(0)",
			scope: map[string]any{
				"identity": func(n int64) int64 { return n },
			},
			expected: int64(0),
		},
		{
			name:             "float argument to int parameter via expression",
			expressionString: "process(10 + 5)",
			scope: map[string]any{
				"process": func(n int64) int64 { return n + 1 },
			},
			expected: int64(16),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tc.expressionString)
			result := EvaluateExpression(expression, tc.scope)
			assert.Equal(t, tc.expected, result)
		})
	}
}
