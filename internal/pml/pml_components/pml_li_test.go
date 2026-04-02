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

func TestListItem_TagName(t *testing.T) {
	li := NewListItem()
	assert.Equal(t, "pml-li", li.TagName())
}

func TestListItem_IsEndingTag(t *testing.T) {
	li := NewListItem()
	assert.False(t, li.IsEndingTag())
}

func TestListItem_AllowedParents(t *testing.T) {
	li := NewListItem()
	parents := li.AllowedParents()

	require.Len(t, parents, 1)
	assert.Contains(t, parents, "pml-ol")
}

func TestListItem_AllowedAttributes(t *testing.T) {
	li := NewListItem()
	attrs := li.AllowedAttributes()

	require.NotEmpty(t, attrs)

	assert.Contains(t, attrs, AttrBulletColor)
	assert.Contains(t, attrs, AttrBulletFontSize)
	assert.Contains(t, attrs, AttrBulletFontFamily)
	assert.Contains(t, attrs, AttrBulletFontWeight)
	assert.Contains(t, attrs, AttrBulletFontStyle)
	assert.Contains(t, attrs, AttrBulletLineHeight)

	assert.Contains(t, attrs, AttrColour)
	assert.Contains(t, attrs, CSSFontSize)
	assert.Contains(t, attrs, CSSFontFamily)
	assert.Contains(t, attrs, CSSFontWeight)
	assert.Contains(t, attrs, CSSFontStyle)
	assert.Contains(t, attrs, CSSLineHeight)

	assert.Contains(t, attrs, AttrMarginBottom)
	assert.Contains(t, attrs, AttrPaddingLeft)
}

func TestListItem_DefaultAttributes(t *testing.T) {
	li := NewListItem()
	defaults := li.DefaultAttributes()

	assert.Empty(t, defaults)
}

func TestListItem_GetStyleTargets(t *testing.T) {
	li := NewListItem()
	targets := li.GetStyleTargets()

	require.NotEmpty(t, targets)

	hasBulletColor := false
	hasBulletFontSize := false
	for _, target := range targets {
		if target.Property == AttrBulletColor && target.Target == TargetBullet {
			hasBulletColor = true
		}
		if target.Property == AttrBulletFontSize && target.Target == TargetBullet {
			hasBulletFontSize = true
		}
	}
	assert.True(t, hasBulletColor)
	assert.True(t, hasBulletFontSize)

	hasTextColor := false
	hasTextFontSize := false
	for _, target := range targets {
		if target.Property == AttrColour && target.Target == TargetText {
			hasTextColor = true
		}
		if target.Property == CSSFontSize && target.Target == TargetText {
			hasTextFontSize = true
		}
	}
	assert.True(t, hasTextColor)
	assert.True(t, hasTextFontSize)
}

func TestListItem_Transform_BasicItem(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("List item text")
	node := NewTestNode().
		WithTagName("pml-li").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, ElementLi, result.TagName)

	require.Len(t, result.Children, 1)
	assert.Equal(t, ast_domain.NodeText, result.Children[0].NodeType)
	assert.Equal(t, "List item text", result.Children[0].TextContent)
}

func TestListItem_Transform_WithBulletColor(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("Coloured bullet")
	node := NewTestNode().
		WithTagName("pml-li").
		WithAttribute(AttrBulletColor, "#ff0000").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "color:#ff0000")
}

func TestListItem_Transform_WithBulletFont(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("Custom bullet font")
	node := NewTestNode().
		WithTagName("pml-li").
		WithAttribute(AttrBulletFontSize, "20px").
		WithAttribute(AttrBulletFontFamily, "Georgia").
		WithAttribute(AttrBulletFontWeight, "bold").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "font-size:20px")
	assert.Contains(t, styleAttr.Value, "font-family:Georgia")
	assert.Contains(t, styleAttr.Value, "font-weight:bold")
}

func TestListItem_Transform_WithTextStyling(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("Styled text")
	node := NewTestNode().
		WithTagName("pml-li").
		WithAttribute(AttrColour, "#0000ff").
		WithAttribute(CSSFontSize, "16px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	require.Len(t, result.Children, 1)
	spanNode := result.Children[0]
	assert.Equal(t, ElementSpan, spanNode.TagName)

	styleAttr, found := FindAttribute(spanNode, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "color:#0000ff")
	assert.Contains(t, styleAttr.Value, "font-size:16px")

	require.Len(t, spanNode.Children, 1)
	assert.Equal(t, "Styled text", spanNode.Children[0].TextContent)
}

func TestListItem_Transform_SeparateBulletAndTextStyles(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("Different styles")
	node := NewTestNode().
		WithTagName("pml-li").
		WithAttribute(AttrBulletColor, "#ff0000").
		WithAttribute(AttrBulletFontSize, "22px").
		WithAttribute(AttrColour, "#0000ff").
		WithAttribute(CSSFontSize, "16px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	liStyleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, liStyleAttr.Value, "color:#ff0000")
	assert.Contains(t, liStyleAttr.Value, "font-size:22px")

	require.Len(t, result.Children, 1)
	spanNode := result.Children[0]
	spanStyleAttr, found := FindAttribute(spanNode, AttrStyle)
	require.True(t, found)
	assert.Contains(t, spanStyleAttr.Value, "color:#0000ff")
	assert.Contains(t, spanStyleAttr.Value, "font-size:16px")
}

func TestListItem_Transform_WithMarginBottom(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("Spaced item")
	node := NewTestNode().
		WithTagName("pml-li").
		WithAttribute(AttrMarginBottom, "15px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "margin-bottom:15px")
}

func TestListItem_Transform_WithPaddingLeft(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("Indented item")
	node := NewTestNode().
		WithTagName("pml-li").
		WithAttribute(AttrPaddingLeft, "20px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "padding-left:20px")
}

func TestListItem_Transform_WithNestedList(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("Parent item")

	nestedListNode := NewTestNode().
		WithTagName("pml-ol").
		Build()

	node := NewTestNode().
		WithTagName("pml-li").
		WithAttribute(AttrColour, "#000000").
		WithChildren(textNode, nestedListNode).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	require.Len(t, result.Children, 2)

	spanNode := result.Children[0]
	assert.Equal(t, ElementSpan, spanNode.TagName)
	require.Len(t, spanNode.Children, 1)
	assert.Equal(t, "Parent item", spanNode.Children[0].TextContent)

	nestedList := result.Children[1]
	assert.Equal(t, "pml-ol", nestedList.TagName)
}

func TestListItem_Transform_WithDivWrappedNestedList(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("Parent item")

	innerUl := NewTestNode().
		WithTagName(ElementUl).
		Build()

	divWrappedList := NewTestNode().
		WithTagName(ElementDiv).
		WithChildren(innerUl).
		Build()

	node := NewTestNode().
		WithTagName("pml-li").
		WithChildren(textNode, divWrappedList).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	require.Len(t, result.Children, 2)

	divNode := result.Children[1]
	assert.Equal(t, ElementDiv, divNode.TagName)
}

func TestListItem_Transform_MultipleTextChildren(t *testing.T) {
	li := NewListItem()
	text1 := NewSimpleTextNode("First part ")
	text2 := NewSimpleTextNode("second part")
	node := NewTestNode().
		WithTagName("pml-li").
		WithAttribute(AttrColour, "#333333").
		WithChildren(text1, text2).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	require.Len(t, result.Children, 1)
	spanNode := result.Children[0]
	assert.Equal(t, ElementSpan, spanNode.TagName)

	require.Len(t, spanNode.Children, 2)
	assert.Equal(t, "First part ", spanNode.Children[0].TextContent)
	assert.Equal(t, "second part", spanNode.Children[1].TextContent)
}

func TestListItem_Transform_PreservesPikoDirectives(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("Dynamic item")
	node := NewTestNode().
		WithTagName("pml-li").
		WithChildren(textNode).
		Build()

	node.DirFor = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "item in items"},
	}

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result.DirFor)
	assert.Equal(t, "item in items", result.DirFor.Expression.(*ast_domain.Identifier).Name)

	assert.Nil(t, node.DirFor)
}

func TestListItem_Transform_NoTextStyles_NoSpanWrapper(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("Plain text")
	node := NewTestNode().
		WithTagName("pml-li").
		WithAttribute(AttrBulletColor, "#ff0000").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	require.Len(t, result.Children, 1)
	assert.Equal(t, ast_domain.NodeText, result.Children[0].NodeType)
	assert.Equal(t, "Plain text", result.Children[0].TextContent)
}

func TestListItem_Transform_AllBulletAttributes(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("Full bullet styling")
	node := NewTestNode().
		WithTagName("pml-li").
		WithAttribute(AttrBulletColor, "#ff0000").
		WithAttribute(AttrBulletFontSize, "24px").
		WithAttribute(AttrBulletFontFamily, "Times New Roman").
		WithAttribute(AttrBulletFontWeight, "bold").
		WithAttribute(AttrBulletFontStyle, "italic").
		WithAttribute(AttrBulletLineHeight, "1.5").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "color:#ff0000")
	assert.Contains(t, styleAttr.Value, "font-size:24px")
	assert.Contains(t, styleAttr.Value, "font-family:Times New Roman")
	assert.Contains(t, styleAttr.Value, "font-weight:bold")
	assert.Contains(t, styleAttr.Value, "font-style:italic")
	assert.Contains(t, styleAttr.Value, "line-height:1.5")
}

func TestListItem_Transform_AllTextAttributes(t *testing.T) {
	li := NewListItem()
	textNode := NewSimpleTextNode("Full text styling")
	node := NewTestNode().
		WithTagName("pml-li").
		WithAttribute(AttrColour, "#0000ff").
		WithAttribute(CSSFontSize, "18px").
		WithAttribute(CSSFontFamily, "Arial").
		WithAttribute(CSSFontWeight, "normal").
		WithAttribute(CSSFontStyle, "normal").
		WithAttribute(CSSLineHeight, "1.8").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().Build(node, li)

	result, errs := li.Transform(node, ctx)

	require.Nil(t, errs)

	require.Len(t, result.Children, 1)
	spanNode := result.Children[0]
	styleAttr, found := FindAttribute(spanNode, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "color:#0000ff")
	assert.Contains(t, styleAttr.Value, "font-size:18px")
	assert.Contains(t, styleAttr.Value, "font-family:Arial")
	assert.Contains(t, styleAttr.Value, "font-weight:normal")
	assert.Contains(t, styleAttr.Value, "font-style:normal")
	assert.Contains(t, styleAttr.Value, "line-height:1.8")
}
