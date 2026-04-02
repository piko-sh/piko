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

// Delete removes the vector with the given key from the graph. If the key
// does not exist, this is a no-op.
//
// After deletion, neighbours of the removed node are repaired to maintain
// graph connectivity.
//
// Takes key (K) which identifies the vector to remove.
//
// Safe for concurrent use.
func (g *Graph[K]) Delete(key K) {
	g.mu.Lock()
	defer g.mu.Unlock()

	target, ok := g.nodes[key]
	if !ok {
		return
	}

	g.disconnectNode(target)
	g.removeFromLayers(target)
	g.freeID(target.id)
	delete(g.nodes, key)

	if g.entry == key {
		g.electNewEntry()
	}

	g.trimEmptyLayers()
}

// disconnectNode removes all edges to and from the node, then repairs each
// former neighbour's links by trying to add edges from the neighbours of
// neighbours.
//
// Takes target (*node[K]) which is the node being deleted.
//
// Caller must hold the write lock.
func (g *Graph[K]) disconnectNode(target *node[K]) {
	for layer := 0; layer <= target.level; layer++ {
		neighbours := target.neighbours[layer]

		for _, neighbour := range neighbours {
			neighbour.neighbours[layer] = neighbourRemove(neighbour.neighbours[layer], target)
		}

		g.repairNeighbours(neighbours, target, layer)
	}
}

// repairNeighbours attempts to reconnect former neighbours of a deleted node
// by finding replacement edges from the neighbours-of-neighbours pool.
//
// Takes neighbours ([]*node[K]) which are the former neighbours.
// Takes deleted (*node[K]) which is the node being removed.
// Takes layer (int) which is the layer being repaired.
//
// Caller must hold the write lock.
func (g *Graph[K]) repairNeighbours(neighbours []*node[K], deleted *node[K], layer int) {
	maxNeighbourCount := maxNeighbours(layer, g.maxNeighboursPerLayer, g.maxNeighboursBaseLayer)

	for _, neighbour := range neighbours {
		if len(neighbour.neighbours[layer]) >= maxNeighbourCount {
			continue
		}

		g.replenishNode(neighbour, deleted, layer, maxNeighbourCount)
	}
}

// replenishNode adds replacement edges from neighbours of neighbours.
//
// Takes target (*node[K]) which is the node that needs more connections.
// Takes deleted (*node[K]) which is the node being removed (to skip).
// Takes layer (int) which is the layer to repair.
// Takes maxNeighbourCount (int) which is the maximum neighbour count.
//
// Caller must hold the write lock.
func (*Graph[K]) replenishNode(target, deleted *node[K], layer, maxNeighbourCount int) {
	for _, neighbour := range target.neighbours[layer] {
		if len(target.neighbours[layer]) >= maxNeighbourCount {
			return
		}

		for _, candidate := range neighbour.neighbours[layer] {
			if len(target.neighbours[layer]) >= maxNeighbourCount {
				return
			}
			if candidate == target || candidate == deleted {
				continue
			}
			if neighbourContains(target.neighbours[layer], candidate) {
				continue
			}

			target.neighbours[layer] = neighbourAdd(target.neighbours[layer], candidate)
			candidate.neighbours[layer] = neighbourAdd(candidate.neighbours[layer], target)
		}
	}
}

// removeFromLayers removes the node from all layer maps.
//
// Takes target (*node[K]) which is the node to remove.
//
// Caller must hold the write lock.
func (g *Graph[K]) removeFromLayers(target *node[K]) {
	for layer := 0; layer <= target.level; layer++ {
		if layer < len(g.layers) {
			delete(g.layers[layer], target.key)
		}
	}
}

// electNewEntry selects a new entry point after the current one is deleted,
// picking a node from the highest non-empty layer or setting hasEntry to
// false if the graph is empty.
//
// Caller must hold the write lock.
func (g *Graph[K]) electNewEntry() {
	if len(g.nodes) == 0 {
		g.hasEntry = false
		return
	}

	for layer := len(g.layers) - 1; layer >= 0; layer-- {
		for k := range g.layers[layer] {
			g.entry = k
			return
		}
	}

	for k := range g.nodes {
		g.entry = k
		return
	}
}

// trimEmptyLayers removes empty layers from the end of the layer list.
//
// The caller must hold the write lock.
func (g *Graph[K]) trimEmptyLayers() {
	for len(g.layers) > 0 && len(g.layers[len(g.layers)-1]) == 0 {
		g.layers = g.layers[:len(g.layers)-1]
	}
}
