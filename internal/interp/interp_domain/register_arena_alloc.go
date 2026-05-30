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
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

// AllocStringBytes bump-allocates n bytes from the arena's byte slab and returns a
// sub-slice the caller must fully write before use.
//
// The returned slice shares memory with the arena; strings created via unsafe.String
// point into this slab rather than the Go heap.
//
// Takes n (int) which is the number of bytes to allocate.
//
// Returns a byte slice of length n backed by the arena's byte slab.
func (a *RegisterArena) AllocStringBytes(n int) []byte {
	if a.byteIndex+n > len(a.byteSlab) {
		a.growByteSlab(n)
	}
	buffer := a.byteSlab[a.byteIndex : a.byteIndex+n]
	a.byteIndex += n
	a.noteAlloc(int64(n))
	return buffer
}

// AllocByteBacking bump-allocates n bytes from the arena's byte slab and returns them in
// the three-index form (len == cap == n) so user appends past cap trigger our
// arenaAppendByte helper rather than silently spilling into the next bump-allocation's
// region.
//
// Shares the byteSlab with AllocStringBytes - both functions bump the same byteIndex, so
// allocations don't overlap. A user []byte slice may end up byte-adjacent to interned
// string data in the slab; that's safe because the slab data lives until the arena is
// Reset, and the three-index form prevents user writes from leaking into the string
// region via append.
//
// For n == 0 returns nil (matching make([]byte, 0)'s nil-data behaviour; saves a slab
// probe).
//
// Takes n (int) which is the number of byte elements (the slice's capacity).
//
// Returns a []byte of length n and capacity n backed by the arena.
func (a *RegisterArena) AllocByteBacking(n int) []byte {
	if n <= 0 {
		return nil
	}
	if a.byteIndex+n > len(a.byteSlab) {
		a.growByteSlab(n)
	}
	index := a.byteIndex
	a.byteIndex += n
	a.noteAlloc(int64(n))
	return a.byteSlab[index : index+n : index+n]
}

// AllocIntBacking bump-allocates n int64 elements from the arena's intBacking slab and
// returns them as a slice with len == cap == n.
//
// The three-index form forces append-past-cap to actually require a grow (Go's append
// cannot accidentally extend into the next allocation's region). Callers using this as a
// make([]int64, len, cap) backing should re-slice via backing[:userLen:n] to expose the
// user's chosen length.
//
// For n == 0 returns nil (matching make([]int64, 0)'s nil-data behaviour; saves a slab
// probe).
//
// Takes n (int) which is the number of int64 elements (the slice's capacity).
//
// Returns an []int64 of length n and capacity n backed by the arena.
func (a *RegisterArena) AllocIntBacking(n int) []int64 {
	if n <= 0 {
		return nil
	}
	if a.intBackingIndex+n > len(a.intBackingSlab) {
		a.growIntBackingSlab(n)
	}
	index := a.intBackingIndex
	a.intBackingIndex += n
	a.noteAlloc(int64(n) * 8)
	return a.intBackingSlab[index : index+n : index+n]
}

// AllocFloatBacking is the float64 sibling of AllocIntBacking.
//
// Takes n (int) which is the number of float64 elements.
//
// Returns an []float64 of length n and capacity n backed by the arena.
func (a *RegisterArena) AllocFloatBacking(n int) []float64 {
	if n <= 0 {
		return nil
	}
	if a.floatBackingIndex+n > len(a.floatBackingSlab) {
		a.growFloatBackingSlab(n)
	}
	index := a.floatBackingIndex
	a.floatBackingIndex += n
	a.noteAlloc(int64(n) * 8)
	return a.floatBackingSlab[index : index+n : index+n]
}

// AllocStringBox bump-allocates one slot in stringBoxSlab.
//
// Used by boxStringToGeneral to avoid the per-call heap allocation that
// reflect.ValueOf(s) does internally (convTstring's mallocgc). The returned pointer is
// valid until the arena's stringBoxSlab is grown (replaced by a larger slice) or Reset()
// clears the slab. reflect.Values produced by the caller's unsafeReadOnlyValue
// construction hold this pointer as their internal storage word, and the slab stays
// GC-reachable while those values are reachable.
//
// Takes s (string) which is the string value to box.
//
// Returns a pointer to the bump-allocated string slot.
func (a *RegisterArena) AllocStringBox(s string) *string {
	if a.stringBoxIndex >= len(a.stringBoxSlab) {
		a.growStringBoxSlab()
	}
	slot := &a.stringBoxSlab[a.stringBoxIndex]
	a.stringBoxIndex++
	*slot = s
	a.noteAlloc(stringBoxBytes)
	return slot
}

// AllocIntBox bump-allocates one int64 slot. See AllocStringBox.
//
// Takes v (int64) which is the integer value to box.
//
// Returns a pointer to the bump-allocated int64 slot.
func (a *RegisterArena) AllocIntBox(v int64) *int64 {
	if a.intBoxIndex >= len(a.intBoxSlab) {
		a.growIntBoxSlab()
	}
	slot := &a.intBoxSlab[a.intBoxIndex]
	a.intBoxIndex++
	*slot = v
	a.noteAlloc(int64BoxBytes)
	return slot
}

// AllocFloatBox bump-allocates one float64 slot. See AllocStringBox.
//
// Takes v (float64) which is the floating-point value to box.
//
// Returns a pointer to the bump-allocated float64 slot.
func (a *RegisterArena) AllocFloatBox(v float64) *float64 {
	if a.floatBoxIndex >= len(a.floatBoxSlab) {
		a.growFloatBoxSlab()
	}
	slot := &a.floatBoxSlab[a.floatBoxIndex]
	a.floatBoxIndex++
	*slot = v
	a.noteAlloc(float64BoxBytes)
	return slot
}

// AllocUintBox bump-allocates one uint64 slot. See AllocStringBox.
//
// Takes v (uint64) which is the unsigned integer value to box.
//
// Returns a pointer to the bump-allocated uint64 slot.
func (a *RegisterArena) AllocUintBox(v uint64) *uint64 {
	if a.uintBoxIndex >= len(a.uintBoxSlab) {
		a.growUintBoxSlab()
	}
	slot := &a.uintBoxSlab[a.uintBoxIndex]
	a.uintBoxIndex++
	*slot = v
	a.noteAlloc(uint64BoxBytes)
	return slot
}

// AllocComplexBox bump-allocates one complex128 slot. Sibling of AllocIntBox /
// AllocFloatBox / AllocUintBox; backs reflect.Value boxing for complex128 scalars to
// replace the per-call mallocgc that reflect.ValueOf(complex128) would incur.
//
// Takes v (complex128) which is the complex value to box.
//
// Returns a pointer to the bump-allocated complex128 slot.
func (a *RegisterArena) AllocComplexBox(v complex128) *complex128 {
	if a.complexBoxIndex >= len(a.complexBoxSlab) {
		a.growComplexBoxSlab()
	}
	slot := &a.complexBoxSlab[a.complexBoxIndex]
	a.complexBoxIndex++
	*slot = v
	a.noteAlloc(complex128BoxBytes)
	return slot
}

// AllocBytes bump-allocates aligned storage from the generic byte slab.
//
// Used by arena-backed copyReflectValue for pointer-free struct and array types. The
// caller must ensure the type stored at the returned address contains no GC pointers;
// pointer-containing types must use reflect.New so the GC tracks them. Alignment of 8 is
// sufficient for every Go primitive and any struct of primitives. Larger alignments are
// not supported here; the caller falls back to reflect.New. Zero-size requests return a
// stable shared sentinel pointer (mirroring Go's runtime.zerobase pattern).
//
// Takes size (uintptr) which is the number of bytes to allocate.
// Takes align (uintptr) which is the required start alignment in bytes.
//
// Returns an unsafe.Pointer to the start of the allocated region.
func (a *RegisterArena) AllocBytes(size uintptr, align uintptr) unsafe.Pointer {
	if size == 0 {
		return zeroSizeAllocPtr
	}
	if align == 0 {
		align = 1
	}
	mask := int(align - 1)
	sizeAsInt := safeconv.Uint64ToInt(uint64(size))
	start := (a.genericBytesIndex + mask) &^ mask
	end := start + sizeAsInt
	if end > len(a.genericBytesSlab) {
		a.growGenericBytesSlab(end)
		start = (a.genericBytesIndex + mask) &^ mask
		end = start + sizeAsInt
	}
	a.genericBytesIndex = end
	a.noteAlloc(safeconv.Uint64ToInt64(uint64(size)))
	return unsafe.Pointer(&a.genericBytesSlab[start])
}

// AllocSliceHeader bump-allocates one zeroed slice-header slot.
//
// The slot's fields are zeroed before return so the caller writes (Data, Len, Cap)
// directly. The returned pointer is stable for the lifetime of the slab - it survives
// bump allocations of later headers but not arena.Reset() (which clears the slab and
// resets the bump index).
//
// Returns a pointer to the bump-allocated arenaSliceHeader slot.
//
//revive:disable-next-line:unexported-return // package-internal type stays unexported.
func (a *RegisterArena) AllocSliceHeader() *arenaSliceHeader {
	if a.sliceHeaderIndex >= len(a.sliceHeaderSlab) {
		a.growSliceHeaderSlab()
	}
	slot := &a.sliceHeaderSlab[a.sliceHeaderIndex]
	a.sliceHeaderIndex++
	*slot = arenaSliceHeader{}
	a.noteAlloc(arenaSliceHeaderBoxBytes)
	return slot
}

// AllocStringBacking is the string sibling of AllocIntBacking. The returned slice is a
// real []string sub-slice, so writes through it go via Go's normal write barrier - the
// slab can safely hold pointers (string data pointers) without unsafe.Pointer trickery.
//
// Takes n (int) which is the number of string elements.
//
// Returns an []string of length n and capacity n backed by the arena.
func (a *RegisterArena) AllocStringBacking(n int) []string {
	if n <= 0 {
		return nil
	}
	if a.stringBackingIndex+n > len(a.stringBackingSlab) {
		a.growStringBackingSlab(n)
	}
	index := a.stringBackingIndex
	a.stringBackingIndex += n
	a.noteAlloc(int64(n) * 16)
	return a.stringBackingSlab[index : index+n : index+n]
}

// AllocBoolBacking is the bool sibling of AllocIntBacking.
//
// Takes n (int) which is the number of bool elements.
//
// Returns an []bool of length n and capacity n backed by the arena.
func (a *RegisterArena) AllocBoolBacking(n int) []bool {
	if n <= 0 {
		return nil
	}
	if a.boolBackingIndex+n > len(a.boolBackingSlab) {
		a.growBoolBackingSlab(n)
	}
	index := a.boolBackingIndex
	a.boolBackingIndex += n
	a.noteAlloc(int64(n))
	return a.boolBackingSlab[index : index+n : index+n]
}

// AllocUintBacking is the uint64 sibling of AllocIntBacking.
//
// Takes n (int) which is the number of uint64 elements.
//
// Returns an []uint64 of length n and capacity n backed by the arena.
func (a *RegisterArena) AllocUintBacking(n int) []uint64 {
	if n <= 0 {
		return nil
	}
	if a.uintBackingIndex+n > len(a.uintBackingSlab) {
		a.growUintBackingSlab(n)
	}
	index := a.uintBackingIndex
	a.uintBackingIndex += n
	a.noteAlloc(int64(n) * 8)
	return a.uintBackingSlab[index : index+n : index+n]
}
