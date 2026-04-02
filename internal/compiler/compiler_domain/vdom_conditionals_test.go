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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func newTestEvents() *eventBindingCollection {
	return newEventBindingCollection(NewRegistryContext())
}

func makeElementNode(tag, keyVal string) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  tag,
		Key:      &ast_domain.StringLiteral{Value: keyVal},
	}
}

func makeTextNode(content, keyVal string) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: content,
		Key:         &ast_domain.StringLiteral{Value: keyVal},
	}
}

func makeCommentNode(content string) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeComment,
		TextContent: content,
		Key:         &ast_domain.StringLiteral{Value: "c"},
	}
}

func makeDirective(rawExpr string) *ast_domain.Directive {
	return &ast_domain.Directive{
		RawExpression: rawExpr,
		Expression:    &ast_domain.Identifier{Name: rawExpr},
	}
}

func makeElseDirective() *ast_domain.Directive {
	return &ast_domain.Directive{}
}

func TestVdomConditional_isInsignificantNode(t *testing.T) {
	tests := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "comment node is insignificant",
			node:     makeCommentNode("this is a comment"),
			expected: true,
		},
		{
			name: "whitespace-only text node is insignificant",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "   \t\n  ",
			},
			expected: true,
		},
		{
			name: "empty text node is insignificant",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "",
			},
			expected: true,
		},
		{
			name: "text node with content is significant",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "hello",
			},
			expected: false,
		},
		{
			name: "text node with mixed whitespace and content is significant",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "  hello  ",
			},
			expected: false,
		},
		{
			name:     "element node is significant",
			node:     makeElementNode("div", "0"),
			expected: false,
		},
		{
			name: "fragment node is significant",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeFragment,
			},
			expected: false,
		},
		{
			name: "raw HTML node is significant",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeRawHTML,
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isInsignificantNode(tc.node)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVdomConditional_isChainContinuation(t *testing.T) {
	tests := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name: "node with DirElseIf is a chain continuation",
			node: &ast_domain.TemplateNode{
				NodeType:  ast_domain.NodeElement,
				TagName:   "div",
				DirElseIf: makeDirective("other"),
			},
			expected: true,
		},
		{
			name: "node with DirElse is a chain continuation",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DirElse:  makeElseDirective(),
			},
			expected: true,
		},
		{
			name: "node with DirIf is not a chain continuation",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DirIf:    makeDirective("show"),
			},
			expected: false,
		},
		{
			name:     "plain element is not a chain continuation",
			node:     makeElementNode("div", "0"),
			expected: false,
		},
		{
			name: "node with both DirElseIf and DirElse returns true",
			node: &ast_domain.TemplateNode{
				NodeType:  ast_domain.NodeElement,
				TagName:   "div",
				DirElseIf: makeDirective("x"),
				DirElse:   makeElseDirective(),
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isChainContinuation(tc.node)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVdomConditional_collectConditionalChain(t *testing.T) {
	t.Run("single if node with no following siblings", func(t *testing.T) {
		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("show")

		children := []*ast_domain.TemplateNode{ifNode}

		chain, nextIndex := collectConditionalChain(ifNode, children, 0)

		assert.Len(t, chain, 1)
		assert.Same(t, ifNode, chain[0])
		assert.Equal(t, 1, nextIndex)
	})

	t.Run("if with else-if and else collects all three", func(t *testing.T) {
		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("a")

		elseIfNode := makeElementNode("span", "1")
		elseIfNode.DirElseIf = makeDirective("b")

		elseNode := makeElementNode("p", "2")
		elseNode.DirElse = makeElseDirective()

		children := []*ast_domain.TemplateNode{ifNode, elseIfNode, elseNode}

		chain, nextIndex := collectConditionalChain(ifNode, children, 0)

		assert.Len(t, chain, 3)
		assert.Same(t, ifNode, chain[0])
		assert.Same(t, elseIfNode, chain[1])
		assert.Same(t, elseNode, chain[2])
		assert.Equal(t, 3, nextIndex)
	})

	t.Run("skips insignificant nodes between chain members", func(t *testing.T) {
		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("a")

		whitespace := makeTextNode("   ", "ws")
		comment := makeCommentNode("separator")

		elseNode := makeElementNode("p", "2")
		elseNode.DirElse = makeElseDirective()

		children := []*ast_domain.TemplateNode{ifNode, whitespace, comment, elseNode}

		chain, nextIndex := collectConditionalChain(ifNode, children, 0)

		assert.Len(t, chain, 2)
		assert.Same(t, ifNode, chain[0])
		assert.Same(t, elseNode, chain[1])
		assert.Equal(t, 4, nextIndex)
	})

	t.Run("stops at significant non-chain node", func(t *testing.T) {
		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("a")

		plainNode := makeElementNode("span", "1")

		elseNode := makeElementNode("p", "2")
		elseNode.DirElse = makeElseDirective()

		children := []*ast_domain.TemplateNode{ifNode, plainNode, elseNode}

		chain, nextIndex := collectConditionalChain(ifNode, children, 0)

		assert.Len(t, chain, 1)
		assert.Same(t, ifNode, chain[0])
		assert.Equal(t, 1, nextIndex)
	})

	t.Run("stops at significant text node", func(t *testing.T) {
		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("a")

		textNode := makeTextNode("visible text", "t")

		elseNode := makeElementNode("p", "2")
		elseNode.DirElse = makeElseDirective()

		children := []*ast_domain.TemplateNode{ifNode, textNode, elseNode}

		chain, nextIndex := collectConditionalChain(ifNode, children, 0)

		assert.Len(t, chain, 1)
		assert.Same(t, ifNode, chain[0])
		assert.Equal(t, 1, nextIndex)
	})

	t.Run("if node not at start index", func(t *testing.T) {
		leadingNode := makeElementNode("header", "h")

		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("a")

		elseNode := makeElementNode("p", "2")
		elseNode.DirElse = makeElseDirective()

		children := []*ast_domain.TemplateNode{leadingNode, ifNode, elseNode}

		chain, nextIndex := collectConditionalChain(ifNode, children, 1)

		assert.Len(t, chain, 2)
		assert.Same(t, ifNode, chain[0])
		assert.Same(t, elseNode, chain[1])
		assert.Equal(t, 3, nextIndex)
	})

	t.Run("multiple else-if nodes collected", func(t *testing.T) {
		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("a")

		elseIf1 := makeElementNode("span", "1")
		elseIf1.DirElseIf = makeDirective("b")

		elseIf2 := makeElementNode("em", "2")
		elseIf2.DirElseIf = makeDirective("c")

		elseNode := makeElementNode("p", "3")
		elseNode.DirElse = makeElseDirective()

		children := []*ast_domain.TemplateNode{ifNode, elseIf1, elseIf2, elseNode}

		chain, nextIndex := collectConditionalChain(ifNode, children, 0)

		assert.Len(t, chain, 4)
		assert.Equal(t, 4, nextIndex)
	})
}

func TestVdomConditional_buildConditionalChainAST(t *testing.T) {
	ctx := context.Background()

	t.Run("empty chain returns null literal", func(t *testing.T) {
		events := newTestEvents()

		result, err := buildConditionalChainAST(ctx, nil, events, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result.Data)
		assert.True(t, isNull(result))
	})

	t.Run("single node chain delegates to buildSingleConditional", func(t *testing.T) {
		events := newTestEvents()

		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("show")

		chain := []*ast_domain.TemplateNode{ifNode}

		result, err := buildConditionalChainAST(ctx, chain, events, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result.Data)

		eif, ok := result.Data.(*js_ast.EIf)
		require.True(t, ok, "expected EIf for single p-if node")
		assert.NotNil(t, eif.Test.Data)
		assert.NotNil(t, eif.Yes.Data)
		assert.True(t, isNull(eif.No), "single conditional should have null as the No branch")
	})

	t.Run("two node chain with else", func(t *testing.T) {
		events := newTestEvents()

		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("show")

		elseNode := makeElementNode("span", "1")
		elseNode.DirElse = makeElseDirective()

		chain := []*ast_domain.TemplateNode{ifNode, elseNode}

		result, err := buildConditionalChainAST(ctx, chain, events, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result.Data)

		eif, ok := result.Data.(*js_ast.EIf)
		require.True(t, ok, "expected EIf for if/else chain")
		assert.NotNil(t, eif.Test.Data)
		assert.NotNil(t, eif.Yes.Data)
		assert.False(t, isNull(eif.No), "else branch should not be null")
	})

	t.Run("three node chain produces nested ternary", func(t *testing.T) {
		events := newTestEvents()

		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("a")

		elseIfNode := makeElementNode("span", "1")
		elseIfNode.DirElseIf = makeDirective("b")

		elseNode := makeElementNode("p", "2")
		elseNode.DirElse = makeElseDirective()

		chain := []*ast_domain.TemplateNode{ifNode, elseIfNode, elseNode}

		result, err := buildConditionalChainAST(ctx, chain, events, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result.Data)

		outerIf, ok := result.Data.(*js_ast.EIf)
		require.True(t, ok, "expected outer EIf")

		innerIf, ok := outerIf.No.Data.(*js_ast.EIf)
		require.True(t, ok, "expected inner EIf for the else-if branch")
		assert.NotNil(t, innerIf.Test.Data)
		assert.NotNil(t, innerIf.Yes.Data)
		assert.NotNil(t, innerIf.No.Data)
	})
}

func TestVdomConditional_buildSingleConditional(t *testing.T) {
	ctx := context.Background()

	t.Run("node without DirIf returns null", func(t *testing.T) {
		events := newTestEvents()
		node := makeElementNode("div", "0")

		result, err := buildSingleConditional(ctx, node, events, nil, nil)

		require.NoError(t, err)
		assert.True(t, isNull(result))
	})

	t.Run("node with DirIf returns ternary", func(t *testing.T) {
		events := newTestEvents()

		node := makeElementNode("div", "0")
		node.DirIf = makeDirective("show")

		result, err := buildSingleConditional(ctx, node, events, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result.Data)

		eif, ok := result.Data.(*js_ast.EIf)
		require.True(t, ok, "expected EIf result")
		assert.NotNil(t, eif.Test.Data, "test condition should be set")
		assert.NotNil(t, eif.Yes.Data, "yes branch should be the rendered node")
		assert.True(t, isNull(eif.No), "no branch should be null for standalone p-if")
	})

	t.Run("passes loop vars through", func(t *testing.T) {
		events := newTestEvents()

		node := makeElementNode("div", "0")
		node.DirIf = makeDirective("active")

		loopVars := map[string]bool{"item": true}

		result, err := buildSingleConditional(ctx, node, events, loopVars, nil)

		require.NoError(t, err)
		require.NotNil(t, result.Data)

		_, ok := result.Data.(*js_ast.EIf)
		assert.True(t, ok, "expected EIf result with loop vars")
	})
}

func TestVdomConditional_buildChainedConditional(t *testing.T) {
	ctx := context.Background()

	t.Run("chain ending with else uses else node as fallback", func(t *testing.T) {
		events := newTestEvents()

		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("a")

		elseNode := makeElementNode("span", "1")
		elseNode.DirElse = makeElseDirective()

		chain := []*ast_domain.TemplateNode{ifNode, elseNode}

		result, err := buildChainedConditional(ctx, chain, events, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result.Data)

		eif, ok := result.Data.(*js_ast.EIf)
		require.True(t, ok)
		assert.False(t, isNull(eif.No), "else branch should render the else node, not null")
	})

	t.Run("chain ending without else uses null fallback", func(t *testing.T) {
		events := newTestEvents()

		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("a")

		elseIfNode := makeElementNode("span", "1")
		elseIfNode.DirElseIf = makeDirective("b")

		chain := []*ast_domain.TemplateNode{ifNode, elseIfNode}

		result, err := buildChainedConditional(ctx, chain, events, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result.Data)

		outerIf, ok := result.Data.(*js_ast.EIf)
		require.True(t, ok)
		assert.True(t, isNull(outerIf.No), "chain without else should use null fallback")
	})

	t.Run("reverse iteration builds correct nesting", func(t *testing.T) {
		events := newTestEvents()

		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("a")

		elseIf1 := makeElementNode("span", "1")
		elseIf1.DirElseIf = makeDirective("b")

		elseIf2 := makeElementNode("em", "2")
		elseIf2.DirElseIf = makeDirective("c")

		elseNode := makeElementNode("p", "3")
		elseNode.DirElse = makeElseDirective()

		chain := []*ast_domain.TemplateNode{ifNode, elseIf1, elseIf2, elseNode}

		result, err := buildChainedConditional(ctx, chain, events, nil, nil)

		require.NoError(t, err)

		level1, ok := result.Data.(*js_ast.EIf)
		require.True(t, ok, "first level should be EIf")

		level2, ok := level1.No.Data.(*js_ast.EIf)
		require.True(t, ok, "second level should be EIf")

		level3, ok := level2.No.Data.(*js_ast.EIf)
		require.True(t, ok, "third level should be EIf")

		assert.False(t, isNull(level3.No), "innermost No should be the else node")
	})
}

func TestVdomConditional_wrapNodeInTernary(t *testing.T) {
	ctx := context.Background()

	t.Run("node with DirIf wraps in ternary", func(t *testing.T) {
		events := newTestEvents()
		alternate := newNullLiteral()

		node := makeElementNode("div", "0")
		node.DirIf = makeDirective("show")

		result, err := wrapNodeInTernary(ctx, node, alternate, events, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result.Data)

		eif, ok := result.Data.(*js_ast.EIf)
		require.True(t, ok)
		assert.NotNil(t, eif.Test.Data)
		assert.NotNil(t, eif.Yes.Data)
		assert.True(t, isNull(eif.No), "alternate should be the passed null literal")
	})

	t.Run("node with DirElseIf wraps in ternary", func(t *testing.T) {
		events := newTestEvents()
		alternate := newNullLiteral()

		node := makeElementNode("span", "1")
		node.DirElseIf = makeDirective("other")

		result, err := wrapNodeInTernary(ctx, node, alternate, events, nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result.Data)

		eif, ok := result.Data.(*js_ast.EIf)
		require.True(t, ok, "DirElseIf should also produce EIf")
		assert.NotNil(t, eif.Test.Data)
	})

	t.Run("node with neither DirIf nor DirElseIf returns alternate directly", func(t *testing.T) {
		events := newTestEvents()
		alternate := newNullLiteral()

		node := makeElementNode("div", "0")

		result, err := wrapNodeInTernary(ctx, node, alternate, events, nil, nil)

		require.NoError(t, err)
		assert.True(t, isNull(result), "should return the alternate when no conditional directive")
	})

	t.Run("alternate expression is preserved in the No branch", func(t *testing.T) {
		events := newTestEvents()

		alternateNode := makeElementNode("footer", "alt")
		alternateExpr, err := buildNodeAST(ctx, alternateNode, events, nil, nil)
		require.NoError(t, err)

		node := makeElementNode("div", "0")
		node.DirIf = makeDirective("show")

		result, err := wrapNodeInTernary(ctx, node, alternateExpr, events, nil, nil)

		require.NoError(t, err)

		eif, ok := result.Data.(*js_ast.EIf)
		require.True(t, ok)
		assert.Equal(t, alternateExpr.Data, eif.No.Data, "alternate should be used as No branch")
	})

	t.Run("node with DirElse returns alternate unchanged", func(t *testing.T) {
		events := newTestEvents()
		alternate := newNullLiteral()

		node := makeElementNode("div", "0")
		node.DirElse = makeElseDirective()

		result, err := wrapNodeInTernary(ctx, node, alternate, events, nil, nil)

		require.NoError(t, err)
		assert.True(t, isNull(result), "DirElse only node should return alternate (no test condition)")
	})
}

func TestVdomConditional_processChainAwareChildren(t *testing.T) {
	ctx := context.Background()

	t.Run("nil children returns nil", func(t *testing.T) {
		events := newTestEvents()

		result, err := processChainAwareChildren(ctx, nil, events, nil, nil)

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty children returns nil", func(t *testing.T) {
		events := newTestEvents()

		result, err := processChainAwareChildren(ctx, []*ast_domain.TemplateNode{}, events, nil, nil)

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("plain element produces one expression", func(t *testing.T) {
		events := newTestEvents()

		node := makeElementNode("div", "0")
		children := []*ast_domain.TemplateNode{node}

		result, err := processChainAwareChildren(ctx, children, events, nil, nil)

		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("if/else chain produces single ternary expression", func(t *testing.T) {
		events := newTestEvents()

		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("show")

		elseNode := makeElementNode("span", "1")
		elseNode.DirElse = makeElseDirective()

		children := []*ast_domain.TemplateNode{ifNode, elseNode}

		result, err := processChainAwareChildren(ctx, children, events, nil, nil)

		require.NoError(t, err)
		assert.Len(t, result, 1, "if/else should be collapsed into a single expression")

		_, ok := result[0].Data.(*js_ast.EIf)
		assert.True(t, ok, "the single expression should be a ternary")
	})

	t.Run("orphan else-if node is skipped", func(t *testing.T) {
		events := newTestEvents()

		orphanElseIf := makeElementNode("div", "0")
		orphanElseIf.DirElseIf = makeDirective("x")

		children := []*ast_domain.TemplateNode{orphanElseIf}

		result, err := processChainAwareChildren(ctx, children, events, nil, nil)

		require.NoError(t, err)
		assert.Empty(t, result, "orphan else-if should be skipped")
	})

	t.Run("orphan else node is skipped", func(t *testing.T) {
		events := newTestEvents()

		orphanElse := makeElementNode("div", "0")
		orphanElse.DirElse = makeElseDirective()

		children := []*ast_domain.TemplateNode{orphanElse}

		result, err := processChainAwareChildren(ctx, children, events, nil, nil)

		require.NoError(t, err)
		assert.Empty(t, result, "orphan else should be skipped")
	})

	t.Run("mixed plain and conditional children", func(t *testing.T) {
		events := newTestEvents()

		leading := makeElementNode("header", "h")

		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("show")

		elseNode := makeElementNode("span", "1")
		elseNode.DirElse = makeElseDirective()

		trailing := makeElementNode("footer", "f")

		children := []*ast_domain.TemplateNode{leading, ifNode, elseNode, trailing}

		result, err := processChainAwareChildren(ctx, children, events, nil, nil)

		require.NoError(t, err)
		assert.Len(t, result, 3, "should have leading + ternary + trailing")
	})

	t.Run("whitespace-only text node produces a whitespace call expression", func(t *testing.T) {
		events := newTestEvents()

		ws := makeTextNode("   ", "ws")
		ws.Key = &ast_domain.StringLiteral{Value: "ws"}

		children := []*ast_domain.TemplateNode{ws}

		result, err := processChainAwareChildren(ctx, children, events, nil, nil)

		require.NoError(t, err)

		assert.Len(t, result, 1)
	})

	t.Run("multiple separate if chains produce separate expressions", func(t *testing.T) {
		events := newTestEvents()

		if1 := makeElementNode("div", "0")
		if1.DirIf = makeDirective("a")

		else1 := makeElementNode("span", "1")
		else1.DirElse = makeElseDirective()

		separator := makeElementNode("hr", "s")

		if2 := makeElementNode("em", "2")
		if2.DirIf = makeDirective("b")

		children := []*ast_domain.TemplateNode{if1, else1, separator, if2}

		result, err := processChainAwareChildren(ctx, children, events, nil, nil)

		require.NoError(t, err)
		assert.Len(t, result, 3, "two conditional chains + separator = 3 expressions")
	})
}

func TestVdomConditional_processChildNode(t *testing.T) {
	ctx := context.Background()

	t.Run("plain element is not skipped", func(t *testing.T) {
		events := newTestEvents()
		node := makeElementNode("div", "0")
		children := []*ast_domain.TemplateNode{node}
		expression, skip, err := processChildNode(ctx, node, children, new(0), events, nil, nil)

		require.NoError(t, err)
		assert.False(t, skip)
		assert.NotNil(t, expression.Data)
	})

	t.Run("DirIf node triggers conditional chain processing", func(t *testing.T) {
		events := newTestEvents()

		ifNode := makeElementNode("div", "0")
		ifNode.DirIf = makeDirective("show")

		elseNode := makeElementNode("span", "1")
		elseNode.DirElse = makeElseDirective()

		children := []*ast_domain.TemplateNode{ifNode, elseNode}
		index := 0

		expression, skip, err := processChildNode(ctx, ifNode, children, &index, events, nil, nil)

		require.NoError(t, err)
		assert.False(t, skip, "conditional chain should not be skipped")
		require.NotNil(t, expression.Data)

		_, ok := expression.Data.(*js_ast.EIf)
		assert.True(t, ok, "should produce a ternary expression")
		assert.Equal(t, 1, index, "index should advance past the else node")
	})

	t.Run("orphan DirElseIf node is skipped", func(t *testing.T) {
		events := newTestEvents()
		node := makeElementNode("div", "0")
		node.DirElseIf = makeDirective("x")
		children := []*ast_domain.TemplateNode{node}
		expression, skip, err := processChildNode(ctx, node, children, new(0), events, nil, nil)

		require.NoError(t, err)
		assert.True(t, skip)
		assert.Nil(t, expression.Data)
	})

	t.Run("orphan DirElse node is skipped", func(t *testing.T) {
		events := newTestEvents()
		node := makeElementNode("div", "0")
		node.DirElse = makeElseDirective()
		children := []*ast_domain.TemplateNode{node}
		expression, skip, err := processChildNode(ctx, node, children, new(0), events, nil, nil)

		require.NoError(t, err)
		assert.True(t, skip)
		assert.Nil(t, expression.Data)
	})

	t.Run("text node with content is not skipped", func(t *testing.T) {
		events := newTestEvents()
		node := makeTextNode("hello", "t")
		children := []*ast_domain.TemplateNode{node}
		expression, skip, err := processChildNode(ctx, node, children, new(0), events, nil, nil)

		require.NoError(t, err)
		assert.False(t, skip)
		assert.NotNil(t, expression.Data)
	})

	t.Run("comment node is not skipped and produces an expression", func(t *testing.T) {
		events := newTestEvents()
		node := makeCommentNode("my comment")
		children := []*ast_domain.TemplateNode{node}
		expression, skip, err := processChildNode(ctx, node, children, new(0), events, nil, nil)

		require.NoError(t, err)
		assert.False(t, skip)
		assert.NotNil(t, expression.Data)
	})
}
