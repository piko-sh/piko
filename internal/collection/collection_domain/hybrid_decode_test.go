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

package collection_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/collection/collection_adapters"
	"piko.sh/piko/internal/collection/collection_dto"
)

type testBlogPost struct {
	Title string `json:"title"`
	URL   string `json:"URL"`
}

func TestDecodeCollectionBlob_EmptyBlob(t *testing.T) {
	t.Parallel()

	t.Run("nil blob", func(t *testing.T) {
		t.Parallel()
		result, err := DecodeCollectionBlob[testBlogPost](nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()
		result, err := DecodeCollectionBlob[testBlogPost]([]byte{})
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestDecodeCollectionBlob_InvalidBlob(t *testing.T) {
	t.Parallel()

	_, err := DecodeCollectionBlob[testBlogPost]([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decoding collection blob")
}

func TestDecodeCollectionBlob_RoundTrip(t *testing.T) {
	t.Parallel()

	encoder := &collection_adapters.FlatBufferEncoder{}
	items := []collection_dto.ContentItem{
		{
			Slug:     "first-post",
			URL:      "/blog/first-post",
			Metadata: map[string]any{"title": "First Post", "URL": "/blog/first-post"},
		},
		{
			Slug:     "second-post",
			URL:      "/blog/second-post",
			Metadata: map[string]any{"title": "Second Post", "URL": "/blog/second-post"},
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)

	result, err := DecodeCollectionBlob[testBlogPost](blob)
	require.NoError(t, err)
	require.Len(t, result, 2)

	titles := map[string]bool{}
	for _, item := range result {
		titles[item.Title] = true
	}
	assert.True(t, titles["First Post"], "expected First Post in results")
	assert.True(t, titles["Second Post"], "expected Second Post in results")
}

func TestDecodeCollectionBlob_UnmarshalFailure(t *testing.T) {
	t.Parallel()

	type strictTarget struct {
		RequiredInt int `json:"required_int"`
	}

	encoder := &collection_adapters.FlatBufferEncoder{}
	items := []collection_dto.ContentItem{
		{
			Slug:     "a",
			URL:      "/a",
			Metadata: map[string]any{"title": "works", "required_int": 42},
		},
		{
			Slug:     "b",
			URL:      "/b",
			Metadata: map[string]any{"title": "also works", "required_int": 99},
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)

	result, err := DecodeCollectionBlob[strictTarget](blob)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}
