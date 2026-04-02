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
	"unicode/utf8"
	"unsafe"

	"piko.sh/piko/internal/mem"
	"piko.sh/piko/wdk/safeconv"
)

// ownsString reports whether s points into the arena's current or
// previous byte slabs. Used to decide whether a string must be cloned
// before it can safely outlive the arena.
//
// Takes s (string) which is the string to test.
//
// Returns true when s is backed by the arena's byte slabs.
func (a *RegisterArena) ownsString(s string) bool {
	if len(s) == 0 {
		return false
	}
	ptr := uintptr(unsafe.Pointer(unsafe.StringData(s)))
	if len(a.byteSlab) > 0 {
		base := uintptr(unsafe.Pointer(&a.byteSlab[0]))
		if ptr >= base && ptr < base+uintptr(len(a.byteSlab)) {
			return true
		}
	}
	for _, slab := range a.oldByteSlabs {
		if len(slab) == 0 {
			continue
		}
		base := uintptr(unsafe.Pointer(&slab[0]))
		if ptr >= base && ptr < base+uintptr(len(slab)) {
			return true
		}
	}
	return false
}

// isStringAtSlabTail reports whether the end of s coincides with the
// current byte-slab write position, enabling in-place extension when
// appending to the most-recently-allocated arena string.
//
// Takes s (string) which is the string to test.
//
// Returns true when s ends exactly at the current slab allocation
// pointer.
func (a *RegisterArena) isStringAtSlabTail(s string) bool {
	if len(s) == 0 || len(a.byteSlab) == 0 {
		return false
	}
	sEnd := uintptr(unsafe.Pointer(unsafe.StringData(s))) + uintptr(len(s))
	slabPos := uintptr(unsafe.Pointer(&a.byteSlab[0])) + safeconv.IntToUintptr(a.byteIndex)
	return sEnd == slabPos
}

// arenaConcatString concatenates a and b, bump-allocating the result
// into the arena's byte slab. When a ends at the slab tail, the bytes
// of b are appended in place without copying a.
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

// arenaConcatRuneString appends rune r to string s using bump
// allocation. When s ends at the slab tail, the rune bytes are
// written in place without copying s.
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

// arenaRuneToString converts rune r to a string using bump allocation
// into the arena's byte slab.
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

// arenaBytesToString converts byte slice b to a string using bump
// allocation into the arena's byte slab.
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
