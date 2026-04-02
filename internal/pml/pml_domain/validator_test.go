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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func Test_newValidator(t *testing.T) {
	registry := newMockRegistry()
	pmlValidator := newValidator(registry)

	require.NotNil(t, pmlValidator)
}

func Test_validator_Validate_ValidAST(t *testing.T) {
	registry := buildMockRegistry()
	pmlValidator := newValidator(registry)

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "pml-row",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-col",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "pml-p",
								Children: []*ast_domain.TemplateNode{
									{NodeType: ast_domain.NodeText, TextContent: "Hello"},
								},
							},
						},
					},
				},
			},
		},
	}

	diagnostics := pmlValidator.Validate(ast)
	assert.Empty(t, diagnostics)
}

func Test_validator_Validate_UnknownComponent(t *testing.T) {
	registry := buildMockRegistry()
	pmlValidator := newValidator(registry)

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "pml-unknown",
				Location: ast_domain.Location{Line: 5, Column: 10},
			},
		},
	}

	diagnostics := pmlValidator.Validate(ast)
	require.Len(t, diagnostics, 1)
	assert.Contains(t, diagnostics[0].Message, "Unknown component")
	assert.Contains(t, diagnostics[0].Message, "pml-unknown")
	assert.Equal(t, SeverityError, diagnostics[0].Severity)
}

func Test_validator_Validate_UnknownComponentWithChildren(t *testing.T) {
	registry := buildMockRegistry()
	pmlValidator := newValidator(registry)

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "pml-unknown",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-also-unknown",
					},
				},
			},
		},
	}

	diagnostics := pmlValidator.Validate(ast)
	require.Len(t, diagnostics, 2)
	assert.Contains(t, diagnostics[0].Message, "pml-unknown")
	assert.Contains(t, diagnostics[1].Message, "pml-also-unknown")
}

func Test_validator_Validate_NonPMLChildren(t *testing.T) {
	registry := buildMockRegistry()
	pmlValidator := newValidator(registry)

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-unknown",
					},
				},
			},
		},
	}

	diagnostics := pmlValidator.Validate(ast)
	require.Len(t, diagnostics, 1)
	assert.Contains(t, diagnostics[0].Message, "pml-unknown")
}

func Test_validator_Validate_ParentChildRelationship(t *testing.T) {
	registry := newMockRegistry()

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:        "pml-col",
		allowedParents: []string{"pml-row"},
	})
	_ = registry.Register(context.Background(), &mockComponent{
		tagName:        "pml-container",
		allowedParents: []string{},
	})

	pmlValidator := newValidator(registry)

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "pml-container",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-col",
						Location: ast_domain.Location{Line: 2, Column: 1},
					},
				},
			},
		},
	}

	diagnostics := pmlValidator.Validate(ast)
	require.Len(t, diagnostics, 1)
	assert.Contains(t, diagnostics[0].Message, "not an allowed child")
	assert.Contains(t, diagnostics[0].Message, "Allowed parents are")
}

func Test_validator_Validate_ParentChildRelationship_AllowedParent(t *testing.T) {
	registry := newMockRegistry()

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:        "pml-row",
		allowedParents: []string{},
	})
	_ = registry.Register(context.Background(), &mockComponent{
		tagName:        "pml-col",
		allowedParents: []string{"pml-row"},
	})

	pmlValidator := newValidator(registry)

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "pml-row",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-col",
					},
				},
			},
		},
	}

	diagnostics := pmlValidator.Validate(ast)
	assert.Empty(t, diagnostics)
}

func Test_validator_Validate_InvalidAttribute(t *testing.T) {
	registry := newMockRegistry()
	_ = registry.Register(context.Background(), &mockComponent{
		tagName: "pml-p",
		allowedAttributes: map[string]AttributeDefinition{
			"align": {Type: TypeEnum, AllowedValues: []string{"left", "center", "right"}},
		},
	})

	pmlValidator := newValidator(registry)

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "pml-p",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "invalid-attr", Value: "something", Location: ast_domain.Location{Line: 1}},
				},
			},
		},
	}

	diagnostics := pmlValidator.Validate(ast)
	require.Len(t, diagnostics, 1)
	assert.Contains(t, diagnostics[0].Message, "invalid-attr")
	assert.Contains(t, diagnostics[0].Message, "not a valid attribute")
	assert.Equal(t, SeverityWarning, diagnostics[0].Severity)
}

func Test_validator_Validate_InvalidEnumValue(t *testing.T) {
	registry := newMockRegistry()
	_ = registry.Register(context.Background(), &mockComponent{
		tagName: "pml-p",
		allowedAttributes: map[string]AttributeDefinition{
			"align": {Type: TypeEnum, AllowedValues: []string{"left", "center", "right"}},
		},
	})

	pmlValidator := newValidator(registry)

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "pml-p",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "align", Value: "invalid-value", Location: ast_domain.Location{Line: 1}},
				},
			},
		},
	}

	diagnostics := pmlValidator.Validate(ast)
	require.Len(t, diagnostics, 1)
	assert.Contains(t, diagnostics[0].Message, "Invalid value")
	assert.Equal(t, SeverityError, diagnostics[0].Severity)
}

func Test_validator_Validate_EndingTag_NoChildValidation(t *testing.T) {
	registry := newMockRegistry()
	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-p",
		isEndingTag: true,
	})

	pmlValidator := newValidator(registry)

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "pml-p",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-unknown",
					},
				},
			},
		},
	}

	diagnostics := pmlValidator.Validate(ast)
	assert.Empty(t, diagnostics, "Ending tag children should not be validated")
}

func TestValidateEnumValue(t *testing.T) {
	testCases := []struct {
		name          string
		value         string
		allowedValues []string
		expectError   bool
	}{
		{
			name:          "Valid value",
			value:         "left",
			allowedValues: []string{"left", "center", "right"},
			expectError:   false,
		},
		{
			name:          "Invalid value",
			value:         "invalid",
			allowedValues: []string{"left", "center", "right"},
			expectError:   true,
		},
		{
			name:          "Empty allowed values",
			value:         "anything",
			allowedValues: []string{},
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEnumValue(tc.value, tc.allowedValues)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIntegerValue(t *testing.T) {
	testCases := []struct {
		name        string
		value       string
		expectError bool
	}{
		{name: "Valid positive integer", value: "42", expectError: false},
		{name: "Valid negative integer", value: "-42", expectError: false},
		{name: "Valid zero", value: "0", expectError: false},
		{name: "Invalid float", value: "3.14", expectError: true},
		{name: "Invalid string", value: "abc", expectError: true},
		{name: "Empty string", value: "", expectError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIntegerValue(tc.value)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFloatValue(t *testing.T) {
	testCases := []struct {
		name        string
		value       string
		expectError bool
	}{
		{name: "Valid integer", value: "42", expectError: false},
		{name: "Valid float", value: "3.14", expectError: false},
		{name: "Valid negative float", value: "-3.14", expectError: false},
		{name: "Valid scientific notation", value: "1e10", expectError: false},
		{name: "Invalid string", value: "abc", expectError: true},
		{name: "Empty string", value: "", expectError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFloatValue(tc.value)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUnitValue(t *testing.T) {
	testCases := []struct {
		name        string
		value       string
		expectError bool
	}{
		{name: "Valid px", value: "10px", expectError: false},
		{name: "Valid percent", value: "50%", expectError: false},
		{name: "Valid uppercase PX", value: "10PX", expectError: false},
		{name: "Invalid unit em", value: "10em", expectError: true},
		{name: "Invalid no unit", value: "10", expectError: true},
		{name: "Empty string", value: "", expectError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUnitValue(tc.value)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBooleanValue(t *testing.T) {
	testCases := []struct {
		name        string
		value       string
		expectError bool
	}{
		{name: "Valid true", value: "true", expectError: false},
		{name: "Valid false", value: "false", expectError: false},
		{name: "Valid TRUE", value: "TRUE", expectError: false},
		{name: "Valid False", value: "False", expectError: false},
		{name: "Invalid yes", value: "yes", expectError: true},
		{name: "Invalid 1", value: "1", expectError: true},
		{name: "Empty string", value: "", expectError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBooleanValue(tc.value)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateColorValue(t *testing.T) {
	testCases := []struct {
		name        string
		value       string
		expectError bool
	}{
		{name: "Valid hex color", value: "#fff", expectError: false},
		{name: "Valid full hex color", value: "#ffffff", expectError: false},
		{name: "Valid named color", value: "red", expectError: false},
		{name: "Valid rgb color", value: "rgb(255,0,0)", expectError: false},
		{name: "Empty string", value: "", expectError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateColorValue(tc.value)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_validator_validateAttributeValue_StringType(t *testing.T) {
	v := &validator{}

	err := v.validateAttributeValue("attr", "anything", AttributeDefinition{Type: TypeString})
	assert.NoError(t, err)

	err = v.validateAttributeValue("attr", "", AttributeDefinition{Type: TypeString})
	assert.NoError(t, err)
}

func Test_validator_validateAttributeValue_UnknownType(t *testing.T) {
	v := &validator{registry: newMockRegistry()}

	err := v.validateAttributeValue("attr", "anything", AttributeDefinition{Type: "unknown-type"})
	assert.NoError(t, err)
}

func Test_validator_Validate_AllAttributeTypes(t *testing.T) {
	registry := newMockRegistry()
	_ = registry.Register(context.Background(), &mockComponent{
		tagName: "pml-test",
		allowedAttributes: map[string]AttributeDefinition{
			"enum-attr":   {Type: TypeEnum, AllowedValues: []string{"a", "b"}},
			"int-attr":    {Type: typeInteger},
			"float-attr":  {Type: typeFloat},
			"unit-attr":   {Type: TypeUnit},
			"bool-attr":   {Type: TypeBoolean},
			"color-attr":  {Type: TypeColor},
			"string-attr": {Type: TypeString},
		},
	})

	pmlValidator := newValidator(registry)

	testCases := []struct {
		name           string
		attributeName  string
		attributeValue string
		expectError    bool
	}{
		{name: "Valid enum", attributeName: "enum-attr", attributeValue: "a", expectError: false},
		{name: "Invalid enum", attributeName: "enum-attr", attributeValue: "invalid", expectError: true},
		{name: "Valid integer", attributeName: "int-attr", attributeValue: "42", expectError: false},
		{name: "Invalid integer", attributeName: "int-attr", attributeValue: "not-a-number", expectError: true},
		{name: "Valid float", attributeName: "float-attr", attributeValue: "3.14", expectError: false},
		{name: "Invalid float", attributeName: "float-attr", attributeValue: "not-a-float", expectError: true},
		{name: "Valid unit px", attributeName: "unit-attr", attributeValue: "10px", expectError: false},
		{name: "Valid unit percent", attributeName: "unit-attr", attributeValue: "50%", expectError: false},
		{name: "Invalid unit", attributeName: "unit-attr", attributeValue: "10em", expectError: true},
		{name: "Valid boolean true", attributeName: "bool-attr", attributeValue: "true", expectError: false},
		{name: "Valid boolean false", attributeName: "bool-attr", attributeValue: "false", expectError: false},
		{name: "Invalid boolean", attributeName: "bool-attr", attributeValue: "yes", expectError: true},
		{name: "Valid color", attributeName: "color-attr", attributeValue: "#fff", expectError: false},
		{name: "Invalid color (empty)", attributeName: "color-attr", attributeValue: "", expectError: true},
		{name: "Any string is valid", attributeName: "string-attr", attributeValue: "anything", expectError: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-test",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: tc.attributeName, Value: tc.attributeValue},
						},
					},
				},
			}

			diagnostics := pmlValidator.Validate(ast)

			if tc.expectError {
				require.NotEmpty(t, diagnostics, "Expected validation error for %s=%s", tc.attributeName, tc.attributeValue)
				assert.Equal(t, SeverityError, diagnostics[0].Severity)
			} else {
				assert.Empty(t, diagnostics, "Expected no validation error for %s=%s", tc.attributeName, tc.attributeValue)
			}
		})
	}
}
