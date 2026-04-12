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

//go:build race

package stablepool

import "sync/atomic"

// privateSlot holds the per-P fast-path object in race builds.
//
// The race-build variant wraps the pointer in atomic.Pointer so the race detector accepts
// the cross-goroutine accesses that occur when goroutines reschedule onto
// previously-occupied Ps. Non-race builds use a plain pointer.
//
// Concurrency: Safe for concurrent use; backed by atomic.Pointer.
type privateSlot[T any] struct {
	// p holds the per-P pooled object pointer, accessed atomically.
	p atomic.Pointer[T]
}

// tryTake atomically removes and returns the slot's current value, or nil.
//
// Returns the previously stored pointer, or nil when the slot was empty.
func (s *privateSlot[T]) tryTake() *T {
	return s.p.Swap(nil)
}

// tryPut stores obj into the slot when the slot is empty.
//
// Takes obj (*T) which is the pointer to attempt to store in the slot.
//
// Returns true when obj was stored, false when the slot already held a value.
func (s *privateSlot[T]) tryPut(obj *T) bool {
	return s.p.CompareAndSwap(nil, obj)
}

// drainSlot atomically clears the slot and returns its previous value.
//
// Returns the previously stored pointer, or nil when the slot was empty.
func (s *privateSlot[T]) drainSlot() *T {
	return s.p.Swap(nil)
}
