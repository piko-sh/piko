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
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

// NewElementNode creates a TemplateNode representing an HTML element. All optional
// TemplateNode fields are explicitly set to their zero values.
//
// Takes tagName (string) which specifies the HTML tag name for the element.
// Takes attributes ([]ast_domain.HTMLAttribute) which provides the element's HTML
// attributes. If nil, an empty slice is used.
// Takes children ([]*ast_domain.TemplateNode) which contains the child nodes. If nil, an
// empty slice is used.
//
// Returns *ast_domain.TemplateNode which is the configured element node.
func NewElementNode(tagName string, attributes []ast_domain.HTMLAttribute, children []*ast_domain.TemplateNode) *ast_domain.TemplateNode {
	if attributes == nil {
		attributes = []ast_domain.HTMLAttribute{}
	}
	if children == nil {
		children = []*ast_domain.TemplateNode{}
	}
	return ast_domain.NewElementNode(tagName, attributes, children)
}

// NewRawHTMLNode creates a TemplateNode representing raw HTML content. This node type
// renders its TextContent directly without escaping, which suited for conditional
// comments and other raw HTML fragments.
//
// Takes content (string) which is the raw HTML to include in the output.
//
// Returns *ast_domain.TemplateNode which contains the raw HTML content.
func NewRawHTMLNode(content string) *ast_domain.TemplateNode {
	node := ast_domain.TemplateNode{}
	node.NodeType = ast_domain.NodeRawHTML
	node.TextContent = content
	return &node
}

// NewFragmentNode creates a TemplateNode representing a document fragment. Fragments do
// not render wrapper elements themselves; they only render their children.
//
// Takes children ([]*ast_domain.TemplateNode) which specifies the child nodes to include
// in the fragment. If nil, an empty slice is used.
//
// Returns *ast_domain.TemplateNode which is the configured fragment node ready for use.
func NewFragmentNode(children []*ast_domain.TemplateNode) *ast_domain.TemplateNode {
	if children == nil {
		children = []*ast_domain.TemplateNode{}
	}

	node := ast_domain.TemplateNode{}
	node.NodeType = ast_domain.NodeFragment
	node.Children = children
	return &node
}

// NewSimpleTextNode creates a text node with fixed string content, such as special
// characters or entities.
//
// Takes content (string) which specifies the text for the node.
//
// Returns *ast_domain.TemplateNode which is a text node with the given content.
func NewSimpleTextNode(content string) *ast_domain.TemplateNode {
	return ast_domain.NewTextNode(content)
}

// NewHTMLAttribute creates an HTMLAttribute with the given name and value. Location
// fields are set to their zero values as they are not needed for generated content.
//
// Takes name (string) which specifies the attribute name.
// Takes value (string) which specifies the attribute value.
//
// Returns ast_domain.HTMLAttribute which is ready to use in generated HTML.
func NewHTMLAttribute(name, value string) ast_domain.HTMLAttribute {
	return ast_domain.HTMLAttribute{
		Name:           name,
		Value:          value,
		Location:       ast_domain.Location{},
		NameLocation:   ast_domain.Location{},
		AttributeRange: ast_domain.Range{},
	}
}

// NewAttributeDefinition creates an AttributeDefinition for a non-enum attribute type.
//
// Takes attributeType (pml_domain.AttributeType) which specifies the type of attribute to
// define.
//
// Returns pml_domain.AttributeDefinition which is the definition with the given type and
// nil allowed values.
func NewAttributeDefinition(attributeType pml_domain.AttributeType) pml_domain.AttributeDefinition {
	return pml_domain.AttributeDefinition{
		Type:          attributeType,
		AllowedValues: nil,
	}
}

// NewEnumAttributeDefinition creates an AttributeDefinition for an enum type with the
// specified allowed values.
//
// Takes allowedValues ([]string) which specifies the valid enum options.
//
// Returns pml_domain.AttributeDefinition which is the configured definition.
func NewEnumAttributeDefinition(allowedValues []string) pml_domain.AttributeDefinition {
	return pml_domain.AttributeDefinition{
		Type:          pml_domain.TypeEnum,
		AllowedValues: allowedValues,
	}
}
