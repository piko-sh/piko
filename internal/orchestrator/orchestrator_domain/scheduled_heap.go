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

package orchestrator_domain

import (
	"time"
)

// scheduledRef is a task reference waiting for its due time.
type scheduledRef struct {
	// due is when the reference is published.
	due time.Time

	// ref identifies the task and attempt to publish.
	ref TaskRef
}

// scheduledRefHeap is a min-heap of scheduled references ordered by due time. It holds
// values, so scheduling a reference allocates nothing once the heap has grown, and never
// keeps a task the caller may still change.
type scheduledRefHeap []scheduledRef

// Len returns the number of references in the heap.
//
// Returns int which is the current heap size.
func (h *scheduledRefHeap) Len() int { return len(*h) }

// Push adds an entry to the heap.
//
// Takes entry (scheduledRef) which is the reference and its due time.
func (h *scheduledRefHeap) Push(entry scheduledRef) {
	*h = append(*h, entry)
	h.up(h.Len() - 1)
}

// Pop removes and returns the entry due first.
//
// Returns scheduledRef which is the removed entry.
// Returns bool which is false when the heap is empty.
func (h *scheduledRefHeap) Pop() (scheduledRef, bool) {
	n := h.Len() - 1
	if n < 0 {
		return scheduledRef{}, false
	}
	h.swap(0, n)
	entry := (*h)[n]
	(*h)[n] = scheduledRef{}
	*h = (*h)[:n]
	if n > 0 {
		h.down(0)
	}
	return entry, true
}

// Peek returns the entry due first without removing it.
//
// Returns scheduledRef which is the next entry.
// Returns bool which is false when the heap is empty.
func (h *scheduledRefHeap) Peek() (scheduledRef, bool) {
	if h.Len() == 0 {
		return scheduledRef{}, false
	}
	return (*h)[0], true
}

// less reports whether the entry at index i is due before the entry at index j.
//
// Takes i (int) which is the index of the first entry.
// Takes j (int) which is the index of the second entry.
//
// Returns bool which is true when entry i is due first.
func (h *scheduledRefHeap) less(i, j int) bool {
	return (*h)[i].due.Before((*h)[j].due)
}

// swap exchanges the entries at positions i and j.
//
// Takes i (int) which is the position of the first entry.
// Takes j (int) which is the position of the second entry.
func (h *scheduledRefHeap) swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
}

// up moves the entry at index i up until the heap property holds.
//
// Takes i (int) which is the index of the entry.
func (h *scheduledRefHeap) up(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if !h.less(i, parent) {
			break
		}
		h.swap(parent, i)
		i = parent
	}
}

// down moves the entry at index i down until the heap property holds.
//
// Takes i (int) which is the index of the entry.
func (h *scheduledRefHeap) down(i int) {
	n := h.Len()
	for {
		left := 2*i + 1
		if left >= n {
			break
		}
		smallest := left
		if right := left + 1; right < n && h.less(right, left) {
			smallest = right
		}
		if !h.less(smallest, i) {
			break
		}
		h.swap(i, smallest)
		i = smallest
	}
}
