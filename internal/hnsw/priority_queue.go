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

// priorityQueueItem pairs a node with its distance to the query vector.
// Storing the distance avoids repeated calculations during heap operations.
type priorityQueueItem[K comparable] struct {
	// node is the heap node that this queue item wraps.
	node *node[K]

	// distance is the distance from the start node.
	distance float32
}

// priorityQueue is a binary heap of priorityQueueItem values ordered by
// cached distance.
//
// When max is false (min-heap), the nearest item is at the root and popped
// first. When max is true (max-heap), the farthest item is at the root.
type priorityQueue[K comparable] struct {
	// items holds the heap elements in priority order.
	items []priorityQueueItem[K]

	// max indicates whether this is a max-heap (true) or min-heap (false).
	max bool
}

// less reports whether the item at index i should be closer to the root
// than the item at index j.
//
// Takes i (int) which is the index of the first item to compare.
// Takes j (int) which is the index of the second item to compare.
//
// Returns bool which is true if item i has higher priority than item j.
func (priorityQueue *priorityQueue[K]) less(i, j int) bool {
	if priorityQueue.max {
		return priorityQueue.items[i].distance > priorityQueue.items[j].distance
	}
	return priorityQueue.items[i].distance < priorityQueue.items[j].distance
}

// push adds an item to the heap.
//
// Takes item (priorityQueueItem[K]) which is the node and distance pair to
// add.
func (priorityQueue *priorityQueue[K]) push(item priorityQueueItem[K]) {
	priorityQueue.items = append(priorityQueue.items, item)
	priorityQueue.siftUp(len(priorityQueue.items) - 1)
}

// pop removes and returns the root element.
//
// Returns priorityQueueItem[K] which is the nearest (min-heap) or farthest
// (max-heap) item.
func (priorityQueue *priorityQueue[K]) pop() priorityQueueItem[K] {
	item := priorityQueue.items[0]
	last := len(priorityQueue.items) - 1
	priorityQueue.items[0] = priorityQueue.items[last]
	priorityQueue.items[last] = priorityQueueItem[K]{}
	priorityQueue.items = priorityQueue.items[:last]
	if len(priorityQueue.items) > 0 {
		priorityQueue.siftDown(0)
	}
	return item
}

// peek returns the root element without removing it.
//
// Returns priorityQueueItem[K] which is the current root.
func (priorityQueue *priorityQueue[K]) peek() priorityQueueItem[K] {
	return priorityQueue.items[0]
}

// replacePeek replaces the root element and re-heapifies.
// More efficient than pop + push for maintaining a fixed-size heap.
//
// Takes item (priorityQueueItem[K]) which is the replacement item.
func (priorityQueue *priorityQueue[K]) replacePeek(item priorityQueueItem[K]) {
	priorityQueue.items[0] = item
	priorityQueue.siftDown(0)
}

// len returns the number of elements in the queue.
//
// Returns int which is the count of elements in the queue.
func (priorityQueue *priorityQueue[K]) len() int {
	return len(priorityQueue.items)
}

// heapSort sorts the heap items in-place by distance (nearest first) using
// classic heapsort. For a max-heap this naturally produces ascending order.
//
// After calling heapSort, the items slice is sorted and the heap property is
// destroyed. The caller can use priorityQueue.items directly as the sorted result.
func (priorityQueue *priorityQueue[K]) heapSort() {
	for end := len(priorityQueue.items) - 1; end > 0; end-- {
		priorityQueue.items[0], priorityQueue.items[end] = priorityQueue.items[end], priorityQueue.items[0]
		priorityQueue.siftDownBounded(0, end)
	}
}

// siftUp restores the heap property by moving an element upward.
//
// Takes i (int) which is the index of the element to move up.
func (priorityQueue *priorityQueue[K]) siftUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if !priorityQueue.less(i, parent) {
			break
		}
		priorityQueue.items[i], priorityQueue.items[parent] = priorityQueue.items[parent], priorityQueue.items[i]
		i = parent
	}
}

// siftDown restores the heap property by moving an element downward.
//
// Takes i (int) which is the index of the element to move down.
func (priorityQueue *priorityQueue[K]) siftDown(i int) {
	priorityQueue.siftDownBounded(i, len(priorityQueue.items))
}

// siftDownBounded restores the heap property within items[0:bound].
//
// Takes i (int) which is the index of the element to move down.
// Takes bound (int) which is the upper bound of the slice to check.
func (priorityQueue *priorityQueue[K]) siftDownBounded(i, bound int) {
	for {
		best := i
		left := 2*i + 1
		right := 2*i + 2

		if left < bound && priorityQueue.less(left, best) {
			best = left
		}
		if right < bound && priorityQueue.less(right, best) {
			best = right
		}
		if best == i {
			break
		}
		priorityQueue.items[i], priorityQueue.items[best] = priorityQueue.items[best], priorityQueue.items[i]
		i = best
	}
}
