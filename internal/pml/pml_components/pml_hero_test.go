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

func TestHero_TagName(t *testing.T) {
	hero := NewHero()
	assert.Equal(t, "pml-hero", hero.TagName())
}

func TestHero_IsEndingTag(t *testing.T) {
	hero := NewHero()
	assert.False(t, hero.IsEndingTag())
}

func TestHero_AllowedParents(t *testing.T) {
	hero := NewHero()
	parents := hero.AllowedParents()

	require.Len(t, parents, 2)
	assert.Contains(t, parents, "pml-body")
	assert.Contains(t, parents, "pml-container")
}

func TestHero_AllowedAttributes(t *testing.T) {
	hero := NewHero()
	attrs := hero.AllowedAttributes()

	require.NotEmpty(t, attrs)

	assert.Contains(t, attrs, AttrMode)
	assert.Contains(t, attrs, AttrHeight)
	assert.Contains(t, attrs, AttrBackgroundURL)
	assert.Contains(t, attrs, AttrBackgroundWidth)
	assert.Contains(t, attrs, AttrBackgroundHeight)

	assert.Contains(t, attrs, CSSBackgroundPosition)
	assert.Contains(t, attrs, CSSBackgroundSize)
	assert.Contains(t, attrs, CSSBackgroundColor)

	assert.Contains(t, attrs, AttrBorderRadius)
	assert.Contains(t, attrs, AttrContainerBackgroundColor)
	assert.Contains(t, attrs, AttrInnerBackgroundColor)
	assert.Contains(t, attrs, AttrInnerPadding)
	assert.Contains(t, attrs, AttrPadding)
	assert.Contains(t, attrs, CSSVerticalAlign)
	assert.Contains(t, attrs, AttrAlign)
}

func TestHero_DefaultAttributes(t *testing.T) {
	hero := NewHero()
	defaults := hero.DefaultAttributes()

	assert.Empty(t, defaults)
}

func TestHero_GetStyleTargets(t *testing.T) {
	hero := NewHero()
	targets := hero.GetStyleTargets()

	require.NotEmpty(t, targets)

	hasBackgroundURL := false
	hasHeight := false
	hasInnerPadding := false

	for _, target := range targets {
		if target.Property == AttrBackgroundURL && target.Target == TargetBackground {
			hasBackgroundURL = true
		}
		if target.Property == "height" && target.Target == TargetContainer {
			hasHeight = true
		}
		if target.Property == AttrInnerPadding && target.Target == "inner" {
			hasInnerPadding = true
		}
	}

	assert.True(t, hasBackgroundURL)
	assert.True(t, hasHeight)
	assert.True(t, hasInnerPadding)
}

func TestHero_Transform_MissingRequiredAttributes(t *testing.T) {
	hero := NewHero()
	node := NewTestNode().
		WithTagName("pml-hero").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.NotNil(t, errs)
	require.NotEmpty(t, errs)

	require.NotNil(t, result)
	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}

func TestHero_Transform_FixedHeightMode(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Hero content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	require.NotEmpty(t, result.Children)

	assert.True(t, ContainsVML(result))
}

func TestHero_Transform_FluidHeightMode(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Hero content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFluidHeight).
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)

	assert.True(t, ContainsVML(result))
}

func TestHero_Transform_WithPadding(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithAttribute(AttrPadding, "40px 20px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}

func TestHero_Transform_WithDirectionalPadding(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithAttribute(AttrPaddingTop, "40px").
		WithAttribute(AttrPaddingRight, "20px").
		WithAttribute(AttrPaddingBottom, "40px").
		WithAttribute(AttrPaddingLeft, "20px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)
}

func TestHero_Transform_WithInnerPadding(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithAttribute(AttrInnerPadding, "20px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}

func TestHero_Transform_WithBackgroundColor(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithAttribute(CSSBackgroundColor, "#ff0000").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)
}

func TestHero_Transform_WithInnerBackgroundColor(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithAttribute(AttrInnerBackgroundColor, "#00ff00").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)
}

func TestHero_Transform_WithContainerBackgroundColor(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithAttribute(AttrContainerBackgroundColor, "#0000ff").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.True(t, ContainsVML(result))
}

func TestHero_Transform_WithBorderRadius(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithAttribute(AttrBorderRadius, "10px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)
}

func TestHero_Transform_WithVerticalAlign(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithAttribute(CSSVerticalAlign, ValueMiddle).
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)
}

func TestHero_Transform_WithAlign(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithAttribute(AttrAlign, ValueLeft).
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)
}

func TestHero_Transform_WithBackgroundPosition(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithAttribute(CSSBackgroundPosition, "top left").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)
}

func TestHero_Transform_PreservesPikoDirectives(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithChildren(textNode).
		Build()

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showHero"},
	}

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result.DirIf)
	assert.Equal(t, "showHero", result.DirIf.Expression.(*ast_domain.Identifier).Name)

	assert.Nil(t, node.DirIf)
}

func TestHero_Transform_ZeroContainerWidth(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(0).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}

func TestHero_Transform_ComplexBackground(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "500px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1600px").
		WithAttribute(AttrBackgroundHeight, "900px").
		WithAttribute(CSSBackgroundColor, "#f0f0f0").
		WithAttribute(CSSBackgroundPosition, "center center").
		WithAttribute(CSSBackgroundSize, "cover").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	assert.True(t, ContainsVML(result))
}

func TestHero_Transform_WithChildren(t *testing.T) {
	hero := NewHero()

	child1 := NewTestNode().
		WithTagName("pml-p").
		Build()
	child2 := NewTestNode().
		WithTagName("pml-button").
		Build()

	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithChildren(child1, child2).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}

func TestHero_Transform_HeightAdjustmentWithPadding(t *testing.T) {
	hero := NewHero()
	textNode := NewSimpleTextNode("Content")
	node := NewTestNode().
		WithTagName("pml-hero").
		WithAttribute(AttrMode, ValueFixedHeight).
		WithAttribute(AttrHeight, "400px").
		WithAttribute(AttrBackgroundURL, "hero.jpg").
		WithAttribute(AttrBackgroundWidth, "1200px").
		WithAttribute(AttrBackgroundHeight, "800px").
		WithAttribute(AttrPaddingTop, "50px").
		WithAttribute(AttrPaddingBottom, "50px").
		WithChildren(textNode).
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, hero)

	result, errs := hero.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}
