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

package annotator_domain

import (
	"context"
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestIsPartialInvocationNode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node *ast_domain.TemplateNode
		name string
		want bool
	}{
		{
			name: "nil node returns false",
			node: nil,
			want: false,
		},
		{
			name: "node with nil GoAnnotations returns false",
			node: &ast_domain.TemplateNode{},
			want: false,
		},
		{
			name: "node with GoAnnotations but nil PartialInfo returns false",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{},
			},
			want: false,
		},
		{
			name: "node with PartialInfo returns true",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialAlias:       "child",
						PartialPackageName: "pkg/child",
					},
				},
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isPartialInvocationNode(tc.node)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsPropUsage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		want       bool
	}{
		{
			name:       "plain identifier named props returns true",
			expression: &ast_domain.Identifier{Name: "props"},
			want:       true,
		},
		{
			name: "member expression with props root returns true",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "props"},
				Property: &ast_domain.Identifier{Name: "Title"},
			},
			want: true,
		},
		{
			name:       "identifier named state returns false",
			expression: &ast_domain.Identifier{Name: "state"},
			want:       false,
		},
		{
			name: "member expression with non-props root returns false",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "state"},
				Property: &ast_domain.Identifier{Name: "Count"},
			},
			want: false,
		},
		{
			name: "nested member expression with props root returns true",
			expression: &ast_domain.MemberExpression{
				Base: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "props"},
					Property: &ast_domain.Identifier{Name: "User"},
				},
				Property: &ast_domain.Identifier{Name: "Name"},
			},
			want: true,
		},
		{
			name:       "string literal returns false",
			expression: &ast_domain.StringLiteral{Value: "props"},
			want:       false,
		},
		{
			name: "index expression with props root returns true",
			expression: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "props"},
				Index: &ast_domain.IntegerLiteral{Value: 0},
			},
			want: true,
		},
		{
			name: "call expression with props callee returns true",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "props"},
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isPropUsage(tc.expression)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetPropertyNameFromMemberExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		want       string
	}{
		{
			name: "member expression with identifier property returns name",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "props"},
				Property: &ast_domain.Identifier{Name: "Title"},
			},
			want: "Title",
		},
		{
			name: "member expression with non-identifier property returns empty",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "props"},
				Property: &ast_domain.IntegerLiteral{Value: 0},
			},
			want: "",
		},
		{
			name:       "non-member expression returns empty",
			expression: &ast_domain.Identifier{Name: "props"},
			want:       "",
		},
		{
			name:       "string literal returns empty",
			expression: &ast_domain.StringLiteral{Value: "hello"},
			want:       "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := getPropertyNameFromMemberExpr(tc.expression)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetRootIdentifier(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		wantName   string
		wantOk     bool
	}{
		{
			name:       "simple identifier returns itself",
			expression: &ast_domain.Identifier{Name: "user"},
			wantName:   "user",
			wantOk:     true,
		},
		{
			name: "member expression returns base identifier",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "state"},
				Property: &ast_domain.Identifier{Name: "Count"},
			},
			wantName: "state",
			wantOk:   true,
		},
		{
			name: "nested member expression returns root identifier",
			expression: &ast_domain.MemberExpression{
				Base: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "props"},
					Property: &ast_domain.Identifier{Name: "User"},
				},
				Property: &ast_domain.Identifier{Name: "Name"},
			},
			wantName: "props",
			wantOk:   true,
		},
		{
			name: "index expression returns base identifier",
			expression: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "items"},
				Index: &ast_domain.IntegerLiteral{Value: 0},
			},
			wantName: "items",
			wantOk:   true,
		},
		{
			name: "call expression returns callee identifier",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "getUser"},
			},
			wantName: "getUser",
			wantOk:   true,
		},
		{
			name: "chained call on member returns root identifier",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "service"},
					Property: &ast_domain.Identifier{Name: "GetUser"},
				},
			},
			wantName: "service",
			wantOk:   true,
		},
		{
			name:       "string literal returns false",
			expression: &ast_domain.StringLiteral{Value: "hello"},
			wantName:   "",
			wantOk:     false,
		},
		{
			name: "binary expression returns false",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.IntegerLiteral{Value: 1},
				Operator: ast_domain.OpPlus,
				Right:    &ast_domain.IntegerLiteral{Value: 2},
			},
			wantName: "",
			wantOk:   false,
		},
		{
			name: "unary expression returns false",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right:    &ast_domain.BooleanLiteral{Value: true},
			},
			wantName: "",
			wantOk:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := getRootIdentifier(tc.expression)
			assert.Equal(t, tc.wantOk, ok)
			if tc.wantOk {
				require.NotNil(t, got)
				assert.Equal(t, tc.wantName, got.Name)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestFindInvokerNodeForComponentHash(t *testing.T) {
	t.Parallel()

	t.Run("returns matching node when hash is found", func(t *testing.T) {
		t.Parallel()
		targetNode := &ast_domain.TemplateNode{TagName: "child-component"}
		invocationMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{
			targetNode: {
				PartialPackageName: "pkg/child",
				PartialAlias:       "child",
			},
		}

		got := findInvokerNodeForComponentHash("pkg/child", invocationMap)
		assert.Same(t, targetNode, got)
	})

	t.Run("returns nil when hash is not found", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{TagName: "other-component"}
		invocationMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{
			node: {
				PartialPackageName: "pkg/other",
				PartialAlias:       "other",
			},
		}

		got := findInvokerNodeForComponentHash("pkg/nonexistent", invocationMap)
		assert.Nil(t, got)
	})

	t.Run("returns nil for empty map", func(t *testing.T) {
		t.Parallel()
		invocationMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{}
		got := findInvokerNodeForComponentHash("pkg/child", invocationMap)
		assert.Nil(t, got)
	})

	t.Run("returns correct node from multiple entries", func(t *testing.T) {
		t.Parallel()
		nodeA := &ast_domain.TemplateNode{TagName: "comp-a"}
		nodeB := &ast_domain.TemplateNode{TagName: "comp-b"}
		nodeC := &ast_domain.TemplateNode{TagName: "comp-c"}

		invocationMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{
			nodeA: {PartialPackageName: "pkg/a"},
			nodeB: {PartialPackageName: "pkg/b"},
			nodeC: {PartialPackageName: "pkg/c"},
		}

		got := findInvokerNodeForComponentHash("pkg/b", invocationMap)
		assert.Same(t, nodeB, got)
	})
}

func TestVisitExpression(t *testing.T) {
	t.Parallel()

	t.Run("nil expression does not call visitor", func(t *testing.T) {
		t.Parallel()
		callCount := 0
		visitExpression(nil, func(_ ast_domain.Expression) bool {
			callCount++
			return true
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("simple identifier visits once", func(t *testing.T) {
		t.Parallel()
		identifier := &ast_domain.Identifier{Name: "x"}
		var visited []string
		visitExpression(identifier, func(expression ast_domain.Expression) bool {
			if id, ok := expression.(*ast_domain.Identifier); ok {
				visited = append(visited, id.Name)
			}
			return true
		})
		assert.Equal(t, []string{"x"}, visited)
	})

	t.Run("member expression visits base and property", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "user"},
			Property: &ast_domain.Identifier{Name: "name"},
		}
		var visited []string
		visitExpression(expression, func(e ast_domain.Expression) bool {
			visited = append(visited, e.String())
			return true
		})

		assert.Contains(t, visited, "user.name")
		assert.Contains(t, visited, "user")
		assert.Contains(t, visited, "name")
		assert.Len(t, visited, 3)
	})

	t.Run("index expression visits base and index", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.IndexExpression{
			Base:  &ast_domain.Identifier{Name: "items"},
			Index: &ast_domain.IntegerLiteral{Value: 0},
		}
		callCount := 0
		visitExpression(expression, func(_ ast_domain.Expression) bool {
			callCount++
			return true
		})

		assert.Equal(t, 3, callCount)
	})

	t.Run("unary expression visits operand", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.UnaryExpression{
			Operator: ast_domain.OpNot,
			Right:    &ast_domain.Identifier{Name: "active"},
		}
		var visited []string
		visitExpression(expression, func(e ast_domain.Expression) bool {
			visited = append(visited, e.String())
			return true
		})
		assert.Contains(t, visited, "!active")
		assert.Contains(t, visited, "active")
	})

	t.Run("binary expression visits both sides", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.BinaryExpression{
			Left:     &ast_domain.Identifier{Name: "a"},
			Operator: ast_domain.OpPlus,
			Right:    &ast_domain.Identifier{Name: "b"},
		}
		var visited []string
		visitExpression(expression, func(e ast_domain.Expression) bool {
			if id, ok := e.(*ast_domain.Identifier); ok {
				visited = append(visited, id.Name)
			}
			return true
		})
		assert.Contains(t, visited, "a")
		assert.Contains(t, visited, "b")
	})

	t.Run("call expression visits callee and arguments", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "format"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "x"},
				&ast_domain.Identifier{Name: "y"},
			},
		}
		var identifiers []string
		visitExpression(expression, func(e ast_domain.Expression) bool {
			if id, ok := e.(*ast_domain.Identifier); ok {
				identifiers = append(identifiers, id.Name)
			}
			return true
		})
		assert.Contains(t, identifiers, "format")
		assert.Contains(t, identifiers, "x")
		assert.Contains(t, identifiers, "y")
	})

	t.Run("visitor returning false stops descent", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "user"},
			Property: &ast_domain.Identifier{Name: "name"},
		}
		callCount := 0
		visitExpression(expression, func(_ ast_domain.Expression) bool {
			callCount++
			return false
		})
		assert.Equal(t, 1, callCount)
	})
}

func TestVisitCompositeLiteralExpr(t *testing.T) {
	t.Parallel()

	t.Run("template literal visits non-literal parts", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "Hello, "},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "name"}},
				{IsLiteral: true, Literal: "!"},
			},
		}
		var identifiers []string
		visitCompositeLiteralExpression(expression, func(e ast_domain.Expression) bool {
			if id, ok := e.(*ast_domain.Identifier); ok {
				identifiers = append(identifiers, id.Name)
			}
			return true
		})
		assert.Equal(t, []string{"name"}, identifiers)
	})

	t.Run("template literal skips literal parts", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "static text"},
			},
		}
		callCount := 0
		visitCompositeLiteralExpression(expression, func(_ ast_domain.Expression) bool {
			callCount++
			return true
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("object literal visits all values", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.ObjectLiteral{
			Pairs: map[string]ast_domain.Expression{
				"colour": &ast_domain.Identifier{Name: "red"},
				"size":   &ast_domain.Identifier{Name: "large"},
			},
		}
		var identifiers []string
		visitCompositeLiteralExpression(expression, func(e ast_domain.Expression) bool {
			if id, ok := e.(*ast_domain.Identifier); ok {
				identifiers = append(identifiers, id.Name)
			}
			return true
		})
		assert.Len(t, identifiers, 2)
		assert.Contains(t, identifiers, "red")
		assert.Contains(t, identifiers, "large")
	})

	t.Run("array literal visits all elements", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.ArrayLiteral{
			Elements: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "alpha"},
				&ast_domain.Identifier{Name: "beta"},
				&ast_domain.Identifier{Name: "gamma"},
			},
		}
		var identifiers []string
		visitCompositeLiteralExpression(expression, func(e ast_domain.Expression) bool {
			if id, ok := e.(*ast_domain.Identifier); ok {
				identifiers = append(identifiers, id.Name)
			}
			return true
		})
		assert.Equal(t, []string{"alpha", "beta", "gamma"}, identifiers)
	})

	t.Run("non-composite expression does nothing", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.Identifier{Name: "plain"}
		callCount := 0
		visitCompositeLiteralExpression(expression, func(_ ast_domain.Expression) bool {
			callCount++
			return true
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("empty object literal visits nothing", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.ObjectLiteral{
			Pairs: map[string]ast_domain.Expression{},
		}
		callCount := 0
		visitCompositeLiteralExpression(expression, func(_ ast_domain.Expression) bool {
			callCount++
			return true
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("empty array literal visits nothing", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.ArrayLiteral{
			Elements: []ast_domain.Expression{},
		}
		callCount := 0
		visitCompositeLiteralExpression(expression, func(_ ast_domain.Expression) bool {
			callCount++
			return true
		})
		assert.Equal(t, 0, callCount)
	})
}

func TestVisitControlFlowExpr(t *testing.T) {
	t.Parallel()

	t.Run("ternary expression visits all three branches", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.TernaryExpression{
			Condition:  &ast_domain.Identifier{Name: "cond"},
			Consequent: &ast_domain.Identifier{Name: "yes"},
			Alternate:  &ast_domain.Identifier{Name: "no"},
		}
		var identifiers []string
		visitControlFlowExpression(expression, func(e ast_domain.Expression) bool {
			if id, ok := e.(*ast_domain.Identifier); ok {
				identifiers = append(identifiers, id.Name)
			}
			return true
		})
		assert.Equal(t, []string{"cond", "yes", "no"}, identifiers)
	})

	t.Run("non-ternary expression does nothing", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.Identifier{Name: "x"}
		callCount := 0
		visitControlFlowExpression(expression, func(_ ast_domain.Expression) bool {
			callCount++
			return true
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("binary expression does nothing", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.BinaryExpression{
			Left:     &ast_domain.IntegerLiteral{Value: 1},
			Operator: ast_domain.OpPlus,
			Right:    &ast_domain.IntegerLiteral{Value: 2},
		}
		callCount := 0
		visitControlFlowExpression(expression, func(_ ast_domain.Expression) bool {
			callCount++
			return true
		})
		assert.Equal(t, 0, callCount)
	})
}

func TestVisitStructuralDirectives(t *testing.T) {
	t.Parallel()

	t.Run("visits DirIf expression", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirIf: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "visible"},
			},
		}
		var visited []string
		visitStructuralDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"visible"}, visited)
	})

	t.Run("visits DirElseIf expression", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirElseIf: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "altCond"},
			},
		}
		var visited []string
		visitStructuralDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"altCond"}, visited)
	})

	t.Run("visits DirFor expression", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirFor: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "collection"},
			},
		}
		var visited []string
		visitStructuralDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"collection"}, visited)
	})

	t.Run("visits DirShow expression", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirShow: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "shown"},
			},
		}
		var visited []string
		visitStructuralDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"shown"}, visited)
	})

	t.Run("visits all structural directives when present", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirIf:     &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "a"}},
			DirElseIf: &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "b"}},
			DirFor:    &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "c"}},
			DirShow:   &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "d"}},
		}
		var visited []string
		visitStructuralDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"a", "b", "c", "d"}, visited)
	})

	t.Run("skips nil directives", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{}
		callCount := 0
		visitStructuralDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("skips directives with nil expressions", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirIf:  &ast_domain.Directive{Expression: nil},
			DirFor: &ast_domain.Directive{Expression: nil},
		}
		callCount := 0
		visitStructuralDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})
}

func TestVisitContentDirectives(t *testing.T) {
	t.Parallel()

	t.Run("visits DirText expression", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirText: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "content"},
			},
		}
		var visited []string
		visitContentDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"content"}, visited)
	})

	t.Run("visits DirHTML expression", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirHTML: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "rawHTML"},
			},
		}
		var visited []string
		visitContentDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"rawHTML"}, visited)
	})

	t.Run("visits all content directives when present", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirText: &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "text"}},
			DirHTML: &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "html"}},
		}
		var visited []string
		visitContentDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"text", "html"}, visited)
	})

	t.Run("skips nil and nil-expression directives", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirText: &ast_domain.Directive{Expression: nil},
		}
		callCount := 0
		visitContentDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})
}

func TestVisitModelDirective(t *testing.T) {
	t.Parallel()

	t.Run("visits model expression when present", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirModel: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "formValue"},
			},
		}
		var visited []string
		visitModelDirective(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"formValue"}, visited)
	})

	t.Run("skips nil model directive", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{}
		callCount := 0
		visitModelDirective(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("skips model directive with nil expression", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirModel: &ast_domain.Directive{Expression: nil},
		}
		callCount := 0
		visitModelDirective(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})
}

func TestVisitBindDirectives(t *testing.T) {
	t.Parallel()

	t.Run("visits all bind expressions", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"title": {Expression: &ast_domain.Identifier{Name: "pageTitle"}},
				"href":  {Expression: &ast_domain.Identifier{Name: "link"}},
			},
		}
		var visited []string
		visitBindDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Len(t, visited, 2)
		assert.Contains(t, visited, "pageTitle")
		assert.Contains(t, visited, "link")
	})

	t.Run("skips nil bind entries", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"title": nil,
			},
		}
		callCount := 0
		visitBindDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("skips bind entries with nil expressions", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"title": {Expression: nil},
			},
		}
		callCount := 0
		visitBindDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("empty binds visits nothing", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{},
		}
		callCount := 0
		visitBindDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("nil binds map visits nothing", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{}
		callCount := 0
		visitBindDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})
}

func TestVisitEventDirectives(t *testing.T) {
	t.Parallel()

	t.Run("visits all event handler expressions", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			OnEvents: map[string][]ast_domain.Directive{
				"click": {
					{Expression: &ast_domain.Identifier{Name: "handleClick"}},
				},
				"submit": {
					{Expression: &ast_domain.Identifier{Name: "handleSubmit"}},
				},
			},
		}
		var visited []string
		visitEventDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Len(t, visited, 2)
		assert.Contains(t, visited, "handleClick")
		assert.Contains(t, visited, "handleSubmit")
	})

	t.Run("visits multiple handlers for same event", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			OnEvents: map[string][]ast_domain.Directive{
				"click": {
					{Expression: &ast_domain.Identifier{Name: "handlerA"}},
					{Expression: &ast_domain.Identifier{Name: "handlerB"}},
				},
			},
		}
		var visited []string
		visitEventDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Len(t, visited, 2)
		assert.Contains(t, visited, "handlerA")
		assert.Contains(t, visited, "handlerB")
	})

	t.Run("skips events with nil expressions", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			OnEvents: map[string][]ast_domain.Directive{
				"click": {
					{Expression: nil},
				},
			},
		}
		callCount := 0
		visitEventDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("empty OnEvents visits nothing", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			OnEvents: map[string][]ast_domain.Directive{},
		}
		callCount := 0
		visitEventDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("nil OnEvents visits nothing", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{}
		callCount := 0
		visitEventDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})
}

func TestVisitCustomEventDirectives(t *testing.T) {
	t.Parallel()

	t.Run("visits all custom event expressions", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			CustomEvents: map[string][]ast_domain.Directive{
				"update": {
					{Expression: &ast_domain.Identifier{Name: "onUpdate"}},
				},
			},
		}
		var visited []string
		visitCustomEventDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"onUpdate"}, visited)
	})

	t.Run("visits multiple handlers for same custom event", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			CustomEvents: map[string][]ast_domain.Directive{
				"change": {
					{Expression: &ast_domain.Identifier{Name: "first"}},
					{Expression: &ast_domain.Identifier{Name: "second"}},
				},
			},
		}
		var visited []string
		visitCustomEventDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Len(t, visited, 2)
		assert.Contains(t, visited, "first")
		assert.Contains(t, visited, "second")
	})

	t.Run("skips custom events with nil expressions", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			CustomEvents: map[string][]ast_domain.Directive{
				"change": {
					{Expression: nil},
				},
			},
		}
		callCount := 0
		visitCustomEventDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("nil CustomEvents visits nothing", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{}
		callCount := 0
		visitCustomEventDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})
}

func TestVisitAttributeLikeExpressions(t *testing.T) {
	t.Parallel()

	t.Run("visits DirClass expression", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirClass: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "classes"},
			},
		}
		var visited []string
		visitAttributeLikeExpressions(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"classes"}, visited)
	})

	t.Run("visits DirStyle expression", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirStyle: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "styles"},
			},
		}
		var visited []string
		visitAttributeLikeExpressions(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"styles"}, visited)
	})

	t.Run("visits dynamic attribute expressions", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", Expression: &ast_domain.Identifier{Name: "pageTitle"}},
				{Name: "href", Expression: &ast_domain.Identifier{Name: "link"}},
			},
		}
		var visited []string
		visitAttributeLikeExpressions(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"pageTitle", "link"}, visited)
	})

	t.Run("skips dynamic attributes with nil expressions", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", Expression: nil},
			},
		}
		callCount := 0
		visitAttributeLikeExpressions(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("visits non-literal rich text expressions", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			RichText: []ast_domain.TextPart{
				{IsLiteral: true, Literal: "Hello "},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "userName"}},
				{IsLiteral: true, Literal: "!"},
			},
		}
		var visited []string
		visitAttributeLikeExpressions(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"userName"}, visited)
	})

	t.Run("skips literal rich text parts", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			RichText: []ast_domain.TextPart{
				{IsLiteral: true, Literal: "static text"},
			},
		}
		callCount := 0
		visitAttributeLikeExpressions(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("skips non-literal rich text with nil expression", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			RichText: []ast_domain.TextPart{
				{IsLiteral: false, Expression: nil},
			},
		}
		callCount := 0
		visitAttributeLikeExpressions(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("skips nil DirClass and DirStyle", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{}
		callCount := 0
		visitAttributeLikeExpressions(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("visits everything combined", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirClass: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "cls"},
			},
			DirStyle: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "sty"},
			},
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", Expression: &ast_domain.Identifier{Name: "ttl"}},
			},
			RichText: []ast_domain.TextPart{
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "txt"}},
			},
		}
		var visited []string
		visitAttributeLikeExpressions(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Equal(t, []string{"cls", "sty", "ttl", "txt"}, visited)
	})
}

func TestVisitBindingDirectives(t *testing.T) {
	t.Parallel()

	t.Run("visits model, bind, on, and custom event directives", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			DirModel: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "model"},
			},
			Binds: map[string]*ast_domain.Directive{
				"value": {Expression: &ast_domain.Identifier{Name: "val"}},
			},
			OnEvents: map[string][]ast_domain.Directive{
				"click": {{Expression: &ast_domain.Identifier{Name: "onClick"}}},
			},
			CustomEvents: map[string][]ast_domain.Directive{
				"update": {{Expression: &ast_domain.Identifier{Name: "onUpdate"}}},
			},
		}
		var visited []string
		visitBindingDirectives(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Len(t, visited, 4)
		assert.Contains(t, visited, "model")
		assert.Contains(t, visited, "val")
		assert.Contains(t, visited, "onClick")
		assert.Contains(t, visited, "onUpdate")
	})

	t.Run("empty node visits nothing", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{}
		callCount := 0
		visitBindingDirectives(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})
}

func TestVisitNodeExpressions(t *testing.T) {
	t.Parallel()

	t.Run("visits structural, content, binding, and attribute expressions", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{

			DirIf: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "ifCond"},
			},

			DirText: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "textVal"},
			},

			DirModel: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "modelVal"},
			},

			DirClass: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "classVal"},
			},
		}
		var visited []string
		visitNodeExpressions(node, func(e ast_domain.Expression) {
			visited = append(visited, e.String())
		})
		assert.Len(t, visited, 4)
		assert.Contains(t, visited, "ifCond")
		assert.Contains(t, visited, "textVal")
		assert.Contains(t, visited, "modelVal")
		assert.Contains(t, visited, "classVal")
	})

	t.Run("empty node visits nothing", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{}
		callCount := 0
		visitNodeExpressions(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})
}

func TestVisitExpression_CompositeAndControlFlow(t *testing.T) {
	t.Parallel()

	t.Run("ternary expression visits condition, consequent, and alternate", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.TernaryExpression{
			Condition:  &ast_domain.Identifier{Name: "isActive"},
			Consequent: &ast_domain.StringLiteral{Value: "yes"},
			Alternate:  &ast_domain.StringLiteral{Value: "no"},
		}
		var identifiers []string
		visitExpression(expression, func(e ast_domain.Expression) bool {
			if id, ok := e.(*ast_domain.Identifier); ok {
				identifiers = append(identifiers, id.Name)
			}
			return true
		})
		assert.Equal(t, []string{"isActive"}, identifiers)
	})

	t.Run("template literal within visitExpression reaches nested expressions", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "Count: "},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "count"}},
			},
		}
		var identifiers []string
		visitExpression(expression, func(e ast_domain.Expression) bool {
			if id, ok := e.(*ast_domain.Identifier); ok {
				identifiers = append(identifiers, id.Name)
			}
			return true
		})
		assert.Equal(t, []string{"count"}, identifiers)
	})

	t.Run("object literal within visitExpression reaches values", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.ObjectLiteral{
			Pairs: map[string]ast_domain.Expression{
				"key": &ast_domain.Identifier{Name: "val"},
			},
		}
		var identifiers []string
		visitExpression(expression, func(e ast_domain.Expression) bool {
			if id, ok := e.(*ast_domain.Identifier); ok {
				identifiers = append(identifiers, id.Name)
			}
			return true
		})
		assert.Equal(t, []string{"val"}, identifiers)
	})

	t.Run("array literal within visitExpression reaches elements", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.ArrayLiteral{
			Elements: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "first"},
				&ast_domain.Identifier{Name: "second"},
			},
		}
		var identifiers []string
		visitExpression(expression, func(e ast_domain.Expression) bool {
			if id, ok := e.(*ast_domain.Identifier); ok {
				identifiers = append(identifiers, id.Name)
			}
			return true
		})
		assert.Equal(t, []string{"first", "second"}, identifiers)
	})

	t.Run("deeply nested expression tree is fully visited", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "format"},
			Args: []ast_domain.Expression{
				&ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "props"},
					Property: &ast_domain.Identifier{Name: "Title"},
				},
				&ast_domain.BinaryExpression{
					Left: &ast_domain.IndexExpression{
						Base:  &ast_domain.Identifier{Name: "items"},
						Index: &ast_domain.IntegerLiteral{Value: 0},
					},
					Operator: ast_domain.OpPlus,
					Right:    &ast_domain.Identifier{Name: "count"},
				},
			},
		}
		var identifiers []string
		visitExpression(expression, func(e ast_domain.Expression) bool {
			if id, ok := e.(*ast_domain.Identifier); ok {
				identifiers = append(identifiers, id.Name)
			}
			return true
		})
		assert.Contains(t, identifiers, "format")
		assert.Contains(t, identifiers, "props")
		assert.Contains(t, identifiers, "Title")
		assert.Contains(t, identifiers, "items")
		assert.Contains(t, identifiers, "count")
		assert.Len(t, identifiers, 5)
	})
}

func TestLinkAllPropDataSources_NilTree(t *testing.T) {
	t.Parallel()
	result := LinkAllPropDataSources(context.Background(), nil, nil, nil)
	assert.Nil(t, result)
}

func TestPdsSetterVisitor_Enter(t *testing.T) {
	t.Parallel()

	t.Run("increments depth and returns self", func(t *testing.T) {
		t.Parallel()

		v := &pdsSetterVisitor{
			diagnostics: new([]*ast_domain.Diagnostic),
			depth:       0,
		}

		node := &ast_domain.TemplateNode{TagName: "div"}
		visitor, err := v.Enter(context.Background(), node)

		assert.NoError(t, err)
		assert.Same(t, v, visitor)
		assert.Equal(t, 1, v.depth)
	})
}

func TestPdsSetterVisitor_Exit_NonPartialNode(t *testing.T) {
	t.Parallel()

	v := &pdsSetterVisitor{
		diagnostics: new([]*ast_domain.Diagnostic),
		depth:       1,
	}

	node := &ast_domain.TemplateNode{TagName: "div"}
	err := v.Exit(context.Background(), node)

	assert.NoError(t, err)
	assert.Equal(t, 0, v.depth, "depth should decrement")
}

func TestPdsSetterVisitor_Exit_PartialInvocationNode_SetsDataSource(t *testing.T) {
	t.Parallel()

	v := &pdsSetterVisitor{
		diagnostics: new([]*ast_domain.Diagnostic),
		depth:       1,
	}

	propExpr := &ast_domain.Identifier{Name: "title"}
	propExpr.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		},
		Symbol: nil,
	})

	node := &ast_domain.TemplateNode{
		TagName: "child",
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			PartialInfo: &ast_domain.PartialInvocationInfo{
				PartialAlias:       "child",
				PartialPackageName: "pkg/child",
				InvocationKey:      "inv_001",
				PassedProps: map[string]ast_domain.PropValue{
					"title": {
						Expression: propExpr,
					},
				},
			},
		},
	}

	err := v.Exit(context.Background(), node)

	assert.NoError(t, err)
	assert.Equal(t, 0, v.depth)

	ann := propExpr.GetGoAnnotation()
	require.NotNil(t, ann)
	require.NotNil(t, ann.PropDataSource, "PropDataSource should be set")
	require.NotNil(t, ann.PropDataSource.ResolvedType)
}

func TestPdsSetterVisitor_Exit_SkipsWhenAnnotationNil(t *testing.T) {
	t.Parallel()

	v := &pdsSetterVisitor{
		diagnostics: new([]*ast_domain.Diagnostic),
		depth:       1,
	}

	propExpr := &ast_domain.Identifier{Name: "title"}

	node := &ast_domain.TemplateNode{
		TagName: "child",
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			PartialInfo: &ast_domain.PartialInvocationInfo{
				PartialAlias:       "child",
				PartialPackageName: "pkg/child",
				InvocationKey:      "inv_002",
				PassedProps: map[string]ast_domain.PropValue{
					"title": {
						Expression: propExpr,
					},
				},
			},
		},
	}

	err := v.Exit(context.Background(), node)

	assert.NoError(t, err)
	ann := propExpr.GetGoAnnotation()
	assert.Nil(t, ann, "No annotation should be set when source has no annotation")
}

func TestPdsSetterVisitor_Exit_DoesNotOverwriteExistingDataSource(t *testing.T) {
	t.Parallel()

	v := &pdsSetterVisitor{
		diagnostics: new([]*ast_domain.Diagnostic),
		depth:       1,
	}

	existingPDS := &ast_domain.PropDataSource{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
		},
	}

	propExpr := &ast_domain.Identifier{Name: "count"}
	propExpr.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
		},
		PropDataSource: existingPDS,
	})

	node := &ast_domain.TemplateNode{
		TagName: "child",
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			PartialInfo: &ast_domain.PartialInvocationInfo{
				PartialAlias:       "child",
				PartialPackageName: "pkg/child",
				InvocationKey:      "inv_003",
				PassedProps: map[string]ast_domain.PropValue{
					"count": {
						Expression: propExpr,
					},
				},
			},
		},
	}

	err := v.Exit(context.Background(), node)

	assert.NoError(t, err)
	ann := propExpr.GetGoAnnotation()
	require.NotNil(t, ann)
	assert.Same(t, existingPDS, ann.PropDataSource, "Existing data source should not be overwritten")
}

func TestPdsLinkerVisitor_Enter_NilNode(t *testing.T) {
	t.Parallel()

	v := &pdsLinkerVisitor{
		diagnostics:        new([]*ast_domain.Diagnostic),
		propMapsByHash:     make(map[string]map[string]string),
		invocationByNode:   make(map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo),
		invocationMapStack: nil,
		depth:              0,
	}

	visitor, err := v.Enter(context.Background(), nil)

	assert.NoError(t, err)
	assert.Nil(t, visitor)
	assert.Equal(t, 0, v.depth, "depth should not change for nil node")
}

func TestPdsLinkerVisitor_Enter_SimpleNode(t *testing.T) {
	t.Parallel()

	v := &pdsLinkerVisitor{
		diagnostics:        new([]*ast_domain.Diagnostic),
		propMapsByHash:     make(map[string]map[string]string),
		invocationByNode:   make(map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo),
		invocationMapStack: nil,
		depth:              0,
	}

	node := &ast_domain.TemplateNode{TagName: "div"}
	visitor, err := v.Enter(context.Background(), node)

	assert.NoError(t, err)
	assert.Same(t, v, visitor)
	assert.Equal(t, 1, v.depth)
	assert.Len(t, v.invocationMapStack, 1, "should push to invocation map stack")
}

func TestPdsLinkerVisitor_Exit_RestoresMapAndDepth(t *testing.T) {
	t.Parallel()

	sentinelNode := &ast_domain.TemplateNode{TagName: "sentinel"}
	previousMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{
		sentinelNode: {PartialAlias: "sentinel-marker"},
	}
	currentMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{
		{TagName: "a"}: {PartialAlias: "a"},
	}

	v := &pdsLinkerVisitor{
		diagnostics:        new([]*ast_domain.Diagnostic),
		propMapsByHash:     make(map[string]map[string]string),
		invocationByNode:   currentMap,
		invocationMapStack: []map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{previousMap},
		depth:              1,
	}

	err := v.Exit(context.Background(), &ast_domain.TemplateNode{TagName: "div"})

	assert.NoError(t, err)
	assert.Equal(t, 0, v.depth)

	assert.Contains(t, v.invocationByNode, sentinelNode, "should restore previous invocation map")
	assert.Equal(t, "sentinel-marker", v.invocationByNode[sentinelNode].PartialAlias)
	assert.Empty(t, v.invocationMapStack, "stack should be popped")
}

func TestPdsLinkerVisitor_Enter_Exit_StackIntegrity(t *testing.T) {
	t.Parallel()

	v := &pdsLinkerVisitor{
		diagnostics:        new([]*ast_domain.Diagnostic),
		propMapsByHash:     make(map[string]map[string]string),
		invocationByNode:   make(map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo),
		invocationMapStack: nil,
		depth:              0,
	}

	node1 := &ast_domain.TemplateNode{TagName: "outer"}
	node2 := &ast_domain.TemplateNode{TagName: "inner"}

	_, err := v.Enter(context.Background(), node1)
	require.NoError(t, err)
	assert.Equal(t, 1, v.depth)
	assert.Len(t, v.invocationMapStack, 1)

	_, err = v.Enter(context.Background(), node2)
	require.NoError(t, err)
	assert.Equal(t, 2, v.depth)
	assert.Len(t, v.invocationMapStack, 2)

	err = v.Exit(context.Background(), node2)
	require.NoError(t, err)
	assert.Equal(t, 1, v.depth)
	assert.Len(t, v.invocationMapStack, 1)

	err = v.Exit(context.Background(), node1)
	require.NoError(t, err)
	assert.Equal(t, 0, v.depth)
	assert.Empty(t, v.invocationMapStack)
}

func TestLinkAllPropDataSources_EmptyTree(t *testing.T) {
	t.Parallel()

	tree := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{},
	}
	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
	}

	result := LinkAllPropDataSources(context.Background(), tree, vm, nil)

	assert.Nil(t, result)
}

func TestLinkAllPropDataSources_SimpleNodes(t *testing.T) {
	t.Parallel()

	tree := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{TagName: "div", NodeType: ast_domain.NodeElement},
			{TagName: "span", NodeType: ast_domain.NodeElement},
		},
	}
	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
	}

	result := LinkAllPropDataSources(context.Background(), tree, vm, nil)

	assert.Nil(t, result)
}

func TestDidPropTransform_UnknownComponent(t *testing.T) {
	t.Parallel()

	v := &pdsLinkerVisitor{
		diagnostics:        new([]*ast_domain.Diagnostic),
		propMapsByHash:     make(map[string]map[string]string),
		validPropInfoCache: nil,
		virtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
		},
	}

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "props"},
		Property: &ast_domain.Identifier{Name: "Title"},
	}

	result := v.didPropTransform(context.Background(), "unknown_hash", expression)
	assert.False(t, result, "should return false when component is unknown")
}

func TestFindNextLinkInChain_NilInvokerNode(t *testing.T) {
	t.Parallel()

	v := &pdsLinkerVisitor{
		diagnostics:    new([]*ast_domain.Diagnostic),
		propMapsByHash: make(map[string]map[string]string),
	}

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "props"},
		Property: &ast_domain.Identifier{Name: "Title"},
	}
	pInfo := &ast_domain.PartialInvocationInfo{
		InvokerPackageAlias: "unknown_invoker",
	}
	invMap := make(map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo)

	_, _, ok := v.findNextLinkInChain(context.Background(), expression, pInfo, invMap)
	assert.False(t, ok, "should return false when invoker node is not found")
}

func TestFollowDataSourceChain_NonPropSource(t *testing.T) {
	t.Parallel()

	v := &pdsLinkerVisitor{
		diagnostics:    new([]*ast_domain.Diagnostic),
		propMapsByHash: make(map[string]map[string]string),
	}

	sourceExpression := &ast_domain.Identifier{Name: "title"}
	sourceExpression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
		PropDataSource: &ast_domain.PropDataSource{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		},
	})

	pInfo := &ast_domain.PartialInvocationInfo{PartialPackageName: "pkg/child"}
	invMap := make(map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo)

	result := v.followDataSourceChain(context.Background(), sourceExpression, pInfo, invMap)

	require.NotNil(t, result)
	assert.NotNil(t, result.ResolvedType)
}

func TestFollowDataSourceChain_NilAnnotation(t *testing.T) {
	t.Parallel()

	v := &pdsLinkerVisitor{
		diagnostics:    new([]*ast_domain.Diagnostic),
		propMapsByHash: make(map[string]map[string]string),
	}

	sourceExpression := &ast_domain.Identifier{Name: "value"}

	pInfo := &ast_domain.PartialInvocationInfo{PartialPackageName: "pkg/child"}
	invMap := make(map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo)

	result := v.followDataSourceChain(context.Background(), sourceExpression, pInfo, invMap)
	assert.Nil(t, result)
}

type testFieldDef struct {
	Name       string
	TypeString string
	RawTag     string
}

func buildVirtualComponent(hash, goPackagePath, goFilePath, sourcePath string, fields []testFieldDef) *annotator_dto.VirtualComponent {
	inspFields := make([]*inspector_dto.Field, len(fields))
	for i, f := range fields {
		inspFields[i] = &inspector_dto.Field{
			Name:       f.Name,
			TypeString: f.TypeString,
			RawTag:     f.RawTag,
		}
	}

	return &annotator_dto.VirtualComponent{
		HashedName:             hash,
		CanonicalGoPackagePath: goPackagePath,
		VirtualGoFilePath:      goFilePath,
		Source: &annotator_dto.ParsedComponent{
			SourcePath: sourcePath,
			Script: &annotator_dto.ParsedScript{
				PropsTypeExpression: goast.NewIdent("Props"),
			},
		},
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("pkg"),
		},
	}
}

func buildMockInspector(componentFields map[string][]*inspector_dto.Field) *inspector_domain.MockTypeQuerier {
	return &inspector_domain.MockTypeQuerier{
		GetImportsForFileFunc: func(_, _ string) map[string]string {
			return map[string]string{}
		},
		ResolveExprToNamedTypeFunc: func(_ goast.Expr, importerPackagePath, _ string) (*inspector_dto.Type, string) {
			if fields, ok := componentFields[importerPackagePath]; ok {
				return &inspector_dto.Type{
					Name:        "Props",
					PackagePath: importerPackagePath,
					Fields:      fields,
				}, importerPackagePath
			}
			return nil, ""
		},
	}
}

func TestDidPropTransform(t *testing.T) {
	t.Parallel()

	const compHash = "pkg/child"
	const compGoPackage = "example.com/child"
	const compGoFile = "/tmp/child.go"
	const compSource = "/tmp/child.pk"

	makeVisitor := func(fields []testFieldDef) *pdsLinkerVisitor {
		vc := buildVirtualComponent(compHash, compGoPackage, compGoFile, compSource, fields)

		inspFields := make([]*inspector_dto.Field, len(fields))
		for i, f := range fields {
			inspFields[i] = &inspector_dto.Field{
				Name:       f.Name,
				TypeString: f.TypeString,
				RawTag:     f.RawTag,
			}
		}
		mockInsp := buildMockInspector(map[string][]*inspector_dto.Field{
			compGoPackage: inspFields,
		})

		return &pdsLinkerVisitor{
			diagnostics:    new([]*ast_domain.Diagnostic),
			propMapsByHash: make(map[string]map[string]string),
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					compHash: vc,
				},
			},
			typeResolver: &TypeResolver{inspector: mockInsp},
		}
	}

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		fields     []testFieldDef
		want       bool
	}{
		{
			name: "returns true when prop has coerce tag",
			fields: []testFieldDef{
				{Name: "Title", TypeString: "string", RawTag: `prop:"title" coerce:"true"`},
			},
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "props"},
				Property: &ast_domain.Identifier{Name: "Title"},
			},
			want: true,
		},
		{
			name: "returns true when prop has default value",
			fields: []testFieldDef{
				{Name: "Count", TypeString: "int", RawTag: `prop:"count" default:"42"`},
			},
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "props"},
				Property: &ast_domain.Identifier{Name: "Count"},
			},
			want: true,
		},
		{
			name: "returns true when prop has factory function",
			fields: []testFieldDef{
				{Name: "Items", TypeString: "[]string", RawTag: `prop:"items" factory:"NewItems"`},
			},
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "props"},
				Property: &ast_domain.Identifier{Name: "Items"},
			},
			want: true,
		},
		{
			name: "returns false when prop has no transform",
			fields: []testFieldDef{
				{Name: "Label", TypeString: "string", RawTag: `prop:"label"`},
			},
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "props"},
				Property: &ast_domain.Identifier{Name: "Label"},
			},
			want: false,
		},
		{
			name: "returns false when go field name not in prop map",
			fields: []testFieldDef{
				{Name: "Label", TypeString: "string", RawTag: `prop:"label"`},
			},
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "props"},
				Property: &ast_domain.Identifier{Name: "Unknown"},
			},
			want: false,
		},
		{
			name: "returns false when HTML prop name not in valid props",
			fields: []testFieldDef{
				{Name: "Label", TypeString: "string", RawTag: `prop:"label"`},
			},
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "props"},
				Property: &ast_domain.Identifier{Name: "Label"},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := makeVisitor(tc.fields)
			got := v.didPropTransform(context.Background(), compHash, tc.expression)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDidPropTransform_CachesValidProps(t *testing.T) {
	t.Parallel()

	const compHash = "pkg/cached"
	const compGoPackage = "example.com/cached"

	vc := buildVirtualComponent(compHash, compGoPackage, "/tmp/cached.go", "/tmp/cached.pk", []testFieldDef{
		{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
	})

	callCount := 0
	mockInsp := &inspector_domain.MockTypeQuerier{
		GetImportsForFileFunc: func(_, _ string) map[string]string {
			return map[string]string{}
		},
		ResolveExprToNamedTypeFunc: func(_ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
			callCount++
			return &inspector_dto.Type{
				Name:        "Props",
				PackagePath: compGoPackage,
				Fields: []*inspector_dto.Field{
					{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
				},
			}, compGoPackage
		},
	}

	v := &pdsLinkerVisitor{
		diagnostics:    new([]*ast_domain.Diagnostic),
		propMapsByHash: make(map[string]map[string]string),
		virtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				compHash: vc,
			},
		},
		typeResolver: &TypeResolver{inspector: mockInsp},
	}

	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "props"},
		Property: &ast_domain.Identifier{Name: "Title"},
	}

	v.didPropTransform(context.Background(), compHash, expression)
	v.didPropTransform(context.Background(), compHash, expression)

	assert.Equal(t, 1, callCount, "inspector should only be called once due to caching")
}

func TestFindNextLinkInChain(t *testing.T) {
	t.Parallel()

	const invokerHash = "pkg/parent"
	const invokerGoPackage = "example.com/parent"
	const childHash = "pkg/child"

	makeVisitorAndMaps := func() (*pdsLinkerVisitor, *ast_domain.TemplateNode) {
		parentVC := buildVirtualComponent(invokerHash, invokerGoPackage, "/tmp/parent.go", "/tmp/parent.pk", []testFieldDef{
			{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
		})

		mockInsp := buildMockInspector(map[string][]*inspector_dto.Field{
			invokerGoPackage: {
				{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
			},
		})

		v := &pdsLinkerVisitor{
			diagnostics:    new([]*ast_domain.Diagnostic),
			propMapsByHash: make(map[string]map[string]string),
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					invokerHash: parentVC,
				},
			},
			typeResolver: &TypeResolver{inspector: mockInsp},
		}

		invokerNode := &ast_domain.TemplateNode{TagName: "parent-comp"}
		return v, invokerNode
	}

	t.Run("returns next expression when full chain exists", func(t *testing.T) {
		t.Parallel()
		v, invokerNode := makeVisitorAndMaps()

		nextSourceExpr := &ast_domain.Identifier{Name: "directValue"}
		nextPInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: invokerHash,
			PartialAlias:       "parent",
			PassedProps: map[string]ast_domain.PropValue{
				"title": {Expression: nextSourceExpr},
			},
		}

		invocationMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{
			invokerNode: nextPInfo,
		}

		currentPropExpr := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "props"},
			Property: &ast_domain.Identifier{Name: "Title"},
		}
		currentPInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName:  childHash,
			InvokerPackageAlias: invokerHash,
		}

		gotExpr, gotPInfo, ok := v.findNextLinkInChain(context.Background(), currentPropExpr, currentPInfo, invocationMap)
		require.True(t, ok)
		assert.Same(t, nextSourceExpr, gotExpr)
		assert.Same(t, nextPInfo, gotPInfo)
	})

	t.Run("returns false when invoker node exists but next pInfo missing", func(t *testing.T) {
		t.Parallel()
		v, invokerNode := makeVisitorAndMaps()

		invocationMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{
			invokerNode: {
				PartialPackageName: invokerHash,
				PassedProps:        map[string]ast_domain.PropValue{},
			},
		}

		currentPropExpr := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "props"},
			Property: &ast_domain.Identifier{Name: "Title"},
		}
		currentPInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName:  childHash,
			InvokerPackageAlias: invokerHash,
		}

		_, _, ok := v.findNextLinkInChain(context.Background(), currentPropExpr, currentPInfo, invocationMap)
		assert.False(t, ok, "should return false when passed prop not found in next pInfo")
	})

	t.Run("returns false when prop map has no matching go field", func(t *testing.T) {
		t.Parallel()
		v, invokerNode := makeVisitorAndMaps()

		invocationMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{
			invokerNode: {
				PartialPackageName: invokerHash,
				PassedProps: map[string]ast_domain.PropValue{
					"title": {Expression: &ast_domain.Identifier{Name: "x"}},
				},
			},
		}

		currentPropExpr := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "props"},
			Property: &ast_domain.Identifier{Name: "Unknown"},
		}
		currentPInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName:  childHash,
			InvokerPackageAlias: invokerHash,
		}

		_, _, ok := v.findNextLinkInChain(context.Background(), currentPropExpr, currentPInfo, invocationMap)
		assert.False(t, ok, "should return false when go field name not in prop map")
	})
}

func TestResolveTransitiveDataSource(t *testing.T) {
	t.Parallel()

	const compHash = "pkg/child"
	const compGoPackage = "example.com/child"

	makeVisitor := func() *pdsLinkerVisitor {
		vc := buildVirtualComponent(compHash, compGoPackage, "/tmp/child.go", "/tmp/child.pk", []testFieldDef{
			{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
		})

		mockInsp := buildMockInspector(map[string][]*inspector_dto.Field{
			compGoPackage: {
				{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
			},
		})

		return &pdsLinkerVisitor{
			diagnostics:    new([]*ast_domain.Diagnostic),
			propMapsByHash: make(map[string]map[string]string),
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					compHash: vc,
				},
			},
			typeResolver: &TypeResolver{inspector: mockInsp},
		}
	}

	t.Run("returns nil when go field not in prop map", func(t *testing.T) {
		t.Parallel()
		v := makeVisitor()

		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "props"},
			Property: &ast_domain.Identifier{Name: "Unknown"},
		}
		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: compHash,
			PassedProps:        map[string]ast_domain.PropValue{},
		}

		propMapForExpr := map[string]string{"Title": "title"}
		result := v.resolveTransitiveDataSource(context.Background(), expression, pInfo, propMapForExpr, nil)
		assert.Nil(t, result)
	})

	t.Run("returns nil when HTML prop not in passed props", func(t *testing.T) {
		t.Parallel()
		v := makeVisitor()

		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "props"},
			Property: &ast_domain.Identifier{Name: "Title"},
		}
		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: compHash,
			PassedProps:        map[string]ast_domain.PropValue{},
		}

		propMapForExpr := map[string]string{"Title": "title"}
		result := v.resolveTransitiveDataSource(context.Background(), expression, pInfo, propMapForExpr, nil)
		assert.Nil(t, result)
	})

	t.Run("returns data source from non-prop passed expression", func(t *testing.T) {
		t.Parallel()
		v := makeVisitor()

		sourceExpression := &ast_domain.Identifier{Name: "directValue"}
		sourceExpression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			PropDataSource: &ast_domain.PropDataSource{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
				BaseCodeGenVarName: new("myVar"),
			},
		})

		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "props"},
			Property: &ast_domain.Identifier{Name: "Title"},
		}
		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: compHash,
			PassedProps: map[string]ast_domain.PropValue{
				"title": {Expression: sourceExpression},
			},
		}

		propMapForExpr := map[string]string{"Title": "title"}
		invocationMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{}
		result := v.resolveTransitiveDataSource(context.Background(), expression, pInfo, propMapForExpr, invocationMap)
		require.NotNil(t, result)
		assert.NotNil(t, result.ResolvedType)
		require.NotNil(t, result.BaseCodeGenVarName)
		assert.Equal(t, "myVar", *result.BaseCodeGenVarName)
	})

	t.Run("returns nil when prop map is empty for the field", func(t *testing.T) {
		t.Parallel()
		v := makeVisitor()

		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "props"},
			Property: &ast_domain.Identifier{Name: "Title"},
		}
		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: compHash,
			PassedProps:        map[string]ast_domain.PropValue{},
		}

		propMapForExpr := map[string]string{}
		result := v.resolveTransitiveDataSource(context.Background(), expression, pInfo, propMapForExpr, nil)
		assert.Nil(t, result, "should return nil when go field name not in prop map")
	})
}

func TestLinkDataSourceForExpression(t *testing.T) {
	t.Parallel()

	const compHash = "pkg/child"
	const compGoPackage = "example.com/child"

	makeVisitor := func() *pdsLinkerVisitor {
		vc := buildVirtualComponent(compHash, compGoPackage, "/tmp/child.go", "/tmp/child.pk", []testFieldDef{
			{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
		})

		mockInsp := buildMockInspector(map[string][]*inspector_dto.Field{
			compGoPackage: {
				{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
			},
		})

		return &pdsLinkerVisitor{
			diagnostics:    new([]*ast_domain.Diagnostic),
			propMapsByHash: make(map[string]map[string]string),
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					compHash: vc,
				},
			},
			typeResolver: &TypeResolver{inspector: mockInsp},
		}
	}

	t.Run("links prop usage expression to its data source", func(t *testing.T) {
		t.Parallel()
		v := makeVisitor()

		sourceExpression := &ast_domain.Identifier{Name: "stateTitle"}
		sourceExpression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			PropDataSource: &ast_domain.PropDataSource{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
				BaseCodeGenVarName: new("state_title"),
			},
		})

		propExpr := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "props"},
			Property: &ast_domain.Identifier{Name: "Title"},
		}
		propExpr.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		})

		activePInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: compHash,
			PassedProps: map[string]ast_domain.PropValue{
				"title": {Expression: sourceExpression},
			},
		}

		propMapForExpr := map[string]string{"Title": "title"}
		invocationMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{}

		v.linkDataSourceForExpression(context.Background(), propExpr, activePInfo, propMapForExpr, invocationMap)

		ann := propExpr.GetGoAnnotation()
		require.NotNil(t, ann)
		require.NotNil(t, ann.PropDataSource, "PropDataSource should be set after linking")
		require.NotNil(t, ann.PropDataSource.BaseCodeGenVarName)
		assert.Equal(t, "state_title", *ann.PropDataSource.BaseCodeGenVarName)
	})

	t.Run("skips non-prop expressions", func(t *testing.T) {
		t.Parallel()
		v := makeVisitor()

		stateExpr := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "state"},
			Property: &ast_domain.Identifier{Name: "Value"},
		}
		stateExpr.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{})

		activePInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: compHash,
			PassedProps:        map[string]ast_domain.PropValue{},
		}

		propMapForExpr := map[string]string{"Title": "title"}

		v.linkDataSourceForExpression(context.Background(), stateExpr, activePInfo, propMapForExpr, nil)

		ann := stateExpr.GetGoAnnotation()
		require.NotNil(t, ann)
		assert.Nil(t, ann.PropDataSource, "PropDataSource should remain nil for non-prop expressions")
	})

	t.Run("skips prop expression without annotation", func(t *testing.T) {
		t.Parallel()
		v := makeVisitor()

		propExpr := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "props"},
			Property: &ast_domain.Identifier{Name: "Title"},
		}

		activePInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: compHash,
			PassedProps:        map[string]ast_domain.PropValue{},
		}

		propMapForExpr := map[string]string{"Title": "title"}

		v.linkDataSourceForExpression(context.Background(), propExpr, activePInfo, propMapForExpr, nil)

		ann := propExpr.GetGoAnnotation()
		assert.Nil(t, ann)
	})
}

func TestLinkPropsForInvocation(t *testing.T) {
	t.Parallel()

	const compHash = "pkg/child"
	const compGoPackage = "example.com/child"

	t.Run("returns early when prop map is nil", func(t *testing.T) {
		t.Parallel()

		mockInsp := &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string {
				return map[string]string{}
			},
		}

		v := &pdsLinkerVisitor{
			diagnostics:    new([]*ast_domain.Diagnostic),
			propMapsByHash: make(map[string]map[string]string),
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
			typeResolver: &TypeResolver{inspector: mockInsp},
		}

		node := &ast_domain.TemplateNode{TagName: "child"}
		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: "pkg/nonexistent",
		}
		invMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{}

		v.linkPropsForInvocation(context.Background(), node, pInfo, invMap)
	})

	t.Run("links prop expression in matching child node", func(t *testing.T) {
		t.Parallel()

		vc := buildVirtualComponent(compHash, compGoPackage, "/tmp/child.go", "/tmp/child.pk", []testFieldDef{
			{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
		})

		mockInsp := buildMockInspector(map[string][]*inspector_dto.Field{
			compGoPackage: {
				{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
			},
		})

		v := &pdsLinkerVisitor{
			diagnostics:    new([]*ast_domain.Diagnostic),
			propMapsByHash: make(map[string]map[string]string),
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					compHash: vc,
				},
			},
			typeResolver: &TypeResolver{inspector: mockInsp},
		}

		sourceExpression := &ast_domain.Identifier{Name: "stateTitle"}
		sourceExpression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			PropDataSource: &ast_domain.PropDataSource{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
				BaseCodeGenVarName: new("state_title"),
			},
		})

		propExpr := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "props"},
			Property: &ast_domain.Identifier{Name: "Title"},
		}
		propExpr.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		})

		childNode := &ast_domain.TemplateNode{
			TagName: "span",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new(compHash),
			},
			DirText: &ast_domain.Directive{
				Expression: propExpr,
			},
		}

		invocationNode := &ast_domain.TemplateNode{
			TagName: "child-comp",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					PartialPackageName: compHash,
					PartialAlias:       "child",
					PassedProps: map[string]ast_domain.PropValue{
						"title": {Expression: sourceExpression},
					},
				},
			},
			Children: []*ast_domain.TemplateNode{childNode},
		}

		pInfo := invocationNode.GoAnnotations.PartialInfo
		invMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{
			invocationNode: pInfo,
		}

		v.linkPropsForInvocation(context.Background(), invocationNode, pInfo, invMap)

		ann := propExpr.GetGoAnnotation()
		require.NotNil(t, ann)
		require.NotNil(t, ann.PropDataSource, "PropDataSource should be linked for child prop usage")
		require.NotNil(t, ann.PropDataSource.BaseCodeGenVarName)
		assert.Equal(t, "state_title", *ann.PropDataSource.BaseCodeGenVarName)
	})

	t.Run("skips child nodes with non-matching pkg alias", func(t *testing.T) {
		t.Parallel()

		vc := buildVirtualComponent(compHash, compGoPackage, "/tmp/child.go", "/tmp/child.pk", []testFieldDef{
			{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
		})

		mockInsp := buildMockInspector(map[string][]*inspector_dto.Field{
			compGoPackage: {
				{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
			},
		})

		v := &pdsLinkerVisitor{
			diagnostics:    new([]*ast_domain.Diagnostic),
			propMapsByHash: make(map[string]map[string]string),
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					compHash: vc,
				},
			},
			typeResolver: &TypeResolver{inspector: mockInsp},
		}

		propExpr := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "props"},
			Property: &ast_domain.Identifier{Name: "Title"},
		}
		propExpr.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		})

		childNode := &ast_domain.TemplateNode{
			TagName: "span",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("pkg/other"),
			},
			DirText: &ast_domain.Directive{
				Expression: propExpr,
			},
		}

		invocationNode := &ast_domain.TemplateNode{
			TagName: "child-comp",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					PartialPackageName: compHash,
					PartialAlias:       "child",
					PassedProps:        map[string]ast_domain.PropValue{},
				},
			},
			Children: []*ast_domain.TemplateNode{childNode},
		}

		pInfo := invocationNode.GoAnnotations.PartialInfo
		invMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{}

		v.linkPropsForInvocation(context.Background(), invocationNode, pInfo, invMap)

		ann := propExpr.GetGoAnnotation()
		require.NotNil(t, ann)
		assert.Nil(t, ann.PropDataSource, "PropDataSource should not be set for non-matching pkg alias")
	})
}

func TestFollowDataSourceChain_PropChainWithTransform(t *testing.T) {
	t.Parallel()

	const childHash = "pkg/child"
	const childGoPackage = "example.com/child"

	childVC := buildVirtualComponent(childHash, childGoPackage, "/tmp/child.go", "/tmp/child.pk", []testFieldDef{
		{Name: "Title", TypeString: "string", RawTag: `prop:"title" coerce:"true"`},
	})

	mockInsp := buildMockInspector(map[string][]*inspector_dto.Field{
		childGoPackage: {
			{Name: "Title", TypeString: "string", RawTag: `prop:"title" coerce:"true"`},
		},
	})

	v := &pdsLinkerVisitor{
		diagnostics:    new([]*ast_domain.Diagnostic),
		propMapsByHash: make(map[string]map[string]string),
		virtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				childHash: childVC,
			},
		},
		typeResolver: &TypeResolver{inspector: mockInsp},
	}

	midPDS := &ast_domain.PropDataSource{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		},
	}
	sourceExpression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "props"},
		Property: &ast_domain.Identifier{Name: "Title"},
	}
	sourceExpression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
		PropDataSource: midPDS,
	})

	sourcePInfo := &ast_domain.PartialInvocationInfo{
		PartialPackageName:  childHash,
		InvokerPackageAlias: "pkg/parent",
	}

	invocationMap := map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo{}

	result := v.followDataSourceChain(context.Background(), sourceExpression, sourcePInfo, invocationMap)
	require.NotNil(t, result, "should return the data source from the prop with transform")
	assert.Same(t, midPDS, result, "should return the mid-chain data source because the prop has coerce")
}

func TestFollowDataSourceChain_PropUsageWithNilPropDataSource(t *testing.T) {
	t.Parallel()

	v := &pdsLinkerVisitor{
		diagnostics:    new([]*ast_domain.Diagnostic),
		propMapsByHash: make(map[string]map[string]string),
	}

	sourceExpression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "props"},
		Property: &ast_domain.Identifier{Name: "Foo"},
	}
	sourceExpression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{})

	pInfo := &ast_domain.PartialInvocationInfo{PartialPackageName: "pkg/child"}
	invMap := make(map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo)

	result := v.followDataSourceChain(context.Background(), sourceExpression, pInfo, invMap)
	assert.Nil(t, result, "should return nil when PropDataSource is nil on the annotation")
}

func TestFollowDataSourceChain_NonPropWithNilAnnotation(t *testing.T) {
	t.Parallel()

	v := &pdsLinkerVisitor{
		diagnostics:    new([]*ast_domain.Diagnostic),
		propMapsByHash: make(map[string]map[string]string),
	}

	sourceExpression := &ast_domain.Identifier{Name: "someVar"}

	pInfo := &ast_domain.PartialInvocationInfo{PartialPackageName: "pkg/child"}
	invMap := make(map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo)

	result := v.followDataSourceChain(context.Background(), sourceExpression, pInfo, invMap)
	assert.Nil(t, result, "should return nil when final source expression has no annotation")
}

func TestFollowDataSourceChain_TerminatesWhenFindNextLinkFails(t *testing.T) {
	t.Parallel()

	const childHash = "pkg/child"
	const childGoPackage = "example.com/child"

	childVC := buildVirtualComponent(childHash, childGoPackage, "/tmp/child.go", "/tmp/child.pk", []testFieldDef{
		{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
	})

	mockInsp := buildMockInspector(map[string][]*inspector_dto.Field{
		childGoPackage: {
			{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
		},
	})

	v := &pdsLinkerVisitor{
		diagnostics:    new([]*ast_domain.Diagnostic),
		propMapsByHash: make(map[string]map[string]string),
		virtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				childHash: childVC,
			},
		},
		typeResolver: &TypeResolver{inspector: mockInsp},
	}

	expectedPDS := &ast_domain.PropDataSource{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		},
	}

	sourceExpression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: "props"},
		Property: &ast_domain.Identifier{Name: "Title"},
	}
	sourceExpression.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
		PropDataSource: expectedPDS,
	})

	pInfo := &ast_domain.PartialInvocationInfo{
		PartialPackageName:  childHash,
		InvokerPackageAlias: "pkg/nonexistent",
	}
	invMap := make(map[*ast_domain.TemplateNode]*ast_domain.PartialInvocationInfo)

	result := v.followDataSourceChain(context.Background(), sourceExpression, pInfo, invMap)
	require.NotNil(t, result)
	assert.Same(t, expectedPDS, result, "should return the current PDS when findNextLinkInChain fails")
}

func TestGetPropMapForComponent(t *testing.T) {
	t.Parallel()

	const compHash = "pkg/comp"
	const compGoPackage = "example.com/comp"

	t.Run("builds and caches prop map from valid props", func(t *testing.T) {
		t.Parallel()

		vc := buildVirtualComponent(compHash, compGoPackage, "/tmp/comp.go", "/tmp/comp.pk", []testFieldDef{
			{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
			{Name: "Count", TypeString: "int", RawTag: `prop:"count"`},
		})

		mockInsp := buildMockInspector(map[string][]*inspector_dto.Field{
			compGoPackage: {
				{Name: "Title", TypeString: "string", RawTag: `prop:"title"`},
				{Name: "Count", TypeString: "int", RawTag: `prop:"count"`},
			},
		})

		v := &pdsLinkerVisitor{
			diagnostics:    new([]*ast_domain.Diagnostic),
			propMapsByHash: make(map[string]map[string]string),
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					compHash: vc,
				},
			},
			typeResolver: &TypeResolver{inspector: mockInsp},
		}

		propMap := v.getPropMapForComponent(context.Background(), compHash)
		require.NotNil(t, propMap)
		assert.Equal(t, "title", propMap["Title"])
		assert.Equal(t, "count", propMap["Count"])

		propMap2 := v.getPropMapForComponent(context.Background(), compHash)
		assert.Equal(t, propMap, propMap2)
	})

	t.Run("returns nil for unknown component", func(t *testing.T) {
		t.Parallel()

		mockInsp := &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string {
				return map[string]string{}
			},
		}

		v := &pdsLinkerVisitor{
			diagnostics:    new([]*ast_domain.Diagnostic),
			propMapsByHash: make(map[string]map[string]string),
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
			typeResolver: &TypeResolver{inspector: mockInsp},
		}

		propMap := v.getPropMapForComponent(context.Background(), "pkg/nonexistent")
		assert.Nil(t, propMap)
	})
}

func TestPdsLinkerVisitor_GetValidPropsForComponent(t *testing.T) {
	t.Parallel()

	const compHash = "pkg/comp"
	const compGoPackage = "example.com/comp"

	t.Run("returns empty map for component with no script", func(t *testing.T) {
		t.Parallel()

		vc := &annotator_dto.VirtualComponent{
			HashedName:             compHash,
			CanonicalGoPackagePath: compGoPackage,
			VirtualGoFilePath:      "/tmp/comp.go",
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/tmp/comp.pk",
				Script:     nil,
			},
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("pkg"),
			},
		}

		mockInsp := &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string {
				return map[string]string{}
			},
		}

		v := &pdsLinkerVisitor{
			diagnostics:    new([]*ast_domain.Diagnostic),
			propMapsByHash: make(map[string]map[string]string),
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					compHash: vc,
				},
			},
			typeResolver: &TypeResolver{inspector: mockInsp},
		}

		result, err := v.getValidPropsForComponent(context.Background(), compHash)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns error for unknown component hash", func(t *testing.T) {
		t.Parallel()

		mockInsp := &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string {
				return map[string]string{}
			},
		}

		v := &pdsLinkerVisitor{
			diagnostics:    new([]*ast_domain.Diagnostic),
			propMapsByHash: make(map[string]map[string]string),
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
			typeResolver: &TypeResolver{inspector: mockInsp},
		}

		result, err := v.getValidPropsForComponent(context.Background(), "unknown_hash")
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("caches result on successful lookup", func(t *testing.T) {
		t.Parallel()

		const cHash = "pkg/cached"
		const cGoPackage = "example.com/cached"

		vc := buildVirtualComponent(cHash, cGoPackage, "/tmp/cached.go", "/tmp/cached.pk", []testFieldDef{
			{Name: "Name", TypeString: "string", RawTag: `prop:"name"`},
		})

		callCount := 0
		mockInsp := &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string {
				return map[string]string{}
			},
			ResolveExprToNamedTypeFunc: func(_ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				callCount++
				return &inspector_dto.Type{
					Name:        "Props",
					PackagePath: cGoPackage,
					Fields: []*inspector_dto.Field{
						{Name: "Name", TypeString: "string", RawTag: `prop:"name"`},
					},
				}, cGoPackage
			},
		}

		v := &pdsLinkerVisitor{
			diagnostics:    new([]*ast_domain.Diagnostic),
			propMapsByHash: make(map[string]map[string]string),
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					cHash: vc,
				},
			},
			typeResolver: &TypeResolver{inspector: mockInsp},
		}

		result1, err1 := v.getValidPropsForComponent(context.Background(), cHash)
		require.NoError(t, err1)
		require.NotNil(t, result1)
		assert.Contains(t, result1, "name")

		result2, err2 := v.getValidPropsForComponent(context.Background(), cHash)
		require.NoError(t, err2)
		assert.Equal(t, result1, result2)
		assert.Equal(t, 1, callCount, "inspector should only be called once due to caching")
	})
}
