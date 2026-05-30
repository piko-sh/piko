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
	"fmt"
	"reflect"
	"unsafe"
)

const (
	// arenaSizeInt64 is the byte size of a single int64 backing slot.
	arenaSizeInt64 uint64 = uint64(unsafe.Sizeof(int64(0)))

	// arenaSizeFloat64 is the byte size of a single float64 backing slot.
	arenaSizeFloat64 uint64 = uint64(unsafe.Sizeof(float64(0)))

	// arenaSizeString is the byte size of a single string header.
	arenaSizeString uint64 = uint64(unsafe.Sizeof(""))

	// arenaSizeBool is the byte size of a single bool backing slot.
	arenaSizeBool uint64 = uint64(unsafe.Sizeof(false))

	// arenaSizeUint64 is the byte size of a single uint64 backing slot.
	arenaSizeUint64 uint64 = uint64(unsafe.Sizeof(uint64(0)))

	// arenaSizeSliceHeader is the byte size of a single arenaSliceHeader.
	arenaSizeSliceHeader uint64 = uint64(unsafe.Sizeof(arenaSliceHeader{}))
)

// arenaBudgetLimit returns the effective per-Execute byte budget for this arena. Zero on
// the arena field selects defaultMaxArenaBytes.
//
// Returns the budget in bytes.
func (a *RegisterArena) arenaBudgetLimit() uint64 {
	if a.maxArenaBytes == 0 {
		return defaultMaxArenaBytes
	}
	return a.maxArenaBytes
}

// chargeArenaAllocation updates the live-bytes counter on slab replace.
//
// Adds the delta between the new and prior capacity to the counter so it tracks the
// arena's current working-set capacity rather than cumulative historical allocations: a
// single slab grown from 0 -> N via halving steps charges N, not 2N - 1. When the running
// total would exceed the configured budget, the helper first asks the owning VM to drive
// a MinorGC so any dead retained old slabs can be dropped from the working set; only if
// the figure remains above budget after that fallback does it panic with
// errArenaBudgetExceeded. The VM execute boundary recovers and surfaces the panic as a
// normal evaluation failure.
//
// Takes newCapacityBytes (uint64) which is the byte capacity of the freshly installed
// slab.
// Takes oldCapacityBytes (uint64) which is the byte capacity of the slab the new one
// replaces (zero for a first allocation).
//
// Panics when the running total exceeds the budget after a MinorGC fallback, with
// errArenaBudgetExceeded.
func (a *RegisterArena) chargeArenaAllocation(newCapacityBytes, oldCapacityBytes uint64) {
	budget := a.arenaBudgetLimit()
	if newCapacityBytes <= oldCapacityBytes {
		if oldCapacityBytes > a.totalAllocatedBytes {
			a.totalAllocatedBytes = 0
		} else {
			a.totalAllocatedBytes -= oldCapacityBytes - newCapacityBytes
		}
		return
	}
	delta := newCapacityBytes - oldCapacityBytes
	if a.totalAllocatedBytes+delta < a.totalAllocatedBytes {
		panic(fmt.Errorf("arena: overflow charging %d bytes: %w", delta, errArenaBudgetExceeded))
	}
	a.totalAllocatedBytes += delta
	if a.totalAllocatedBytes <= budget {
		return
	}
	if a.ownerVM != nil {
		a.MinorGC(a.ownerVM)
		if a.totalAllocatedBytes <= budget {
			return
		}
	}
	panic(fmt.Errorf("arena: %d bytes exceeds budget %d: %w", a.totalAllocatedBytes, budget, errArenaBudgetExceeded))
}

// trimOldByteSlabs drops the oldest retained byte slab when the retention list exceeds
// maxRetainedOldSlabsPerType so the Go GC can reclaim memory once no live string holds a
// pointer into it.
func (a *RegisterArena) trimOldByteSlabs() {
	if len(a.oldByteSlabs) > maxRetainedOldSlabsPerType {
		a.oldByteSlabs = a.oldByteSlabs[1:]
	}
}

// trimOldGenericByteSlabs is the generic-byte sibling of trimOldByteSlabs.
func (a *RegisterArena) trimOldGenericByteSlabs() {
	if len(a.oldGenericByteSlabs) > maxRetainedOldSlabsPerType {
		a.oldGenericByteSlabs = a.oldGenericByteSlabs[1:]
	}
}

// trimOldSliceHeaderSlabs is the slice-header sibling of trimOldByteSlabs.
func (a *RegisterArena) trimOldSliceHeaderSlabs() {
	if len(a.oldSliceHeaderSlabs) > maxRetainedOldSlabsPerType {
		a.oldSliceHeaderSlabs = a.oldSliceHeaderSlabs[1:]
	}
}

// trimOldIntBackings is the int64-backing sibling of trimOldByteSlabs.
func (a *RegisterArena) trimOldIntBackings() {
	if len(a.oldIntBackings) > maxRetainedOldSlabsPerType {
		a.oldIntBackings = a.oldIntBackings[1:]
	}
}

// trimOldFloatBackings is the float64-backing sibling of trimOldByteSlabs.
func (a *RegisterArena) trimOldFloatBackings() {
	if len(a.oldFloatBackings) > maxRetainedOldSlabsPerType {
		a.oldFloatBackings = a.oldFloatBackings[1:]
	}
}

// trimOldStringBackings is the string-backing sibling of trimOldByteSlabs.
func (a *RegisterArena) trimOldStringBackings() {
	if len(a.oldStringBackings) > maxRetainedOldSlabsPerType {
		a.oldStringBackings = a.oldStringBackings[1:]
	}
}

// trimOldBoolBackings is the bool-backing sibling of trimOldByteSlabs.
func (a *RegisterArena) trimOldBoolBackings() {
	if len(a.oldBoolBackings) > maxRetainedOldSlabsPerType {
		a.oldBoolBackings = a.oldBoolBackings[1:]
	}
}

// trimOldUintBackings is the uint64-backing sibling of trimOldByteSlabs.
func (a *RegisterArena) trimOldUintBackings() {
	if len(a.oldUintBackings) > maxRetainedOldSlabsPerType {
		a.oldUintBackings = a.oldUintBackings[1:]
	}
}

// growStringBoxSlab doubles stringBoxSlab capacity.
//
// Uses a 64-slot starting size when empty. The retired slab is dropped from the arena
// root - existing reflect.Values still holding pointers into it keep it alive via their
// own pointer fields, so no oldStringBoxes retention list is needed (unlike the typed
// backing slabs whose lifetime is bound by user-visible slice headers crossing Execute
// boundaries).
func (a *RegisterArena) growStringBoxSlab() {
	newCap := max(len(a.stringBoxSlab)*2, initialBoxSlabCapacity)
	a.stringBoxSlab = make([]string, newCap)
	a.stringBoxIndex = 0
}

// growIntBoxSlab doubles intBoxSlab capacity (see growStringBoxSlab).
func (a *RegisterArena) growIntBoxSlab() {
	newCap := max(len(a.intBoxSlab)*2, initialBoxSlabCapacity)
	a.intBoxSlab = make([]int64, newCap)
	a.intBoxIndex = 0
}

// growFloatBoxSlab doubles floatBoxSlab capacity (see growStringBoxSlab).
func (a *RegisterArena) growFloatBoxSlab() {
	newCap := max(len(a.floatBoxSlab)*2, initialBoxSlabCapacity)
	a.floatBoxSlab = make([]float64, newCap)
	a.floatBoxIndex = 0
}

// growUintBoxSlab doubles uintBoxSlab capacity (see growStringBoxSlab).
func (a *RegisterArena) growUintBoxSlab() {
	newCap := max(len(a.uintBoxSlab)*2, initialBoxSlabCapacity)
	a.uintBoxSlab = make([]uint64, newCap)
	a.uintBoxIndex = 0
}

// growComplexBoxSlab doubles complexBoxSlab capacity (see growStringBoxSlab).
func (a *RegisterArena) growComplexBoxSlab() {
	newCap := max(len(a.complexBoxSlab)*2, initialBoxSlabCapacity)
	a.complexBoxSlab = make([]complex128, newCap)
	a.complexBoxIndex = 0
}

// growGenericBytesSlab doubles genericBytesSlab to at least minCap bytes.
//
// Resets the bump pointer and appends the retired slab to oldGenericByteSlabs so
// ownsBytePointer can recognise pointers into a retired slab. This is essential when
// arena-backed reflect.Values cross an escape boundary after the slab grew - without the
// retention list, ownsBytePointer would false-negative and the escape-copy guard would
// skip a value that still needs materialising. Existing reflect.Values keep the retired
// slab alive through GC tracing of their unsafe.Pointer ptr field; the retention slice
// keeps it discoverable by the arena itself.
//
// Takes minCap (int) which is the minimum byte capacity the new slab must satisfy.
func (a *RegisterArena) growGenericBytesSlab(minCap int) {
	oldCap := uint64(cap(a.genericBytesSlab))
	if len(a.genericBytesSlab) > 0 {
		a.oldGenericByteSlabs = append(a.oldGenericByteSlabs, a.genericBytesSlab)
		a.trimOldGenericByteSlabs()
	}
	newCap := max(max(len(a.genericBytesSlab)*2, minCap), initialGenericBytesCapacity)
	a.genericBytesSlab = make([]byte, newCap)
	a.genericBytesIndex = 0
	a.chargeArenaAllocation(uint64(cap(a.genericBytesSlab)), oldCap)
}

// growSliceHeaderSlab doubles sliceHeaderSlab capacity.
//
// Appends the retired slab to oldSliceHeaderSlabs so reflect.Values whose ptr is an
// arenaSliceHeader in a grown-away slab remain reachable from the arena root. Mirrors the
// retention pattern used by growGenericBytesSlab and growByteSlab.
func (a *RegisterArena) growSliceHeaderSlab() {
	oldCap := uint64(cap(a.sliceHeaderSlab)) * arenaSizeSliceHeader
	if len(a.sliceHeaderSlab) > 0 {
		a.oldSliceHeaderSlabs = append(a.oldSliceHeaderSlabs, a.sliceHeaderSlab)
		a.trimOldSliceHeaderSlabs()
	}
	newCap := max(len(a.sliceHeaderSlab)*2, initialSliceHeaderCapacity)
	a.sliceHeaderSlab = make([]arenaSliceHeader, newCap)
	a.sliceHeaderIndex = 0
	a.chargeArenaAllocation(uint64(cap(a.sliceHeaderSlab))*arenaSizeSliceHeader, oldCap)
}

// growFrameStack grows the frameSlab, callInfoBasesSlab, and dispatchSavesSlab together
// to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity for the slabs.
//
// Returns the new frame, call-info base, and dispatch save slabs.
//
//go:noinline
func (a *RegisterArena) growFrameStack(minCap int) ([]callFrame, []uintptr, []asmDispatchSave) {
	newCap := max(len(a.frameSlab)*2, minCap)

	newFrames := make([]callFrame, newCap)
	copy(newFrames, a.frameSlab)
	a.frameSlab = newFrames

	newCI := make([]uintptr, newCap)
	copy(newCI, a.callInfoBasesSlab)
	a.callInfoBasesSlab = newCI

	newDisp := make([]asmDispatchSave, newCap)
	copy(newDisp, a.dispatchSavesSlab)
	a.dispatchSavesSlab = newDisp

	return newFrames, newCI, newDisp
}

// growUpvalueCellSlab grows the upvalueCell slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
//
//go:noinline
func (a *RegisterArena) growUpvalueCellSlab(minCap int) {
	newCap := max(len(a.upvalueCellSlab)*2, minCap)
	newSlab := make([]upvalueCell, newCap)
	copy(newSlab, a.upvalueCellSlab)
	a.upvalueCellSlab = newSlab
}

// growUpvalueRefSlab grows the upvalue ref slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
//
//go:noinline
func (a *RegisterArena) growUpvalueRefSlab(minCap int) {
	newCap := max(len(a.upvalueReferenceSlab)*2, minCap)
	newSlab := make([]upvalue, newCap)
	copy(newSlab, a.upvalueReferenceSlab)
	a.upvalueReferenceSlab = newSlab
}

// growByteSlab allocates a new byte slab, preserving the old one in oldByteSlabs so that
// existing strings pointing into it remain valid.
//
// Takes minExtra (int) which is the minimum number of bytes the new slab must hold.
//
//go:noinline
func (a *RegisterArena) growByteSlab(minExtra int) {
	oldCap := uint64(cap(a.byteSlab))
	a.oldByteSlabs = append(a.oldByteSlabs, a.byteSlab)
	a.trimOldByteSlabs()
	newSize := max(len(a.byteSlab)*2, minExtra)
	a.byteSlab = make([]byte, newSize)
	a.byteIndex = 0
	a.chargeArenaAllocation(uint64(cap(a.byteSlab)), oldCap)
}

// growIntBackingSlab grows the int64 backing slab.
//
// The retired slab is kept in oldIntBackings so live []int64 slices the user holds (e.g.
// from prior make/append calls) remain valid.
//
// Takes minExtra (int) which is the minimum number of elements the new slab must hold.
//
//go:noinline
func (a *RegisterArena) growIntBackingSlab(minExtra int) {
	oldCap := uint64(cap(a.intBackingSlab)) * arenaSizeInt64
	a.oldIntBackings = append(a.oldIntBackings, a.intBackingSlab)
	a.trimOldIntBackings()
	newSize := max(len(a.intBackingSlab)*2, minExtra)
	a.intBackingSlab = make([]int64, newSize)
	a.intBackingIndex = 0
	a.chargeArenaAllocation(uint64(cap(a.intBackingSlab))*arenaSizeInt64, oldCap)
}

// growFloatBackingSlab is the float64 sibling of growIntBackingSlab.
//
// Takes minExtra (int) which is the minimum number of elements the new slab must hold.
//
//go:noinline
func (a *RegisterArena) growFloatBackingSlab(minExtra int) {
	oldCap := uint64(cap(a.floatBackingSlab)) * arenaSizeFloat64
	a.oldFloatBackings = append(a.oldFloatBackings, a.floatBackingSlab)
	a.trimOldFloatBackings()
	newSize := max(len(a.floatBackingSlab)*2, minExtra)
	a.floatBackingSlab = make([]float64, newSize)
	a.floatBackingIndex = 0
	a.chargeArenaAllocation(uint64(cap(a.floatBackingSlab))*arenaSizeFloat64, oldCap)
}

// growStringBackingSlab is the string sibling of growIntBackingSlab.
//
// Takes minExtra (int) which is the minimum number of elements the new slab must hold.
//
//go:noinline
func (a *RegisterArena) growStringBackingSlab(minExtra int) {
	oldCap := uint64(cap(a.stringBackingSlab)) * arenaSizeString
	a.oldStringBackings = append(a.oldStringBackings, a.stringBackingSlab)
	a.trimOldStringBackings()
	newSize := max(len(a.stringBackingSlab)*2, minExtra)
	a.stringBackingSlab = make([]string, newSize)
	a.stringBackingIndex = 0
	a.chargeArenaAllocation(uint64(cap(a.stringBackingSlab))*arenaSizeString, oldCap)
}

// growBoolBackingSlab is the bool sibling of growIntBackingSlab.
//
// Takes minExtra (int) which is the minimum number of elements the new slab must hold.
//
//go:noinline
func (a *RegisterArena) growBoolBackingSlab(minExtra int) {
	oldCap := uint64(cap(a.boolBackingSlab)) * arenaSizeBool
	a.oldBoolBackings = append(a.oldBoolBackings, a.boolBackingSlab)
	a.trimOldBoolBackings()
	newSize := max(len(a.boolBackingSlab)*2, minExtra)
	a.boolBackingSlab = make([]bool, newSize)
	a.boolBackingIndex = 0
	a.chargeArenaAllocation(uint64(cap(a.boolBackingSlab))*arenaSizeBool, oldCap)
}

// growUintBackingSlab is the uint64 sibling of growIntBackingSlab.
//
// Takes minExtra (int) which is the minimum number of elements the new slab must hold.
//
//go:noinline
func (a *RegisterArena) growUintBackingSlab(minExtra int) {
	oldCap := uint64(cap(a.uintBackingSlab)) * arenaSizeUint64
	a.oldUintBackings = append(a.oldUintBackings, a.uintBackingSlab)
	a.trimOldUintBackings()
	newSize := max(len(a.uintBackingSlab)*2, minExtra)
	a.uintBackingSlab = make([]uint64, newSize)
	a.uintBackingIndex = 0
	a.chargeArenaAllocation(uint64(cap(a.uintBackingSlab))*arenaSizeUint64, oldCap)
}

// growSlabs handles the rare case where arena slabs need to grow.
//
// Takes counts (typedSlabCounts) which is the per-bank request size driving the grow
// decision. Each bank's slab is grown only when its current allocation index plus the
// requested count exceeds its existing capacity, leaving untouched banks alone.
//
//go:noinline
func (a *RegisterArena) growSlabs(counts typedSlabCounts) {
	if a.intIndex+counts.ints > len(a.intSlab) {
		a.growIntSlab(a.intIndex + counts.ints)
	}
	if a.floatIndex+counts.floats > len(a.floatSlab) {
		a.growFloatSlab(a.floatIndex + counts.floats)
	}
	if a.stringIndex+counts.strings > len(a.stringSlab) {
		a.growStringSlab(a.stringIndex + counts.strings)
	}
	if a.generalIndex+counts.generals > len(a.generalSlab) {
		a.growGeneralSlab(a.generalIndex + counts.generals)
	}
	if a.boolIndex+counts.bools > len(a.boolSlab) {
		a.growBoolSlab(a.boolIndex + counts.bools)
	}
	if a.uintIndex+counts.uints > len(a.uintSlab) {
		a.growUintSlab(a.uintIndex + counts.uints)
	}
	if a.complexIndex+counts.complexes > len(a.complexSlab) {
		a.growComplexSlab(a.complexIndex + counts.complexes)
	}
	if a.slicesIntIndex+counts.slicesInts > len(a.slicesIntSlab) {
		a.growSlicesIntSlab(a.slicesIntIndex + counts.slicesInts)
	}
	if a.slicesFloatIndex+counts.slicesFloats > len(a.slicesFloatSlab) {
		a.growSlicesFloatSlab(a.slicesFloatIndex + counts.slicesFloats)
	}
	if a.slicesStringIndex+counts.slicesStrings > len(a.slicesStringSlab) {
		a.growSlicesStringSlab(a.slicesStringIndex + counts.slicesStrings)
	}
	if a.slicesBoolIndex+counts.slicesBools > len(a.slicesBoolSlab) {
		a.growSlicesBoolSlab(a.slicesBoolIndex + counts.slicesBools)
	}
	if a.slicesUintIndex+counts.slicesUints > len(a.slicesUintSlab) {
		a.growSlicesUintSlab(a.slicesUintIndex + counts.slicesUints)
	}
	if a.slicesByteIndex+counts.slicesBytes > len(a.slicesByteSlab) {
		a.growSlicesByteSlab(a.slicesByteIndex + counts.slicesBytes)
	}
}

// growSlicesIntSlab grows the slicesInt slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growSlicesIntSlab(minCap int) {
	newCap := max(len(a.slicesIntSlab)*2, minCap)
	newSlab := make([][]int64, newCap)
	copy(newSlab, a.slicesIntSlab)
	a.slicesIntSlab = newSlab
}

// growSlicesFloatSlab grows the slicesFloat slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growSlicesFloatSlab(minCap int) {
	newCap := max(len(a.slicesFloatSlab)*2, minCap)
	newSlab := make([][]float64, newCap)
	copy(newSlab, a.slicesFloatSlab)
	a.slicesFloatSlab = newSlab
}

// growSlicesStringSlab grows the slicesString slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growSlicesStringSlab(minCap int) {
	newCap := max(len(a.slicesStringSlab)*2, minCap)
	newSlab := make([][]string, newCap)
	copy(newSlab, a.slicesStringSlab)
	a.slicesStringSlab = newSlab
}

// growSlicesBoolSlab grows the slicesBool slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growSlicesBoolSlab(minCap int) {
	newCap := max(len(a.slicesBoolSlab)*2, minCap)
	newSlab := make([][]bool, newCap)
	copy(newSlab, a.slicesBoolSlab)
	a.slicesBoolSlab = newSlab
}

// growSlicesUintSlab grows the slicesUint slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growSlicesUintSlab(minCap int) {
	newCap := max(len(a.slicesUintSlab)*2, minCap)
	newSlab := make([][]uint64, newCap)
	copy(newSlab, a.slicesUintSlab)
	a.slicesUintSlab = newSlab
}

// growSlicesByteSlab grows the slicesByte slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growSlicesByteSlab(minCap int) {
	newCap := max(len(a.slicesByteSlab)*2, minCap)
	newSlab := make([][]byte, newCap)
	copy(newSlab, a.slicesByteSlab)
	a.slicesByteSlab = newSlab
}

// growIntSlab grows the int slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growIntSlab(minCap int) {
	newCap := max(len(a.intSlab)*2, minCap)
	newSlab := make([]int64, newCap)
	copy(newSlab, a.intSlab)
	a.intSlab = newSlab
}

// growFloatSlab grows the float slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growFloatSlab(minCap int) {
	newCap := max(len(a.floatSlab)*2, minCap)
	newSlab := make([]float64, newCap)
	copy(newSlab, a.floatSlab)
	a.floatSlab = newSlab
}

// growStringSlab grows the string slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growStringSlab(minCap int) {
	newCap := max(len(a.stringSlab)*2, minCap)
	newSlab := make([]string, newCap)
	copy(newSlab, a.stringSlab)
	a.stringSlab = newSlab
}

// growGeneralSlab grows the general slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growGeneralSlab(minCap int) {
	newCap := max(len(a.generalSlab)*2, minCap)
	newSlab := make([]reflect.Value, newCap)
	copy(newSlab, a.generalSlab)
	a.generalSlab = newSlab
}

// growBoolSlab grows the bool slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growBoolSlab(minCap int) {
	newCap := max(len(a.boolSlab)*2, minCap)
	newSlab := make([]bool, newCap)
	copy(newSlab, a.boolSlab)
	a.boolSlab = newSlab
}

// growUintSlab grows the uint slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growUintSlab(minCap int) {
	newCap := max(len(a.uintSlab)*2, minCap)
	newSlab := make([]uint64, newCap)
	copy(newSlab, a.uintSlab)
	a.uintSlab = newSlab
}

// growComplexSlab grows the complex slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growComplexSlab(minCap int) {
	newCap := max(len(a.complexSlab)*2, minCap)
	newSlab := make([]complex128, newCap)
	copy(newSlab, a.complexSlab)
	a.complexSlab = newSlab
}
