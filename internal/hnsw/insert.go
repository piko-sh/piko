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
)

// minRandomValue is the smallest value used when a random draw returns zero,
// to avoid log(0) in the level calculation formula.
const minRandomValue = 1e-10

// Insert adds a vector to the graph with the given key. If the key already
// exists, its vector is updated in place.
//
// Takes key (K) which identifies the vector.
// Takes vector ([]float32) which is the vector data. Must match the graph's
// dimension.
//
// Safe for concurrent use.
func (g *Graph[K]) Insert(key K, vector []float32) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.insertLocked(key, vector)
}

// InsertBatch adds multiple vectors to the graph in a single operation.
// More efficient than calling Insert in a loop because it acquires the write
// lock once and amortises the overhead across all insertions.
//
// Takes keys ([]K) which identifies each vector to insert.
// Takes vectors ([][]float32) which contains the vector data to insert.
//
// Each vector must match the graph's dimension. If lengths differ or either
// slice is empty, this is a no-op.
//
// Safe for concurrent use.
func (g *Graph[K]) InsertBatch(keys []K, vectors [][]float32) {
	if len(keys) != len(vectors) || len(keys) == 0 {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	for i, key := range keys {
		g.insertLocked(key, vectors[i])
	}
}

// insertLocked performs the insert logic without acquiring the lock.
//
// Takes key (K) which identifies the node to insert or update.
// Takes vector ([]float32) which contains the embedding data for the node.
//
// Caller must hold the write lock.
func (g *Graph[K]) insertLocked(key K, vector []float32) {
	if existing, ok := g.nodes[key]; ok {
		existing.vector = vector
		return
	}

	level := g.randomLevel()
	id := g.allocID()
	insertedNode := newNode(key, id, vector, level, g.maxNeighboursPerLayer, g.maxNeighboursBaseLayer)
	g.nodes[key] = insertedNode
	g.nodeByID[id] = insertedNode

	g.ensureLayers(level)

	for i := 0; i <= level; i++ {
		g.layers[i][key] = insertedNode
	}

	if !g.hasEntry {
		g.entry = key
		g.hasEntry = true
		return
	}

	g.connectNode(insertedNode)
}

// randomLevel assigns a random layer using the exponential distribution from
// the HNSW paper: level = floor(-ln(uniform(0,1)) * ml).
//
// Returns int which is the assigned level (0-based).
func (g *Graph[K]) randomLevel() int {
	randomValue := g.randomSource.Float64()
	if randomValue == 0 {
		randomValue = minRandomValue
	}
	level := int(math.Floor(-math.Log(randomValue) * g.levelNormalisationFactor))
	return level
}

// ensureLayers grows the layer slice to fit the given level.
//
// Takes level (int) which is the required maximum layer index.
func (g *Graph[K]) ensureLayers(level int) {
	for len(g.layers) <= level {
		g.layers = append(g.layers, make(map[K]*node[K]))
	}
}

// connectNode wires a newly inserted node into the graph by finding its
// nearest neighbours at each layer and adding bidirectional edges.
//
// Takes target (*node[K]) which is the node to connect.
//
// Caller must hold the write lock.
func (g *Graph[K]) connectNode(target *node[K]) {
	entryNode := g.nodes[g.entry]
	entryLevel := entryNode.level

	current := entryNode

	for layer := entryLevel; layer > target.level; layer-- {
		current = g.greedyClosest(current, target.vector, layer)
	}

	for layer := min(target.level, entryLevel); layer >= 0; layer-- {
		maxNeighbourCount := maxNeighbours(layer, g.maxNeighboursPerLayer, g.maxNeighboursBaseLayer)
		candidates := g.searchLayer(current, target.vector, g.constructionCandidateCount, layer)

		neighbours := selectNearest(candidates, maxNeighbourCount)
		for _, item := range neighbours {
			neighbour := item.node
			target.neighbours[layer] = neighbourAdd(target.neighbours[layer], neighbour)
			neighbour.neighbours[layer] = neighbourAdd(neighbour.neighbours[layer], target)

			g.shrinkNeighbours(neighbour, layer, maxNeighbourCount)
		}

		if len(candidates) > 0 {
			current = candidates[0].node
		}

		g.putPQSlice(candidates)
	}

	if target.level > entryLevel {
		g.entry = target.key
	}
}

// greedyClosest descends greedily from the start node to find the closest
// node to the target vector at the given layer. It follows the best neighbour
// until no improvement is found.
//
// Takes start (*node[K]) which is the starting node.
// Takes target ([]float32) which is the query vector.
// Takes layer (int) which is the layer to traverse.
//
// Returns *node[K] which is the closest node found.
//
// Caller must hold at least a read lock.
func (g *Graph[K]) greedyClosest(start *node[K], target []float32, layer int) *node[K] {
	current := start
	currentDistance := g.distance(current.vector, target)

	for {
		improved := false
		for _, neighbour := range current.neighbours[layer] {
			neighbourDistance := g.distance(neighbour.vector, target)
			if neighbourDistance < currentDistance {
				current = neighbour
				currentDistance = neighbourDistance
				improved = true
			}
		}
		if !improved {
			return current
		}
	}
}

// shrinkNeighbours prunes a node's neighbour list at the given layer if it
// exceeds the limit, keeping only the closest neighbours.
//
// Scores all neighbours, sorts by distance using insertion sort, then keeps
// only the closest maxNeighbourCount. Removes back-references from evicted
// nodes.
//
// Takes target (*node[K]) which is the node to prune.
// Takes layer (int) which is the layer index.
// Takes maxNeighbourCount (int) which is the neighbour limit.
//
// Caller must hold the write lock.
func (g *Graph[K]) shrinkNeighbours(target *node[K], layer, maxNeighbourCount int) {
	neighbours := target.neighbours[layer]
	if len(neighbours) <= maxNeighbourCount {
		return
	}

	type scored struct {
		node     *node[K]
		distance float32
	}

	var buffer [64]scored
	var items []scored
	if len(neighbours) <= len(buffer) {
		items = buffer[:0]
	} else {
		items = make([]scored, 0, len(neighbours))
	}
	for _, neighbour := range neighbours {
		items = append(items, scored{node: neighbour, distance: g.distance(neighbour.vector, target.vector)})
	}

	for i := 1; i < len(items); i++ {
		j := i
		for j > 0 && items[j].distance < items[j-1].distance {
			items[j], items[j-1] = items[j-1], items[j]
			j--
		}
	}

	for i := maxNeighbourCount; i < len(items); i++ {
		evicted := items[i]
		evicted.node.neighbours[layer] = neighbourRemove(evicted.node.neighbours[layer], target)
	}

	for i := range maxNeighbourCount {
		neighbours[i] = items[i].node
	}
	clear(neighbours[maxNeighbourCount:])
	target.neighbours[layer] = neighbours[:maxNeighbourCount]
}

// selectNearest selects up to limit items from the candidate list, keeping
// those closest to the query. The candidates must be sorted by distance with
// the nearest first.
//
// Takes candidates ([]priorityQueueItem[K]) which are the candidate items
// sorted by distance, nearest first.
// Takes limit (int) which is the maximum number of neighbours to return.
//
// Returns []priorityQueueItem[K] containing at most limit nearest items.
func selectNearest[K comparable](candidates []priorityQueueItem[K], limit int) []priorityQueueItem[K] {
	if len(candidates) <= limit {
		return candidates
	}
	return candidates[:limit]
}
