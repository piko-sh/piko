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

//go:build bench

package hnsw

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"

	"piko.sh/piko/internal/vectormaths"
)

func benchRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewPCG(uint64(seed), uint64(seed>>1|1)))
}

func randomVector(randomSource *rand.Rand, dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = randomSource.Float32()*2 - 1
	}
	return v
}

func normalise(v []float32) []float32 {
	var norm float64
	for _, value := range v {
		norm += float64(value) * float64(value)
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return v
	}
	out := make([]float32, len(v))
	for i, value := range v {
		out[i] = float32(float64(value) / norm)
	}
	return out
}

func seedGraph(b *testing.B, g *Graph[int], n, dim int) [][]float32 {
	b.Helper()
	randomSource := benchRNG(42)
	vectors := make([][]float32, n)
	for i := range n {
		vectors[i] = normalise(randomVector(randomSource, dim))
		g.Insert(i, vectors[i])
	}
	return vectors
}

func newSeededGraph(b *testing.B, n, dim int) (*Graph[int], [][]float32) {
	b.Helper()
	g := New[int](dim, vectormaths.Cosine,
		WithRandomSeed(42),
		WithMaxNeighboursPerLayer(16),
		WithConstructionCandidateCount(100),
	)
	vecs := seedGraph(b, g, n, dim)
	return g, vecs
}

func BenchmarkInsert(b *testing.B) {
	cases := []struct {
		name      string
		existing  int
		dimension int
	}{
		{name: "empty/dim=128", existing: 0, dimension: 128},
		{name: "100/dim=128", existing: 100, dimension: 128},
		{name: "1K/dim=128", existing: 1_000, dimension: 128},
		{name: "10K/dim=128", existing: 10_000, dimension: 128},
		{name: "1K/dim=768", existing: 1_000, dimension: 768},
		{name: "1K/dim=1536", existing: 1_000, dimension: 1536},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			g := New[int](tc.dimension, vectormaths.Cosine,
				WithRandomSeed(42),
				WithMaxNeighboursPerLayer(16),
				WithConstructionCandidateCount(100),
			)

			randomSource := benchRNG(42)
			for i := range tc.existing {
				g.Insert(i, normalise(randomVector(randomSource, tc.dimension)))
			}

			insertRNG := benchRNG(99)
			vectors := make([][]float32, 1000)
			for i := range vectors {
				vectors[i] = normalise(randomVector(insertRNG, tc.dimension))
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; b.Loop(); i++ {
				g.Insert(tc.existing+i, vectors[i%len(vectors)])
			}
		})
	}
}

func BenchmarkInsertDuplicate(b *testing.B) {
	const dim = 128
	g, vecs := newSeededGraph(b, 1_000, dim)

	randomSource := benchRNG(99)
	newVec := normalise(randomVector(randomSource, dim))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {

		g.Insert(i%len(vecs), newVec)
	}
}

func BenchmarkInsertBatch(b *testing.B) {
	cases := []struct {
		name      string
		batchSize int
		dimension int
	}{
		{name: "10/dim=128", batchSize: 10, dimension: 128},
		{name: "100/dim=128", batchSize: 100, dimension: 128},
		{name: "1K/dim=128", batchSize: 1_000, dimension: 128},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			randomSource := benchRNG(99)
			keys := make([]int, tc.batchSize)
			vecs := make([][]float32, tc.batchSize)
			for i := range tc.batchSize {
				keys[i] = i
				vecs[i] = normalise(randomVector(randomSource, tc.dimension))
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				g := New[int](tc.dimension, vectormaths.Cosine,
					WithRandomSeed(42),
					WithMaxNeighboursPerLayer(16),
					WithConstructionCandidateCount(100),
				)
				g.InsertBatch(keys, vecs)
			}
		})
	}
}

func BenchmarkSearch(b *testing.B) {
	cases := []struct {
		name      string
		n         int
		dimension int
		topK      int
		efSearch  int
	}{
		{name: "100/dim=128/topK=10", n: 100, dimension: 128, topK: 10, efSearch: 50},
		{name: "1K/dim=128/topK=10", n: 1_000, dimension: 128, topK: 10, efSearch: 50},
		{name: "10K/dim=128/topK=10", n: 10_000, dimension: 128, topK: 10, efSearch: 50},
		{name: "10K/dim=128/topK=50", n: 10_000, dimension: 128, topK: 50, efSearch: 100},
		{name: "10K/dim=768/topK=10", n: 10_000, dimension: 768, topK: 10, efSearch: 50},
		{name: "10K/dim=1536/topK=10", n: 10_000, dimension: 1536, topK: 10, efSearch: 50},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			g, _ := newSeededGraph(b, tc.n, tc.dimension)

			randomSource := benchRNG(99)
			query := normalise(randomVector(randomSource, tc.dimension))

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = g.Search(query, tc.topK, tc.efSearch)
			}
		})
	}
}

func BenchmarkSearchEfScaling(b *testing.B) {
	const (
		n   = 10_000
		dim = 128
	)
	g, _ := newSeededGraph(b, n, dim)

	randomSource := benchRNG(99)
	query := normalise(randomVector(randomSource, dim))

	efValues := []int{10, 50, 100, 200, 400}

	for _, ef := range efValues {
		b.Run(fmt.Sprintf("ef=%d", ef), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = g.Search(query, 10, ef)
			}
		})
	}
}

func BenchmarkSearchParallel(b *testing.B) {
	const (
		n   = 10_000
		dim = 128
	)
	g, _ := newSeededGraph(b, n, dim)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		randomSource := benchRNG(rand.Int64())
		for pb.Next() {
			query := normalise(randomVector(randomSource, dim))
			_ = g.Search(query, 10, 50)
		}
	})
}

func BenchmarkDelete(b *testing.B) {
	cases := []struct {
		name string
		n    int
		dim  int
	}{
		{name: "100/dim=128", n: 100, dim: 128},
		{name: "1K/dim=128", n: 1_000, dim: 128},
		{name: "10K/dim=128", n: 10_000, dim: 128},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				b.StopTimer()
				g, _ := newSeededGraph(b, tc.n, tc.dim)
				b.StartTimer()

				for i := 0; i < tc.n/2; i++ {
					g.Delete(i)
				}
			}
		})
	}
}

func BenchmarkDeleteSingle(b *testing.B) {
	const (
		n   = 10_000
		dim = 128
	)

	g, _ := newSeededGraph(b, n, dim)

	randomSource := benchRNG(99)
	extraBase := n
	extraVecs := make([][]float32, 10_000)
	for i := range extraVecs {
		extraVecs[i] = normalise(randomVector(randomSource, dim))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		key := extraBase + i
		g.Insert(key, extraVecs[i%len(extraVecs)])
		g.Delete(key)
	}
}

func BenchmarkDistance(b *testing.B) {
	metrics := []struct {
		name   string
		metric vectormaths.Metric
	}{
		{name: "cosine", metric: vectormaths.Cosine},
		{name: "euclidean", metric: vectormaths.Euclidean},
		{name: "dot_product", metric: vectormaths.DotProduct},
	}

	dims := []int{128, 768, 1536}

	for _, m := range metrics {
		for _, dim := range dims {
			b.Run(fmt.Sprintf("%s/dim=%d", m.name, dim), func(b *testing.B) {
				g := New[int](dim, m.metric, WithRandomSeed(42))

				randomSource := benchRNG(42)
				a := normalise(randomVector(randomSource, dim))
				query := normalise(randomVector(randomSource, dim))

				b.ResetTimer()
				b.ReportAllocs()

				for b.Loop() {
					_ = g.distance(a, query)
				}
			})
		}
	}
}

func BenchmarkPriorityQueue_PushPop(b *testing.B) {
	sizes := []int{50, 200, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			randomSource := benchRNG(42)
			items := make([]priorityQueueItem[int], size)
			for i := range items {
				items[i] = priorityQueueItem[int]{
					node:     &node[int]{key: i, vector: randomVector(randomSource, 3)},
					distance: randomSource.Float32(),
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				priorityQueue := &priorityQueue[int]{
					items: make([]priorityQueueItem[int], 0, size),
				}
				for _, item := range items {
					priorityQueue.push(item)
				}
				for priorityQueue.len() > 0 {
					priorityQueue.pop()
				}
			}
		})
	}
}

func BenchmarkPriorityQueue_ReplacePeek(b *testing.B) {
	const size = 200
	randomSource := benchRNG(42)

	priorityQueue := &priorityQueue[int]{
		items: make([]priorityQueueItem[int], 0, size),
	}

	for range size {
		priorityQueue.push(priorityQueueItem[int]{
			node:     &node[int]{key: 0, vector: randomVector(randomSource, 3)},
			distance: randomSource.Float32(),
		})
	}

	replacements := make([]priorityQueueItem[int], 1000)
	for i := range replacements {
		replacements[i] = priorityQueueItem[int]{
			node:     &node[int]{key: 0, vector: randomVector(randomSource, 3)},
			distance: randomSource.Float32(),
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		priorityQueue.replacePeek(replacements[i%len(replacements)])
	}
}

func BenchmarkSearchLayer(b *testing.B) {
	cases := []struct {
		name string
		n    int
		dim  int
		ef   int
	}{
		{name: "1K/dim=128/ef=50", n: 1_000, dim: 128, ef: 50},
		{name: "10K/dim=128/ef=50", n: 10_000, dim: 128, ef: 50},
		{name: "10K/dim=128/ef=200", n: 10_000, dim: 128, ef: 200},
		{name: "10K/dim=768/ef=50", n: 10_000, dim: 768, ef: 50},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			g, _ := newSeededGraph(b, tc.n, tc.dim)

			randomSource := benchRNG(99)
			query := normalise(randomVector(randomSource, tc.dim))

			g.mu.RLock()
			entryNode := g.nodes[g.entry]
			g.mu.RUnlock()

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				g.mu.RLock()
				_ = g.searchLayer(entryNode, query, tc.ef, 0)
				g.mu.RUnlock()
			}
		})
	}
}

func BenchmarkMixedInsertSearch(b *testing.B) {
	const (
		n   = 5_000
		dim = 128
	)
	g, _ := newSeededGraph(b, n, dim)

	randomSource := benchRNG(99)
	vectors := make([][]float32, 1000)
	for i := range vectors {
		vectors[i] = normalise(randomVector(randomSource, dim))
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		localRandomSource := benchRNG(rand.Int64())
		i := 0
		for pb.Next() {
			if i%5 == 0 {
				g.Insert(n+i, vectors[i%len(vectors)])
			} else {
				query := normalise(randomVector(localRandomSource, dim))
				_ = g.Search(query, 10, 50)
			}
			i++
		}
	})
}

func BenchmarkMixedInsertDeleteSearch(b *testing.B) {
	const (
		n   = 5_000
		dim = 128
	)
	g, _ := newSeededGraph(b, n, dim)

	randomSource := benchRNG(99)
	vectors := make([][]float32, 1000)
	for i := range vectors {
		vectors[i] = normalise(randomVector(randomSource, dim))
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		localRandomSource := benchRNG(rand.Int64())
		i := 0
		for pb.Next() {
			switch i % 10 {
			case 0:
				key := n + i
				g.Insert(key, vectors[i%len(vectors)])
			case 1:
				g.Delete(n + i - 1)
			default:
				query := normalise(randomVector(localRandomSource, dim))
				_ = g.Search(query, 10, 50)
			}
			i++
		}
	})
}
