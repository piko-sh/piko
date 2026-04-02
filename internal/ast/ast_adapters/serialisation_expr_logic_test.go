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

package ast_adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestEncodeDecodeExpr_UnaryExpr(t *testing.T) {
	testCases := []struct {
		name     string
		operator ast_domain.UnaryOp
	}{
		{name: "NOT operator", operator: ast_domain.OpNot},
		{name: "NEG operator", operator: ast_domain.OpNeg},
		{name: "TRUTHY operator", operator: ast_domain.OpTruthy},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DirIf: &ast_domain.Directive{
							Type:          ast_domain.DirectiveIf,
							RawExpression: "!value",
							Expression: &ast_domain.UnaryExpression{
								Operator: tc.operator,
								Right:    &ast_domain.Identifier{Name: "value"},
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.NotNil(t, decoded.RootNodes[0].DirIf)
			require.NotNil(t, decoded.RootNodes[0].DirIf.Expression)

			unary, ok := decoded.RootNodes[0].DirIf.Expression.(*ast_domain.UnaryExpression)
			require.True(t, ok, "expected UnaryExpr")
			assert.Equal(t, tc.operator, unary.Operator)
			require.NotNil(t, unary.Right)
		})
	}
}

func TestEncodeDecodeExpr_NestedUnary(t *testing.T) {
	t.Run("double negation", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirIf: &ast_domain.Directive{
						Type:          ast_domain.DirectiveIf,
						RawExpression: "!!value",
						Expression: &ast_domain.UnaryExpression{
							Operator: ast_domain.OpNot,
							Right: &ast_domain.UnaryExpression{
								Operator: ast_domain.OpNot,
								Right:    &ast_domain.Identifier{Name: "value"},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		unary, ok := decoded.RootNodes[0].DirIf.Expression.(*ast_domain.UnaryExpression)
		require.True(t, ok)
		assert.Equal(t, ast_domain.OpNot, unary.Operator)

		inner, ok := unary.Right.(*ast_domain.UnaryExpression)
		require.True(t, ok)
		assert.Equal(t, ast_domain.OpNot, inner.Operator)

		identifier, ok := inner.Right.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "value", identifier.Name)
	})
}

func TestEncodeDecodeExpr_BinaryExpr(t *testing.T) {
	testCases := []struct {
		name     string
		operator ast_domain.BinaryOp
	}{
		{name: "equal", operator: ast_domain.OpEq},
		{name: "not equal", operator: ast_domain.OpNe},
		{name: "loose equal", operator: ast_domain.OpLooseEq},
		{name: "loose not equal", operator: ast_domain.OpLooseNe},
		{name: "greater than", operator: ast_domain.OpGt},
		{name: "less than", operator: ast_domain.OpLt},
		{name: "greater or equal", operator: ast_domain.OpGe},
		{name: "less or equal", operator: ast_domain.OpLe},
		{name: "logical AND", operator: ast_domain.OpAnd},
		{name: "logical OR", operator: ast_domain.OpOr},
		{name: "plus", operator: ast_domain.OpPlus},
		{name: "minus", operator: ast_domain.OpMinus},
		{name: "multiply", operator: ast_domain.OpMul},
		{name: "divide", operator: ast_domain.OpDiv},
		{name: "modulo", operator: ast_domain.OpMod},
		{name: "coalesce", operator: ast_domain.OpCoalesce},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						DirIf: &ast_domain.Directive{
							Type:          ast_domain.DirectiveIf,
							RawExpression: "a op b",
							Expression: &ast_domain.BinaryExpression{
								Operator: tc.operator,
								Left:     &ast_domain.Identifier{Name: "a"},
								Right:    &ast_domain.Identifier{Name: "b"},
							},
						},
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.NotNil(t, decoded.RootNodes[0].DirIf)
			require.NotNil(t, decoded.RootNodes[0].DirIf.Expression)

			binary, ok := decoded.RootNodes[0].DirIf.Expression.(*ast_domain.BinaryExpression)
			require.True(t, ok, "expected BinaryExpr")
			assert.Equal(t, tc.operator, binary.Operator)
			require.NotNil(t, binary.Left)
			require.NotNil(t, binary.Right)
		})
	}
}

func TestEncodeDecodeExpr_NestedBinary(t *testing.T) {
	t.Run("chained AND conditions", func(t *testing.T) {

		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirIf: &ast_domain.Directive{
						Type:          ast_domain.DirectiveIf,
						RawExpression: "a && b && c",
						Expression: &ast_domain.BinaryExpression{
							Operator: ast_domain.OpAnd,
							Left: &ast_domain.BinaryExpression{
								Operator: ast_domain.OpAnd,
								Left:     &ast_domain.Identifier{Name: "a"},
								Right:    &ast_domain.Identifier{Name: "b"},
							},
							Right: &ast_domain.Identifier{Name: "c"},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		outer, ok := decoded.RootNodes[0].DirIf.Expression.(*ast_domain.BinaryExpression)
		require.True(t, ok)
		assert.Equal(t, ast_domain.OpAnd, outer.Operator)

		inner, ok := outer.Left.(*ast_domain.BinaryExpression)
		require.True(t, ok)
		assert.Equal(t, ast_domain.OpAnd, inner.Operator)
	})

	t.Run("mixed operators with precedence", func(t *testing.T) {

		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirIf: &ast_domain.Directive{
						Type:          ast_domain.DirectiveIf,
						RawExpression: "a > 5 && b < 10",
						Expression: &ast_domain.BinaryExpression{
							Operator: ast_domain.OpAnd,
							Left: &ast_domain.BinaryExpression{
								Operator: ast_domain.OpGt,
								Left:     &ast_domain.Identifier{Name: "a"},
								Right:    &ast_domain.IntegerLiteral{Value: 5},
							},
							Right: &ast_domain.BinaryExpression{
								Operator: ast_domain.OpLt,
								Left:     &ast_domain.Identifier{Name: "b"},
								Right:    &ast_domain.IntegerLiteral{Value: 10},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		outer, ok := decoded.RootNodes[0].DirIf.Expression.(*ast_domain.BinaryExpression)
		require.True(t, ok)
		assert.Equal(t, ast_domain.OpAnd, outer.Operator)

		left, ok := outer.Left.(*ast_domain.BinaryExpression)
		require.True(t, ok)
		assert.Equal(t, ast_domain.OpGt, left.Operator)

		right, ok := outer.Right.(*ast_domain.BinaryExpression)
		require.True(t, ok)
		assert.Equal(t, ast_domain.OpLt, right.Operator)
	})
}

func TestEncodeDecodeExpr_MemberExpr(t *testing.T) {
	t.Run("simple member access", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "user.name",
						Expression: &ast_domain.MemberExpression{
							Base:     &ast_domain.Identifier{Name: "user"},
							Property: &ast_domain.Identifier{Name: "name"},
							Computed: false,
							Optional: false,
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		member, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.MemberExpression)
		require.True(t, ok)
		assert.False(t, member.Computed)
		assert.False(t, member.Optional)

		base, ok := member.Base.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "user", base.Name)

		prop, ok := member.Property.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "name", prop.Name)
	})

	t.Run("optional chaining", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "user?.name",
						Expression: &ast_domain.MemberExpression{
							Base:     &ast_domain.Identifier{Name: "user"},
							Property: &ast_domain.Identifier{Name: "name"},
							Computed: false,
							Optional: true,
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		member, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.MemberExpression)
		require.True(t, ok)
		assert.True(t, member.Optional)
	})

	t.Run("computed member access", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "obj[key]",
						Expression: &ast_domain.MemberExpression{
							Base:     &ast_domain.Identifier{Name: "obj"},
							Property: &ast_domain.Identifier{Name: "key"},
							Computed: true,
							Optional: false,
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		member, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.MemberExpression)
		require.True(t, ok)
		assert.True(t, member.Computed)
	})

	t.Run("deeply nested member access", func(t *testing.T) {

		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "user.profile.settings.theme",
						Expression: &ast_domain.MemberExpression{
							Base: &ast_domain.MemberExpression{
								Base: &ast_domain.MemberExpression{
									Base:     &ast_domain.Identifier{Name: "user"},
									Property: &ast_domain.Identifier{Name: "profile"},
								},
								Property: &ast_domain.Identifier{Name: "settings"},
							},
							Property: &ast_domain.Identifier{Name: "theme"},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		m1, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.MemberExpression)
		require.True(t, ok)
		assertIdentifier(t, m1.Property, "theme")

		m2, ok := m1.Base.(*ast_domain.MemberExpression)
		require.True(t, ok)
		assertIdentifier(t, m2.Property, "settings")

		m3, ok := m2.Base.(*ast_domain.MemberExpression)
		require.True(t, ok)
		assertIdentifier(t, m3.Property, "profile")

		assertIdentifier(t, m3.Base, "user")
	})
}

func TestEncodeDecodeExpr_IndexExpr(t *testing.T) {
	t.Run("array index with integer", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "items[0]",
						Expression: &ast_domain.IndexExpression{
							Base:     &ast_domain.Identifier{Name: "items"},
							Index:    &ast_domain.IntegerLiteral{Value: 0},
							Optional: false,
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		index, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.IndexExpression)
		require.True(t, ok)
		assert.False(t, index.Optional)
		assertIdentifier(t, index.Base, "items")

		lit, ok := index.Index.(*ast_domain.IntegerLiteral)
		require.True(t, ok)
		assert.Equal(t, int64(0), lit.Value)
	})

	t.Run("optional index access", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "items?.[0]",
						Expression: &ast_domain.IndexExpression{
							Base:     &ast_domain.Identifier{Name: "items"},
							Index:    &ast_domain.IntegerLiteral{Value: 0},
							Optional: true,
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		index, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.IndexExpression)
		require.True(t, ok)
		assert.True(t, index.Optional)
	})

	t.Run("dynamic index with variable", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "items[index]",
						Expression: &ast_domain.IndexExpression{
							Base:  &ast_domain.Identifier{Name: "items"},
							Index: &ast_domain.Identifier{Name: "index"},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		index, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.IndexExpression)
		require.True(t, ok)
		assertIdentifier(t, index.Index, "index")
	})
}

func TestEncodeDecodeExpr_CallExpr(t *testing.T) {
	t.Run("function call with no arguments", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "getData()",
						Expression: &ast_domain.CallExpression{
							Callee: &ast_domain.Identifier{Name: "getData"},
							Args:   []ast_domain.Expression{},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		call, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.CallExpression)
		require.True(t, ok)
		assertIdentifier(t, call.Callee, "getData")
		assert.Empty(t, call.Args)
	})

	t.Run("function call with multiple arguments", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "format(value, 'USD')",
						Expression: &ast_domain.CallExpression{
							Callee: &ast_domain.Identifier{Name: "format"},
							Args: []ast_domain.Expression{
								&ast_domain.Identifier{Name: "value"},
								&ast_domain.StringLiteral{Value: "USD"},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		call, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.CallExpression)
		require.True(t, ok)
		require.Len(t, call.Args, 2)
		assertIdentifier(t, call.Args[0], "value")

		str, ok := call.Args[1].(*ast_domain.StringLiteral)
		require.True(t, ok)
		assert.Equal(t, "USD", str.Value)
	})

	t.Run("method call on member expression", func(t *testing.T) {

		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "user.getName()",
						Expression: &ast_domain.CallExpression{
							Callee: &ast_domain.MemberExpression{
								Base:     &ast_domain.Identifier{Name: "user"},
								Property: &ast_domain.Identifier{Name: "getName"},
							},
							Args: []ast_domain.Expression{},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		call, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.CallExpression)
		require.True(t, ok)

		member, ok := call.Callee.(*ast_domain.MemberExpression)
		require.True(t, ok)
		assertIdentifier(t, member.Base, "user")
		assertIdentifier(t, member.Property, "getName")
	})
}

func TestEncodeDecodeExpr_TernaryExpr(t *testing.T) {
	t.Run("simple ternary", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "isActive ? 'yes' : 'no'",
						Expression: &ast_domain.TernaryExpression{
							Condition:  &ast_domain.Identifier{Name: "isActive"},
							Consequent: &ast_domain.StringLiteral{Value: "yes"},
							Alternate:  &ast_domain.StringLiteral{Value: "no"},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		ternary, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.TernaryExpression)
		require.True(t, ok)
		assertIdentifier(t, ternary.Condition, "isActive")

		cons, ok := ternary.Consequent.(*ast_domain.StringLiteral)
		require.True(t, ok)
		assert.Equal(t, "yes", cons.Value)

		alt, ok := ternary.Alternate.(*ast_domain.StringLiteral)
		require.True(t, ok)
		assert.Equal(t, "no", alt.Value)
	})

	t.Run("nested ternary expressions", func(t *testing.T) {

		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "a ? b : (c ? d : e)",
						Expression: &ast_domain.TernaryExpression{
							Condition:  &ast_domain.Identifier{Name: "a"},
							Consequent: &ast_domain.Identifier{Name: "b"},
							Alternate: &ast_domain.TernaryExpression{
								Condition:  &ast_domain.Identifier{Name: "c"},
								Consequent: &ast_domain.Identifier{Name: "d"},
								Alternate:  &ast_domain.Identifier{Name: "e"},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		outer, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.TernaryExpression)
		require.True(t, ok)
		assertIdentifier(t, outer.Condition, "a")
		assertIdentifier(t, outer.Consequent, "b")

		inner, ok := outer.Alternate.(*ast_domain.TernaryExpression)
		require.True(t, ok)
		assertIdentifier(t, inner.Condition, "c")
		assertIdentifier(t, inner.Consequent, "d")
		assertIdentifier(t, inner.Alternate, "e")
	})
}

func TestEncodeDecodeExpr_ForInExpr(t *testing.T) {
	t.Run("simple for-in with value only", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "li",
					DirFor: &ast_domain.Directive{
						Type:          ast_domain.DirectiveFor,
						RawExpression: "item in items",
						Expression: &ast_domain.ForInExpression{
							ItemVariable: &ast_domain.Identifier{Name: "item"},
							Collection:   &ast_domain.Identifier{Name: "items"},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		forIn, ok := decoded.RootNodes[0].DirFor.Expression.(*ast_domain.ForInExpression)
		require.True(t, ok)
		assertIdentifier(t, forIn.ItemVariable, "item")
		assertIdentifier(t, forIn.Collection, "items")
		assert.Nil(t, forIn.IndexVariable)
	})

	t.Run("for-in with index and value", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "li",
					DirFor: &ast_domain.Directive{
						Type:          ast_domain.DirectiveFor,
						RawExpression: "(index, item) in items",
						Expression: &ast_domain.ForInExpression{
							IndexVariable: &ast_domain.Identifier{Name: "index"},
							ItemVariable:  &ast_domain.Identifier{Name: "item"},
							Collection:    &ast_domain.Identifier{Name: "items"},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		forIn, ok := decoded.RootNodes[0].DirFor.Expression.(*ast_domain.ForInExpression)
		require.True(t, ok)
		require.NotNil(t, forIn.IndexVariable)
		assertIdentifier(t, forIn.IndexVariable, "index")
		assertIdentifier(t, forIn.ItemVariable, "item")
		assertIdentifier(t, forIn.Collection, "items")
	})

	t.Run("for-in with member expression collection", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "li",
					DirFor: &ast_domain.Directive{
						Type:          ast_domain.DirectiveFor,
						RawExpression: "item in user.items",
						Expression: &ast_domain.ForInExpression{
							ItemVariable: &ast_domain.Identifier{Name: "item"},
							Collection: &ast_domain.MemberExpression{
								Base:     &ast_domain.Identifier{Name: "user"},
								Property: &ast_domain.Identifier{Name: "items"},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		forIn, ok := decoded.RootNodes[0].DirFor.Expression.(*ast_domain.ForInExpression)
		require.True(t, ok)

		member, ok := forIn.Collection.(*ast_domain.MemberExpression)
		require.True(t, ok)
		assertIdentifier(t, member.Base, "user")
	})
}

func TestEncodeDecodeExpr_ComplexCombined(t *testing.T) {
	t.Run("method call in ternary", func(t *testing.T) {

		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "items.length > 0 ? items[0].name : 'empty'",
						Expression: &ast_domain.TernaryExpression{
							Condition: &ast_domain.BinaryExpression{
								Operator: ast_domain.OpGt,
								Left: &ast_domain.MemberExpression{
									Base:     &ast_domain.Identifier{Name: "items"},
									Property: &ast_domain.Identifier{Name: "length"},
								},
								Right: &ast_domain.IntegerLiteral{Value: 0},
							},
							Consequent: &ast_domain.MemberExpression{
								Base: &ast_domain.IndexExpression{
									Base:  &ast_domain.Identifier{Name: "items"},
									Index: &ast_domain.IntegerLiteral{Value: 0},
								},
								Property: &ast_domain.Identifier{Name: "name"},
							},
							Alternate: &ast_domain.StringLiteral{Value: "empty"},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		ternary, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.TernaryExpression)
		require.True(t, ok)

		condition, ok := ternary.Condition.(*ast_domain.BinaryExpression)
		require.True(t, ok)
		assert.Equal(t, ast_domain.OpGt, condition.Operator)

		consExpr, ok := ternary.Consequent.(*ast_domain.MemberExpression)
		require.True(t, ok)

		indexExpr, ok := consExpr.Base.(*ast_domain.IndexExpression)
		require.True(t, ok)
		assertIdentifier(t, indexExpr.Base, "items")

		alt, ok := ternary.Alternate.(*ast_domain.StringLiteral)
		require.True(t, ok)
		assert.Equal(t, "empty", alt.Value)
	})

	t.Run("chained optional calls", func(t *testing.T) {

		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "user?.profile?.name",
						Expression: &ast_domain.MemberExpression{
							Base: &ast_domain.MemberExpression{
								Base:     &ast_domain.Identifier{Name: "user"},
								Property: &ast_domain.Identifier{Name: "profile"},
								Optional: true,
							},
							Property: &ast_domain.Identifier{Name: "name"},
							Optional: true,
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		outer, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.MemberExpression)
		require.True(t, ok)
		assert.True(t, outer.Optional)

		inner, ok := outer.Base.(*ast_domain.MemberExpression)
		require.True(t, ok)
		assert.True(t, inner.Optional)
	})
}

func TestEncodeDecodeExpr_LocationPreserved(t *testing.T) {
	t.Run("member expression preserves location", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "user.name",
						Expression: &ast_domain.MemberExpression{
							Base:     &ast_domain.Identifier{Name: "user"},
							Property: &ast_domain.Identifier{Name: "name"},
							RelativeLocation: ast_domain.Location{
								Line:   5,
								Column: 10,
								Offset: 50,
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		member, ok := decoded.RootNodes[0].DirText.Expression.(*ast_domain.MemberExpression)
		require.True(t, ok)
		assert.Equal(t, 5, member.RelativeLocation.Line)
		assert.Equal(t, 10, member.RelativeLocation.Column)
		assert.Equal(t, 50, member.RelativeLocation.Offset)
	})

	t.Run("binary expression preserves location", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirIf: &ast_domain.Directive{
						Type:          ast_domain.DirectiveIf,
						RawExpression: "a > b",
						Expression: &ast_domain.BinaryExpression{
							Operator: ast_domain.OpGt,
							Left:     &ast_domain.Identifier{Name: "a"},
							Right:    &ast_domain.Identifier{Name: "b"},
							RelativeLocation: ast_domain.Location{
								Line:   3,
								Column: 8,
								Offset: 30,
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		binary, ok := decoded.RootNodes[0].DirIf.Expression.(*ast_domain.BinaryExpression)
		require.True(t, ok)
		assert.Equal(t, 3, binary.RelativeLocation.Line)
		assert.Equal(t, 8, binary.RelativeLocation.Column)
		assert.Equal(t, 30, binary.RelativeLocation.Offset)
	})
}

func assertIdentifier(t *testing.T, expression ast_domain.Expression, expectedName string) {
	t.Helper()
	identifier, ok := expression.(*ast_domain.Identifier)
	require.True(t, ok, "expected Identifier, got %T", expression)
	assert.Equal(t, expectedName, identifier.Name)
}
