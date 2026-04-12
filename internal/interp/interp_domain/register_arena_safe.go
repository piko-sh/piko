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

//go:build safe || (js && wasm)

package interp_domain

import (
	"strconv"
	"unsafe"
)

// arenaUsesUnsafeSlabs is false in the safe build: the arena does not expose the
// slab-internal fields (byteSlab, sliceHeaderSlab, byteIndex, oldByteSlabs, ...) used by
// some white-box unit tests. Those tests guard themselves with `if !arenaUsesUnsafeSlabs
// { t.Skip(...) }` so the safe build can exercise public arena semantics without
// triggering missing-internal panics.
const arenaUsesUnsafeSlabs = false

// ownsBytePointer reports whether p points into the arena's byte slabs. The safe build
// returns false so callers route through the escape- promote copy path that always
// allocates a fresh heap location.
//
// Takes p (unsafe.Pointer) which is the storage pointer to test.
//
// Returns false always in the safe build.
func (*RegisterArena) ownsBytePointer(_ unsafe.Pointer) bool {
	return false
}

// ownsSliceHeaderPointer reports whether p falls inside any arena slice-header slab. The
// safe build returns false so callers conservatively skip the slab-aware fast path.
//
// Takes p (unsafe.Pointer) which is the slice header pointer to test.
//
// Returns false always in the safe build.
func (*RegisterArena) ownsSliceHeaderPointer(_ unsafe.Pointer) bool {
	return false
}

// ownsString reports whether s points into the arena's byte slabs. The safe build always
// returns true so callers unconditionally clone strings before they escape, preserving
// correctness without unsafe.
//
// Returns true always in the safe build.
func (*RegisterArena) ownsString(_ string) bool {
	return true
}

// arenaConcatString concatenates a and b. The safe build delegates to the standard string
// concatenation operator.
//
// Takes a (string) which is the left operand.
// Takes b (string) which is the right operand.
//
// Returns the concatenated string.
func arenaConcatString(_ *RegisterArena, a, b string) string {
	return a + b
}

// arenaConcatRuneString appends the rune r to s. The safe build delegates to standard
// string conversion.
//
// Takes s (string) which is the base string.
// Takes r (rune) which is the rune to append.
//
// Returns the resulting string.
func arenaConcatRuneString(_ *RegisterArena, s string, r rune) string {
	return s + string(r)
}

// arenaRuneToString converts rune r to a string. The safe build delegates to the standard
// string conversion.
//
// Takes r (rune) which is the rune to convert.
//
// Returns the single-rune string.
func arenaRuneToString(_ *RegisterArena, r rune) string {
	return string(r)
}

// arenaBytesToString converts byte slice b to a string. The safe build delegates to the
// standard string conversion.
//
// Takes b ([]byte) which is the byte slice to convert.
//
// Returns the resulting string.
func arenaBytesToString(_ *RegisterArena, b []byte) string {
	return string(b)
}

// arenaItoaString formats x in base 10. The safe build delegates to strconv.Itoa; the
// unsafe build writes into the arena byte slab.
//
// Takes x (int64) which is the integer to format.
//
// Returns the decimal string.
func arenaItoaString(_ *RegisterArena, x int64) string {
	return strconv.FormatInt(x, 10)
}

// arenaFormatIntString formats x in the given base. The safe build delegates to
// strconv.FormatInt; the unsafe build writes into the arena byte slab.
//
// Takes x (int64) which is the integer to format.
// Takes base (int) which is the radix.
//
// Returns the formatted string.
func arenaFormatIntString(_ *RegisterArena, x int64, base int) string {
	return strconv.FormatInt(x, base)
}

// arenaAppendInt is the arena-aware append for []int64.
//
// The safe build delegates to Go's builtin append. The unsafe build intercepts the grow
// path to allocate the new backing from the arena rather than the Go heap.
//
// Takes s ([]int64) which is the source slice.
// Takes x (int64) which is the element to append.
//
// Returns the resulting slice.
func arenaAppendInt(_ *RegisterArena, s []int64, x int64) []int64 {
	return append(s, x)
}

// arenaAppendFloat is the float64 sibling of arenaAppendInt.
//
// Takes s ([]float64) which is the source slice.
// Takes x (float64) which is the element to append.
//
// Returns the resulting slice.
func arenaAppendFloat(_ *RegisterArena, s []float64, x float64) []float64 {
	return append(s, x)
}

// arenaAppendString is the string sibling of arenaAppendInt.
//
// Takes s ([]string) which is the source slice.
// Takes x (string) which is the element to append.
//
// Returns the resulting slice.
func arenaAppendString(_ *RegisterArena, s []string, x string) []string {
	return append(s, x)
}

// arenaAppendBool is the bool sibling of arenaAppendInt.
//
// Takes s ([]bool) which is the source slice.
// Takes x (bool) which is the element to append.
//
// Returns the resulting slice.
func arenaAppendBool(_ *RegisterArena, s []bool, x bool) []bool {
	return append(s, x)
}

// arenaAppendUint is the uint64 sibling of arenaAppendInt.
//
// Takes s ([]uint64) which is the source slice.
// Takes x (uint64) which is the element to append.
//
// Returns the resulting slice.
func arenaAppendUint(_ *RegisterArena, s []uint64, x uint64) []uint64 {
	return append(s, x)
}

// arenaAppendByte is the byte sibling of arenaAppendInt.
//
// Takes s ([]byte) which is the source slice.
// Takes x (byte) which is the element to append.
//
// Returns the resulting slice.
func arenaAppendByte(_ *RegisterArena, s []byte, x byte) []byte {
	return append(s, x)
}
