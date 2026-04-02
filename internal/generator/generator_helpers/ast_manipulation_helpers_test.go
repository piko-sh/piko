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

package generator_helpers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/wdk/maths"
)

func TestGetContentAST(t *testing.T) {
	t.Parallel()

	validAST := &ast_domain.TemplateAST{}

	testCases := []struct {
		input any
		want  *ast_domain.TemplateAST
		name  string
	}{
		{name: "nil collectionData", input: nil, want: nil},
		{name: "non-map collectionData", input: "string", want: nil},
		{name: "map missing contentAST key", input: map[string]any{"other": "val"}, want: nil},
		{name: "map with wrong type for contentAST", input: map[string]any{"contentAST": "not_ast"}, want: nil},
		{name: "map with nil contentAST", input: map[string]any{"contentAST": (*ast_domain.TemplateAST)(nil)}, want: nil},
		{name: "valid contentAST", input: map[string]any{"contentAST": validAST}, want: validAST},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := GetContentAST(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

type testStringer struct{ value string }

func (s testStringer) String() string { return s.value }

func TestValueToString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input any
		want  string
	}{
		{name: "nil", input: nil, want: ""},
		{name: "string", input: "hello", want: "hello"},
		{name: "byte slice", input: []byte("bytes"), want: "bytes"},
		{name: "int", input: int(42), want: "42"},
		{name: "int8", input: int8(-10), want: "-10"},
		{name: "int16", input: int16(300), want: "300"},
		{name: "int32", input: int32(100), want: "100"},
		{name: "int64", input: int64(-999), want: "-999"},
		{name: "uint", input: uint(5), want: "5"},
		{name: "uint8", input: uint8(255), want: "255"},
		{name: "uint16", input: uint16(1000), want: "1000"},
		{name: "uint32", input: uint32(50000), want: "50000"},
		{name: "uint64", input: uint64(123456), want: "123456"},
		{name: "float32", input: float32(2.5), want: "2.5"},
		{name: "float64", input: float64(3.14), want: "3.14"},
		{name: "bool true", input: true, want: "true"},
		{name: "bool false", input: false, want: "false"},
		{name: "Stringer", input: testStringer{value: "custom"}, want: "custom"},
		{name: "maths.Decimal", input: maths.NewDecimalFromString("19.99"), want: "19.99"},
		{name: "maths.Decimal zero", input: maths.NewDecimalFromString("0"), want: "0"},
		{name: "*maths.Decimal", input: func() *maths.Decimal { d := maths.NewDecimalFromString("3.14"); return &d }(), want: "3.14"},
		{name: "*maths.Decimal nil", input: (*maths.Decimal)(nil), want: ""},
		{name: "maths.BigInt", input: maths.NewBigIntFromInt(42), want: "42"},
		{name: "maths.BigInt overflow", input: maths.NewBigIntFromString("9223372036854775808"), want: "9223372036854775808"},
		{name: "*maths.BigInt", input: func() *maths.BigInt { b := maths.NewBigIntFromInt(99); return &b }(), want: "99"},
		{name: "*maths.BigInt nil", input: (*maths.BigInt)(nil), want: ""},
		{name: "maths.Money", input: maths.NewMoneyFromString("1500.00", "USD"), want: "1500 USD"},
		{name: "*maths.Money", input: func() *maths.Money { m := maths.NewMoneyFromString("25.50", "GBP"); return &m }(), want: "25.5 GBP"},
		{name: "*maths.Money nil", input: (*maths.Money)(nil), want: ""},
		{name: "fallback struct", input: struct{ X int }{X: 42}, want: "{42}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ValueToString(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPointerValueToString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input any
		want  string
	}{
		{name: "nil any", input: nil, want: ""},
		{name: "non-nil *string", input: new("hello"), want: "hello"},
		{name: "nil *string", input: (*string)(nil), want: ""},
		{name: "non-nil *int", input: new(42), want: "42"},
		{name: "nil *int", input: (*int)(nil), want: ""},
		{name: "non-nil *int64", input: new(int64(999)), want: "999"},
		{name: "nil *int64", input: (*int64)(nil), want: ""},
		{name: "non-nil *bool", input: new(true), want: "true"},
		{name: "nil *bool", input: (*bool)(nil), want: ""},
		{name: "non-nil *float64", input: new(3.14), want: "3.14"},
		{name: "nil *float64", input: (*float64)(nil), want: ""},
		{name: "unsupported type", input: new(struct{ X int }{X: 1}), want: fmt.Sprintf("%v", new(struct{ X int }{X: 1}))},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := PointerValueToString(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCheckExpressionForIdentifierUsage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		expression     ast_domain.Expression
		identifierName string
		want           bool
	}{
		{name: "nil expression", expression: nil, identifierName: "item", want: false},

		{name: "exact match", expression: &ast_domain.Identifier{Name: "item"}, identifierName: "item", want: true},
		{name: "prefix match", expression: &ast_domain.Identifier{Name: "item.name"}, identifierName: "item", want: true},
		{name: "no match", expression: &ast_domain.Identifier{Name: "other"}, identifierName: "item", want: false},
		{name: "partial name not prefix", expression: &ast_domain.Identifier{Name: "item"}, identifierName: "it", want: false},
		{name: "similar but no dot prefix", expression: &ast_domain.Identifier{Name: "items"}, identifierName: "item", want: false},

		{name: "string literal", expression: &ast_domain.StringLiteral{Value: "item"}, identifierName: "item", want: false},
		{name: "integer literal", expression: &ast_domain.IntegerLiteral{Value: 42}, identifierName: "item", want: false},
		{name: "float literal", expression: &ast_domain.FloatLiteral{Value: 3.14}, identifierName: "item", want: false},
		{name: "boolean literal", expression: &ast_domain.BooleanLiteral{Value: true}, identifierName: "item", want: false},
		{name: "nil literal", expression: &ast_domain.NilLiteral{}, identifierName: "item", want: false},

		{
			name:           "unary with match in right",
			expression:     &ast_domain.UnaryExpression{Right: &ast_domain.Identifier{Name: "item"}},
			identifierName: "item",
			want:           true,
		},
		{
			name:           "unary without match",
			expression:     &ast_domain.UnaryExpression{Right: &ast_domain.Identifier{Name: "other"}},
			identifierName: "item",
			want:           false,
		},

		{
			name: "binary match in left",
			expression: &ast_domain.BinaryExpression{
				Left:  &ast_domain.Identifier{Name: "item"},
				Right: &ast_domain.IntegerLiteral{Value: 5},
			},
			identifierName: "item",
			want:           true,
		},
		{
			name: "binary match in right",
			expression: &ast_domain.BinaryExpression{
				Left:  &ast_domain.IntegerLiteral{Value: 5},
				Right: &ast_domain.Identifier{Name: "item"},
			},
			identifierName: "item",
			want:           true,
		},
		{
			name: "binary no match",
			expression: &ast_domain.BinaryExpression{
				Left:  &ast_domain.Identifier{Name: "a"},
				Right: &ast_domain.Identifier{Name: "b"},
			},
			identifierName: "item",
			want:           false,
		},

		{
			name: "call match in callee",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "item.method"},
				Args:   nil,
			},
			identifierName: "item",
			want:           true,
		},
		{
			name: "call match in arguments",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "fn"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "item"}},
			},
			identifierName: "item",
			want:           true,
		},
		{
			name: "call no match",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "fn"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "other"}},
			},
			identifierName: "item",
			want:           false,
		},

		{
			name: "for-in match in collection",
			expression: &ast_domain.ForInExpression{
				Collection: &ast_domain.Identifier{Name: "item.list"},
			},
			identifierName: "item",
			want:           true,
		},
		{
			name: "for-in no match in collection",
			expression: &ast_domain.ForInExpression{
				Collection: &ast_domain.Identifier{Name: "otherList"},
			},
			identifierName: "item",
			want:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := CheckExpressionForIdentifierUsage(tc.expression, tc.identifierName)
			assert.Equal(t, tc.want, got)
		})
	}
}

func newMinimalNode() *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
	}
}

func TestIsLoopVarUsedInNode(t *testing.T) {
	t.Parallel()

	t.Run("nil node returns false", func(t *testing.T) {
		t.Parallel()

		assert.False(t, IsLoopVarUsedInNode(nil, "item"))
	})

	t.Run("empty loopVarName returns false", func(t *testing.T) {
		t.Parallel()

		node := newMinimalNode()
		assert.False(t, IsLoopVarUsedInNode(node, ""))
	})

	t.Run("node with nil DirIf does not panic", func(t *testing.T) {
		t.Parallel()

		node := newMinimalNode()

		assert.NotPanics(t, func() {
			IsLoopVarUsedInNode(node, "item")
		})
		assert.False(t, IsLoopVarUsedInNode(node, "item"))
	})

	t.Run("match in DirIf expression", func(t *testing.T) {
		t.Parallel()

		node := newMinimalNode()
		node.DirIf = &ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "item.visible"},
		}
		assert.True(t, IsLoopVarUsedInNode(node, "item"))
	})

	t.Run("match in DirFor expression", func(t *testing.T) {
		t.Parallel()

		node := newMinimalNode()
		node.DirFor = &ast_domain.Directive{
			Expression: &ast_domain.ForInExpression{
				Collection: &ast_domain.Identifier{Name: "item.children"},
			},
		}
		assert.True(t, IsLoopVarUsedInNode(node, "item"))
	})

	t.Run("match in DirShow expression", func(t *testing.T) {
		t.Parallel()

		node := newMinimalNode()
		node.DirShow = &ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "item.active"},
		}
		assert.True(t, IsLoopVarUsedInNode(node, "item"))
	})

	t.Run("match in DynamicAttributes", func(t *testing.T) {
		t.Parallel()

		node := newMinimalNode()
		node.DynamicAttributes = []ast_domain.DynamicAttribute{
			{
				Name:       "title",
				Expression: &ast_domain.Identifier{Name: "item.title"},
			},
		}
		assert.True(t, IsLoopVarUsedInNode(node, "item"))
	})

	t.Run("match in Binds map", func(t *testing.T) {
		t.Parallel()

		node := newMinimalNode()
		node.Binds = map[string]*ast_domain.Directive{
			"value": {
				Expression: &ast_domain.Identifier{Name: "item.value"},
			},
		}
		assert.True(t, IsLoopVarUsedInNode(node, "item"))
	})

	t.Run("match in OnEvents map", func(t *testing.T) {
		t.Parallel()

		node := newMinimalNode()
		node.OnEvents = map[string][]ast_domain.Directive{
			"click": {
				{Expression: &ast_domain.Identifier{Name: "item.handler"}},
			},
		}
		assert.True(t, IsLoopVarUsedInNode(node, "item"))
	})

	t.Run("match in CustomEvents map", func(t *testing.T) {
		t.Parallel()

		node := newMinimalNode()
		node.CustomEvents = map[string][]ast_domain.Directive{
			"update": {
				{Expression: &ast_domain.Identifier{Name: "item.onUpdate"}},
			},
		}
		assert.True(t, IsLoopVarUsedInNode(node, "item"))
	})

	t.Run("match in Directives slice", func(t *testing.T) {
		t.Parallel()

		node := newMinimalNode()
		node.Directives = []ast_domain.Directive{
			{Expression: &ast_domain.Identifier{Name: "item.custom"}},
		}
		assert.True(t, IsLoopVarUsedInNode(node, "item"))
	})

	t.Run("match in child node", func(t *testing.T) {
		t.Parallel()

		child := newMinimalNode()
		child.DirText = &ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "item.text"},
		}
		parent := newMinimalNode()
		parent.Children = []*ast_domain.TemplateNode{child}
		assert.True(t, IsLoopVarUsedInNode(parent, "item"))
	})

	t.Run("no match anywhere returns false", func(t *testing.T) {
		t.Parallel()

		child := newMinimalNode()
		child.DirText = &ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "other.text"},
		}
		parent := newMinimalNode()
		parent.DirIf = &ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "other.visible"},
		}
		parent.Children = []*ast_domain.TemplateNode{child}
		assert.False(t, IsLoopVarUsedInNode(parent, "item"))
	})

	t.Run("nil directive in Binds map skipped", func(t *testing.T) {
		t.Parallel()

		node := newMinimalNode()
		node.Binds = map[string]*ast_domain.Directive{
			"value": nil,
		}
		assert.False(t, IsLoopVarUsedInNode(node, "item"))
	})
}
