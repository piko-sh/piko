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
)

func TestMetadataKeyConsistency(t *testing.T) {
	item := &ContentItem{
		ID:             "test-id",
		Slug:           "test-slug",
		Locale:         "en",
		TranslationKey: "test-key",
		URL:            "/test/url",
		ReadingTime:    5,
		CreatedAt:      "2025-01-01",
		UpdatedAt:      "2025-01-02",
		PublishedAt:    "2025-01-03",
		Metadata: map[string]any{
			"Title":       "Test Title",
			"Description": "Test Description",
			"Draft":       false,
			"Tags":        []string{"go", "testing"},
			"WordCount":   100,
		},
	}

	assert.Equal(t, "ID", MetaKeyID, "MetaKeyID constant mismatch")
	assert.Equal(t, "Slug", MetaKeySlug, "MetaKeySlug constant mismatch")
	assert.Equal(t, "Locale", MetaKeyLocale, "MetaKeyLocale constant mismatch")
	assert.Equal(t, "TranslationKey", MetaKeyTranslationKey, "MetaKeyTranslationKey constant mismatch")
	assert.Equal(t, "URL", MetaKeyURL, "MetaKeyURL constant mismatch")
	assert.Equal(t, "Title", MetaKeyTitle, "MetaKeyTitle constant mismatch")
	assert.Equal(t, "Description", MetaKeyDescription, "MetaKeyDescription constant mismatch")
	assert.Equal(t, "Draft", MetaKeyDraft, "MetaKeyDraft constant mismatch")
	assert.Equal(t, "Tags", MetaKeyTags, "MetaKeyTags constant mismatch")
	assert.Equal(t, "WordCount", MetaKeyWordCount, "MetaKeyWordCount constant mismatch")
	assert.Equal(t, "ReadingTime", MetaKeyReadingTime, "MetaKeyReadingTime constant mismatch")
	assert.Equal(t, "Sections", MetaKeySections, "MetaKeySections constant mismatch")
	assert.Equal(t, "CreatedAt", MetaKeyCreatedAt, "MetaKeyCreatedAt constant mismatch")
	assert.Equal(t, "UpdatedAt", MetaKeyUpdatedAt, "MetaKeyUpdatedAt constant mismatch")
	assert.Equal(t, "PublishedAt", MetaKeyPublishedAt, "MetaKeyPublishedAt constant mismatch")
	assert.Equal(t, "Navigation", MetaKeyNavigation, "MetaKeyNavigation constant mismatch")

	item.Metadata[MetaKeyTitle] = "Test Title"
	title := item.GetMetadataString(MetaKeyTitle, "fallback")
	assert.Equal(t, "Test Title", title, "GetMetadataString with constant should work")
}

func TestMetadataKeyUsageInConvertItemMetadata(t *testing.T) {
	expectedKeys := []string{
		MetaKeyID,
		MetaKeySlug,
		MetaKeyLocale,
		MetaKeyTranslationKey,
		MetaKeyURL,
		MetaKeyReadingTime,
		MetaKeyCreatedAt,
		MetaKeyUpdatedAt,
		MetaKeyPublishedAt,
	}

	t.Run("RequiredKeysForVirtualisation", func(t *testing.T) {
		criticalKeys := []string{
			MetaKeySlug,
			MetaKeyURL,
		}

		for _, key := range criticalKeys {
			assert.NotEmpty(t, key, "Critical metadata key constant must not be empty")
			assert.True(t, len(key) > 0, "Critical metadata key '%s' must have a value", key)
		}
	})

	t.Run("AllStandardKeysDefined", func(t *testing.T) {
		for _, key := range expectedKeys {
			assert.NotEmpty(t, key, "Standard metadata key constant must not be empty")
		}
	})
}
