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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpressionFeature_Has(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		feature  ExpressionFeature
		check    ExpressionFeature
		expected bool
	}{
		{
			name:     "Single feature present",
			feature:  FeatureIdentifier,
			check:    FeatureIdentifier,
			expected: true,
		},
		{
			name:     "Single feature absent",
			feature:  FeatureIdentifier,
			check:    FeatureMemberExpr,
			expected: false,
		},
		{
			name:     "Combined features - first present",
			feature:  FeatureIdentifier | FeatureMemberExpr,
			check:    FeatureIdentifier,
			expected: true,
		},
		{
			name:     "Combined features - second present",
			feature:  FeatureIdentifier | FeatureMemberExpr,
			check:    FeatureMemberExpr,
			expected: true,
		},
		{
			name:     "Combined features - absent",
			feature:  FeatureIdentifier | FeatureMemberExpr,
			check:    FeatureCallExpression,
			expected: false,
		},
		{
			name:     "FeaturesPath has Identifier",
			feature:  FeaturesPath,
			check:    FeatureIdentifier,
			expected: true,
		},
		{
			name:     "FeaturesPath has MemberExpr",
			feature:  FeaturesPath,
			check:    FeatureMemberExpr,
			expected: true,
		},
		{
			name:     "FeaturesPath has IndexExpr",
			feature:  FeaturesPath,
			check:    FeatureIndexExpr,
			expected: true,
		},
		{
			name:     "FeaturesPath has Literals",
			feature:  FeaturesPath,
			check:    FeatureLiterals,
			expected: true,
		},
		{
			name:     "FeaturesPath does not have CallExpr",
			feature:  FeaturesPath,
			check:    FeatureCallExpression,
			expected: false,
		},
		{
			name:     "FeaturesPath does not have BinaryExpr",
			feature:  FeaturesPath,
			check:    FeatureBinaryExpression,
			expected: false,
		},
		{
			name:     "FeaturesI18n has LinkedMessage",
			feature:  FeaturesI18n,
			check:    FeatureLinkedMessage,
			expected: true,
		},
		{
			name:     "FeaturesCompiler does not have LinkedMessage",
			feature:  FeaturesCompiler,
			check:    FeatureLinkedMessage,
			expected: false,
		},
		{
			name:     "FeaturesAll has LinkedMessage",
			feature:  FeaturesAll,
			check:    FeatureLinkedMessage,
			expected: true,
		},
		{
			name:     "FeaturesNone has nothing",
			feature:  FeaturesNone,
			check:    FeatureIdentifier,
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.feature.Has(tc.check)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExpressionFeature_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected string
		feature  ExpressionFeature
	}{
		{name: "Identifier", feature: FeatureIdentifier, expected: "identifiers"},
		{name: "MemberExpr", feature: FeatureMemberExpr, expected: "member access (.)"},
		{name: "IndexExpr", feature: FeatureIndexExpr, expected: "index access ([])"},
		{name: "LiteralIndex", feature: FeatureLiteralIndex, expected: "literal indices"},
		{name: "BinaryExpr", feature: FeatureBinaryExpression, expected: "binary operators"},
		{name: "UnaryExpr", feature: FeatureUnaryExpression, expected: "unary operators"},
		{name: "CallExpr", feature: FeatureCallExpression, expected: "function calls"},
		{name: "TernaryExpr", feature: FeatureTernaryExpression, expected: "ternary expressions (?:)"},
		{name: "TemplateLiteral", feature: FeatureTemplateLiteral, expected: "template literals"},
		{name: "ForInExpr", feature: FeatureForInExpr, expected: "for-in expressions"},
		{name: "ObjectLiteral", feature: FeatureObjectLiteral, expected: "object literals"},
		{name: "ArrayLiteral", feature: FeatureArrayLiteral, expected: "array literals"},
		{name: "Literals", feature: FeatureLiterals, expected: "literals"},
		{name: "LinkedMessage", feature: FeatureLinkedMessage, expected: "linked messages (@)"},
		{name: "OptionalChaining", feature: FeatureOptionalChaining, expected: "optional chaining (?.)"},
		{name: "NullishCoalescing", feature: FeatureNullishCoalescing, expected: "nullish coalescing (??)"},
		{name: "Unknown combined", feature: FeaturesPath, expected: "expression features"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.feature.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateExpressionFeatures_PathExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		errorMessage string
		allowed      ExpressionFeature
		expectError  bool
	}{

		{
			name:        "Simple identifier allowed",
			input:       "user",
			allowed:     FeaturesPath,
			expectError: false,
		},
		{
			name:        "Member access allowed",
			input:       "user.name",
			allowed:     FeaturesPath,
			expectError: false,
		},
		{
			name:        "Nested member access allowed",
			input:       "user.address.city",
			allowed:     FeaturesPath,
			expectError: false,
		},
		{
			name:        "Index with integer literal allowed",
			input:       "items[0]",
			allowed:     FeaturesPath,
			expectError: false,
		},
		{
			name:        "Index with string literal allowed",
			input:       `items["key"]`,
			allowed:     FeaturesPath,
			expectError: false,
		},
		{
			name:        "Mixed path with member and index",
			input:       "user.addresses[0].city",
			allowed:     FeaturesPath,
			expectError: false,
		},

		{
			name:         "Binary expression forbidden",
			input:        "a + b",
			allowed:      FeaturesPath,
			expectError:  true,
			errorMessage: "Binary operators not allowed",
		},
		{
			name:         "Function call forbidden",
			input:        "format(x)",
			allowed:      FeaturesPath,
			expectError:  true,
			errorMessage: "Function calls not allowed",
		},
		{
			name:         "Ternary forbidden",
			input:        "a ? b : c",
			allowed:      FeaturesPath,
			expectError:  true,
			errorMessage: "Ternary expressions not allowed",
		},
		{
			name:         "Unary forbidden",
			input:        "!valid",
			allowed:      FeaturesPath,
			expectError:  true,
			errorMessage: "Unary operators not allowed",
		},
		{
			name:         "Array literal forbidden",
			input:        "[1, 2, 3]",
			allowed:      FeaturesPath,
			expectError:  true,
			errorMessage: "Array literals not allowed",
		},
		{
			name:         "Object literal forbidden",
			input:        "{a: 1}",
			allowed:      FeaturesPath,
			expectError:  true,
			errorMessage: "Object literals not allowed",
		},

		{
			name:         "Dynamic index forbidden when literal index required",
			input:        "items[i]",
			allowed:      FeaturesPath,
			expectError:  true,
			errorMessage: "Dynamic index access not allowed",
		},

		{
			name:        "Binary expression allowed in compiler",
			input:       "a + b",
			allowed:     FeaturesCompiler,
			expectError: false,
		},
		{
			name:        "Function call allowed in compiler",
			input:       "format(x)",
			allowed:     FeaturesCompiler,
			expectError: false,
		},
		{
			name:        "Ternary allowed in compiler",
			input:       "a ? b : c",
			allowed:     FeaturesCompiler,
			expectError: false,
		},
		{
			name:        "Array literal allowed in compiler",
			input:       "[1, 2, 3]",
			allowed:     FeaturesCompiler,
			expectError: false,
		},
		{
			name:        "Object literal allowed in compiler",
			input:       "{a: 1, b: 2}",
			allowed:     FeaturesCompiler,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tc.input)
			diagnostics := ValidateExpressionFeatures(expression, tc.allowed, "test.html")

			if tc.expectError {
				require.NotEmpty(t, diagnostics, "Expected validation errors but got none")
				if tc.errorMessage != "" {
					found := false
					for _, d := range diagnostics {
						if d.Severity == Error && contains(d.Message, tc.errorMessage) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error containing %q, got: %v", tc.errorMessage, diagnostics)
				}
			} else {
				assert.Empty(t, diagnostics, "Expected no validation errors, got: %v", formatDiagsForTest(diagnostics))
			}
		})
	}
}

func TestValidateExpressionFeatures_OptionalChaining(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		allowed     ExpressionFeature
		expectError bool
	}{
		{
			name:        "Optional member access allowed",
			input:       "user?.name",
			allowed:     FeaturesCompiler,
			expectError: false,
		},
		{
			name:        "Optional index access allowed",
			input:       "items?.[0]",
			allowed:     FeaturesCompiler,
			expectError: false,
		},
		{
			name:        "Optional chaining forbidden in path context",
			input:       "user?.name",
			allowed:     FeaturesPath,
			expectError: true,
		},
		{
			name:        "Optional index forbidden in path context",
			input:       "items?.[0]",
			allowed:     FeaturesPath,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tc.input)
			diagnostics := ValidateExpressionFeatures(expression, tc.allowed, "test.html")

			if tc.expectError {
				require.NotEmpty(t, diagnostics, "Expected validation errors but got none")
			} else {
				assert.Empty(t, diagnostics, "Expected no validation errors, got: %v", formatDiagsForTest(diagnostics))
			}
		})
	}
}

func TestValidateExpressionFeatures_NullishCoalescing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		allowed     ExpressionFeature
		expectError bool
	}{
		{
			name:        "Nullish coalescing allowed in compiler",
			input:       "value ?? defaultValue",
			allowed:     FeaturesCompiler,
			expectError: false,
		},
		{
			name:        "Nullish coalescing forbidden in path context",
			input:       "value ?? defaultValue",
			allowed:     FeaturesPath,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tc.input)
			diagnostics := ValidateExpressionFeatures(expression, tc.allowed, "test.html")

			if tc.expectError {
				require.NotEmpty(t, diagnostics, "Expected validation errors but got none")
			} else {
				assert.Empty(t, diagnostics, "Expected no validation errors, got: %v", formatDiagsForTest(diagnostics))
			}
		})
	}
}

func TestValidateExpressionFeatures_Nil(t *testing.T) {
	t.Parallel()

	diagnostics := ValidateExpressionFeatures(nil, FeaturesPath, "test.html")
	assert.Nil(t, diagnostics, "Should return nil for nil expression")
}

func TestValidatePathExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{name: "Simple identifier", input: "user", expectError: false},
		{name: "Member access", input: "user.name", expectError: false},
		{name: "Index access", input: "items[0]", expectError: false},
		{name: "Binary forbidden", input: "a + b", expectError: true},
		{name: "Call forbidden", input: "fn()", expectError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tc.input)
			diagnostics := ValidatePathExpression(expression, "test.html")

			if tc.expectError {
				require.NotEmpty(t, diagnostics)
			} else {
				assert.Empty(t, diagnostics)
			}
		})
	}
}

func TestIsPathExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "Simple identifier", input: "user", expected: true},
		{name: "Member access", input: "user.name", expected: true},
		{name: "Nested member", input: "user.address.city", expected: true},
		{name: "Index with literal", input: "items[0]", expected: true},
		{name: "Index with string", input: `items["key"]`, expected: true},
		{name: "Mixed path", input: "user.items[0].name", expected: true},
		{name: "Binary expression", input: "a + b", expected: false},
		{name: "Function call", input: "fn()", expected: false},
		{name: "Ternary", input: "a ? b : c", expected: false},
		{name: "Dynamic index", input: "items[index]", expected: false},
		{name: "Unary", input: "!x", expected: false},
		{name: "Array literal", input: "[1, 2]", expected: false},
		{name: "Object literal", input: "{a: 1}", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tc.input)
			result := IsPathExpression(expression)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateExpressionFeatures_NestedInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		allowed     ExpressionFeature
		expectError bool
		errorCount  int
	}{
		{
			name:        "Call inside member expression",
			input:       "user.getName()",
			allowed:     FeaturesPath,
			expectError: true,
			errorCount:  1,
		},
		{
			name:        "Binary inside array",
			input:       "[a + b]",
			allowed:     FeaturesPath,
			expectError: true,
			errorCount:  2,
		},
		{
			name:        "Multiple invalid nested",
			input:       "{x: fn(), y: a + b}",
			allowed:     FeaturesPath,
			expectError: true,
			errorCount:  3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tc.input)
			diagnostics := ValidateExpressionFeatures(expression, tc.allowed, "test.html")

			if tc.expectError {
				require.NotEmpty(t, diagnostics)
				assert.Len(t, diagnostics, tc.errorCount, "Expected %d errors, got %d: %v",
					tc.errorCount, len(diagnostics), formatDiagsForTest(diagnostics))
			}
		})
	}
}

func TestValidateExpressionFeatures_LiteralTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		allowed     ExpressionFeature
		expectError bool
	}{
		{name: "String literal allowed", input: `"hello"`, allowed: FeaturesPath, expectError: false},
		{name: "Integer literal allowed", input: "123", allowed: FeaturesPath, expectError: false},
		{name: "Float literal allowed", input: "123.45", allowed: FeaturesPath, expectError: false},
		{name: "Boolean literal allowed", input: "true", allowed: FeaturesPath, expectError: false},
		{name: "Nil literal allowed", input: "nil", allowed: FeaturesPath, expectError: false},
		{name: "Literals forbidden when no feature", input: "123", allowed: FeaturesNone, expectError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tc.input)
			diagnostics := ValidateExpressionFeatures(expression, tc.allowed, "test.html")

			if tc.expectError {
				require.NotEmpty(t, diagnostics)
			} else {
				assert.Empty(t, diagnostics)
			}
		})
	}
}

func TestParseLinkedMessage_Simple(t *testing.T) {
	t.Parallel()

	expression := mustParseExpr(t, "@greeting")

	lm, ok := expression.(*LinkedMessageExpression)
	require.True(t, ok, "Expected LinkedMessageExpression, got %T", expression)

	assert.Equal(t, "@greeting", lm.String())

	id, ok := lm.Path.(*Identifier)
	require.True(t, ok, "Expected path to be Identifier, got %T", lm.Path)
	assert.Equal(t, "greeting", id.Name)
}

func TestParseLinkedMessage_MemberChain(t *testing.T) {
	t.Parallel()

	expression := mustParseExpr(t, "@common.greeting")

	lm, ok := expression.(*LinkedMessageExpression)
	require.True(t, ok, "Expected LinkedMessageExpression, got %T", expression)

	assert.Equal(t, "@common.greeting", lm.String())

	me, ok := lm.Path.(*MemberExpression)
	require.True(t, ok, "Expected path to be MemberExpression, got %T", lm.Path)

	base, ok := me.Base.(*Identifier)
	require.True(t, ok)
	assert.Equal(t, "common", base.Name)

	prop, ok := me.Property.(*Identifier)
	require.True(t, ok)
	assert.Equal(t, "greeting", prop.Name)
}

func TestParseLinkedMessage_DeepNesting(t *testing.T) {
	t.Parallel()

	expression := mustParseExpr(t, "@messages.errors.notFound")

	lm, ok := expression.(*LinkedMessageExpression)
	require.True(t, ok, "Expected LinkedMessageExpression, got %T", expression)

	assert.Equal(t, "@messages.errors.notFound", lm.String())
}

func TestParseLinkedMessage_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		errorMessage string
	}{
		{
			name:         "Number after @",
			input:        "@123",
			errorMessage: "Expected identifier after '@'",
		},
		{
			name:         "Nothing after @",
			input:        "@",
			errorMessage: "Expected identifier after '@'",
		},
		{
			name:         "Missing identifier after dot",
			input:        "@common.",
			errorMessage: "Expected identifier after '.'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser := NewExpressionParser(context.Background(), tc.input, "test")
			_, diagnostics := parser.ParseExpression(context.Background())
			require.NotEmpty(t, diagnostics, "Expected error but got none")

			found := false
			for _, d := range diagnostics {
				if d.Severity == Error && contains(d.Message, tc.errorMessage) {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected error containing %q, got: %v", tc.errorMessage, diagnostics)
		})
	}
}

func TestValidateExpressionFeatures_LinkedMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		allowed     ExpressionFeature
		expectError bool
	}{
		{
			name:        "Linked message allowed in i18n",
			input:       "@common.greeting",
			allowed:     FeaturesI18n,
			expectError: false,
		},
		{
			name:        "Linked message forbidden in compiler",
			input:       "@common.greeting",
			allowed:     FeaturesCompiler,
			expectError: true,
		},
		{
			name:        "Linked message forbidden in path",
			input:       "@common.greeting",
			allowed:     FeaturesPath,
			expectError: true,
		},
		{
			name:        "Linked message allowed in FeaturesAll",
			input:       "@common.greeting",
			allowed:     FeaturesAll,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expression := mustParseExpr(t, tc.input)
			diagnostics := ValidateExpressionFeatures(expression, tc.allowed, "test.html")

			if tc.expectError {
				require.NotEmpty(t, diagnostics, "Expected validation errors but got none")
			} else {
				assert.Empty(t, diagnostics, "Expected no validation errors, got: %v", formatDiagsForTest(diagnostics))
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
