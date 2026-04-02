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

// scheduledTaskHeap is a min-heap of tasks ordered by their scheduled time.
// It allows quick insertion and removal of tasks for timer-based scheduling.
type scheduledTaskHeap []*Task

// Len returns the number of tasks in the heap.
//
// Returns int which is the current heap size.
func (h *scheduledTaskHeap) Len() int { return len(*h) }

// Less reports whether the element with index i should sort before the
// element with index j.
//
// Takes i (int) which is the index of the first element to compare.
// Takes j (int) which is the index of the second element to compare.
//
// Returns bool which is true when element i has an earlier scheduled time
// than element j.
func (h *scheduledTaskHeap) Less(i, j int) bool {
	return (*h)[i].ScheduledExecuteAt.Before((*h)[j].ScheduledExecuteAt)
}

// Swap exchanges the elements at positions i and j.
//
// Takes i (int) which is the position of the first element.
// Takes j (int) which is the position of the second element.
func (h *scheduledTaskHeap) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
}

// Push adds a task to the heap, maintaining the min-heap property.
//
// Takes x (any) which is the task to add; must be of type *Task.
func (h *scheduledTaskHeap) Push(x any) {
	*h = append(*h, x.(*Task))
	h.up(h.Len() - 1)
}

// Pop removes and returns the task with the earliest scheduled run time.
//
// Returns *Task which is the removed task, or nil if the heap is empty.
func (h *scheduledTaskHeap) Pop() *Task {
	if h.Len() == 0 {
		return nil
	}
	n := h.Len() - 1
	h.Swap(0, n)
	task := (*h)[n]
	*h = (*h)[:n]
	if h.Len() > 0 {
		h.down(0)
	}
	return task
}

// Peek returns the task with the earliest ScheduledExecuteAt without removing
// it.
//
// Returns *Task which is the next scheduled task, or nil if the heap is empty.
func (h *scheduledTaskHeap) Peek() *Task {
	if h.Len() == 0 {
		return nil
	}
	return (*h)[0]
}

// up bubbles the element at index i up to maintain heap property.
//
// Takes i (int) which is the index of the element to bubble up.
func (h *scheduledTaskHeap) up(i int) {
	for {
		parent := (i - 1) / 2
		if parent == i || !h.Less(i, parent) {
			break
		}
		h.Swap(parent, i)
		i = parent
	}
}

// down bubbles the element at index i down to maintain heap property.
//
// Takes i (int) which is the index of the element to bubble down.
func (h *scheduledTaskHeap) down(i int) {
	n := h.Len()
	for {
		left := 2*i + 1
		if left >= n {
			break
		}
		smallest := left
		if right := left + 1; right < n && h.Less(right, left) {
			smallest = right
		}
		if !h.Less(smallest, i) {
			break
		}
		h.Swap(i, smallest)
		i = smallest
	}
}

// newScheduledTaskHeap creates an empty scheduled task heap.
//
// Returns *scheduledTaskHeap which is an empty heap ready for use.
func newScheduledTaskHeap() *scheduledTaskHeap {
	return new(make(scheduledTaskHeap, 0))
}
