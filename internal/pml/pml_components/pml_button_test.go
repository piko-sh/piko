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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

func TestButton_TagName(t *testing.T) {
	btn := NewButton()
	assert.Equal(t, "pml-button", btn.TagName())
}

func TestButton_IsEndingTag(t *testing.T) {
	btn := NewButton()
	assert.True(t, btn.IsEndingTag())
}

func TestButton_AllowedParents(t *testing.T) {
	btn := NewButton()
	parents := btn.AllowedParents()

	require.Len(t, parents, 2)
	assert.Contains(t, parents, "pml-col")
	assert.Contains(t, parents, "pml-hero")
}

func TestButton_AllowedAttributes(t *testing.T) {
	btn := NewButton()
	attrs := btn.AllowedAttributes()

	require.NotEmpty(t, attrs)

	assert.Contains(t, attrs, AttrHref)
	assert.Contains(t, attrs, AttrTarget)
	assert.Contains(t, attrs, AttrTitle)
	assert.Contains(t, attrs, AttrRel)

	assert.Contains(t, attrs, AttrAlign)
	assert.Contains(t, attrs, AttrWidth)
	assert.Contains(t, attrs, AttrHeight)
	assert.Contains(t, attrs, AttrPadding)
	assert.Contains(t, attrs, AttrInnerPadding)
	assert.Contains(t, attrs, CSSVerticalAlign)

	assert.Contains(t, attrs, CSSBackgroundColor)
	assert.Contains(t, attrs, AttrBorder)
	assert.Contains(t, attrs, AttrBorderRadius)
	assert.Contains(t, attrs, AttrColour)
	assert.Contains(t, attrs, AttrContainerBackgroundColor)

	assert.Contains(t, attrs, CSSFontFamily)
	assert.Contains(t, attrs, CSSFontSize)
	assert.Contains(t, attrs, CSSFontWeight)
	assert.Contains(t, attrs, CSSLineHeight)
	assert.Contains(t, attrs, CSSTextDecoration)
	assert.Contains(t, attrs, CSSTextTransform)

	alignDef := attrs[AttrAlign]
	assert.Equal(t, pml_domain.TypeEnum, alignDef.Type)
	require.Len(t, alignDef.AllowedValues, 3)
	assert.Contains(t, alignDef.AllowedValues, ValueLeft)
	assert.Contains(t, alignDef.AllowedValues, ValueCentre)
	assert.Contains(t, alignDef.AllowedValues, ValueRight)

	valignDef := attrs[CSSVerticalAlign]
	assert.Equal(t, pml_domain.TypeEnum, valignDef.Type)
	require.Len(t, valignDef.AllowedValues, 3)
	assert.Contains(t, valignDef.AllowedValues, ValueTop)
	assert.Contains(t, valignDef.AllowedValues, ValueMiddle)
	assert.Contains(t, valignDef.AllowedValues, ValueBottom)
}

func TestButton_DefaultAttributes(t *testing.T) {
	btn := NewButton()
	defaults := btn.DefaultAttributes()

	require.NotEmpty(t, defaults)
	assert.Equal(t, defaultButtonAlign, defaults[AttrAlign])
	assert.Equal(t, defaultButtonBgColor, defaults[CSSBackgroundColor])
	assert.Equal(t, defaultButtonBorder, defaults[AttrBorder])
	assert.Equal(t, defaultButtonBorderRadius, defaults[AttrBorderRadius])
	assert.Equal(t, defaultButtonColor, defaults[AttrColour])
	assert.Equal(t, defaultButtonFontFamily, defaults[CSSFontFamily])
	assert.Equal(t, defaultButtonFontSize, defaults[CSSFontSize])
	assert.Equal(t, defaultButtonFontWeight, defaults[CSSFontWeight])
	assert.Equal(t, defaultButtonInnerPadding, defaults[AttrInnerPadding])
	assert.Equal(t, defaultButtonLineHeight, defaults[CSSLineHeight])
	assert.Equal(t, defaultButtonPadding, defaults[AttrPadding])
	assert.Equal(t, defaultButtonTarget, defaults[AttrTarget])
	assert.Equal(t, defaultButtonTextDecoration, defaults[CSSTextDecoration])
	assert.Equal(t, defaultButtonTextTransform, defaults[CSSTextTransform])
	assert.Equal(t, defaultButtonVerticalAlign, defaults[CSSVerticalAlign])
}

func TestButton_GetStyleTargets(t *testing.T) {
	btn := NewButton()
	targets := btn.GetStyleTargets()

	require.Len(t, targets, 16)

	targetMap := make(map[string]string)
	for _, target := range targets {
		targetMap[target.Property] = target.Target
	}

	assert.Equal(t, TargetContainer, targetMap[AttrAlign])
	assert.Equal(t, TargetContainer, targetMap[AttrWidth])
	assert.Equal(t, TargetContainer, targetMap[AttrPadding])

	assert.Equal(t, TargetCell, targetMap[CSSBackgroundColor])
	assert.Equal(t, TargetCell, targetMap[AttrBorderRadius])
	assert.Equal(t, TargetCell, targetMap[AttrBorder])
	assert.Equal(t, TargetCell, targetMap[AttrHeight])
	assert.Equal(t, TargetCell, targetMap[CSSVerticalAlign])

	assert.Equal(t, TargetLink, targetMap[AttrInnerPadding])
	assert.Equal(t, TargetLink, targetMap[AttrColour])
	assert.Equal(t, TargetLink, targetMap[CSSFontFamily])
	assert.Equal(t, TargetLink, targetMap[CSSFontSize])
	assert.Equal(t, TargetLink, targetMap[CSSFontWeight])
	assert.Equal(t, TargetLink, targetMap[CSSLineHeight])
	assert.Equal(t, TargetLink, targetMap[CSSTextDecoration])
	assert.Equal(t, TargetLink, targetMap[CSSTextTransform])
}

func TestButton_Transform_LinkedButton(t *testing.T) {
	btn := NewButton()
	textNode := NewSimpleTextNode("Click Here")
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrHref, "https://example.com").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, btn)

	result, errs := btn.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	require.Len(t, result.Children, 4)

	vmlNode := result.Children[0]
	assert.Equal(t, ast_domain.NodeRawHTML, vmlNode.NodeType)
	assert.Contains(t, vmlNode.TextContent, "<!--[if mso | IE]>")
	assert.Contains(t, vmlNode.TextContent, "<v:roundrect")
	assert.Contains(t, vmlNode.TextContent, "href=\"https://example.com\"")
	assert.Contains(t, vmlNode.TextContent, "Click Here")

	assert.Contains(t, result.Children[1].TextContent, "<!--[if !mso")

	tableNode := result.Children[2]
	assert.Equal(t, ast_domain.NodeElement, tableNode.NodeType)
	assert.Equal(t, ElementTable, tableNode.TagName)

	tbody := tableNode.Children[0]
	tr := tbody.Children[0]
	td := tr.Children[0]
	linkNode := td.Children[0]

	assert.Equal(t, ElementA, linkNode.TagName)
	hrefAttr, found := FindAttribute(linkNode, AttrHref)
	require.True(t, found)
	assert.Equal(t, "https://example.com", hrefAttr.Value)

	assert.Contains(t, result.Children[3].TextContent, "<![endif]-->")
}

func TestButton_Transform_NonLinkedButton(t *testing.T) {
	btn := NewButton()
	textNode := NewSimpleTextNode("Button Text")
	node := NewTestNode().
		WithTagName("pml-button").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, btn)

	result, errs := btn.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, ElementTable, result.TagName)

	tbody := result.Children[0]
	tr := tbody.Children[0]
	td := tr.Children[0]
	pNode := td.Children[0]

	assert.Equal(t, ElementP, pNode.TagName)
	require.Len(t, pNode.Children, 1)
	assert.Equal(t, "Button Text", pNode.Children[0].TextContent)
}

func TestButton_Transform_PreservesPikoDirectives(t *testing.T) {
	btn := NewButton()
	textNode := NewSimpleTextNode("Text")
	node := NewTestNode().
		WithTagName("pml-button").
		WithChildren(textNode).
		Build()

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showButton"},
	}

	ctx := NewTestContext().Build(node, btn)

	result, errs := btn.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result.DirIf)
	assert.Equal(t, "showButton", result.DirIf.Expression.(*ast_domain.Identifier).Name)

	assert.Nil(t, node.DirIf)
}

func TestButton_Transform_DataAttributesForPadding(t *testing.T) {
	btn := NewButton()
	textNode := NewSimpleTextNode("Text")
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrPadding, "15px 20px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, btn)

	result, errs := btn.Transform(node, ctx)

	require.Nil(t, errs)

	paddingAttr, found := FindAttribute(result, "data-pml-padding")
	require.True(t, found)
	assert.Equal(t, "15px 20px", paddingAttr.Value)
}

func TestButton_Transform_CustomStyles(t *testing.T) {
	btn := NewButton()
	textNode := NewSimpleTextNode("Styled Button")
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrHref, "https://example.com").
		WithAttribute(CSSBackgroundColor, "#ff0000").
		WithAttribute(AttrColour, "#ffffff").
		WithAttribute(CSSFontSize, "18px").
		WithAttribute(AttrBorderRadius, "10px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, btn)

	result, errs := btn.Transform(node, ctx)

	require.Nil(t, errs)

	tableNode := result.Children[2]
	tbody := tableNode.Children[0]
	tr := tbody.Children[0]
	td := tr.Children[0]
	linkNode := td.Children[0]

	styleAttr, found := FindAttribute(linkNode, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "background:#ff0000;")
	assert.Contains(t, styleAttr.Value, "color:#ffffff;")
	assert.Contains(t, styleAttr.Value, "font-size:18px;")
	assert.Contains(t, styleAttr.Value, "border-radius:10px;")
}

func TestExtractTextContent_PlainText(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		NewSimpleTextNode("Hello World"),
	}

	result := extractTextContent(nodes)

	assert.Equal(t, "Hello World", result)
}

func TestExtractTextContent_MixedWithElements(t *testing.T) {
	spanNode := NewElementNode("span", nil, []*ast_domain.TemplateNode{
		NewSimpleTextNode("Bold"),
	})

	nodes := []*ast_domain.TemplateNode{
		NewSimpleTextNode("Click "),
		spanNode,
		NewSimpleTextNode(" here"),
	}

	result := extractTextContent(nodes)

	assert.Equal(t, "Click Bold here", result)
}

func TestExtractTextContent_NestedElements(t *testing.T) {
	innerSpan := NewElementNode("strong", nil, []*ast_domain.TemplateNode{
		NewSimpleTextNode("emphasized"),
	})
	outerSpan := NewElementNode("span", nil, []*ast_domain.TemplateNode{
		NewSimpleTextNode("text with "),
		innerSpan,
	})

	nodes := []*ast_domain.TemplateNode{outerSpan}

	result := extractTextContent(nodes)

	assert.Equal(t, "text with emphasized", result)
}

func TestExtractTextContent_NilNode(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{nil}

	result := extractTextContent(nodes)

	assert.Equal(t, "", result)
}

func TestGetStyleWithDefault_HasValue(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrTarget, "_self").
		Build()

	ctx := NewTestContext().Build(node, btn)

	result := getStyleWithDefault(ctx.StyleManager, AttrTarget, "_blank")

	assert.Equal(t, "_self", result)
}

func TestGetStyleWithDefault_NoValue(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		Build()

	ctx := NewTestContext().Build(node, btn)

	result := getStyleWithDefault(ctx.StyleManager, AttrTarget, "_blank")

	assert.Equal(t, "_blank", result)
}

func TestGetButtonPadding_InnerPaddingTakesPrecedence(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrPadding, "10px").
		WithAttribute(AttrInnerPadding, "20px 30px").
		Build()

	ctx := NewTestContext().Build(node, btn)

	result := getButtonPadding(ctx.StyleManager)

	assert.Equal(t, "20px 30px", result)
}

func TestGetButtonPadding_FallbackToPadding(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrPadding, "15px 25px").
		Build()

	ctx := NewTestContext().Build(node, btn)

	result := getButtonPadding(ctx.StyleManager)

	assert.Equal(t, "10px 25px", result)
}

func TestGetButtonPadding_DefaultValue(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		Build()

	ctx := NewTestContext().Build(node, btn)

	result := getButtonPadding(ctx.StyleManager)

	assert.Equal(t, defaultButtonInnerPadding, result)
}

func TestBuildLinkStyles_DefaultValues(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		Build()

	ctx := NewTestContext().Build(node, btn)

	styles := buildLinkStyles(ctx.StyleManager)

	assert.Equal(t, defaultButtonBgColor, styles[CSSBackground])
	assert.Equal(t, defaultButtonBorderRadius, styles[AttrBorderRadius])
	assert.Equal(t, defaultButtonColor, styles[AttrColour])
	assert.Equal(t, ValueInlineBlock, styles[CSSDisplay])
	assert.Equal(t, defaultButtonFontFamily, styles[CSSFontFamily])
	assert.Equal(t, defaultButtonFontSize, styles[CSSFontSize])
	assert.Equal(t, defaultButtonFontWeight, styles[CSSFontWeight])
	assert.Equal(t, defaultButtonLineHeight, styles[CSSLineHeight])
	assert.Equal(t, ValueZero, styles[CSSMargin])
	assert.Equal(t, defaultButtonInnerPadding, styles[CSSPadding])
	assert.Equal(t, defaultButtonTextDecoration, styles[CSSTextDecoration])
	assert.Equal(t, defaultButtonTextTransform, styles[CSSTextTransform])
}

func TestBuildLinkStyles_CustomValues(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(CSSBackgroundColor, "#00ff00").
		WithAttribute(AttrColour, "#000000").
		WithAttribute(CSSFontSize, "16px").
		Build()

	ctx := NewTestContext().Build(node, btn)

	styles := buildLinkStyles(ctx.StyleManager)

	assert.Equal(t, "#00ff00", styles[CSSBackground])
	assert.Equal(t, "#000000", styles[AttrColour])
	assert.Equal(t, "16px", styles[CSSFontSize])
}

func TestBuildCellStyles_WithBorder(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrBorder, "2px solid #000000").
		Build()

	ctx := NewTestContext().Build(node, btn)

	styles := buildCellStyles(ctx.StyleManager)

	assert.Equal(t, "2px solid #000000", styles[AttrBorder])
}

func TestBuildCellStyles_NoBorder(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		Build()

	ctx := NewTestContext().Build(node, btn)

	styles := buildCellStyles(ctx.StyleManager)

	assert.Equal(t, ValueNone, styles[AttrBorder])
}

func TestBuildCellStyles_WithHeight(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrHeight, "50px").
		Build()

	ctx := NewTestContext().Build(node, btn)

	styles := buildCellStyles(ctx.StyleManager)

	assert.Equal(t, "50px", styles[AttrHeight])
}

func TestBuildTableStyles_WithWidth(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrWidth, "300px").
		Build()

	ctx := NewTestContext().Build(node, btn)

	styles := buildTableStyles(ctx.StyleManager)

	assert.Equal(t, "300px", styles[AttrWidth])
	assert.Equal(t, ValueSeparate, styles[CSSBorderCollapse])
	assert.Equal(t, Value100, styles[CSSLineHeight])
}

func TestCalculateArcsize_NoRadius(t *testing.T) {
	result := calculateArcsize("0px", "40px")
	assert.Equal(t, "0%", result)
}

func TestCalculateArcsize_WithHeightAndRadius(t *testing.T) {
	testCases := []struct {
		name            string
		borderRadius    string
		height          string
		expectedArcsize string
	}{
		{name: "10% arcsize", borderRadius: "4px", height: "40px", expectedArcsize: "10%"},
		{name: "25% arcsize", borderRadius: "10px", height: "40px", expectedArcsize: "25%"},
		{name: "capped at 50%", borderRadius: "30px", height: "40px", expectedArcsize: "50%"},
		{name: "large radius", borderRadius: "25px", height: "30px", expectedArcsize: "50%"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateArcsize(tc.borderRadius, tc.height)
			assert.Equal(t, tc.expectedArcsize, result)
		})
	}
}

func TestCalculateArcsize_NoHeight(t *testing.T) {

	result := calculateArcsize("20px", "")
	assert.Equal(t, "50%", result)

	result = calculateArcsize("5px", "")
	assert.Equal(t, "10%", result)
}

func TestParseVMLStroke_HasBorder(t *testing.T) {
	strokeColor, strokeWeight, hasStroke := parseVMLStroke("2px solid #ff0000", "#000000")

	assert.True(t, hasStroke)
	assert.Equal(t, "2px", strokeWeight)
	assert.Equal(t, "#ff0000", strokeColor)
}

func TestParseVMLStroke_NoBorder(t *testing.T) {
	strokeColor, strokeWeight, hasStroke := parseVMLStroke("none", "#414141")

	assert.False(t, hasStroke)
	assert.Equal(t, "", strokeWeight)
	assert.Equal(t, "#414141", strokeColor)
}

func TestParseVMLStroke_EmptyBorder(t *testing.T) {
	strokeColor, strokeWeight, hasStroke := parseVMLStroke("", "#414141")

	assert.False(t, hasStroke)
	assert.Equal(t, "", strokeWeight)
	assert.Equal(t, "#414141", strokeColor)
}

func TestBuildButtonVMLRectStyle_WithBothDimensions(t *testing.T) {
	result := buildButtonVMLRectStyle("200px", "50px")

	assert.Contains(t, result, "height:50px;")
	assert.Contains(t, result, "width:200px;")
}

func TestBuildButtonVMLRectStyle_OnlyWidth(t *testing.T) {
	result := buildButtonVMLRectStyle("300px", "")

	assert.Contains(t, result, "width:300px;")
	assert.NotContains(t, result, "height:")
}

func TestBuildButtonVMLRectStyle_OnlyHeight(t *testing.T) {
	result := buildButtonVMLRectStyle("", "60px")

	assert.Contains(t, result, "height:60px;")
	assert.NotContains(t, result, "width:")
}

func TestBuildButtonVMLRectStyle_NoDimensions(t *testing.T) {
	result := buildButtonVMLRectStyle("", "")

	assert.Equal(t, "height:40px;width:200px;", result)
}

func TestBuildVMLCentreStyles(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrColour, "#ff0000").
		WithAttribute(CSSFontFamily, "Georgia, serif").
		WithAttribute(CSSFontSize, "16px").
		WithAttribute(CSSFontWeight, "bold").
		Build()

	ctx := NewTestContext().Build(node, btn)

	result := buildVMLCentreStyles(ctx.StyleManager)

	assert.Contains(t, result, "color:#ff0000;")
	assert.Contains(t, result, "font-family:Georgia, serif;")
	assert.Contains(t, result, "font-size:16px;")
	assert.Contains(t, result, "font-weight:bold;")
}

func TestBuildVMLAttributes_WithStroke(t *testing.T) {
	result := buildVMLAttributes(
		"https://example.com",
		"height:40px;width:200px;",
		"10%",
		true,
		"#000000",
		"2px",
		"#414141",
	)

	assert.Contains(t, result, `href="https://example.com"`)
	assert.Contains(t, result, `style="height:40px;width:200px;"`)
	assert.Contains(t, result, `arcsize="10%"`)
	assert.Contains(t, result, `strokecolor="#000000"`)
	assert.Contains(t, result, `strokeweight="2px"`)
	assert.Contains(t, result, `fillcolor="#414141"`)
}

func TestBuildVMLAttributes_NoStroke(t *testing.T) {
	result := buildVMLAttributes(
		"https://example.com",
		"height:40px;width:200px;",
		"10%",
		false,
		"",
		"",
		"#414141",
	)

	assert.Contains(t, result, `stroke="false"`)
	assert.NotContains(t, result, "strokecolor")
	assert.NotContains(t, result, "strokeweight")
}

func TestButton_RenderVML_BasicStructure(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrHref, "https://example.com").
		Build()

	ctx := NewTestContext().Build(node, btn)

	result := btn.renderVML(ctx.StyleManager, "Click Here", "https://example.com")

	assert.Equal(t, ast_domain.NodeRawHTML, result.NodeType)

	htmlContent := result.TextContent

	assert.Contains(t, htmlContent, "<!--[if mso | IE]>")
	assert.Contains(t, htmlContent, "<![endif]-->")

	assert.Contains(t, htmlContent, "<v:roundrect")
	assert.Contains(t, htmlContent, "<w:anchorlock/>")
	assert.Contains(t, htmlContent, "<v:textbox")
	assert.Contains(t, htmlContent, "<center")
	assert.Contains(t, htmlContent, "Click Here")
	assert.Contains(t, htmlContent, `href="https://example.com"`)
}

func TestButton_Transform_ComplexLinkedButton(t *testing.T) {
	btn := NewButton()
	textNode := NewSimpleTextNode("Get Started")
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrHref, "https://example.com/signup").
		WithAttribute(CSSBackgroundColor, "#3498db").
		WithAttribute(AttrColour, "#ffffff").
		WithAttribute(CSSFontSize, "16px").
		WithAttribute(CSSFontWeight, "bold").
		WithAttribute(AttrBorderRadius, "8px").
		WithAttribute(AttrWidth, "250px").
		WithAttribute(AttrHeight, "50px").
		WithAttribute(AttrInnerPadding, "15px 30px").
		WithAttribute(AttrAlign, "center").
		WithAttribute(AttrTarget, "_self").
		WithAttribute(AttrTitle, "Sign up now").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, btn)

	result, errs := btn.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	require.Len(t, result.Children, 4)

	vmlNode := result.Children[0]
	assert.Contains(t, vmlNode.TextContent, `href="https://example.com/signup"`)
	assert.Contains(t, vmlNode.TextContent, "fillcolor=\"#3498db\"")
	assert.Contains(t, vmlNode.TextContent, "Get Started")

	tableNode := result.Children[2]
	tbody := tableNode.Children[0]
	tr := tbody.Children[0]
	td := tr.Children[0]
	linkNode := td.Children[0]

	hrefAttr, found := FindAttribute(linkNode, AttrHref)
	require.True(t, found)
	assert.Equal(t, "https://example.com/signup", hrefAttr.Value)

	targetAttr, found := FindAttribute(linkNode, AttrTarget)
	require.True(t, found)
	assert.Equal(t, "_self", targetAttr.Value)

	titleAttr, found := FindAttribute(linkNode, AttrTitle)
	require.True(t, found)
	assert.Equal(t, "Sign up now", titleAttr.Value)

	styleAttr, found := FindAttribute(linkNode, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "background:#3498db;")
	assert.Contains(t, styleAttr.Value, "color:#ffffff;")
	assert.Contains(t, styleAttr.Value, "font-size:16px;")
	assert.Contains(t, styleAttr.Value, "font-weight:bold;")
	assert.Contains(t, styleAttr.Value, "border-radius:8px;")
	assert.Contains(t, styleAttr.Value, "padding:15px 30px;")

	alignAttr, found := FindAttribute(tableNode, "data-pml-align")
	require.True(t, found)
	assert.Equal(t, "center", alignAttr.Value)
}

func TestButton_Transform_WithDirectionalPadding(t *testing.T) {
	btn := NewButton()
	textNode := NewSimpleTextNode("Text")
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute("padding-top", "5px").
		WithAttribute("padding-right", "10px").
		WithAttribute("padding-bottom", "15px").
		WithAttribute("padding-left", "20px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, btn)

	result, errs := btn.Transform(node, ctx)

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

func TestButton_Transform_WithContainerBackground(t *testing.T) {
	btn := NewButton()
	textNode := NewSimpleTextNode("Text")
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrContainerBackgroundColor, "#f0f0f0").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, btn)

	result, errs := btn.Transform(node, ctx)

	require.Nil(t, errs)

	bgAttr, found := FindAttribute(result, "data-pml-container-background-color")
	require.True(t, found)
	assert.Equal(t, "#f0f0f0", bgAttr.Value)
}

func TestButton_Transform_WithRichContent(t *testing.T) {
	btn := NewButton()

	strongNode := NewElementNode("strong", nil, []*ast_domain.TemplateNode{
		NewSimpleTextNode("Bold"),
	})

	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(AttrHref, "https://example.com").
		WithChildren(
			NewSimpleTextNode("Click "),
			strongNode,
			NewSimpleTextNode(" here"),
		).
		Build()

	ctx := NewTestContext().Build(node, btn)

	result, errs := btn.Transform(node, ctx)

	require.Nil(t, errs)

	vmlNode := result.Children[0]
	assert.Contains(t, vmlNode.TextContent, "Click Bold here")

	tableNode := result.Children[2]
	tbody := tableNode.Children[0]
	tr := tbody.Children[0]
	td := tr.Children[0]
	linkNode := td.Children[0]

	require.Len(t, linkNode.Children, 3)
	assert.Equal(t, "Click ", linkNode.Children[0].TextContent)
	assert.Equal(t, "strong", linkNode.Children[1].TagName)
	assert.Equal(t, " here", linkNode.Children[2].TextContent)
}

func TestBuildParagraphStyles_DefaultValues(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		Build()

	ctx := NewTestContext().Build(node, btn)

	styles := buildParagraphStyles(ctx.StyleManager)

	assert.Equal(t, defaultButtonBgColor, styles[CSSBackground])
	assert.Equal(t, ValueZeroPx, styles[CSSMsoPaddingAlt])
	assert.Equal(t, defaultButtonColor, styles[AttrColour])
	assert.Equal(t, ValueInlineBlock, styles[CSSDisplay])
}

func TestBuildParagraphStyles_IncludesLinkStyles(t *testing.T) {
	btn := NewButton()
	node := NewTestNode().
		WithTagName("pml-button").
		WithAttribute(CSSBackgroundColor, "#123456").
		Build()

	ctx := NewTestContext().Build(node, btn)

	pStyles := buildParagraphStyles(ctx.StyleManager)
	lStyles := buildLinkStyles(ctx.StyleManager)

	for key, value := range lStyles {
		assert.Equal(t, value, pStyles[key], "paragraph style should include link style for %s", key)
	}

	assert.Equal(t, ValueZeroPx, pStyles[CSSMsoPaddingAlt])
}
