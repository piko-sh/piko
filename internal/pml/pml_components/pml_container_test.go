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

func TestContainer_TagName(t *testing.T) {
	container := NewContainer()
	assert.Equal(t, "pml-container", container.TagName())
}

func TestContainer_IsEndingTag(t *testing.T) {
	container := NewContainer()

	assert.False(t, container.IsEndingTag())
}

func TestContainer_AllowedParents(t *testing.T) {
	container := NewContainer()
	parents := container.AllowedParents()

	require.Len(t, parents, 1)
	assert.Contains(t, parents, "pml-body")
}

func TestContainer_AllowedAttributes(t *testing.T) {
	container := NewContainer()
	attrs := container.AllowedAttributes()

	require.NotEmpty(t, attrs)

	assert.Contains(t, attrs, CSSBackgroundColor)
	assert.Contains(t, attrs, AttrPadding)
	assert.Contains(t, attrs, AttrBackgroundURL)
}

func TestContainer_DefaultAttributes(t *testing.T) {
	container := NewContainer()
	defaults := container.DefaultAttributes()

	require.Contains(t, defaults, AttrPadding)
	assert.Equal(t, ValueZero, defaults[AttrPadding])
}

func TestContainer_GetStyleTargets(t *testing.T) {
	container := NewContainer()
	targets := container.GetStyleTargets()

	require.NotEmpty(t, targets)
}

func TestContainer_Transform_InjectsStackChildren(t *testing.T) {
	container := NewContainer()
	node := NewTestNode().
		WithTagName("pml-container").
		Build()

	ctx := NewTestContext().Build(node, container)

	_, _ = container.Transform(node, ctx)

	hasStackChildren := false
	for _, attr := range node.Attributes {
		if attr.Name == "stack-children" && attr.Value == "true" {
			hasStackChildren = true
			break
		}
	}
	assert.True(t, hasStackChildren, "stack-children attribute should be injected before delegating to Row")
}

func TestContainer_Transform_PreservesExistingStackChildren(t *testing.T) {
	container := NewContainer()
	node := NewTestNode().
		WithTagName("pml-container").
		WithAttribute("stack-children", "false").
		Build()

	ctx := NewTestContext().Build(node, container)

	_, _ = container.Transform(node, ctx)

	stackChildrenCount := 0
	var stackChildrenValue string
	for _, attr := range node.Attributes {
		if attr.Name == "stack-children" {
			stackChildrenCount++
			stackChildrenValue = attr.Value
		}
	}

	assert.Equal(t, 1, stackChildrenCount)
	assert.Equal(t, "false", stackChildrenValue)
}
