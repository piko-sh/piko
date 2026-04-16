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

package encoding

import (
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// nanoPerMilli is the number of nanoseconds in a millisecond.
	nanoPerMilli = 1_000_000

	// uuidShift8 is the bit shift amount for extracting the third byte.
	uuidShift8 = 8

	// uuidShift12 is the bit shift for extracting milliseconds from UUID v7 time.
	uuidShift12 = 12

	// uuidShift16 is the bit shift to extract the third byte of a timestamp.
	uuidShift16 = 16

	// uuidShift24 is the bit shift for extracting byte 2 of the UUID timestamp.
	uuidShift24 = 24

	// uuidShift32 is the bit shift for the second byte of the UUID timestamp.
	uuidShift32 = 32

	// uuidShift40 is the bit shift for extracting the most significant byte.
	uuidShift40 = 40

	// uuidVersion7Nibble is the version nibble for UUID version 7.
	uuidVersion7Nibble = 0x70

	// uuidSeqHighMask masks the high 4 bits of the sub-millisecond sequence.
	uuidSeqHighMask = 0x0F

	// uuidSeqMaxValue is the largest value for the 12-bit sequence field in UUIDv7.
	uuidSeqMaxValue = 0xFFF

	// uuidVariantMin is the smallest value for UUID variant bits (binary 10xxxxxx).
	uuidVariantMin = 0x80

	// uuidVariantMax is the largest byte value that keeps the variant bits as 10.
	uuidVariantMax = 0xBF

	// uuidByteMax is the largest value a single UUID byte can hold (0xFF).
	uuidByteMax = 0xFF

	// uuidByteZero is the zero byte value used to clear UUID random bytes.
	uuidByteZero = 0x00

	// uuidByte0Index is the byte index for the first timestamp byte in a UUID.
	uuidByte0Index = 0

	// uuidByte1Index is the index of the second byte in a UUID byte slice.
	uuidByte1Index = 1

	// uuidByte2Index is the byte position for bits 24-31 of the timestamp.
	uuidByte2Index = 2

	// uuidByte3Index is the index of the fourth byte in a UUID.
	uuidByte3Index = 3

	// uuidByte4Index is the index of the fifth byte in a UUID.
	uuidByte4Index = 4

	// uuidByte5Index is the byte offset for the low byte of the millisecond timestamp.
	uuidByte5Index = 5

	// uuidLastByteIndex is the index of the last byte in a UUID slice.
	uuidLastByteIndex = 15

	// uuidSize is the length of a UUID in bytes.
	uuidSize = 16

	// uuidByte6Index is the byte position for the version nibble in a UUID.
	uuidByte6Index = 6

	// uuidByte7Index is the position of the low byte in the sub-millisecond sequence.
	uuidByte7Index = 7

	// uuidByte8Index is the index of the variant byte in a UUID.
	uuidByte8Index = 8

	// uuidRandomStart is the byte index where random data begins in a UUID v7.
	uuidRandomStart = 9
)

var (
	// lastV7timeAt holds the last combined timestamp-plus-sequence value used for
	// UUIDv7 generation, ensuring monotonic ordering.
	lastV7timeAt int64

	// timeAtMu guards concurrent access to lastV7timeAt.
	timeAtMu sync.Mutex
)

// NewV7At generates a Version 7 UUID using the provided time for the
// timestamp portion. It uses random data for the lower bits and maintains
// ascending output even if timestamps are out of order or identical.
//
// Takes t (time.Time) which specifies the Unix Epoch time for the UUID.
//
// Returns uuid.UUID which is the generated Version 7 UUID.
// Returns error when random number generation fails.
func NewV7At(t time.Time) (uuid.UUID, error) {
	uuidVal, err := uuid.NewRandom()
	if err != nil {
		return uuidVal, err
	}
	makeV7At(uuidVal[:], t)
	return uuidVal, nil
}

// NewV7AtFromReader is like NewV7At but uses a custom random source for
// the lower bits.
//
// Takes r (io.Reader) which provides the random source for UUID generation.
// Takes t (time.Time) which specifies the timestamp to embed in the UUID.
//
// Returns uuid.UUID which is the generated version 7 UUID.
// Returns error when reading from the random source fails.
func NewV7AtFromReader(r io.Reader, t time.Time) (uuid.UUID, error) {
	uuidVal, err := uuid.NewRandomFromReader(r)
	if err != nil {
		return uuidVal, err
	}
	makeV7At(uuidVal[:], t)
	return uuidVal, nil
}

// ResetLastV7timeAt resets the last V7 timestamp to zero.
//
// This is a helper for tests, ensuring consistent behaviour when running
// code again with the same time and seed.
//
// Safe for concurrent use.
func ResetLastV7timeAt() {
	timeAtMu.Lock()
	defer timeAtMu.Unlock()
	lastV7timeAt = 0
}

// NewV7MinAt returns a Version 7 UUID at the given time with all random bits
// and sub-millisecond bits set to zero. Use it as a lower bound when
// performing searches based on V7 UUIDs.
//
// Takes t (time.Time) which specifies the timestamp for the UUID.
//
// Returns uuid.UUID which is the minimum UUID for the given time.
func NewV7MinAt(t time.Time) uuid.UUID {
	var u uuid.UUID
	makeV7BoundaryAt(u[:], t, false)
	return u
}

// NewV7MaxAt returns a Version 7 UUID at the given time with all random bits
// set to one and sub-millisecond bits set to 0xFFF. Use it as an upper
// bound when performing searches based on V7 UUIDs.
//
// Takes t (time.Time) which specifies the timestamp for the UUID.
//
// Returns uuid.UUID which is the maximum possible V7 UUID at the given time.
func NewV7MaxAt(t time.Time) uuid.UUID {
	var u uuid.UUID
	makeV7BoundaryAt(u[:], t, true)
	return u
}

// makeV7At fills a UUID byte slice with version 7 format using a given time.
// This mirrors makeV7 but uses getV7TimeAt to encode the timestamp.
//
// Takes u ([]byte) which is the UUID byte slice to fill.
// Takes t (time.Time) which is the timestamp to encode.
//
//nolint:gosec // intentional byte extraction
func makeV7At(u []byte, t time.Time) {
	_ = u[uuidLastByteIndex]

	milli, sequence := getV7TimeAt(t)

	u[uuidByte0Index] = byte(milli >> uuidShift40)
	u[uuidByte1Index] = byte(milli >> uuidShift32)
	u[uuidByte2Index] = byte(milli >> uuidShift24)
	u[uuidByte3Index] = byte(milli >> uuidShift16)
	u[uuidByte4Index] = byte(milli >> uuidShift8)
	u[uuidByte5Index] = byte(milli)

	u[uuidByte6Index] = uuidVersion7Nibble | (uuidSeqHighMask & byte(sequence>>uuidShift8))
	u[uuidByte7Index] = byte(sequence)
}

// getV7TimeAt converts a time to milliseconds and a sub-millisecond sequence.
// It guarantees values always increase across calls, even for the same timestamp.
//
// Takes t (time.Time) which is the timestamp to convert.
//
// Returns milli (int64) which is the millisecond part of the time.
// Returns sequence (int64) which is the sub-millisecond sequence value.
//
// Safe for concurrent use; protected by the timeAtMu mutex.
func getV7TimeAt(t time.Time) (milli, sequence int64) {
	timeAtMu.Lock()
	defer timeAtMu.Unlock()

	nano := t.UnixNano()
	milli = nano / nanoPerMilli
	sequence = (nano - milli*nanoPerMilli) >> uuidShift8

	now := (milli << uuidShift12) + sequence
	if now <= lastV7timeAt {
		now = lastV7timeAt + 1
		milli = now >> uuidShift12
		sequence = now & uuidSeqMaxValue
	}
	lastV7timeAt = now
	return milli, sequence
}

// makeV7BoundaryAt builds a V7 UUID at a given time with boundary random bits.
//
// Takes u ([]byte) which is the UUID byte slice to fill.
// Takes t (time.Time) which sets the timestamp for the UUID.
// Takes isMax (bool) which when true sets all random bits to one, otherwise
// sets them to zero.
//
//nolint:gosec // intentional byte extraction
func makeV7BoundaryAt(u []byte, t time.Time, isMax bool) {
	_ = u[uuidLastByteIndex]

	milli := t.UnixNano() / nanoPerMilli

	u[uuidByte0Index] = byte(milli >> uuidShift40)
	u[uuidByte1Index] = byte(milli >> uuidShift32)
	u[uuidByte2Index] = byte(milli >> uuidShift24)
	u[uuidByte3Index] = byte(milli >> uuidShift16)
	u[uuidByte4Index] = byte(milli >> uuidShift8)
	u[uuidByte5Index] = byte(milli)

	var sequence int64
	if isMax {
		sequence = uuidSeqMaxValue
	}

	u[uuidByte6Index] = uuidVersion7Nibble | (uuidSeqHighMask & byte(sequence>>uuidShift8))
	u[uuidByte7Index] = byte(sequence)

	if isMax {
		u[uuidByte8Index] = uuidVariantMax
		for i := uuidRandomStart; i < uuidSize; i++ {
			u[i] = uuidByteMax
		}
	} else {
		u[uuidByte8Index] = uuidVariantMin
		for i := uuidRandomStart; i < uuidSize; i++ {
			u[i] = uuidByteZero
		}
	}
}
