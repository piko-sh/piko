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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/vectormaths"
)

func TestVectorIndex_AddAndSearch(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	index.Add("a", []float32{1, 0, 0})
	index.Add("b", []float32{0, 1, 0})
	index.Add("c", []float32{0.9, 0.1, 0})

	hits := index.Search([]float32{1, 0, 0}, 2, nil)
	require.Len(t, hits, 2)

	assert.Equal(t, "a", hits[0].Key)
	assert.InDelta(t, 1.0, hits[0].Score, 0.01)
}

func TestVectorIndex_Remove(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	index.Add("a", []float32{1, 0, 0})
	index.Add("b", []float32{0, 1, 0})

	index.Remove("a")

	hits := index.Search([]float32{1, 0, 0}, 5, nil)
	for _, h := range hits {
		assert.NotEqual(t, "a", h.Key)
	}
}

func TestVectorIndex_Clear(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	index.Add("a", []float32{1, 0, 0})
	index.Add("b", []float32{0, 1, 0})

	index.Clear()

	hits := index.Search([]float32{1, 0, 0}, 5, nil)
	assert.Nil(t, hits)
}

func TestVectorIndex_MinScore(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	index.Add("exact", []float32{1, 0, 0})
	index.Add("orthogonal", []float32{0, 1, 0})

	hits := index.Search([]float32{1, 0, 0}, 10, new(float32(0.9)))

	require.Len(t, hits, 1)
	assert.Equal(t, "exact", hits[0].Key)
}

func TestVectorIndex_DimensionMismatch(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	index.Add("wrong", []float32{1, 0})
	hits := index.Search([]float32{1, 0, 0}, 1, nil)
	assert.Nil(t, hits)

	index.Add("ok", []float32{1, 0, 0})
	hits = index.Search([]float32{1, 0}, 1, nil)
	assert.Nil(t, hits)
}

func TestVectorIndex_EuclideanMetric(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Euclidean)

	index.Add("origin", []float32{0, 0, 0})
	index.Add("near", []float32{0.1, 0, 0})
	index.Add("far", []float32{10, 10, 10})

	hits := index.Search([]float32{0, 0, 0}, 3, nil)
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

	hits := index.Search([]float32{1, 1, 1}, 3, nil)
	require.Len(t, hits, 3)

	assert.Equal(t, "high", hits[0].Key)
	assert.Equal(t, "medium", hits[1].Key)
}

func TestVectorIndex_EmptySearch(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	hits := index.Search([]float32{1, 0, 0}, 5, nil)
	assert.Nil(t, hits)
}

func TestVectorIndex_ZeroTopK(t *testing.T) {
	index := NewVectorIndex[string](3, vectormaths.Cosine)

	index.Add("a", []float32{1, 0, 0})

	hits := index.Search([]float32{1, 0, 0}, 0, nil)
	assert.Nil(t, hits)
}

func TestVectorIndex_AutoDetectDimension(t *testing.T) {
	index := NewVectorIndex[string](0, vectormaths.Cosine)

	hits := index.Search([]float32{1, 0, 0}, 5, nil)
	assert.Nil(t, hits)

	index.Add("a", []float32{1, 0, 0})
	index.Add("b", []float32{0, 1, 0})

	hits = index.Search([]float32{1, 0, 0}, 2, nil)
	require.Len(t, hits, 2)
	assert.Equal(t, "a", hits[0].Key)
	assert.InDelta(t, 1.0, hits[0].Score, 0.01)

	index.Add("wrong", []float32{1, 0})
	hits = index.Search([]float32{1, 0, 0}, 10, nil)
	require.Len(t, hits, 2, "wrong-dimension vector should not be indexed")
}

func TestVectorIndex_AutoDetect_RemoveAndClear(t *testing.T) {
	index := NewVectorIndex[string](0, vectormaths.Cosine)

	index.Remove("nonexistent")
	index.Clear()

	index.Add("a", []float32{1, 0, 0})
	index.Remove("a")
	hits := index.Search([]float32{1, 0, 0}, 5, nil)
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
