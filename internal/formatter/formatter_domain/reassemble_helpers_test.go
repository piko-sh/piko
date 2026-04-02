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

package formatter_domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/sfcparser"
)

func TestWriteAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes map[string]string
		want       string
	}{
		{
			"empty attributes",
			map[string]string{},
			"",
		},
		{
			"single attribute with value",
			map[string]string{"class": "container"},
			` class="container"`,
		},
		{
			"single boolean attribute",
			map[string]string{"disabled": ""},
			" disabled",
		},
		{
			"multiple attributes",
			map[string]string{
				"class": "btn",
				"id":    "submit",
				"type":  "button",
			},
			``,
		},
		{
			"attribute with special characters",
			map[string]string{"data-value": "test\"quoted\""},
			` data-value="test"quoted""`,
		},
		{
			"attribute with empty string value",
			map[string]string{"placeholder": ""},
			" placeholder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			writeAttributes(&builder, tt.attributes)
			result := builder.String()

			if tt.want != "" {
				assert.Equal(t, tt.want, result)
			} else if len(tt.attributes) > 0 {

				for key, value := range tt.attributes {
					if value == "" {
						assert.Contains(t, result, " "+key)
					} else {
						assert.Contains(t, result, key+`="`+value+`"`)
					}
				}
			}
		})
	}
}

func TestWriteBlockContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			"empty content",
			"",
			"",
		},
		{
			"simple content without newline",
			"package main",
			"package main\n",
		},
		{
			"content with trailing newline",
			"package main\n",
			"package main\n",
		},
		{
			"content with leading/trailing spaces",
			"  content  ",
			"content\n",
		},
		{
			"multi-line content",
			"line 1\nline 2\nline 3",
			"line 1\nline 2\nline 3\n",
		},
		{
			"content with multiple trailing newlines",
			"content\n\n\n",
			"content\n",
		},
		{
			"whitespace-only content",
			"   \n   \t   ",
			"\n",
		},
		{
			"single newline",
			"\n",
			"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			writeBlockContent(&builder, tt.content)
			result := builder.String()

			assert.Equal(t, tt.want, result)
		})
	}
}

func TestWriteTemplateBlock(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			"empty template",
			"",
			"",
		},
		{
			"single line template",
			"<div>Hello</div>",
			"<template>\n  <div>Hello</div>\n</template>\n",
		},
		{
			"multi-line template",
			"<div>\n  <p>Hello</p>\n</div>",
			"<template>\n  <div>\n    <p>Hello</p>\n  </div>\n</template>\n",
		},
		{
			"template with empty lines",
			"<div>\n\n<p>Hello</p>\n</div>",
			"<template>\n  <div>\n\n  <p>Hello</p>\n  </div>\n</template>\n",
		},
		{
			"template with leading/trailing whitespace",
			"  <div>Hello</div>  ",
			"<template>\n  <div>Hello</div>\n</template>\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			writeTemplateBlock(&builder, tt.template)
			result := builder.String()

			assert.Equal(t, tt.want, result)
		})
	}
}

func TestWriteScriptBlocks(t *testing.T) {
	t.Run("empty scripts", func(t *testing.T) {
		var builder strings.Builder
		writeScriptBlocks(&builder, []sfcparser.Script{})
		assert.Equal(t, "", builder.String())
	})

	t.Run("single script with attributes", func(t *testing.T) {
		var builder strings.Builder
		scripts := []sfcparser.Script{
			{
				Attributes: map[string]string{"type": "application/x-go"},
				Content:    "package main",
			},
		}
		writeScriptBlocks(&builder, scripts)
		result := builder.String()

		assert.Contains(t, result, "\n<script")
		assert.Contains(t, result, `type="application/x-go"`)
		assert.Contains(t, result, "package main")
		assert.Contains(t, result, "</script>")
	})

	t.Run("multiple scripts", func(t *testing.T) {
		var builder strings.Builder
		scripts := []sfcparser.Script{
			{Content: "// Script 1"},
			{Content: "// Script 2"},
		}
		writeScriptBlocks(&builder, scripts)
		result := builder.String()

		assert.Equal(t, 2, strings.Count(result, "<script>"))
		assert.Contains(t, result, "// Script 1")
		assert.Contains(t, result, "// Script 2")
	})
}

func TestWriteStyleBlocks(t *testing.T) {
	t.Run("empty styles", func(t *testing.T) {
		var builder strings.Builder
		writeStyleBlocks(&builder, []sfcparser.Style{})
		assert.Equal(t, "", builder.String())
	})

	t.Run("single style", func(t *testing.T) {
		var builder strings.Builder
		styles := []sfcparser.Style{
			{Content: ".container { color: red; }"},
		}
		writeStyleBlocks(&builder, styles)
		result := builder.String()

		assert.Contains(t, result, "\n<style>")
		assert.Contains(t, result, ".container { color: red; }")
		assert.Contains(t, result, "</style>")
	})
}

func TestWriteI18nBlocks(t *testing.T) {
	t.Run("empty i18n blocks", func(t *testing.T) {
		var builder strings.Builder
		writeI18nBlocks(&builder, []sfcparser.I18nBlock{})
		assert.Equal(t, "", builder.String())
	})

	t.Run("single i18n block with lang attribute", func(t *testing.T) {
		var builder strings.Builder
		i18nBlocks := []sfcparser.I18nBlock{
			{
				Attributes: map[string]string{"lang": "json"},
				Content:    `{"en": {"greeting": "Hello"}}`,
			},
		}
		writeI18nBlocks(&builder, i18nBlocks)
		result := builder.String()

		assert.Contains(t, result, "\n<i18n")
		assert.Contains(t, result, `lang="json"`)
		assert.Contains(t, result, `{"en": {"greeting": "Hello"}}`)
		assert.Contains(t, result, "</i18n>")
	})

	t.Run("multiple i18n blocks", func(t *testing.T) {
		var builder strings.Builder
		i18nBlocks := []sfcparser.I18nBlock{
			{
				Attributes: map[string]string{"locale": "en"},
				Content:    `{"hello": "Hello"}`,
			},
			{
				Attributes: map[string]string{"locale": "fr"},
				Content:    `{"hello": "Bonjour"}`,
			},
		}
		writeI18nBlocks(&builder, i18nBlocks)
		result := builder.String()

		assert.Equal(t, 2, strings.Count(result, "<i18n"))
		assert.Contains(t, result, `{"hello": "Hello"}`)
		assert.Contains(t, result, `{"hello": "Bonjour"}`)
	})
}
