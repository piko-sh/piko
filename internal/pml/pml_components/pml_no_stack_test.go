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
)

func TestNoStack_TagName(t *testing.T) {
	ns := NewNoStack()
	assert.Equal(t, "pml-no-stack", ns.TagName())
}

func TestNoStack_IsEndingTag(t *testing.T) {
	ns := NewNoStack()
	assert.False(t, ns.IsEndingTag())
}

func TestNoStack_AllowedParents(t *testing.T) {
	ns := NewNoStack()
	parents := ns.AllowedParents()

	require.Len(t, parents, 1)
	assert.Contains(t, parents, "pml-row")
}

func TestNoStack_AllowedAttributes(t *testing.T) {
	ns := NewNoStack()
	attrs := ns.AllowedAttributes()

	require.NotEmpty(t, attrs)
	assert.Contains(t, attrs, CSSWidth)
	assert.Contains(t, attrs, CSSVerticalAlign)
	assert.Contains(t, attrs, CSSBackgroundColor)
	assert.Contains(t, attrs, CSSDirection)
}

func TestNoStack_DefaultAttributes(t *testing.T) {
	ns := NewNoStack()
	defaults := ns.DefaultAttributes()

	assert.Empty(t, defaults)
}

func TestNoStack_GetStyleTargets(t *testing.T) {
	ns := NewNoStack()
	targets := ns.GetStyleTargets()

	require.NotEmpty(t, targets)

	hasWidth := false
	hasVerticalAlign := false
	hasBackgroundColor := false

	for _, target := range targets {
		if target.Property == CSSWidth && target.Target == TargetContainer {
			hasWidth = true
		}
		if target.Property == CSSVerticalAlign && target.Target == TargetContainer {
			hasVerticalAlign = true
		}
		if target.Property == CSSBackgroundColor && target.Target == TargetContainer {
			hasBackgroundColor = true
		}
	}

	assert.True(t, hasWidth)
	assert.True(t, hasVerticalAlign)
	assert.True(t, hasBackgroundColor)
}

func TestNoStack_Transform_BasicStructure(t *testing.T) {
	ns := NewNoStack()
	childNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-no-stack").
		WithChildren(childNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, ns)

	result, errs := ns.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, ElementDiv, result.TagName)

	require.GreaterOrEqual(t, len(result.Children), 3)

	outlookStart := result.Children[0]
	assert.Equal(t, ast_domain.NodeRawHTML, outlookStart.NodeType)
	assert.Contains(t, outlookStart.TextContent, "<!--[if mso | IE]>")
	assert.Contains(t, outlookStart.TextContent, "<table")
	assert.Contains(t, outlookStart.TextContent, "<tr>")

	outlookEnd := result.Children[len(result.Children)-1]
	assert.Equal(t, ast_domain.NodeRawHTML, outlookEnd.NodeType)
	assert.Contains(t, outlookEnd.TextContent, "</tr></table>")
	assert.Contains(t, outlookEnd.TextContent, "<![endif]-->")
}

func TestNoStack_Transform_DivStyles(t *testing.T) {
	ns := NewNoStack()
	node := NewTestNode().
		WithTagName("pml-no-stack").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, ns)

	result, errs := ns.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "font-size:0px")
	assert.Contains(t, styleAttr.Value, "line-height:0")
	assert.Contains(t, styleAttr.Value, "text-align:left")
	assert.Contains(t, styleAttr.Value, "display:inline-block")
	assert.Contains(t, styleAttr.Value, "width:100%")
	assert.Contains(t, styleAttr.Value, "direction:ltr")
}

func TestNoStack_Transform_WithPercentageWidth(t *testing.T) {
	ns := NewNoStack()
	node := NewTestNode().
		WithTagName("pml-no-stack").
		WithAttribute(CSSWidth, "50%").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, ns)

	result, errs := ns.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "width:100%")
}

func TestNoStack_Transform_WithPixelWidth(t *testing.T) {
	ns := NewNoStack()
	node := NewTestNode().
		WithTagName("pml-no-stack").
		WithAttribute(CSSWidth, "300px").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, ns)

	result, errs := ns.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "width:100%")
}

func TestNoStack_Transform_WithBackgroundColor(t *testing.T) {
	ns := NewNoStack()
	node := NewTestNode().
		WithTagName("pml-no-stack").
		WithAttribute(CSSBackgroundColor, "#ff0000").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, ns)

	result, errs := ns.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "background-color:#ff0000")

	outlookStart := result.Children[0]
	assert.Contains(t, outlookStart.TextContent, "bgcolor=")
	assert.Contains(t, outlookStart.TextContent, "#ff0000")
}

func TestNoStack_Transform_WithDirection(t *testing.T) {
	ns := NewNoStack()
	node := NewTestNode().
		WithTagName("pml-no-stack").
		WithAttribute(CSSDirection, "rtl").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, ns)

	result, errs := ns.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "direction:rtl")

	outlookStart := result.Children[0]
	assert.Contains(t, outlookStart.TextContent, "dir=")
	assert.Contains(t, outlookStart.TextContent, "rtl")
}

func TestNoStack_Transform_WithVerticalAlign(t *testing.T) {
	ns := NewNoStack()
	node := NewTestNode().
		WithTagName("pml-no-stack").
		WithAttribute(CSSVerticalAlign, "middle").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, ns)

	result, errs := ns.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "vertical-align:middle")
}

func TestNoStack_Transform_OutlookTableAttributes(t *testing.T) {
	ns := NewNoStack()
	node := NewTestNode().
		WithTagName("pml-no-stack").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, ns)

	result, errs := ns.Transform(node, ctx)

	require.Nil(t, errs)

	outlookStart := result.Children[0]
	htmlContent := outlookStart.TextContent

	assert.Contains(t, htmlContent, "role=\"presentation\"")
	assert.Contains(t, htmlContent, "border=\"0\"")
	assert.Contains(t, htmlContent, "cellpadding=\"0\"")
	assert.Contains(t, htmlContent, "cellspacing=\"0\"")
}

func TestNoStack_Transform_MultipleChildren(t *testing.T) {
	ns := NewNoStack()
	child1 := NewTestNode().
		WithTagName("pml-col").
		Build()
	child2 := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-no-stack").
		WithChildren(child1, child2).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, ns)

	result, errs := ns.Transform(node, ctx)

	require.Nil(t, errs)

	colCount := 0
	for _, c := range result.Children {
		if c.NodeType == ast_domain.NodeElement && c.TagName == "pml-col" {
			colCount++
		}
	}
	assert.Equal(t, 2, colCount)
}

func TestNoStack_Transform_PreservesPikoDirectives(t *testing.T) {
	ns := NewNoStack()
	childNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-no-stack").
		WithChildren(childNode).
		Build()

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showNoStack"},
	}

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, ns)

	result, errs := ns.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result.DirIf)
	assert.Equal(t, "showNoStack", result.DirIf.Expression.(*ast_domain.Identifier).Name)

	assert.Nil(t, node.DirIf)
}

func TestNoStack_Transform_DefaultWidthWhenNotSpecified(t *testing.T) {
	ns := NewNoStack()
	node := NewTestNode().
		WithTagName("pml-no-stack").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, ns)

	result, errs := ns.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "width:100%")
}

func TestNoStack_Transform_ZeroContainerWidth(t *testing.T) {
	ns := NewNoStack()
	node := NewTestNode().
		WithTagName("pml-no-stack").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(0).
		Build(node, ns)

	result, errs := ns.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result)
	assert.Equal(t, ElementDiv, result.TagName)
}
