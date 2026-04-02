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

func TestImage_TagName(t *testing.T) {
	img := NewImage()
	assert.Equal(t, "pml-img", img.TagName())
}

func TestImage_IsEndingTag(t *testing.T) {
	img := NewImage()

	assert.True(t, img.IsEndingTag())
}

func TestImage_AllowedParents(t *testing.T) {
	img := NewImage()
	parents := img.AllowedParents()

	require.Len(t, parents, 2)
	assert.Contains(t, parents, "pml-col")
	assert.Contains(t, parents, "pml-hero")
}

func TestImage_AllowedAttributes(t *testing.T) {
	img := NewImage()
	attrs := img.AllowedAttributes()

	require.NotEmpty(t, attrs)

	assert.Contains(t, attrs, AttrSrc)
	assert.Contains(t, attrs, AttrAlt)
	assert.Contains(t, attrs, AttrHref)
	assert.Contains(t, attrs, AttrWidth)
	assert.Contains(t, attrs, AttrHeight)

	assert.Contains(t, attrs, AttrBorder)
	assert.Contains(t, attrs, AttrBorderRadius)
	assert.Contains(t, attrs, AttrPadding)
	assert.Contains(t, attrs, AttrAlign)

	assert.Contains(t, attrs, AttrFluidOnMobile)
	assert.Contains(t, attrs, AttrFullWidth)

	assert.Contains(t, attrs, AttrSrcset)
	assert.Contains(t, attrs, AttrTarget)
}

func TestImage_DefaultAttributes(t *testing.T) {
	img := NewImage()
	defaults := img.DefaultAttributes()

	require.NotEmpty(t, defaults)
	assert.Contains(t, defaults, AttrAlt)
	assert.Contains(t, defaults, AttrAlign)
	assert.Contains(t, defaults, AttrBorder)
	assert.Contains(t, defaults, AttrHeight)
	assert.Contains(t, defaults, AttrPadding)
}

func TestImage_GetStyleTargets(t *testing.T) {
	img := NewImage()
	targets := img.GetStyleTargets()

	require.NotEmpty(t, targets)

	hasWidth := false
	hasHeight := false
	hasBorder := false

	for _, target := range targets {
		if target.Property == AttrWidth && target.Target == TargetContainer {
			hasWidth = true
		}
		if target.Property == AttrHeight && target.Target == TargetImage {
			hasHeight = true
		}
		if target.Property == AttrBorder && target.Target == TargetImage {
			hasBorder = true
		}
	}

	assert.True(t, hasWidth)
	assert.True(t, hasHeight)
	assert.True(t, hasBorder)
}

func TestImage_Transform_BasicImage(t *testing.T) {
	img := NewImage()
	node := NewTestNode().
		WithTagName("pml-img").
		WithAttribute(AttrSrc, "image.jpg").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, img)

	result, errs := img.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, ElementTable, result.TagName)

	tbody := result.Children[0]
	tr := tbody.Children[0]
	td := tr.Children[0]
	imgNode := td.Children[0]

	assert.Equal(t, ElementImg, imgNode.TagName)

	srcAttr, found := FindAttribute(imgNode, AttrSrc)
	require.True(t, found)
	assert.Equal(t, "image.jpg", srcAttr.Value)
}

func TestImage_Transform_WithHref(t *testing.T) {
	img := NewImage()
	node := NewTestNode().
		WithTagName("pml-img").
		WithAttribute(AttrSrc, "image.jpg").
		WithAttribute(AttrHref, "https://example.com").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, img)

	result, errs := img.Transform(node, ctx)

	require.Nil(t, errs)

	tbody := result.Children[0]
	tr := tbody.Children[0]
	td := tr.Children[0]

	anchorNode := td.Children[0]
	assert.Equal(t, ElementA, anchorNode.TagName)

	hrefAttr, found := FindAttribute(anchorNode, AttrHref)
	require.True(t, found)
	assert.Equal(t, "https://example.com", hrefAttr.Value)

	require.Len(t, anchorNode.Children, 1)
	imgNode := anchorNode.Children[0]
	assert.Equal(t, ElementImg, imgNode.TagName)
}

func TestImage_Transform_WithWidth(t *testing.T) {
	img := NewImage()
	node := NewTestNode().
		WithTagName("pml-img").
		WithAttribute(AttrSrc, "image.jpg").
		WithAttribute(AttrWidth, "300px").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, img)

	result, errs := img.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result)
}

func TestImage_Transform_WithBorder(t *testing.T) {
	img := NewImage()
	node := NewTestNode().
		WithTagName("pml-img").
		WithAttribute(AttrSrc, "image.jpg").
		WithAttribute(AttrBorder, "2px solid #000").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, img)

	result, errs := img.Transform(node, ctx)

	require.Nil(t, errs)

	tbody := result.Children[0]
	tr := tbody.Children[0]
	td := tr.Children[0]
	imgNode := td.Children[0]

	styleAttr, found := FindAttribute(imgNode, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "border:")
}

func TestImage_Transform_WithBorderRadius(t *testing.T) {
	img := NewImage()
	node := NewTestNode().
		WithTagName("pml-img").
		WithAttribute(AttrSrc, "image.jpg").
		WithAttribute(AttrBorderRadius, "10px").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, img)

	result, errs := img.Transform(node, ctx)

	require.Nil(t, errs)

	tbody := result.Children[0]
	tr := tbody.Children[0]
	td := tr.Children[0]
	imgNode := td.Children[0]

	styleAttr, found := FindAttribute(imgNode, AttrStyle)
	require.True(t, found)
	assert.Contains(t, styleAttr.Value, "border-radius:")
}

func TestImage_Transform_WithPadding(t *testing.T) {
	img := NewImage()
	node := NewTestNode().
		WithTagName("pml-img").
		WithAttribute(AttrSrc, "image.jpg").
		WithAttribute(AttrPadding, "20px").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, img)

	result, errs := img.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result)
}

func TestImage_Transform_WithAlign(t *testing.T) {
	img := NewImage()
	node := NewTestNode().
		WithTagName("pml-img").
		WithAttribute(AttrSrc, "image.jpg").
		WithAttribute(AttrAlign, "center").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, img)

	result, errs := img.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result)
	assert.Equal(t, ElementTable, result.TagName)
}

func TestImage_Transform_WithAlt(t *testing.T) {
	img := NewImage()
	node := NewTestNode().
		WithTagName("pml-img").
		WithAttribute(AttrSrc, "image.jpg").
		WithAttribute(AttrAlt, "Test image").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, img)

	result, errs := img.Transform(node, ctx)

	require.Nil(t, errs)

	tbody := result.Children[0]
	tr := tbody.Children[0]
	td := tr.Children[0]
	imgNode := td.Children[0]

	altAttr, found := FindAttribute(imgNode, AttrAlt)
	require.True(t, found)
	assert.Equal(t, "Test image", altAttr.Value)
}

func TestImage_Transform_PreservesPikoDirectives(t *testing.T) {
	img := NewImage()
	node := NewTestNode().
		WithTagName("pml-img").
		WithAttribute(AttrSrc, "image.jpg").
		Build()

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showImage"},
	}

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, img)

	result, errs := img.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result.DirIf)
	assert.Equal(t, "showImage", result.DirIf.Expression.(*ast_domain.Identifier).Name)

	assert.Nil(t, node.DirIf)
}

func TestImage_Transform_WithContainerBackgroundColor(t *testing.T) {
	img := NewImage()
	node := NewTestNode().
		WithTagName("pml-img").
		WithAttribute(AttrSrc, "image.jpg").
		WithAttribute(AttrContainerBackgroundColor, "#f0f0f0").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, img)

	result, errs := img.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result)
	assert.Equal(t, ElementTable, result.TagName)
}

func TestImage_Transform_TableStructure(t *testing.T) {
	img := NewImage()
	node := NewTestNode().
		WithTagName("pml-img").
		WithAttribute(AttrSrc, "image.jpg").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, img)

	result, errs := img.Transform(node, ctx)

	require.Nil(t, errs)

	assert.Equal(t, ElementTable, result.TagName)
	require.Len(t, result.Children, 1)

	tbody := result.Children[0]
	assert.Equal(t, ElementTbody, tbody.TagName)
	require.Len(t, tbody.Children, 1)

	tr := tbody.Children[0]
	assert.Equal(t, ElementTr, tr.TagName)
	require.Len(t, tr.Children, 1)

	td := tr.Children[0]
	assert.Equal(t, ElementTd, td.TagName)

	require.NotEmpty(t, td.Children)
}

func TestImage_Transform_WithTarget(t *testing.T) {
	img := NewImage()
	node := NewTestNode().
		WithTagName("pml-img").
		WithAttribute(AttrSrc, "image.jpg").
		WithAttribute(AttrHref, "https://example.com").
		WithAttribute(AttrTarget, "_self").
		Build()

	ctx := NewTestContext().
		WithContainerWidth(600).
		Build(node, img)

	result, errs := img.Transform(node, ctx)

	require.Nil(t, errs)

	tbody := result.Children[0]
	tr := tbody.Children[0]
	td := tr.Children[0]
	anchorNode := td.Children[0]

	targetAttr, found := FindAttribute(anchorNode, AttrTarget)
	require.True(t, found)
	assert.Equal(t, "_self", targetAttr.Value)
}
