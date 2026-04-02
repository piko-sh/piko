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

package retry

import (
	"container/heap"
	"time"
)

// Heap is a generic min-heap that orders items by time, wrapping
// container/heap with type-safe push, pop, and peek operations.
//
// Behaviour:
//   - Items with earlier times are returned first.
type Heap[T any] struct {
	// timeFunc extracts the priority time from each item.
	timeFunc func(T) time.Time

	// items holds the heap-ordered elements.
	items []T
}

// Len returns the number of items in the heap.
//
// Returns int which is the current heap size.
func (h *Heap[T]) Len() int { return len(h.items) }

// Less reports whether item i has an earlier priority time than item j.
// Required by heap.Interface.
//
// Takes i (int) which is the index of the first item.
// Takes j (int) which is the index of the second item.
//
// Returns bool which is true when item i should be popped before item j.
func (h *Heap[T]) Less(i, j int) bool {
	return h.timeFunc(h.items[i]).Before(h.timeFunc(h.items[j]))
}

// Swap exchanges the elements at indices i and j. Required by
// heap.Interface.
//
// Takes i (int) which is the index of the first element.
// Takes j (int) which is the index of the second element.
func (h *Heap[T]) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
}

// Push adds x to the heap. Required by heap.Interface; prefer PushItem
// for type-safe access.
//
// Takes x (any) which must be of type T; other types are silently ignored.
func (h *Heap[T]) Push(x any) {
	if item, ok := x.(T); ok {
		h.items = append(h.items, item)
	}
}

// Pop removes and returns the last element. Required by heap.Interface;
// prefer PopItem for type-safe access.
//
// Returns any which is the removed item of type T.
func (h *Heap[T]) Pop() any {
	old := h.items
	n := len(old)
	item := old[n-1]
	var zero T
	old[n-1] = zero
	h.items = old[:n-1]
	return item
}

// PushItem adds an item to the heap, maintaining min-heap ordering by
// priority time.
//
// Takes item (T) which is the item to add.
func (h *Heap[T]) PushItem(item T) {
	heap.Push(h, item)
}

// PopItem removes and returns the item with the earliest priority time.
//
// Returns T which is the item with the smallest priority time.
// Returns bool which is false if the heap is empty.
func (h *Heap[T]) PopItem() (T, bool) {
	if len(h.items) == 0 {
		var zero T
		return zero, false
	}
	item, ok := heap.Pop(h).(T)
	return item, ok
}

// Peek returns the item with the earliest priority time without removing
// it.
//
// Returns T which is the item with the smallest priority time.
// Returns bool which is false if the heap is empty.
func (h *Heap[T]) Peek() (T, bool) {
	if len(h.items) == 0 {
		var zero T
		return zero, false
	}
	return h.items[0], true
}

// NewHeap creates an empty min-heap.
//
// Takes timeFunc (func(T) time.Time) which extracts the priority time from
// each item; items with earlier times are popped first.
//
// Returns *Heap[T] which is ready to use.
func NewHeap[T any](timeFunc func(T) time.Time) *Heap[T] {
	return &Heap[T]{
		items:    nil,
		timeFunc: timeFunc,
	}
}
