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
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

var _ validatorPort = (*validator)(nil)

// validator implements the validatorPort interface.
// It uses a ComponentRegistry to look up the rules for each component.
type validator struct {
	// registry holds registered components used to validate tags.
	registry ComponentRegistry
}

// Validate checks each PikoML node in the AST against its component
// definition.
//
// Takes ast (*ast_domain.TemplateAST) which is the parsed template to check.
//
// Returns []*Error which contains all validation errors found, or nil if valid.
func (v *validator) Validate(ast *ast_domain.TemplateAST) []*Error {
	diagnostics := make([]*Error, 0, len(ast.RootNodes))
	for _, rootNode := range ast.RootNodes {
		diagnostics = append(diagnostics, v.validateNode(rootNode, nil)...)
	}
	return diagnostics
}

// validateNode is the recursive heart of the validator. It checks a single
// node and all of its descendants.
//
// Takes node (*ast_domain.TemplateNode) which is the current node to validate.
// Takes parentComponent (Component) which is the parent's component definition.
//
// Returns []*Error which contains all validation errors found in this subtree.
func (v *validator) validateNode(node *ast_domain.TemplateNode, parentComponent Component) []*Error {
	if !strings.HasPrefix(node.TagName, "pml-") {
		return v.validateNonPMLChildren(node, parentComponent)
	}

	component, exists := v.registry.Get(node.TagName)
	if !exists {
		return v.handleUnknownComponent(node, parentComponent)
	}

	var diagnostics []*Error

	if parentErr := v.validateParentChildRelationship(node, component, parentComponent); parentErr != nil {
		diagnostics = append(diagnostics, parentErr)
	}

	diagnostics = append(diagnostics, v.validateNodeAttributes(node, component)...)

	diagnostics = append(diagnostics, v.validateComponentChildren(node, component)...)

	return diagnostics
}

// validateNonPMLChildren recursively validates children of non-PML nodes.
//
// Takes node (*ast_domain.TemplateNode) which is the parent node whose children
// will be validated.
// Takes parentComponent (Component) which provides the component context for
// validation.
//
// Returns []*Error which contains any validation errors found in child nodes.
func (v *validator) validateNonPMLChildren(node *ast_domain.TemplateNode, parentComponent Component) []*Error {
	diagnostics := make([]*Error, 0, len(node.Children))
	for _, child := range node.Children {
		diagnostics = append(diagnostics, v.validateNode(child, parentComponent)...)
	}
	return diagnostics
}

// handleUnknownComponent creates an error for unknown components and validates
// their children.
//
// Takes node (*ast_domain.TemplateNode) which is the unknown component node.
// Takes parentComponent (Component) which is the parent for child validation.
//
// Returns []*Error which contains the unknown component error plus any child
// validation errors.
func (v *validator) handleUnknownComponent(node *ast_domain.TemplateNode, parentComponent Component) []*Error {
	diagnostics := make([]*Error, 0, 1+len(node.Children))
	diagnostics = append(diagnostics, newError(
		fmt.Sprintf("Unknown component <%s>. Make sure it is a built-in component or correctly registered.", node.TagName),
		node.TagName,
		SeverityError,
		node.Location,
	))
	for _, child := range node.Children {
		diagnostics = append(diagnostics, v.validateNode(child, parentComponent)...)
	}
	return diagnostics
}

// validateParentChildRelationship checks if the node is allowed as a child
// of its parent.
//
// Takes node (*ast_domain.TemplateNode) which is the node to validate.
// Takes component (Component) which provides the allowed parents list.
// Takes parentComponent (Component) which is the parent to check against.
//
// Returns *Error when the node is not allowed as a child of its parent.
func (*validator) validateParentChildRelationship(node *ast_domain.TemplateNode, component Component, parentComponent Component) *Error {
	allowedParents := component.AllowedParents()
	if len(allowedParents) == 0 {
		return nil
	}

	parentTagName := ""
	if parentComponent != nil {
		parentTagName = parentComponent.TagName()
	}

	if slices.Contains(allowedParents, parentTagName) {
		return nil
	}

	return newError(
		fmt.Sprintf("<%s> is not an allowed child of <%s>. Allowed parents are: [%s].",
			node.TagName, parentTagName, strings.Join(allowedParents, ", ")),
		node.TagName,
		SeverityError,
		node.Location,
	)
}

// validateNodeAttributes validates all attributes on a node.
//
// Takes node (*ast_domain.TemplateNode) which is the node to validate.
// Takes component (Component) which defines the allowed attributes.
//
// Returns []*Error which contains any validation diagnostics found.
func (v *validator) validateNodeAttributes(node *ast_domain.TemplateNode, component Component) []*Error {
	var diagnostics []*Error
	allowedAttributes := component.AllowedAttributes()

	for i := range node.Attributes {
		attr := &node.Attributes[i]
		definition, isAllowed := allowedAttributes[attr.Name]
		if !isAllowed {
			diagnostics = append(diagnostics, newError(
				fmt.Sprintf("Attribute '%s' is not a valid attribute for <%s>.", attr.Name, node.TagName),
				node.TagName,
				SeverityWarning,
				attr.Location,
			))
			continue
		}

		if err := v.validateAttributeValue(attr.Name, attr.Value, definition); err != nil {
			diagnostics = append(diagnostics, newError(
				fmt.Sprintf("Invalid value for attribute '%s': %v", attr.Name, err),
				node.TagName,
				SeverityError,
				attr.Location,
			))
		}
	}
	return diagnostics
}

// validateComponentChildren checks all child nodes of a component.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes component (Component) which gives the component context.
//
// Returns []*Error which holds any errors found in children.
func (v *validator) validateComponentChildren(node *ast_domain.TemplateNode, component Component) []*Error {
	if component.IsEndingTag() {
		return nil
	}

	var diagnostics []*Error
	for _, child := range node.Children {
		diagnostics = append(diagnostics, v.validateNode(child, component)...)
	}
	return diagnostics
}

// validateAttributeValue checks if a given value conforms to the attribute's
// type definition.
//
// Takes value (string) which is the value to check.
// Takes definition (AttributeDefinition) which specifies the type constraints.
//
// Returns error when the value does not match the expected type.
func (*validator) validateAttributeValue(_ /* attributeName */, value string, definition AttributeDefinition) error {
	switch definition.Type {
	case TypeEnum:
		return validateEnumValue(value, definition.AllowedValues)
	case typeInteger:
		return validateIntegerValue(value)
	case typeFloat:
		return validateFloatValue(value)
	case TypeUnit:
		return validateUnitValue(value)
	case TypeBoolean:
		return validateBooleanValue(value)
	case TypeColor:
		return validateColorValue(value)
	default:
		return nil
	}
}

// newValidator creates a new PikoML validator.
//
// Takes registry (ComponentRegistry) which provides access to registered
// components for validation.
//
// Returns validatorPort which validates PikoML documents against the registry.
func newValidator(registry ComponentRegistry) validatorPort {
	return &validator{registry: registry}
}

// validateEnumValue checks if a value is in a list of allowed values.
//
// Takes value (string) which is the value to check.
// Takes allowedValues ([]string) which contains the permitted values.
//
// Returns error when the value is not in the allowed list.
func validateEnumValue(value string, allowedValues []string) error {
	if slices.Contains(allowedValues, value) {
		return nil
	}
	return fmt.Errorf("value '%s' is not allowed. Must be one of: [%s]", value, strings.Join(allowedValues, ", "))
}

// validateIntegerValue checks if a value is a valid integer.
//
// Takes value (string) which is the string to check.
//
// Returns error when the value cannot be parsed as an integer.
func validateIntegerValue(value string) error {
	if _, err := strconv.Atoi(value); err != nil {
		return fmt.Errorf("value '%s' must be an integer", value)
	}
	return nil
}

// validateFloatValue checks if a value is a valid float.
//
// Takes value (string) which is the string to validate as a float.
//
// Returns error when the value cannot be parsed as a valid number.
func validateFloatValue(value string) error {
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		return fmt.Errorf("value '%s' must be a number", value)
	}
	return nil
}

// validateUnitValue checks if a value has a valid unit suffix (px or %).
//
// Takes value (string) which is the value to check.
//
// Returns error when the value does not end with px or %.
func validateUnitValue(value string) error {
	vLower := strings.ToLower(value)
	if !strings.HasSuffix(vLower, "px") && !strings.HasSuffix(vLower, "%") {
		return fmt.Errorf("value '%s' must be a valid unit (e.g., '10px', '50%%')", value)
	}
	return nil
}

// validateBooleanValue checks whether a string is a valid boolean value.
//
// Takes value (string) which is the string to check.
//
// Returns error when the value is not "true" or "false" (case-insensitive).
func validateBooleanValue(value string) error {
	vLower := strings.ToLower(value)
	if vLower != "true" && vLower != "false" {
		return fmt.Errorf("value '%s' must be 'true' or 'false'", value)
	}
	return nil
}

// validateColorValue checks whether a value is a valid colour.
//
// Takes value (string) which is the colour value to check.
//
// Returns error when the value is empty.
func validateColorValue(value string) error {
	if value == "" {
		return errors.New("colour value cannot be empty")
	}
	return nil
}
