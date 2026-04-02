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

	"piko.sh/piko/internal/vectormaths"
)

// Search finds the topK nearest neighbours to the query vector.
//
// Takes query ([]float32) which is the vector to search for.
// Takes topK (int) which is the maximum number of results to return.
// Takes efSearch (int) which is the search-time candidate list size. A value
// of 0 uses the graph's default.
//
// Returns []Result[K] sorted by distance (nearest first). Returns nil for
// an empty graph or zero topK.
//
// Safe for concurrent use.
func (g *Graph[K]) Search(query []float32, topK, efSearch int) []Result[K] {
	if topK <= 0 {
		return nil
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.hasEntry {
		return nil
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

	if len(candidates) > topK {
		candidates = candidates[:topK]
	}

	results := make([]Result[K], len(candidates))
	for i, candidate := range candidates {
		results[i] = Result[K]{
			Key:      candidate.node.key,
			Distance: candidate.distance,
		}
	}

	g.putPQSlice(candidates)

	return results
}

// searchLayer performs a beam search at the given layer, returning the closest
// candidateCount nodes to the target vector. Results are sorted nearest-first.
//
// Takes entry (*node[K]) which is the starting node for the search.
// Takes target ([]float32) which is the query vector.
// Takes candidateCount (int) which is the candidate list size (beam width).
// Takes layer (int) which is the layer to search.
//
// Returns []priorityQueueItem[K] sorted by distance (nearest first),
// containing at most candidateCount items. Each item's distance field holds
// the pre-computed distance to target.
//
// Caller must hold at least a read lock.
func (g *Graph[K]) searchLayer(entry *node[K], target []float32, candidateCount, layer int) []priorityQueueItem[K] {
	needed := int(g.nextID)
	visited := g.getVisited(needed)

	visited[entry.id] = true

	entryDistance := g.distance(entry.vector, target)
	entryItem := priorityQueueItem[K]{node: entry, distance: entryDistance}

	candidateItems := g.getPQSlice(candidateCount)
	resultItems := g.getPQSlice(candidateCount)

	candidates := priorityQueue[K]{items: candidateItems}
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

// distance computes the distance between two vectors using the graph's metric.
// For similarity metrics (cosine, dot product), the distance is 1 - similarity
// so that lower values mean closer vectors.
//
// Takes a ([]float32) which is the first vector.
// Takes b ([]float32) which is the second vector.
//
// Returns float32 which is the distance value (lower means more similar).
func (g *Graph[K]) distance(a, b []float32) float32 {
	return g.distanceFunction(a, b)
}

// euclideanDistance computes the squared Euclidean distance between two
// vectors. Using squared distance avoids the sqrt call while keeping the
// correct ordering for comparisons.
//
// Takes a ([]float32) which is the first vector.
// Takes b ([]float32) which is the second vector.
//
// Returns float32 which is the squared distance, or MaxFloat32 if the vectors
// have different lengths.
func euclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.MaxFloat32)
	}
	return vectormaths.EuclidSqF32(a, b)
}
