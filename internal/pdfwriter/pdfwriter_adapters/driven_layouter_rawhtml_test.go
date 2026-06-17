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

package pdfwriter_adapters

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func collectExpandedText(node *ast_domain.TemplateNode) string {
	var sb strings.Builder
	var walk func(n *ast_domain.TemplateNode)
	walk = func(n *ast_domain.TemplateNode) {
		if n == nil {
			return
		}
		if n.NodeType == ast_domain.NodeText {
			sb.WriteString(n.TextContent)
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(node)
	return sb.String()
}

func hasDescendantTag(node *ast_domain.TemplateNode, tag string) bool {
	for _, child := range node.Children {
		if child.NodeType == ast_domain.NodeElement && child.TagName == tag {
			return true
		}
		if hasDescendantTag(child, tag) {
			return true
		}
	}
	return false
}

func TestExpandRawHTMLNodes_PHtmlElement(t *testing.T) {
	node := &ast_domain.TemplateNode{
		NodeType:  ast_domain.NodeElement,
		TagName:   "li",
		InnerHTML: "Alpha — <b>bold</b> emphasis",
	}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}}

	expandRawHTMLNodes(context.Background(), tree)

	assert.Empty(t, node.InnerHTML, "InnerHTML should be cleared after expansion")
	require.NotEmpty(t, node.Children, "expected children grafted from InnerHTML, got none")

	text := collectExpandedText(node)
	for _, want := range []string{"Alpha", "bold", "emphasis"} {
		assert.Contains(t, text, want)
	}
	assert.True(t, hasDescendantTag(node, "b"), "expected a <b> element among the grafted children (for UA bold styling)")
}

func TestExpandRawHTMLNodes_PreservesSpacesAroundInline(t *testing.T) {
	node := &ast_domain.TemplateNode{
		NodeType:  ast_domain.NodeElement,
		TagName:   "p",
		InnerHTML: "a strong <b>DevOps</b> culture is how",
	}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}}
	expandRawHTMLNodes(context.Background(), tree)

	got := collectExpandedText(node)
	assert.Equal(t, "a strong DevOps culture is how", got, "spaces around inline element lost")
}

func TestExpandRawHTMLNodes_RawHTMLNode(t *testing.T) {
	parent := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeRawHTML, InnerHTML: "<em>hi</em> there"},
		},
	}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{parent}}

	expandRawHTMLNodes(context.Background(), tree)

	assert.True(t, hasDescendantTag(parent, "em"), "expected the NodeRawHTML node to be replaced by parsed <em> content")
	got := collectExpandedText(parent)
	assert.Contains(t, got, "hi", "expanded text missing expected content")
	assert.Contains(t, got, "there", "expanded text missing expected content")
}
