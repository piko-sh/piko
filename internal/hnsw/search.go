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
	"cmp"
	"fmt"
	"math"
	"slices"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/vectormaths"
)

var (
	// log is the package-level logger for the HNSW index.
	log = logger_domain.GetLogger("piko/internal/hnsw")
)

// rankedNode is a node's key and dense ID with its distance to a query, held while an
// exhaustive search ranks the graph.
type rankedNode[K comparable] struct {
	// key identifies the node.
	key K

	// id is the node's dense ID, which breaks distance ties.
	id uint32

	// distance is the node's distance to the query.
	distance float32
}

// Search finds the topK nearest neighbours to the query vector.
//
// Calls can run concurrently with other graph operations.
//
// Takes query ([]float32) which is the vector to search for.
// Takes topK (int) which is the maximum number of results to return.
// Takes efSearch (int) which is the search-time candidate list size. A value of 0 uses
// the graph's default.
//
// Returns []Result[K] which is sorted by distance, nearest first, or nil for an empty
// graph or zero or negative topK after a valid query.
// Returns error which wraps ErrVectorDimensionMismatch, ErrZeroVector or
// ErrNonFiniteVector when the query fails validation, even if the graph is empty or topK
// is zero or negative.
func (g *Graph[K]) Search(query []float32, topK, efSearch int) ([]Result[K], error) {
	if err := g.validateVector(query); err != nil {
		return nil, fmt.Errorf("checking query vector: %w", err)
	}
	if topK <= 0 {
		return nil, nil
	}

	results, reachable, nodeCount := g.searchLocked(query, topK, efSearch)
	if reachable >= 0 {
		log.Debug("HNSW search fell back to an exhaustive scan",
			logger_domain.Int("reachable", reachable),
			logger_domain.Int("top_k", topK),
			logger_domain.Int("nodes", nodeCount))
	}
	return results, nil
}

// searchLocked runs a search under the read lock.
//
// Takes query ([]float32) which is the validated query vector.
// Takes topK (int) which is the positive number of results to return.
// Takes efSearch (int) which is the search-time candidate list size, or 0 for the graph's
// default.
//
// Returns []Result[K] which is sorted by distance, nearest first.
// Returns reachable (int) which is the number of candidates the beam search reached when
// it found too few and the search ranked every node instead, or -1 otherwise.
// Returns nodeCount (int) which is the number of nodes in the graph.
//
// Concurrency: Takes the read lock.
func (g *Graph[K]) searchLocked(query []float32, topK, efSearch int) (results []Result[K], reachable, nodeCount int) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nodeCount = len(g.nodes)
	if nodeCount == 0 {
		return nil, -1, 0
	}
	if topK >= nodeCount {
		return g.searchAll(query, nodeCount), -1, nodeCount
	}

	if efSearch <= 0 {
		efSearch = g.searchCandidateCount
	}

	if efSearch < topK {
		efSearch = topK
	}

	entryNode := g.nodes[g.entry]
	topLevel := len(g.layers) - 1

	current := entryNode
	for layer := topLevel; layer > 0; layer-- {
		current = g.greedyClosest(current, query, layer)
	}

	candidates := g.searchLayer(current, query, efSearch, 0)
	if len(candidates) < topK {
		reachable = len(candidates)
		g.putPQSlice(candidates)
		return g.searchAll(query, topK), reachable, nodeCount
	}

	if len(candidates) > topK {
		candidates = candidates[:topK]
	}

	results = make([]Result[K], len(candidates))
	for i, candidate := range candidates {
		results[i] = Result[K]{
			Key:      candidate.node.key,
			Distance: candidate.distance,
		}
	}

	g.putPQSlice(candidates)

	return results, -1, nodeCount
}

// searchAll ranks every live node, including nodes unreachable from the entry point, and
// keeps the nearest limit of them in a bounded heap.
//
// Callers must hold the graph lock.
//
// Takes query ([]float32) which is the validated query vector.
// Takes limit (int) which is the positive number of results to keep.
//
// Returns []Result[K] which is ordered by distance, with node IDs breaking ties.
func (g *Graph[K]) searchAll(query []float32, limit int) []Result[K] {
	limit = min(limit, len(g.nodes))
	nearest := make([]rankedNode[K], 0, limit)
	for key, node := range g.nodes {
		candidate := rankedNode[K]{key: key, id: node.id, distance: g.distance(node.vector, query)}
		if len(nearest) < limit {
			nearest = append(nearest, candidate)
			siftUpFarthest(nearest, len(nearest)-1)
			continue
		}
		if compareRanked(candidate, nearest[0]) < 0 {
			nearest[0] = candidate
			siftDownFarthest(nearest, 0)
		}
	}

	slices.SortFunc(nearest, compareRanked[K])
	results := make([]Result[K], len(nearest))
	for index, ranked := range nearest {
		results[index] = Result[K]{Key: ranked.key, Distance: ranked.distance}
	}
	return results
}

// searchLayer performs a beam search at the given layer, returning the closest
// candidateCount nodes to the target vector.
//
// Callers must hold at least a read lock.
//
// Takes entry (*node[K]) which is the starting node for the search.
// Takes target ([]float32) which is the query vector.
// Takes candidateCount (int) which is the candidate list size (beam width).
// Takes layer (int) which is the layer to search.
//
// Returns []priorityQueueItem[K] sorted by distance (nearest first), containing at most
// candidateCount items. Each item's distance field holds the pre-computed distance to
// target.
func (g *Graph[K]) searchLayer(entry *node[K], target []float32, candidateCount, layer int) []priorityQueueItem[K] {
	needed := int(g.nextID)
	visited := g.getVisited(needed)

	visited[entry.id] = true

	entryDistance := g.distance(entry.vector, target)
	entryItem := priorityQueueItem[K]{node: entry, distance: entryDistance}

	candidateItems := g.getPQSlice(candidateCount)
	resultItems := g.getPQSlice(candidateCount)

	candidates := priorityQueue[K]{items: candidateItems, max: false}
	candidates.push(entryItem)

	results := priorityQueue[K]{items: resultItems, max: true}
	results.push(entryItem)

	for candidates.len() > 0 {
		nearest := candidates.pop()
		farthestResult := results.peek()

		if nearest.distance > farthestResult.distance && results.len() >= candidateCount {
			break
		}

		for _, neighbour := range nearest.node.neighbours[layer] {
			if visited[neighbour.id] {
				continue
			}
			visited[neighbour.id] = true

			neighbourDistance := g.distance(neighbour.vector, target)
			neighbourItem := priorityQueueItem[K]{node: neighbour, distance: neighbourDistance}

			if results.len() < candidateCount {
				results.push(neighbourItem)
				candidates.push(neighbourItem)
				continue
			}

			if neighbourDistance < results.peek().distance {
				results.replacePeek(neighbourItem)
				candidates.push(neighbourItem)
			}
		}
	}

	results.heapSort()
	sorted := results.items

	g.putPQSlice(candidates.items)
	g.putVisited(visited)

	return sorted
}

// distance computes the distance between two validated vectors.
//
// Cosine and dot product use 1 - similarity so smaller distances rank first. Validation
// requires finite vectors with the graph's dimension and nonzero magnitude for cosine.
//
// Takes a ([]float32) which is the first vector.
// Takes b ([]float32) which is the second vector.
//
// Returns float32 which is the distance value (lower means more similar).
func (g *Graph[K]) distance(a, b []float32) float32 {
	return g.distanceFunction(a, b)
}

// euclideanDistance computes the squared Euclidean distance between two vectors. Using
// squared distance avoids the square root while keeping the ordering for comparisons.
//
// Takes a ([]float32) which is the first vector.
// Takes b ([]float32) which is the second vector.
//
// Returns float32 which is the squared distance, or MaxFloat32 if the vectors have
// different lengths. Validation makes that case unreachable, but the check is kept
// because EuclidSqF32 would return 0 for it, which would rank the vector as the closest.
func euclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.MaxFloat32)
	}
	return vectormaths.EuclidSqF32(a, b)
}

// compareRanked orders ranked nodes nearest first, breaking distance ties by node ID.
//
// Takes a (rankedNode[K]) which is the first node.
// Takes b (rankedNode[K]) which is the second node.
//
// Returns int which is negative when a ranks before b, positive when after, and zero when
// they are the same node.
func compareRanked[K comparable](a, b rankedNode[K]) int {
	return cmp.Or(cmp.Compare(a.distance, b.distance), cmp.Compare(a.id, b.id))
}

// siftUpFarthest restores the max-heap order, with the farthest node at the root, after
// the node at index was added.
//
// Takes heap ([]rankedNode[K]) which is the heap.
// Takes index (int) which is the position of the added node.
func siftUpFarthest[K comparable](heap []rankedNode[K], index int) {
	for index > 0 {
		parent := (index - 1) / 2
		if compareRanked(heap[index], heap[parent]) <= 0 {
			return
		}
		heap[index], heap[parent] = heap[parent], heap[index]
		index = parent
	}
}

// siftDownFarthest restores the max-heap order, with the farthest node at the root, after
// the node at index was replaced.
//
// Takes heap ([]rankedNode[K]) which is the heap.
// Takes index (int) which is the position of the replaced node.
func siftDownFarthest[K comparable](heap []rankedNode[K], index int) {
	for {
		farthest := index
		for _, child := range [2]int{2*index + 1, 2*index + 2} {
			if child < len(heap) && compareRanked(heap[child], heap[farthest]) > 0 {
				farthest = child
			}
		}
		if farthest == index {
			return
		}
		heap[index], heap[farthest] = heap[farthest], heap[index]
		index = farthest
	}
}
