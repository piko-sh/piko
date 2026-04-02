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

package tui_domain

// historyRingDefaultCapacity is the default capacity for HistoryRing buffers.
// This stores 1 hour of history when using 2-second refresh intervals.
const historyRingDefaultCapacity = 1800

// HistoryRing is a fixed-size circular buffer for sparkline data.
// It maintains a sliding window of the most recent values, automatically
// evicting the oldest values when capacity is exceeded.
type HistoryRing struct {
	// values holds the recorded values in the order they were added.
	values []float64

	// capacity is the maximum number of values the buffer can hold.
	capacity int
}

// NewHistoryRing creates a new history ring with the given capacity.
//
// Capacity must be positive; if zero or negative, defaults to
// historyRingDefaultCapacity (equivalent to 1 hour at 2-second refresh
// intervals).
//
// Takes capacity (int) which specifies the maximum number of values to store.
//
// Returns *HistoryRing which is the initialised ring buffer ready for use.
func NewHistoryRing(capacity int) *HistoryRing {
	if capacity <= 0 {
		capacity = historyRingDefaultCapacity
	}
	return &HistoryRing{
		values:   make([]float64, 0, capacity),
		capacity: capacity,
	}
}

// Append adds a value to the history ring buffer.
// When the buffer is full, the oldest value is removed.
//
// Takes value (float64) which is the value to add.
func (h *HistoryRing) Append(value float64) {
	h.values = append(h.values, value)
	if len(h.values) > h.capacity {
		h.values = h.values[len(h.values)-h.capacity:]
	}
}

// AppendAll adds multiple values to the history.
// Values are added in order, with the oldest first.
//
// Takes values ([]float64) which contains the values to append.
func (h *HistoryRing) AppendAll(values []float64) {
	for _, v := range values {
		h.Append(v)
	}
}

// Values returns all values in chronological order (oldest first).
//
// Returns []float64 which is a copy of the values, safe to modify.
func (h *HistoryRing) Values() []float64 {
	if len(h.values) == 0 {
		return nil
	}
	result := make([]float64, len(h.values))
	copy(result, h.values)
	return result
}

// Len returns the current number of values in the buffer.
//
// Returns int which is the count of values stored in the ring.
func (h *HistoryRing) Len() int {
	return len(h.values)
}

// Capacity returns the maximum number of values the buffer can hold.
//
// Returns int which is the buffer's maximum capacity.
func (h *HistoryRing) Capacity() int {
	return h.capacity
}

// Latest returns the most recent value, or 0 if the ring is empty.
//
// Returns float64 which is the most recently added value.
func (h *HistoryRing) Latest() float64 {
	if len(h.values) == 0 {
		return 0
	}
	return h.values[len(h.values)-1]
}

// Clear removes all values from the buffer.
func (h *HistoryRing) Clear() {
	h.values = h.values[:0]
}

// Stats returns basic statistics about the values in the buffer.
// If the buffer is empty, all values are zero.
//
// Returns minVal (float64) which is the smallest value in the buffer.
// Returns maxVal (float64) which is the largest value in the buffer.
// Returns avg (float64) which is the arithmetic mean of all values.
func (h *HistoryRing) Stats() (minVal, maxVal, avg float64) {
	if len(h.values) == 0 {
		return 0, 0, 0
	}

	minVal, maxVal = h.values[0], h.values[0]
	sum := 0.0

	for _, v := range h.values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
		sum += v
	}

	avg = sum / float64(len(h.values))
	return minVal, maxVal, avg
}
