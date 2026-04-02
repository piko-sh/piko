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

package collection_adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
)

func TestNewFlatBufferEncoder(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()
	require.NotNil(t, encoder)
}

func TestFlatBufferEncoder_EncodeCollection_Empty(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()
	_, err := encoder.EncodeCollection(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot encode empty collection")
}

func TestFlatBufferEncoder_EncodeCollection_EmptySlice(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()
	_, err := encoder.EncodeCollection([]collection_dto.ContentItem{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot encode empty collection")
}

func TestFlatBufferEncoder_EncodeCollection_SingleItem(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{
			URL:      "/docs/getting-started",
			Metadata: map[string]any{"title": "Getting Started"},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{TagName: "div"},
				},
			},
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)
	assert.NotEmpty(t, blob)
}

func TestFlatBufferEncoder_EncodeCollection_MultipleItems(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{
			URL:      "/docs/b-second",
			Metadata: map[string]any{"title": "Second"},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "p"}},
			},
		},
		{
			URL:      "/docs/a-first",
			Metadata: map[string]any{"title": "First"},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
			},
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)
	assert.NotEmpty(t, blob)
}

func TestFlatBufferEncoder_EncodeCollection_WithExcerptAST(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{
			URL:      "/blog/hello",
			Metadata: map[string]any{"title": "Hello World"},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "article"}},
			},
			ExcerptAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "p"}},
			},
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)
	assert.NotEmpty(t, blob)
}

func TestFlatBufferEncoder_EncodeCollection_NilContentAST(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{
			URL:        "/blog/no-content",
			Metadata:   map[string]any{"title": "No Content"},
			ContentAST: nil,
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)
	assert.NotEmpty(t, blob)
}

func TestFlatBufferEncoder_DecodeCollectionItem_EmptyBlob(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()
	_, _, _, err := encoder.DecodeCollectionItem(nil, "/docs/test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot decode from empty blob")
}

func TestFlatBufferEncoder_DecodeCollectionItem_EmptyByteSlice(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()
	_, _, _, err := encoder.DecodeCollectionItem([]byte{}, "/docs/test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot decode from empty blob")
}

func TestFlatBufferEncoder_RoundTrip(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{
			URL:      "/docs/actions",
			Metadata: map[string]any{"title": "Actions", "order": float64(1)},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
			},
		},
		{
			URL:      "/docs/getting-started",
			Metadata: map[string]any{"title": "Getting Started", "order": float64(0)},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "section"}},
			},
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)

	metaJSON, contentAST, excerptAST, err := encoder.DecodeCollectionItem(blob, "/docs/actions")
	require.NoError(t, err)
	assert.NotEmpty(t, metaJSON)
	assert.NotEmpty(t, contentAST)
	assert.Empty(t, excerptAST)

	assert.Contains(t, string(metaJSON), "Actions")
}

func TestFlatBufferEncoder_RoundTrip_WithExcerpt(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{
			URL:      "/blog/post",
			Metadata: map[string]any{"title": "Post"},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "article"}},
			},
			ExcerptAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "p"}},
			},
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)

	metaJSON, contentAST, excerptAST, err := encoder.DecodeCollectionItem(blob, "/blog/post")
	require.NoError(t, err)
	assert.NotEmpty(t, metaJSON)
	assert.NotEmpty(t, contentAST)
	assert.NotEmpty(t, excerptAST)
}

func TestFlatBufferEncoder_DecodeCollectionItem_RouteNotFound(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{
			URL:      "/docs/existing",
			Metadata: map[string]any{"title": "Existing"},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
			},
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)

	_, _, _, err = encoder.DecodeCollectionItem(blob, "/docs/nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "route not found")
}

func TestFlatBufferEncoder_DecodeCollectionItem_InvalidBlob(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()
	_, _, _, err := encoder.DecodeCollectionItem([]byte("not a flatbuffer"), "/docs/test")
	require.Error(t, err)
}

func TestParseHybridKey_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		wantProv string
		wantColl string
	}{
		{name: "standard key", key: "cms:articles", wantProv: "cms", wantColl: "articles"},
		{name: "colons in collection", key: "cms:some:nested:name", wantProv: "cms", wantColl: "some:nested:name"},
		{name: "empty key", key: "", wantProv: "", wantColl: ""},
		{name: "no colon", key: "justtext", wantProv: "", wantColl: ""},
		{name: "leading colon", key: ":value", wantProv: "", wantColl: "value"},
		{name: "trailing colon", key: "key:", wantProv: "key", wantColl: ""},
		{name: "only colon", key: ":", wantProv: "", wantColl: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider, coll := parseHybridKey(tt.key)
			assert.Equal(t, tt.wantProv, provider)
			assert.Equal(t, tt.wantColl, coll)
		})
	}
}

func TestFlatBufferEncoder_SortsByURL(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{URL: "/z-last", Metadata: map[string]any{}, ContentAST: &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}}}},
		{URL: "/a-first", Metadata: map[string]any{}, ContentAST: &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}}}},
		{URL: "/m-middle", Metadata: map[string]any{}, ContentAST: &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}}}},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)

	for _, item := range items {
		_, _, _, lookupErr := encoder.DecodeCollectionItem(blob, item.URL)
		require.NoError(t, lookupErr, "Should find route: %s", item.URL)
	}
}
