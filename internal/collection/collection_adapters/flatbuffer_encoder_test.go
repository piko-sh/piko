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
	"fmt"
	"strings"
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
	assert.ErrorIs(t, err, ErrEmptyCollection)
}

func TestFlatBufferEncoder_EncodeCollection_EmptySlice(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()
	_, err := encoder.EncodeCollection([]collection_dto.ContentItem{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyCollection)
}

func TestFlatBufferEncoder_EncodeCollection_SingleItem(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{
			Slug:     "getting-started",
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
			Slug:     "b-second",
			URL:      "/docs/b-second",
			Metadata: map[string]any{"title": "Second"},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "p"}},
			},
		},
		{
			Slug:     "a-first",
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
			Slug:     "hello",
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
			Slug:       "no-content",
			URL:        "/blog/no-content",
			Metadata:   map[string]any{"title": "No Content"},
			ContentAST: nil,
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)
	assert.NotEmpty(t, blob)
}

func TestFlatBufferEncoder_EncodeCollection_RejectsEmptySlug(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{
			ID:       "missing-slug",
			URL:      "/docs/foo",
			Metadata: map[string]any{},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
			},
		},
	}

	_, err := encoder.EncodeCollection(items)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptySlug)
	assert.Contains(t, err.Error(), "missing-slug")
}

func TestFlatBufferEncoder_EncodeCollection_RejectsDuplicateSlug(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{
			Slug:     "duplicate",
			URL:      "/docs/a",
			Metadata: map[string]any{},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
			},
		},
		{
			Slug:     "duplicate",
			URL:      "/docs/b",
			Metadata: map[string]any{},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "p"}},
			},
		},
	}

	_, err := encoder.EncodeCollection(items)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicateSlug)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestFlatBufferEncoder_EncodeCollection_RejectsInvalidSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		slug string
	}{
		{name: "control character", slug: "blog/po\x01st"},
		{name: "del character", slug: "blog/po\x7Fst"},
		{name: "leading slash", slug: "/blog"},
		{name: "trailing slash", slug: "blog/"},
		{name: "traversal segment", slug: "blog/../escape"},
		{name: "dot segment", slug: "blog/./post"},
		{name: "empty segment", slug: "blog//post"},
		{name: "invalid utf8", slug: "blog/\xC0\x80"},
		{name: "exceeds length cap", slug: "blog/" + strings.Repeat("a", MaxSlugBytes)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			encoder := NewFlatBufferEncoder()
			items := []collection_dto.ContentItem{
				{
					Slug:     tt.slug,
					Metadata: map[string]any{},
					ContentAST: &ast_domain.TemplateAST{
						RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
					},
				},
			}
			_, err := encoder.EncodeCollection(items)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidSlug)
		})
	}
}

func TestFlatBufferEncoder_RoundTrip_NestedSlug(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()
	items := []collection_dto.ContentItem{
		{
			Slug:     "get-started/introduction",
			Metadata: map[string]any{"title": "Introduction"},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "p"}},
			},
		},
		{
			Slug:     "tutorials",
			Metadata: map[string]any{"title": "Tutorials"},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "h1"}},
			},
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)

	for _, item := range items {
		_, _, _, lookupErr := encoder.DecodeCollectionItem(blob, item.Slug)
		require.NoError(t, lookupErr, "Should find slug: %s", item.Slug)
	}
}

func TestFlatBufferEncoder_EncodeCollection_RejectsTooManyItems(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()
	items := make([]collection_dto.ContentItem, MaxCollectionItems+1)
	for i := range items {
		items[i] = collection_dto.ContentItem{
			Slug:     fmt.Sprintf("item-%06d", i),
			Metadata: map[string]any{},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
			},
		}
	}

	_, err := encoder.EncodeCollection(items)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTooManyItems)
}

func TestFlatBufferEncoder_DecodeCollectionItem_EmptyBlob(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()
	_, _, _, err := encoder.DecodeCollectionItem(nil, "/docs/test")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyBlob)
}

func TestFlatBufferEncoder_DecodeCollectionItem_EmptyByteSlice(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()
	_, _, _, err := encoder.DecodeCollectionItem([]byte{}, "/docs/test")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyBlob)
}

func TestFlatBufferEncoder_RoundTrip(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{
			Slug:     "actions",
			URL:      "/docs/actions",
			Metadata: map[string]any{"title": "Actions", "order": float64(1)},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
			},
		},
		{
			Slug:     "getting-started",
			URL:      "/docs/getting-started",
			Metadata: map[string]any{"title": "Getting Started", "order": float64(0)},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "section"}},
			},
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)

	metaJSON, contentAST, excerptAST, err := encoder.DecodeCollectionItem(blob, "actions")
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
			Slug:     "post",
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

	metaJSON, contentAST, excerptAST, err := encoder.DecodeCollectionItem(blob, "post")
	require.NoError(t, err)
	assert.NotEmpty(t, metaJSON)
	assert.NotEmpty(t, contentAST)
	assert.NotEmpty(t, excerptAST)
}

func TestFlatBufferEncoder_DecodeCollectionItem_SlugNotFound(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{
			Slug:     "existing",
			URL:      "/docs/existing",
			Metadata: map[string]any{"title": "Existing"},
			ContentAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
			},
		},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)

	_, _, _, err = encoder.DecodeCollectionItem(blob, "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSlugNotInBlob)
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

func TestFlatBufferEncoder_SortsBySlug(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	items := []collection_dto.ContentItem{
		{Slug: "z-last", URL: "/z-last", Metadata: map[string]any{}, ContentAST: &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}}}},
		{Slug: "a-first", URL: "/a-first", Metadata: map[string]any{}, ContentAST: &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}}}},
		{Slug: "m-middle", URL: "/m-middle", Metadata: map[string]any{}, ContentAST: &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}}}},
	}

	blob, err := encoder.EncodeCollection(items)
	require.NoError(t, err)

	for _, item := range items {
		_, _, _, lookupErr := encoder.DecodeCollectionItem(blob, item.Slug)
		require.NoError(t, lookupErr, "Should find slug: %s", item.Slug)
	}
}

func TestFlatBufferEncoder_DeterministicMetadataMapEncoding(t *testing.T) {
	t.Parallel()

	encoder := NewFlatBufferEncoder()

	metadata := map[string]any{
		"title":       "Sample post",
		"description": "Why deterministic builds matter",
		"tags":        []string{"build", "determinism"},
		"navigation": map[string]any{
			"groups": map[string]any{
				"sidebar": map[string]any{"label": "Sidebar", "order": 1, "hidden": false},
				"footer":  map[string]any{"label": "Footer", "order": 2, "hidden": false},
				"top":     map[string]any{"label": "Top", "order": 3, "hidden": true},
			},
		},
		"authors": []string{"alice", "bob"},
		"meta":    map[string]any{"alpha": 1, "beta": 2, "gamma": 3, "delta": 4, "epsilon": 5},
	}

	items := []collection_dto.ContentItem{
		{
			Slug:       "deterministic",
			URL:        "/deterministic",
			Metadata:   metadata,
			ContentAST: &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}}},
		},
	}

	const runs = 16
	first, err := encoder.EncodeCollection(items)
	require.NoError(t, err)

	for i := 1; i < runs; i++ {
		again, err := encoder.EncodeCollection(items)
		require.NoError(t, err)
		require.Equal(t, first, again, "encoding run %d differs from run 0; metadata map iteration is not deterministic", i)
	}
}
