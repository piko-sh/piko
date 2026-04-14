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

package cache_domain

import (
	"math"
	"sync"

	"piko.sh/piko/internal/hnsw"
	"piko.sh/piko/internal/vectormaths"
)

// VectorHit represents a single result from a vector similarity search.
type VectorHit[K comparable] struct {
	// Key is the cache key of the matched entry.
	Key K

	// Score is the similarity score (higher is more similar). For cosine
	// similarity this is in [-1, 1]; for euclidean, in [0, 1].
	Score float32
}

// VectorIndex provides approximate nearest-neighbour search using an HNSW
// graph. It wraps hnsw.Graph and converts between distance (lower is better)
// and similarity (higher is better).
//
// Thread-safe for concurrent read/write access (the underlying HNSW graph
// handles its own synchronisation).
type VectorIndex[K comparable] struct {
	// graph is the HNSW graph structure for approximate nearest neighbour search.
	// When dimension is 0 (auto-detect), graph is nil until the first Add call.
	graph *hnsw.Graph[K]

	// metric specifies the distance function used for similarity search.
	metric vectormaths.Metric

	// dimension is the number of elements in each vector. Zero means
	// auto-detect from the first vector added.
	dimension int

	// maxVectors limits the number of vectors in the index. Zero means unlimited.
	maxVectors int

	// mu guards lazy graph initialisation when dimension is 0.
	mu sync.Mutex
}

// SetMaxVectors sets the maximum number of vectors in the index.
// Zero means unlimited.
//
// Takes maxVectors (int) which specifies the vector limit.
func (idx *VectorIndex[K]) SetMaxVectors(maxVectors int) {
	idx.maxVectors = maxVectors
}

// Add indexes a vector for the given key. If the key already exists, its
// vector is replaced.
//
// Takes key (K) which identifies the cache entry.
// Takes vector ([]float32) which is the vector to index.
func (idx *VectorIndex[K]) Add(key K, vector []float32) {
	if len(vector) == 0 {
		return
	}
	idx.initialiseOnce(len(vector))
	if len(vector) != idx.dimension {
		return
	}
	if idx.maxVectors > 0 && idx.graph.Len() >= idx.maxVectors {
		return
	}
	idx.graph.Insert(key, vector)
}

// initialiseOnce lazily initialises the HNSW graph on the first Add call when
// dimension was set to 0 (auto-detect).
//
// Takes dim (int) which specifies the vector dimension for the graph.
//
// Safe for concurrent use. Uses double-checked locking to ensure the graph
// is initialised exactly once.
func (idx *VectorIndex[K]) initialiseOnce(dim int) {
	if idx.graph != nil {
		return
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if idx.graph != nil {
		return
	}
	idx.dimension = dim
	idx.graph = hnsw.New[K](dim, idx.metric)
}

// Remove removes the vector for the given key from the index.
//
// Takes key (K) which identifies the cache entry to remove.
func (idx *VectorIndex[K]) Remove(key K) {
	if idx.graph == nil {
		return
	}
	idx.graph.Delete(key)
}

// Search finds the nearest vectors to the query and returns them as scored
// hits. Results are sorted by score descending (most similar first).
//
// Takes query ([]float32) which is the vector to search for.
// Takes topK (int) which is the maximum number of results.
// Takes minScore (*float32) which filters out results below this threshold.
// Pass nil to disable the threshold.
//
// Returns []VectorHit[K] sorted by score descending.
func (idx *VectorIndex[K]) Search(query []float32, topK int, minScore *float32) []VectorHit[K] {
	if topK <= 0 || idx.graph == nil || len(query) != idx.dimension {
		return nil
	}

	results := idx.graph.Search(query, topK, 0)
	if len(results) == 0 {
		return nil
	}

	hits := make([]VectorHit[K], 0, len(results))
	for _, r := range results {
		score := DistanceToSimilarity(r.Distance, idx.metric)
		if minScore != nil && score < *minScore {
			continue
		}
		hits = append(hits, VectorHit[K]{
			Key:   r.Key,
			Score: score,
		})
	}

	return hits
}

// Clear removes all vectors from the index.
func (idx *VectorIndex[K]) Clear() {
	if idx.graph == nil {
		return
	}
	idx.graph.Clear()
}

// NewVectorIndex creates a new HNSW-backed vector index.
//
// Takes dimension (int) which is the expected vector dimensionality.
// Pass 0 to auto-detect the dimension from the first vector added.
// Takes metric (vectormaths.Metric) which selects the distance function.
//
// Returns *VectorIndex[K] ready for use.
func NewVectorIndex[K comparable](dimension int, metric vectormaths.Metric) *VectorIndex[K] {
	index := &VectorIndex[K]{
		metric:    metric,
		dimension: dimension,
	}
	if dimension > 0 {
		index.graph = hnsw.New[K](dimension, metric)
	}
	return index
}

// DistanceToSimilarity converts HNSW distance values back to similarity scores.
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
func DistanceToSimilarity(distance float32, metric vectormaths.Metric) float32 {
	switch metric {
	case vectormaths.Euclidean:
		return float32(1.0 / (1.0 + math.Sqrt(float64(distance))))
	default:
		return 1 - distance
	}
}
