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

package ast_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildAttributeMap(t *testing.T) {
	t.Parallel()

	t.Run("returns empty map for node with no attributes", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
		}
		result := buildAttributeMap(node)
		assert.Empty(t, result)
	})

	t.Run("maps static attributes by lowercase name", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			Attributes: []HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "id", Value: "main"},
			},
		}
		result := buildAttributeMap(node)
		assert.Equal(t, "container", result["class"])
		assert.Equal(t, "main", result["id"])
		assert.Len(t, result, 2)
	})

	t.Run("lowercases attribute names", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			Attributes: []HTMLAttribute{
				{Name: "DataValue", Value: "test"},
			},
		}
		result := buildAttributeMap(node)
		assert.Equal(t, "test", result["datavalue"])
		_, exists := result["DataValue"]
		assert.False(t, exists, "original cased key should not exist")
	})

	t.Run("includes DirectWriter attributes", func(t *testing.T) {
		t.Parallel()

		dw := GetDirectWriter()
		dw.Name = "title"
		dw.AppendString("My Title")
		defer PutDirectWriter(dw)

		node := &TemplateNode{
			NodeType:         NodeElement,
			TagName:          "div",
			AttributeWriters: []*DirectWriter{dw},
		}
		result := buildAttributeMap(node)
		assert.Equal(t, "My Title", result["title"])
	})

	t.Run("static attributes take precedence over DirectWriter attributes", func(t *testing.T) {
		t.Parallel()

		dw := GetDirectWriter()
		dw.Name = "class"
		dw.AppendString("dynamic-class")
		defer PutDirectWriter(dw)

		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			Attributes: []HTMLAttribute{
				{Name: "class", Value: "static-class"},
			},
			AttributeWriters: []*DirectWriter{dw},
		}
		result := buildAttributeMap(node)
		assert.Equal(t, "static-class", result["class"])
	})

	t.Run("skips nil DirectWriter entries", func(t *testing.T) {
		t.Parallel()

		dw := GetDirectWriter()
		dw.Name = "href"
		dw.AppendString("/page")
		defer PutDirectWriter(dw)

		node := &TemplateNode{
			NodeType:         NodeElement,
			TagName:          "a",
			AttributeWriters: []*DirectWriter{nil, dw, nil},
		}
		result := buildAttributeMap(node)
		assert.Equal(t, "/page", result["href"])
		assert.Len(t, result, 1)
	})

	t.Run("handles boolean attributes with empty value", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "input",
			Attributes: []HTMLAttribute{
				{Name: "disabled", Value: ""},
				{Name: "type", Value: "text"},
			},
		}
		result := buildAttributeMap(node)
		assert.Equal(t, "", result["disabled"])
		assert.Equal(t, "text", result["type"])
		assert.Len(t, result, 2)
	})

	t.Run("DirectWriter name is lowercased for map key", func(t *testing.T) {
		t.Parallel()

		dw := GetDirectWriter()
		dw.Name = "DataCustom"
		dw.AppendString("value")
		defer PutDirectWriter(dw)

		node := &TemplateNode{
			NodeType:         NodeElement,
			TagName:          "div",
			AttributeWriters: []*DirectWriter{dw},
		}
		result := buildAttributeMap(node)
		assert.Equal(t, "value", result["datacustom"])
	})

	t.Run("combines static and DirectWriter attributes", func(t *testing.T) {
		t.Parallel()

		dw := GetDirectWriter()
		dw.Name = "title"
		dw.AppendString("Dynamic Title")
		defer PutDirectWriter(dw)

		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			Attributes: []HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "id", Value: "main"},
			},
			AttributeWriters: []*DirectWriter{dw},
		}
		result := buildAttributeMap(node)
		assert.Equal(t, "container", result["class"])
		assert.Equal(t, "main", result["id"])
		assert.Equal(t, "Dynamic Title", result["title"])
		assert.Len(t, result, 3)
	})

	t.Run("case-insensitive precedence between static and DirectWriter", func(t *testing.T) {
		t.Parallel()

		dw := GetDirectWriter()
		dw.Name = "CLASS"
		dw.AppendString("dynamic-class")
		defer PutDirectWriter(dw)

		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			Attributes: []HTMLAttribute{
				{Name: "Class", Value: "static-class"},
			},
			AttributeWriters: []*DirectWriter{dw},
		}
		result := buildAttributeMap(node)
		assert.Equal(t, "static-class", result["class"])
		assert.Len(t, result, 1)
	})
}
