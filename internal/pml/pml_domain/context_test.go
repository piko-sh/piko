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

package pml_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_dto"
)

func TestStyleManager_Get(t *testing.T) {
	testCases := []struct {
		name           string
		styles         map[string]string
		queryKey       string
		expectedValue  string
		expectedExists bool
	}{
		{
			name:           "Key exists",
			styles:         map[string]string{"background-color": "#fff"},
			queryKey:       "background-color",
			expectedValue:  "#fff",
			expectedExists: true,
		},
		{
			name:           "Key does not exist",
			styles:         map[string]string{"background-color": "#fff"},
			queryKey:       "padding",
			expectedValue:  "",
			expectedExists: false,
		},
		{
			name:           "Empty styles map",
			styles:         map[string]string{},
			queryKey:       "anything",
			expectedValue:  "",
			expectedExists: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sm := &StyleManager{styles: tc.styles}
			value, exists := sm.Get(tc.queryKey)
			assert.Equal(t, tc.expectedValue, value)
			assert.Equal(t, tc.expectedExists, exists)
		})
	}
}

func TestStyleManager_All(t *testing.T) {
	testCases := []struct {
		styles   map[string]string
		expected map[string]string
		name     string
	}{
		{
			name:     "Returns all styles",
			styles:   map[string]string{"background-color": "#fff", "padding": "10px"},
			expected: map[string]string{"background-color": "#fff", "padding": "10px"},
		},
		{
			name:     "Empty styles",
			styles:   map[string]string{},
			expected: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sm := &StyleManager{styles: tc.styles}
			result := sm.all()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNewStyleManager(t *testing.T) {
	testCases := []struct {
		component      Component
		node           *ast_domain.TemplateNode
		config         *pml_dto.Config
		expectedStyles map[string]string
		name           string
	}{
		{
			name:           "Nil component returns empty StyleManager",
			node:           &ast_domain.TemplateNode{},
			component:      nil,
			config:         pml_dto.DefaultConfig(),
			expectedStyles: map[string]string{},
		},
		{
			name: "Default attributes are applied",
			node: &ast_domain.TemplateNode{},
			component: &mockComponent{
				tagName:           "pml-test",
				defaultAttributes: map[string]string{"padding": "10px", "background-color": "#fff"},
				precedence:        []AttributeSource{SourceDefault, SourceInline},
			},
			config:         pml_dto.DefaultConfig(),
			expectedStyles: map[string]string{"padding": "10px", "background-color": "#fff"},
		},
		{
			name: "Inline attributes override defaults",
			node: &ast_domain.TemplateNode{
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "padding", Value: "20px"},
				},
			},
			component: &mockComponent{
				tagName:           "pml-test",
				defaultAttributes: map[string]string{"padding": "10px"},
				precedence:        []AttributeSource{SourceDefault, SourceInline},
			},
			config:         pml_dto.DefaultConfig(),
			expectedStyles: map[string]string{"padding": "20px"},
		},
		{
			name: "Inline style attribute overrides individual attributes",
			node: &ast_domain.TemplateNode{
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "padding", Value: "20px"},
					{Name: "style", Value: "padding: 30px; margin: 5px"},
				},
			},
			component: &mockComponent{
				tagName:           "pml-test",
				defaultAttributes: map[string]string{},
				precedence:        []AttributeSource{SourceDefault, SourceInline},
			},
			config:         pml_dto.DefaultConfig(),
			expectedStyles: map[string]string{"padding": "30px", "margin": "5px"},
		},
		{
			name: "Config override attributes are applied",
			node: &ast_domain.TemplateNode{},
			component: &mockComponent{
				tagName:           "pml-test",
				defaultAttributes: map[string]string{"padding": "10px"},
				precedence:        []AttributeSource{SourceDefault, SourceInline},
			},
			config: &pml_dto.Config{
				OverrideAttributes: map[string]map[string]string{
					"pml-test": {"padding": "15px", "margin": "5px"},
				},
			},
			expectedStyles: map[string]string{"padding": "15px", "margin": "5px"},
		},
		{
			name: "ClearDefaultAttributes prevents defaults",
			node: &ast_domain.TemplateNode{},
			component: &mockComponent{
				tagName:           "pml-test",
				defaultAttributes: map[string]string{"padding": "10px"},
				precedence:        []AttributeSource{SourceDefault, SourceInline},
			},
			config: &pml_dto.Config{
				ClearDefaultAttributes: true,
			},
			expectedStyles: map[string]string{},
		},
		{
			name: "Data-pml attributes are excluded",
			node: &ast_domain.TemplateNode{
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "data-pml-internal", Value: "something"},
					{Name: "padding", Value: "20px"},
				},
			},
			component: &mockComponent{
				tagName:           "pml-test",
				defaultAttributes: map[string]string{},
				precedence:        []AttributeSource{SourceDefault, SourceInline},
			},
			config:         pml_dto.DefaultConfig(),
			expectedStyles: map[string]string{"padding": "20px"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sm := NewStyleManager(tc.node, tc.component, tc.config)
			require.NotNil(t, sm)
			assert.Equal(t, tc.expectedStyles, sm.all())
		})
	}
}

func TestParseInlineStyle(t *testing.T) {
	testCases := []struct {
		expected map[string]string
		name     string
		input    string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "Single property",
			input:    "padding: 10px",
			expected: map[string]string{"padding": "10px"},
		},
		{
			name:     "Multiple properties",
			input:    "padding: 10px; margin: 5px; background-color: #fff",
			expected: map[string]string{"padding": "10px", "margin": "5px", "background-color": "#fff"},
		},
		{
			name:     "Property with trailing semicolon",
			input:    "padding: 10px;",
			expected: map[string]string{"padding": "10px"},
		},
		{
			name:     "Property with spaces",
			input:    "  padding  :   10px  ;  margin  :  5px  ",
			expected: map[string]string{"padding": "10px", "margin": "5px"},
		},
		{
			name:     "Invalid declaration without colon",
			input:    "padding10px",
			expected: map[string]string{},
		},
		{
			name:     "Empty property name",
			input:    ": 10px",
			expected: map[string]string{},
		},
		{
			name:     "Empty value",
			input:    "padding:",
			expected: map[string]string{},
		},
		{
			name:     "Mixed valid and invalid",
			input:    "padding: 10px; invalid; margin: 5px",
			expected: map[string]string{"padding": "10px", "margin": "5px"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseInlineStyle(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func Test_newRootTransformationContext(t *testing.T) {
	config := pml_dto.DefaultConfig()
	registry := newMockRegistry()

	ctx := newRootTransformationContext(config, 600.0, registry)

	require.NotNil(t, ctx)
	assert.Equal(t, config, ctx.Config)
	assert.NotNil(t, ctx.StyleManager)
	assert.Nil(t, ctx.ParentNode)
	assert.Nil(t, ctx.ParentComponent)
	assert.Equal(t, 600.0, ctx.ContainerWidth)
	assert.Empty(t, ctx.ComponentPath)
	assert.Equal(t, 0, ctx.SiblingCount)
	assert.False(t, ctx.IsInsideGroup)
	assert.False(t, ctx.IsEmailContext)
	assert.Nil(t, ctx.EmailAssetRegistry)
}

func Test_newRootTransformationContextForEmail(t *testing.T) {
	config := pml_dto.DefaultConfig()
	registry := newMockRegistry()

	ctx := newRootTransformationContextForEmail(config, 600.0, registry)

	require.NotNil(t, ctx)
	assert.Equal(t, config, ctx.Config)
	assert.NotNil(t, ctx.StyleManager)
	assert.Nil(t, ctx.ParentNode)
	assert.Nil(t, ctx.ParentComponent)
	assert.Equal(t, 600.0, ctx.ContainerWidth)
	assert.Empty(t, ctx.ComponentPath)
	assert.Equal(t, 0, ctx.SiblingCount)
	assert.False(t, ctx.IsInsideGroup)
	assert.True(t, ctx.IsEmailContext)
	assert.NotNil(t, ctx.EmailAssetRegistry)
}

func TestTransformationContext_CloneForChild(t *testing.T) {
	config := pml_dto.DefaultConfig()
	registry := newMockRegistry()

	parentCtx := newRootTransformationContext(config, 600.0, registry)
	parentCtx.IsInsideGroup = true
	parentCtx.SiblingCount = 3
	parentCtx.MediaQueryCollector = newMockMediaQueryCollector()
	parentCtx.MSOConditionalCollector = newMockMSOConditionalCollector()

	childNode := &ast_domain.TemplateNode{TagName: "pml-p"}
	childComp := &mockComponent{tagName: "pml-p", precedence: []AttributeSource{SourceDefault, SourceInline}}
	parentNode := &ast_domain.TemplateNode{TagName: "pml-col"}
	parentComp := &mockComponent{tagName: "pml-col"}

	childCtx := parentCtx.CloneForChild(childNode, childComp, parentNode, parentComp)

	require.NotNil(t, childCtx)
	assert.Equal(t, parentCtx.Config, childCtx.Config)
	assert.Equal(t, parentNode, childCtx.ParentNode)
	assert.Equal(t, parentComp, childCtx.ParentComponent)
	assert.Equal(t, parentCtx.ContainerWidth, childCtx.ContainerWidth)
	assert.Equal(t, parentCtx.IsInsideGroup, childCtx.IsInsideGroup)
	assert.Equal(t, parentCtx.SiblingCount, childCtx.SiblingCount)
	assert.Equal(t, parentCtx.MediaQueryCollector, childCtx.MediaQueryCollector)
	assert.Equal(t, parentCtx.MSOConditionalCollector, childCtx.MSOConditionalCollector)
	assert.Equal(t, []string{"pml-col"}, childCtx.ComponentPath)
	assert.Empty(t, childCtx.Diagnostics())
}

func TestTransformationContext_CloneForChild_NilParentComponent(t *testing.T) {
	config := pml_dto.DefaultConfig()
	registry := newMockRegistry()

	parentCtx := newRootTransformationContext(config, 600.0, registry)

	childNode := &ast_domain.TemplateNode{TagName: "pml-p"}
	childComp := &mockComponent{tagName: "pml-p", precedence: []AttributeSource{SourceDefault, SourceInline}}

	childCtx := parentCtx.CloneForChild(childNode, childComp, nil, nil)

	require.NotNil(t, childCtx)
	assert.Nil(t, childCtx.ParentNode)
	assert.Nil(t, childCtx.ParentComponent)
	assert.Equal(t, []string{""}, childCtx.ComponentPath)
}

func TestTransformationContext_AddDiagnostic(t *testing.T) {
	ctx := newRootTransformationContext(pml_dto.DefaultConfig(), 600.0, nil)

	assert.Empty(t, ctx.Diagnostics())

	ctx.AddDiagnostic("Test error", "pml-test", SeverityError, ast_domain.Location{Line: 10, Column: 5})

	diagnostics := ctx.Diagnostics()
	require.Len(t, diagnostics, 1)
	assert.Equal(t, "Test error", diagnostics[0].Message)
	assert.Equal(t, "pml-test", diagnostics[0].TagName)
	assert.Equal(t, SeverityError, diagnostics[0].Severity)
	assert.Equal(t, 10, diagnostics[0].Location.Line)
	assert.Equal(t, 5, diagnostics[0].Location.Column)
}

func TestTransformationContext_Diagnostics(t *testing.T) {
	ctx := newRootTransformationContext(pml_dto.DefaultConfig(), 600.0, nil)

	ctx.AddDiagnostic("Error 1", "pml-a", SeverityError, ast_domain.Location{Line: 1})
	ctx.AddDiagnostic("Warning 1", "pml-b", SeverityWarning, ast_domain.Location{Line: 2})

	diagnostics := ctx.Diagnostics()
	require.Len(t, diagnostics, 2)
	assert.Equal(t, "Error 1", diagnostics[0].Message)
	assert.Equal(t, "Warning 1", diagnostics[1].Message)
}

func TestExtractSourceStyles_UnknownSource(t *testing.T) {
	node := &ast_domain.TemplateNode{}
	comp := &mockComponent{tagName: "pml-test", precedence: []AttributeSource{}}
	config := pml_dto.DefaultConfig()

	result := extractSourceStyles(AttributeSource(999), node, comp, config)
	assert.Nil(t, result)
}

func TestExtractDefaultStyles_WithOverrides(t *testing.T) {
	comp := &mockComponent{
		tagName:           "pml-test",
		defaultAttributes: nil,
		precedence:        []AttributeSource{SourceDefault},
	}
	config := &pml_dto.Config{
		OverrideAttributes: map[string]map[string]string{
			"pml-test": {"padding": "20px"},
		},
	}

	result := extractDefaultStyles(comp, config)
	assert.Equal(t, map[string]string{"padding": "20px"}, result)
}

func TestMergeStyles_EmptySource(t *testing.T) {
	target := map[string]string{"a": "1"}
	mergeStyles(target, nil)
	assert.Equal(t, map[string]string{"a": "1"}, target)

	mergeStyles(target, map[string]string{})
	assert.Equal(t, map[string]string{"a": "1"}, target)
}
