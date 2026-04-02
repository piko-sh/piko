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

package vector_cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func otterFactory(namespace string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
	metric := "cosine"
	dimension := 3
	if config != nil {
		if config.Metric != "" {
			metric = string(config.Metric)
		}
		if config.Dimension > 0 {
			dimension = config.Dimension
		}
	}

	schema := cache_dto.NewSearchSchema(
		cache_dto.VectorFieldWithMetric("Vector", dimension, metric),
		cache_dto.TextField("Content"),
	)

	opts := cache_dto.Options[string, llm_dto.VectorDocument]{
		MaximumSize:  10000,
		SearchSchema: schema,
	}

	return provider_otter.OtterProviderFactory(opts)
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s := New(otterFactory)
	t.Cleanup(func() {
		_ = s.Close(context.Background())
	})
	return s
}

func createTestNamespace(t *testing.T, s *Store, ns string, dim int) {
	t.Helper()
	err := s.CreateNamespace(context.Background(), ns, &llm_domain.VectorNamespaceConfig{
		Metric:    llm_dto.SimilarityCosine,
		Dimension: dim,
	})
	require.NoError(t, err)
}

func TestStore_StoreAndGet(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "test", 3)
	ctx := context.Background()

	document := &llm_dto.VectorDocument{
		ID:      "doc1",
		Content: "hello world",
		Vector:  []float32{1, 0, 0},
		Metadata: map[string]any{
			"source": "test",
		},
	}

	err := s.Store(ctx, "test", document)
	require.NoError(t, err)

	got, err := s.Get(ctx, "test", "doc1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "doc1", got.ID)
	assert.Equal(t, "hello world", got.Content)
	assert.Equal(t, []float32{1, 0, 0}, got.Vector)
	assert.Equal(t, "test", got.Metadata["source"])
}

func TestStore_StoreValidation(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "test", 3)
	ctx := context.Background()

	err := s.Store(ctx, "test", nil)
	assert.Error(t, err)

	err = s.Store(ctx, "test", &llm_dto.VectorDocument{ID: ""})
	assert.Error(t, err)
}

func TestStore_GetNonexistent(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "test", 3)
	ctx := context.Background()

	got, err := s.Get(ctx, "test", "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestStore_Delete(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "test", 3)
	ctx := context.Background()

	document := &llm_dto.VectorDocument{
		ID:      "doc1",
		Content: "hello",
		Vector:  []float32{1, 0, 0},
	}

	err := s.Store(ctx, "test", document)
	require.NoError(t, err)

	err = s.Delete(ctx, "test", "doc1")
	require.NoError(t, err)

	got, err := s.Get(ctx, "test", "doc1")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestStore_BulkStore(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "test", 3)
	ctx := context.Background()

	docs := []*llm_dto.VectorDocument{
		{ID: "a", Content: "alpha", Vector: []float32{1, 0, 0}},
		{ID: "b", Content: "beta", Vector: []float32{0, 1, 0}},
		{ID: "c", Content: "gamma", Vector: []float32{0, 0, 1}},
	}

	err := s.BulkStore(ctx, "test", docs)
	require.NoError(t, err)

	for _, d := range docs {
		got, err := s.Get(ctx, "test", d.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, d.Content, got.Content)
	}
}

func TestStore_Search(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "test", 3)
	ctx := context.Background()

	docs := []*llm_dto.VectorDocument{
		{ID: "a", Content: "alpha", Vector: []float32{1, 0, 0}},
		{ID: "b", Content: "beta", Vector: []float32{0, 1, 0}},
		{ID: "c", Content: "gamma", Vector: []float32{0.9, 0.1, 0}},
	}

	err := s.BulkStore(ctx, "test", docs)
	require.NoError(t, err)

	response, err := s.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace:       "test",
		Vector:          []float32{1, 0, 0},
		TopK:            2,
		IncludeMetadata: true,
	})
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Results, 2)

	assert.Equal(t, "a", response.Results[0].ID)
	assert.InDelta(t, 1.0, response.Results[0].Score, 0.01)
}

func TestStore_SearchWithMinScore(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "test", 3)
	ctx := context.Background()

	docs := []*llm_dto.VectorDocument{
		{ID: "exact", Content: "exact", Vector: []float32{1, 0, 0}},
		{ID: "orthogonal", Content: "orthogonal", Vector: []float32{0, 1, 0}},
	}

	err := s.BulkStore(ctx, "test", docs)
	require.NoError(t, err)

	response, err := s.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "test",
		Vector:    []float32{1, 0, 0},
		TopK:      10,
		MinScore:  new(float32(0.9)),
	})
	require.NoError(t, err)
	require.Len(t, response.Results, 1)
	assert.Equal(t, "exact", response.Results[0].ID)
}

func TestStore_SearchIncludeVectors(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "test", 3)
	ctx := context.Background()

	err := s.Store(ctx, "test", &llm_dto.VectorDocument{
		ID:      "doc1",
		Content: "hello",
		Vector:  []float32{1, 0, 0},
	})
	require.NoError(t, err)

	response, err := s.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace:      "test",
		Vector:         []float32{1, 0, 0},
		TopK:           1,
		IncludeVectors: true,
	})
	require.NoError(t, err)
	require.Len(t, response.Results, 1)
	assert.Equal(t, []float32{1, 0, 0}, response.Results[0].Vector)

	response, err = s.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace:      "test",
		Vector:         []float32{1, 0, 0},
		TopK:           1,
		IncludeVectors: false,
	})
	require.NoError(t, err)
	require.Len(t, response.Results, 1)
	assert.Nil(t, response.Results[0].Vector)
}

func TestStore_DeleteByFilter(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "test", 3)
	ctx := context.Background()

	docs := []*llm_dto.VectorDocument{
		{ID: "a", Content: "alpha", Vector: []float32{1, 0, 0}, Metadata: map[string]any{"type": "foo"}},
		{ID: "b", Content: "beta", Vector: []float32{0, 1, 0}, Metadata: map[string]any{"type": "bar"}},
		{ID: "c", Content: "gamma", Vector: []float32{0, 0, 1}, Metadata: map[string]any{"type": "foo"}},
	}

	err := s.BulkStore(ctx, "test", docs)
	require.NoError(t, err)

	count, err := s.DeleteByFilter(ctx, "test", map[string]any{"type": "foo"})
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	got, err := s.Get(ctx, "test", "b")
	require.NoError(t, err)
	require.NotNil(t, got)

	got, err = s.Get(ctx, "test", "a")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestStore_CreateAndDeleteNamespace(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	err := s.CreateNamespace(ctx, "ns1", &llm_domain.VectorNamespaceConfig{
		Metric:    llm_dto.SimilarityCosine,
		Dimension: 3,
	})
	require.NoError(t, err)

	err = s.Store(ctx, "ns1", &llm_dto.VectorDocument{
		ID:      "doc1",
		Content: "hello",
		Vector:  []float32{1, 0, 0},
	})
	require.NoError(t, err)

	err = s.DeleteNamespace(ctx, "ns1")
	require.NoError(t, err)

	got, err := s.Get(ctx, "ns1", "doc1")
	require.NoError(t, err)
	assert.Nil(t, got, "document should not exist in freshly recreated namespace")
}

func TestStore_NamespaceAutoCreated(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	err := s.Store(ctx, "auto", &llm_dto.VectorDocument{
		ID:      "doc1",
		Content: "hello",
		Vector:  []float32{1, 0, 0},
	})
	require.NoError(t, err)

	got, err := s.Get(ctx, "auto", "doc1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "doc1", got.ID)
}

func TestStore_CreateNamespaceIdempotent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	config := &llm_domain.VectorNamespaceConfig{
		Metric:    llm_dto.SimilarityCosine,
		Dimension: 3,
	}

	err := s.CreateNamespace(ctx, "test", config)
	require.NoError(t, err)

	err = s.CreateNamespace(ctx, "test", config)
	require.NoError(t, err)
}

func TestStore_Close(t *testing.T) {
	s := New(otterFactory)
	ctx := context.Background()

	err := s.CreateNamespace(ctx, "ns1", &llm_domain.VectorNamespaceConfig{
		Metric:    llm_dto.SimilarityCosine,
		Dimension: 3,
	})
	require.NoError(t, err)

	err = s.Close(ctx)
	require.NoError(t, err)

	_, err = s.Get(ctx, "ns1", "doc1")
	assert.Error(t, err)
}

func TestStore_NilSearchRequest(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	response, err := s.Search(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Empty(t, response.Results)
}

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		filter   map[string]any
		name     string
		document llm_dto.VectorDocument
		expected bool
	}{
		{
			name:     "empty filter matches all",
			document: llm_dto.VectorDocument{Metadata: map[string]any{"a": "b"}},
			filter:   nil,
			expected: true,
		},
		{
			name:     "exact match",
			document: llm_dto.VectorDocument{Metadata: map[string]any{"type": "foo"}},
			filter:   map[string]any{"type": "foo"},
			expected: true,
		},
		{
			name:     "mismatch",
			document: llm_dto.VectorDocument{Metadata: map[string]any{"type": "foo"}},
			filter:   map[string]any{"type": "bar"},
			expected: false,
		},
		{
			name:     "missing key",
			document: llm_dto.VectorDocument{Metadata: map[string]any{"type": "foo"}},
			filter:   map[string]any{"missing": "value"},
			expected: false,
		},
		{
			name:     "nil metadata",
			document: llm_dto.VectorDocument{},
			filter:   map[string]any{"type": "foo"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, matchesFilter(tt.document, tt.filter))
		})
	}
}
