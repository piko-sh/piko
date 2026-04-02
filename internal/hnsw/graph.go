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
	"math"
	"math/rand/v2"
	"sync"

	"piko.sh/piko/internal/vectormaths"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// defaultMaxNeighboursPerLayer is the maximum number of
	// neighbours per node in upper layers.
	defaultMaxNeighboursPerLayer = 16

	// defaultConstructionCandidateCount is the number of
	// candidates to consider when building the graph.
	// Larger values improve search quality but slow down inserts.
	defaultConstructionCandidateCount = 200

	// defaultSearchCandidateCount is the default size of the
	// candidate list used during search.
	defaultSearchCandidateCount = 50
)

// Result holds a single search result with the key and its distance to the
// query vector. Lower distance values indicate more similar vectors.
type Result[K comparable] struct {
	// Key is the identifier of the matched vector.
	Key K

	// Distance is how far from the query vector. Lower values mean more similar.
	Distance float32
}

// Graph is a generic HNSW index mapping keys of type K to float32 vectors.
// It supports concurrent reads and writes.
//
// K must be comparable (used as map keys).
type Graph[K comparable] struct {
	// visitedPool holds reusable []bool slices for the visited set in searchLayer.
	visitedPool sync.Pool

	// pqItemPool holds reusable []priorityQueueItem[K] slices for priority
	// queues.
	pqItemPool sync.Pool

	// entry is the key used to look up this node in the graph.
	entry K

	// randomSource provides random number generation for layout algorithms.
	randomSource *rand.Rand

	// distanceFunction calculates the distance between two vectors.
	distanceFunction func(a, b []float32) float32

	// nodes maps keys to their graph nodes.
	nodes map[K]*node[K]

	// metric is the distance function used for vector comparisons.
	metric vectormaths.Metric

	// freeIDs holds reused IDs from deleted nodes.
	freeIDs []uint32

	// nodeByID maps dense internal IDs to nodes for fast lookup.
	nodeByID []*node[K]

	// layers holds nodes grouped by their depth in the topological order.
	layers []map[K]*node[K]

	// maxNeighboursPerLayer is the maximum number of neighbours per node in
	// upper layers.
	maxNeighboursPerLayer int

	// maxNeighboursBaseLayer is the maximum number of connections at layer 0.
	maxNeighboursBaseLayer int

	// constructionCandidateCount is the build-time setting for the HNSW graph.
	constructionCandidateCount int

	// searchCandidateCount is the number of candidates to check during search.
	searchCandidateCount int

	// levelNormalisationFactor is the multiplier for the level assignment
	// formula: 1/ln(maxNeighboursPerLayer).
	levelNormalisationFactor float64

	// dimension is the number of values in each vector stored in the graph.
	dimension int

	// mu protects the graph data from concurrent access.
	mu sync.RWMutex

	// nextID is the next internal node ID to assign; increases with each new node.
	nextID uint32

	// hasEntry indicates whether an entry node has been added to the graph.
	hasEntry bool
}

// Option configures the HNSW graph at construction time.
type Option func(*graphConfig)

// graphConfig holds settings for building and searching the HNSW graph.
type graphConfig struct {
	// maxNeighboursPerLayer is the number of connections per node; used to
	// calculate maxNeighboursBaseLayer and levelNormalisationFactor.
	maxNeighboursPerLayer int

	// constructionCandidateCount is the number of candidates to
	// consider when building the index.
	constructionCandidateCount int

	// searchCandidateCount is the size of the dynamic candidate
	// list used during search.
	searchCandidateCount int

	// seed is the random number generator seed for reproducible results.
	seed int64

	// hasSeed indicates whether a random seed was set explicitly.
	hasSeed bool
}

// Len returns the number of vectors in the graph.
//
// Returns int which is the current vector count.
//
// Safe for concurrent use.
func (g *Graph[K]) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// Clear removes all vectors from the graph.
//
// Safe for concurrent use.
func (g *Graph[K]) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodes = make(map[K]*node[K])
	g.nodeByID = g.nodeByID[:0]
	g.layers = nil
	g.hasEntry = false
	g.nextID = 0
	g.freeIDs = g.freeIDs[:0]
}

// allocID returns a dense internal ID for a new node. It reuses IDs from
// deleted nodes when available, otherwise increments the counter.
//
// Returns uint32 which is the allocated internal node ID.
//
// Caller must hold the write lock.
func (g *Graph[K]) allocID() uint32 {
	if n := len(g.freeIDs); n > 0 {
		id := g.freeIDs[n-1]
		g.freeIDs = g.freeIDs[:n-1]
		return id
	}
	id := g.nextID
	g.nextID++
	if int(id) >= len(g.nodeByID) {
		g.nodeByID = append(g.nodeByID, nil)
	}
	return id
}

// freeID returns a node ID to the free list for reuse and clears the lookup.
//
// Takes id (uint32) which is the node identifier to release.
//
// Caller must hold the write lock.
func (g *Graph[K]) freeID(id uint32) {
	g.nodeByID[id] = nil
	g.freeIDs = append(g.freeIDs, id)
}

// getVisited returns a []bool slice of at least the given size from the pool,
// or creates a new one. The slice is zeroed and ready for use.
//
// Takes needed (int) which specifies the minimum slice length required.
//
// Returns []bool which is a slice ready for use as a visited marker.
func (g *Graph[K]) getVisited(needed int) []bool {
	if v := g.visitedPool.Get(); v != nil {
		if buffer, ok := v.([]bool); ok && len(buffer) >= needed {
			return buffer
		}
	}
	return make([]bool, needed)
}

// putVisited clears and returns a visited slice to the pool.
//
// Takes buffer ([]bool) which is the visited slice to clear and return.
func (g *Graph[K]) putVisited(buffer []bool) {
	clear(buffer)
	g.visitedPool.Put(buffer)
}

// getPQSlice returns a []priorityQueueItem[K] slice with at least the given
// capacity from the pool, or allocates a new one. The returned slice has
// length 0.
//
// Takes capacity (int) which specifies the minimum slice capacity.
//
// Returns []priorityQueueItem[K] which is a zero-length slice with at least
// the requested capacity.
func (g *Graph[K]) getPQSlice(capacity int) []priorityQueueItem[K] {
	if v := g.pqItemPool.Get(); v != nil {
		if buffer, ok := v.([]priorityQueueItem[K]); ok && cap(buffer) >= capacity {
			return buffer[:0]
		}
	}
	return make([]priorityQueueItem[K], 0, capacity)
}

// putPQSlice clears and returns a PQ item slice to the pool.
//
// Takes buffer ([]priorityQueueItem[K]) which is the slice to clear and
// return.
func (g *Graph[K]) putPQSlice(buffer []priorityQueueItem[K]) {
	clear(buffer[:cap(buffer)])
	g.pqItemPool.Put(buffer[:0])
}

// WithMaxNeighboursPerLayer sets the maximum number of neighbours per node
// in upper layers.
//
// Takes maxNeighboursPerLayer (int) which specifies the maximum neighbours
// per node. Default is 16.
//
// Returns Option which configures the graph with the specified value.
//
// The base layer (layer 0) uses 2 * maxNeighboursPerLayer.
func WithMaxNeighboursPerLayer(maxNeighboursPerLayer int) Option {
	return func(config *graphConfig) {
		if maxNeighboursPerLayer > 0 {
			config.maxNeighboursPerLayer = maxNeighboursPerLayer
		}
	}
}

// WithConstructionCandidateCount sets the construction-time candidate list
// size.
//
// Takes candidateCount (int) which specifies the candidate list size.
//
// Returns Option which configures the graph with the specified value.
//
// Larger values improve index quality at the cost of slower inserts.
// Default is 200.
func WithConstructionCandidateCount(candidateCount int) Option {
	return func(config *graphConfig) {
		if candidateCount > 0 {
			config.constructionCandidateCount = candidateCount
		}
	}
}

// WithSearchCandidateCount sets the default search-time candidate list size.
//
// Takes candidateCount (int) which specifies the candidate list size for
// search operations.
//
// Returns Option which configures the graph with the specified value.
//
// Can be overridden per query via the candidateCount parameter in Search.
// Default is 50.
func WithSearchCandidateCount(candidateCount int) Option {
	return func(config *graphConfig) {
		if candidateCount > 0 {
			config.searchCandidateCount = candidateCount
		}
	}
}

// WithRandomSeed sets a fixed seed for layer assignment, ensuring
// deterministic results across test runs.
//
// Takes seed (int64) which specifies the random seed value.
//
// Returns Option which sets the graph to use the given seed.
func WithRandomSeed(seed int64) Option {
	return func(c *graphConfig) {
		c.seed = seed
		c.hasSeed = true
	}
}

// New creates a new HNSW graph for vectors of the given dimension using the
// specified distance metric.
//
// Takes dimension (int) which specifies the vector dimensionality.
// Takes metric (vectormaths.Metric) which selects the distance function.
// Takes opts (...Option) which configure graph parameters.
//
// Returns *Graph[K] ready for use.
func New[K comparable](dimension int, metric vectormaths.Metric, opts ...Option) *Graph[K] {
	config := &graphConfig{
		maxNeighboursPerLayer:      defaultMaxNeighboursPerLayer,
		constructionCandidateCount: defaultConstructionCandidateCount,
		searchCandidateCount:       defaultSearchCandidateCount,
	}
	for _, opt := range opts {
		opt(config)
	}

	var randomSource *rand.Rand //nolint:gosec // HNSW level assignment, not security
	if config.hasSeed {
		s := safeconv.Int64ToUint64(config.seed)
		randomSource = rand.New(rand.NewPCG(s, s>>1|1)) //nolint:gosec // HNSW level assignment, not security
	} else {
		randomSource = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())) //nolint:gosec // HNSW level assignment, not security
	}

	g := &Graph[K]{
		nodes:                      make(map[K]*node[K]),
		layers:                     nil,
		metric:                     metric,
		dimension:                  dimension,
		maxNeighboursPerLayer:      config.maxNeighboursPerLayer,
		maxNeighboursBaseLayer:     config.maxNeighboursPerLayer * 2,
		constructionCandidateCount: config.constructionCandidateCount,
		searchCandidateCount:       config.searchCandidateCount,
		levelNormalisationFactor:   1.0 / math.Log(float64(config.maxNeighboursPerLayer)),
		randomSource:               randomSource,
	}

	switch metric {
	case vectormaths.Euclidean:
		g.distanceFunction = euclideanDistance
	case vectormaths.DotProduct:
		g.distanceFunction = func(a, b []float32) float32 {
			return 1 - vectormaths.DotProductSimilarity(a, b)
		}
	default:
		g.distanceFunction = func(a, b []float32) float32 {
			return 1 - vectormaths.CosineSimilarity(a, b)
		}
	}

	return g
}
