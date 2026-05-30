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

//go:build !safe && !(js && wasm)

package interp_domain

import (
	"strconv"
	"unicode/utf8"
	"unsafe"

	"piko.sh/piko/internal/mem"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// arenaUsesUnsafeSlabs is true in the unsafe build, signalling to white-box unit tests
	// that the arena's slab-internal fields and methods (byteSlab, sliceHeaderSlab,
	// byteIndex, oldByteSlabs, growByteSlab, and so on) are real and may be exercised.
	arenaUsesUnsafeSlabs = true

	// itoaMaxDigitsBase10 is the worst-case byte length of an int64 formatted in base 10
	// (the longest is "-9223372036854775808", 20 bytes including the sign).
	itoaMaxDigitsBase10 = 20

	// formatIntMaxDigitsBase2 is the worst-case byte length of an int64 formatted in any
	// base accepted by strconv.FormatInt. Base 2 is the widest at 64 binary digits, plus 1
	// for the sign, equals 65.
	formatIntMaxDigitsBase2 = 65

	// decimalRadix is the base passed to strconv.AppendInt when formatting integers as
	// standard decimal strings.
	decimalRadix = 10
)

// ownsString reports whether s points into the arena's current or previous byte slabs.
// Used to decide whether a string must be cloned before it can safely outlive the arena.
//
// Takes s (string) which is the string to test.
//
// Returns true when s is backed by the arena's byte slabs.
func (a *RegisterArena) ownsString(s string) bool {
	if len(s) == 0 {
		return false
	}
	pointer := uintptr(unsafe.Pointer(unsafe.StringData(s)))
	if len(a.byteSlab) > 0 {
		base := uintptr(unsafe.Pointer(&a.byteSlab[0]))
		if pointer >= base && pointer < base+uintptr(len(a.byteSlab)) {
			return true
		}
	}
	for _, slab := range a.oldByteSlabs {
		if len(slab) == 0 {
			continue
		}
		base := uintptr(unsafe.Pointer(&slab[0]))
		if pointer >= base && pointer < base+uintptr(len(slab)) {
			return true
		}
	}
	return false
}

// ownsBytePointer reports whether p falls inside an arena byte slab.
//
// Covers either the current generic byte slab or any retired slab kept alive in
// oldGenericByteSlabs after a grow. Mirrors ownsString for the string-backed byteSlab;
// required by materialiseArenaValue so escape-copy guards can decide whether a
// reflect.Value's storage will outlive the local register lifetime without
// false-negativing on values that landed in a grown-away slab.
//
// Takes p (unsafe.Pointer) which is the storage pointer to test. Typically the .ptr field
// of a flagAddr|flagIndir reflect.Value, extracted via reflectValuePtr.
//
// Returns true when p is in the arena's pointer-free byte slabs.
func (a *RegisterArena) ownsBytePointer(p unsafe.Pointer) bool {
	if p == nil {
		return false
	}
	pointer := uintptr(p)
	if len(a.genericBytesSlab) > 0 {
		base := uintptr(unsafe.Pointer(&a.genericBytesSlab[0]))
		if pointer >= base && pointer < base+uintptr(len(a.genericBytesSlab)) {
			return true
		}
	}
	for _, slab := range a.oldGenericByteSlabs {
		if len(slab) == 0 {
			continue
		}
		base := uintptr(unsafe.Pointer(&slab[0]))
		if pointer >= base && pointer < base+uintptr(len(slab)) {
			return true
		}
	}
	return false
}

// ownsSliceHeaderPointer reports whether p references an arenaSliceHeader slot in the
// current or any retired slice-header slab. Used by materialiseArenaValue when the
// reflect.Value's .ptr is the arenaSliceHeader produced by copyReflectValueArena /
// appendGenericFastPath / arenaMakeStructSliceBacking and the value is escaping the
// arena's lifetime.
//
// Takes p (unsafe.Pointer) which is the slice header pointer to test.
//
// Returns true when p falls inside any arena slice-header slab.
func (a *RegisterArena) ownsSliceHeaderPointer(p unsafe.Pointer) bool {
	if p == nil {
		return false
	}
	pointer := uintptr(p)
	if len(a.sliceHeaderSlab) > 0 {
		base := uintptr(unsafe.Pointer(&a.sliceHeaderSlab[0]))
		stride := unsafe.Sizeof(a.sliceHeaderSlab[0])
		if pointer >= base && pointer < base+uintptr(len(a.sliceHeaderSlab))*stride {
			return true
		}
	}
	for _, slab := range a.oldSliceHeaderSlabs {
		if len(slab) == 0 {
			continue
		}
		base := uintptr(unsafe.Pointer(&slab[0]))
		stride := unsafe.Sizeof(slab[0])
		if pointer >= base && pointer < base+uintptr(len(slab))*stride {
			return true
		}
	}
	return false
}

// isStringAtSlabTail reports whether the end of s coincides with the current byte-slab
// write position, enabling in-place extension when appending to the
// most-recently-allocated arena string.
//
// Takes s (string) which is the string to test.
//
// Returns true when s ends exactly at the current slab allocation pointer.
func (a *RegisterArena) isStringAtSlabTail(s string) bool {
	if len(s) == 0 || len(a.byteSlab) == 0 {
		return false
	}
	sEnd := uintptr(unsafe.Pointer(unsafe.StringData(s))) + uintptr(len(s))
	slabPos := uintptr(unsafe.Pointer(&a.byteSlab[0])) + safeconv.IntToUintptr(a.byteIndex)
	return sEnd == slabPos
}

// arenaConcatString concatenates a and b, bump-allocating the result into the arena's
// byte slab. When a ends at the slab tail, the bytes of b are appended in place without
// copying a.
//
// Takes arena (*RegisterArena) which provides the byte slab.
// Takes a (string) which is the left operand.
// Takes b (string) which is the right operand.
//
// Returns the concatenated string backed by the arena.
func arenaConcatString(arena *RegisterArena, a, b string) string {
	n := len(a) + len(b)
	if n == 0 {
		return ""
	}
	if len(a) > 0 && arena.isStringAtSlabTail(a) && arena.byteIndex+len(b) <= len(arena.byteSlab) {
		buffer := arena.AllocStringBytes(len(b))
		copy(buffer, b)
		return unsafe.String(unsafe.StringData(a), n)
	}
	buffer := arena.AllocStringBytes(n)
	copy(buffer, a)
	copy(buffer[len(a):], b)
	return mem.String(buffer)
}

// arenaConcatRuneString appends rune r to string s using bump allocation. When s ends at
// the slab tail, the rune bytes are written in place without copying s.
//
// Takes arena (*RegisterArena) which provides the byte slab.
// Takes s (string) which is the base string.
// Takes r (rune) which is the rune to append.
//
// Returns the resulting string backed by the arena.
func arenaConcatRuneString(arena *RegisterArena, s string, r rune) string {
	runeLen := utf8.RuneLen(r)
	if runeLen < 0 {
		runeLen = utf8.RuneLen(utf8.RuneError)
	}
	n := len(s) + runeLen
	if n == runeLen {
		return arenaRuneToString(arena, r)
	}
	if arena.isStringAtSlabTail(s) && arena.byteIndex+runeLen <= len(arena.byteSlab) {
		buffer := arena.AllocStringBytes(runeLen)
		utf8.EncodeRune(buffer, r)
		return unsafe.String(unsafe.StringData(s), n)
	}
	buffer := arena.AllocStringBytes(n)
	copy(buffer, s)
	utf8.EncodeRune(buffer[len(s):], r)
	return mem.String(buffer)
}

// arenaRuneToString converts rune r to a string using bump allocation into the arena's
// byte slab.
//
// Takes arena (*RegisterArena) which provides the byte slab.
// Takes r (rune) which is the rune to convert.
//
// Returns the single-rune string backed by the arena.
func arenaRuneToString(arena *RegisterArena, r rune) string {
	n := utf8.RuneLen(r)
	if n < 0 {
		n = utf8.RuneLen(utf8.RuneError)
	}
	buffer := arena.AllocStringBytes(n)
	utf8.EncodeRune(buffer, r)
	return mem.String(buffer)
}

// arenaBytesToString converts byte slice b to a string using bump allocation into the
// arena's byte slab.
//
// Takes arena (*RegisterArena) which provides the byte slab.
// Takes b ([]byte) which is the byte slice to convert.
//
// Returns the resulting string backed by the arena.
func arenaBytesToString(arena *RegisterArena, b []byte) string {
	n := len(b)
	if n == 0 {
		return ""
	}
	buffer := arena.AllocStringBytes(n)
	copy(buffer, b)
	return mem.String(buffer)
}

// arenaItoaString formats x in base 10 as a string backed by the arena's byte slab,
// avoiding the heap allocation that strconv.Itoa would otherwise perform.
//
// The implementation over-allocates the worst-case 20 bytes from the byte slab, writes
// the decimal representation into that buffer via strconv.AppendInt, then rewinds the
// arena's byteIndex to give back the unused bytes so the next allocator call starts
// immediately after the formatted string. The returned string is a mem.String view into
// the slab, valid for the arena's lifetime.
//
// Takes arena (*RegisterArena) which provides the byte slab.
// Takes x (int64) which is the integer to format.
//
// Returns the resulting decimal string backed by the arena.
//
//go:nosplit
func arenaItoaString(arena *RegisterArena, x int64) string {
	buffer := arena.AllocStringBytes(itoaMaxDigitsBase10)
	written := strconv.AppendInt(buffer[:0], x, decimalRadix)
	arena.byteIndex -= itoaMaxDigitsBase10 - len(written)
	return mem.String(written)
}

// arenaFormatIntString formats x in the given base as a string backed by the arena's byte
// slab. Mirrors arenaItoaString but reserves the worst-case 65 bytes (base 2 of a 64-bit
// signed integer) before rewinding to the actual written length.
//
// Takes arena (*RegisterArena) which provides the byte slab.
// Takes x (int64) which is the integer to format.
// Takes base (int) which is the radix passed to strconv.AppendInt.
//
// Returns the resulting string backed by the arena.
//
//go:nosplit
func arenaFormatIntString(arena *RegisterArena, x int64, base int) string {
	buffer := arena.AllocStringBytes(formatIntMaxDigitsBase2)
	written := strconv.AppendInt(buffer[:0], x, base)
	arena.byteIndex -= formatIntMaxDigitsBase2 - len(written)
	return mem.String(written)
}

// arenaAppendInt appends x to an arena-backed []int64.
//
// When the slice has spare capacity, it delegates to Go's builtin append (in-place write,
// zero allocation). When the slice is full, it bump-allocates a 2x-sized backing in the
// arena, copies the old elements, and returns the grown slice, avoiding the mallocgc that
// Go's normal append-grow would trigger.
//
// Takes arena (*RegisterArena) which provides the backing slab.
// Takes s ([]int64) which is the slice to append to.
// Takes x (int64) which is the value to append.
//
// Returns the resulting slice, either reusing s (cap available) or pointing at a fresh
// arena-backed backing.
func arenaAppendInt(arena *RegisterArena, s []int64, x int64) []int64 {
	if len(s) < cap(s) {
		return append(s, x)
	}
	newCap := max(2*cap(s), len(s)+1)
	backing := arena.AllocIntBacking(newCap)
	copy(backing, s)
	backing = backing[:len(s)+1]
	backing[len(s)] = x
	return backing
}

// arenaAppendFloat is the float64 sibling of arenaAppendInt.
//
// Takes arena (*RegisterArena) which provides the backing slab.
// Takes s ([]float64) which is the slice to append to.
// Takes x (float64) which is the value to append.
//
// Returns the resulting slice, either reusing s or pointing at a fresh arena-backed
// backing.
func arenaAppendFloat(arena *RegisterArena, s []float64, x float64) []float64 {
	if len(s) < cap(s) {
		return append(s, x)
	}
	newCap := max(2*cap(s), len(s)+1)
	backing := arena.AllocFloatBacking(newCap)
	copy(backing, s)
	backing = backing[:len(s)+1]
	backing[len(s)] = x
	return backing
}

// arenaAppendString is the string sibling of arenaAppendInt.
//
// Writes to the arena-backed slot go through the normal Go store barrier because the
// underlying allocation is a real []string sub-slice.
//
// Takes arena (*RegisterArena) which provides the backing slab.
// Takes s ([]string) which is the slice to append to.
// Takes x (string) which is the value to append.
//
// Returns the resulting slice, either reusing s or pointing at a fresh arena-backed
// backing.
func arenaAppendString(arena *RegisterArena, s []string, x string) []string {
	if len(s) < cap(s) {
		return append(s, x)
	}
	newCap := max(2*cap(s), len(s)+1)
	backing := arena.AllocStringBacking(newCap)
	copy(backing, s)
	backing = backing[:len(s)+1]
	backing[len(s)] = x
	return backing
}

// arenaAppendBool is the bool sibling of arenaAppendInt.
//
// Takes arena (*RegisterArena) which provides the backing slab.
// Takes s ([]bool) which is the slice to append to.
// Takes x (bool) which is the value to append.
//
// Returns the resulting slice, either reusing s or pointing at a fresh arena-backed
// backing.
func arenaAppendBool(arena *RegisterArena, s []bool, x bool) []bool {
	if len(s) < cap(s) {
		return append(s, x)
	}
	newCap := max(2*cap(s), len(s)+1)
	backing := arena.AllocBoolBacking(newCap)
	copy(backing, s)
	backing = backing[:len(s)+1]
	backing[len(s)] = x
	return backing
}

// arenaAppendUint is the uint64 sibling of arenaAppendInt.
//
// Takes arena (*RegisterArena) which provides the backing slab.
// Takes s ([]uint64) which is the slice to append to.
// Takes x (uint64) which is the value to append.
//
// Returns the resulting slice, either reusing s or pointing at a fresh arena-backed
// backing.
func arenaAppendUint(arena *RegisterArena, s []uint64, x uint64) []uint64 {
	if len(s) < cap(s) {
		return append(s, x)
	}
	newCap := max(2*cap(s), len(s)+1)
	backing := arena.AllocUintBacking(newCap)
	copy(backing, s)
	backing = backing[:len(s)+1]
	backing[len(s)] = x
	return backing
}

// arenaAppendByte is the byte sibling of arenaAppendInt.
//
// Shares the arena's byte slab with AllocStringBytes / AllocByteBacking via 3-index
// slicing.
//
// Takes arena (*RegisterArena) which provides the backing slab.
// Takes s ([]byte) which is the slice to append to.
// Takes x (byte) which is the value to append.
//
// Returns the resulting slice, either reusing s or pointing at a fresh arena-backed
// backing.
func arenaAppendByte(arena *RegisterArena, s []byte, x byte) []byte {
	if len(s) < cap(s) {
		return append(s, x)
	}
	newCap := max(2*cap(s), len(s)+1)
	backing := arena.AllocByteBacking(newCap)
	copy(backing, s)
	backing = backing[:len(s)+1]
	backing[len(s)] = x
	return backing
}
