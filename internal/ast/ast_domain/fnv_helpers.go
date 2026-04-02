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

package ast_domain

// Provides FNV-1a 32-bit hashing utilities for generating stable, HTML-safe keys from strings, floats, and arbitrary values.
// Uses pooled hashers and fast hex encoding to produce consistent 8-character identifiers for p-key values without allocation overhead.

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"sync"

	"piko.sh/piko/internal/mem"
)

const (
	// fnvHexBufSize is the size of the hex output buffer (8 chars for 32-bit hash).
	fnvHexBufSize = 8

	// nibbleShift0 is the bit shift for the most significant nibble.
	nibbleShift0 = 28

	// nibbleShift1 is the bit shift for the second nibble of a 32-bit hash.
	nibbleShift1 = 24

	// nibbleShift2 is the bit shift for the third nibble in a 32-bit value.
	nibbleShift2 = 20

	// nibbleShift3 is the bit shift for the fourth nibble (bits 16-19).
	nibbleShift3 = 16

	// nibbleShift4 is the bit shift for the fifth nibble (bits 12-15).
	nibbleShift4 = 12

	// nibbleShift5 is the bit shift for the sixth nibble in a 32-bit value.
	nibbleShift5 = 8

	// nibbleShift6 is the bit shift for extracting the seventh nibble.
	nibbleShift6 = 4

	// nibbleMask is the bitmask to extract a single nibble (4 bits) from a value.
	nibbleMask = 0xf
)

var (
	// fnvPool reuses FNV-1a 32-bit hashers to reduce allocation overhead.
	// FNV-32 produces 8 hex characters, which is a good balance between
	// key length and collision resistance for p-key values.
	fnvPool = sync.Pool{
		New: func() any {
			return fnv.New32a()
		},
	}

	fnvHexBufPool = sync.Pool{
		New: func() any {
			return new(make([]byte, fnvHexBufSize))
		},
	}

	// hexTable is a lookup table for fast hex encoding.
	hexTable = [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}
)

// FNVHexBuf wraps a pooled buffer for FNV hex output.
// Use Bytes to get the buffer contents, then call Release when done.
type FNVHexBuf struct {
	// ptr holds the pooled byte slice; nil when released.
	ptr *[]byte
}

// Bytes returns the underlying byte slice containing the hex-encoded hash.
//
// Returns []byte which is the hex-encoded hash data, or nil if uninitialised.
func (b *FNVHexBuf) Bytes() []byte {
	if b.ptr == nil {
		return nil
	}
	return *b.ptr
}

// Release returns the buffer to the pool. Safe to call more than once.
func (b *FNVHexBuf) Release() {
	if b.ptr != nil {
		fnvHexBufPool.Put(b.ptr)
		b.ptr = nil
	}
}

// AppendFNVString hashes a string using FNV-1a 32-bit and appends the
// 8-character hex-encoded result to the buffer.
//
// This is used for p-key values where the string content is untrusted or
// could contain problematic characters (HTML special chars, very long
// strings, etc.).
//
// The FNV hash provides:
//   - Consistent 8-character output regardless of input length
//   - No HTML escaping needed (output is always hex: 0-9, a-f)
//   - Deterministic hashing for stable key identity
//
// Takes buffer ([]byte) which is the buffer to append the hex result to.
// Takes s (string) which is the string to hash.
//
// Returns []byte which is the buffer with the 8-character hex hash appended.
func AppendFNVString(buffer []byte, s string) []byte {
	h := getFNVHasher()
	_, _ = h.Write(mem.Bytes(s))
	sum := h.Sum32()
	putFNVHasher(h)
	return appendHex32(buffer, sum)
}

// AppendFNVFloat hashes a float64 using its binary representation via FNV-1a
// 32-bit and appends the result to the buffer.
//
// This avoids confusion with key path delimiters (floats contain '.') and
// precision issues like 0.1 + 0.2 = 0.30000000000000004. The binary
// representation provides consistent hashing regardless of decimal
// representation, correct handling of NaN and Infinity, and no floating-point
// precision surprises in keys.
//
// Takes buffer ([]byte) which is the buffer to append the hash to.
// Takes f (float64) which is the value to hash.
//
// Returns []byte which is the buffer with the appended hex-encoded hash.
func AppendFNVFloat(buffer []byte, f float64) []byte {
	h := getFNVHasher()
	bits := math.Float64bits(f)
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], bits)
	_, _ = h.Write(tmp[:])
	sum := h.Sum32()
	putFNVHasher(h)
	return appendHex32(buffer, sum)
}

// AppendFNVAny hashes an arbitrary value using its fmt.Stringer
// representation (or fmt.Sprint fallback) via FNV-1a 32-bit.
//
// This is the fallback for unknown types that cannot be optimised at
// generation time. The hash provides:
//   - Bounded output length (8 chars) for any input
//   - Safe for HTML attributes (hex-only output)
//
// When v is nil, returns buffer unchanged.
//
// Takes buffer ([]byte) which is the buffer to append the hash to.
// Takes v (any) which is the value to hash.
//
// Returns []byte which is the buffer with the 8-character hex hash appended.
func AppendFNVAny(buffer []byte, v any) []byte {
	if v == nil {
		return buffer
	}

	h := getFNVHasher()

	switch typed := v.(type) {
	case interface{ String() string }:
		_, _ = h.Write(mem.Bytes(typed.String()))
	case string:
		_, _ = h.Write(mem.Bytes(typed))
	default:
		_, _ = h.Write(mem.Bytes(fmt.Sprint(v)))
	}

	sum := h.Sum32()
	putFNVHasher(h)
	return appendHex32(buffer, sum)
}

// GetFNVStringBuf computes the FNV-1a hash of a string and returns a pooled
// buffer with the 8-character hex result.
//
// The caller MUST call Release() on the returned buffer when done.
//
// Takes s (string) which is the input to hash.
//
// Returns FNVHexBuf which holds a pooled buffer with the hex-encoded hash.
func GetFNVStringBuf(s string) FNVHexBuf {
	h := getFNVHasher()
	_, _ = h.Write(mem.Bytes(s))
	sum := h.Sum32()
	putFNVHasher(h)

	bufferPointer, ok := fnvHexBufPool.Get().(*[]byte)
	if !ok {
		bufferPointer = new(make([]byte, fnvHexBufSize))
	}
	writeHex32ToBuf(*bufferPointer, sum)
	return FNVHexBuf{ptr: bufferPointer}
}

// GetFNVFloatBuf computes the FNV-1a hash of a float64 and returns a pooled
// buffer containing the 8-character hex result.
//
// The caller MUST call Release() on the returned buffer when done.
//
// Takes f (float64) which is the value to hash.
//
// Returns FNVHexBuf which wraps a pooled buffer containing the hex-encoded
// hash.
func GetFNVFloatBuf(f float64) FNVHexBuf {
	h := getFNVHasher()
	bits := math.Float64bits(f)
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], bits)
	_, _ = h.Write(tmp[:])
	sum := h.Sum32()
	putFNVHasher(h)

	bufferPointer, ok := fnvHexBufPool.Get().(*[]byte)
	if !ok {
		bufferPointer = new(make([]byte, fnvHexBufSize))
	}
	writeHex32ToBuf(*bufferPointer, sum)
	return FNVHexBuf{ptr: bufferPointer}
}

// GetFNVAnyBuf computes the FNV-1a hash of any value and returns a pooled
// buffer containing the 8-character hex result.
//
// The caller MUST call Release() on the returned buffer when done. When v is
// nil, returns an empty FNVHexBuf (Bytes() returns nil, Release() is a no-op).
//
// Takes v (any) which is the value to hash.
//
// Returns FNVHexBuf which wraps a pooled buffer containing the hex-encoded
// hash.
func GetFNVAnyBuf(v any) FNVHexBuf {
	if v == nil {
		return FNVHexBuf{}
	}

	h := getFNVHasher()

	switch typed := v.(type) {
	case interface{ String() string }:
		_, _ = h.Write(mem.Bytes(typed.String()))
	case string:
		_, _ = h.Write(mem.Bytes(typed))
	default:
		_, _ = h.Write(mem.Bytes(fmt.Sprint(v)))
	}

	sum := h.Sum32()
	putFNVHasher(h)

	bufferPointer, ok := fnvHexBufPool.Get().(*[]byte)
	if !ok {
		bufferPointer = new(make([]byte, fnvHexBufSize))
	}
	writeHex32ToBuf(*bufferPointer, sum)
	return FNVHexBuf{ptr: bufferPointer}
}

// ResetFNVPool resets the FNV hasher pool to its initial state.
// Call this via t.Cleanup(ResetFNVPool) in tests to ensure test isolation.
func ResetFNVPool() {
	fnvPool = sync.Pool{
		New: func() any {
			return fnv.New32a()
		},
	}
}

// getFNVHasher returns a FNV-32a hasher from the pool.
//
// Returns hash.Hash32 which is a reset hasher from the pool, or a new one if
// the pool is empty.
func getFNVHasher() hash.Hash32 {
	h, ok := fnvPool.Get().(hash.Hash32)
	if !ok {
		return fnv.New32a()
	}
	return h
}

// putFNVHasher returns a hasher to the pool after resetting it.
//
// Takes h (hash.Hash32) which is the hasher to return.
func putFNVHasher(h hash.Hash32) {
	h.Reset()
	fnvPool.Put(h)
}

// appendHex32 appends an 8-character hex string of a uint32 to a buffer.
// Uses a lookup table for fast encoding without memory allocation.
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes sum (uint32) which is the value to encode as hex.
//
// Returns []byte which is the buffer with the hex characters added.
func appendHex32(buffer []byte, sum uint32) []byte {
	return append(buffer,
		hexTable[(sum>>nibbleShift0)&nibbleMask],
		hexTable[(sum>>nibbleShift1)&nibbleMask],
		hexTable[(sum>>nibbleShift2)&nibbleMask],
		hexTable[(sum>>nibbleShift3)&nibbleMask],
		hexTable[(sum>>nibbleShift4)&nibbleMask],
		hexTable[(sum>>nibbleShift5)&nibbleMask],
		hexTable[(sum>>nibbleShift6)&nibbleMask],
		hexTable[sum&nibbleMask],
	)
}

// writeHex32ToBuf writes a uint32 value as an 8-character hex string into a
// buffer. The buffer must be at least 8 bytes long.
//
// Takes buffer ([]byte) which is the target buffer to write into.
// Takes sum (uint32) which is the value to convert to hex.
func writeHex32ToBuf(buffer []byte, sum uint32) {
	var nibbleShifts = [fnvHexBufSize]uint{
		nibbleShift0, nibbleShift1, nibbleShift2, nibbleShift3,
		nibbleShift4, nibbleShift5, nibbleShift6, 0,
	}
	for i := range fnvHexBufSize {
		buffer[i] = hexTable[(sum>>nibbleShifts[i])&nibbleMask] //nolint:gosec // loop bounded by array size
	}
}
