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

func TestColumn_TagName(t *testing.T) {
	col := NewColumn()
	assert.Equal(t, "pml-col", col.TagName())
}

func TestColumn_IsEndingTag(t *testing.T) {
	col := NewColumn()
	assert.False(t, col.IsEndingTag())
}

func TestColumn_AllowedParents(t *testing.T) {
	col := NewColumn()
	parents := col.AllowedParents()

	require.Len(t, parents, 2)
	assert.Contains(t, parents, "pml-row")
	assert.Contains(t, parents, "pml-no-stack")
}

func TestColumn_AllowedAttributes(t *testing.T) {
	col := NewColumn()
	attrs := col.AllowedAttributes()

	require.NotEmpty(t, attrs)
	assert.Contains(t, attrs, AttrWidth)
	assert.Contains(t, attrs, CSSVerticalAlign)
	assert.Contains(t, attrs, CSSBackgroundColor)
	assert.Contains(t, attrs, AttrInnerBackgroundColor)
	assert.Contains(t, attrs, AttrBorderRadius)
	assert.Contains(t, attrs, AttrPadding)
}

func TestColumn_DefaultAttributes(t *testing.T) {
	col := NewColumn()
	defaults := col.DefaultAttributes()

	assert.Empty(t, defaults)
}

func TestColumn_GetStyleTargets(t *testing.T) {
	col := NewColumn()
	targets := col.GetStyleTargets()

	require.NotEmpty(t, targets)

	hasWidth := false
	hasVerticalAlign := false
	hasBackgroundColor := false

	for _, target := range targets {
		if target.Property == AttrWidth && target.Target == TargetContainer {
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

func TestColumn_Transform_BasicColumn(t *testing.T) {
	col := NewColumn()
	textNode := NewSimpleTextNode("Column content")
	node := NewTestNode().
		WithTagName("pml-col").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		WithSiblingCount(1).
		Build(node, col)

	result, errs := col.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, ElementDiv, result.TagName)

	classAttr, found := FindAttribute(result, AttrClass)
	require.True(t, found)
	assert.Contains(t, classAttr.Value, "pml-col-100")
}

func TestColumn_Transform_WithPercentageWidth(t *testing.T) {
	col := NewColumn()
	textNode := NewSimpleTextNode("Half width")
	node := NewTestNode().
		WithTagName("pml-col").
		WithAttribute(AttrWidth, "50%").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, col)

	result, errs := col.Transform(node, ctx)

	require.Nil(t, errs)

	classAttr, found := FindAttribute(result, AttrClass)
	require.True(t, found)
	assert.Contains(t, classAttr.Value, "pml-col-50")

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "width:50")
}

func TestColumn_Transform_WithPixelWidth(t *testing.T) {
	col := NewColumn()
	node := NewTestNode().
		WithTagName("pml-col").
		WithAttribute(AttrWidth, "300px").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, col)

	result, errs := col.Transform(node, ctx)

	require.Nil(t, errs)

	classAttr, found := FindAttribute(result, AttrClass)
	require.True(t, found)
	assert.Contains(t, classAttr.Value, "pml-col-50")
}

func TestColumn_Transform_WithMultipleSiblings(t *testing.T) {
	col := NewColumn()
	node := NewTestNode().
		WithTagName("pml-col").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		WithSiblingCount(3).
		Build(node, col)

	result, errs := col.Transform(node, ctx)

	require.Nil(t, errs)

	classAttr, found := FindAttribute(result, AttrClass)
	require.True(t, found)
	assert.Contains(t, classAttr.Value, "pml-col-33-33")
}

func TestColumn_Transform_WithPadding_HasGutter(t *testing.T) {
	col := NewColumn()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-col").
		WithAttribute(AttrPadding, "20px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, col)

	result, errs := col.Transform(node, ctx)

	require.Nil(t, errs)

	assert.Equal(t, ElementDiv, result.TagName)

	require.NotEmpty(t, result.Children)

	tables := FindNodesByTagName(result, ElementTable)
	require.NotEmpty(t, tables)

	gutterTable := tables[0]
	cellpadding, found := FindAttribute(gutterTable, "cellpadding")
	require.True(t, found)
	assert.Equal(t, "0", cellpadding.Value)
}

func TestColumn_Transform_WithDirectionalPadding(t *testing.T) {
	col := NewColumn()
	node := NewTestNode().
		WithTagName("pml-col").
		WithAttribute("padding-top", "10px").
		WithAttribute("padding-left", "20px").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, col)

	result, errs := col.Transform(node, ctx)

	require.Nil(t, errs)

	tables := FindNodesByTagName(result, ElementTable)
	require.NotEmpty(t, tables)
}

func TestColumn_Transform_NoPadding_NoGutter(t *testing.T) {
	col := NewColumn()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-col").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, col)

	result, errs := col.Transform(node, ctx)

	require.Nil(t, errs)

	assert.Equal(t, ElementDiv, result.TagName)
	require.NotEmpty(t, result.Children)

	firstChild := result.Children[0]
	assert.Equal(t, ElementTable, firstChild.TagName)
}

func TestColumn_Transform_WithBackgroundColor(t *testing.T) {
	col := NewColumn()
	node := NewTestNode().
		WithTagName("pml-col").
		WithAttribute(CSSBackgroundColor, "#ff0000").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, col)

	result, errs := col.Transform(node, ctx)

	require.Nil(t, errs)

	styleAttr, found := FindAttribute(result, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "background-color:#ff0000")
}

func TestColumn_Transform_WithBorderRadius(t *testing.T) {
	col := NewColumn()
	node := NewTestNode().
		WithTagName("pml-col").
		WithAttribute(AttrBorderRadius, "10px").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, col)

	result, errs := col.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result)
}

func TestColumn_Transform_MultipleChildren(t *testing.T) {
	col := NewColumn()
	child1 := NewSimpleTextNode("First")
	child2 := NewSimpleTextNode("Second")
	node := NewTestNode().
		WithTagName("pml-col").
		WithChildren(child1, child2).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, col)

	result, errs := col.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result)

	assert.Equal(t, ElementDiv, result.TagName)

	require.NotEmpty(t, result.Children)
}

func TestColumn_Transform_PreservesPikoDirectives(t *testing.T) {
	col := NewColumn()
	node := NewTestNode().
		WithTagName("pml-col").
		Build()

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showColumn"},
	}

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, col)

	result, errs := col.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result.DirIf)
	assert.Equal(t, "showColumn", result.DirIf.Expression.(*ast_domain.Identifier).Name)

	assert.Nil(t, node.DirIf)
}

func FindNodesByTagName(node *ast_domain.TemplateNode, tagName string) []*ast_domain.TemplateNode {
	var result []*ast_domain.TemplateNode

	if node.TagName == tagName {
		result = append(result, node)
	}

	for _, child := range node.Children {
		result = append(result, FindNodesByTagName(child, tagName)...)
	}

	return result
}
