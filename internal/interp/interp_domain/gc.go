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

package interp_domain

const (
	// gcInitialThreshold is the default value for nextGCAt on first allocation, sized to
	// roughly the initial byte-slab capacity so a short-running Execute that stays under the
	// threshold never pays for a GC cycle. Long-running scripts allocate past this and
	// trigger the first MinorGC, after which the adaptive tuner takes over.
	gcInitialThreshold int64 = 1 << 20

	// gcMinThresholdDelta is the minimum number of bytes that must accumulate between
	// consecutive MinorGC calls. Prevents pathological tight loops where a single allocation
	// immediately re-triggers GC because postGCBytes is already near the previous threshold.
	gcMinThresholdDelta int64 = 1 << 20

	// gcMaxThresholdDelta caps the upper bound on nextGCAt growth per cycle. Without it the
	// threshold could grow unboundedly on mostly-live arenas, hiding genuine bloat behind
	// ever-deferred GC cycles.
	gcMaxThresholdDelta int64 = int64(initialByteSlabSize) * int64(maxArenaMultiplier) * 64

	// gcRatioMostlyLive is the live/total ratio above which the adaptive tuner backs off the
	// GC frequency. When most of the arena is reachable, GC is doing little useful work, so
	// we let allocation push further before trying again.
	gcRatioMostlyLive = 0.75

	// gcRatioMostlyGarbage is the live/total ratio below which the tuner keeps the GC
	// frequency tight. When the arena is mostly garbage each cycle is recovering significant
	// memory, so a modest threshold makes sense.
	gcRatioMostlyGarbage = 0.25

	// gcMostlyLiveGrowthMultiplier scales the observed allocation growth when the arena is
	// mostly live. A larger multiplier means the next GC is deferred further so allocation
	// can continue without thrashing.
	gcMostlyLiveGrowthMultiplier int64 = 4

	// gcBalancedGrowthMultiplier is the growth multiplier used when the live/total ratio is
	// between the mostly-live and mostly-garbage thresholds.
	gcBalancedGrowthMultiplier int64 = 2
)

// noteAlloc accounts for a just-completed data-slab bump allocation by adding its size to
// the arena's running byte total. Pure bookkeeping; the GC trigger condition is evaluated
// at the dispatch loop's existing safe-point (every cancellationCheckMask+1 instructions)
// rather than on every allocation, keeping the hot allocation path at a single int64
// increment.
//
// Called only by the data-slab bump-alloc paths (byteSlab, genericBytesSlab,
// sliceHeaderSlab, the backing slabs, the box slabs). Frame-tied register banks (intSlab,
// floatSlab, etc.) bypass noteAlloc because they self-manage via frame push/pop and never
// drive GC pressure.
//
// Takes bytes (int64) which is the size of the just-completed allocation.
//
//go:nosplit
func (a *RegisterArena) noteAlloc(bytes int64) {
	a.bytesAllocated += bytes
}

// gcShouldRun reports whether MinorGC should run at the next safe point.
//
// Returns true when bytesAllocated has crossed the nextGCAt threshold (using
// gcInitialThreshold when nextGCAt is still zero).
//
//go:nosplit
func (a *RegisterArena) gcShouldRun() bool {
	threshold := a.nextGCAt
	if threshold == 0 {
		threshold = gcInitialThreshold
	}
	return a.bytesAllocated >= threshold
}

// runPendingCheckpoints services pending dispatch-loop checkpoint signals.
//
// Cold path; only called when checkpointFlags is non-zero or the arena has crossed its GC
// threshold. Dispatches the GC trigger when the checkpointFlagGCPending bit is set or the
// arena's bytesAllocated exceeds its threshold, then clears the flag.
//
// Cross-goroutine signals (cancellation, goroutine panic) are handled by the immediately
// preceding atomic checks in the dispatch loop and are not seen here.
func (vm *VM) runPendingCheckpoints() {
	if vm.arena != nil && (vm.checkpointFlags&checkpointFlagGCPending != 0 || vm.arena.gcShouldRun()) {
		vm.checkpointFlags &^= checkpointFlagGCPending
		vm.arena.MinorGC(vm)
	}
}

// MinorGC runs a single stop-the-world arena garbage-collection cycle.
//
// The owning VM's dispatch loop is paused for the duration; no other goroutine is
// affected because each arena is owned by exactly one VM goroutine (the per-arena
// single-owner invariant established by runCompiledGoroutine spawning child arenas, and
// by the materialise* escape barrier preventing cross-arena pointers).
//
// Strategy: identify and drop fully-dead OLD slab generations. The current-generation
// slabs are left intact and their bytes are reused by subsequent bump allocations.
// Reclaims memory most effectively when long-running scripts have generated multiple
// slab-growth events and the retained old generations are entirely dead.
//
// Takes vm (*VM) which is the owning VM, used for root enumeration.
func (a *RegisterArena) MinorGC(vm *VM) {
	a.gcCount++
	preGCBytes := a.bytesAllocated
	state := acquireGCMarkState(a)
	defer releaseGCMarkState(state)
	vm.markPhase(state)
	reclaimed := a.compactPhase(state)
	postGCBytes := max(preGCBytes-reclaimed, 0)
	a.bytesAllocated = postGCBytes
	a.bytesAtLastGC = postGCBytes
	a.deductReclaimedFromBudget(reclaimed)
	a.updateNextGCAt(preGCBytes, postGCBytes)
}

// deductReclaimedFromBudget reduces the arena's live-bytes counter by the capacity
// reclaimed during compactPhase. The two counters share the same backing accounting: a
// slab charges itself into totalAllocatedBytes at grow time, and surrenders the same
// number of bytes when MinorGC drops it from the retention list.
//
// Takes reclaimedBytes (int64) which is the byte capacity dropped by compactPhase.
// Negative or zero values are no-ops.
func (a *RegisterArena) deductReclaimedFromBudget(reclaimedBytes int64) {
	if reclaimedBytes <= 0 {
		return
	}
	reclaimed := uint64(reclaimedBytes)
	if reclaimed >= a.totalAllocatedBytes {
		a.totalAllocatedBytes = 0
		return
	}
	a.totalAllocatedBytes -= reclaimed
}

// updateNextGCAt sets the next GC trigger threshold based on the observed live ratio.
// Called at the end of MinorGC after compaction has revealed how much memory survived.
//
// The algorithm distinguishes three cases. When the live ratio is above gcRatioMostlyLive
// the threshold is backed off to postGCBytes + 4 * growth so allocation has more room
// because GC was doing little useful work. When the ratio is below gcRatioMostlyGarbage
// the threshold is kept tight at postGCBytes + 1 * growth. Otherwise the threshold lands
// at postGCBytes + 2 * growth.
//
// The result is floored at postGCBytes + gcMinThresholdDelta and ceilinged at postGCBytes
// + gcMaxThresholdDelta to prevent pathological extremes.
//
// Takes preGCBytes (int64) which is bytesAllocated at trigger time.
// Takes postGCBytes (int64) which is bytesAllocated after compaction.
func (a *RegisterArena) updateNextGCAt(preGCBytes, postGCBytes int64) {
	if preGCBytes <= 0 {
		a.nextGCAt = postGCBytes + gcInitialThreshold
		return
	}
	liveRatio := float64(postGCBytes) / float64(preGCBytes)
	growth := preGCBytes - a.bytesAtLastGC
	if growth <= 0 {
		growth = gcInitialThreshold
	}
	var delta int64
	switch {
	case liveRatio > gcRatioMostlyLive:
		delta = growth * gcMostlyLiveGrowthMultiplier
	case liveRatio < gcRatioMostlyGarbage:
		delta = growth
	default:
		delta = growth * gcBalancedGrowthMultiplier
	}
	delta = min(max(delta, gcMinThresholdDelta), gcMaxThresholdDelta)
	a.nextGCAt = postGCBytes + delta
}
