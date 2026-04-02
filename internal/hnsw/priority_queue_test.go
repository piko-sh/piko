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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriorityQueue_PushPopPeek(t *testing.T) {
	priorityQueue := &priorityQueue[string]{}

	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "c"}, distance: 3})
	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "a"}, distance: 1})
	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "b"}, distance: 2})

	assert.Equal(t, 3, priorityQueue.len())
	assert.Equal(t, "a", priorityQueue.peek().node.key)

	popped := priorityQueue.pop()
	assert.Equal(t, "a", popped.node.key)
	assert.Equal(t, float32(1), popped.distance)
	assert.Equal(t, 2, priorityQueue.len())

	popped = priorityQueue.pop()
	assert.Equal(t, "b", popped.node.key)

	popped = priorityQueue.pop()
	assert.Equal(t, "c", popped.node.key)
	assert.Equal(t, 0, priorityQueue.len())
}

func TestPriorityQueue_MaxHeap(t *testing.T) {
	priorityQueue := &priorityQueue[string]{max: true}

	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "a"}, distance: 1})
	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "c"}, distance: 3})
	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "b"}, distance: 2})

	assert.Equal(t, "c", priorityQueue.peek().node.key, "max-heap root should be farthest")

	popped := priorityQueue.pop()
	assert.Equal(t, "c", popped.node.key)

	popped = priorityQueue.pop()
	assert.Equal(t, "b", popped.node.key)

	popped = priorityQueue.pop()
	assert.Equal(t, "a", popped.node.key)
}

func TestPriorityQueue_HeapSort(t *testing.T) {
	priorityQueue := &priorityQueue[string]{max: true}

	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "c"}, distance: 3})
	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "a"}, distance: 1})
	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "b"}, distance: 2})

	priorityQueue.heapSort()
	require.Len(t, priorityQueue.items, 3)
	assert.Equal(t, "a", priorityQueue.items[0].node.key)
	assert.Equal(t, "b", priorityQueue.items[1].node.key)
	assert.Equal(t, "c", priorityQueue.items[2].node.key)
}

func TestPriorityQueue_ReplacePeek(t *testing.T) {
	priorityQueue := &priorityQueue[string]{}

	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "a"}, distance: 1})
	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "b"}, distance: 2})
	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "c"}, distance: 3})

	assert.Equal(t, "a", priorityQueue.peek().node.key)

	priorityQueue.replacePeek(priorityQueueItem[string]{node: &node[string]{key: "d"}, distance: 2.5})

	assert.Equal(t, "b", priorityQueue.peek().node.key)
	assert.Equal(t, 3, priorityQueue.len())
}

func TestPriorityQueue_SingleElement(t *testing.T) {
	priorityQueue := &priorityQueue[string]{}

	priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "x"}, distance: 5})
	assert.Equal(t, 1, priorityQueue.len())
	assert.Equal(t, "x", priorityQueue.peek().node.key)

	popped := priorityQueue.pop()
	assert.Equal(t, "x", popped.node.key)
	assert.Equal(t, 0, priorityQueue.len())
}

func TestPriorityQueue_ManyElements(t *testing.T) {
	priorityQueue := &priorityQueue[string]{}

	for i := 20; i > 0; i-- {
		priorityQueue.push(priorityQueueItem[string]{node: &node[string]{key: "n"}, distance: float32(i)})
	}

	previous := float32(0)
	for priorityQueue.len() > 0 {
		item := priorityQueue.pop()
		assert.GreaterOrEqual(t, item.distance, previous, "should pop in ascending order")
		previous = item.distance
	}
}
