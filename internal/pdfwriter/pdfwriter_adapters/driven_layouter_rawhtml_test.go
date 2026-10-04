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
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
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

	require.NoError(t, expandRawHTMLNodes(context.Background(), tree, layouter_dto.LayoutLimits{}))

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
	require.NoError(t, expandRawHTMLNodes(context.Background(), tree, layouter_dto.LayoutLimits{}))

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

	require.NoError(t, expandRawHTMLNodes(context.Background(), tree, layouter_dto.LayoutLimits{}))

	assert.True(t, hasDescendantTag(parent, "em"), "expected the NodeRawHTML node to be replaced by parsed <em> content")
	got := collectExpandedText(parent)
	assert.Contains(t, got, "hi", "expanded text missing expected content")
	assert.Contains(t, got, "there", "expanded text missing expected content")
}

func TestExpandRawHTMLNodes_EnforcesLimits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		raw     string
		name    string
		limits  layouter_dto.LayoutLimits
	}{
		{
			name:    "raw HTML larger than the byte budget",
			raw:     strings.Repeat("<b>x</b>", 20),
			limits:  layouter_dto.LayoutLimits{MaxRawHTMLBytes: 100},
			wantErr: layouter_dto.ErrRawHTMLTooLarge,
		},
		{
			name:    "raw HTML nesting deeper than the limit",
			raw:     strings.Repeat("<div>", 40) + "x" + strings.Repeat("</div>", 40),
			limits:  layouter_dto.LayoutLimits{MaxNestingDepth: 20},
			wantErr: layouter_dto.ErrNestingTooDeep,
		},
		{
			name:    "raw HTML with more nodes than the limit",
			raw:     strings.Repeat("<span>x</span>", 50),
			limits:  layouter_dto.LayoutLimits{MaxBoxNodes: 30},
			wantErr: layouter_dto.ErrTooManyBoxes,
		},
		{
			name:   "raw HTML exactly at the byte budget",
			raw:    strings.Repeat("a", 64),
			limits: layouter_dto.LayoutLimits{MaxRawHTMLBytes: 64},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "div", InnerHTML: tt.raw}
			tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}}

			err := expandRawHTMLNodes(context.Background(), tree, tt.limits)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.ErrorIs(t, err, layouter_dto.ErrLayoutLimitExceeded)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestExpandRawHTMLNodes_BudgetCoversEveryFragment(t *testing.T) {
	t.Parallel()

	first := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "p", InnerHTML: strings.Repeat("a", 60)}
	second := &ast_domain.TemplateNode{NodeType: ast_domain.NodeRawHTML, InnerHTML: strings.Repeat("b", 60)}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{first, second}}

	err := expandRawHTMLNodes(context.Background(), tree, layouter_dto.LayoutLimits{MaxRawHTMLBytes: 100})

	assert.ErrorIs(t, err, layouter_dto.ErrRawHTMLTooLarge)
}

func TestExpandRawHTMLNodes_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("stopped by test"))
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{{NodeType: ast_domain.NodeElement, TagName: "div", InnerHTML: "<b>x</b>"}}}

	err := expandRawHTMLNodes(ctx, tree, layouter_dto.LayoutLimits{})

	assert.ErrorIs(t, err, context.Canceled)
}

func TestExpandRawHTMLNodes_NilTree(t *testing.T) {
	t.Parallel()

	assert.NoError(t, expandRawHTMLNodes(context.Background(), nil, layouter_dto.LayoutLimits{}))
}

func TestExpandRawHTMLNodes_RejectsElementFloodBeforeParsing(t *testing.T) {
	t.Parallel()

	node := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "div", InnerHTML: strings.Repeat("<b>x</b>", 5000)}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}}

	err := expandRawHTMLNodes(context.Background(), tree, layouter_dto.LayoutLimits{MaxBoxNodes: 1000})

	assert.ErrorIs(t, err, layouter_dto.ErrTooManyBoxes)
	assert.Empty(t, node.Children, "the flood is rejected without being parsed")
}

func TestEstimateElementCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want int
	}{
		{name: "text only", raw: "hello", want: 0},
		{name: "paired elements", raw: "<b>x</b><i>y</i>", want: 2},
		{name: "void element", raw: "a<br>b", want: 1},
		{name: "literal less-than counts conservatively", raw: "1 < 2", want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, estimateElementCount(tt.raw))
		})
	}
}
