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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	require.NotNil(t, registry)

	all := registry.GetAll()
	assert.Empty(t, all)
}

func TestRegister_Success(t *testing.T) {
	registry := NewRegistry()
	component := NewParagraph()

	err := registry.Register(context.Background(), component)
	require.NoError(t, err)

	retrieved, ok := registry.Get("pml-p")
	require.True(t, ok)
	assert.Equal(t, component, retrieved)
}

func TestRegister_NilComponent(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot register a nil component")
}

func TestRegister_EmptyTagName(t *testing.T) {
	registry := NewRegistry()

	mockComponent := &mockEmptyTagComponent{}

	err := registry.Register(context.Background(), mockComponent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TagName() cannot be empty")
}

func TestRegister_Overwrite(t *testing.T) {
	registry := NewRegistry()

	firstComponent := NewParagraph()
	err := registry.Register(context.Background(), firstComponent)
	require.NoError(t, err)

	secondComponent := NewParagraph()
	err = registry.Register(context.Background(), secondComponent)
	require.NoError(t, err)

	retrieved, ok := registry.Get("pml-p")
	require.True(t, ok)
	assert.Equal(t, secondComponent, retrieved)
}

func TestGet_ExistingComponent(t *testing.T) {
	registry := NewRegistry()
	component := NewButton()
	err := registry.Register(context.Background(), component)
	require.NoError(t, err)

	retrieved, ok := registry.Get("pml-button")
	require.True(t, ok)
	assert.Equal(t, component, retrieved)
}

func TestGet_NonExistentComponent(t *testing.T) {
	registry := NewRegistry()

	retrieved, ok := registry.Get("pml-nonexistent")
	assert.False(t, ok)
	assert.Nil(t, retrieved)
}

func TestMustGet_ExistingComponent(t *testing.T) {
	registry := NewRegistry()
	component := NewImage()
	err := registry.Register(context.Background(), component)
	require.NoError(t, err)

	retrieved := registry.MustGet("pml-img")
	assert.Equal(t, component, retrieved)
}

func TestMustGet_Panic(t *testing.T) {
	registry := NewRegistry()

	assert.Panics(t, func() {
		registry.MustGet("pml-nonexistent")
	})
}

func TestGetAll(t *testing.T) {
	registry := NewRegistry()

	components := []pml_domain.Component{
		NewParagraph(),
		NewButton(),
		NewImage(),
	}

	for _, comp := range components {
		err := registry.Register(context.Background(), comp)
		require.NoError(t, err)
	}

	all := registry.GetAll()
	assert.Len(t, all, 3)

	tagNames := make(map[string]bool)
	for _, comp := range all {
		tagNames[comp.TagName()] = true
	}

	assert.True(t, tagNames["pml-p"])
	assert.True(t, tagNames["pml-button"])
	assert.True(t, tagNames["pml-img"])
}

func TestGetAll_EmptyRegistry(t *testing.T) {
	registry := NewRegistry()

	all := registry.GetAll()
	assert.Empty(t, all)
}

func TestRegisterBuiltIns_Success(t *testing.T) {
	registry, err := RegisterBuiltIns(context.Background())
	require.NoError(t, err)
	require.NotNil(t, registry)

	all := registry.GetAll()
	assert.NotEmpty(t, all)
}

func TestRegisterBuiltIns_Count(t *testing.T) {
	registry, err := RegisterBuiltIns(context.Background())
	require.NoError(t, err)

	all := registry.GetAll()

	assert.Len(t, all, 13)
}

func TestRegisterBuiltIns_AllComponentsRetrievable(t *testing.T) {
	registry, err := RegisterBuiltIns(context.Background())
	require.NoError(t, err)

	expectedComponents := []string{
		"pml-row",
		"pml-container",
		"pml-no-stack",
		"pml-col",
		"pml-p",
		"pml-img",
		"pml-button",
		"pml-hr",
		"pml-br",
		"pml-hero",
		"pml-ol",
		"pml-ul",
		"pml-li",
	}

	for _, tagName := range expectedComponents {
		comp, ok := registry.Get(tagName)
		assert.True(t, ok, "Component %s should be registered", tagName)
		assert.NotNil(t, comp, "Component %s should not be nil", tagName)
		assert.Equal(t, tagName, comp.TagName(), "Component tag name should match")
	}
}

type mockEmptyTagComponent struct {
	BaseComponent
}

func (m *mockEmptyTagComponent) TagName() string {
	return ""
}

func (m *mockEmptyTagComponent) AllowedParents() []string {
	return []string{}
}

func (m *mockEmptyTagComponent) AllowedAttributes() map[string]pml_domain.AttributeDefinition {
	return map[string]pml_domain.AttributeDefinition{}
}

func (m *mockEmptyTagComponent) GetStyleTargets() []pml_domain.StyleTarget {
	return []pml_domain.StyleTarget{}
}

func (m *mockEmptyTagComponent) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	return node, nil
}
