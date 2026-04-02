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
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_LiteralsAndIdentifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect    any
		validator func(t *testing.T, expression Expression)
		name      string
		input     string
	}{
		{
			name:   "Integer Literal",
			input:  "123",
			expect: &IntegerLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*IntegerLiteral)
				require.True(t, ok)
				assert.Equal(t, int64(123), lit.Value)
			},
		},
		{
			name:   "Float Literal",
			input:  "123.45",
			expect: &FloatLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*FloatLiteral)
				require.True(t, ok)
				assert.Equal(t, 123.45, lit.Value)
			},
		},
		{
			name:   "String Literal (single quotes)",
			input:  `'hello world'`,
			expect: &StringLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*StringLiteral)
				require.True(t, ok)
				assert.Equal(t, "hello world", lit.Value)
			},
		},
		{
			name:   "String Literal (double quotes)",
			input:  `"hello world"`,
			expect: &StringLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*StringLiteral)
				require.True(t, ok)
				assert.Equal(t, "hello world", lit.Value)
			},
		},
		{
			name:   "Boolean Literal (true)",
			input:  "true",
			expect: &BooleanLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*BooleanLiteral)
				require.True(t, ok)
				assert.True(t, lit.Value)
			},
		},
		{
			name:   "Boolean Literal (false)",
			input:  "false",
			expect: &BooleanLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*BooleanLiteral)
				require.True(t, ok)
				assert.False(t, lit.Value)
			},
		},
		{
			name:   "Nil Literal",
			input:  "nil",
			expect: &NilLiteral{},
		},
		{
			name:   "Decimal Literal (with fraction)",
			input:  "99.99d",
			expect: &DecimalLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*DecimalLiteral)
				require.True(t, ok)
				assert.Equal(t, "99.99", lit.Value)
			},
		},
		{
			name:   "Decimal Literal (integer)",
			input:  "100d",
			expect: &DecimalLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*DecimalLiteral)
				require.True(t, ok)
				assert.Equal(t, "100", lit.Value)
			},
		},
		{
			name:   "BigInt Literal",
			input:  "12345678901234567890n",
			expect: &BigIntLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*BigIntLiteral)
				require.True(t, ok)
				assert.Equal(t, "12345678901234567890", lit.Value)
			},
		},
		{
			name:   "BigInt Literal zero",
			input:  "0n",
			expect: &BigIntLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*BigIntLiteral)
				require.True(t, ok)
				assert.Equal(t, "0", lit.Value)
			},
		},
		{
			name:   "Rune Literal (ASCII)",
			input:  "r'a'",
			expect: &RuneLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*RuneLiteral)
				require.True(t, ok)
				assert.Equal(t, rune('a'), lit.Value)
			},
		},
		{
			name:   "Rune Literal (Unicode)",
			input:  "r'🚀'",
			expect: &RuneLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*RuneLiteral)
				require.True(t, ok)
				assert.Equal(t, rune('🚀'), lit.Value)
			},
		},
		{
			name:   "Rune Literal (Escaped)",
			input:  `r'\n'`,
			expect: &RuneLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*RuneLiteral)
				require.True(t, ok)
				assert.Equal(t, rune('\n'), lit.Value)
			},
		},
		{
			name:   "DateTime Literal",
			input:  "dt'2025-08-30T15:04:05Z'",
			expect: &DateTimeLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*DateTimeLiteral)
				require.True(t, ok)
				assert.Equal(t, "2025-08-30T15:04:05Z", lit.Value)
			},
		},
		{
			name:   "Date Literal",
			input:  "d'2025-08-30'",
			expect: &DateLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*DateLiteral)
				require.True(t, ok)
				assert.Equal(t, "2025-08-30", lit.Value)
			},
		},
		{
			name:   "Time Literal",
			input:  "t'15:04:05'",
			expect: &TimeLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*TimeLiteral)
				require.True(t, ok)
				assert.Equal(t, "15:04:05", lit.Value)
			},
		},
		{
			name:   "Duration Literal",
			input:  "du'1h30m15s'",
			expect: &DurationLiteral{},
			validator: func(t *testing.T, expression Expression) {
				lit, ok := expression.(*DurationLiteral)
				require.True(t, ok)
				assert.Equal(t, "1h30m15s", lit.Value)
			},
		},
		{
			name:   "Simple Identifier",
			input:  "myVar",
			expect: &Identifier{},
			validator: func(t *testing.T, expression Expression) {
				identifier, ok := expression.(*Identifier)
				require.True(t, ok)
				assert.Equal(t, "myVar", identifier.Name)
			},
		},
		{
			name:   "Identifier with underscores",
			input:  "my_var_1",
			expect: &Identifier{},
			validator: func(t *testing.T, expression Expression) {
				identifier, ok := expression.(*Identifier)
				require.True(t, ok)
				assert.Equal(t, "my_var_1", identifier.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.input)
			assert.IsType(t, tt.expect, expression)
			if tt.validator != nil {
				tt.validator(t, expression)
			}
		})
	}
}

func TestParser_UnaryAndBinaryOperators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		expectedString string
	}{

		{name: "Unary Not", input: "!isActive", expectedString: "!isActive"},
		{name: "Unary Negation", input: "-count", expectedString: "-count"},

		{name: "Addition", input: "a + b", expectedString: "(a + b)"},
		{name: "Subtraction", input: "a - b", expectedString: "(a - b)"},
		{name: "Multiplication", input: "a * b", expectedString: "(a * b)"},
		{name: "Division", input: "a / b", expectedString: "(a / b)"},
		{name: "Modulo", input: "a % b", expectedString: "(a % b)"},

		{name: "Equal", input: "a == b", expectedString: "(a == b)"},
		{name: "Not Equal", input: "a != b", expectedString: "(a != b)"},
		{name: "Greater Than", input: "a > b", expectedString: "(a > b)"},
		{name: "Less Than", input: "a < b", expectedString: "(a < b)"},
		{name: "Greater Than or Equal", input: "a >= b", expectedString: "(a >= b)"},
		{name: "Less Than or Equal", input: "a <= b", expectedString: "(a <= b)"},

		{name: "And", input: "a && b", expectedString: "(a && b)"},
		{name: "Or", input: "a || b", expectedString: "(a || b)"},
		{name: "Nullish Coalescing", input: "a ?? b", expectedString: "(a ?? b)"},
		{name: "Coalescing vs Or", input: "a || b ?? c", expectedString: "(a || (b ?? c))"},
		{name: "Coalescing vs And", input: "a && b ?? c", expectedString: "((a && b) ?? c)"},
		{name: "Coalescing with grouping", input: "(a || b) ?? c", expectedString: "((a || b) ?? c)"},

		{name: "Addition then Multiplication", input: "a + b * c", expectedString: "(a + (b * c))"},
		{name: "Multiplication then Addition", input: "a * b + c", expectedString: "((a * b) + c)"},
		{name: "Parentheses override precedence", input: "(a + b) * c", expectedString: "((a + b) * c)"},
		{name: "Complex logical precedence", input: "a || b && c", expectedString: "(a || (b && c))"},
		{name: "Complex logical with grouping", input: "(a || b) && c", expectedString: "((a || b) && c)"},
		{name: "Unary precedence", input: "!a && b", expectedString: "(!a && b)"},
		{name: "Nested unary", input: "-(-a)", expectedString: "-(-a)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.input)
			assertExprString(t, tt.expectedString, expression)
		})
	}
}

func TestParser_FirstClassLiterals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		expectedString string
	}{
		{name: "Decimal addition", input: "10.5d + 2.5d", expectedString: "(10.5d + 2.5d)"},
		{name: "Decimal comparison", input: "price > 99.99d", expectedString: "(price > 99.99d)"},
		{name: "Decimal with float", input: "10d * 1.5", expectedString: "(10d * 1.5)"},
		{name: "Decimal with negative unary", input: "-cost", expectedString: "-cost"},
		{name: "Date plus duration", input: "d'2025-01-01' + du'24h'", expectedString: "(d'2025-01-01' + du'24h')"},
		{name: "Datetime minus datetime", input: "dt'2025-01-02T00:00:00Z' - dt'2025-01-01T00:00:00Z'", expectedString: "(dt'2025-01-02T00:00:00Z' - dt'2025-01-01T00:00:00Z')"},
		{name: "Duration comparison", input: "du'1h' > du'30m'", expectedString: "(du'1h' > du'30m')"},
		{name: "Now() plus duration", input: "now() + du'5m'", expectedString: "(now() + du'5m')"},
		{name: "Decimal in array", input: "[1.0d, 2.5d, 3d]", expectedString: "[1.0d, 2.5d, 3d]"},
		{name: "Date in object", input: "{ start: d'2025-01-01', end: d'2025-12-31' }", expectedString: `{"end": d'2025-12-31', "start": d'2025-01-01'}`},
		{name: "Duration in ternary", input: "isRush ? du'1h' : du'24h'", expectedString: "(isRush ? du'1h' : du'24h')"},
		{name: "Datetime in template literal", input: "`Event starts at ${dt'2025-01-01T10:00:00Z'}`", expectedString: "`Event starts at ${dt'2025-01-01T10:00:00Z'}`"},
		{name: "Duration with multiple units", input: "du'1h30m15.5s'", expectedString: "du'1h30m15.5s'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.input)
			assertExprString(t, tt.expectedString, expression)
		})
	}
}

func TestParser_MemberAndCallExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		validator      func(t *testing.T, expression Expression)
		name           string
		input          string
		expectedString string
	}{
		{name: "Member Access", input: "user.name", expectedString: "user.name", validator: nil},
		{name: "Chained Member Access", input: "user.profile.email", expectedString: "user.profile.email", validator: nil},
		{name: "Optional Chaining", input: "user?.profile", expectedString: "user?.profile", validator: nil},
		{name: "Index Access (number)", input: "items[0]", expectedString: "items[0]", validator: nil},
		{name: "Index Access (string)", input: `items['key']`, expectedString: `items["key"]`, validator: nil},
		{name: "Index Access (identifier)", input: "items[key]", expectedString: "items[key]", validator: nil},
		{
			name:           "Optional Index Access",
			input:          "items?.[0]",
			expectedString: "items?.[0]",
			validator: func(t *testing.T, expression Expression) {
				ie, ok := expression.(*IndexExpression)
				require.True(t, ok, "Expected expression to be of type *IndexExpression")

				assert.True(t, ie.Optional, "IndexExpression.Optional flag should be true")
			},
		},
		{name: "Chained Mixed Access", input: "user.roles[0].name", expectedString: "user.roles[0].name", validator: nil},
		{name: "Function Call (no arguments)", input: "doSomething()", expectedString: "doSomething()", validator: nil},
		{name: "Function Call (with arguments)", input: `doSomething(1, 'hello', user)`, expectedString: `doSomething(1, "hello", user)`, validator: nil},
		{name: "Method Call", input: "user.getName()", expectedString: "user.getName()", validator: nil},
		{name: "Chained Method Call", input: "user.getProfile().getEmail()", expectedString: "user.getProfile().getEmail()", validator: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.input)
			assertExprString(t, tt.expectedString, expression)

			if tt.validator != nil {
				tt.validator(t, expression)
			}
		})
	}
}

func TestParser_ForInExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		validator      func(t *testing.T, expression Expression)
		expectedString string
	}{
		{
			name:           "Simple for-in (value only)",
			input:          "item in items",
			expectedString: "item in items",
			validator: func(t *testing.T, expression Expression) {
				forExpr, ok := expression.(*ForInExpression)
				require.True(t, ok)
				assert.Nil(t, forExpr.IndexVariable)
				require.NotNil(t, forExpr.ItemVariable)
				assert.Equal(t, "item", forExpr.ItemVariable.Name)
				assertExprString(t, "items", forExpr.Collection)
			},
		},
		{
			name:           "For-in with index and value",
			input:          "(index, item) in items",
			expectedString: "(index, item) in items",
			validator: func(t *testing.T, expression Expression) {
				forExpr, ok := expression.(*ForInExpression)
				require.True(t, ok)
				require.NotNil(t, forExpr.IndexVariable)
				assert.Equal(t, "index", forExpr.IndexVariable.Name)
				require.NotNil(t, forExpr.ItemVariable)
				assert.Equal(t, "item", forExpr.ItemVariable.Name)
				assertExprString(t, "items", forExpr.Collection)
			},
		},
		{
			name:           "For-in with complex collection",
			input:          "user in getUsers(filter)",
			expectedString: "user in getUsers(filter)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.input)
			assertExprString(t, tt.expectedString, expression)
			if tt.validator != nil {
				tt.validator(t, expression)
			}
		})
	}
}

func TestParser_TemplateLiterals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		validator func(t *testing.T, tl *TemplateLiteral)
		name      string
		input     string
	}{
		{
			name:  "simple literal",
			input: "``",
			validator: func(t *testing.T, tl *TemplateLiteral) {
				assert.Len(t, tl.Parts, 0)
			},
		},
		{
			name:  "literal with text",
			input: "`hello world`",
			validator: func(t *testing.T, tl *TemplateLiteral) {
				require.Len(t, tl.Parts, 1)
				assert.True(t, tl.Parts[0].IsLiteral)
				assert.Equal(t, "hello world", tl.Parts[0].Literal)
			},
		},
		{
			name:  "literal with expression",
			input: "`Hello, ${user.name}!`",
			validator: func(t *testing.T, tl *TemplateLiteral) {
				require.Len(t, tl.Parts, 3)
				assert.True(t, tl.Parts[0].IsLiteral)
				assert.Equal(t, "Hello, ", tl.Parts[0].Literal)

				assert.False(t, tl.Parts[1].IsLiteral)
				assertExprString(t, "user.name", tl.Parts[1].Expression)

				assert.True(t, tl.Parts[2].IsLiteral)
				assert.Equal(t, "!", tl.Parts[2].Literal)
			},
		},
		{
			name:  "literal with complex expression",
			input: "`Total: ${items.count * item.price}`",
			validator: func(t *testing.T, tl *TemplateLiteral) {
				require.Len(t, tl.Parts, 2)
				assert.False(t, tl.Parts[1].IsLiteral)
				assertExprString(t, "(items.count * item.price)", tl.Parts[1].Expression)
			},
		},
		{
			name:  "escaped characters",
			input: "`\\`escaped backtick\\` and \\${not an expression}`",
			validator: func(t *testing.T, tl *TemplateLiteral) {
				require.Len(t, tl.Parts, 1)
				assert.True(t, tl.Parts[0].IsLiteral)
				assert.Equal(t, "`escaped backtick` and ${not an expression}", tl.Parts[0].Literal)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.input)
			tl, ok := expression.(*TemplateLiteral)
			require.True(t, ok)
			if tt.validator != nil {
				tt.validator(t, tl)
			}
		})
	}
}

func TestParser_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		errorContains string
	}{

		{name: "Unmatched parenthesis", input: "(a + b", errorContains: "Expected ')'"},

		{name: "Invalid for-in (no collection)", input: "(index, item) in", errorContains: "Expected an expression for the collection"},
		{name: "Invalid token", input: "a # b", errorContains: "unrecognised character '#'"},
		{name: "Unterminated string", input: "'hello", errorContains: "unterminated string literal"},
		{name: "Unterminated template literal", input: "`hello", errorContains: "unterminated template literal"},
		{name: "Unterminated template expression", input: "`hello ${name`", errorContains: "Unterminated expression in template literal"},

		{name: "Missing closing bracket for array", input: "[1, 2", errorContains: "Expected ']' to close array literal"},
		{name: "Missing comma in array", input: "[1 2]", errorContains: "Expected ']' to close array literal"},
		{name: "Dangling comma", input: "1, 2,", errorContains: "Invalid syntax. A comma is not a valid operator here"},
		{name: "Unexpected token at end", input: "1 + 2 3", errorContains: "Unexpected tokens after expression: '3'"},
		{name: "Unterminated string (dup)", input: "'hello", errorContains: "unterminated string literal"},
		{name: "Unterminated rune literal", input: "r'a", errorContains: "unterminated prefixed literal"},
		{name: "Empty rune literal", input: "r''", errorContains: "must contain exactly one character"},
		{name: "Multi-character rune literal", input: "r'ab'", errorContains: "must contain exactly one character"},
		{name: "Unterminated template literal (dup)", input: "`hello", errorContains: "unterminated template literal"},

		{name: "Unterminated date literal", input: "d'2025-01-01", errorContains: "unterminated prefixed literal"},
		{name: "Unterminated time literal", input: "t'12:00:00", errorContains: "unterminated prefixed literal"},
		{name: "Unterminated datetime literal", input: "dt'2025-01-01T12:00:00Z", errorContains: "unterminated prefixed literal"},
		{name: "Unterminated duration", input: "du'1h", errorContains: "unterminated prefixed literal"},

		{name: "Invalid duration string", input: "du'1year'", errorContains: "Invalid duration format: time: unknown unit \"year\""},
		{name: "Invalid date format", input: "d'2025/01/01'", errorContains: "Invalid date 'YYYY-MM-DD' format (expected '2006-01-02')"},
		{name: "Invalid date value (month)", input: "d'2025-13-01'", errorContains: "Invalid date 'YYYY-MM-DD' format (expected '2006-01-02')"},
		{name: "Invalid time format", input: "t'3 PM'", errorContains: "Invalid time 'HH:mm:ss' format (expected '15:04:05')"},
		{name: "Invalid time value (hour)", input: "t'25:00:00'", errorContains: "Invalid time 'HH:mm:ss' format (expected '15:04:05')"},
		{name: "Invalid datetime format", input: "dt'not-a-date'", errorContains: "Invalid datetime format (expected RFC3339"},
		{name: "Empty duration literal", input: "du''", errorContains: "Invalid duration format: time: invalid duration \"\""},
		{name: "Whitespace duration literal", input: "du'  '", errorContains: "Invalid duration format: time: invalid duration \"  \""},
		{name: "Empty date literal", input: "d''", errorContains: "Invalid date 'YYYY-MM-DD' format (expected '2006-01-02')"},
		{name: "Invalid bigint (float)", input: "1.23n", errorContains: "invalid bigint literal; fractional part not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := NewExpressionParser(context.Background(), tt.input, "test")
			expression, diagnostics := parser.ParseExpression(context.Background())

			if len(diagnostics) > 0 && HasErrors(diagnostics) {
				assert.Nil(t, expression, "Expression should be nil when a parsing error occurs")
			}
			assertHasError(t, diagnostics, tt.errorContains)
		})
	}
}

func TestParser_ObjectLiterals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		validator      func(t *testing.T, expression Expression)
		name           string
		input          string
		expectedString string
	}{
		{
			name:           "Empty object",
			input:          `{}`,
			expectedString: `{}`,
			validator: func(t *testing.T, expression Expression) {
				obj, ok := expression.(*ObjectLiteral)
				require.True(t, ok)
				assert.Empty(t, obj.Pairs)
			},
		},
		{
			name:           "Simple object with bare keys",
			input:          `{ active: true, name: 'test' }`,
			expectedString: `{"active": true, "name": "test"}`,
			validator: func(t *testing.T, expression Expression) {
				obj, ok := expression.(*ObjectLiteral)
				require.True(t, ok)
				require.Len(t, obj.Pairs, 2)
				assert.IsType(t, &BooleanLiteral{}, obj.Pairs["active"])
				assert.IsType(t, &StringLiteral{}, obj.Pairs["name"])
			},
		},
		{
			name:           "Simple object with quoted string keys",
			input:          `{ "is-active": true, 'user-name': "test" }`,
			expectedString: `{"is-active": true, "user-name": "test"}`,
			validator: func(t *testing.T, expression Expression) {
				obj, ok := expression.(*ObjectLiteral)
				require.True(t, ok)
				require.Len(t, obj.Pairs, 2)
				assert.Contains(t, obj.Pairs, "is-active")
				assert.Contains(t, obj.Pairs, "user-name")
			},
		},
		{
			name:           "Object with complex expression values",
			input:          `{ value: 1 + 2, valid: user.isActive && check() }`,
			expectedString: `{"valid": (user.isActive && check()), "value": (1 + 2)}`,
			validator: func(t *testing.T, expression Expression) {
				obj, ok := expression.(*ObjectLiteral)
				require.True(t, ok)
				require.Len(t, obj.Pairs, 2)
				assert.IsType(t, &BinaryExpression{}, obj.Pairs["value"])
				assert.IsType(t, &BinaryExpression{}, obj.Pairs["valid"])
			},
		},
		{
			name:           "Object with trailing comma",
			input:          `{ a: 1, b: 2, }`,
			expectedString: `{"a": 1, "b": 2}`,
			validator: func(t *testing.T, expression Expression) {
				obj, ok := expression.(*ObjectLiteral)
				require.True(t, ok)
				assert.Len(t, obj.Pairs, 2)
			},
		},
		{
			name:           "Nested object literal",
			input:          `{ config: { theme: 'dark', fontSize: 16 } }`,
			expectedString: `{"config": {"fontSize": 16, "theme": "dark"}}`,
			validator: func(t *testing.T, expression Expression) {
				obj, ok := expression.(*ObjectLiteral)
				require.True(t, ok)
				require.Len(t, obj.Pairs, 1)
				nestedObj, ok := obj.Pairs["config"].(*ObjectLiteral)
				require.True(t, ok)
				assert.Len(t, nestedObj.Pairs, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.input)
			assertExprString(t, tt.expectedString, expression)
			if tt.validator != nil {
				tt.validator(t, expression)
			}
		})
	}
}

func TestParser_ArrayLiterals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		validator      func(t *testing.T, expression Expression)
		name           string
		input          string
		expectedString string
	}{
		{
			name:           "Empty array",
			input:          `[]`,
			expectedString: `[]`,
			validator: func(t *testing.T, expression Expression) {
				arr, ok := expression.(*ArrayLiteral)
				require.True(t, ok)
				assert.Empty(t, arr.Elements)
			},
		},
		{
			name:           "Array with single literal",
			input:          `[1]`,
			expectedString: `[1]`,
			validator: func(t *testing.T, expression Expression) {
				arr, ok := expression.(*ArrayLiteral)
				require.True(t, ok)
				require.Len(t, arr.Elements, 1)
				assert.IsType(t, &IntegerLiteral{}, arr.Elements[0])
			},
		},
		{
			name:           "Array with multiple literals",
			input:          `[1, 'two', true, nil]`,
			expectedString: `[1, "two", true, nil]`,
			validator: func(t *testing.T, expression Expression) {
				arr, ok := expression.(*ArrayLiteral)
				require.True(t, ok)
				require.Len(t, arr.Elements, 4)
				assert.IsType(t, &IntegerLiteral{}, arr.Elements[0])
				assert.IsType(t, &StringLiteral{}, arr.Elements[1])
				assert.IsType(t, &BooleanLiteral{}, arr.Elements[2])
				assert.IsType(t, &NilLiteral{}, arr.Elements[3])
			},
		},
		{
			name:           "Array with complex expressions",
			input:          `[1 + 2, user.name, getItems()]`,
			expectedString: `[(1 + 2), user.name, getItems()]`,
			validator: func(t *testing.T, expression Expression) {
				arr, ok := expression.(*ArrayLiteral)
				require.True(t, ok)
				require.Len(t, arr.Elements, 3)
				assert.IsType(t, &BinaryExpression{}, arr.Elements[0])
				assert.IsType(t, &MemberExpression{}, arr.Elements[1])
				assert.IsType(t, &CallExpression{}, arr.Elements[2])
			},
		},
		{
			name:           "Array with trailing comma",
			input:          `[1, 2, ]`,
			expectedString: `[1, 2]`,
			validator: func(t *testing.T, expression Expression) {
				arr, ok := expression.(*ArrayLiteral)
				require.True(t, ok)
				assert.Len(t, arr.Elements, 2)
			},
		},
		{
			name:           "Nested array literal",
			input:          `[[1, 2], [3, 4]]`,
			expectedString: `[[1, 2], [3, 4]]`,
			validator: func(t *testing.T, expression Expression) {
				arr, ok := expression.(*ArrayLiteral)
				require.True(t, ok)
				require.Len(t, arr.Elements, 2)
				assert.IsType(t, &ArrayLiteral{}, arr.Elements[0])
				assert.IsType(t, &ArrayLiteral{}, arr.Elements[1])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.input)
			assertExprString(t, tt.expectedString, expression)
			if tt.validator != nil {
				tt.validator(t, expression)
			}
		})
	}
}

func TestParser_LooseEqualityOperators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		scope          map[string]any
		name           string
		input          string
		expectedString string
		expectedEval   bool
	}{
		{
			name:           "Loose Equal (numbers, same type)",
			input:          "5 ~= 5",
			expectedString: "(5 ~= 5)",
			scope:          nil,
			expectedEval:   true,
		},
		{
			name:           "Loose Equal (numbers, different type - coerced)",
			input:          `5 ~= '5'`,
			expectedString: `(5 ~= "5")`,
			scope:          nil,
			expectedEval:   true,
		},
		{
			name:           "Loose Not Equal (numbers, same type)",
			input:          `5 !~= 6`,
			expectedString: `(5 !~= 6)`,
			scope:          nil,
			expectedEval:   true,
		},
		{
			name:           "Loose Not Equal (numbers, different type - coerced)",
			input:          `5 !~= '6'`,
			expectedString: `(5 !~= "6")`,
			scope:          nil,
			expectedEval:   true,
		},
		{
			name:           "Loose Equal (zero and string zero)",
			input:          `0 ~= '0'`,
			expectedString: `(0 ~= "0")`,
			scope:          nil,
			expectedEval:   true,
		},
		{
			name:           "Loose Not Equal (strings)",
			input:          `'a' !~= 'b'`,
			expectedString: `("a" !~= "b")`,
			scope:          nil,
			expectedEval:   true,
		},
		{
			name:           "Chained with other operators",
			input:          "a > b && c ~= d",
			expectedString: "((a > b) && (c ~= d))",
			scope: map[string]any{
				"a": 10.0,
				"b": 5.0,
				"c": "foo",
				"d": "foo",
			},
			expectedEval: true,
		},
		{
			name:           "Chained with strict equality",
			input:          "a == b && c ~= d",
			expectedString: "((a == b) && (c ~= d))",
			scope: map[string]any{
				"a": 5.0,
				"b": 5.0,
				"c": 5.0,
				"d": "5",
			},
			expectedEval: true,
		},
		{
			name:           "Loose equal with null",
			input:          "a ~= null",
			expectedString: "(a ~= nil)",
			scope: map[string]any{
				"a": nil,
			},
			expectedEval: true,
		},
		{
			name:           "Loose not equal with null",
			input:          "b !~= null",
			expectedString: "(b !~= nil)",
			scope: map[string]any{
				"b": 0.0,
			},
			expectedEval: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.input)
			assertExprString(t, tt.expectedString, expression)

			if strings.Contains(tt.input, "&&") {
				rootExpr, ok := expression.(*BinaryExpression)
				require.True(t, ok)
				require.Equal(t, OpAnd, rootExpr.Operator)

				equalityExpr, ok := rootExpr.Right.(*BinaryExpression)
				require.True(t, ok)

				if strings.Contains(tt.input, "~=") {
					assert.Equal(t, OpLooseEq, equalityExpr.Operator)
				} else if strings.Contains(tt.input, "==") {
					assert.Equal(t, OpEq, equalityExpr.Operator)
				}
			} else {
				binExpr, ok := expression.(*BinaryExpression)
				require.True(t, ok)
				if strings.Contains(tt.input, "!~=") {
					assert.Equal(t, OpLooseNe, binExpr.Operator)
				} else if strings.Contains(tt.input, "~=") {
					assert.Equal(t, OpLooseEq, binExpr.Operator)
				}
			}

			result := EvaluateExpression(expression, tt.scope)
			assert.Equal(t, tt.expectedEval, result, "Evaluation result was incorrect")
		})
	}
}

func TestParser_TruthyOperator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		scope          map[string]any
		name           string
		input          string
		expectedString string
		expectedEval   bool
	}{
		{
			name:           "Truthy - non-empty string",
			input:          `~"hello"`,
			expectedString: `~"hello"`,
			scope:          nil,
			expectedEval:   true,
		},
		{
			name:           "Truthy - empty string",
			input:          `~""`,
			expectedString: `~""`,
			scope:          nil,
			expectedEval:   false,
		},
		{
			name:           "Truthy - non-zero number",
			input:          `~42`,
			expectedString: `~42`,
			scope:          nil,
			expectedEval:   true,
		},
		{
			name:           "Truthy - zero",
			input:          `~0`,
			expectedString: `~0`,
			scope:          nil,
			expectedEval:   false,
		},
		{
			name:           "Truthy - true",
			input:          `~true`,
			expectedString: `~true`,
			scope:          nil,
			expectedEval:   true,
		},
		{
			name:           "Truthy - false",
			input:          `~false`,
			expectedString: `~false`,
			scope:          nil,
			expectedEval:   false,
		},
		{
			name:           "Truthy - variable",
			input:          `~value`,
			expectedString: `~value`,
			scope: map[string]any{
				"value": "non-empty",
			},
			expectedEval: true,
		},
		{
			name:           "Truthy - nil variable",
			input:          `~value`,
			expectedString: `~value`,
			scope: map[string]any{
				"value": nil,
			},
			expectedEval: false,
		},
		{
			name:           "Truthy in condition",
			input:          `~value && active`,
			expectedString: `(~value && active)`,
			scope: map[string]any{
				"value":  "something",
				"active": true,
			},
			expectedEval: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.input)
			assertExprString(t, tt.expectedString, expression)

			if !strings.Contains(tt.input, "&&") {
				unaryExpr, ok := expression.(*UnaryExpression)
				require.True(t, ok)
				assert.Equal(t, OpTruthy, unaryExpr.Operator)
			}

			result := EvaluateExpression(expression, tt.scope)
			assert.Equal(t, tt.expectedEval, result, "Evaluation result was incorrect")
		})
	}
}

func TestParser_UnicodeIdentifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expected       any
		scope          map[string]any
		name           string
		input          string
		expectedString string
	}{
		{
			name:           "Arabic identifier",
			input:          "مستخدم.اسم",
			expectedString: "مستخدم.اسم",
			scope: map[string]any{
				"مستخدم": map[string]any{"اسم": "علي"},
			},
			expected: "علي",
		},
		{
			name:           "Japanese identifier in binary expression",
			input:          "価格 > 1000",
			expectedString: "(価格 > 1000)",
			scope: map[string]any{
				"価格": 1500.0,
			},
			expected: true,
		},
		{
			name:           "Cyrillic identifier in function call",
			input:          "получитьПользователя(42)",
			expectedString: "получитьПользователя(42)",
			scope: map[string]any{
				"получитьПользователя": func(id any) any {
					idFloat, ok := id.(float64)
					if ok && idFloat == 42.0 {
						return "Иван"
					}
					return nil
				},
			},
			expected: "Иван",
		},
		{
			name:           "Mixed script identifiers",
			input:          "user.ชื่อ + ' ' + user.lastName",
			expectedString: `(user.ชื่อ + (" " + user.lastName))`,
			scope: map[string]any{
				"user": map[string]any{
					"ชื่อ":     "สมชาย",
					"lastName": "Smith",
				},
			},
			expected: "สมชาย Smith",
		},
		{
			name:           "Identifier starting with underscore and unicode",
			input:          "_你好",
			expectedString: "_你好",
			scope: map[string]any{
				"_你好": "World",
			},
			expected: "World",
		},
		{
			name:           "Zalgo/Cthulhu text identifier",
			input:          "ţ̶̛͇͙̩͍̤̙̬̈́̆̉̈́͊ͅé̸̡̪̩̰̮̞̬̜̖̤͔̤̺͍̰͍̏̿̎̑͛̅̈͒̀̏̈́̔͌͌̽́̉̈́͘͠ͅs̶̨̭̻̺͙̠̫͎̘̳̔͗̑̚͠ṯ̸͙̠̰̥͉̖͕̼̀̾̎͊͐͊̾̈͐̀̽̉̍̌͌͌͂̃̓ͅͅ.value",
			expectedString: "ţ̶̛͇͙̩͍̤̙̬̈́̆̉̈́͊ͅé̸̡̪̩̰̮̞̬̜̖̤͔̤̺͍̰͍̏̿̎̑͛̅̈͒̀̏̈́̔͌͌̽́̉̈́͘͠ͅs̶̨̭̻̺͙̠̫͎̘̳̔͗̑̚͠ṯ̸͙̠̰̥͉̖͕̼̀̾̎͊͐͊̾̈͐̀̽̉̍̌͌͌͂̃̓ͅͅ.value",
			scope: map[string]any{
				"ţ̶̛͇͙̩͍̤̙̬̈́̆̉̈́͊ͅé̸̡̪̩̰̮̞̬̜̖̤͔̤̺͍̰͍̏̿̎̑͛̅̈͒̀̏̈́̔͌͌̽́̉̈́͘͠ͅs̶̨̭̻̺͙̠̫͎̘̳̔͗̑̚͠ṯ̸͙̠̰̥͉̖͕̼̀̾̎͊͐͊̾̈͐̀̽̉̍̌͌͌͂̃̓ͅͅ": map[string]any{"value": "it works!"},
			},
			expected: "it works!",
		},
		{
			name:           "Full-width vs Half-width identifiers",
			input:          "ユーザー",
			expectedString: "ユーザー",
			scope: map[string]any{
				"ユーザー":  "Correct Katakana User",
				"ﾕｰｻﾞｰ": "Incorrect Half-width Katakana",
				"user":  "Incorrect ASCII",
			},
			expected: "Correct Katakana User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tt.input)
			assertExprString(t, tt.expectedString, expression)

			result := EvaluateExpression(expression, tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParser_IncompleteExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		expectedExpr string
		diagContains string
	}{
		{
			name:         "Incomplete member access - EOF after dot",
			input:        "state.User.",
			expectedExpr: "state.User",
			diagContains: "Expected identifier after '.' or '?.'",
		},
		{
			name:         "Incomplete member access - EOF after optional dot",
			input:        "state.User?.",
			expectedExpr: "state.User",
			diagContains: "Expected identifier after '.' or '?.'",
		},
		{
			name:         "Simple identifier followed by dot and EOF",
			input:        "user.",
			expectedExpr: "user",
			diagContains: "Expected identifier after '.' or '?.'",
		},
		{
			name:         "Chained member access incomplete at end",
			input:        "app.config.settings.",
			expectedExpr: "app.config.settings",
			diagContains: "Expected identifier after '.' or '?.'",
		},
		{
			name:         "Invalid token after dot",
			input:        "state.User.123",
			expectedExpr: "state.User",
			diagContains: "Expected identifier after '.' or '?.'",
		},
		{
			name:         "Invalid token after optional dot",
			input:        "state.User?.@invalid",
			expectedExpr: "state.User",
			diagContains: "Expected identifier after '.' or '?.'",
		},
		{
			name:         "Complex expression with incomplete member at end",
			input:        "(getUser() || defaultUser).profile.",
			expectedExpr: "(getUser() || defaultUser).profile",
			diagContains: "Expected identifier after '.' or '?.'",
		},
		{
			name:         "Array index followed by incomplete member",
			input:        "users[0].",
			expectedExpr: "users[0]",
			diagContains: "Expected identifier after '.' or '?.'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := NewExpressionParser(context.Background(), tt.input, "test")
			expression, diagnostics := parser.ParseExpression(context.Background())

			require.NotNil(t, expression, "Parser should return base expression for incomplete input, not nil")

			assertExprString(t, tt.expectedExpr, expression)

			require.NotEmpty(t, diagnostics, "Expected diagnostic for incomplete expression")

			assertHasError(t, diagnostics, tt.diagContains)

			assert.NotEmpty(t, diagnostics, "Expected at least one diagnostic for incomplete expression")
		})
	}
}

func TestParser_IncompleteExpressions_TemplateContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		template     string
		diagContains string
	}{
		{
			name:         "Incomplete expression in mustache",
			template:     `<div>{{ state.User. }}</div>`,
			diagContains: "Expected identifier after '.' or '?.'",
		},
		{
			name:         "Incomplete expression in dynamic attribute",
			template:     `<div :class="state.theme.">Content</div>`,
			diagContains: "Expected identifier after '.' or '?.'",
		},
		{
			name:         "Incomplete optional chaining",
			template:     `<span>{{ user?.profile?. }}</span>`,
			diagContains: "Expected identifier after '.' or '?.'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tree, err := ParseAndTransform(context.Background(), tt.template, "test")

			require.NoError(t, err, "ParseAndTransform should not fail on incomplete expressions")
			require.NotNil(t, tree, "Tree should not be nil")

			require.NotEmpty(t, tree.Diagnostics, "Expected diagnostics for incomplete expression")

			assertHasError(t, tree.Diagnostics, tt.diagContains)
		})
	}
}

func TestParser_IncompleteFunctionCalls(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		expectedExpr string
		diagContains string
	}{
		{
			name:         "Function call with just opening paren",
			input:        "myFunc(",
			expectedExpr: "myFunc",
			diagContains: "Incomplete function call",
		},
		{
			name:         "Function call with one argument incomplete",
			input:        "calculate(a + ",
			expectedExpr: "calculate",
			diagContains: "Expected expression on the right side",
		},
		{
			name:         "Function call with comma but no next argument",
			input:        "myFunc(arg1, ",
			expectedExpr: "myFunc",
			diagContains: "Incomplete function call: expected argument after ','",
		},
		{
			name:         "Method call incomplete",
			input:        "user.getName(",
			expectedExpr: "getName",
			diagContains: "Incomplete function call",
		},
		{
			name:         "Chained call incomplete",
			input:        "getUser().getProfile(",
			expectedExpr: "getProfile",
			diagContains: "Incomplete function call",
		},
		{
			name:         "Function call missing closing paren",
			input:        "myFunc(arg1, arg2",
			expectedExpr: "myFunc",
			diagContains: "Expected ',' or ')'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := NewExpressionParser(context.Background(), tt.input, "test")
			expression, diagnostics := parser.ParseExpression(context.Background())

			require.NotNil(t, expression, "Parser should return partial expression for incomplete function call")

			require.NotEmpty(t, diagnostics, "Expected diagnostic for incomplete function call")
			assertHasError(t, diagnostics, tt.diagContains)

			expressionString := expression.String()
			assert.Contains(t, expressionString, tt.expectedExpr, "Expression should preserve the callee identifier")
		})
	}
}

func TestParser_IncompleteIndexExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		expectedExpr string
		diagContains string
	}{
		{
			name:         "Index with just opening bracket",
			input:        "items[",
			expectedExpr: "items",
			diagContains: "Incomplete index expression",
		},
		{
			name:         "Index with empty brackets",
			input:        "items[]",
			expectedExpr: "items",
			diagContains: "Empty index expression",
		},
		{
			name:         "Index missing closing bracket",
			input:        "items[0",
			expectedExpr: "items",
			diagContains: "Expected ']'",
		},
		{
			name:         "Index with incomplete expression",
			input:        "items[key.",
			expectedExpr: "items",
			diagContains: "Expected identifier after '.'",
		},
		{
			name:         "Chained index incomplete",
			input:        "matrix[0][",
			expectedExpr: "matrix[0]",
			diagContains: "Incomplete index expression",
		},
		{
			name:         "Optional index incomplete",
			input:        "data?.[",
			expectedExpr: "data",
			diagContains: "Incomplete index expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := NewExpressionParser(context.Background(), tt.input, "test")
			expression, diagnostics := parser.ParseExpression(context.Background())

			require.NotNil(t, expression, "Parser should return base expression for incomplete index")

			require.NotEmpty(t, diagnostics, "Expected diagnostic for incomplete index expression")
			assertHasError(t, diagnostics, tt.diagContains)

			expressionString := expression.String()
			assert.Contains(t, expressionString, tt.expectedExpr, "Expression should preserve the base")
		})
	}
}

func TestParser_IncompleteBinaryOperators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		expectedExpr string
		diagContains string
	}{
		{
			name:         "Addition with no right operand",
			input:        "a + ",
			expectedExpr: "a",
			diagContains: "Expected expression on the right side",
		},
		{
			name:         "Multiplication with no right operand",
			input:        "value * ",
			expectedExpr: "value",
			diagContains: "Expected expression on the right side",
		},
		{
			name:         "Comparison with no right operand",
			input:        "count > ",
			expectedExpr: "count",
			diagContains: "Expected expression on the right side",
		},
		{
			name:         "Logical AND with no right operand",
			input:        "isActive && ",
			expectedExpr: "isActive",
			diagContains: "Expected expression on the right side",
		},
		{
			name:         "Complex left expression with dangling operator",
			input:        "user.age + ",
			expectedExpr: "user.age",
			diagContains: "Expected expression on the right side",
		},
		{
			name:         "Right operand incomplete",
			input:        "a + b.",
			expectedExpr: "a",
			diagContains: "Expected identifier after '.'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := NewExpressionParser(context.Background(), tt.input, "test")
			expression, diagnostics := parser.ParseExpression(context.Background())

			require.NotNil(t, expression, "Parser should return left expression for incomplete binary operator")

			require.NotEmpty(t, diagnostics, "Expected diagnostic for incomplete binary operator")
			assertHasError(t, diagnostics, tt.diagContains)

			expressionString := expression.String()
			assert.Contains(t, expressionString, tt.expectedExpr, "Expression should preserve the left operand")
		})
	}
}

func TestParser_IncompleteTernaryExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		expectedExpr string
		diagContains string
	}{
		{
			name:         "Ternary with just question mark",
			input:        "condition ? ",
			expectedExpr: "condition",
			diagContains: "Incomplete ternary expression",
		},
		{
			name:         "Ternary missing colon",
			input:        "isActive ? trueValue",
			expectedExpr: "isActive",
			diagContains: "expected ':' after consequent",
		},
		{
			name:         "Ternary with colon but no alternate",
			input:        "count > 0 ? yes : ",
			expectedExpr: "count",
			diagContains: "Incomplete ternary expression",
		},
		{
			name:         "Ternary with incomplete consequent",
			input:        "active ? value.",
			expectedExpr: "active",
			diagContains: "Expected identifier after '.'",
		},
		{
			name:         "Ternary with incomplete alternate",
			input:        "test ? a : b.",
			expectedExpr: "test",
			diagContains: "Expected identifier after '.'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parser := NewExpressionParser(context.Background(), tt.input, "test")
			expression, diagnostics := parser.ParseExpression(context.Background())

			require.NotNil(t, expression, "Parser should return condition for incomplete ternary")

			require.NotEmpty(t, diagnostics, "Expected diagnostic for incomplete ternary")
			assertHasError(t, diagnostics, tt.diagContains)

			expressionString := expression.String()
			assert.Contains(t, expressionString, tt.expectedExpr, "Expression should preserve the condition")
		})
	}
}

func TestParser_ParenthesisedExpressionLocations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		checkLocFunc func(t *testing.T, expression Expression)
		name         string
		input        string
		expectType   string
	}{
		{
			name:       "parenthesised binary expression",
			input:      "(a + b)",
			expectType: "*ast_domain.BinaryExpression",
			checkLocFunc: func(t *testing.T, expression Expression) {
				be, ok := expression.(*BinaryExpression)
				require.True(t, ok)

				assert.Equal(t, 0, be.RelativeLocation.Offset)

				assert.Equal(t, 7, be.SourceLength)
			},
		},
		{
			name:       "parenthesised unary expression",
			input:      "(-x)",
			expectType: "*ast_domain.UnaryExpression",
			checkLocFunc: func(t *testing.T, expression Expression) {
				ue, ok := expression.(*UnaryExpression)
				require.True(t, ok)
				assert.Equal(t, 0, ue.RelativeLocation.Offset)
				assert.Equal(t, 4, ue.SourceLength)
			},
		},
		{
			name:       "parenthesised member expression",
			input:      "(a.b)",
			expectType: "*ast_domain.MemberExpression",
			checkLocFunc: func(t *testing.T, expression Expression) {
				me, ok := expression.(*MemberExpression)
				require.True(t, ok)
				assert.Equal(t, 0, me.RelativeLocation.Offset)
				assert.Equal(t, 5, me.SourceLength)
			},
		},
		{
			name:       "parenthesised index expression",
			input:      "(arr[0])",
			expectType: "*ast_domain.IndexExpression",
			checkLocFunc: func(t *testing.T, expression Expression) {
				ie, ok := expression.(*IndexExpression)
				require.True(t, ok)
				assert.Equal(t, 0, ie.RelativeLocation.Offset)
				assert.Equal(t, 8, ie.SourceLength)
			},
		},
		{
			name:       "parenthesised call expression",
			input:      "(fn())",
			expectType: "*ast_domain.CallExpression",
			checkLocFunc: func(t *testing.T, expression Expression) {
				ce, ok := expression.(*CallExpression)
				require.True(t, ok)
				assert.Equal(t, 0, ce.RelativeLocation.Offset)
				assert.Equal(t, 6, ce.SourceLength)
			},
		},
		{
			name:       "parenthesised ternary expression",
			input:      "(a ? b : c)",
			expectType: "*ast_domain.TernaryExpression",
			checkLocFunc: func(t *testing.T, expression Expression) {
				te, ok := expression.(*TernaryExpression)
				require.True(t, ok)
				assert.Equal(t, 0, te.RelativeLocation.Offset)
				assert.Equal(t, 11, te.SourceLength)
			},
		},
		{
			name:       "parenthesised identifier",
			input:      "(x)",
			expectType: "*ast_domain.Identifier",
			checkLocFunc: func(t *testing.T, expression Expression) {
				id, ok := expression.(*Identifier)
				require.True(t, ok)
				assert.Equal(t, 0, id.RelativeLocation.Offset)
				assert.Equal(t, 3, id.SourceLength)
			},
		},
		{
			name:       "parenthesised integer literal",
			input:      "(123)",
			expectType: "*ast_domain.IntegerLiteral",
			checkLocFunc: func(t *testing.T, expression Expression) {
				il, ok := expression.(*IntegerLiteral)
				require.True(t, ok)
				assert.Equal(t, 0, il.RelativeLocation.Offset)
				assert.Equal(t, 5, il.SourceLength)
			},
		},
		{
			name:       "parenthesised float literal",
			input:      "(1.5)",
			expectType: "*ast_domain.FloatLiteral",
			checkLocFunc: func(t *testing.T, expression Expression) {
				fl, ok := expression.(*FloatLiteral)
				require.True(t, ok)
				assert.Equal(t, 0, fl.RelativeLocation.Offset)
				assert.Equal(t, 5, fl.SourceLength)
			},
		},
		{
			name:       "parenthesised string literal",
			input:      `("hello")`,
			expectType: "*ast_domain.StringLiteral",
			checkLocFunc: func(t *testing.T, expression Expression) {
				sl, ok := expression.(*StringLiteral)
				require.True(t, ok)
				assert.Equal(t, 0, sl.RelativeLocation.Offset)
				assert.Equal(t, 9, sl.SourceLength)
			},
		},
		{
			name:       "parenthesised boolean literal",
			input:      "(true)",
			expectType: "*ast_domain.BooleanLiteral",
			checkLocFunc: func(t *testing.T, expression Expression) {
				bl, ok := expression.(*BooleanLiteral)
				require.True(t, ok)
				assert.Equal(t, 0, bl.RelativeLocation.Offset)
				assert.Equal(t, 6, bl.SourceLength)
			},
		},
		{
			name:       "parenthesised nil literal",
			input:      "(nil)",
			expectType: "*ast_domain.NilLiteral",
			checkLocFunc: func(t *testing.T, expression Expression) {
				nl, ok := expression.(*NilLiteral)
				require.True(t, ok)
				assert.Equal(t, 0, nl.RelativeLocation.Offset)
				assert.Equal(t, 5, nl.SourceLength)
			},
		},
		{
			name:       "parenthesised decimal literal",
			input:      "(1.5d)",
			expectType: "*ast_domain.DecimalLiteral",
			checkLocFunc: func(t *testing.T, expression Expression) {
				dl, ok := expression.(*DecimalLiteral)
				require.True(t, ok)
				assert.Equal(t, 0, dl.RelativeLocation.Offset)
				assert.Equal(t, 6, dl.SourceLength)
			},
		},
		{
			name:       "parenthesised bigint literal",
			input:      "(123n)",
			expectType: "*ast_domain.BigIntLiteral",
			checkLocFunc: func(t *testing.T, expression Expression) {
				bi, ok := expression.(*BigIntLiteral)
				require.True(t, ok)
				assert.Equal(t, 0, bi.RelativeLocation.Offset)
				assert.Equal(t, 6, bi.SourceLength)
			},
		},
		{
			name:       "parenthesised date literal",
			input:      "(d'2025-01-01')",
			expectType: "*ast_domain.DateLiteral",
			checkLocFunc: func(t *testing.T, expression Expression) {
				dl, ok := expression.(*DateLiteral)
				require.True(t, ok)
				assert.Equal(t, 0, dl.RelativeLocation.Offset)
				assert.Equal(t, 15, dl.SourceLength)
			},
		},
		{
			name:       "parenthesised array literal",
			input:      "([1, 2])",
			expectType: "*ast_domain.ArrayLiteral",
			checkLocFunc: func(t *testing.T, expression Expression) {
				al, ok := expression.(*ArrayLiteral)
				require.True(t, ok)
				assert.Equal(t, 0, al.RelativeLocation.Offset)
				assert.Equal(t, 8, al.SourceLength)
			},
		},
		{
			name:       "parenthesised object literal",
			input:      "({a: 1})",
			expectType: "*ast_domain.ObjectLiteral",
			checkLocFunc: func(t *testing.T, expression Expression) {
				ol, ok := expression.(*ObjectLiteral)
				require.True(t, ok)
				assert.Equal(t, 0, ol.RelativeLocation.Offset)
				assert.Equal(t, 8, ol.SourceLength)
			},
		},
		{
			name:       "nested parentheses",
			input:      "((x))",
			expectType: "*ast_domain.Identifier",
			checkLocFunc: func(t *testing.T, expression Expression) {
				id, ok := expression.(*Identifier)
				require.True(t, ok)
				assert.Equal(t, 0, id.RelativeLocation.Offset)
				assert.Equal(t, 5, id.SourceLength)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser := NewExpressionParser(context.Background(), tc.input, "test")
			expression, diagnostics := parser.ParseExpression(context.Background())

			require.Empty(t, diagnostics, "Expected no diagnostics")
			require.NotNil(t, expression)
			tc.checkLocFunc(t, expression)
		})
	}
}

func TestParser_ObjectLiteralEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("missing colon after key", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "{ key value }", "test")
		_, diagnostics := parser.ParseExpression(context.Background())

		require.NotEmpty(t, diagnostics)
		assertHasError(t, diagnostics, "Expected ':'")
	})

	t.Run("invalid key type - number", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "{ 123: value }", "test")
		_, diagnostics := parser.ParseExpression(context.Background())

		require.NotEmpty(t, diagnostics)
		assertHasError(t, diagnostics, "Expected an identifier or string")
	})

	t.Run("object with single quoted string key", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "{ 'key': 42 }", "test")
		expression, diagnostics := parser.ParseExpression(context.Background())

		require.Empty(t, diagnostics)
		obj, ok := expression.(*ObjectLiteral)
		require.True(t, ok)
		_, exists := obj.Pairs["key"]
		assert.True(t, exists)
	})

	t.Run("object with double quoted string key", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), `{ "key": 42 }`, "test")
		expression, diagnostics := parser.ParseExpression(context.Background())

		require.Empty(t, diagnostics)
		obj, ok := expression.(*ObjectLiteral)
		require.True(t, ok)
		_, exists := obj.Pairs["key"]
		assert.True(t, exists)
	})

	t.Run("unclosed object brace", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "{ a: 1", "test")
		_, diagnostics := parser.ParseExpression(context.Background())

		require.NotEmpty(t, diagnostics)
		assertHasError(t, diagnostics, "Expected ',' or '}'")
	})

	t.Run("empty value expression after colon", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "{ a: }", "test")
		_, diagnostics := parser.ParseExpression(context.Background())

		require.NotEmpty(t, diagnostics)

	})

	t.Run("object with expression values", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "{ sum: a + b, product: x * y }", "test")
		expression, diagnostics := parser.ParseExpression(context.Background())

		require.Empty(t, diagnostics)
		obj, ok := expression.(*ObjectLiteral)
		require.True(t, ok)
		assert.Len(t, obj.Pairs, 2)

		sum, ok := obj.Pairs["sum"].(*BinaryExpression)
		require.True(t, ok)
		assert.Equal(t, OpPlus, sum.Operator)

		product, ok := obj.Pairs["product"].(*BinaryExpression)
		require.True(t, ok)
		assert.Equal(t, OpMul, product.Operator)
	})

	t.Run("duplicate keys overwrites", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "{ a: 1, a: 2 }", "test")
		expression, diagnostics := parser.ParseExpression(context.Background())

		require.Empty(t, diagnostics)
		obj, ok := expression.(*ObjectLiteral)
		require.True(t, ok)

		value, ok := obj.Pairs["a"].(*IntegerLiteral)
		require.True(t, ok)
		assert.Equal(t, int64(2), value.Value)
	})
}

func TestParser_ArrayLiteralEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty array", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "[]", "test")
		expression, diagnostics := parser.ParseExpression(context.Background())

		require.Empty(t, diagnostics)
		arr, ok := expression.(*ArrayLiteral)
		require.True(t, ok)
		assert.Empty(t, arr.Elements)
	})

	t.Run("array with trailing comma", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "[1, 2, 3,]", "test")
		expression, diagnostics := parser.ParseExpression(context.Background())

		require.Empty(t, diagnostics)
		arr, ok := expression.(*ArrayLiteral)
		require.True(t, ok)
		assert.Len(t, arr.Elements, 3)
	})

	t.Run("unclosed array bracket", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "[1, 2", "test")
		_, diagnostics := parser.ParseExpression(context.Background())

		require.NotEmpty(t, diagnostics)
		assertHasError(t, diagnostics, "Expected ']'")
	})

	t.Run("nested arrays", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "[[1, 2], [3, 4]]", "test")
		expression, diagnostics := parser.ParseExpression(context.Background())

		require.Empty(t, diagnostics)
		arr, ok := expression.(*ArrayLiteral)
		require.True(t, ok)
		assert.Len(t, arr.Elements, 2)

		inner1, ok := arr.Elements[0].(*ArrayLiteral)
		require.True(t, ok)
		assert.Len(t, inner1.Elements, 2)
	})

	t.Run("array with mixed expression types", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "[1, 'hello', true, nil, a + b]", "test")
		expression, diagnostics := parser.ParseExpression(context.Background())

		require.Empty(t, diagnostics)
		arr, ok := expression.(*ArrayLiteral)
		require.True(t, ok)
		assert.Len(t, arr.Elements, 5)

		_, ok = arr.Elements[0].(*IntegerLiteral)
		assert.True(t, ok)
		_, ok = arr.Elements[1].(*StringLiteral)
		assert.True(t, ok)
		_, ok = arr.Elements[2].(*BooleanLiteral)
		assert.True(t, ok)
		_, ok = arr.Elements[3].(*NilLiteral)
		assert.True(t, ok)
		_, ok = arr.Elements[4].(*BinaryExpression)
		assert.True(t, ok)
	})
}

func TestParseExpressionCached(t *testing.T) {
	t.Cleanup(ClearExpressionCache)

	t.Run("empty string returns nil", func(t *testing.T) {
		ClearExpressionCache()
		expression, diagnostics := ParseExpressionCached(context.Background(), "", "test")
		assert.Nil(t, expression)
		assert.Nil(t, diagnostics)
	})

	t.Run("parses and caches expression", func(t *testing.T) {
		ClearExpressionCache()

		expression1, diags1 := ParseExpressionCached(context.Background(), "1 + 2", "test")
		require.NotNil(t, expression1)
		assertNoError(t, diags1, "1 + 2")

		expression2, diags2 := ParseExpressionCached(context.Background(), "1 + 2", "test")
		require.NotNil(t, expression2)
		assertNoError(t, diags2, "1 + 2")

		assert.Equal(t, expression1.String(), expression2.String())
		assert.NotSame(t, expression1, expression2, "Cached result should be a clone, not the same pointer")
	})

	t.Run("different expressions get different results", func(t *testing.T) {
		ClearExpressionCache()

		expressionA, diagsA := ParseExpressionCached(context.Background(), "1 + 2", "test")
		require.NotNil(t, expressionA)
		assertNoError(t, diagsA, "1 + 2")

		expressionB, diagsB := ParseExpressionCached(context.Background(), "3 * 4", "test")
		require.NotNil(t, expressionB)
		assertNoError(t, diagsB, "3 * 4")

		assert.NotEqual(t, expressionA.String(), expressionB.String())
	})

	t.Run("ClearExpressionCache resets cache", func(t *testing.T) {
		ClearExpressionCache()

		expression1, _ := ParseExpressionCached(context.Background(), "a + b", "test")
		require.NotNil(t, expression1)

		ClearExpressionCache()

		expression2, _ := ParseExpressionCached(context.Background(), "a + b", "test")
		require.NotNil(t, expression2)
		assert.Equal(t, expression1.String(), expression2.String())
	})

	t.Run("caches expressions with diagnostics", func(t *testing.T) {
		ClearExpressionCache()

		expression1, diags1 := ParseExpressionCached(context.Background(), "1 +", "test")

		_ = expression1

		expression2, diags2 := ParseExpressionCached(context.Background(), "1 +", "test")
		_ = expression2

		assert.Equal(t, len(diags1), len(diags2))
	})

	t.Run("identifier expression caching", func(t *testing.T) {
		ClearExpressionCache()

		expression1, diags1 := ParseExpressionCached(context.Background(), "user.name", "test")
		require.NotNil(t, expression1)
		assertNoError(t, diags1, "user.name")

		expression2, diags2 := ParseExpressionCached(context.Background(), "user.name", "test")
		require.NotNil(t, expression2)
		assertNoError(t, diags2, "user.name")

		assert.Equal(t, expression1.String(), expression2.String())
		assert.NotSame(t, expression1, expression2)
	})
}
