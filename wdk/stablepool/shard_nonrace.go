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

//go:build !race

package stablepool

// privateSlot holds the per-P fast-path object accessed only under runtime.procPin in
// non-race builds.
//
// Concurrency: Not safe outside runtime.procPin; the owning P guarantees single-threaded
// access without atomics.
type privateSlot[T any] struct {
	// p holds the cached object, or nil when the slot is empty.
	p *T
}

// tryTake reads the slot and clears it.
//
// Returns the previously held object, or nil if the slot was empty.
func (s *privateSlot[T]) tryTake() *T {
	obj := s.p
	s.p = nil
	return obj
}

// tryPut stores obj into the slot when it is empty.
//
// Takes obj (*T) which is the object to cache in the slot.
//
// Returns true when the value was stored, false when the slot was already occupied.
func (s *privateSlot[T]) tryPut(obj *T) bool {
	if s.p == nil {
		s.p = obj
		return true
	}
	return false
}

// drainSlot reads and clears the slot for shutdown or test paths and is not safe under
// concurrent Get/Put.
//
// Returns the previously held object, or nil if the slot was empty.
func (s *privateSlot[T]) drainSlot() *T {
	obj := s.p
	s.p = nil
	return obj
}
