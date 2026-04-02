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

package compiler_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestSquashWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no whitespace changes needed",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "carriage return to space",
			input:    "hello\rworld",
			expected: "hello world",
		},
		{
			name:     "newline to space",
			input:    "hello\nworld",
			expected: "hello world",
		},
		{
			name:     "tab to space",
			input:    "hello\tworld",
			expected: "hello world",
		},
		{
			name:     "multiple spaces collapsed",
			input:    "hello     world",
			expected: "hello world",
		},
		{
			name:     "mixed whitespace collapsed",
			input:    "hello\r\n\t  world",
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "  \t\n\r  ",
			expected: " ",
		},
		{
			name:     "preserves leading and trailing",
			input:    "  hello  ",
			expected: " hello ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := squashWhitespace(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no escaping needed",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "backslash",
			input:    `hello\world`,
			expected: `hello\\world`,
		},
		{
			name:     "double quote",
			input:    `hello "world"`,
			expected: `hello \"world\"`,
		},
		{
			name:     "newline",
			input:    "hello\nworld",
			expected: `hello\nworld`,
		},
		{
			name:     "carriage return",
			input:    "hello\rworld",
			expected: `hello\rworld`,
		},
		{
			name:     "tab",
			input:    "hello\tworld",
			expected: `hello\tworld`,
		},
		{
			name:     "mixed special characters",
			input:    "line1\nline2\t\"quoted\"",
			expected: `line1\nline2\t\"quoted\"`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "backslash before quote",
			input:    `\"`,
			expected: `\\\"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNumericOrBoolOrNull(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "true literal",
			input:    "true",
			expected: true,
		},
		{
			name:     "false literal",
			input:    "false",
			expected: true,
		},
		{
			name:     "TRUE uppercase",
			input:    "TRUE",
			expected: true,
		},
		{
			name:     "False mixed case",
			input:    "False",
			expected: true,
		},
		{
			name:     "null literal",
			input:    "null",
			expected: true,
		},
		{
			name:     "undefined literal",
			input:    "undefined",
			expected: true,
		},
		{
			name:     "integer",
			input:    "42",
			expected: true,
		},
		{
			name:     "negative integer",
			input:    "-42",
			expected: true,
		},
		{
			name:     "float",
			input:    "3.14",
			expected: true,
		},
		{
			name:     "negative float",
			input:    "-3.14",
			expected: true,
		},
		{
			name:     "zero",
			input:    "0",
			expected: true,
		},
		{
			name:     "string value",
			input:    "hello",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "whitespace padded number",
			input:    "  42  ",
			expected: true,
		},
		{
			name:     "whitespace padded bool",
			input:    "  true  ",
			expected: true,
		},
		{
			name:     "not a number",
			input:    "abc123",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNumericOrBoolOrNull(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSscanfFloat(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    float64
		expectError bool
	}{
		{
			name:        "integer",
			input:       "42",
			expected:    42.0,
			expectError: false,
		},
		{
			name:        "float",
			input:       "3.14",
			expected:    3.14,
			expectError: false,
		},
		{
			name:        "negative",
			input:       "-10.5",
			expected:    -10.5,
			expectError: false,
		},
		{
			name:        "zero",
			input:       "0",
			expected:    0.0,
			expectError: false,
		},
		{
			name:        "invalid string",
			input:       "abc",
			expected:    0.0,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    0.0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SscanfFloat(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.expected, result, 0.0001)
			}
		})
	}
}

func TestFilterOutKeyAttrs(t *testing.T) {
	tests := []struct {
		name     string
		input    []ast_domain.DynamicAttribute
		expected int
	}{
		{
			name:     "empty slice",
			input:    []ast_domain.DynamicAttribute{},
			expected: 0,
		},
		{
			name: "no key attributes",
			input: []ast_domain.DynamicAttribute{
				{Name: "class"},
				{Name: "id"},
			},
			expected: 2,
		},
		{
			name: "has key attribute",
			input: []ast_domain.DynamicAttribute{
				{Name: "class"},
				{Name: "key"},
				{Name: "id"},
			},
			expected: 2,
		},
		{
			name: "key with different case",
			input: []ast_domain.DynamicAttribute{
				{Name: "class"},
				{Name: "KEY"},
				{Name: "Key"},
			},
			expected: 1,
		},
		{
			name: "only key attributes",
			input: []ast_domain.DynamicAttribute{
				{Name: "key"},
				{Name: "KEY"},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterOutKeyAttrs(tt.input)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestFilterOutKeyAttrsHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    []ast_domain.HTMLAttribute
		expected int
	}{
		{
			name:     "empty slice",
			input:    []ast_domain.HTMLAttribute{},
			expected: 0,
		},
		{
			name: "no key attributes",
			input: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "id", Value: "main"},
			},
			expected: 2,
		},
		{
			name: "has key attribute",
			input: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "key", Value: "item-1"},
				{Name: "id", Value: "main"},
			},
			expected: 2,
		},
		{
			name: "key with different case",
			input: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "KEY", Value: "item-1"},
				{Name: "Key", Value: "item-2"},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterOutKeyAttrsHTML(tt.input)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestCopyLoopVars(t *testing.T) {
	t.Run("nil input returns empty map", func(t *testing.T) {
		result := copyLoopVars(nil)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("empty map returns empty map", func(t *testing.T) {
		result := copyLoopVars(map[string]bool{})
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("copies all entries", func(t *testing.T) {
		src := map[string]bool{
			"item":  true,
			"index": true,
		}
		result := copyLoopVars(src)
		assert.Len(t, result, 2)
		assert.True(t, result["item"])
		assert.True(t, result["index"])
	})

	t.Run("copy is independent of source", func(t *testing.T) {
		src := map[string]bool{"item": true}
		result := copyLoopVars(src)

		src["newKey"] = true

		assert.Len(t, result, 1)
		assert.False(t, result["newKey"])
	})
}

func TestCopyLoopVarsWith(t *testing.T) {
	t.Run("adds item and index to nil source", func(t *testing.T) {
		result := copyLoopVarsWith(nil, "item", "i")
		assert.Len(t, result, 2)
		assert.True(t, result["item"])
		assert.True(t, result["i"])
	})

	t.Run("adds item and index to existing map", func(t *testing.T) {
		src := map[string]bool{"existing": true}
		result := copyLoopVarsWith(src, "item", "i")
		assert.Len(t, result, 3)
		assert.True(t, result["existing"])
		assert.True(t, result["item"])
		assert.True(t, result["i"])
	})

	t.Run("handles empty item name", func(t *testing.T) {
		result := copyLoopVarsWith(nil, "", "i")
		assert.Len(t, result, 1)
		assert.True(t, result["i"])
	})

	t.Run("handles empty index name", func(t *testing.T) {
		result := copyLoopVarsWith(nil, "item", "")
		assert.Len(t, result, 1)
		assert.True(t, result["item"])
	})

	t.Run("handles both names empty", func(t *testing.T) {
		result := copyLoopVarsWith(nil, "", "")
		assert.Empty(t, result)
	})
}

func TestGetLoopVarNames(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result := getLoopVarNames(nil)
		assert.Nil(t, result)
	})

	t.Run("empty map returns empty slice", func(t *testing.T) {
		result := getLoopVarNames(map[string]bool{})
		assert.Empty(t, result)
	})

	t.Run("returns all keys", func(t *testing.T) {
		input := map[string]bool{
			"item":  true,
			"index": true,
		}
		result := getLoopVarNames(input)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "item")
		assert.Contains(t, result, "index")
	})
}

func TestCloneNode(t *testing.T) {
	t.Run("clones basic properties", func(t *testing.T) {
		original := &ast_domain.TemplateNode{
			NodeType:    ast_domain.NodeElement,
			TagName:     "div",
			TextContent: "hello",
		}

		cloned := cloneNode(original)

		assert.Equal(t, original.NodeType, cloned.NodeType)
		assert.Equal(t, original.TagName, cloned.TagName)
		assert.Equal(t, original.TextContent, cloned.TextContent)
		assert.NotSame(t, original, cloned)
	})

	t.Run("clones attributes independently", func(t *testing.T) {
		original := &ast_domain.TemplateNode{
			TagName: "div",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
			},
		}

		cloned := cloneNode(original)

		original.Attributes[0].Value = "modified"

		assert.Equal(t, "container", cloned.Attributes[0].Value)
	})

	t.Run("clones dynamic attributes independently", func(t *testing.T) {
		original := &ast_domain.TemplateNode{
			TagName: "div",
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "class"},
			},
		}

		cloned := cloneNode(original)

		original.DynamicAttributes = append(original.DynamicAttributes, ast_domain.DynamicAttribute{Name: "id"})

		assert.Len(t, cloned.DynamicAttributes, 1)
	})

	t.Run("clones children recursively", func(t *testing.T) {
		original := &ast_domain.TemplateNode{
			TagName: "div",
			Children: []*ast_domain.TemplateNode{
				{
					TagName:     "span",
					TextContent: "child",
				},
			},
		}

		cloned := cloneNode(original)

		original.Children[0].TextContent = "modified"

		assert.Equal(t, "child", cloned.Children[0].TextContent)
	})

	t.Run("clones on events map independently", func(t *testing.T) {
		original := &ast_domain.TemplateNode{
			TagName: "button",
			OnEvents: map[string][]ast_domain.Directive{
				"click": {{Arg: "click"}},
			},
		}

		cloned := cloneNode(original)

		original.OnEvents["click"] = append(original.OnEvents["click"], ast_domain.Directive{Arg: "dblclick"})

		assert.Len(t, cloned.OnEvents["click"], 1)
	})

	t.Run("clones custom events map independently", func(t *testing.T) {
		original := &ast_domain.TemplateNode{
			TagName: "custom-element",
			CustomEvents: map[string][]ast_domain.Directive{
				"custom-event": {{Arg: "custom-event"}},
			},
		}

		cloned := cloneNode(original)

		original.CustomEvents["new-event"] = []ast_domain.Directive{{Arg: "new-event"}}

		_, hasNewKey := cloned.CustomEvents["new-event"]
		assert.False(t, hasNewKey)
	})

	t.Run("handles nil slices and maps", func(t *testing.T) {
		original := &ast_domain.TemplateNode{
			TagName: "div",
		}

		cloned := cloneNode(original)

		assert.Nil(t, cloned.Attributes)
		assert.Nil(t, cloned.DynamicAttributes)
		assert.Nil(t, cloned.Directives)
		assert.Nil(t, cloned.OnEvents)
		assert.Nil(t, cloned.CustomEvents)
		assert.Nil(t, cloned.Children)
	})
}

func TestExprToJSString(t *testing.T) {
	tests := []struct {
		name       string
		expression js_ast.Expr
		expected   string
	}{
		{
			name:       "nil data",
			expression: js_ast.Expr{Data: nil},
			expected:   "null",
		},
		{
			name:       "null expression",
			expression: js_ast.Expr{Data: js_ast.ENullShared},
			expected:   "null",
		},
		{
			name:       "boolean true",
			expression: js_ast.Expr{Data: &js_ast.EBoolean{Value: true}},
			expected:   "true",
		},
		{
			name:       "boolean false",
			expression: js_ast.Expr{Data: &js_ast.EBoolean{Value: false}},
			expected:   "false",
		},
		{
			name:       "number",
			expression: js_ast.Expr{Data: &js_ast.ENumber{Value: 42}},
			expected:   "42",
		},
		{
			name:       "float number",
			expression: js_ast.Expr{Data: &js_ast.ENumber{Value: 3.14}},
			expected:   "3.14",
		},
		{
			name:       "identifier",
			expression: js_ast.Expr{Data: &js_ast.EIdentifier{}},
			expected:   "identifier",
		},
		{
			name:       "other expression type",
			expression: js_ast.Expr{Data: &js_ast.ECall{}},
			expected:   "expr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expressionToJSValueString(tt.expression)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNull(t *testing.T) {
	t.Run("null expression returns true", func(t *testing.T) {
		expression := js_ast.Expr{Data: js_ast.ENullShared}
		assert.True(t, isNull(expression))
	})

	t.Run("non-null expression returns false", func(t *testing.T) {
		expression := js_ast.Expr{Data: &js_ast.EString{}}
		assert.False(t, isNull(expression))
	})

	t.Run("nil data returns false", func(t *testing.T) {
		expression := js_ast.Expr{Data: nil}
		assert.False(t, isNull(expression))
	})
}

func TestNewStringLiteral(t *testing.T) {
	t.Run("creates string literal", func(t *testing.T) {
		expression := newStringLiteral("hello")
		require.NotNil(t, expression.Data)

		strExpr, ok := expression.Data.(*js_ast.EString)
		require.True(t, ok)
		assert.NotNil(t, strExpr.Value)
	})

	t.Run("empty string", func(t *testing.T) {
		expression := newStringLiteral("")
		require.NotNil(t, expression.Data)

		strExpr, ok := expression.Data.(*js_ast.EString)
		require.True(t, ok)
		assert.NotNil(t, strExpr.Value)
	})
}

func TestNewBooleanLiteral(t *testing.T) {
	t.Run("true value", func(t *testing.T) {
		expression := newBooleanLiteral(true)
		require.NotNil(t, expression.Data)

		boolExpr, ok := expression.Data.(*js_ast.EBoolean)
		require.True(t, ok)
		assert.True(t, boolExpr.Value)
	})

	t.Run("false value", func(t *testing.T) {
		expression := newBooleanLiteral(false)
		require.NotNil(t, expression.Data)

		boolExpr, ok := expression.Data.(*js_ast.EBoolean)
		require.True(t, ok)
		assert.False(t, boolExpr.Value)
	})
}

func TestNewNullLiteral(t *testing.T) {
	expression := newNullLiteral()
	require.NotNil(t, expression.Data)

	_, ok := expression.Data.(*js_ast.ENull)
	assert.True(t, ok)
}

func TestParseSnippetAsExpr(t *testing.T) {
	t.Run("parses simple expression", func(t *testing.T) {
		expression, err := parseSnippetAsExpr("42")
		require.NoError(t, err)
		require.NotNil(t, expression.Data)

		numExpr, ok := expression.Data.(*js_ast.ENumber)
		require.True(t, ok)
		assert.Equal(t, float64(42), numExpr.Value)
	})

	t.Run("parses string literal", func(t *testing.T) {

		_, err := parseSnippetAsExpr(`"hello"`)
		assert.Error(t, err)
	})

	t.Run("parses boolean", func(t *testing.T) {
		expression, err := parseSnippetAsExpr("true")
		require.NoError(t, err)
		require.NotNil(t, expression.Data)

		boolExpr, ok := expression.Data.(*js_ast.EBoolean)
		require.True(t, ok)
		assert.True(t, boolExpr.Value)
	})

	t.Run("parses binary expression", func(t *testing.T) {
		expression, err := parseSnippetAsExpr("1 + 2")
		require.NoError(t, err)
		require.NotNil(t, expression.Data)

		_, ok := expression.Data.(*js_ast.EBinary)
		assert.True(t, ok)
	})

	t.Run("parses function call", func(t *testing.T) {
		expression, err := parseSnippetAsExpr("foo()")
		require.NoError(t, err)
		require.NotNil(t, expression.Data)

		_, ok := expression.Data.(*js_ast.ECall)
		assert.True(t, ok)
	})

	t.Run("parses member access", func(t *testing.T) {
		expression, err := parseSnippetAsExpr("obj.prop")
		require.NoError(t, err)
		require.NotNil(t, expression.Data)

		_, ok := expression.Data.(*js_ast.EDot)
		assert.True(t, ok)
	})
}
