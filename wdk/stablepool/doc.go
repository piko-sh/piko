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

// Package stablepool is a pre-warmed, GC-surviving, sharded object pool.
//
// The defining property is in the name: objects are stable. Once the pool is constructed,
// its capacity is committed up-front and the contents persist across every Go GC cycle.
// There is no first-call allocation penalty, no victim-cache cliff after GC, and no
// surprise re-initialisation. In return, the pool's memory footprint matches the
// requested capacity. stablepool does not shrink under memory pressure (use ModeGCAware
// if you want that trade-off).
//
// # When to use stablepool
//
//   - Long-lived services where post-GC latency must stay flat.
//   - Hot paths whose pooled objects carry non-trivial allocated state (preallocated
//     buffers, maps, builders) that you want preserved across reuses, because sync.Pool
//     drops these on GC and forces re-allocation.
//   - Workloads where you know the working set up front and prefer a fixed memory floor
//     over dynamic shrinkage.
//
// # When NOT to use stablepool
//
//   - Sporadic / bursty workloads where the memory floor matters more than post-GC
//     latency: use ModeGCAware (sync.Pool semantics under the same typed API) or use
//     sync.Pool directly.
//   - Short-lived pools where the pre-warm cost dominates total cost.
//
// # Modes
//
// ModePersistent (default): slab-backed, GC-surviving, pre-warmed at New. ModeGCAware:
// sync.Pool-equivalent reclamation, typed API surface.
//
// # Concurrency
//
// Get and Put are safe for any number of concurrent goroutines. Each goroutine takes a
// per-P fast slot under runtime.procPin (non-atomic in non-race builds, atomic.Pointer in
// race builds), then falls through to a sharded counted-pointer Treiber free stack on
// miss, and finally to a cross-shard steal walk. Growth, if enabled via WithGrowth,
// appends new slabs under a single mutex; growth events are rare and only fire when every
// shard is empty.
//
// The free stack is ABA-safe: a 16-bit generation counter is packed into the high bits of
// the 48-bit slab-slot pointer; every successful CAS bumps the counter, closing the
// Treiber ABA window.
//
// # Intrusive Link
//
// Types stored in a stablepool Pool[T] MUST embed Link as their first field:
//
//	type MyObject struct {
//	    stablepool.Link
//	    Name string
//	    Buf  []byte
//	}
//
// Link is one pointer wide. The pool writes it when the object is on the free stack and
// leaves it untouched while the object is in user hands.
//
// # Build tags
//
// Production builds use non-atomic per-P slot access under procPin. Race builds (-race)
// automatically swap in atomic.Pointer[T] for the slot so the detector accepts the
// cross-goroutine accesses produced by Ps that rotate between goroutines.
package stablepool
