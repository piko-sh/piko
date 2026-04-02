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
)

func TestUnorderedList_TagName(t *testing.T) {
	ul := NewUnorderedList()
	assert.Equal(t, "pml-ul", ul.TagName())
}

func TestUnorderedList_IsEndingTag(t *testing.T) {
	ul := NewUnorderedList()

	assert.False(t, ul.IsEndingTag())
}

func TestUnorderedList_AllowedParents(t *testing.T) {
	ul := NewUnorderedList()
	parents := ul.AllowedParents()

	require.NotEmpty(t, parents)
	assert.Contains(t, parents, "pml-col")
}

func TestUnorderedList_AllowedAttributes(t *testing.T) {
	ul := NewUnorderedList()
	attrs := ul.AllowedAttributes()

	require.NotEmpty(t, attrs)

	assert.Contains(t, attrs, AttrListStyle)
	assert.Contains(t, attrs, CSSColor)
	assert.Contains(t, attrs, CSSFontSize)
}

func TestUnorderedList_DefaultAttributes(t *testing.T) {
	ul := NewUnorderedList()
	defaults := ul.DefaultAttributes()

	assert.Empty(t, defaults)
}

func TestUnorderedList_GetStyleTargets(t *testing.T) {
	ul := NewUnorderedList()
	targets := ul.GetStyleTargets()

	require.NotEmpty(t, targets)
}

func TestUnorderedList_Transform_InjectsListStyleUnordered(t *testing.T) {
	ul := NewUnorderedList()
	node := NewTestNode().
		WithTagName("pml-ul").
		Build()

	ctx := NewTestContext().Build(node, ul)

	_, _ = ul.Transform(node, ctx)

	hasListStyle := false
	var listStyleValue string
	for _, attr := range node.Attributes {
		if attr.Name == "list-style" {
			hasListStyle = true
			listStyleValue = attr.Value
			break
		}
	}
	assert.True(t, hasListStyle, "list-style attribute should be injected")
	assert.Equal(t, "unordered", listStyleValue, "list-style should be set to 'unordered'")
}

func TestUnorderedList_Transform_PreservesExistingListStyle(t *testing.T) {
	ul := NewUnorderedList()
	node := NewTestNode().
		WithTagName("pml-ul").
		WithAttribute("list-style", "ordered").
		Build()

	ctx := NewTestContext().Build(node, ul)

	originalListStyle, hasListStyle := ctx.StyleManager.Get("list-style")
	require.True(t, hasListStyle, "list-style should be in StyleManager")
	require.Equal(t, "ordered", originalListStyle)

	_, _ = ul.Transform(node, ctx)

	listStyleCount := 0
	var listStyleValue string
	for _, attr := range node.Attributes {
		if attr.Name == "list-style" {
			listStyleCount++
			listStyleValue = attr.Value
		}
	}

	assert.Equal(t, 1, listStyleCount)
	assert.Equal(t, "ordered", listStyleValue)
}

func TestUnorderedList_Transform_WithNoExistingListStyle(t *testing.T) {
	ul := NewUnorderedList()
	node := NewTestNode().
		WithTagName("pml-ul").
		WithAttribute(CSSColor, "#333333").
		Build()

	ctx := NewTestContext().Build(node, ul)

	_, hasListStyle := ctx.StyleManager.Get("list-style")
	assert.False(t, hasListStyle)

	_, _ = ul.Transform(node, ctx)

	hasListStyleAfter := false
	var listStyleValue string
	for _, attr := range node.Attributes {
		if attr.Name == "list-style" {
			hasListStyleAfter = true
			listStyleValue = attr.Value
			break
		}
	}
	assert.True(t, hasListStyleAfter, "list-style should be injected")
	assert.Equal(t, "unordered", listStyleValue)
}
