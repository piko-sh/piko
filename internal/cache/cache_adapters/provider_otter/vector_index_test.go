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

package provider_otter

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/hnsw"
	"piko.sh/vectormaths"
)

func TestVectorIndex_AddAndSearch(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	index.Add("a", []float32{1, 0, 0})
	index.Add("b", []float32{0, 1, 0})
	index.Add("c", []float32{0.9, 0.1, 0})

	hits := mustSearchIndex(t, index, []float32{1, 0, 0}, 2, nil)
	require.Len(t, hits, 2)

	assert.Equal(t, "a", hits[0].Key)
	assert.InDelta(t, 1.0, hits[0].Score, 0.01)
}

func TestVectorIndex_Remove(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	index.Add("a", []float32{1, 0, 0})
	index.Add("b", []float32{0, 1, 0})

	index.Remove("a")

	hits := mustSearchIndex(t, index, []float32{1, 0, 0}, 5, nil)
	for _, h := range hits {
		assert.NotEqual(t, "a", h.Key)
	}
}

func TestVectorIndex_Clear(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	index.Add("a", []float32{1, 0, 0})
	index.Add("b", []float32{0, 1, 0})

	index.Clear()

	hits := mustSearchIndex(t, index, []float32{1, 0, 0}, 5, nil)
	assert.Nil(t, hits)
}

func TestVectorIndex_MinScore(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	index.Add("exact", []float32{1, 0, 0})
	index.Add("orthogonal", []float32{0, 1, 0})

	hits := mustSearchIndex(t, index, []float32{1, 0, 0}, 10, new(float32(0.9)))

	require.Len(t, hits, 1)
	assert.Equal(t, "exact", hits[0].Key)
}

func TestVectorIndex_DimensionMismatch(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	require.ErrorIs(t, index.Add("wrong", []float32{1, 0}), hnsw.ErrVectorDimensionMismatch)
	hits := mustSearchIndex(t, index, []float32{1, 0, 0}, 1, nil)
	assert.Nil(t, hits)

	require.NoError(t, index.Add("ok", []float32{1, 0, 0}))
	hits, err := index.Search([]float32{1, 0}, 1, nil)
	require.ErrorIs(t, err, cache_domain.ErrInvalidQueryVector)
	require.ErrorIs(t, err, hnsw.ErrVectorDimensionMismatch)
	assert.Nil(t, hits)
}

func TestVectorIndex_EuclideanMetric(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Euclidean)

	index.Add("origin", []float32{0, 0, 0})
	index.Add("near", []float32{0.1, 0, 0})
	index.Add("far", []float32{10, 10, 10})

	hits := mustSearchIndex(t, index, []float32{0, 0, 0}, 3, nil)
	require.Len(t, hits, 3)

	assert.Equal(t, "origin", hits[0].Key)
	assert.InDelta(t, 1.0, hits[0].Score, 0.01)

	assert.Equal(t, "near", hits[1].Key)
	assert.Greater(t, hits[1].Score, hits[2].Score)
}

func TestVectorIndex_DotProductMetric(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.DotProduct)

	index.Add("high", []float32{1, 1, 1})
	index.Add("medium", []float32{0.5, 0.5, 0.5})
	index.Add("low", []float32{0, 0, 0})

	hits := mustSearchIndex(t, index, []float32{1, 1, 1}, 3, nil)
	require.Len(t, hits, 3)

	assert.Equal(t, "high", hits[0].Key)
	assert.Equal(t, "medium", hits[1].Key)
}

func TestVectorIndex_EmptySearch(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	hits := mustSearchIndex(t, index, []float32{1, 0, 0}, 5, nil)
	assert.Nil(t, hits)
}

func TestVectorIndex_ZeroTopK(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	index.Add("a", []float32{1, 0, 0})

	hits := mustSearchIndex(t, index, []float32{1, 0, 0}, 0, nil)
	assert.Nil(t, hits)
}

func TestVectorIndex_AutoDetectDimension(t *testing.T) {
	index := NewVectorIndex[string](0, vectormaths.Cosine)

	hits := mustSearchIndex(t, index, []float32{1, 0, 0}, 5, nil)
	assert.Nil(t, hits)

	index.Add("a", []float32{1, 0, 0})
	index.Add("b", []float32{0, 1, 0})

	hits = mustSearchIndex(t, index, []float32{1, 0, 0}, 2, nil)
	require.Len(t, hits, 2)
	assert.Equal(t, "a", hits[0].Key)
	assert.InDelta(t, 1.0, hits[0].Score, 0.01)

	index.Add("wrong", []float32{1, 0})
	hits = mustSearchIndex(t, index, []float32{1, 0, 0}, 10, nil)
	require.Len(t, hits, 2, "wrong-dimension vector should not be indexed")
}

func TestVectorIndex_AutoDetect_RemoveAndClear(t *testing.T) {
	index := NewVectorIndex[string](0, vectormaths.Cosine)

	index.Remove("nonexistent")
	index.Clear()

	index.Add("a", []float32{1, 0, 0})
	index.Remove("a")
	hits := mustSearchIndex(t, index, []float32{1, 0, 0}, 5, nil)
	assert.Empty(t, hits)
}

func TestDistanceToSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		metric   vectormaths.Metric
		distance float32
		expected float32
	}{
		{
			name:     "cosine zero distance",
			distance: 0,
			metric:   vectormaths.Cosine,
			expected: 1.0,
		},
		{
			name:     "cosine full distance",
			distance: 2.0,
			metric:   vectormaths.Cosine,
			expected: -1.0,
		},
		{
			name:     "euclidean zero distance",
			distance: 0,
			metric:   vectormaths.Euclidean,
			expected: 1.0,
		},
		{
			name:     "euclidean unit squared distance",
			distance: 1.0,
			metric:   vectormaths.Euclidean,
			expected: 0.5,
		},
		{
			name:     "dot product zero distance",
			distance: 0,
			metric:   vectormaths.DotProduct,
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := distanceToSimilarity(tt.distance, tt.metric)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestVectorIndex_Replace(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		replacement []float32
		wantLen     int
	}{
		{name: "a new vector replaces the old one", replacement: []float32{0, 1, 0}, wantLen: 1},
		{name: "no vector removes the key", replacement: nil, wantLen: 0},
		{name: "a vector of the wrong dimension removes the key", replacement: []float32{1, 0}, wantLen: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			index := NewVectorIndex[string](3, vectormaths.Cosine)
			index.Replace("a", []float32{1, 0, 0})

			index.Replace("a", testCase.replacement)

			assert.Equal(t, testCase.wantLen, index.graph.Load().Len())
			if testCase.wantLen == 0 {
				assert.Empty(t, mustSearchIndex(t, index, []float32{1, 0, 0}, 1, nil))
				return
			}
			hits := mustSearchIndex(t, index, testCase.replacement, 1, nil)
			require.Len(t, hits, 1)
			assert.InDelta(t, 1.0, hits[0].Score, 1e-5)
		})
	}
}

func TestVectorIndex_Replace_ReplacesAHeldKeyWhenFull(t *testing.T) {
	t.Parallel()

	index := NewVectorIndex[string](3, vectormaths.Cosine)
	index.maxVectors = 2
	index.Replace("a", []float32{1, 0, 0})
	index.Replace("b", []float32{0, 1, 0})

	index.Replace("a", []float32{0, 0, 1})
	index.Replace("c", []float32{1, 1, 0})

	assert.Equal(t, 2, index.graph.Load().Len())
	hits := mustSearchIndex(t, index, []float32{0, 0, 1}, 1, nil)
	require.Len(t, hits, 1)
	assert.Equal(t, "a", hits[0].Key, "a held key must take its new vector even when the index is full")
	assert.InDelta(t, 1.0, hits[0].Score, 1e-5)
}

func TestVectorIndex_Replace_AutoDetectsTheDimension(t *testing.T) {
	t.Parallel()

	index := NewVectorIndex[string](0, vectormaths.Cosine)
	index.Replace("a", []float32{1, 0})
	index.Replace("b", []float32{1, 0, 0})

	assert.Equal(t, 1, index.graph.Load().Len(), "the first vector fixes the dimension, so a longer one is not indexed")
	hits := mustSearchIndex(t, index, []float32{1, 0}, 2, nil)
	require.Len(t, hits, 1)
	assert.Equal(t, "a", hits[0].Key)
}

func TestVectorIndex_SearchRejectsAZeroQueryUnderCosine(t *testing.T) {
	t.Parallel()

	index := NewVectorIndex[string](3, vectormaths.Cosine)
	require.NoError(t, index.Add("a", []float32{1, 0, 0}))

	hits, err := index.Search([]float32{0, 0, 0}, 1, nil)

	require.ErrorIs(t, err, cache_domain.ErrInvalidQueryVector)
	require.ErrorIs(t, err, hnsw.ErrZeroVector)
	assert.Nil(t, hits)
}

func TestVectorIndex_SearchBeforeAnythingIsIndexedFindsNothing(t *testing.T) {
	t.Parallel()

	index := NewVectorIndex[string](0, vectormaths.Cosine)

	hits, err := index.Search([]float32{0, 0, 0}, 1, nil)

	require.NoError(t, err, "an index that has not seen a vector has no dimension to check against")
	assert.Nil(t, hits)
}

func TestVectorIndex_AddRejectsAnInvalidVector(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantErr error
		name    string
		vector  []float32
	}{
		{name: "an empty vector", vector: nil, wantErr: hnsw.ErrVectorDimensionMismatch},
		{name: "a zero vector under cosine", vector: []float32{0, 0, 0}, wantErr: hnsw.ErrZeroVector},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			index := NewVectorIndex[string](3, vectormaths.Cosine)

			require.ErrorIs(t, index.Add("a", testCase.vector), testCase.wantErr)
			assert.Zero(t, index.graph.Load().Len())
		})
	}
}

func TestVectorIndex_AddOfAnEmptyVectorLeavesTheDimensionUnset(t *testing.T) {
	t.Parallel()

	index := NewVectorIndex[string](0, vectormaths.Cosine)

	require.Error(t, index.Add("empty", nil))
	assert.Nil(t, index.graph.Load(), "an empty vector must not fix the dimension at zero")
	require.NoError(t, index.Add("a", []float32{1, 0}))
	assert.Equal(t, 1, index.graph.Load().Len())
}

func TestVectorIndex_ReplaceWithAZeroVectorRemovesTheKey(t *testing.T) {
	t.Parallel()

	index := NewVectorIndex[string](3, vectormaths.Cosine)
	index.Replace("a", []float32{1, 0, 0})

	index.Replace("a", []float32{0, 0, 0})

	assert.Zero(t, index.graph.Load().Len(), "the old vector must not stay indexed once the value has a zero vector")
}

func TestVectorIndex_ConcurrentFirstWriteAndSearch(t *testing.T) {
	t.Parallel()

	index := NewVectorIndex[string](0, vectormaths.Cosine)
	var group sync.WaitGroup
	for writer := range 4 {
		group.Go(func() {
			index.Replace(fmt.Sprintf("key-%d", writer), []float32{1, float32(writer), 0})
		})
		group.Go(func() {
			_, err := index.Search([]float32{1, 0, 0}, 1, nil)
			assert.NoError(t, err)
		})
	}
	group.Wait()

	assert.Equal(t, 4, index.graph.Load().Len())
}

func mustSearchIndex(t *testing.T, index *VectorIndex[string], query []float32, topK int, minScore *float32) []VectorHit[string] {
	t.Helper()

	hits, err := index.Search(query, topK, minScore)
	require.NoError(t, err)
	return hits
}
