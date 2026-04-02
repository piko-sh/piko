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

package llm_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripFrontmatter_NoFrontmatter(t *testing.T) {
	input := "# Hello\n\nThis is content."
	assert.Equal(t, input, stripFrontmatter(input))
}

func TestStripFrontmatter_ValidFrontmatter(t *testing.T) {
	input := "---\ntitle: Test\ndate: 2026-01-01\n---\n# Hello\n\nContent here."
	expected := "# Hello\n\nContent here."
	assert.Equal(t, expected, stripFrontmatter(input))
}

func TestStripFrontmatter_UnclosedFrontmatter(t *testing.T) {
	input := "---\ntitle: Test\nno closing delimiter"
	assert.Equal(t, input, stripFrontmatter(input))
}

func TestStripFrontmatter_EmptyContent(t *testing.T) {
	assert.Equal(t, "", stripFrontmatter(""))
}

func TestStripFrontmatter_OnlyDelimiters(t *testing.T) {
	input := "---\n---\nContent after."
	assert.Equal(t, "Content after.", stripFrontmatter(input))
}

func TestStripFrontmatter_MultipleNewlinesAfter(t *testing.T) {
	input := "---\ntitle: Test\n---\n\n\nContent."
	assert.Equal(t, "Content.", stripFrontmatter(input))
}

func TestStripFrontmatter_TransformFunc(t *testing.T) {
	transformer := StripFrontmatter()
	document := Document{
		ID:      "test.md",
		Content: "---\ntitle: Hello\n---\n# Hello World",
		Metadata: map[string]any{
			"source": "test.md",
		},
	}

	result := transformer(document)
	assert.Equal(t, "# Hello World", result.Content)
	assert.Equal(t, "test.md", result.ID)
	assert.Equal(t, "test.md", result.Metadata["source"])
}

func TestStripFrontmatter_DashesInContent(t *testing.T) {
	input := "---\ntitle: Test\n---\nSome content\n---\nMore content with dashes."
	expected := "Some content\n---\nMore content with dashes."
	assert.Equal(t, expected, stripFrontmatter(input))
}

func TestExtractFrontmatter_AllKeys(t *testing.T) {
	transformer := ExtractFrontmatter()
	document := Document{
		ID:      "doc.md",
		Content: "---\ntitle: Getting Started\nsection: guide\norder: 5\n---\n# Hello",
		Metadata: map[string]any{
			"source": "doc.md",
		},
	}

	result := transformer(document)

	assert.Equal(t, "# Hello", result.Content)
	assert.Equal(t, "doc.md", result.Metadata["source"])
	assert.Equal(t, "Getting Started", result.Metadata["title"])
	assert.Equal(t, "guide", result.Metadata["section"])
	assert.Equal(t, 5, result.Metadata["order"])
}

func TestExtractFrontmatter_FilteredKeys(t *testing.T) {
	transformer := ExtractFrontmatter(WithFrontmatterKeys("title", "section"))
	document := Document{
		ID:      "doc.md",
		Content: "---\ntitle: Hello\nsection: guide\norder: 5\ndraft: true\n---\nBody",
	}

	result := transformer(document)

	assert.Equal(t, "Body", result.Content)
	assert.Equal(t, "Hello", result.Metadata["title"])
	assert.Equal(t, "guide", result.Metadata["section"])
	assert.Nil(t, result.Metadata["order"], "unselected key should not be extracted")
	assert.Nil(t, result.Metadata["draft"], "unselected key should not be extracted")
}

func TestExtractFrontmatter_Prefix(t *testing.T) {
	transformer := ExtractFrontmatter(WithFrontmatterPrefix("doc_"))
	document := Document{
		ID:      "doc.md",
		Content: "---\ntitle: Hello\n---\nBody",
	}

	result := transformer(document)

	assert.Equal(t, "Hello", result.Metadata["doc_title"])
	assert.Nil(t, result.Metadata["title"], "unprefixed key should not exist")
}

func TestExtractFrontmatter_PrefixAndKeys(t *testing.T) {
	transformer := ExtractFrontmatter(
		WithFrontmatterKeys("title"),
		WithFrontmatterPrefix("doc_"),
	)
	document := Document{
		ID:      "doc.md",
		Content: "---\ntitle: Hello\nsection: guide\n---\nBody",
	}

	result := transformer(document)

	assert.Equal(t, "Hello", result.Metadata["doc_title"])
	assert.Nil(t, result.Metadata["doc_section"], "unselected key should not be extracted")
}

func TestExtractFrontmatter_NoFrontmatter(t *testing.T) {
	transformer := ExtractFrontmatter()
	document := Document{
		ID:      "doc.md",
		Content: "# Just content, no frontmatter",
	}

	result := transformer(document)

	assert.Equal(t, "# Just content, no frontmatter", result.Content)
	assert.Nil(t, result.Metadata)
}

func TestExtractFrontmatter_NilMetadata(t *testing.T) {
	transformer := ExtractFrontmatter()
	document := Document{
		ID:      "doc.md",
		Content: "---\ntitle: Hello\n---\nBody",
	}

	result := transformer(document)

	assert.Equal(t, "Body", result.Content)
	assert.Equal(t, "Hello", result.Metadata["title"])
}

func TestExtractFrontmatter_InvalidYAML(t *testing.T) {
	transformer := ExtractFrontmatter()
	document := Document{
		ID:      "doc.md",
		Content: "---\n: invalid: yaml: [broken\n---\nBody",
	}

	result := transformer(document)

	assert.Equal(t, document.Content, result.Content)
}

func TestExtractFrontmatter_PreservesExistingMetadata(t *testing.T) {
	transformer := ExtractFrontmatter()
	document := Document{
		ID:      "doc.md",
		Content: "---\ntitle: Hello\n---\nBody",
		Metadata: map[string]any{
			"source": "docs/doc.md",
		},
	}

	result := transformer(document)

	assert.Equal(t, "docs/doc.md", result.Metadata["source"])
	assert.Equal(t, "Hello", result.Metadata["title"])
}

func TestPrependChunkContext_TitleAndHeading(t *testing.T) {
	transformer := PrependChunkContext()
	document := Document{
		Content:  "Some code example here",
		Metadata: map[string]any{"doc_title": "Your First Page", "heading": "Add a template"},
	}

	result := transformer(document)
	assert.Equal(t, "Your First Page > Add a template\n\nSome code example here", result.Content)
}

func TestPrependChunkContext_TitleOnly(t *testing.T) {
	transformer := PrependChunkContext()
	document := Document{
		Content:  "Content",
		Metadata: map[string]any{"doc_title": "Getting Started"},
	}

	result := transformer(document)
	assert.Equal(t, "Getting Started\n\nContent", result.Content)
}

func TestPrependChunkContext_HeadingOnly(t *testing.T) {
	transformer := PrependChunkContext()
	document := Document{
		Content:  "Content",
		Metadata: map[string]any{"heading": "Installation"},
	}

	result := transformer(document)
	assert.Equal(t, "Installation\n\nContent", result.Content)
}

func TestPrependChunkContext_NoMetadata(t *testing.T) {
	transformer := PrependChunkContext()
	document := Document{Content: "Content"}

	result := transformer(document)
	assert.Equal(t, "Content", result.Content)
}

func TestPrependChunkContext_EmptyMetadata(t *testing.T) {
	transformer := PrependChunkContext()
	document := Document{
		Content:  "Content",
		Metadata: map[string]any{"doc_title": "", "heading": ""},
	}

	result := transformer(document)
	assert.Equal(t, "Content", result.Content)
}

func TestPrependChunkContext_SkipsContentWithHeading(t *testing.T) {
	transformer := PrependChunkContext()
	document := Document{
		Content:  "# Already has heading\n\nContent here",
		Metadata: map[string]any{"doc_title": "My Doc", "heading": "Section"},
	}

	result := transformer(document)
	assert.Equal(t, "# Already has heading\n\nContent here", result.Content, "should not prepend when content starts with heading")
}

func TestPrependChunkContext_SkipsContentWithLeadingWhitespace(t *testing.T) {
	transformer := PrependChunkContext()
	document := Document{
		Content:  "\n  # Heading with whitespace\n\nContent",
		Metadata: map[string]any{"doc_title": "My Doc"},
	}

	result := transformer(document)
	assert.Equal(t, "\n  # Heading with whitespace\n\nContent", result.Content)
}

func TestExtractRawFrontmatter(t *testing.T) {
	t.Run("valid frontmatter", func(t *testing.T) {
		raw, body, ok := extractRawFrontmatter("---\ntitle: Test\n---\nBody")
		assert.True(t, ok)
		assert.Equal(t, "title: Test", raw)
		assert.Equal(t, "Body", body)
	})

	t.Run("no frontmatter", func(t *testing.T) {
		_, body, ok := extractRawFrontmatter("# Just content")
		assert.False(t, ok)
		assert.Equal(t, "# Just content", body)
	})

	t.Run("unclosed frontmatter", func(t *testing.T) {
		_, body, ok := extractRawFrontmatter("---\ntitle: Test\nno close")
		assert.False(t, ok)
		assert.Equal(t, "---\ntitle: Test\nno close", body)
	})
}
