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
	"math"
	"sync"
	"sync/atomic"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/hnsw"
	"piko.sh/vectormaths"
)

// VectorHit represents a single result from a vector similarity search.
type VectorHit[K comparable] struct {
	// Key is the cache key of the matched entry.
	Key K

	// Score is the similarity score (higher is more similar). For cosine similarity this is
	// in [-1, 1]; for euclidean, in [0, 1].
	Score float32
}

// VectorIndex provides approximate nearest-neighbour search using an HNSW graph. It wraps
// hnsw.Graph and converts between distance (lower is better) and similarity (higher is
// better).
//
// Thread-safe for concurrent read/write access (the underlying HNSW graph handles its own
// synchronisation).
type VectorIndex[K comparable] struct {
	// graph holds the HNSW graph, or nil until the first vector fixes its dimension. Its
	// atomic pointer lets searches read the graph while a write creates it.
	graph atomic.Pointer[hnsw.Graph[K]]

	// metric specifies the distance function used for similarity search.
	metric vectormaths.Metric

	// maxVectors limits the number of vectors in the index. Zero means unlimited.
	maxVectors int

	// mu guards lazy graph creation when the dimension is auto-detected.
	mu sync.Mutex
}

// Add indexes vector for key, replacing an existing vector. At capacity, Add leaves the
// index unchanged, even for an existing key.
//
// Takes key (K) which identifies the cache entry.
// Takes vector ([]float32) which is the vector to index.
//
// Returns error which wraps hnsw.ErrVectorDimensionMismatch when the vector is empty or
// does not match the index's dimension, or hnsw.ErrZeroVector when the index uses cosine
// similarity and the vector has zero magnitude. An empty vector never fixes the dimension
// of an index that auto-detects it.
func (idx *VectorIndex[K]) Add(key K, vector []float32) error {
	if len(vector) == 0 {
		return fmt.Errorf("indexing vector: %w: got an empty vector", hnsw.ErrVectorDimensionMismatch)
	}
	graph := idx.graphFor(len(vector))
	if idx.maxVectors > 0 && graph.Len() >= idx.maxVectors {
		return nil
	}
	if err := graph.Insert(key, vector); err != nil {
		return fmt.Errorf("indexing vector: %w", err)
	}
	return nil
}

// Replace updates the indexed vector for key or removes an invalid vector.
//
// Empty vectors, dimension mismatches and zero vectors under cosine similarity remove the
// key from vector search. The cached value remains. An existing key can be replaced even
// when the index is full.
//
// Takes key (K) which identifies the cache entry.
// Takes vector ([]float32) which is the vector to index; empty removes the key.
func (idx *VectorIndex[K]) Replace(key K, vector []float32) {
	if len(vector) == 0 {
		idx.Remove(key)
		return
	}
	graph := idx.graphFor(len(vector))
	if idx.maxVectors > 0 && graph.Len() >= idx.maxVectors {
		graph.Delete(key)
		if graph.Len() >= idx.maxVectors {
			return
		}
	}
	if err := graph.Insert(key, vector); err != nil {
		graph.Delete(key)
	}
}

// graphFor returns the HNSW graph, creating it with the given dimension when the index
// auto-detects its dimension and holds no graph yet.
//
// Takes dimension (int) which is the dimension to create the graph with.
//
// Returns *hnsw.Graph[K] which is the index's graph.
//
// Safe for concurrent use. Uses double-checked locking so the graph is created exactly
// once.
func (idx *VectorIndex[K]) graphFor(dimension int) *hnsw.Graph[K] {
	if graph := idx.graph.Load(); graph != nil {
		return graph
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if graph := idx.graph.Load(); graph != nil {
		return graph
	}
	graph := hnsw.New[K](dimension, idx.metric)
	idx.graph.Store(graph)
	return graph
}

// Remove removes the vector for the given key from the index.
//
// Takes key (K) which identifies the cache entry to remove.
func (idx *VectorIndex[K]) Remove(key K) {
	if graph := idx.graph.Load(); graph != nil {
		graph.Delete(key)
	}
}

// Search finds the nearest vectors to the query and returns them as scored hits. Results
// are sorted by score descending (most similar first).
//
// Takes query ([]float32) which is the vector to search for.
// Takes topK (int) which is the maximum number of results.
// Takes minScore (*float32) which filters out results below this threshold. Pass nil to
// disable the threshold.
//
// Returns []VectorHit[K] which is sorted by descending score, or nil when topK is not
// positive or nothing has been indexed yet.
// Returns error which wraps cache_domain.ErrInvalidQueryVector, together with
// hnsw.ErrVectorDimensionMismatch or hnsw.ErrZeroVector, when the query cannot be
// compared with the indexed vectors.
func (idx *VectorIndex[K]) Search(query []float32, topK int, minScore *float32) ([]VectorHit[K], error) {
	graph := idx.graph.Load()
	if topK <= 0 || graph == nil {
		return nil, nil
	}

	results, err := graph.Search(query, topK, 0)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", cache_domain.ErrInvalidQueryVector, err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	hits := make([]VectorHit[K], 0, len(results))
	for _, r := range results {
		score := distanceToSimilarity(r.Distance, idx.metric)
		if minScore != nil && score < *minScore {
			continue
		}
		hits = append(hits, VectorHit[K]{
			Key:   r.Key,
			Score: score,
		})
	}

	return hits, nil
}

// Clear removes all vectors from the index.
func (idx *VectorIndex[K]) Clear() {
	if graph := idx.graph.Load(); graph != nil {
		graph.Clear()
	}
}

// NewVectorIndex creates a new HNSW-backed vector index.
//
// Takes dimension (int) which is the expected vector dimensionality. Pass 0 to
// auto-detect the dimension from the first vector added.
// Takes metric (vectormaths.Metric) which selects the distance function.
//
// Returns *VectorIndex[K] ready for use.
func NewVectorIndex[K comparable](dimension int, metric vectormaths.Metric) *VectorIndex[K] {
	index := &VectorIndex[K]{
		graph:      atomic.Pointer[hnsw.Graph[K]]{},
		metric:     metric,
		maxVectors: 0,
		mu:         sync.Mutex{},
	}
	if dimension > 0 {
		index.graph.Store(hnsw.New[K](dimension, metric))
	}
	return index
}

// distanceToSimilarity converts HNSW distance values back to similarity scores.
//
// The conversion depends on the metric:
//   - Cosine: distance = 1 - similarity, so similarity = 1 - distance
//   - DotProduct: distance = 1 - similarity, so similarity = 1 - distance
//   - Euclidean: distance = squared_euclidean, similarity = 1 / (1 + sqrt(d))
//
// Takes distance (float32) which is the HNSW distance value to convert.
// Takes metric (vectormaths.Metric) which specifies the distance metric used.
//
// Returns float32 which is the similarity score in the range [0, 1].
func distanceToSimilarity(distance float32, metric vectormaths.Metric) float32 {
	switch metric {
	case vectormaths.Euclidean:
		return float32(1.0 / (1.0 + math.Sqrt(float64(distance))))
	default:
		return 1 - distance
	}
}
