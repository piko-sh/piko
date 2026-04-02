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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseFormatJSON(t *testing.T) {
	t.Parallel()

	rf := ResponseFormatJSON()
	assert.Equal(t, ResponseFormatJSONObject, rf.Type)
	assert.Nil(t, rf.JSONSchema)
}

func TestResponseFormatStructured(t *testing.T) {
	t.Parallel()

	schema := ObjectSchema(map[string]*JSONSchema{
		"name": new(StringSchema()),
		"age":  new(IntegerSchema()),
	}, []string{"name", "age"})

	rf := ResponseFormatStructured("person", schema)

	assert.Equal(t, ResponseFormatJSONSchema, rf.Type)
	require.NotNil(t, rf.JSONSchema)
	assert.Equal(t, "person", rf.JSONSchema.Name)
	assert.Equal(t, "object", rf.JSONSchema.Schema.Type)
	require.NotNil(t, rf.JSONSchema.Strict)
	assert.True(t, *rf.JSONSchema.Strict)
}

func TestStringSchema(t *testing.T) {
	t.Parallel()

	s := StringSchema()
	assert.Equal(t, "string", s.Type)
}

func TestIntegerSchema(t *testing.T) {
	t.Parallel()

	s := IntegerSchema()
	assert.Equal(t, "integer", s.Type)
}

func TestNumberSchema(t *testing.T) {
	t.Parallel()

	s := NumberSchema()
	assert.Equal(t, "number", s.Type)
}

func TestBooleanSchema(t *testing.T) {
	t.Parallel()

	s := BooleanSchema()
	assert.Equal(t, "boolean", s.Type)
}

func TestArraySchema(t *testing.T) {
	t.Parallel()

	s := ArraySchema(StringSchema())
	assert.Equal(t, "array", s.Type)
	require.NotNil(t, s.Items)
	assert.Equal(t, "string", s.Items.Type)
}

func TestObjectSchema(t *testing.T) {
	t.Parallel()

	props := map[string]*JSONSchema{
		"name": new(StringSchema()),
		"age":  new(IntegerSchema()),
	}
	s := ObjectSchema(props, []string{"name"})

	assert.Equal(t, "object", s.Type)
	assert.Len(t, s.Properties, 2)
	assert.Equal(t, []string{"name"}, s.Required)
	require.NotNil(t, s.AdditionalProperties)
	assert.False(t, *s.AdditionalProperties)
}

func TestEnumSchema(t *testing.T) {
	t.Parallel()

	s := EnumSchema("red", "green", "blue")

	assert.Equal(t, "string", s.Type)
	assert.Len(t, s.Enum, 3)
	assert.Equal(t, "red", s.Enum[0])
	assert.Equal(t, "green", s.Enum[1])
	assert.Equal(t, "blue", s.Enum[2])
}

func TestNullableSchema(t *testing.T) {
	t.Parallel()

	s := NullableSchema(StringSchema())

	require.Len(t, s.AnyOf, 2)
	assert.Equal(t, "string", s.AnyOf[0].Type)
	assert.Equal(t, "null", s.AnyOf[1].Type)
}

func TestJSONSchema_DeepCopy_NestedProperties(t *testing.T) {
	t.Parallel()

	original := ObjectSchema(map[string]*JSONSchema{
		"name": new(StringSchema()),
		"address": new(ObjectSchema(map[string]*JSONSchema{
			"street": new(StringSchema()),
			"city":   new(StringSchema()),
		}, []string{"street"})),
	}, []string{"name"})

	copied := original.DeepCopy()

	copied.Properties["name"].Type = "integer"
	assert.Equal(t, "string", original.Properties["name"].Type)

	copied.Properties["address"].Properties["street"].Type = "number"
	assert.Equal(t, "string", original.Properties["address"].Properties["street"].Type)

	copied.Required = []string{"name", "address"}
	assert.Equal(t, []string{"name"}, original.Required)
}

func TestJSONSchema_DeepCopy_NilFields(t *testing.T) {
	t.Parallel()

	original := JSONSchema{Type: "string"}
	copied := original.DeepCopy()

	assert.Equal(t, "string", copied.Type)
	assert.Nil(t, copied.Properties)
	assert.Nil(t, copied.Items)
	assert.Nil(t, copied.AnyOf)
}

func TestJSONSchema_DeepCopy_ArrayWithItems(t *testing.T) {
	t.Parallel()

	original := ArraySchema(StringSchema())
	copied := original.DeepCopy()

	copied.Items.Type = "number"
	assert.Equal(t, "string", original.Items.Type)
}

func TestJSONSchema_DeepCopy_AnyOf(t *testing.T) {
	t.Parallel()

	original := NullableSchema(StringSchema())
	copied := original.DeepCopy()

	copied.AnyOf[0].Type = "integer"
	assert.Equal(t, "string", original.AnyOf[0].Type)
	assert.Len(t, copied.AnyOf, 2)
}

func TestJSONSchema_DeepCopy_PointerFields(t *testing.T) {
	t.Parallel()

	original := JSONSchema{
		Type:        "string",
		Description: new("a description"),
	}
	copied := original.DeepCopy()

	*copied.Description = "changed"
	assert.Equal(t, "a description", *original.Description)
}
