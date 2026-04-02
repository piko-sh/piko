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

func TestThematicBreak_TagName(t *testing.T) {
	hr := NewThematicBreak()
	assert.Equal(t, "pml-hr", hr.TagName())
}

func TestThematicBreak_IsEndingTag(t *testing.T) {
	hr := NewThematicBreak()
	assert.True(t, hr.IsEndingTag())
}

func TestThematicBreak_AllowedParents(t *testing.T) {
	hr := NewThematicBreak()
	parents := hr.AllowedParents()

	require.Len(t, parents, 2)
	assert.Contains(t, parents, "pml-col")
	assert.Contains(t, parents, "pml-hero")
}

func TestThematicBreak_AllowedAttributes(t *testing.T) {
	hr := NewThematicBreak()
	attrs := hr.AllowedAttributes()

	require.NotEmpty(t, attrs)
	assert.Contains(t, attrs, AttrAlign)
	assert.Contains(t, attrs, CSSBorderColor)
	assert.Contains(t, attrs, CSSBorderStyle)
	assert.Contains(t, attrs, CSSBorderWidth)
	assert.Contains(t, attrs, AttrContainerBackgroundColor)
	assert.Contains(t, attrs, AttrPadding)
	assert.Contains(t, attrs, "padding-top")
	assert.Contains(t, attrs, "padding-bottom")
	assert.Contains(t, attrs, "padding-left")
	assert.Contains(t, attrs, "padding-right")
	assert.Contains(t, attrs, AttrWidth)

	alignDef := attrs[AttrAlign]
	assert.Equal(t, pml_domain.TypeEnum, alignDef.Type)
	require.Len(t, alignDef.AllowedValues, 3)
	assert.Contains(t, alignDef.AllowedValues, "left")
	assert.Contains(t, alignDef.AllowedValues, ValueCentre)
	assert.Contains(t, alignDef.AllowedValues, "right")

	borderStyleDef := attrs[CSSBorderStyle]
	assert.Equal(t, pml_domain.TypeEnum, borderStyleDef.Type)
	require.Len(t, borderStyleDef.AllowedValues, 3)
	assert.Contains(t, borderStyleDef.AllowedValues, "dashed")
	assert.Contains(t, borderStyleDef.AllowedValues, "dotted")
	assert.Contains(t, borderStyleDef.AllowedValues, "solid")
}

func TestThematicBreak_GetStyleTargets(t *testing.T) {
	hr := NewThematicBreak()
	targets := hr.GetStyleTargets()

	require.Len(t, targets, 6)

	targetProps := make([]string, len(targets))
	for i, target := range targets {
		targetProps[i] = target.Property
		assert.Equal(t, TargetContainer, target.Target)
	}

	assert.Contains(t, targetProps, CSSBorderColor)
	assert.Contains(t, targetProps, CSSBorderStyle)
	assert.Contains(t, targetProps, CSSBorderWidth)
	assert.Contains(t, targetProps, CSSWidth)
	assert.Contains(t, targetProps, CSSPadding)
	assert.Contains(t, targetProps, CSSAlign)
}

func TestThematicBreak_DefaultAttributes(t *testing.T) {
	hr := NewThematicBreak()
	defaults := hr.DefaultAttributes()

	require.NotEmpty(t, defaults)
	assert.Equal(t, defaultBorderColor, defaults[CSSBorderColor])
	assert.Equal(t, defaultBorderStyle, defaults[CSSBorderStyle])
	assert.Equal(t, defaultBorderWidth, defaults[CSSBorderWidth])
	assert.Equal(t, defaultHRPadding, defaults[CSSPadding])
	assert.Equal(t, Value100, defaults[CSSWidth])
	assert.Equal(t, defaultHRAlign, defaults[CSSAlign])
}

func TestThematicBreak_Transform_DefaultStyles(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		Build()

	ctx := NewTestContext().Build(node, hr)

	result, errs := hr.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	require.Len(t, result.Children, 2)

	pNode := result.Children[0]
	assert.Equal(t, ast_domain.NodeElement, pNode.NodeType)
	assert.Equal(t, "p", pNode.TagName)

	styleAttr, found := FindAttribute(pNode, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "font-size:1px;")
	assert.Contains(t, styleAttr.Value, "margin:0px auto;")
	assert.Contains(t, styleAttr.Value, "width:100%;")

	assert.Contains(t, styleAttr.Value, "border-top:solid 4px #000000;")

	outlookTable := result.Children[1]
	assert.Equal(t, ast_domain.NodeRawHTML, outlookTable.NodeType)
	assert.Contains(t, outlookTable.TextContent, "<!--[if mso | IE]>")
	assert.Contains(t, outlookTable.TextContent, "<![endif]-->")
}

func TestThematicBreak_Transform_CustomBorderStyles(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(CSSBorderColor, "#ff0000").
		WithAttribute(CSSBorderStyle, "dashed").
		WithAttribute(CSSBorderWidth, "2px").
		Build()

	ctx := NewTestContext().Build(node, hr)

	result, errs := hr.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	pNode := result.Children[0]
	styleAttr, found := FindAttribute(pNode, AttrStyle)
	require.True(t, found)

	assert.Contains(t, styleAttr.Value, "border-top:dashed 2px #ff0000;")

	outlookTable := result.Children[1]
	assert.Contains(t, outlookTable.TextContent, "border-top:dashed 2px #ff0000")
}

func TestThematicBreak_Transform_CustomWidth(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(AttrWidth, "50%").
		Build()

	ctx := NewTestContext().Build(node, hr)

	result, errs := hr.Transform(node, ctx)

	require.Nil(t, errs)

	pNode := result.Children[0]
	styleAttr, found := FindAttribute(pNode, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "width:50%;")
}

func TestThematicBreak_Transform_PreservesPikoDirectives(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		Build()

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showDivider"},
	}

	ctx := NewTestContext().Build(node, hr)

	result, errs := hr.Transform(node, ctx)

	require.Nil(t, errs)

	pNode := result.Children[0]
	assert.NotNil(t, pNode.DirIf)
	assert.Equal(t, "showDivider", pNode.DirIf.Expression.(*ast_domain.Identifier).Name)

	assert.Nil(t, node.DirIf)
}

func TestBuildHRParagraphStyles_DefaultValues(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().WithTagName("pml-hr").Build()
	ctx := NewTestContext().Build(node, hr)

	styles := buildHRParagraphStyles(ctx.StyleManager)

	assert.Equal(t, "1px", styles[CSSFontSize])
	assert.Equal(t, ValueMarginAuto, styles["margin"])
	assert.Equal(t, Value100, styles[CSSWidth])
	assert.Equal(t, "solid 4px #000000", styles["border-top"])
}

func TestBuildHRParagraphStyles_CustomBorder(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(CSSBorderStyle, "dotted").
		WithAttribute(CSSBorderWidth, "3px").
		WithAttribute(CSSBorderColor, "#00ff00").
		Build()
	ctx := NewTestContext().Build(node, hr)

	styles := buildHRParagraphStyles(ctx.StyleManager)

	assert.Equal(t, "dotted 3px #00ff00", styles["border-top"])
}

func TestBuildHRParagraphStyles_CustomWidth(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(AttrWidth, "75%").
		Build()
	ctx := NewTestContext().Build(node, hr)

	styles := buildHRParagraphStyles(ctx.StyleManager)

	assert.Equal(t, "75%", styles[CSSWidth])
}

func TestApplyHRAlignment_Centre(t *testing.T) {
	pStyles := map[string]string{"margin": ValueMarginAuto}
	applyHRAlignment(pStyles, ValueCentre)

	assert.Equal(t, ValueMarginAuto, pStyles["margin"])
}

func TestApplyHRAlignment_Left(t *testing.T) {
	pStyles := map[string]string{"margin": ValueMarginAuto}
	applyHRAlignment(pStyles, "left")

	assert.Equal(t, ValueZero, pStyles["margin"])
}

func TestApplyHRAlignment_Right(t *testing.T) {
	pStyles := map[string]string{"margin": ValueMarginAuto}
	applyHRAlignment(pStyles, "right")

	assert.Equal(t, "0 0 0 auto", pStyles["margin"])
}

func TestGetOutlookHRWidth_PixelWidth(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(AttrWidth, "300px").
		Build()
	ctx := NewTestContext().Build(node, hr)

	width := getOutlookHRWidth("300px", ctx.StyleManager, ctx)

	assert.Equal(t, 300, width)
}

func TestGetOutlookHRWidth_PercentageWidth(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(AttrWidth, "50%").
		WithAttribute(AttrPadding, "0px").
		Build()
	ctx := NewTestContext().WithContainerWidth(600).Build(node, hr)

	width := getOutlookHRWidth("50%", ctx.StyleManager, ctx)

	assert.Equal(t, 300, width)
}

func TestGetOutlookHRWidth_PercentageWithPadding(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(AttrWidth, "100%").
		WithAttribute(AttrPadding, "10px 25px").
		Build()
	ctx := NewTestContext().WithContainerWidth(600).Build(node, hr)

	width := getOutlookHRWidth("100%", ctx.StyleManager, ctx)

	assert.Equal(t, 550, width)
}

func TestGetOutlookHRWidth_DirectionalPaddingOverrides(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(AttrWidth, "100%").
		WithAttribute(AttrPadding, "10px 25px").
		WithAttribute("padding-left", "50px").
		WithAttribute("padding-right", "30px").
		Build()
	ctx := NewTestContext().WithContainerWidth(600).Build(node, hr)

	width := getOutlookHRWidth("100%", ctx.StyleManager, ctx)

	assert.Equal(t, 520, width)
}

func TestGetOutlookHRWidth_OneSidedPadding(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(AttrWidth, "100%").
		WithAttribute(AttrPadding, "20px").
		Build()
	ctx := NewTestContext().WithContainerWidth(600).Build(node, hr)

	width := getOutlookHRWidth("100%", ctx.StyleManager, ctx)

	assert.Equal(t, 560, width)
}

func TestGetOutlookHRWidth_DefaultContainerWidth(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(AttrWidth, "100%").
		WithAttribute(AttrPadding, "0px").
		Build()

	ctx := NewTestContext().Build(node, hr)

	width := getOutlookHRWidth("100%", ctx.StyleManager, ctx)

	assert.Equal(t, 600, width)
}

func TestGetOutlookHRWidth_FourValuePadding(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(AttrWidth, "100%").
		WithAttribute(AttrPadding, "10px 20px 30px 40px").
		Build()
	ctx := NewTestContext().WithContainerWidth(600).Build(node, hr)

	width := getOutlookHRWidth("100%", ctx.StyleManager, ctx)

	assert.Equal(t, 540, width)
}

func TestBuildOutlookHRTableHTML_BasicStructure(t *testing.T) {
	html := buildOutlookHRTableHTML("center", "solid", "4px", "#000000", 580)

	assert.Contains(t, html, `align="center"`)
	assert.Contains(t, html, `border="0"`)
	assert.Contains(t, html, `cellpadding="0"`)
	assert.Contains(t, html, `cellspacing="0"`)
	assert.Contains(t, html, `role="presentation"`)
	assert.Contains(t, html, `width="580px"`)

	assert.Contains(t, html, `border-top:solid 4px #000000`)

	assert.Contains(t, html, `width:580px`)

	assert.Contains(t, html, "<table")
	assert.Contains(t, html, "<tr>")
	assert.Contains(t, html, "<td")
	assert.Contains(t, html, "&nbsp;")
	assert.Contains(t, html, "</td>")
	assert.Contains(t, html, "</tr>")
	assert.Contains(t, html, "</table>")
}

func TestBuildOutlookHRTableHTML_CustomStyles(t *testing.T) {
	html := buildOutlookHRTableHTML("right", "dashed", "2px", "#ff0000", 300)

	assert.Contains(t, html, `align="right"`)
	assert.Contains(t, html, `border-top:dashed 2px #ff0000`)
	assert.Contains(t, html, `width:300px`)
	assert.Contains(t, html, `width="300px"`)
}

func TestBuildOutlookHRTableHTML_LeftAlignment(t *testing.T) {
	html := buildOutlookHRTableHTML("left", "solid", "3px", "#0000ff", 400)

	assert.Contains(t, html, `align="left"`)
	assert.Contains(t, html, `width:400px`)
}

func TestRenderOutlookHRTable_BasicStructure(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		Build()
	ctx := NewTestContext().WithContainerWidth(600).Build(node, hr)

	result := renderOutlookHRTable(ctx.StyleManager, ctx)

	assert.Equal(t, ast_domain.NodeRawHTML, result.NodeType)

	htmlContent := result.TextContent

	assert.True(t, strings.HasPrefix(htmlContent, "<!--[if mso | IE]>"))
	assert.True(t, strings.HasSuffix(htmlContent, "<![endif]-->"))

	assert.Contains(t, htmlContent, "<table")
	assert.Contains(t, htmlContent, "</table>")
}

func TestRenderOutlookHRTable_CustomAttributes(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(CSSBorderStyle, "dotted").
		WithAttribute(CSSBorderWidth, "5px").
		WithAttribute(CSSBorderColor, "#00ff00").
		WithAttribute(AttrWidth, "50%").
		WithAttribute(AttrAlign, "left").
		WithAttribute(AttrPadding, "0px").
		Build()
	ctx := NewTestContext().WithContainerWidth(600).Build(node, hr)

	result := renderOutlookHRTable(ctx.StyleManager, ctx)

	htmlContent := result.TextContent

	assert.Contains(t, htmlContent, "border-top:dotted 5px #00ff00")
	assert.Contains(t, htmlContent, `align="left"`)

	assert.Contains(t, htmlContent, "width:300px")
}

func TestThematicBreak_Transform_LeftAlignedDivider(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(AttrAlign, "left").
		WithAttribute(AttrWidth, "200px").
		Build()

	ctx := NewTestContext().Build(node, hr)

	result, errs := hr.Transform(node, ctx)

	require.Nil(t, errs)

	pNode := result.Children[0]
	styleAttr, found := FindAttribute(pNode, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "margin:0;")
	assert.Contains(t, styleAttr.Value, "width:200px;")

	outlookTable := result.Children[1]
	assert.Contains(t, outlookTable.TextContent, `align="left"`)
}

func TestThematicBreak_Transform_RightAlignedDivider(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(AttrAlign, "right").
		WithAttribute(AttrWidth, "300px").
		Build()

	ctx := NewTestContext().Build(node, hr)

	result, errs := hr.Transform(node, ctx)

	require.Nil(t, errs)

	pNode := result.Children[0]
	styleAttr, found := FindAttribute(pNode, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "margin:0 0 0 auto;")

	outlookTable := result.Children[1]
	assert.Contains(t, outlookTable.TextContent, `align="right"`)
}

func TestThematicBreak_Transform_ComplexConfiguration(t *testing.T) {
	hr := NewThematicBreak()
	node := NewTestNode().
		WithTagName("pml-hr").
		WithAttribute(CSSBorderColor, "#3498db").
		WithAttribute(CSSBorderStyle, "dashed").
		WithAttribute(CSSBorderWidth, "3px").
		WithAttribute(AttrWidth, "80%").
		WithAttribute(AttrAlign, ValueCentre).
		WithAttribute(AttrPadding, "15px 30px").
		Build()

	ctx := NewTestContext().WithContainerWidth(600).Build(node, hr)

	result, errs := hr.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	pNode := result.Children[0]
	styleAttr, found := FindAttribute(pNode, AttrStyle)
	require.True(t, found)

	assert.Contains(t, styleAttr.Value, "border-top:dashed 3px #3498db;")
	assert.Contains(t, styleAttr.Value, "width:80%;")
	assert.Contains(t, styleAttr.Value, "margin:0px auto;")

	outlookTable := result.Children[1]
	assert.Contains(t, outlookTable.TextContent, "border-top:dashed 3px #3498db")
	assert.Contains(t, outlookTable.TextContent, `align="center"`)

	assert.Contains(t, outlookTable.TextContent, "width:432px")
}
