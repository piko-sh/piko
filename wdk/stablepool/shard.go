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

package stablepool

import (
	"sync/atomic"
	"unsafe"
)

// shard is one per-P cache region.
//
// Each shard carries a per-P "private" slot (a single object, accessed only under procPin
// by the owning P; see privateSlot[T] in shard_nonrace.go and shard_race.go) plus a
// lock-free Treiber free stack (freeHead) holding the shard's overflow objects,
// accessible from any P for cross-shard steal.
//
// freeHead is an atomic.Pointer[T] (not a counted-pointer uint64) so Go's GC traces it as
// a live reference. This is critical for objects reached only via the free stack: without
// GC tracing, heap-allocated objects in the chain become unreachable from GC's
// perspective and get reclaimed while the pool still references them, producing "pointer
// to unallocated span" GC corruption. The trade-off is that explicit ABA protection is
// lost, but the use case is per-shard with low cross-shard contention, and Go's heap
// allocator does not reuse pointers fast enough in practice to trigger ABA windows
// shorter than the CAS retry cost.
//
// Cache-line padded to 128 bytes so neighbouring shards in the shards array never
// false-share.
type shard[T any] struct {
	// slot is the per-P private object accessed only under procPin.
	slot privateSlot[T]

	// freeHead is the lock-free Treiber stack of overflow objects.
	freeHead atomic.Pointer[T]

	_ [128 - 16]byte
}

// popFree removes and returns the head of this shard's free stack.
//
// The next pointer is accessed via atomic.LoadPointer so the race detector sees
// synchronisation across the push-then-pop chain.
//
// Returns the detached head object, or nil when the stack is empty.
//
//go:nocheckptr
func (s *shard[T]) popFree() *T {
	for {
		head := s.freeHead.Load()
		if head == nil {
			return nil
		}
		next := (*T)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(head))))
		if s.freeHead.CompareAndSwap(head, next) {
			return head
		}
	}
}

// pushFree prepends obj to this shard's free stack. The intrusive Link.next at offset 0
// of T is written via atomic.StorePointer before the CAS, so a concurrent popFree never
// observes a stale next pointer and the race detector sees the happens-before edge.
//
// Takes obj which is the object to push onto the free stack head.
//
//go:nocheckptr
func (s *shard[T]) pushFree(obj *T) {
	objPtr := unsafe.Pointer(obj)
	for {
		head := s.freeHead.Load()
		atomic.StorePointer((*unsafe.Pointer)(objPtr), unsafe.Pointer(head))
		if s.freeHead.CompareAndSwap(head, obj) {
			return
		}
	}
}
