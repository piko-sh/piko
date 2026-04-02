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

package ast_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_adapters"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestSourceLength_AllExpressionTypes(t *testing.T) {
	testCases := []struct {
		expectedType      any
		additionalChecks  func(t *testing.T, expression ast_domain.Expression)
		name              string
		input             string
		expectedSourceLen int
	}{

		{
			name:              "Integer literal",
			input:             "12345",
			expectedType:      (*ast_domain.IntegerLiteral)(nil),
			expectedSourceLen: 5,
		},
		{
			name:              "Float literal",
			input:             "123.456",
			expectedType:      (*ast_domain.FloatLiteral)(nil),
			expectedSourceLen: 7,
		},
		{
			name:              "String literal",
			input:             "'hello world'",
			expectedType:      (*ast_domain.StringLiteral)(nil),
			expectedSourceLen: 13,
		},
		{
			name:              "Boolean literal true",
			input:             "true",
			expectedType:      (*ast_domain.BooleanLiteral)(nil),
			expectedSourceLen: 4,
		},
		{
			name:              "Boolean literal false",
			input:             "false",
			expectedType:      (*ast_domain.BooleanLiteral)(nil),
			expectedSourceLen: 5,
		},
		{
			name:              "Nil literal",
			input:             "nil",
			expectedType:      (*ast_domain.NilLiteral)(nil),
			expectedSourceLen: 3,
		},
		{
			name:              "Decimal literal",
			input:             "123.45d",
			expectedType:      (*ast_domain.DecimalLiteral)(nil),
			expectedSourceLen: 7,
		},
		{
			name:              "BigInt literal",
			input:             "123456789n",
			expectedType:      (*ast_domain.BigIntLiteral)(nil),
			expectedSourceLen: 10,
		},
		{
			name:              "Rune literal",
			input:             "r'a'",
			expectedType:      (*ast_domain.RuneLiteral)(nil),
			expectedSourceLen: 4,
		},
		{
			name:              "DateTime literal",
			input:             "dt'2025-01-01T00:00:00Z'",
			expectedType:      (*ast_domain.DateTimeLiteral)(nil),
			expectedSourceLen: 24,
		},
		{
			name:              "Date literal",
			input:             "d'2025-01-01'",
			expectedType:      (*ast_domain.DateLiteral)(nil),
			expectedSourceLen: 13,
		},
		{
			name:              "Time literal",
			input:             "t'14:30:00'",
			expectedType:      (*ast_domain.TimeLiteral)(nil),
			expectedSourceLen: 11,
		},
		{
			name:              "Duration literal",
			input:             "du'1h30m'",
			expectedType:      (*ast_domain.DurationLiteral)(nil),
			expectedSourceLen: 9,
		},
		{
			name:              "Identifier",
			input:             "userName",
			expectedType:      (*ast_domain.Identifier)(nil),
			expectedSourceLen: 8,
		},

		{
			name:              "Simple addition",
			input:             "1 + 2",
			expectedType:      (*ast_domain.BinaryExpression)(nil),
			expectedSourceLen: 5,
		},
		{
			name:              "Addition with spaces",
			input:             "10  +  20",
			expectedType:      (*ast_domain.BinaryExpression)(nil),
			expectedSourceLen: 9,
		},
		{
			name:              "Multiplication",
			input:             "3 * 4",
			expectedType:      (*ast_domain.BinaryExpression)(nil),
			expectedSourceLen: 5,
		},
		{
			name:              "Logical AND",
			input:             "true && false",
			expectedType:      (*ast_domain.BinaryExpression)(nil),
			expectedSourceLen: 13,
		},
		{
			name:              "Comparison",
			input:             "x > 10",
			expectedType:      (*ast_domain.BinaryExpression)(nil),
			expectedSourceLen: 6,
		},
		{
			name:              "Nested binary expression",
			input:             "1 + 2 * 3",
			expectedType:      (*ast_domain.BinaryExpression)(nil),
			expectedSourceLen: 9,
		},

		{
			name:              "Logical NOT",
			input:             "!true",
			expectedType:      (*ast_domain.UnaryExpression)(nil),
			expectedSourceLen: 5,
		},
		{
			name:              "Negation",
			input:             "-42",
			expectedType:      (*ast_domain.UnaryExpression)(nil),
			expectedSourceLen: 3,
		},
		{
			name:              "Unary NOT with complex expression",
			input:             "!(x > 10)",
			expectedType:      (*ast_domain.UnaryExpression)(nil),
			expectedSourceLen: 9,
		},

		{
			name:              "Simple member access",
			input:             "user.name",
			expectedType:      (*ast_domain.MemberExpression)(nil),
			expectedSourceLen: 9,
		},
		{
			name:              "Chained member access",
			input:             "user.profile.avatar",
			expectedType:      (*ast_domain.MemberExpression)(nil),
			expectedSourceLen: 19,
		},
		{
			name:              "Optional member access",
			input:             "user?.name",
			expectedType:      (*ast_domain.MemberExpression)(nil),
			expectedSourceLen: 10,
		},

		{
			name:              "Array index",
			input:             "items[0]",
			expectedType:      (*ast_domain.IndexExpression)(nil),
			expectedSourceLen: 8,
		},
		{
			name:              "Map access",
			input:             "config['key']",
			expectedType:      (*ast_domain.IndexExpression)(nil),
			expectedSourceLen: 13,
		},
		{
			name:              "Optional index",
			input:             "items?.[0]",
			expectedType:      (*ast_domain.IndexExpression)(nil),
			expectedSourceLen: 10,
		},

		{
			name:              "Function call no arguments",
			input:             "getUser()",
			expectedType:      (*ast_domain.CallExpression)(nil),
			expectedSourceLen: 9,
		},
		{
			name:              "Function call one argument",
			input:             "getUser(123)",
			expectedType:      (*ast_domain.CallExpression)(nil),
			expectedSourceLen: 12,
		},
		{
			name:              "Function call multiple arguments",
			input:             "add(1, 2, 3)",
			expectedType:      (*ast_domain.CallExpression)(nil),
			expectedSourceLen: 12,
		},
		{
			name:              "Method call",
			input:             "user.getName()",
			expectedType:      (*ast_domain.CallExpression)(nil),
			expectedSourceLen: 14,
		},
		{
			name:              "Chained call",
			input:             "getUser().getName()",
			expectedType:      (*ast_domain.CallExpression)(nil),
			expectedSourceLen: 19,
		},

		{
			name:              "Simple ternary",
			input:             "x > 0 ? 'pos' : 'neg'",
			expectedType:      (*ast_domain.TernaryExpression)(nil),
			expectedSourceLen: 21,
		},
		{
			name:              "Nested ternary",
			input:             "a ? b : c ? d : e",
			expectedType:      (*ast_domain.TernaryExpression)(nil),
			expectedSourceLen: 17,
		},

		{
			name:              "Empty object",
			input:             "{}",
			expectedType:      (*ast_domain.ObjectLiteral)(nil),
			expectedSourceLen: 2,
		},
		{
			name:              "Object with one property",
			input:             "{name: 'John'}",
			expectedType:      (*ast_domain.ObjectLiteral)(nil),
			expectedSourceLen: 14,
		},
		{
			name:              "Object with multiple properties",
			input:             "{name: 'John', age: 30}",
			expectedType:      (*ast_domain.ObjectLiteral)(nil),
			expectedSourceLen: 23,
		},

		{
			name:              "Empty array",
			input:             "[]",
			expectedType:      (*ast_domain.ArrayLiteral)(nil),
			expectedSourceLen: 2,
		},
		{
			name:              "Array with one element",
			input:             "[1]",
			expectedType:      (*ast_domain.ArrayLiteral)(nil),
			expectedSourceLen: 3,
		},
		{
			name:              "Array with multiple elements",
			input:             "[1, 2, 3, 4, 5]",
			expectedType:      (*ast_domain.ArrayLiteral)(nil),
			expectedSourceLen: 15,
		},

		{
			name:              "Simple for-in",
			input:             "item in items",
			expectedType:      (*ast_domain.ForInExpression)(nil),
			expectedSourceLen: 13,
		},
		{
			name:              "For-in with index",
			input:             "(i, item) in items",
			expectedType:      (*ast_domain.ForInExpression)(nil),
			expectedSourceLen: 18,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := ast_domain.NewExpressionParser(context.Background(), tc.input, "test.pkc")
			expression, diagnostics := parser.ParseExpression(context.Background())

			require.Empty(t, diagnostics, "Expected no parsing diagnostics for input: %s", tc.input)
			require.NotNil(t, expression, "Expected non-nil expression for input: %s", tc.input)

			require.IsType(t, tc.expectedType, expression, "Expression type mismatch for input: %s", tc.input)

			actualLen := expression.GetSourceLength()
			require.Equal(t, tc.expectedSourceLen, actualLen,
				"SourceLength mismatch for input: %s\nExpected: %d\nActual: %d",
				tc.input, tc.expectedSourceLen, actualLen)

			loc := expression.GetRelativeLocation()
			require.GreaterOrEqual(t, loc.Offset, 0, "Offset should be non-negative")

			if tc.additionalChecks != nil {
				tc.additionalChecks(t, expression)
			}
		})
	}
}

func TestLocation_Offset(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedTokens []struct {
			value  string
			offset int
		}
	}{
		{
			name:  "Simple tokens",
			input: "a + b",
			expectedTokens: []struct {
				value  string
				offset int
			}{
				{value: "a", offset: 0},
				{value: "+", offset: 2},
				{value: "b", offset: 4},
			},
		},
		{
			name:  "Tokens with different spacing",
			input: "  abc   +   definition",
			expectedTokens: []struct {
				value  string
				offset int
			}{
				{value: "abc", offset: 2},
				{value: "+", offset: 8},
				{value: "definition", offset: 12},
			},
		},
		{
			name:  "String literal position",
			input: "name + 'hello'",
			expectedTokens: []struct {
				value  string
				offset int
			}{
				{value: "name", offset: 0},
				{value: "+", offset: 5},
				{value: "'hello'", offset: 7},
			},
		},
		{
			name:  "Nested expression",
			input: "(a + b) * c",
			expectedTokens: []struct {
				value  string
				offset int
			}{
				{value: "(", offset: 0},
				{value: "a", offset: 1},
				{value: "+", offset: 3},
				{value: "b", offset: 5},
				{value: ")", offset: 6},
				{value: "*", offset: 8},
				{value: "c", offset: 10},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			parser := ast_domain.NewExpressionParser(context.Background(), tc.input, "test.pkc")
			expression, diagnostics := parser.ParseExpression(context.Background())

			require.Empty(t, diagnostics)
			require.NotNil(t, expression)

			loc := expression.GetRelativeLocation()
			require.GreaterOrEqual(t, loc.Offset, 0)

			expectedEndOffset := len(tc.input)
			actualEndOffset := loc.Offset + expression.GetSourceLength()
			require.LessOrEqual(t, actualEndOffset, expectedEndOffset,
				"Expression span should not exceed input length")
		})
	}
}

func TestSourceLength_Encoding(t *testing.T) {

	input := `<div p-if="user.isActive && user.age > 18" p-class="{active: isActive}" p-for="item in items">
		<span>{{ user.name }}</span>
		<p>{{ items[0] }}</p>
		<button p-on:click="handleClick(item.id)">Click {{ index + 1 }}</button>
	</div>`

	ast := mustParse(t, input)
	require.NotNil(t, ast)

	var expressions []ast_domain.Expression
	for node := range ast.Nodes() {

		if node.DirIf != nil && node.DirIf.Expression != nil {
			expressions = append(expressions, node.DirIf.Expression)
		}
		if node.DirFor != nil && node.DirFor.Expression != nil {
			expressions = append(expressions, node.DirFor.Expression)
		}
		if node.DirClass != nil && node.DirClass.Expression != nil {
			expressions = append(expressions, node.DirClass.Expression)
		}

		for _, handlers := range node.OnEvents {
			for _, handler := range handlers {
				if handler.Expression != nil {
					expressions = append(expressions, handler.Expression)
				}
			}
		}

		for _, part := range node.RichText {
			if !part.IsLiteral && part.Expression != nil {
				expressions = append(expressions, part.Expression)
			}
		}
	}

	require.NotEmpty(t, expressions, "Should have collected expressions from AST")

	type expressionSnapshot struct {
		sourceLength int
		offset       int
	}
	originalValues := make([]expressionSnapshot, len(expressions))
	for i, expression := range expressions {
		originalValues[i] = expressionSnapshot{
			sourceLength: expression.GetSourceLength(),
			offset:       expression.GetRelativeLocation().Offset,
		}
	}

	data, err := ast_adapters.EncodeAST(ast)
	require.NoError(t, err)

	roundTripped, err := ast_adapters.DecodeAST(context.Background(), data)
	require.NoError(t, err)
	require.NotNil(t, roundTripped)

	var rtExpressions []ast_domain.Expression
	for node := range roundTripped.Nodes() {
		if node.DirIf != nil && node.DirIf.Expression != nil {
			rtExpressions = append(rtExpressions, node.DirIf.Expression)
		}
		if node.DirFor != nil && node.DirFor.Expression != nil {
			rtExpressions = append(rtExpressions, node.DirFor.Expression)
		}
		if node.DirClass != nil && node.DirClass.Expression != nil {
			rtExpressions = append(rtExpressions, node.DirClass.Expression)
		}
		for _, handlers := range node.OnEvents {
			for _, handler := range handlers {
				if handler.Expression != nil {
					rtExpressions = append(rtExpressions, handler.Expression)
				}
			}
		}
		for _, part := range node.RichText {
			if !part.IsLiteral && part.Expression != nil {
				rtExpressions = append(rtExpressions, part.Expression)
			}
		}
	}

	require.Equal(t, len(expressions), len(rtExpressions),
		"Should have same number of expressions after round-trip")

	for i := range rtExpressions {
		actualLen := rtExpressions[i].GetSourceLength()
		actualOffset := rtExpressions[i].GetRelativeLocation().Offset

		require.Equal(t, originalValues[i].sourceLength, actualLen,
			"SourceLength should be preserved after round-trip for expression %d", i)
		require.Equal(t, originalValues[i].offset, actualOffset,
			"Offset should be preserved after round-trip for expression %d", i)
	}
}

func TestSourceLength_NestedExpressions(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "Deeply nested binary",
			input: "((a + b) * (c - d)) / e",
		},
		{
			name:  "Nested ternary",
			input: "a ? (b ? c : d) : (e ? f : g)",
		},
		{
			name:  "Complex call chain",
			input: "user.getProfile().getName().toUpperCase()",
		},
		{
			name:  "Array of objects",
			input: "[{a: 1}, {b: 2}, {c: 3}]",
		},
		{
			name:  "Object with nested arrays",
			input: "{users: [1, 2, 3], ids: [4, 5, 6]}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := ast_domain.NewExpressionParser(context.Background(), tc.input, "test.pkc")
			expression, diagnostics := parser.ParseExpression(context.Background())

			require.Empty(t, diagnostics)
			require.NotNil(t, expression)

			expectedLen := len(tc.input)
			actualLen := expression.GetSourceLength()

			require.Equal(t, expectedLen, actualLen,
				"Root expression SourceLength should match input length.\nInput: %s\nExpected: %d\nActual: %d",
				tc.input, expectedLen, actualLen)

			verifyAllExpressionsHaveSourceLength(t, expression)
		})
	}
}

func verifyAllExpressionsHaveSourceLength(t *testing.T, expression ast_domain.Expression) {
	t.Helper()
	if expression == nil {
		return
	}

	sourceLen := expression.GetSourceLength()
	require.Greater(t, sourceLen, 0, "SourceLength should be positive for %T", expression)

	switch e := expression.(type) {
	case *ast_domain.BinaryExpression:
		verifyAllExpressionsHaveSourceLength(t, e.Left)
		verifyAllExpressionsHaveSourceLength(t, e.Right)
	case *ast_domain.UnaryExpression:
		verifyAllExpressionsHaveSourceLength(t, e.Right)
	case *ast_domain.MemberExpression:
		verifyAllExpressionsHaveSourceLength(t, e.Base)
		verifyAllExpressionsHaveSourceLength(t, e.Property)
	case *ast_domain.IndexExpression:
		verifyAllExpressionsHaveSourceLength(t, e.Base)
		verifyAllExpressionsHaveSourceLength(t, e.Index)
	case *ast_domain.CallExpression:
		verifyAllExpressionsHaveSourceLength(t, e.Callee)
		for _, argument := range e.Args {
			verifyAllExpressionsHaveSourceLength(t, argument)
		}
	case *ast_domain.TernaryExpression:
		verifyAllExpressionsHaveSourceLength(t, e.Condition)
		verifyAllExpressionsHaveSourceLength(t, e.Consequent)
		verifyAllExpressionsHaveSourceLength(t, e.Alternate)
	case *ast_domain.ObjectLiteral:
		for _, value := range e.Pairs {
			verifyAllExpressionsHaveSourceLength(t, value)
		}
	case *ast_domain.ArrayLiteral:
		for _, element := range e.Elements {
			verifyAllExpressionsHaveSourceLength(t, element)
		}
	case *ast_domain.ForInExpression:
		verifyAllExpressionsHaveSourceLength(t, e.Collection)
	}
}
