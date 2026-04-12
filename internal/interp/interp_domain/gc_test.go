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

import (
	"strings"
	"testing"

	"piko.sh/piko/internal/mem"
)

func newGCTestVM() *VM {
	return &VM{
		globals:   &globalStore{},
		arena:     newRegisterArena(),
		callStack: []callFrame{},
	}
}

func TestMinorGC_DropsFullyDeadOldByteSlabs(t *testing.T) {
	vm := newGCTestVM()
	arena := vm.arena

	for range 5 {
		arena.growByteSlab(initialByteSlabSize)
	}
	if len(arena.oldByteSlabs) == 0 {
		t.Fatal("expected oldByteSlabs to be populated after growth")
	}
	beforeCount := len(arena.oldByteSlabs)

	arena.MinorGC(vm)

	if len(arena.oldByteSlabs) != 0 {
		t.Errorf("expected oldByteSlabs to be drained (no live refs), got %d (started with %d)",
			len(arena.oldByteSlabs), beforeCount)
	}
}

func TestMinorGC_PreservesLiveOldByteSlabs(t *testing.T) {
	if !arenaUsesUnsafeSlabs {
		t.Skip("safe build: MinorGC slab-retention semantics require the unsafe arena")
	}
	vm := newGCTestVM()
	arena := vm.arena

	buffer := arena.AllocStringBytes(64)
	copy(buffer, strings.Repeat("a", 64))
	liveStr := mem.String(buffer)

	arena.growByteSlab(initialByteSlabSize)
	if len(arena.oldByteSlabs) != 1 {
		t.Fatalf("expected 1 old byteSlab after growth, got %d", len(arena.oldByteSlabs))
	}

	vm.globals.strings = []string{liveStr}

	arena.MinorGC(vm)

	if len(arena.oldByteSlabs) != 1 {
		t.Errorf("expected live old byteSlab to survive GC, got %d remaining", len(arena.oldByteSlabs))
	}
}

func TestMinorGC_DropsDeadOldIntBackings(t *testing.T) {
	vm := newGCTestVM()
	arena := vm.arena

	for range 3 {
		arena.growIntBackingSlab(initialIntBackingSize)
	}
	if len(arena.oldIntBackings) == 0 {
		t.Fatal("expected oldIntBackings to be populated after growth")
	}

	arena.MinorGC(vm)

	if len(arena.oldIntBackings) != 0 {
		t.Errorf("expected oldIntBackings to be drained (no live refs), got %d", len(arena.oldIntBackings))
	}
}

func TestMinorGC_PreservesLiveOldIntBackings(t *testing.T) {
	vm := newGCTestVM()
	arena := vm.arena

	liveSlice := arena.AllocIntBacking(16)
	for i := range liveSlice {
		liveSlice[i] = int64(i)
	}

	arena.growIntBackingSlab(initialIntBackingSize)
	if len(arena.oldIntBackings) != 1 {
		t.Fatalf("expected 1 old int backing after growth, got %d", len(arena.oldIntBackings))
	}

	vm.callStack = []callFrame{{
		registers: Registers{
			slicesInt: [][]int64{liveSlice},
		},
	}}
	vm.framePointer = 0

	arena.MinorGC(vm)

	if len(arena.oldIntBackings) != 1 {
		t.Errorf("expected live old int backing to survive GC, got %d remaining", len(arena.oldIntBackings))
	}
}

func TestMinorGC_UpdatesGCCount(t *testing.T) {
	vm := newGCTestVM()
	arena := vm.arena
	if arena.gcCount != 0 {
		t.Errorf("fresh arena should have gcCount=0, got %d", arena.gcCount)
	}
	arena.MinorGC(vm)
	arena.MinorGC(vm)
	if arena.gcCount != 2 {
		t.Errorf("expected gcCount=2 after two MinorGCs, got %d", arena.gcCount)
	}
}

func TestMinorGC_ResetsBytesAllocated(t *testing.T) {
	vm := newGCTestVM()
	arena := vm.arena

	arena.AllocStringBytes(1024)
	arena.AllocIntBacking(128)
	if arena.bytesAllocated == 0 {
		t.Fatal("expected bytesAllocated > 0 after allocations")
	}

	arena.MinorGC(vm)

	if arena.bytesAllocated < 0 {
		t.Errorf("bytesAllocated should not go negative, got %d", arena.bytesAllocated)
	}
}

func TestMinorGC_AdvancesThreshold(t *testing.T) {
	vm := newGCTestVM()
	arena := vm.arena
	arena.bytesAllocated = gcInitialThreshold + 1
	arena.MinorGC(vm)
	if arena.nextGCAt <= 0 {
		t.Errorf("expected nextGCAt to be set after MinorGC, got %d", arena.nextGCAt)
	}
	if arena.nextGCAt < arena.bytesAllocated+gcMinThresholdDelta {
		t.Errorf("expected nextGCAt >= bytesAllocated+gcMinThresholdDelta, got nextGCAt=%d bytesAllocated=%d",
			arena.nextGCAt, arena.bytesAllocated)
	}
}

func TestUpdateNextGCAt_MostlyLiveBacksOff(t *testing.T) {
	arena := newRegisterArena()
	arena.bytesAtLastGC = 0
	preGC := int64(10 * 1024 * 1024)
	postGC := int64(9 * 1024 * 1024)
	arena.updateNextGCAt(preGC, postGC)
	delta := arena.nextGCAt - postGC
	if delta < gcMinThresholdDelta {
		t.Errorf("mostly-live should produce large delta (>= min), got %d", delta)
	}
}

func TestUpdateNextGCAt_MostlyGarbageKeepsTight(t *testing.T) {
	arena := newRegisterArena()
	arena.bytesAtLastGC = 0
	preGC := int64(10 * 1024 * 1024)
	postGC := int64(1 * 1024 * 1024)
	arena.updateNextGCAt(preGC, postGC)
	delta := arena.nextGCAt - postGC
	if delta > gcMaxThresholdDelta {
		t.Errorf("mostly-garbage should produce bounded delta (<= max), got %d", delta)
	}
}

func TestUpdateNextGCAt_RespectsFloorAndCeiling(t *testing.T) {
	arena := newRegisterArena()
	arena.bytesAtLastGC = 0
	preGC := int64(100)
	postGC := int64(50)
	arena.updateNextGCAt(preGC, postGC)
	if arena.nextGCAt-postGC < gcMinThresholdDelta {
		t.Errorf("threshold delta below floor, got %d expected >= %d",
			arena.nextGCAt-postGC, gcMinThresholdDelta)
	}
}

func TestGCShouldRun_HonoursThreshold(t *testing.T) {
	arena := newRegisterArena()
	if arena.gcShouldRun() {
		t.Error("fresh arena should not trigger GC")
	}
	arena.bytesAllocated = gcInitialThreshold - 1
	if arena.gcShouldRun() {
		t.Error("arena below threshold should not trigger GC")
	}
	arena.bytesAllocated = gcInitialThreshold + 1
	if !arena.gcShouldRun() {
		t.Error("arena above threshold should trigger GC")
	}
}
