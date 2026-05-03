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
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/vectormaths"
)

func TestGraph_Insert(t *testing.T) {
	t.Run("single insert", func(t *testing.T) {
		g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))

		g.Insert("a", []float32{1, 0, 0})
		assert.Equal(t, 1, g.Len())
	})

	t.Run("multiple inserts", func(t *testing.T) {
		g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))

		g.Insert("x", []float32{1, 0, 0})
		g.Insert("y", []float32{0, 1, 0})
		g.Insert("z", []float32{0.9, 0.1, 0})
		assert.Equal(t, 3, g.Len())
	})

	t.Run("duplicate key updates vector", func(t *testing.T) {
		g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))

		g.Insert("a", []float32{1, 0, 0})
		g.Insert("a", []float32{0, 1, 0})
		assert.Equal(t, 1, g.Len())

		results := g.Search([]float32{0, 1, 0}, 1, 0)
		require.Len(t, results, 1)
		assert.Equal(t, "a", results[0].Key)
		assert.InDelta(t, 0.0, results[0].Distance, 0.001)
	})
}

func TestGraph_InsertLargeCorpus(t *testing.T) {
	const (
		dim  = 32
		n    = 500
		topK = 10
	)

	g := New[int](dim, vectormaths.Cosine, WithRandomSeed(42), WithMaxNeighboursPerLayer(16), WithConstructionCandidateCount(100))

	randomSource := rand.New(rand.NewPCG(42, 42>>1|1))
	vectors := make([][]float32, n)
	for i := range n {
		vec := make([]float32, dim)
		for j := range dim {
			vec[j] = randomSource.Float32()*2 - 1
		}
		vectors[i] = vec
		g.Insert(i, vec)
	}

	assert.Equal(t, n, g.Len())

	query := vectors[0]
	results := g.Search(query, topK, 100)
	require.Len(t, results, topK)

	assert.Equal(t, 0, results[0].Key, "query vector itself should be the nearest match")
	assert.InDelta(t, 0.0, results[0].Distance, 0.001)

	for i := 1; i < len(results); i++ {
		assert.LessOrEqual(t, results[i-1].Distance, results[i].Distance,
			"results should be sorted by distance")
	}
}

func TestGraph_RecallQuality(t *testing.T) {
	const (
		dim  = 64
		n    = 1000
		topK = 10
	)

	g := New[int](dim, vectormaths.Cosine,
		WithRandomSeed(42),
		WithMaxNeighboursPerLayer(16),
		WithConstructionCandidateCount(200),
	)

	randomSource := rand.New(rand.NewPCG(42, 42>>1|1))
	vectors := make([][]float32, n)
	for i := range n {
		vec := make([]float32, dim)
		for j := range dim {
			vec[j] = randomSource.Float32()*2 - 1
		}
		vectors[i] = vec
		g.Insert(i, vec)
	}

	query := vectors[0]
	hnswResults := g.Search(query, topK, 200)

	bruteForce := bruteForceSearch(vectors, query, topK)

	hnswKeys := make(map[int]struct{})
	for _, r := range hnswResults {
		hnswKeys[r.Key] = struct{}{}
	}

	hits := 0
	for _, k := range bruteForce {
		if _, ok := hnswKeys[k]; ok {
			hits++
		}
	}

	recall := float64(hits) / float64(topK)
	assert.GreaterOrEqual(t, recall, 0.8,
		"recall should be at least 80%% with ef=200, got %.1f%%", recall*100)
}

func TestRandomLevel(t *testing.T) {
	g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))

	levels := make(map[int]int)
	const iterations = 10000
	for range iterations {
		l := g.randomLevel()
		levels[l]++
	}

	assert.Greater(t, levels[0], levels[1],
		"level 0 should be more frequent than level 1")

	totalHigher := 0
	for l, count := range levels {
		if l > 0 {
			totalHigher += count
		}
	}
	assert.Greater(t, totalHigher, 0, "some nodes should be assigned to higher levels")
}

func TestSelectNearest(t *testing.T) {
	tests := []struct {
		name  string
		count int
		limit int
		want  int
	}{
		{name: "fewer than limit", count: 2, limit: 5, want: 2},
		{name: "exactly limit", count: 5, limit: 5, want: 5},
		{name: "more than limit", count: 10, limit: 3, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := make([]priorityQueueItem[string], tt.count)
			for i := range tt.count {
				candidates[i] = priorityQueueItem[string]{
					node:     &node[string]{key: fmt.Sprintf("n%d", i)},
					distance: float32(i),
				}
			}

			result := selectNearest(candidates, tt.limit)
			assert.Len(t, result, tt.want)
		})
	}
}

func TestGraph_InsertBatch(t *testing.T) {
	t.Run("empty slices is no-op", func(t *testing.T) {
		g := New[int](3, vectormaths.Cosine, WithRandomSeed(42))
		g.InsertBatch(nil, nil)
		assert.Equal(t, 0, g.Len())
	})

	t.Run("mismatched lengths is no-op", func(t *testing.T) {
		g := New[int](3, vectormaths.Cosine, WithRandomSeed(42))
		g.InsertBatch([]int{1, 2}, [][]float32{{1, 0, 0}})
		assert.Equal(t, 0, g.Len())
	})

	t.Run("single element", func(t *testing.T) {
		g := New[int](3, vectormaths.Cosine, WithRandomSeed(42))
		g.InsertBatch([]int{1}, [][]float32{{1, 0, 0}})
		assert.Equal(t, 1, g.Len())
	})

	t.Run("multiple elements", func(t *testing.T) {
		g := New[int](3, vectormaths.Cosine, WithRandomSeed(42))
		keys := []int{0, 1, 2, 3, 4}
		vecs := [][]float32{
			{1, 0, 0},
			{0, 1, 0},
			{0, 0, 1},
			{0.7, 0.7, 0},
			{0.5, 0.5, 0.5},
		}
		g.InsertBatch(keys, vecs)
		assert.Equal(t, 5, g.Len())

		results := g.Search([]float32{1, 0, 0}, 2, 0)
		require.Len(t, results, 2)
		assert.Equal(t, 0, results[0].Key)
	})

	t.Run("duplicate keys in batch", func(t *testing.T) {
		g := New[int](3, vectormaths.Cosine, WithRandomSeed(42))
		keys := []int{1, 2, 1}
		vecs := [][]float32{
			{1, 0, 0},
			{0, 1, 0},
			{0, 0, 1},
		}
		g.InsertBatch(keys, vecs)
		assert.Equal(t, 2, g.Len())

		results := g.Search([]float32{0, 0, 1}, 1, 0)
		require.Len(t, results, 1)
		assert.Equal(t, 1, results[0].Key)
	})

	t.Run("equivalent to sequential inserts", func(t *testing.T) {
		const n = 100

		gBatch := New[int](16, vectormaths.Cosine, WithRandomSeed(42))
		gSeq := New[int](16, vectormaths.Cosine, WithRandomSeed(42))

		keys := make([]int, n)
		vecs := make([][]float32, n)
		randomSource := rand.New(rand.NewPCG(99, 99>>1|1))
		for i := range n {
			keys[i] = i
			vec := make([]float32, 16)
			for j := range vec {
				vec[j] = randomSource.Float32()*2 - 1
			}
			vecs[i] = vec
		}

		gBatch.InsertBatch(keys, vecs)
		for i, key := range keys {
			gSeq.Insert(key, vecs[i])
		}

		assert.Equal(t, gSeq.Len(), gBatch.Len())

		query := vecs[0]
		rBatch := gBatch.Search(query, 5, 100)
		rSeq := gSeq.Search(query, 5, 100)

		require.Len(t, rBatch, 5)
		require.Len(t, rSeq, 5)

		for i := range rBatch {
			assert.Equal(t, rSeq[i].Key, rBatch[i].Key,
				"result %d should match sequential insert", i)
		}
	})
}

func TestHNSW_Insert_RejectsWrongDimensionVector(t *testing.T) {
	t.Parallel()

	g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))

	t.Run("longer vector returns sentinel", func(t *testing.T) {
		err := g.Insert("oversize", []float32{1, 0, 0, 1})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrVectorDimensionMismatch)
	})

	t.Run("shorter vector returns sentinel", func(t *testing.T) {
		err := g.Insert("undersize", []float32{1, 0})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrVectorDimensionMismatch)
	})

	t.Run("matching dimension succeeds", func(t *testing.T) {
		require.NoError(t, g.Insert("ok", []float32{1, 0, 0}))
	})

	t.Run("InsertBatch rejects mismatched vector", func(t *testing.T) {
		batchGraph := New[int](3, vectormaths.Cosine, WithRandomSeed(42))
		err := batchGraph.InsertBatch([]int{1, 2}, [][]float32{{1, 0, 0}, {1, 0}})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrVectorDimensionMismatch)
		assert.Equal(t, 0, batchGraph.Len(), "no vectors should be inserted on mismatch")
	})
}

func bruteForceSearch(vectors [][]float32, query []float32, topK int) []int {
	type scored struct {
		index    int
		distance float32
	}

	scores := make([]scored, len(vectors))
	for i, v := range vectors {
		scores[i] = scored{index: i, distance: 1 - vectormaths.CosineSimilarity(query, v)}
	}

	for i := 1; i < len(scores); i++ {
		j := i
		for j > 0 && scores[j].distance < scores[j-1].distance {
			scores[j], scores[j-1] = scores[j-1], scores[j]
			j--
		}
	}

	result := make([]int, topK)
	for i := range topK {
		result[i] = scores[i].index
	}
	return result
}
