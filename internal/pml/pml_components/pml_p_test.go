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

func TestParagraph_TagName(t *testing.T) {
	p := NewParagraph()
	assert.Equal(t, "pml-p", p.TagName())
}

func TestParagraph_IsEndingTag(t *testing.T) {
	p := NewParagraph()
	assert.True(t, p.IsEndingTag())
}

func TestParagraph_AllowedParents(t *testing.T) {
	p := NewParagraph()
	parents := p.AllowedParents()

	require.Len(t, parents, 2)
	assert.Contains(t, parents, "pml-col")
	assert.Contains(t, parents, "pml-hero")
}

func TestParagraph_AllowedAttributes(t *testing.T) {
	p := NewParagraph()
	attrs := p.AllowedAttributes()

	require.NotEmpty(t, attrs)

	assert.Contains(t, attrs, AttrAlign)
	assert.Contains(t, attrs, AttrColour)
	assert.Contains(t, attrs, CSSFontFamily)
	assert.Contains(t, attrs, CSSFontSize)
	assert.Contains(t, attrs, CSSFontStyle)
	assert.Contains(t, attrs, CSSFontWeight)
	assert.Contains(t, attrs, CSSLineHeight)
	assert.Contains(t, attrs, "letter-spacing")
	assert.Contains(t, attrs, CSSTextDecoration)
	assert.Contains(t, attrs, CSSTextTransform)

	assert.Contains(t, attrs, AttrHeight)
	assert.Contains(t, attrs, AttrContainerBackgroundColor)
	assert.Contains(t, attrs, AttrPadding)
	assert.Contains(t, attrs, "padding-top")
	assert.Contains(t, attrs, "padding-bottom")
	assert.Contains(t, attrs, "padding-left")
	assert.Contains(t, attrs, "padding-right")

	alignDef := attrs[AttrAlign]
	assert.Equal(t, pml_domain.TypeEnum, alignDef.Type)
	require.Len(t, alignDef.AllowedValues, 4)
	assert.Contains(t, alignDef.AllowedValues, "left")
	assert.Contains(t, alignDef.AllowedValues, "right")
	assert.Contains(t, alignDef.AllowedValues, ValueCentre)
	assert.Contains(t, alignDef.AllowedValues, "justify")
}

func TestParagraph_DefaultAttributes(t *testing.T) {
	p := NewParagraph()
	defaults := p.DefaultAttributes()

	require.NotEmpty(t, defaults)
	assert.Equal(t, defaultParagraphAlign, defaults[AttrAlign])
	assert.Equal(t, defaultParagraphColor, defaults[AttrColour])
	assert.Equal(t, defaultParagraphFontFamily, defaults[CSSFontFamily])
	assert.Equal(t, defaultParagraphFontSize, defaults[CSSFontSize])
	assert.Equal(t, defaultParagraphLineHeight, defaults[CSSLineHeight])
	assert.Equal(t, defaultParagraphPadding, defaults[AttrPadding])
}

func TestParagraph_GetStyleTargets(t *testing.T) {
	p := NewParagraph()
	targets := p.GetStyleTargets()

	require.Len(t, targets, 21)

	for _, target := range targets {
		assert.Equal(t, TargetContainer, target.Target)
	}

	targetProps := make([]string, len(targets))
	for i, target := range targets {
		targetProps[i] = target.Property
	}

	assert.Contains(t, targetProps, AttrAlign)
	assert.Contains(t, targetProps, AttrColour)
	assert.Contains(t, targetProps, CSSFontFamily)
	assert.Contains(t, targetProps, CSSFontSize)
	assert.Contains(t, targetProps, CSSFontStyle)
	assert.Contains(t, targetProps, CSSFontWeight)
	assert.Contains(t, targetProps, CSSLineHeight)
	assert.Contains(t, targetProps, "letter-spacing")
	assert.Contains(t, targetProps, AttrHeight)
	assert.Contains(t, targetProps, CSSTextDecoration)
	assert.Contains(t, targetProps, CSSTextTransform)
	assert.Contains(t, targetProps, AttrPadding)
	assert.Contains(t, targetProps, AttrContainerBackgroundColor)
}

func TestParagraph_Transform_DefaultStyles(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Hello World")
	node := NewTestNode().
		WithTagName("pml-p").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, ElementDiv, result.TagName)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.NotContains(t, styleAttr.Value, "font-family:")
	assert.Contains(t, styleAttr.Value, "font-size:13px;")
	assert.Contains(t, styleAttr.Value, "line-height:1;")
	assert.NotContains(t, styleAttr.Value, "text-align:")
	assert.Contains(t, styleAttr.Value, "color:#000000;")

	require.Len(t, result.Children, 1)
	assert.Equal(t, "Hello World", result.Children[0].TextContent)
}

func TestParagraph_Transform_CustomTypography(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Custom Text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute(CSSFontFamily, "Arial, sans-serif").
		WithAttribute(CSSFontSize, "16px").
		WithAttribute(CSSFontWeight, "bold").
		WithAttribute(CSSFontStyle, "italic").
		WithAttribute(CSSLineHeight, "1.5").
		WithAttribute(AttrColour, "#ff0000").
		WithAttribute(AttrAlign, "center").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "font-family:Arial, sans-serif;")
	assert.Contains(t, styleAttr.Value, "font-size:16px;")
	assert.Contains(t, styleAttr.Value, "font-weight:bold;")
	assert.Contains(t, styleAttr.Value, "font-style:italic;")
	assert.Contains(t, styleAttr.Value, "line-height:1.5;")
	assert.Contains(t, styleAttr.Value, "color:#ff0000;")
	assert.Contains(t, styleAttr.Value, "text-align:center;")
}

func TestParagraph_Transform_TextDecoration(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Underlined Text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute(CSSTextDecoration, "underline").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "text-decoration:underline;")
}

func TestParagraph_Transform_TextTransform(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("uppercase text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute(CSSTextTransform, "uppercase").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "text-transform:uppercase;")
}

func TestParagraph_Transform_LetterSpacing(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Spaced Text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute("letter-spacing", "2px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "letter-spacing:2px;")
}

func TestParagraph_Transform_DataAttributesForPadding(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute(AttrPadding, "20px 30px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	paddingAttr, found := FindAttribute(result, "data-pml-padding")
	require.True(t, found)
	assert.Equal(t, "20px 30px", paddingAttr.Value)

	_, found = FindAttribute(result, "data-pml-align")
	assert.False(t, found, "data-pml-align should not be present when align is not explicitly set")
}

func TestParagraph_Transform_DataAttributesForDirectionalPadding(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute(AttrPadding, "10px").
		WithAttribute("padding-top", "20px").
		WithAttribute("padding-right", "30px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	paddingAttr, found := FindAttribute(result, "data-pml-padding")
	require.True(t, found)
	assert.Equal(t, "10px", paddingAttr.Value)

	topAttr, found := FindAttribute(result, "data-pml-padding-top")
	require.True(t, found)
	assert.Equal(t, "20px", topAttr.Value)

	rightAttr, found := FindAttribute(result, "data-pml-padding-right")
	require.True(t, found)
	assert.Equal(t, "30px", rightAttr.Value)

	_, found = FindAttribute(result, "data-pml-padding-bottom")
	assert.False(t, found)

	_, found = FindAttribute(result, "data-pml-padding-left")
	assert.False(t, found)
}

func TestParagraph_Transform_DataAttributeForContainerBackground(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute(AttrContainerBackgroundColor, "#f0f0f0").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	bgAttr, found := FindAttribute(result, "data-pml-container-background-color")
	require.True(t, found)
	assert.Equal(t, "#f0f0f0", bgAttr.Value)
}

func TestParagraph_Transform_WithHeight_NoOutlookWrapper(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, ElementDiv, result.TagName)
}

func TestParagraph_Transform_WithHeight_OutlookWrapper(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute(AttrHeight, "100px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	require.Len(t, result.Children, 3)

	startComment := result.Children[0]
	assert.Equal(t, ast_domain.NodeRawHTML, startComment.NodeType)
	assert.Contains(t, startComment.TextContent, "<!--[if mso | IE]>")
	assert.Contains(t, startComment.TextContent, `height="100"`)
	assert.Contains(t, startComment.TextContent, "height:100px")
	assert.Contains(t, startComment.TextContent, "vertical-align:top")

	divNode := result.Children[1]
	assert.Equal(t, ast_domain.NodeElement, divNode.NodeType)
	assert.Equal(t, ElementDiv, divNode.TagName)
	require.Len(t, divNode.Children, 1)
	assert.Equal(t, "Text", divNode.Children[0].TextContent)

	endComment := result.Children[2]
	assert.Equal(t, ast_domain.NodeRawHTML, endComment.NodeType)
	assert.Contains(t, endComment.TextContent, "<![endif]-->")
}

func TestParagraph_Transform_PreservesChildrenStructure(t *testing.T) {
	p := NewParagraph()

	boldText := NewElementNode("b", nil, []*ast_domain.TemplateNode{
		NewSimpleTextNode("Bold"),
	})
	plainText := NewSimpleTextNode(" and ")
	italicText := NewElementNode("i", nil, []*ast_domain.TemplateNode{
		NewSimpleTextNode("Italic"),
	})

	node := NewTestNode().
		WithTagName("pml-p").
		WithChildren(boldText, plainText, italicText).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	require.Len(t, result.Children, 3)

	assert.Equal(t, "b", result.Children[0].TagName)
	assert.Equal(t, "Bold", result.Children[0].Children[0].TextContent)

	assert.Equal(t, " and ", result.Children[1].TextContent)

	assert.Equal(t, "i", result.Children[2].TagName)
	assert.Equal(t, "Italic", result.Children[2].Children[0].TextContent)
}

func TestParagraph_Transform_PreservesPikoDirectives_NoHeight(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithChildren(textNode).
		Build()

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showText"},
	}

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.NotNil(t, result.DirIf)
	assert.Equal(t, "showText", result.DirIf.Expression.(*ast_domain.Identifier).Name)

	assert.Nil(t, node.DirIf)
}

func TestParagraph_Transform_PreservesPikoDirectives_WithHeight(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute(AttrHeight, "50px").
		WithChildren(textNode).
		Build()

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showText"},
	}

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	assert.NotNil(t, result.DirIf)
	assert.Equal(t, "showText", result.DirIf.Expression.(*ast_domain.Identifier).Name)

	assert.Nil(t, node.DirIf)
}

func TestParagraph_Transform_RightAlignment(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Right aligned text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute(AttrAlign, "right").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "text-align:right;")

	alignAttr, found := FindAttribute(result, "data-pml-align")
	require.True(t, found)
	assert.Equal(t, "right", alignAttr.Value)
}

func TestParagraph_Transform_JustifyAlignment(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Justified text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute(AttrAlign, "justify").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "text-align:justify;")
}

func TestRenderOutlookParagraphHeightWrapper_BasicStructure(t *testing.T) {
	contentNode := NewElementNode(ElementDiv, nil, []*ast_domain.TemplateNode{
		NewSimpleTextNode("Content"),
	})

	result := renderOutlookParagraphHeightWrapper("80px", contentNode)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	require.Len(t, result.Children, 3)

	startComment := result.Children[0]
	assert.Equal(t, ast_domain.NodeRawHTML, startComment.NodeType)
	assert.True(t, strings.HasPrefix(startComment.TextContent, "<!--[if mso | IE]>"))
	assert.Contains(t, startComment.TextContent, "<table")
	assert.Contains(t, startComment.TextContent, `role="presentation"`)
	assert.Contains(t, startComment.TextContent, `border="0"`)
	assert.Contains(t, startComment.TextContent, `cellpadding="0"`)
	assert.Contains(t, startComment.TextContent, `cellspacing="0"`)
	assert.Contains(t, startComment.TextContent, `height="80"`)
	assert.Contains(t, startComment.TextContent, "height:80px")
	assert.Contains(t, startComment.TextContent, "vertical-align:top")
	assert.Contains(t, startComment.TextContent, "<![endif]-->")

	assert.Equal(t, contentNode, result.Children[1])

	endComment := result.Children[2]
	assert.Equal(t, ast_domain.NodeRawHTML, endComment.NodeType)
	assert.Equal(t, "<!--[if mso | IE]></td></tr></table><![endif]-->", endComment.TextContent)
}

func TestRenderOutlookParagraphHeightWrapper_DifferentHeights(t *testing.T) {
	testCases := []struct {
		name       string
		height     string
		expectedPx string
	}{
		{name: "50px", height: "50px", expectedPx: `height="50"`},
		{name: "100px", height: "100px", expectedPx: `height="100"`},
		{name: "200px", height: "200px", expectedPx: `height="200"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contentNode := NewElementNode(ElementDiv, nil, nil)
			result := renderOutlookParagraphHeightWrapper(tc.height, contentNode)

			startComment := result.Children[0]
			assert.Contains(t, startComment.TextContent, tc.expectedPx)
			assert.Contains(t, startComment.TextContent, "height:"+tc.height)
		})
	}
}

func TestParagraph_Transform_ComplexTypography(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Complex styled text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute(CSSFontFamily, "Georgia, serif").
		WithAttribute(CSSFontSize, "18px").
		WithAttribute(CSSFontWeight, "300").
		WithAttribute(CSSFontStyle, "normal").
		WithAttribute(CSSLineHeight, "1.8").
		WithAttribute("letter-spacing", "1px").
		WithAttribute(AttrColour, "#333333").
		WithAttribute(AttrAlign, "center").
		WithAttribute(CSSTextDecoration, "none").
		WithAttribute(CSSTextTransform, "capitalize").
		WithAttribute(AttrPadding, "15px 20px").
		WithAttribute(AttrContainerBackgroundColor, "#fafafa").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "font-family:Georgia, serif;")
	assert.Contains(t, styleAttr.Value, "font-size:18px;")
	assert.Contains(t, styleAttr.Value, "font-weight:300;")
	assert.Contains(t, styleAttr.Value, "font-style:normal;")
	assert.Contains(t, styleAttr.Value, "line-height:1.8;")
	assert.Contains(t, styleAttr.Value, "letter-spacing:1px;")
	assert.Contains(t, styleAttr.Value, "color:#333333;")
	assert.Contains(t, styleAttr.Value, "text-align:center;")
	assert.Contains(t, styleAttr.Value, "text-decoration:none;")
	assert.Contains(t, styleAttr.Value, "text-transform:capitalize;")

	paddingAttr, found := FindAttribute(result, "data-pml-padding")
	require.True(t, found)
	assert.Equal(t, "15px 20px", paddingAttr.Value)

	bgAttr, found := FindAttribute(result, "data-pml-container-background-color")
	require.True(t, found)
	assert.Equal(t, "#fafafa", bgAttr.Value)

	alignAttr, found := FindAttribute(result, "data-pml-align")
	require.True(t, found)
	assert.Equal(t, "center", alignAttr.Value)
}

func TestParagraph_Transform_WithHeightAndComplexContent(t *testing.T) {
	p := NewParagraph()

	link := NewElementNode("a", []ast_domain.HTMLAttribute{
		NewHTMLAttribute("href", "https://example.com"),
	}, []*ast_domain.TemplateNode{
		NewSimpleTextNode("Click here"),
	})

	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute(AttrHeight, "120px").
		WithAttribute(CSSFontSize, "14px").
		WithAttribute(AttrAlign, "center").
		WithChildren(
			NewSimpleTextNode("Visit "),
			link,
			NewSimpleTextNode(" for more info."),
		).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	require.Len(t, result.Children, 3)

	divNode := result.Children[1]
	assert.Equal(t, ElementDiv, divNode.TagName)

	require.Len(t, divNode.Children, 3)
	assert.Equal(t, "Visit ", divNode.Children[0].TextContent)
	assert.Equal(t, "a", divNode.Children[1].TagName)
	assert.Equal(t, " for more info.", divNode.Children[2].TextContent)
}

func TestParagraph_Transform_AllDirectionalPadding(t *testing.T) {
	p := NewParagraph()
	textNode := NewSimpleTextNode("Text")
	node := NewTestNode().
		WithTagName("pml-p").
		WithAttribute("padding-top", "5px").
		WithAttribute("padding-right", "10px").
		WithAttribute("padding-bottom", "15px").
		WithAttribute("padding-left", "20px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	topAttr, found := FindAttribute(result, "data-pml-padding-top")
	require.True(t, found)
	assert.Equal(t, "5px", topAttr.Value)

	rightAttr, found := FindAttribute(result, "data-pml-padding-right")
	require.True(t, found)
	assert.Equal(t, "10px", rightAttr.Value)

	bottomAttr, found := FindAttribute(result, "data-pml-padding-bottom")
	require.True(t, found)
	assert.Equal(t, "15px", bottomAttr.Value)

	leftAttr, found := FindAttribute(result, "data-pml-padding-left")
	require.True(t, found)
	assert.Equal(t, "20px", leftAttr.Value)
}

func TestParagraph_Transform_ConvertsPmlBrToBr(t *testing.T) {
	p := NewParagraph()
	node := NewTestNode().
		WithTagName("pml-p").
		WithChildren(
			NewSimpleTextNode("Line one"),
			NewTestNode().WithTagName("pml-br").Build(),
			NewSimpleTextNode("Line two"),
		).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	require.Len(t, result.Children, 3)
	assert.Equal(t, ast_domain.NodeText, result.Children[0].NodeType)
	assert.Equal(t, "Line one", result.Children[0].TextContent)

	assert.Equal(t, ast_domain.NodeElement, result.Children[1].NodeType)
	assert.Equal(t, "br", result.Children[1].TagName)

	assert.Equal(t, ast_domain.NodeText, result.Children[2].NodeType)
	assert.Equal(t, "Line two", result.Children[2].TextContent)
}

func TestParagraph_Transform_ConvertsPmlBrInsideNestedElements(t *testing.T) {
	p := NewParagraph()

	strongNode := NewElementNode("strong", nil, []*ast_domain.TemplateNode{
		NewSimpleTextNode("bold"),
		NewTestNode().WithTagName("pml-br").Build(),
		NewSimpleTextNode("more"),
	})

	node := NewTestNode().
		WithTagName("pml-p").
		WithChildren(
			NewSimpleTextNode("Text "),
			strongNode,
		).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)
	require.Len(t, result.Children, 2)

	strong := result.Children[1]
	assert.Equal(t, "strong", strong.TagName)
	require.Len(t, strong.Children, 3)

	assert.Equal(t, "bold", strong.Children[0].TextContent)
	assert.Equal(t, "br", strong.Children[1].TagName)
	assert.Equal(t, "more", strong.Children[2].TextContent)
}

func TestParagraph_Transform_MultiplePmlBrElements(t *testing.T) {
	p := NewParagraph()
	node := NewTestNode().
		WithTagName("pml-p").
		WithChildren(
			NewSimpleTextNode("Line 1"),
			NewTestNode().WithTagName("pml-br").Build(),
			NewSimpleTextNode("Line 2"),
			NewTestNode().WithTagName("pml-br").Build(),
			NewSimpleTextNode("Line 3"),
		).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)
	require.Len(t, result.Children, 5)

	assert.Equal(t, "br", result.Children[1].TagName)
	assert.Equal(t, "br", result.Children[3].TagName)
}

func TestParagraph_Transform_PmlBrPreservesPikoDirectives(t *testing.T) {
	p := NewParagraph()

	brNode := NewTestNode().WithTagName("pml-br").Build()
	brNode.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showBreak"},
	}

	node := NewTestNode().
		WithTagName("pml-p").
		WithChildren(
			NewSimpleTextNode("Before"),
			brNode,
			NewSimpleTextNode("After"),
		).
		Build()

	ctx := NewTestContext().Build(node, p)

	result, errs := p.Transform(node, ctx)

	require.Nil(t, errs)

	convertedBr := result.Children[1]
	assert.Equal(t, "br", convertedBr.TagName)
	assert.NotNil(t, convertedBr.DirIf)
	assert.Equal(t, "showBreak", convertedBr.DirIf.Expression.(*ast_domain.Identifier).Name)
}

func TestProcessInlineChildren_EmptySlice(t *testing.T) {
	result := processInlineChildren(nil)
	assert.Nil(t, result)

	result = processInlineChildren([]*ast_domain.TemplateNode{})
	assert.Empty(t, result)
}

func TestProcessInlineChildren_PreservesNonPmlElements(t *testing.T) {
	children := []*ast_domain.TemplateNode{
		NewSimpleTextNode("text"),
		NewElementNode("strong", nil, []*ast_domain.TemplateNode{NewSimpleTextNode("bold")}),
		NewElementNode("a", []ast_domain.HTMLAttribute{NewHTMLAttribute("href", "url")}, nil),
	}

	result := processInlineChildren(children)

	require.Len(t, result, 3)
	assert.Equal(t, "text", result[0].TextContent)
	assert.Equal(t, "strong", result[1].TagName)
	assert.Equal(t, "a", result[2].TagName)
}
