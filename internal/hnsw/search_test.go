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

package hnsw

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/vectormaths"
)

func TestGraph_Search(t *testing.T) {
	tests := []struct {
		inserts   map[string][]float32
		name      string
		wantFirst string
		query     []float32
		topK      int
		efSearch  int
		wantLen   int
		wantNil   bool
	}{
		{
			name:    "empty graph",
			inserts: nil,
			query:   []float32{1, 0, 0},
			topK:    5,
			wantNil: true,
		},
		{
			name:    "zero topK",
			inserts: map[string][]float32{"a": {1, 0, 0}},
			query:   []float32{1, 0, 0},
			topK:    0,
			wantNil: true,
		},
		{
			name:      "single element",
			inserts:   map[string][]float32{"a": {1, 0, 0}},
			query:     []float32{1, 0, 0},
			topK:      1,
			wantLen:   1,
			wantFirst: "a",
		},
		{
			name: "nearest of two",
			inserts: map[string][]float32{
				"near": {0.9, 0.1, 0},
				"far":  {0, 1, 0},
			},
			query:     []float32{1, 0, 0},
			topK:      1,
			wantLen:   1,
			wantFirst: "near",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))
			for k, v := range tt.inserts {
				g.Insert(k, v)
			}

			results := g.Search(tt.query, tt.topK, tt.efSearch)

			if tt.wantNil {
				assert.Nil(t, results)
				return
			}

			require.Len(t, results, tt.wantLen)
			if tt.wantFirst != "" {
				assert.Equal(t, tt.wantFirst, results[0].Key)
			}
		})
	}
}

func TestGraph_SearchTopKLimiting(t *testing.T) {
	g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))

	for i := range 10 {
		vec := []float32{float32(i), float32(10 - i), 0}
		g.Insert(fmt.Sprintf("n%d", i), vec)
	}

	results := g.Search([]float32{0, 10, 0}, 3, 0)
	require.Len(t, results, 3)

	for i := 1; i < len(results); i++ {
		assert.LessOrEqual(t, results[i-1].Distance, results[i].Distance,
			"results should be sorted by distance")
	}
}

func TestGraph_SearchWithExactMatch(t *testing.T) {
	g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))

	g.Insert("x", []float32{1, 0, 0})
	g.Insert("y", []float32{0, 1, 0})
	g.Insert("z", []float32{0.9, 0.1, 0})

	results := g.Search([]float32{1, 0, 0}, 2, 0)
	require.Len(t, results, 2)

	assert.Equal(t, "x", results[0].Key)
	assert.InDelta(t, 0.0, results[0].Distance, 0.001)
	assert.Equal(t, "z", results[1].Key)
}

func TestGraph_SearchMetrics(t *testing.T) {
	tests := []struct {
		name    string
		metric  vectormaths.Metric
		inserts map[string][]float32
		query   []float32
		want    []string
	}{
		{
			name:   "euclidean",
			metric: vectormaths.Euclidean,
			inserts: map[string][]float32{
				"origin": {0, 0, 0},
				"near":   {0.1, 0, 0},
				"far":    {10, 10, 10},
			},
			query: []float32{0, 0, 0},
			want:  []string{"origin", "near", "far"},
		},
		{
			name:   "dot product",
			metric: vectormaths.DotProduct,
			inserts: map[string][]float32{
				"high":   {1, 1, 1},
				"medium": {0.5, 0.5, 0.5},
				"low":    {0, 0, 0},
			},
			query: []float32{1, 1, 1},
			want:  []string{"high", "medium"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New[string](3, tt.metric, WithRandomSeed(42))
			for k, v := range tt.inserts {
				g.Insert(k, v)
			}

			results := g.Search(tt.query, len(tt.want), 0)
			require.Len(t, results, len(tt.want))

			for i, wantKey := range tt.want {
				assert.Equal(t, wantKey, results[i].Key)
			}
		})
	}
}

func TestGraph_SearchCustomEfSearch(t *testing.T) {
	g := New[string](3, vectormaths.Cosine, WithRandomSeed(42), WithSearchCandidateCount(10))

	g.Insert("a", []float32{1, 0, 0})
	g.Insert("b", []float32{0.9, 0.1, 0})
	g.Insert("c", []float32{0, 1, 0})

	r1 := g.Search([]float32{1, 0, 0}, 2, 0)
	require.Len(t, r1, 2)

	r2 := g.Search([]float32{1, 0, 0}, 2, 50)
	require.Len(t, r2, 2)

	assert.Equal(t, r1[0].Key, r2[0].Key)
}

func TestEuclideanDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 0,
		},
		{
			name:     "unit distance",
			a:        []float32{0, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "different lengths returns max",
			a:        []float32{1, 0},
			b:        []float32{1, 0, 0},
			expected: float32(math.MaxFloat32),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := euclideanDistance(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}
