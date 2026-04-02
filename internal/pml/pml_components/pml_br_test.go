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

package pml_components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

func TestLineBreak_TagName(t *testing.T) {
	br := NewLineBreak()
	assert.Equal(t, "pml-br", br.TagName())
}

func TestLineBreak_IsEndingTag(t *testing.T) {
	br := NewLineBreak()
	assert.True(t, br.IsEndingTag())
}

func TestLineBreak_AllowedParents_Empty(t *testing.T) {
	br := NewLineBreak()
	parents := br.AllowedParents()

	assert.Empty(t, parents, "AllowedParents should be empty to allow all contexts")
}

func TestLineBreak_AllowedAttributes(t *testing.T) {
	br := NewLineBreak()
	attrs := br.AllowedAttributes()

	require.NotEmpty(t, attrs)
	assert.Contains(t, attrs, AttrHeight)
	assert.Contains(t, attrs, AttrContainerBackgroundColor)
	assert.Contains(t, attrs, AttrPadding)
	assert.Contains(t, attrs, "padding-top")
	assert.Contains(t, attrs, "padding-bottom")
	assert.Contains(t, attrs, "padding-left")
	assert.Contains(t, attrs, "padding-right")
}

func TestLineBreak_DefaultAttributes(t *testing.T) {
	br := NewLineBreak()
	defaults := br.DefaultAttributes()

	assert.Empty(t, defaults)
}

func TestLineBreak_GetStyleTargets(t *testing.T) {
	br := NewLineBreak()
	targets := br.GetStyleTargets()

	require.Len(t, targets, 2)
	assert.Equal(t, AttrHeight, targets[0].Property)
	assert.Equal(t, TargetContainer, targets[0].Target)
	assert.Equal(t, AttrPadding, targets[1].Property)
	assert.Equal(t, TargetContainer, targets[1].Target)
}

func TestLineBreak_Transform_DefaultHeight(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		Build()

	ctx := NewTestContext().Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	require.Len(t, result.Children, 2)

	outlookTable := result.Children[0]
	assert.Equal(t, ast_domain.NodeRawHTML, outlookTable.NodeType)
	assert.Contains(t, outlookTable.TextContent, "<!--[if mso | IE]>")
	assert.Contains(t, outlookTable.TextContent, "<![endif]-->")
	assert.Contains(t, outlookTable.TextContent, "height:20px;")

	modernDiv := result.Children[1]
	assert.Equal(t, ast_domain.NodeElement, modernDiv.NodeType)
	assert.Equal(t, ElementDiv, modernDiv.TagName)

	styleAttr, found := FindAttribute(modernDiv, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "height:20px;")
	assert.Contains(t, styleAttr.Value, "line-height:20px;")
	assert.Contains(t, styleAttr.Value, "font-size:0px;")

	require.Len(t, modernDiv.Children, 1)
	hairSpace := modernDiv.Children[0]
	assert.Equal(t, ast_domain.NodeText, hairSpace.NodeType)
	assert.Equal(t, "&#8202;", hairSpace.TextContent)
}

func TestLineBreak_Transform_CustomHeight(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		WithAttribute(AttrHeight, "50px").
		Build()

	ctx := NewTestContext().Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)
	require.Len(t, result.Children, 2)

	outlookTable := result.Children[0]
	assert.Contains(t, outlookTable.TextContent, "height:50px;")
	assert.Contains(t, outlookTable.TextContent, `height="50"`)

	modernDiv := result.Children[1]
	styleAttr, found := FindAttribute(modernDiv, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "height:50px;")
	assert.Contains(t, styleAttr.Value, "line-height:50px;")
}

func TestLineBreak_Transform_OutlookTableStructure(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		WithAttribute(AttrHeight, "30px").
		Build()

	ctx := NewTestContext().Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	outlookTable := result.Children[0]
	htmlContent := outlookTable.TextContent

	assert.Contains(t, htmlContent, `role="presentation"`)
	assert.Contains(t, htmlContent, `border="0"`)
	assert.Contains(t, htmlContent, `cellpadding="0"`)
	assert.Contains(t, htmlContent, `cellspacing="0"`)
	assert.Contains(t, htmlContent, "&nbsp;")
	assert.Contains(t, htmlContent, "<table")
	assert.Contains(t, htmlContent, "<tr>")
	assert.Contains(t, htmlContent, "<td")
	assert.Contains(t, htmlContent, "</td>")
	assert.Contains(t, htmlContent, "</tr>")
	assert.Contains(t, htmlContent, "</table>")
}

func TestLineBreak_Transform_PreservesPikoDirectives(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		WithAttribute(AttrHeight, "25px").
		Build()

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showSpacer"},
	}

	ctx := NewTestContext().Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.NotNil(t, result.DirIf)
	assert.Equal(t, "showSpacer", result.DirIf.Expression.(*ast_domain.Identifier).Name)

	assert.Nil(t, node.DirIf)
}

func TestLineBreak_Transform_MultipleHeights(t *testing.T) {
	testCases := []struct {
		name           string
		height         string
		expectedHeight string
		expectedPx     string
	}{
		{name: "small spacer", height: "10px", expectedHeight: "10px", expectedPx: `height="10"`},
		{name: "medium spacer", height: "40px", expectedHeight: "40px", expectedPx: `height="40"`},
		{name: "large spacer", height: "100px", expectedHeight: "100px", expectedPx: `height="100"`},
		{name: "zero spacer", height: "0px", expectedHeight: "0px", expectedPx: `height="0"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			br := NewLineBreak()
			node := NewTestNode().
				WithTagName("pml-br").
				WithAttribute(AttrHeight, tc.height).
				Build()

			ctx := NewTestContext().Build(node, br)
			result, errs := br.Transform(node, ctx)

			require.Nil(t, errs)
			require.NotNil(t, result)

			outlookTable := result.Children[0]
			assert.Contains(t, outlookTable.TextContent, "height:"+tc.expectedHeight)
			assert.Contains(t, outlookTable.TextContent, tc.expectedPx)

			modernDiv := result.Children[1]
			styleAttr, found := FindAttribute(modernDiv, AttrStyle)
			require.True(t, found)
			assert.Contains(t, styleAttr.Value, "height:"+tc.expectedHeight)
			assert.Contains(t, styleAttr.Value, "line-height:"+tc.expectedHeight)
		})
	}
}

func TestLineBreak_Transform_HairSpaceIsPresent(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		Build()

	ctx := NewTestContext().Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)

	modernDiv := result.Children[1]
	require.Len(t, modernDiv.Children, 1)

	textNode := modernDiv.Children[0]
	assert.Equal(t, ast_domain.NodeText, textNode.NodeType)

	assert.Equal(t, "&#8202;", textNode.TextContent)
}

func TestLineBreak_Transform_DivHasZeroFontSize(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		WithAttribute(AttrHeight, "35px").
		Build()

	ctx := NewTestContext().Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)

	modernDiv := result.Children[1]
	styleAttr, found := FindAttribute(modernDiv, AttrStyle)
	require.True(t, found)

	assert.Contains(t, styleAttr.Value, "font-size:0px;")
}

func TestRenderOutlookBreakTable_BasicStructure(t *testing.T) {
	result := renderOutlookBreakTable("30px")

	assert.Equal(t, ast_domain.NodeRawHTML, result.NodeType)

	htmlContent := result.TextContent

	assert.True(t, strings.HasPrefix(htmlContent, "<!--[if mso | IE]>"))
	assert.True(t, strings.HasSuffix(htmlContent, "<![endif]-->"))

	assert.Contains(t, htmlContent, "<table")
	assert.Contains(t, htmlContent, "</table>")
	assert.Contains(t, htmlContent, "<tr>")
	assert.Contains(t, htmlContent, "<td")
	assert.Contains(t, htmlContent, "&nbsp;")
}

func TestRenderOutlookBreakTable_HeightAttribute(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		expectedPx   string
		expectedAttr string
	}{
		{name: "20px", input: "20px", expectedPx: "height:20px;", expectedAttr: `height="20"`},
		{name: "50px", input: "50px", expectedPx: "height:50px;", expectedAttr: `height="50"`},
		{name: "0px", input: "0px", expectedPx: "height:0px;", expectedAttr: `height="0"`},
		{name: "100px", input: "100px", expectedPx: "height:100px;", expectedAttr: `height="100"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderOutlookBreakTable(tc.input)

			htmlContent := result.TextContent

			assert.Contains(t, htmlContent, tc.expectedPx)
			assert.Contains(t, htmlContent, tc.expectedAttr)
		})
	}
}

func TestRenderOutlookBreakTable_TableAttributes(t *testing.T) {
	result := renderOutlookBreakTable("25px")

	htmlContent := result.TextContent

	assert.Contains(t, htmlContent, `role="presentation"`)
	assert.Contains(t, htmlContent, `border="0"`)
	assert.Contains(t, htmlContent, `cellpadding="0"`)
	assert.Contains(t, htmlContent, `cellspacing="0"`)
}

func TestRenderOutlookBreakTable_NonBreakingSpace(t *testing.T) {
	result := renderOutlookBreakTable("15px")

	htmlContent := result.TextContent

	assert.Contains(t, htmlContent, "&nbsp;")
}

func TestLineBreak_Transform_InlineBr_InParagraph(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		Build()

	ctx := NewTestContext().
		WithParentComponent(NewParagraph()).
		Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, "br", result.TagName)
	assert.Empty(t, result.Children)
}

func TestLineBreak_Transform_InlineBr_InListItem(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		Build()

	ctx := NewTestContext().
		WithParentComponent(NewListItem()).
		Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)
	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, "br", result.TagName)
}

func TestLineBreak_Transform_InlineBr_InButton(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		Build()

	ctx := NewTestContext().
		WithParentComponent(NewButton()).
		Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)
	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, "br", result.TagName)
}

func TestLineBreak_Transform_BlockSpacer_InColumn(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		Build()

	ctx := NewTestContext().
		WithParentComponent(NewColumn()).
		Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	require.Len(t, result.Children, 2)
}

func TestLineBreak_Transform_BlockSpacer_InHero(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		Build()

	ctx := NewTestContext().
		WithParentComponent(NewHero()).
		Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)
	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	require.Len(t, result.Children, 2)
}

func TestLineBreak_Transform_ExplicitHeight_OverridesInlineContext(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		WithAttribute(AttrHeight, "30px").
		Build()

	ctx := NewTestContext().
		WithParentComponent(NewParagraph()).
		Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	require.Len(t, result.Children, 2)

	outlookTable := result.Children[0]
	assert.Contains(t, outlookTable.TextContent, "height:30px;")
}

func TestLineBreak_Transform_NoParent_DefaultsToBlock(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		Build()

	ctx := NewTestContext().Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	require.Len(t, result.Children, 2)
}

func TestLineBreak_Transform_InlineBr_PreservesPikoDirectives(t *testing.T) {
	br := NewLineBreak()
	node := NewTestNode().
		WithTagName("pml-br").
		Build()

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showBreak"},
	}

	ctx := NewTestContext().
		WithParentComponent(NewParagraph()).
		Build(node, br)

	result, errs := br.Transform(node, ctx)

	require.Nil(t, errs)
	assert.Equal(t, "br", result.TagName)
	assert.NotNil(t, result.DirIf)
	assert.Equal(t, "showBreak", result.DirIf.Expression.(*ast_domain.Identifier).Name)
}

func TestLineBreak_Transform_ContextAware_TableDriven(t *testing.T) {
	testCases := []struct {
		parentComponent pml_domain.Component
		name            string
		height          string
		description     string
		expectInline    bool
	}{
		{
			name:            "no parent defaults to block",
			parentComponent: nil,
			height:          "",
			expectInline:    false,
			description:     "Without a parent component, should output block-level spacer",
		},
		{
			name:            "in paragraph is inline",
			parentComponent: NewParagraph(),
			height:          "",
			expectInline:    true,
			description:     "Inside pml-p should output simple <br>",
		},
		{
			name:            "in list item is inline",
			parentComponent: NewListItem(),
			height:          "",
			expectInline:    true,
			description:     "Inside pml-li should output simple <br>",
		},
		{
			name:            "in button is inline",
			parentComponent: NewButton(),
			height:          "",
			expectInline:    true,
			description:     "Inside pml-button should output simple <br>",
		},
		{
			name:            "in column is block",
			parentComponent: NewColumn(),
			height:          "",
			expectInline:    false,
			description:     "Inside pml-col should output block-level spacer",
		},
		{
			name:            "in hero is block",
			parentComponent: NewHero(),
			height:          "",
			expectInline:    false,
			description:     "Inside pml-hero should output block-level spacer",
		},
		{
			name:            "explicit height in paragraph overrides to block",
			parentComponent: NewParagraph(),
			height:          "20px",
			expectInline:    false,
			description:     "Explicit height should always output block-level spacer",
		},
		{
			name:            "explicit height in column is block",
			parentComponent: NewColumn(),
			height:          "30px",
			expectInline:    false,
			description:     "Explicit height in column should output block-level spacer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			br := NewLineBreak()
			nodeBuilder := NewTestNode().WithTagName("pml-br")

			if tc.height != "" {
				nodeBuilder = nodeBuilder.WithAttribute(AttrHeight, tc.height)
			}
			node := nodeBuilder.Build()

			ctxBuilder := NewTestContext()
			if tc.parentComponent != nil {
				ctxBuilder = ctxBuilder.WithParentComponent(tc.parentComponent)
			}
			ctx := ctxBuilder.Build(node, br)

			result, errs := br.Transform(node, ctx)

			require.Nil(t, errs, tc.description)
			require.NotNil(t, result, tc.description)

			if tc.expectInline {
				assert.Equal(t, ast_domain.NodeElement, result.NodeType, tc.description)
				assert.Equal(t, "br", result.TagName, tc.description)
			} else {
				assert.Equal(t, ast_domain.NodeFragment, result.NodeType, tc.description)
				require.Len(t, result.Children, 2, tc.description)
			}
		})
	}
}

func TestIsInlineContext(t *testing.T) {
	testCases := []struct {
		parent   pml_domain.Component
		name     string
		expected bool
	}{
		{name: "nil parent", parent: nil, expected: false},
		{name: "pml-p parent", parent: NewParagraph(), expected: true},
		{name: "pml-li parent", parent: NewListItem(), expected: true},
		{name: "pml-button parent", parent: NewButton(), expected: true},
		{name: "pml-col parent", parent: NewColumn(), expected: false},
		{name: "pml-hero parent", parent: NewHero(), expected: false},
		{name: "pml-row parent", parent: NewSection(), expected: false},
		{name: "pml-container parent", parent: NewContainer(), expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			br := NewLineBreak()
			node := NewTestNode().WithTagName("pml-br").Build()

			ctx := NewTestContext().
				WithParentComponent(tc.parent).
				Build(node, br)

			result := isInlineContext(ctx)

			assert.Equal(t, tc.expected, result)
		})
	}
}
