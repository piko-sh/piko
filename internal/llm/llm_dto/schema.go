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

package llm_dto

// ResponseFormat specifies the format of a model's output.
type ResponseFormat struct {
	// JSONSchema defines the schema when Type is ResponseFormatJSONSchema.
	JSONSchema *JSONSchemaDefinition

	// Type specifies the output format type (e.g. JSON schema).
	Type ResponseFormatType
}

// ResponseFormatType specifies the format in which a response should be given.
type ResponseFormatType string

const (
	// ResponseFormatText is the default format that returns plain text.
	ResponseFormatText ResponseFormatType = "text"

	// ResponseFormatJSONObject requests a valid JSON object in the response.
	ResponseFormatJSONObject ResponseFormatType = "json_object"

	// ResponseFormatJSONSchema requests a response that follows a JSON schema.
	ResponseFormatJSONSchema ResponseFormatType = "json_schema"
)

// JSONSchemaDefinition wraps a JSON Schema with metadata for structured output.
type JSONSchemaDefinition struct {
	// Description gives context about what the schema is for.
	Description *string

	// Strict enables strict schema adherence when true.
	Strict *bool

	// Name is the schema name used for identification in some providers.
	Name string

	// Schema is the JSON Schema definition.
	Schema JSONSchema
}

// JSONSchema represents a JSON Schema definition for structured output.
// This is a simplified representation supporting the most common schema features.
type JSONSchema struct {
	// Type is the JSON type: "string", "number", "integer", "boolean", "array",
	// "object", or "null".
	Type string `json:"type,omitempty"`

	// Description provides context about this schema element.
	Description *string `json:"description,omitempty"`

	// Properties maps property names to their schema definitions for object types.
	Properties map[string]*JSONSchema `json:"properties,omitempty"`

	// Required lists the names of properties that must be present for an object type.
	Required []string `json:"required,omitempty"`

	// Items defines the schema for elements within an array.
	Items *JSONSchema `json:"items,omitempty"`

	// Enum restricts the value to one of the listed values.
	Enum []any `json:"enum,omitempty"`

	// AdditionalProperties controls whether extra properties are allowed for
	// object types. Set to false for strict schemas.
	AdditionalProperties *bool `json:"additionalProperties,omitempty"`

	// Minimum specifies the minimum value for number or integer types.
	Minimum *float64 `json:"minimum,omitempty"`

	// Maximum specifies the highest allowed value for number or integer types.
	Maximum *float64 `json:"maximum,omitempty"`

	// MinLength sets the smallest allowed length for string values.
	MinLength *int `json:"minLength,omitempty"`

	// MaxLength specifies the maximum allowed length for string values.
	MaxLength *int `json:"maxLength,omitempty"`

	// MinItems sets the smallest number of items allowed for array types.
	MinItems *int `json:"minItems,omitempty"`

	// MaxItems specifies the maximum number of items allowed in an array.
	MaxItems *int `json:"maxItems,omitempty"`

	// Pattern specifies a regular expression pattern for string validation.
	Pattern *string `json:"pattern,omitempty"`

	// Default specifies a default value for this field.
	Default any `json:"default,omitempty"`

	// AnyOf lists schemas where the value must match at least one.
	AnyOf []*JSONSchema `json:"anyOf,omitempty"`
}

// DeepCopy returns an independent copy of the schema with all nested
// pointers, maps, and slices duplicated.
//
// Returns JSONSchema which is a deep copy of the receiver.
func (s JSONSchema) DeepCopy() JSONSchema {
	cp := s

	cp.Description = copyPtr(s.Description)
	cp.AdditionalProperties = copyPtr(s.AdditionalProperties)
	cp.Minimum = copyPtr(s.Minimum)
	cp.Maximum = copyPtr(s.Maximum)
	cp.MinLength = copyPtr(s.MinLength)
	cp.MaxLength = copyPtr(s.MaxLength)
	cp.MinItems = copyPtr(s.MinItems)
	cp.MaxItems = copyPtr(s.MaxItems)
	cp.Pattern = copyPtr(s.Pattern)

	if s.Properties != nil {
		cp.Properties = make(map[string]*JSONSchema, len(s.Properties))
		for k, v := range s.Properties {
			if v != nil {
				cp.Properties[k] = new(v.DeepCopy())
			}
		}
	}

	if s.Required != nil {
		cp.Required = make([]string, len(s.Required))
		copy(cp.Required, s.Required)
	}

	if s.Items != nil {
		cp.Items = new(s.Items.DeepCopy())
	}

	if s.Enum != nil {
		cp.Enum = make([]any, len(s.Enum))
		copy(cp.Enum, s.Enum)
	}

	if s.AnyOf != nil {
		cp.AnyOf = make([]*JSONSchema, len(s.AnyOf))
		for i, v := range s.AnyOf {
			if v != nil {
				cp.AnyOf[i] = new(v.DeepCopy())
			}
		}
	}

	return cp
}

// ResponseFormatJSON returns a ResponseFormat for JSON object output.
// The model will output valid JSON but without schema constraints.
//
// Returns *ResponseFormat which is set up for JSON object output.
func ResponseFormatJSON() *ResponseFormat {
	return &ResponseFormat{Type: ResponseFormatJSONObject}
}

// ResponseFormatStructured returns a ResponseFormat with a JSON schema.
//
// Takes name (string) which identifies the schema.
// Takes schema (JSONSchema) which defines the required structure.
//
// Returns *ResponseFormat configured for structured JSON output.
func ResponseFormatStructured(name string, schema JSONSchema) *ResponseFormat {
	return &ResponseFormat{
		Type: ResponseFormatJSONSchema,
		JSONSchema: &JSONSchemaDefinition{
			Name:   name,
			Schema: schema,
			Strict: new(true),
		},
	}
}

// StringSchema returns a JSON schema for a string type.
//
// Returns JSONSchema which is set up to validate string values.
func StringSchema() JSONSchema {
	return JSONSchema{Type: "string"}
}

// IntegerSchema returns a JSONSchema configured for integer values.
//
// Returns JSONSchema which describes an integer type for JSON validation.
func IntegerSchema() JSONSchema {
	return JSONSchema{Type: "integer"}
}

// NumberSchema returns a JSON schema for a number type.
//
// Returns JSONSchema which is set up for number values.
func NumberSchema() JSONSchema {
	return JSONSchema{Type: "number"}
}

// BooleanSchema returns a JSON schema for a boolean type.
//
// Returns JSONSchema which is set up to validate boolean values.
func BooleanSchema() JSONSchema {
	return JSONSchema{Type: "boolean"}
}

// ArraySchema returns a JSONSchema for an array with the given item type.
//
// Takes items (JSONSchema) which defines the schema for each element in the
// array.
//
// Returns JSONSchema which is set up for array values.
func ArraySchema(items JSONSchema) JSONSchema {
	return JSONSchema{
		Type:  "array",
		Items: &items,
	}
}

// ObjectSchema returns a JSONSchema for an object with the given properties.
//
// Takes properties (map[string]*JSONSchema) which maps property names to their
// schemas.
// Takes required ([]string) which lists the names of required properties.
//
// Returns JSONSchema set up for object values with additional properties
// disabled.
func ObjectSchema(properties map[string]*JSONSchema, required []string) JSONSchema {
	return JSONSchema{
		Type:                 "object",
		Properties:           properties,
		Required:             required,
		AdditionalProperties: new(false),
	}
}

// EnumSchema returns a JSONSchema for a string enum.
//
// Takes values (...string) which are the allowed enum values.
//
// Returns JSONSchema which is set up with the given enum values.
func EnumSchema(values ...string) JSONSchema {
	enumVals := make([]any, len(values))
	for i, v := range values {
		enumVals[i] = v
	}
	return JSONSchema{
		Type: "string",
		Enum: enumVals,
	}
}

// NullableSchema returns a schema that allows null values.
//
// Takes schema (JSONSchema) which is the base schema to make nullable.
//
// Returns JSONSchema which allows the base type or null.
func NullableSchema(schema JSONSchema) JSONSchema {
	return JSONSchema{
		AnyOf: []*JSONSchema{
			&schema,
			{Type: "null"},
		},
	}
}

// copyPtr returns a pointer to a copy of the value, or nil if the
// input is nil.
//
// Takes p (*T) which is the pointer to copy.
//
// Returns *T which is a new pointer to a copied value, or nil.
func copyPtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	return new(*p)
}
