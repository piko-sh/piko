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
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func otterFactory(_ context.Context, namespace string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
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
		MaximumEntries: 10000,
		SearchSchema:   schema,
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

	err = s.Store(ctx, "test", &llm_dto.VectorDocument{})
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

func TestStore_SearchRanksEligibleDocumentsBeforeLimiting(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "catalog", 2)
	docs := []*llm_dto.VectorDocument{
		{ID: "nearest", Content: "unselected", Vector: []float32{1, 0}, Metadata: map[string]any{"group": "other"}},
		{ID: "next", Content: "selected", Vector: []float32{1, 0.5}, Metadata: map[string]any{"group": 2, "key.with.dots": true}},
		{ID: "third", Content: "selected", Vector: []float32{1, 1}, Metadata: map[string]any{"group": "2", "key.with.dots": true}},
		{ID: "far", Content: "selected", Vector: []float32{-1, 0}, Metadata: map[string]any{"group": "2", "key.with.dots": true}},
		{ID: "no-metadata", Content: "unselected", Vector: []float32{-1, 0.5}},
	}
	require.NoError(t, s.BulkStore(t.Context(), "catalog", docs))
	for _, query := range []string{"", "selected"} {
		window, err := s.lookupCache("catalog").Search(t.Context(), query, &cache_dto.SearchOptions{Vector: []float32{1, 0}, TopK: len(docs), Offset: 1, Limit: 1})
		require.NoError(t, err)
		assert.EqualValues(t, len(docs), window.Total)
		assert.Len(t, window.Items, 1)
		assert.Equal(t, 1, cap(window.Items), "the page must not retain all vector documents")
	}
	request := llm_dto.VectorSearchRequest{Namespace: "catalog", Vector: []float32{1, 0}, TopK: 1}
	all, err := s.Search(t.Context(), &request)
	require.NoError(t, err)
	require.Len(t, all.Results, 1)
	assert.Equal(t, "nearest", all.Results[0].ID)
	assert.Equal(t, 1, all.TotalCount, "a KNN search totals the ranked candidates it was asked for")

	request.Filter = map[string]any{"group": "2", "key.with.dots": true}
	got, err := s.Search(t.Context(), &request)
	require.NoError(t, err)
	require.Len(t, got.Results, 1)
	assert.Equal(t, "next", got.Results[0].ID)
	assert.Equal(t, 1, got.TotalCount, "the window widens only until TopK matches survive")
	assert.Nil(t, got.Results[0].Metadata)
	assert.Nil(t, got.Results[0].Vector)

	request.MinScore = new(float32(.8))
	got, err = s.Search(t.Context(), &request)
	require.NoError(t, err)
	require.Len(t, got.Results, 1)
	assert.Equal(t, "next", got.Results[0].ID)
	assert.Equal(t, 1, got.TotalCount)

	request.Filter = map[string]any{"missing": "value"}
	got, err = s.Search(t.Context(), &request)
	require.NoError(t, err)
	assert.Empty(t, got.Results)
	assert.Zero(t, got.TotalCount)

	request.Filter = map[string]any{"group": "2", "key.with.dots": true}
	request.MinScore = nil
	request.TopK = 3
	got, err = s.Search(t.Context(), &request)
	require.NoError(t, err)
	require.Len(t, got.Results, 3)
	assert.Equal(t, []string{"next", "third", "far"}, []string{got.Results[0].ID, got.Results[1].ID, got.Results[2].ID})
	assert.Equal(t, 3, got.TotalCount, "an exhausted window counts every match")

	request.Filter = nil
	request.MinScore = nil
	request.TopK = math.MaxInt
	got, err = s.Search(t.Context(), &request)
	require.NoError(t, err)
	assert.Len(t, got.Results, len(docs))
	assert.Equal(t, len(docs), got.TotalCount)

	text, err := s.Search(t.Context(), &llm_dto.VectorSearchRequest{Namespace: "catalog", TextQuery: "selected", TopK: 1})
	require.NoError(t, err)
	assert.Len(t, text.Results, 1)
	assert.Equal(t, 3, text.TotalCount)
}

func TestStore_SearchPreservesHybridProviderScoresWhenFiltering(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "hybrid", 2)
	require.NoError(t, s.BulkStore(t.Context(), "hybrid", []*llm_dto.VectorDocument{
		{ID: "excluded", Content: "other", Vector: []float32{1, 0}},
		{ID: "semantic", Content: "other", Vector: []float32{1, 0.1}, Metadata: map[string]any{"eligible": true}},
		{ID: "both", Content: "needle", Vector: []float32{.9, .3}, Metadata: map[string]any{"eligible": true}},
	}))
	recording := installRecordingCache(t, s, "hybrid")
	query := []float32{1, 0}
	got, err := s.Search(t.Context(), &llm_dto.VectorSearchRequest{Namespace: "hybrid", TextQuery: "needle", Vector: query, TopK: 1, Filter: map[string]any{"eligible": true}, IncludeMetadata: true, IncludeVectors: true})
	require.NoError(t, err)
	require.Len(t, got.Results, 1)
	require.NotEmpty(t, recording.windows)

	window := recording.windows[len(recording.windows)-1]
	ranked, err := recording.Cache.Search(t.Context(), "needle", &cache_dto.SearchOptions{Vector: query, TopK: window, Limit: window})
	require.NoError(t, err)
	var eligible []cache_dto.SearchHit[string, llm_dto.VectorDocument]
	for _, hit := range ranked.Items {
		if hit.Value.ID != "excluded" {
			eligible = append(eligible, hit)
		}
	}
	require.NotEmpty(t, eligible)
	assert.Equal(t, len(eligible), got.TotalCount)
	assert.Equal(t, eligible[0].Value.ID, got.Results[0].ID)
	assert.Equal(t, float32(eligible[0].Score), got.Results[0].Score)
	assert.Equal(t, eligible[0].Value.Vector, got.Results[0].Vector)
	assert.Equal(t, eligible[0].Value.Metadata, got.Results[0].Metadata)
}

type recordingCache struct {
	cache_domain.Cache[string, llm_dto.VectorDocument]
	windows []int
	limits  []int
}

func (c *recordingCache) Search(ctx context.Context, query string, opts *cache_dto.SearchOptions) (cache_dto.SearchResult[string, llm_dto.VectorDocument], error) {
	c.windows = append(c.windows, opts.TopK)
	c.limits = append(c.limits, opts.Limit)
	page, err := c.Cache.Search(ctx, query, opts)
	page.Total = 3
	return page, err
}

func installRecordingCache(t *testing.T, s *Store, namespace string) *recordingCache {
	t.Helper()
	recording := &recordingCache{Cache: s.lookupCache(namespace), windows: nil, limits: nil}
	s.mu.Lock()
	s.caches[namespace] = recording
	s.mu.Unlock()
	return recording
}

func TestStore_SearchProviderWindowsAndRawTotals(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "test", 2)
	require.NoError(t, s.BulkStore(t.Context(), "test", []*llm_dto.VectorDocument{
		{ID: "a", Vector: []float32{1, 0}}, {ID: "b", Vector: []float32{1, 1}}, {ID: "c", Vector: []float32{-1, 0}},
	}))
	recording := installRecordingCache(t, s, "test")

	request := llm_dto.VectorSearchRequest{Namespace: "test", Vector: []float32{1, 0}, TopK: 1}
	got, err := s.Search(t.Context(), &request)
	require.NoError(t, err)
	assert.Equal(t, 3, got.TotalCount, "unfiltered searches report the provider total")
	assert.Equal(t, []int{1}, recording.windows, "unfiltered searches ask only for TopK candidates")
	assert.Equal(t, []int{1}, recording.limits)

	recording.windows, recording.limits = nil, nil
	request.MinScore = new(float32(.5))
	got, err = s.Search(t.Context(), &request)
	require.NoError(t, err)
	assert.Equal(t, 1, got.TotalCount)
	assert.Len(t, got.Results, 1)
	assert.Equal(t, []int{1}, recording.windows, "a satisfied threshold search does not widen")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = s.Search(ctx, &request)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestStore_SearchWidensFilteredWindowUpToCeiling(t *testing.T) {
	s := New(otterFactory, WithMaxFilteredCandidates(4))
	t.Cleanup(func() { assert.NoError(t, s.Close(context.Background())) })
	createTestNamespace(t, s, "wide", 2)
	documents := make([]*llm_dto.VectorDocument, 0, 8)
	for index := range 8 {
		group := "common"
		if index == 7 {
			group = "rare"
		}
		documents = append(documents, &llm_dto.VectorDocument{
			ID:       fmt.Sprintf("document-%d", index),
			Vector:   []float32{1, float32(index) / 4},
			Metadata: map[string]any{"group": group},
		})
	}
	require.NoError(t, s.BulkStore(t.Context(), "wide", documents))
	recording := installRecordingCache(t, s, "wide")

	got, err := s.Search(t.Context(), &llm_dto.VectorSearchRequest{Namespace: "wide", Vector: []float32{1, 0}, TopK: 1, Filter: map[string]any{"group": "rare"}})
	require.NoError(t, err)
	assert.Empty(t, got.Results, "the only match lies beyond the candidate ceiling")
	assert.Equal(t, []int{1, 2, 4}, recording.windows, "the window doubles until it reaches the ceiling")

	recording.windows = nil
	got, err = s.Search(t.Context(), &llm_dto.VectorSearchRequest{Namespace: "wide", Vector: []float32{1, 0}, TopK: 100})
	require.NoError(t, err)
	assert.Len(t, got.Results, 4, "TopK is capped at the candidate ceiling")
	assert.Equal(t, []int{4}, recording.windows)
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

func TestStore_RejectsAVectorTheNamespaceCannotHold(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		vector []float32
	}{
		{name: "an empty vector", vector: nil},
		{name: "a shorter vector", vector: []float32{1, 0}},
		{name: "a longer vector", vector: []float32{1, 0, 0, 0}},
		{name: "a zero vector under cosine", vector: []float32{0, 0, 0}},
		{name: "a vector holding NaN", vector: []float32{1, float32(math.NaN()), 0}},
		{name: "a vector holding infinity", vector: []float32{1, float32(math.Inf(1)), 0}},
		{name: "a vector whose magnitude overflows", vector: []float32{3e38, 3e38, 0}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			s := newTestStore(t)
			createTestNamespace(t, s, "test", 3)
			ctx := t.Context()

			err := s.Store(ctx, "test", &llm_dto.VectorDocument{ID: "single", Content: "single", Vector: testCase.vector})
			require.ErrorIs(t, err, llm_domain.ErrInvalidVector)

			err = s.BulkStore(ctx, "test", []*llm_dto.VectorDocument{
				{ID: "valid", Content: "valid", Vector: []float32{1, 0, 0}},
				{ID: "invalid", Content: "invalid", Vector: testCase.vector},
			})
			require.ErrorIs(t, err, llm_domain.ErrInvalidVector)

			for _, id := range []string{"single", "valid", "invalid"} {
				got, err := s.Get(ctx, "test", id)
				require.NoError(t, err)
				assert.Nil(t, got, "nothing is stored when a vector is refused: %s", id)
			}
		})
	}
}

func TestStore_AcceptsAZeroVectorUnderEuclidean(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := t.Context()
	require.NoError(t, s.CreateNamespace(ctx, "origin", &llm_domain.VectorNamespaceConfig{
		Metric:    llm_dto.SimilarityEuclidean,
		Dimension: 3,
	}))

	require.NoError(t, s.Store(ctx, "origin", &llm_dto.VectorDocument{ID: "origin", Content: "origin", Vector: []float32{0, 0, 0}}))

	response, err := s.Search(ctx, &llm_dto.VectorSearchRequest{Namespace: "origin", Vector: []float32{0, 0, 0}, TopK: 1})
	require.NoError(t, err)
	require.Len(t, response.Results, 1)
	assert.Equal(t, "origin", response.Results[0].ID)
}

func TestStore_SearchRejectsAQueryVectorTheNamespaceCannotCompare(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		vector []float32
	}{
		{name: "a query of the wrong dimension", vector: []float32{1, 0}},
		{name: "a zero query under cosine", vector: []float32{0, 0, 0}},
		{name: "a query holding NaN", vector: []float32{1, float32(math.NaN()), 0}},
		{name: "a query holding infinity", vector: []float32{float32(math.Inf(-1)), 0, 0}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			s := newTestStore(t)
			createTestNamespace(t, s, "test", 3)
			ctx := t.Context()
			require.NoError(t, s.Store(ctx, "test", &llm_dto.VectorDocument{ID: "a", Content: "alpha", Vector: []float32{1, 0, 0}}))

			response, err := s.Search(ctx, &llm_dto.VectorSearchRequest{Namespace: "test", Vector: testCase.vector, TopK: 1})

			require.ErrorIs(t, err, cache_domain.ErrInvalidQueryVector)
			require.ErrorIs(t, err, llm_domain.ErrInvalidVector)
			assert.Nil(t, response)
		})
	}
}

type failingCache struct {
	cache_domain.Cache[string, llm_dto.VectorDocument]
	invalidateErr error
	closeErr      error
	closes        *atomic.Int32
}

func (c *failingCache) InvalidateAll(context.Context) error {
	return c.invalidateErr
}

func (c *failingCache) Close(ctx context.Context) error {
	c.closes.Add(1)
	return errors.Join(c.Cache.Close(ctx), c.closeErr)
}

func TestStore_LifecycleReportsCacheErrors(t *testing.T) {
	invalidateErr := errors.New("invalidate failed")
	closeErr := errors.New("close failed")
	var closes atomic.Int32
	factory := func(ctx context.Context, namespace string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		inner, err := otterFactory(ctx, namespace, config)
		if err != nil {
			return nil, err
		}
		return &failingCache{Cache: inner, invalidateErr: invalidateErr, closeErr: closeErr, closes: &closes}, nil
	}
	s := New(factory)

	require.NoError(t, s.CreateNamespace(t.Context(), "first", nil))
	require.NoError(t, s.CreateNamespace(t.Context(), "second", nil))
	require.ErrorIs(t, s.DeleteNamespace(t.Context(), "first"), invalidateErr)
	require.NoError(t, s.DeleteNamespace(t.Context(), "missing"))

	err := s.Close(t.Context())
	require.ErrorIs(t, err, closeErr)
	assert.Equal(t, int32(2), closes.Load(), "a namespace whose deletion failed is kept and closed with the rest")

	err = s.Store(t.Context(), "third", &llm_dto.VectorDocument{ID: "a", Vector: []float32{1, 0}})
	require.ErrorIs(t, err, errStoreClosed)
	require.ErrorIs(t, s.CreateNamespace(t.Context(), "third", nil), errStoreClosed)
}

func TestStore_ConcurrentAutoCreateKeepsOneCache(t *testing.T) {
	var created atomic.Int32
	factory := func(ctx context.Context, namespace string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		created.Add(1)
		return otterFactory(ctx, namespace, config)
	}
	s := New(factory)
	t.Cleanup(func() { assert.NoError(t, s.Close(context.Background())) })

	var wg sync.WaitGroup
	for index := range 16 {
		wg.Go(func() {
			document := &llm_dto.VectorDocument{ID: fmt.Sprintf("document-%d", index), Vector: []float32{1, float32(index)}}
			assert.NoError(t, s.Store(t.Context(), "shared", document))
		})
	}
	wg.Wait()

	keys := 0
	for range s.lookupCache("shared").Keys() {
		keys++
	}
	assert.Equal(t, 16, keys, "every document lands in the one surviving cache")
	assert.Equal(t, int32(1), created.Load(), "racing first writes share one factory call")
}

type blockingInvalidateCache struct {
	cache_domain.Cache[string, llm_dto.VectorDocument]
	started chan struct{}
	release chan struct{}
	err     error
	closes  *atomic.Int32
}

func (c *blockingInvalidateCache) InvalidateAll(ctx context.Context) error {
	close(c.started)
	<-c.release
	if c.err != nil {
		return c.err
	}
	return c.Cache.InvalidateAll(ctx)
}

func (c *blockingInvalidateCache) Close(ctx context.Context) error {
	c.closes.Add(1)
	return c.Cache.Close(ctx)
}

func newBlockingInvalidateStore(t *testing.T, err error) (*Store, *blockingInvalidateCache, *atomic.Int32) {
	t.Helper()
	var created, closes atomic.Int32
	var first *blockingInvalidateCache
	factory := func(ctx context.Context, namespace string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		inner, factoryErr := otterFactory(ctx, namespace, config)
		if factoryErr != nil {
			return nil, factoryErr
		}
		if created.Add(1) > 1 {
			return inner, nil
		}
		first = &blockingInvalidateCache{Cache: inner, started: make(chan struct{}), release: make(chan struct{}), err: err, closes: &closes}
		return first, nil
	}
	s := New(factory)
	createTestNamespace(t, s, "docs", 3)
	require.NotNil(t, first)
	return s, first, &created
}

func TestStore_DeleteNamespaceHoldsBackRecreationUntilInvalidationEnds(t *testing.T) {
	s, blocking, created := newBlockingInvalidateStore(t, nil)
	t.Cleanup(func() { assert.NoError(t, s.Close(context.Background())) })
	require.NoError(t, s.Store(t.Context(), "docs", &llm_dto.VectorDocument{ID: "old", Vector: []float32{1, 0, 0}}))

	deleted := make(chan error, 1)
	go func() { deleted <- s.DeleteNamespace(t.Context(), "docs") }()
	<-blocking.started

	stored := make(chan error, 1)
	go func() {
		stored <- s.Store(t.Context(), "docs", &llm_dto.VectorDocument{ID: "new", Vector: []float32{0, 1, 0}})
	}()

	select {
	case err := <-stored:
		require.Failf(t, "store finished during the deletion", "error: %v", err)
	default:
	}
	assert.Equal(t, int32(1), created.Load(), "no cache is created while the namespace is deleting")

	close(blocking.release)
	require.NoError(t, <-deleted)
	require.NoError(t, <-stored)

	got, err := s.Get(t.Context(), "docs", "new")
	require.NoError(t, err)
	require.NotNil(t, got, "a write that waited for the deletion survives it")
	old, err := s.Get(t.Context(), "docs", "old")
	require.NoError(t, err)
	assert.Nil(t, old)
	assert.Equal(t, int32(2), created.Load())
}

func TestStore_WaitingForADeletionHonoursCancellation(t *testing.T) {
	s, blocking, _ := newBlockingInvalidateStore(t, nil)
	t.Cleanup(func() { assert.NoError(t, s.Close(context.Background())) })

	deleted := make(chan error, 1)
	go func() { deleted <- s.DeleteNamespace(t.Context(), "docs") }()
	<-blocking.started

	ctx, cancel := context.WithCancelCause(t.Context())
	cause := errors.New("caller gave up")
	cancel(cause)
	err := s.Store(ctx, "docs", &llm_dto.VectorDocument{ID: "late", Vector: []float32{1, 0, 0}})
	require.ErrorIs(t, err, cause)

	close(blocking.release)
	require.NoError(t, <-deleted)
}

func TestStore_FailedDeleteNamespaceKeepsTheNamespaceForARetry(t *testing.T) {
	invalidateErr := errors.New("invalidate failed")
	s, blocking, created := newBlockingInvalidateStore(t, invalidateErr)
	t.Cleanup(func() { assert.NoError(t, s.Close(context.Background())) })
	require.NoError(t, s.Store(t.Context(), "docs", &llm_dto.VectorDocument{ID: "kept", Vector: []float32{1, 0, 0}}))

	close(blocking.release)
	require.ErrorIs(t, s.DeleteNamespace(t.Context(), "docs"), invalidateErr)

	got, err := s.Get(t.Context(), "docs", "kept")
	require.NoError(t, err)
	require.NotNil(t, got, "the namespace and its data stay until a deletion succeeds")
	assert.Equal(t, int32(1), created.Load(), "the original cache is restored rather than replaced")

	blocking.err = nil
	blocking.started = make(chan struct{})
	require.NoError(t, s.DeleteNamespace(t.Context(), "docs"))
	got, err = s.Get(t.Context(), "docs", "kept")
	require.NoError(t, err)
	assert.Nil(t, got, "the retried deletion removes the data")
}

func TestStore_DeleteNamespaceClosesACacheWhenCloseRanDuringIt(t *testing.T) {
	s, blocking, _ := newBlockingInvalidateStore(t, nil)

	deleted := make(chan error, 1)
	go func() { deleted <- s.DeleteNamespace(t.Context(), "docs") }()
	<-blocking.started

	require.NoError(t, s.Close(t.Context()))
	assert.Equal(t, int32(0), blocking.closes.Load(), "Close cannot see a cache that is deleting")

	close(blocking.release)
	require.NoError(t, <-deleted)
	assert.Equal(t, int32(1), blocking.closes.Load(), "the deletion closes the cache itself")
}

func TestStore_CacheCreatedAfterCloseIsClosed(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var closes atomic.Int32
	factory := func(ctx context.Context, namespace string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		close(entered)
		<-release
		inner, err := otterFactory(ctx, namespace, config)
		if err != nil {
			return nil, err
		}
		return &failingCache{Cache: inner, invalidateErr: nil, closeErr: nil, closes: &closes}, nil
	}
	s := New(factory)

	created := make(chan error, 1)
	go func() { created <- s.CreateNamespace(t.Context(), "late", nil) }()
	<-entered
	require.NoError(t, s.Close(t.Context()))
	close(release)

	require.ErrorIs(t, <-created, errStoreClosed)
	assert.Equal(t, int32(1), closes.Load(), "a cache finished after Close is closed straight away")
}

func TestStore_FactoryReceivesTheCallersContext(t *testing.T) {
	type contextKey struct{}
	var seen atomic.Value
	factory := func(ctx context.Context, namespace string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		seen.Store(ctx.Value(contextKey{}))
		return otterFactory(ctx, namespace, config)
	}
	s := New(factory)
	t.Cleanup(func() { assert.NoError(t, s.Close(context.Background())) })

	ctx := context.WithValue(t.Context(), contextKey{}, "caller")
	require.NoError(t, s.CreateNamespace(ctx, "docs", nil))
	assert.Equal(t, "caller", seen.Load())

	factoryErr := errors.New("factory failed")
	failing := New(func(context.Context, string, *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		return nil, factoryErr
	})
	t.Cleanup(func() { assert.NoError(t, failing.Close(context.Background())) })
	require.ErrorIs(t, failing.CreateNamespace(t.Context(), "docs", nil), factoryErr)

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, failing.CreateNamespace(cancelled, "docs", nil), context.Canceled)
}

type sheddingCache struct {
	cache_domain.Cache[string, llm_dto.VectorDocument]
	windows []int
}

func (c *sheddingCache) Search(ctx context.Context, query string, opts *cache_dto.SearchOptions) (cache_dto.SearchResult[string, llm_dto.VectorDocument], error) {
	c.windows = append(c.windows, opts.TopK)
	page, err := c.Cache.Search(ctx, query, opts)
	if len(page.Items) > 0 {
		page.Items = page.Items[1:]
	}
	return page, err
}

func TestStore_SearchWidensWhenTheProviderDropsCandidates(t *testing.T) {
	s := newTestStore(t)
	createTestNamespace(t, s, "shed", 2)
	documents := make([]*llm_dto.VectorDocument, 0, 8)
	for index := range 8 {
		documents = append(documents, &llm_dto.VectorDocument{
			ID:       fmt.Sprintf("document-%d", index),
			Vector:   []float32{1, float32(index) / 4},
			Metadata: map[string]any{"group": "common"},
		})
	}
	require.NoError(t, s.BulkStore(t.Context(), "shed", documents))
	shedding := &sheddingCache{Cache: s.lookupCache("shed"), windows: nil}
	s.mu.Lock()
	s.caches["shed"] = shedding
	s.mu.Unlock()

	got, err := s.Search(t.Context(), &llm_dto.VectorSearchRequest{Namespace: "shed", Vector: []float32{1, 0}, TopK: 2, Filter: map[string]any{"group": "common"}})
	require.NoError(t, err)
	assert.Len(t, got.Results, 2, "a page shortened by dropped entries does not end the widening")
	assert.Equal(t, []int{2, 4}, shedding.windows)
}

func TestNextWindowNeverPassesTheCeiling(t *testing.T) {
	testCases := []struct {
		name    string
		window  int
		ceiling int
		want    int
	}{
		{name: "doubles below the ceiling", window: 4, ceiling: 100, want: 8},
		{name: "stops at the ceiling", window: 60, ceiling: 100, want: 100},
		{name: "does not overflow near the largest int", window: math.MaxInt/2 + 1, ceiling: math.MaxInt, want: math.MaxInt},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, nextWindow(testCase.window, testCase.ceiling))
		})
	}
}
