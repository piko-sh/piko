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

import "slices"

// node represents a single vector in the HNSW graph. Each node exists at one
// or more layers, with its neighbours stored per layer.
type node[K comparable] struct {
	// key is the external identifier for this vector.
	key K

	// vector is the stored float32 vector.
	vector []float32

	// neighbours stores the neighbour set for each layer of this node.
	// Index 0 is the base layer, and higher indices are upper layers; each
	// layer is an unsorted slice of node pointers.
	neighbours [][]*node[K]

	// level is the highest layer this node is assigned to (0-based).
	level int

	// id is a unique number used to track visited nodes.
	// The graph sets this when a node is added and reuses it when removed.
	id uint32
}

// newNode creates a node at the given level with pre-sized neighbour slices.
//
// Takes key (K) which identifies the node.
// Takes id (uint32) which is the dense internal identifier.
// Takes vector ([]float32) which is the vector data.
// Takes level (int) which is the highest layer this node exists in.
// Takes maxNeighboursPerLayer (int) which is the upper-layer neighbour limit.
// Takes maxNeighboursBaseLayer (int) which is the base-layer neighbour limit.
//
// Returns *node[K] with allocated neighbour slices.
func newNode[K comparable](key K, id uint32, vector []float32, level, maxNeighboursPerLayer, maxNeighboursBaseLayer int) *node[K] {
	neighbours := make([][]*node[K], level+1)
	for i := range neighbours {
		hint := maxNeighboursPerLayer
		if i == 0 {
			hint = maxNeighboursBaseLayer
		}
		neighbours[i] = make([]*node[K], 0, hint)
	}
	return &node[K]{
		key:        key,
		id:         id,
		vector:     vector,
		neighbours: neighbours,
		level:      level,
	}
}

// neighbourContains reports whether the target node is in the neighbour slice.
//
// Takes neighbours ([]*node[K]) which is the slice of neighbour nodes to
// search.
// Takes target (*node[K]) which is the node to find.
//
// Returns bool which is true if the target is found, false otherwise.
func neighbourContains[K comparable](neighbours []*node[K], target *node[K]) bool {
	return slices.Contains(neighbours, target)
}

// neighbourAdd appends a node to the neighbour slice.
//
// Takes neighbours ([]*node[K]) which is the current neighbour slice.
// Takes target (*node[K]) which is the node to append.
//
// Returns []*node[K] which is the updated neighbour slice.
func neighbourAdd[K comparable](neighbours []*node[K], target *node[K]) []*node[K] {
	return append(neighbours, target)
}

// neighbourRemove removes a node from the slice using swap-with-last.
//
// Takes neighbours ([]*node[K]) which is the current neighbour slice.
// Takes target (*node[K]) which is the node to remove.
//
// Returns []*node[K] which is the updated slice with the target removed.
func neighbourRemove[K comparable](neighbours []*node[K], target *node[K]) []*node[K] {
	for i, neighbour := range neighbours {
		if neighbour == target {
			last := len(neighbours) - 1
			neighbours[i] = neighbours[last]
			neighbours[last] = nil
			return neighbours[:last]
		}
	}
	return neighbours
}

// maxNeighbours returns the maximum number of neighbours for a given layer.
//
// Takes layer (int) which is the layer index.
// Takes maxNeighboursPerLayer (int) which is the limit for upper layers.
// Takes maxNeighboursBaseLayer (int) which is the limit for the base layer
// (usually 2 * maxNeighboursPerLayer).
//
// Returns int which is the neighbour limit for the given layer.
func maxNeighbours(layer, maxNeighboursPerLayer, maxNeighboursBaseLayer int) int {
	if layer == 0 {
		return maxNeighboursBaseLayer
	}
	return maxNeighboursPerLayer
}
