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

package compiler_domain

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"piko.sh/piko/internal/ast/ast_domain"
)

// ComponentMetadata holds type information taken from TypeScript source code.
// This replaces the previous decorator-based metadata system.
type ComponentMetadata struct {
	// StateProperties maps property names to their type metadata.
	StateProperties map[string]*PropertyMetadata

	// Methods maps method names to their metadata.
	Methods map[string]*MethodMetadata

	// BooleanProps lists state properties with boolean type, used for VDOM
	// generation.
	BooleanProps []string
}

// PropertyMetadata contains type and value information for a state property.
type PropertyMetadata struct {
	// Name is the property name as it appears in the source code.
	Name string

	// JSType is the JavaScript type: "string", "number",
	// "boolean", "array", "object", or "any".
	JSType string

	// ElementType specifies the type of elements for array properties.
	ElementType string

	// KeyType is the key type for Map<K,V> properties; empty for non-map types.
	KeyType string

	// ValueType is the value type for maps; for Map<K, V> this holds V.
	ValueType string

	// InitialValue holds the JavaScript expression as a string for use as the
	// default value.
	InitialValue string

	// Location specifies where this property is defined in the source file.
	Location ast_domain.Location

	// IsNullable indicates whether this is a union type
	// containing null or undefined.
	IsNullable bool
}

// MethodMetadata contains information about a user-defined method.
type MethodMetadata struct {
	// Name is the method name.
	Name string

	// Location is the position in the source file where the method is declared.
	Location ast_domain.Location
}

// GetPropType returns the property type in the format expected by the
// propTypes getter, matching the Web Components API format.
//
// Returns string which is the formatted type name such as
// "String", "Number", "Array<String>", or "Map<String,Number>".
func (p *PropertyMetadata) GetPropType() string {
	titleCaser := cases.Title(language.English)
	switch p.JSType {
	case "string":
		return "String"
	case "number":
		return "Number"
	case "boolean":
		return "Boolean"
	case "array":
		if p.ElementType != "" {
			elemType := titleCaser.String(p.ElementType)
			return fmt.Sprintf("Array<%s>", elemType)
		}
		return "Array"
	case "object":
		if p.KeyType != "" && p.ValueType != "" {
			return fmt.Sprintf("Map<%s,%s>",
				titleCaser.String(p.KeyType),
				titleCaser.String(p.ValueType))
		}
		return "Object"
	default:
		return "Any"
	}
}

// GetDefaultValue returns the initial value for the defaultProps getter.
//
// Returns string which is the property's initial value if set, otherwise a
// sensible default based on the JavaScript type.
func (p *PropertyMetadata) GetDefaultValue() string {
	if p.InitialValue != "" {
		return p.InitialValue
	}

	switch p.JSType {
	case "string":
		return `""`
	case "number":
		return "0"
	case "boolean":
		return "false"
	case "array":
		return "[]"
	case "object":
		return "{}"
	default:
		return "null"
	}
}

// IsBoolean reports whether this property is a boolean type.
//
// Returns bool which is true if the property's JavaScript type is boolean.
func (p *PropertyMetadata) IsBoolean() bool {
	return p.JSType == "boolean"
}

// NewComponentMetadata creates an empty ComponentMetadata with initialised
// maps and slices ready for use.
//
// Returns *ComponentMetadata which contains empty but initialised collections
// for state properties, methods, and boolean props.
func NewComponentMetadata() *ComponentMetadata {
	return &ComponentMetadata{
		StateProperties: make(map[string]*PropertyMetadata),
		Methods:         make(map[string]*MethodMetadata),
		BooleanProps:    []string{},
	}
}
