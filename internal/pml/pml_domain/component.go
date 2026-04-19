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
	"piko.sh/piko/internal/ast/ast_domain"
)

// AttributeType defines the type of value an attribute can accept.
// Used for validation.
type AttributeType string

const (
	// TypeColor is an attribute type for colour values such as "#FFF", "red", or
	// "rgb(0,0,0)".
	TypeColor AttributeType = "color"

	// TypeUnit is an attribute type for unit values such as "10px" or "50%".
	TypeUnit AttributeType = "unit"

	// TypeString is the attribute type for string values.
	TypeString AttributeType = "string"

	// TypeBoolean is an attribute type for boolean values ("true" or "false").
	TypeBoolean AttributeType = "boolean"

	// TypeEnum is an attribute type for values that must match one of a set of
	// allowed values.
	TypeEnum AttributeType = "enum"

	// typeInteger is the attribute type for whole number values.
	typeInteger AttributeType = "integer"

	// typeFloat is the attribute type for decimal number values (e.g. "1.5").
	typeFloat AttributeType = "float"
)

// AttributeDefinition defines the contract for a PikoML component attribute.
// It is used by the validator to check for correctness.
type AttributeDefinition struct {
	// Type specifies the expected data type for the attribute value.
	Type AttributeType

	// AllowedValues lists the valid string values when Type is "enum".
	AllowedValues []string
}

// AttributeSource defines where a style property comes from.
type AttributeSource int

const (
	// SourceDefault represents attributes from the component's
	// built-in defaults, optionally overridden by project
	// configuration.
	SourceDefault AttributeSource = iota

	// SourceInline represents attributes directly on the component tag.
	SourceInline
)

// StyleTarget defines the mapping of a CSS property to an internal part of a
// component's rendered HTML structure, enabling declarative style projection.
type StyleTarget struct {
	// Property is the CSS property name (e.g. "background-colour").
	Property string

	// Target is a logical name for the sub-element where the style should be
	// applied (e.g., "container", "text", "link").
	Target string
}

// Component defines the interface that all PikoML components (e.g., <pml-row>,
// <pml-button>) must implement. The contract allows the transformation engine
// to be decoupled from concrete component implementations.
type Component interface {
	// TagName returns the name of the component tag (e.g., "pml-row").
	TagName() string

	// IsEndingTag determines if the component's children should be treated as raw
	// HTML content. If true, the transformer will not recurse into its children.
	//
	// Returns bool which is true when children should be treated as raw content.
	IsEndingTag() bool

	// AllowedParents returns a list of tag names that are allowed to be direct
	// parents of this component. An empty slice means any parent is allowed.
	AllowedParents() []string

	// AllowedAttributes returns a map defining all valid attributes for this
	// component and their types. This is used by the validator.
	//
	// Returns map[string]AttributeDefinition which maps attribute names to their
	// type definitions.
	AllowedAttributes() map[string]AttributeDefinition

	// DefaultAttributes returns the built-in default values for attributes on this
	// component.
	//
	// These defaults are applied first with lowest priority and can be overridden
	// by config or inline attributes, so each component can define
	// built-in defaults internally.
	//
	// Returns map[string]string which contains attribute names mapped to their
	// default values.
	DefaultAttributes() map[string]string

	// GetAttributePrecedence returns the order in which style sources should be
	// merged.
	//
	// The standard order is [Default, Inline], where later
	// sources override earlier ones. A component could override
	// this for special cases.
	//
	// Returns []AttributeSource which lists the sources in merge order.
	GetAttributePrecedence() []AttributeSource

	// GetStyleTargets returns the mapping of CSS properties to the component's
	// internal targets. This is the core of the style projection logic.
	//
	// Returns []StyleTarget which contains the property-to-target mappings.
	GetStyleTargets() []StyleTarget

	// Transform is the core method of a component.
	//
	// It takes the original <pml-*> node and the current transformation context,
	// and returns a new AST subtree composed of standard, email-safe HTML, along
	// with any diagnostics generated during the process.
	//
	// Takes node (*ast_domain.TemplateNode) which is the original <pml-*> node.
	// Takes ctx (*TransformationContext) which provides the transformation state.
	//
	// Returns *ast_domain.TemplateNode which is the transformed HTML subtree.
	// Returns []*Error which contains any diagnostics from the transformation.
	Transform(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error)
}
