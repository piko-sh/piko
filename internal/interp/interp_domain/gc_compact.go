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
	// arenaSliceHeaderBytes is the size in bytes of a slice header tracked per arena slab
	// when reclaiming dead generations.
	arenaSliceHeaderBytes int64 = 24

	// int64BackingBytes is the size in bytes of an int64 backing slot used to compute
	// reclaimed memory for dead generations.
	int64BackingBytes int64 = 8

	// float64BackingBytes is the size in bytes of a float64 backing slot used to compute
	// reclaimed memory for dead generations.
	float64BackingBytes int64 = 8

	// stringBackingBytes is the size in bytes of a string header backing slot used to
	// compute reclaimed memory for dead generations.
	stringBackingBytes int64 = 16

	// uint64BackingBytes is the size in bytes of a uint64 backing slot used to compute
	// reclaimed memory for dead generations.
	uint64BackingBytes int64 = 8
)

// compactPhase consumes mark-phase liveness output to reclaim memory.
//
// Strategy: drop fully-dead OLD slab generations. The current- generation slabs are left
// untouched; their bytes are reused by subsequent bump allocations. This conservative
// strategy avoids the pointer-rewriting machinery that full Cheney-style compaction would
// require, at the cost of not reclaiming dead bytes within an actively-growing slab
// generation.
//
// For the typical long-running script bloat pattern (allocation pressure causes byteSlab
// to grow into oldByteSlabs[N], live data is concentrated in recent allocations),
// dropping fully-dead old generations is highly effective: most old generations contain
// zero live references shortly after they are retired.
//
// Takes state (*gcMarkState) which carries the per-slab liveness bitmaps populated during
// the mark phase.
//
// Returns the number of bytes reclaimed (the total backing capacity of all dropped old
// slabs).
func (a *RegisterArena) compactPhase(state *gcMarkState) int64 {
	if state == nil {
		return 0
	}
	var reclaimed int64
	reclaimed += compactOldByteSlabs(&a.oldByteSlabs, state.oldByteSlabLive)
	reclaimed += compactOldGenericBytes(&a.oldGenericByteSlabs, state.oldGenericBytesLive)
	reclaimed += compactOldSliceHeaders(&a.oldSliceHeaderSlabs, state.oldSliceHeaderLive)
	reclaimed += compactOldIntBackings(&a.oldIntBackings, state.oldIntBackingLive)
	reclaimed += compactOldFloatBackings(&a.oldFloatBackings, state.oldFloatBackingLive)
	reclaimed += compactOldStringBackings(&a.oldStringBackings, state.oldStringBackingLive)
	reclaimed += compactOldBoolBackings(&a.oldBoolBackings, state.oldBoolBackingLive)
	reclaimed += compactOldUintBackings(&a.oldUintBackings, state.oldUintBackingLive)
	return reclaimed
}

// compactOldByteSlabs drops byte slabs from the retention list whose corresponding
// liveness flag is false, preserving relative order of surviving slabs.
//
// Modifies *slabs in place: shifts live entries down and reslices to the live count.
// Dropped slabs become unreachable and Go GC will reclaim their backing.
//
// Takes slabs (*[][]byte) which is the retention list to compact.
// Takes live ([]bool) which is the per-slab liveness bitmap from the mark phase.
//
// Returns total bytes reclaimed across dropped slabs.
func compactOldByteSlabs(slabs *[][]byte, live []bool) int64 {
	var reclaimed int64
	writeIndex := 0
	source := *slabs
	for i, slab := range source {
		if i < len(live) && live[i] {
			source[writeIndex] = slab
			writeIndex++
			continue
		}
		reclaimed += int64(cap(slab))
		source[i] = nil
	}
	for i := writeIndex; i < len(source); i++ {
		source[i] = nil
	}
	*slabs = source[:writeIndex]
	return reclaimed
}

// compactOldGenericBytes drops generic-bytes slabs whose liveness flag is false. Thin
// wrapper over compactOldByteSlabs because the storage shape is identical.
//
// Takes slabs (*[][]byte) which is the retention list to compact.
// Takes live ([]bool) which is the per-slab liveness bitmap.
//
// Returns total bytes reclaimed across dropped slabs.
func compactOldGenericBytes(slabs *[][]byte, live []bool) int64 {
	return compactOldByteSlabs(slabs, live)
}

// compactOldSliceHeaders mirrors compactOldByteSlabs for the slice- header retention
// list. Each entry's backing capacity is (cap * element size).
//
// Takes slabs (*[][]arenaSliceHeader) which is the retention list to compact.
// Takes live ([]bool) which is the per-slab liveness bitmap.
//
// Returns total bytes reclaimed across dropped slabs.
func compactOldSliceHeaders(slabs *[][]arenaSliceHeader, live []bool) int64 {
	var reclaimed int64
	writeIndex := 0
	source := *slabs
	for i, slab := range source {
		if i < len(live) && live[i] {
			source[writeIndex] = slab
			writeIndex++
			continue
		}
		reclaimed += int64(cap(slab)) * arenaSliceHeaderBytes
		source[i] = nil
	}
	for i := writeIndex; i < len(source); i++ {
		source[i] = nil
	}
	*slabs = source[:writeIndex]
	return reclaimed
}

// compactOldIntBackings drops int64-backing slabs whose liveness flag is false. Each
// entry's backing capacity is (cap * 8 bytes).
//
// Takes slabs (*[][]int64) which is the retention list to compact.
// Takes live ([]bool) which is the per-slab liveness bitmap.
//
// Returns total bytes reclaimed across dropped slabs.
func compactOldIntBackings(slabs *[][]int64, live []bool) int64 {
	var reclaimed int64
	writeIndex := 0
	source := *slabs
	for i, slab := range source {
		if i < len(live) && live[i] {
			source[writeIndex] = slab
			writeIndex++
			continue
		}
		reclaimed += int64(cap(slab)) * int64BackingBytes
		source[i] = nil
	}
	for i := writeIndex; i < len(source); i++ {
		source[i] = nil
	}
	*slabs = source[:writeIndex]
	return reclaimed
}

// compactOldFloatBackings drops float64-backing slabs whose liveness flag is false. Each
// entry's backing capacity is (cap * 8 bytes).
//
// Takes slabs (*[][]float64) which is the retention list to compact.
// Takes live ([]bool) which is the per-slab liveness bitmap.
//
// Returns total bytes reclaimed across dropped slabs.
func compactOldFloatBackings(slabs *[][]float64, live []bool) int64 {
	var reclaimed int64
	writeIndex := 0
	source := *slabs
	for i, slab := range source {
		if i < len(live) && live[i] {
			source[writeIndex] = slab
			writeIndex++
			continue
		}
		reclaimed += int64(cap(slab)) * float64BackingBytes
		source[i] = nil
	}
	for i := writeIndex; i < len(source); i++ {
		source[i] = nil
	}
	*slabs = source[:writeIndex]
	return reclaimed
}

// compactOldStringBackings drops string-backing slabs whose liveness flag is false. Each
// entry's backing capacity is (cap * 16 bytes, the size of a Go string header).
//
// Takes slabs (*[][]string) which is the retention list to compact.
// Takes live ([]bool) which is the per-slab liveness bitmap.
//
// Returns total bytes reclaimed across dropped slabs.
func compactOldStringBackings(slabs *[][]string, live []bool) int64 {
	var reclaimed int64
	writeIndex := 0
	source := *slabs
	for i, slab := range source {
		if i < len(live) && live[i] {
			source[writeIndex] = slab
			writeIndex++
			continue
		}
		reclaimed += int64(cap(slab)) * stringBackingBytes
		source[i] = nil
	}
	for i := writeIndex; i < len(source); i++ {
		source[i] = nil
	}
	*slabs = source[:writeIndex]
	return reclaimed
}

// compactOldBoolBackings drops bool-backing slabs whose liveness flag is false. Each
// entry's backing capacity is (cap * 1 byte).
//
// Takes slabs (*[][]bool) which is the retention list to compact.
// Takes live ([]bool) which is the per-slab liveness bitmap.
//
// Returns total bytes reclaimed across dropped slabs.
func compactOldBoolBackings(slabs *[][]bool, live []bool) int64 {
	var reclaimed int64
	writeIndex := 0
	source := *slabs
	for i, slab := range source {
		if i < len(live) && live[i] {
			source[writeIndex] = slab
			writeIndex++
			continue
		}
		reclaimed += int64(cap(slab))
		source[i] = nil
	}
	for i := writeIndex; i < len(source); i++ {
		source[i] = nil
	}
	*slabs = source[:writeIndex]
	return reclaimed
}

// compactOldUintBackings drops uint64-backing slabs whose liveness flag is false. Each
// entry's backing capacity is (cap * 8 bytes).
//
// Takes slabs (*[][]uint64) which is the retention list to compact.
// Takes live ([]bool) which is the per-slab liveness bitmap.
//
// Returns total bytes reclaimed across dropped slabs.
func compactOldUintBackings(slabs *[][]uint64, live []bool) int64 {
	var reclaimed int64
	writeIndex := 0
	source := *slabs
	for i, slab := range source {
		if i < len(live) && live[i] {
			source[writeIndex] = slab
			writeIndex++
			continue
		}
		reclaimed += int64(cap(slab)) * uint64BackingBytes
		source[i] = nil
	}
	for i := writeIndex; i < len(source); i++ {
		source[i] = nil
	}
	*slabs = source[:writeIndex]
	return reclaimed
}
