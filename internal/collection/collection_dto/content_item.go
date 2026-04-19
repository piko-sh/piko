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

package collection_dto

import (
	"maps"

	"piko.sh/piko/internal/ast/ast_domain"
)

// ContentItem represents a single piece of content from any provider.
//
// Acts as the common format for content in the collection system. Provides a
// standard way to represent content from markdown files, CMS entries, database
// records, and other data sources. The design supports multiple providers
// through the Metadata map, stores both raw content and parsed AST forms, and
// includes built-in support for translations via the Locale and TranslationKey
// fields.
type ContentItem struct {
	// Metadata holds key-value pairs from frontmatter, CMS fields, or database
	// columns. This map lets providers store extra data without changing the
	// ContentItem structure.
	Metadata map[string]any

	// ExcerptAST is the parsed AST of a short content summary; nil if no
	// excerpt is available.
	ExcerptAST *ast_domain.TemplateAST

	// ContentAST holds the parsed content as a Piko AST for rendering.
	ContentAST *ast_domain.TemplateAST

	// PlainContent is the plain text version of the content, used for search indexing.
	PlainContent string

	// TranslationKey groups related translations together (e.g. "blog/post-1").
	TranslationKey string

	// RawContent is the original content before processing, such as markdown,
	// HTML, or JSON.
	RawContent string

	// ID is the unique identifier for this item within its collection.
	ID string

	// Locale is the language code in ISO 639-1 format (e.g. "en", "fr", "de").
	Locale string

	// Slug is the URL-friendly identifier for this content item.
	Slug string

	// URL is the public web address where this content can be accessed.
	URL string

	// CreatedAt is when the content was created, in ISO 8601 format.
	CreatedAt string

	// UpdatedAt is the time the content was last changed (ISO 8601 format).
	UpdatedAt string

	// PublishedAt is when the content was published in ISO 8601 format.
	// May differ from CreatedAt; empty means unpublished.
	PublishedAt string

	// ReadingTime is the estimated reading time in minutes.
	ReadingTime int
}

// GetMetadataString retrieves a string value from metadata with a fallback.
//
// This helper simplifies accessing common string fields from the metadata map.
//
// Takes key (string) which specifies the metadata field to retrieve.
// Takes defaultValue (string) which is returned if the key is missing or not a
// string.
//
// Returns string which is the metadata value or the default.
func (c *ContentItem) GetMetadataString(key, defaultValue string) string {
	if value, ok := c.Metadata[key]; ok {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

// GetMetadataInt retrieves an integer value from metadata with a fallback.
//
// Takes key (string) which specifies the metadata field to retrieve.
// Takes defaultValue (int) which is returned when the key is missing or not
// numeric.
//
// Returns int which is the metadata value or the default if not found.
func (c *ContentItem) GetMetadataInt(key string, defaultValue int) int {
	if value, ok := c.Metadata[key]; ok {
		switch v := value.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

// GetMetadataBool retrieves a boolean value from metadata with a fallback.
//
// Takes key (string) which specifies the metadata key to look up.
// Takes defaultValue (bool) which is returned if the key is missing or not a
// boolean.
//
// Returns bool which is the metadata value if found, or defaultValue otherwise.
func (c *ContentItem) GetMetadataBool(key string, defaultValue bool) bool {
	if value, ok := c.Metadata[key]; ok {
		if boolValue, ok := value.(bool); ok {
			return boolValue
		}
	}
	return defaultValue
}

// GetMetadataStringSlice gets a string slice from the metadata.
//
// Takes key (string) which specifies the metadata key to look up.
// Takes defaultValue ([]string) which is returned when the key is not found.
//
// Returns []string which contains the metadata value or the default.
func (c *ContentItem) GetMetadataStringSlice(key string, defaultValue []string) []string {
	if value, ok := c.Metadata[key]; ok {
		switch v := value.(type) {
		case []string:
			return v
		case []any:
			result := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return defaultValue
}

// HasMetadata checks if a metadata key exists.
//
// Takes key (string) which specifies the metadata key to look for.
//
// Returns bool which is true if the key exists in the metadata map.
func (c *ContentItem) HasMetadata(key string) bool {
	_, ok := c.Metadata[key]
	return ok
}

// IsPublished checks if the content has been published.
//
// Content is considered published if it has a PublishedAt timestamp.
//
// Returns bool which is true when the content has been published.
func (c *ContentItem) IsPublished() bool {
	return c.PublishedAt != ""
}

// IsDraft checks if the content is a draft.
//
// Content is a draft if either:
// - It has no PublishedAt timestamp, OR
// - It explicitly has draft: true in metadata
//
// Returns bool which is true when the content is a draft.
func (c *ContentItem) IsDraft() bool {
	return c.PublishedAt == "" || c.GetMetadataBool(MetaKeyDraft, false)
}

// Clone creates a deep copy of the ContentItem.
//
// Use it when a provider needs to change content without affecting the
// original. The metadata map is copied, but AST pointers are shallow-copied.
//
// Returns *ContentItem which is the copied item.
func (c *ContentItem) Clone() *ContentItem {
	metadataCopy := make(map[string]any, len(c.Metadata))
	maps.Copy(metadataCopy, c.Metadata)

	return &ContentItem{
		ID:             c.ID,
		Slug:           c.Slug,
		Locale:         c.Locale,
		TranslationKey: c.TranslationKey,
		Metadata:       metadataCopy,
		RawContent:     c.RawContent,
		PlainContent:   c.PlainContent,
		ContentAST:     c.ContentAST,
		ExcerptAST:     c.ExcerptAST,
		URL:            c.URL,
		ReadingTime:    c.ReadingTime,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		PublishedAt:    c.PublishedAt,
	}
}
