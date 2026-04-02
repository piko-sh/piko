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

package binder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestIsPathExpression(t *testing.T) {
	testCases := []struct {
		name       string
		expression string
		expected   bool
	}{
		{
			name:       "Simple identifier",
			expression: "Name",
			expected:   true,
		},
		{
			name:       "Simple member access",
			expression: "User.Name",
			expected:   true,
		},
		{
			name:       "Deep member access",
			expression: "User.Profile.Email",
			expected:   true,
		},
		{
			name:       "Simple index access with integer literal",
			expression: "Items[0]",
			expected:   true,
		},
		{
			name:       "Index access on a nested field",
			expression: "Order.Items[1]",
			expected:   true,
		},
		{
			name:       "Member access on an indexed element",
			expression: "Items[0].Name",
			expected:   true,
		},
		{
			name:       "Deeply nested with multiple indexes",
			expression: "Pages[0].Sections[1].Columns[2].Content",
			expected:   true,
		},
		{
			name:       "Binary expression with +",
			expression: "User.Age + 1",
			expected:   false,
		},
		{
			name:       "Binary expression with ==",
			expression: "User.ID == 1",
			expected:   false,
		},
		{
			name:       "Unary expression with !",
			expression: "!IsActive",
			expected:   false,
		},
		{
			name:       "Ternary expression",
			expression: "IsAdmin ? 'Admin' : 'User'",
			expected:   false,
		},
		{
			name:       "Standalone string literal",
			expression: `"FieldName"`,
			expected:   false,
		},
		{
			name:       "Standalone integer literal",
			expression: "123",
			expected:   false,
		},
		{
			name:       "Standalone boolean literal",
			expression: "true",
			expected:   false,
		},
		{
			name:       "Function call",
			expression: "GetUser()",
			expected:   false,
		},
		{
			name:       "Method call on a path",
			expression: "User.GetName()",
			expected:   false,
		},
		{
			name:       "Index access with string literal (valid for map keys)",
			expression: `Items["key"]`,
			expected:   true,
		},
		{
			name:       "Index access with variable identifier",
			expression: "Items[i]",
			expected:   false,
		},
		{
			name:       "Index access with binary expression",
			expression: "Items[1 + 1]",
			expected:   false,
		},
		{
			name:       "Nil expression",
			expression: "",
			expected:   false,
		},
		{
			name:       "Expression with only an operator",
			expression: "+",
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := ast_domain.NewExpressionParser(context.Background(), tc.expression, "validate_test.go")
			parsedExpr, diagnostics := parser.ParseExpression(context.Background())

			if ast_domain.HasErrors(diagnostics) && tc.expression != "" {
				assert.False(t, tc.expected, "Expression failed to parse, should be considered invalid")
				return
			}

			isValid := isPathExpression(parsedExpr)

			assert.Equal(t, tc.expected, isValid, "Validation result for expression '%s' did not match expectation", tc.expression)
		})
	}
}
