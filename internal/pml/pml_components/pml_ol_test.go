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

func TestOrderedList_TagName(t *testing.T) {
	ol := NewOrderedList()
	assert.Equal(t, "pml-ol", ol.TagName())
}

func TestOrderedList_IsEndingTag(t *testing.T) {
	ol := NewOrderedList()
	assert.False(t, ol.IsEndingTag())
}

func TestOrderedList_AllowedParents(t *testing.T) {
	ol := NewOrderedList()
	parents := ol.AllowedParents()

	require.Len(t, parents, 3)
	assert.Contains(t, parents, "pml-col")
	assert.Contains(t, parents, "pml-hero")
	assert.Contains(t, parents, "pml-li")
}

func TestOrderedList_AllowedAttributes(t *testing.T) {
	ol := NewOrderedList()
	attrs := ol.AllowedAttributes()

	require.NotEmpty(t, attrs)

	assert.Contains(t, attrs, AttrListStyle)
	assert.Contains(t, attrs, AttrType)

	assert.Contains(t, attrs, CSSFontFamily)
	assert.Contains(t, attrs, CSSFontSize)
	assert.Contains(t, attrs, CSSLineHeight)
	assert.Contains(t, attrs, CSSColor)

	assert.Contains(t, attrs, CSSMarginLeft)
	assert.Contains(t, attrs, AttrPadding)
	assert.Contains(t, attrs, AttrPaddingTop)
	assert.Contains(t, attrs, AttrPaddingBottom)
	assert.Contains(t, attrs, AttrPaddingLeft)
	assert.Contains(t, attrs, AttrPaddingRight)

	assert.Contains(t, attrs, AttrAlign)

	assert.Contains(t, attrs, AttrContainerBackgroundColor)
}

func TestOrderedList_DefaultAttributes(t *testing.T) {
	ol := NewOrderedList()
	defaults := ol.DefaultAttributes()

	assert.Empty(t, defaults)
}

func TestOrderedList_GetStyleTargets(t *testing.T) {
	ol := NewOrderedList()
	targets := ol.GetStyleTargets()

	require.NotEmpty(t, targets)

	hasFont := false
	hasColor := false
	hasMarginLeft := false

	for _, target := range targets {
		if target.Property == CSSFontFamily && target.Target == TargetContainer {
			hasFont = true
		}
		if target.Property == CSSColor && target.Target == TargetContainer {
			hasColor = true
		}
		if target.Property == CSSMarginLeft && target.Target == TargetContainer {
			hasMarginLeft = true
		}
	}

	assert.True(t, hasFont)
	assert.True(t, hasColor)
	assert.True(t, hasMarginLeft)
}

func TestOrderedList_Transform_DefaultOrderedList(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, ElementDiv, result.TagName)

	classAttr, found := FindAttribute(result, AttrClass)
	require.True(t, found)
	assert.Equal(t, ClassListWrapper, classAttr.Value)

	require.Len(t, result.Children, 1)
	olElement := result.Children[0]
	assert.Equal(t, ElementOl, olElement.TagName)

	classAttr, found = FindAttribute(olElement, AttrClass)
	require.True(t, found)
	assert.Equal(t, ClassList, classAttr.Value)
}

func TestOrderedList_Transform_UnorderedList(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithAttribute("list-style", "unordered").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	wrapperDiv := result
	ulElement := wrapperDiv.Children[0]
	assert.Equal(t, ElementUl, ulElement.TagName)
}

func TestOrderedList_Transform_WithCustomType(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithAttribute(AttrType, "A").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	wrapperDiv := result
	listElement := wrapperDiv.Children[0]
	typeAttr, found := FindAttribute(listElement, AttrType)
	require.True(t, found)
	assert.Equal(t, "A", typeAttr.Value)
}

func TestOrderedList_Transform_WithCustomMarginLeft(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithAttribute(CSSMarginLeft, "40px").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	wrapperDiv := result
	listElement := wrapperDiv.Children[0]
	styleAttr, found := FindAttribute(listElement, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "margin-left:40px")
}

func TestOrderedList_Transform_WithTypography(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithAttribute(CSSFontFamily, "Arial").
		WithAttribute(CSSFontSize, "16px").
		WithAttribute(CSSColor, "#333333").
		WithAttribute(CSSLineHeight, "1.5").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	wrapperDiv := result
	listElement := wrapperDiv.Children[0]
	styleAttr, found := FindAttribute(listElement, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "font-family:Arial")
	assert.Contains(t, styleAttr.Value, "font-size:16px")
	assert.Contains(t, styleAttr.Value, "color:#333333")
	assert.Contains(t, styleAttr.Value, "line-height:1.5")
}

func TestOrderedList_Transform_WithPadding(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithAttribute(AttrPadding, "10px 20px").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	wrapperDiv := result
	listElement := wrapperDiv.Children[0]
	styleAttr, found := FindAttribute(listElement, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "padding:")
}

func TestOrderedList_Transform_WithDirectionalPadding(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithAttribute(AttrPaddingTop, "10px").
		WithAttribute(AttrPaddingBottom, "20px").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	wrapperDiv := result
	listElement := wrapperDiv.Children[0]
	styleAttr, found := FindAttribute(listElement, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "padding:")
}

func TestOrderedList_Transform_WithAlignment(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithAttribute(AttrAlign, "center").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	wrapperDiv := result
	listElement := wrapperDiv.Children[0]
	alignAttr, found := FindAttribute(listElement, AttrAlign)
	require.True(t, found)
	assert.Equal(t, "center", alignAttr.Value)
}

func TestOrderedList_Transform_WithContainerBackgroundColor(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithAttribute(AttrContainerBackgroundColor, "#f0f0f0").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	assert.Equal(t, ElementTd, result.TagName)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "background-color:#f0f0f0")

	require.Len(t, result.Children, 1)
	divWrapper := result.Children[0]
	assert.Equal(t, ElementDiv, divWrapper.TagName)
}

func TestOrderedList_Transform_PreservesPikoDirectives(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithChildren(liNode).
		Build()

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showList"},
	}

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result.DirIf)
	assert.Equal(t, "showList", result.DirIf.Expression.(*ast_domain.Identifier).Name)

	assert.Nil(t, node.DirIf)
}

func TestOrderedList_Transform_MultipleListItems(t *testing.T) {
	ol := NewOrderedList()
	li1 := NewTestNode().
		WithTagName("pml-li").
		Build()
	li2 := NewTestNode().
		WithTagName("pml-li").
		Build()
	li3 := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithChildren(li1, li2, li3).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	wrapperDiv := result
	listElement := wrapperDiv.Children[0]

	require.Len(t, listElement.Children, 3)
}

func TestOrderedList_Transform_DefaultMarginLeft(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	wrapperDiv := result
	listElement := wrapperDiv.Children[0]
	styleAttr, found := FindAttribute(listElement, AttrStyle)
	require.True(t, found)

	assert.Contains(t, styleAttr.Value, "margin-left:")
}

func TestOrderedList_Transform_ListStyles(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	wrapperDiv := result
	listElement := wrapperDiv.Children[0]
	styleAttr, found := FindAttribute(listElement, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "margin:0")
	assert.Contains(t, styleAttr.Value, "padding:0")
}

func TestOrderedList_Transform_DiscTypeForUnordered(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithAttribute("list-style", "unordered").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	wrapperDiv := result
	listElement := wrapperDiv.Children[0]
	typeAttr, found := FindAttribute(listElement, AttrType)
	require.True(t, found)
	assert.Equal(t, "disc", typeAttr.Value)
}

func TestOrderedList_Transform_OneTypeForOrdered(t *testing.T) {
	ol := NewOrderedList()
	liNode := NewTestNode().
		WithTagName("pml-li").
		Build()

	node := NewTestNode().
		WithTagName("pml-ol").
		WithAttribute("list-style", "ordered").
		WithChildren(liNode).
		Build()

	ctx := NewTestContext().Build(node, ol)

	result, errs := ol.Transform(node, ctx)

	require.Nil(t, errs)

	wrapperDiv := result
	listElement := wrapperDiv.Children[0]
	typeAttr, found := FindAttribute(listElement, AttrType)
	require.True(t, found)
	assert.Equal(t, "1", typeAttr.Value)
}
