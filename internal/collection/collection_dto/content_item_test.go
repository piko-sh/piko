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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentItem_GetMetadataString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		metadata     map[string]any
		key          string
		defaultValue string
		want         string
	}{
		{name: "string value", metadata: map[string]any{"title": "Hello"}, key: "title", defaultValue: "fallback", want: "Hello"},
		{name: "missing key", metadata: map[string]any{"title": "Hello"}, key: "author", defaultValue: "Unknown", want: "Unknown"},
		{name: "non-string value", metadata: map[string]any{"count": 42}, key: "count", defaultValue: "default", want: "default"},
		{name: "empty string value", metadata: map[string]any{"title": ""}, key: "title", defaultValue: "fallback", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			item := &ContentItem{Metadata: tt.metadata}
			assert.Equal(t, tt.want, item.GetMetadataString(tt.key, tt.defaultValue))
		})
	}
}

func TestContentItem_GetMetadataInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		metadata     map[string]any
		key          string
		defaultValue int
		want         int
	}{
		{name: "int value", metadata: map[string]any{"views": 100}, key: "views", defaultValue: 0, want: 100},
		{name: "int64 value", metadata: map[string]any{"views": int64(200)}, key: "views", defaultValue: 0, want: 200},
		{name: "float64 value", metadata: map[string]any{"views": float64(300)}, key: "views", defaultValue: 0, want: 300},
		{name: "missing key", metadata: map[string]any{}, key: "views", defaultValue: 42, want: 42},
		{name: "non-numeric value", metadata: map[string]any{"views": "many"}, key: "views", defaultValue: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			item := &ContentItem{Metadata: tt.metadata}
			assert.Equal(t, tt.want, item.GetMetadataInt(tt.key, tt.defaultValue))
		})
	}
}

func TestContentItem_GetMetadataBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		metadata     map[string]any
		key          string
		defaultValue bool
		want         bool
	}{
		{name: "true value", metadata: map[string]any{"draft": true}, key: "draft", defaultValue: false, want: true},
		{name: "false value", metadata: map[string]any{"draft": false}, key: "draft", defaultValue: true, want: false},
		{name: "missing key", metadata: map[string]any{}, key: "draft", defaultValue: true, want: true},
		{name: "non-bool value", metadata: map[string]any{"draft": "yes"}, key: "draft", defaultValue: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			item := &ContentItem{Metadata: tt.metadata}
			assert.Equal(t, tt.want, item.GetMetadataBool(tt.key, tt.defaultValue))
		})
	}
}

func TestContentItem_GetMetadataStringSlice(t *testing.T) {
	t.Parallel()

	t.Run("string slice", func(t *testing.T) {
		t.Parallel()

		item := &ContentItem{Metadata: map[string]any{
			"tags": []string{"go", "testing"},
		}}
		assert.Equal(t, []string{"go", "testing"}, item.GetMetadataStringSlice("tags", nil))
	})

	t.Run("any slice with strings", func(t *testing.T) {
		t.Parallel()

		item := &ContentItem{Metadata: map[string]any{
			"tags": []any{"go", "testing"},
		}}
		assert.Equal(t, []string{"go", "testing"}, item.GetMetadataStringSlice("tags", nil))
	})

	t.Run("any slice with mixed types", func(t *testing.T) {
		t.Parallel()

		item := &ContentItem{Metadata: map[string]any{
			"tags": []any{"go", 42, "testing"},
		}}
		assert.Equal(t, []string{"go", "testing"}, item.GetMetadataStringSlice("tags", nil))
	})

	t.Run("missing key", func(t *testing.T) {
		t.Parallel()

		item := &ContentItem{Metadata: map[string]any{}}
		assert.Equal(t, []string{"default"}, item.GetMetadataStringSlice("tags", []string{"default"}))
	})

	t.Run("non-slice value", func(t *testing.T) {
		t.Parallel()

		item := &ContentItem{Metadata: map[string]any{"tags": "not-a-slice"}}
		assert.Nil(t, item.GetMetadataStringSlice("tags", nil))
	})
}

func TestContentItem_HasMetadata(t *testing.T) {
	t.Parallel()

	item := &ContentItem{Metadata: map[string]any{"title": "Hello", "draft": nil}}

	assert.True(t, item.HasMetadata("title"))
	assert.True(t, item.HasMetadata("draft"))
	assert.False(t, item.HasMetadata("missing"))
}

func TestContentItem_IsPublished(t *testing.T) {
	t.Parallel()

	t.Run("published", func(t *testing.T) {
		t.Parallel()

		item := &ContentItem{PublishedAt: "2026-01-01T00:00:00Z"}
		assert.True(t, item.IsPublished())
	})

	t.Run("not published", func(t *testing.T) {
		t.Parallel()

		item := &ContentItem{}
		assert.False(t, item.IsPublished())
	})
}

func TestContentItem_IsDraft(t *testing.T) {
	t.Parallel()

	t.Run("no published at", func(t *testing.T) {
		t.Parallel()

		item := &ContentItem{Metadata: map[string]any{}}
		assert.True(t, item.IsDraft())
	})

	t.Run("published with draft metadata", func(t *testing.T) {
		t.Parallel()

		item := &ContentItem{
			PublishedAt: "2026-01-01",
			Metadata:    map[string]any{MetaKeyDraft: true},
		}
		assert.True(t, item.IsDraft())
	})

	t.Run("published without draft metadata", func(t *testing.T) {
		t.Parallel()

		item := &ContentItem{
			PublishedAt: "2026-01-01",
			Metadata:    map[string]any{},
		}
		assert.False(t, item.IsDraft())
	})

	t.Run("lowercase draft key ignored", func(t *testing.T) {
		t.Parallel()

		item := &ContentItem{
			PublishedAt: "2026-01-01",
			Metadata:    map[string]any{"draft": true},
		}
		assert.False(t, item.IsDraft(), "lowercase 'draft' key should not be recognised; use MetaKeyDraft")
	})

	t.Run("draft false in metadata with published date", func(t *testing.T) {
		t.Parallel()

		item := &ContentItem{
			PublishedAt: "2026-01-01",
			Metadata:    map[string]any{MetaKeyDraft: false},
		}
		assert.False(t, item.IsDraft())
	})
}

func TestContentItem_Clone(t *testing.T) {
	t.Parallel()

	original := &ContentItem{
		ID:             "item-1",
		Slug:           "test-item",
		Locale:         "en",
		TranslationKey: "blog/test",
		Metadata:       map[string]any{"title": "Original", "views": 100},
		RawContent:     "raw content",
		PlainContent:   "plain content",
		URL:            "/blog/test",
		ReadingTime:    5,
		CreatedAt:      "2026-01-01",
		UpdatedAt:      "2026-01-02",
		PublishedAt:    "2026-01-03",
	}

	clone := original.Clone()

	require.NotNil(t, clone)
	assert.Equal(t, original.ID, clone.ID)
	assert.Equal(t, original.Slug, clone.Slug)
	assert.Equal(t, original.Locale, clone.Locale)
	assert.Equal(t, original.TranslationKey, clone.TranslationKey)
	assert.Equal(t, original.RawContent, clone.RawContent)
	assert.Equal(t, original.URL, clone.URL)
	assert.Equal(t, original.ReadingTime, clone.ReadingTime)
	assert.Equal(t, original.PublishedAt, clone.PublishedAt)

	clone.Metadata["title"] = "Modified"
	assert.Equal(t, "Original", original.Metadata["title"])
}
