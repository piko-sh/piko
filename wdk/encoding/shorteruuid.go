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
	"fmt"
	"math/bits"

	"github.com/google/uuid"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// ShorterUUIDLength is the length of a version-stripped base58-encoded UUID.
	// By removing the 4-bit version and 2-bit variant (6 bits total), we reduce
	// the encoded length from 22 to 21 characters.
	ShorterUUIDLength = 21

	// variantRFC4122 is the variant bits for RFC 4122 UUIDs (binary 10xxxxxx).
	variantRFC4122 = 0x80

	// randBMask is the bitmask for the top 6 bits of rand_b from byte 8.
	randBMask = 0x3F

	// versionShift is the bit shift for the version nibble in byte 6.
	versionShift = 4

	// timestampShift40 is the bit shift for the top byte of the timestamp.
	timestampShift40 = 40

	// timestampShift32 is the bit shift for the second timestamp byte.
	timestampShift32 = 32

	// timestampShift24 is the bit shift for the third byte of the timestamp.
	timestampShift24 = 24

	// timestampShift16 is the bit shift for the fourth byte of a timestamp.
	timestampShift16 = 16

	// timestampShift8 is the bit shift for the second-lowest byte in timestamp
	// operations.
	timestampShift8 = 8

	// randBShift56 is the bit shift amount for the high byte of rand_b.
	randBShift56 = 56

	// randBShift48 is the bit shift offset for the second byte of randB.
	randBShift48 = 48

	// randBShift40 is the bit shift for byte 10 of the randB field.
	randBShift40 = 40

	// randBShift32 is the bit shift for the fourth byte of randB.
	randBShift32 = 32

	// randBShift24 is the bit shift for byte 12 of the random B field.
	randBShift24 = 24

	// randBShift16 is the bit shift for byte 13 of randB.
	randBShift16 = 16

	// randBShift8 is the bit shift for the second byte of randB.
	randBShift8 = 8

	// base58Divisor is the base for Base58 encoding.
	base58Divisor = 58

	// uint128RandAShift is the bit position where randA's top bits start in hi.
	uint128RandAShift = 10

	// uint128RandAHiMask masks the bottom 10 bits of hi (randA's contribution).
	uint128RandAHiMask = 0x3FF

	// uint128RandBMask masks the bottom 62 bits of lo (randB).
	uint128RandBMask = 0x3FFFFFFFFFFFFFFF

	// uint128RandALoShift is the bit shift for randA's bottom 2 bits in lo.
	uint128RandALoShift = 62

	// uuidVersionMin is the minimum valid UUID version number.
	uuidVersionMin = 1

	// uuidVersionMax is the maximum valid UUID version number.
	uuidVersionMax = 15

	// uuidVersion1 is the UUID version 1 (time-based).
	uuidVersion1 = 1

	// uuidVersion4 is the UUID version 4 (random).
	uuidVersion4 = 4

	// uuidVersion7 is the UUID version 7 (time-ordered).
	uuidVersion7 = 7

	// uuidByteTimestamp0 is the byte index for the first timestamp byte in a UUID.
	uuidByteTimestamp0 = 0

	// uuidByteTimestamp1 is the byte index for the second timestamp byte in a UUID.
	uuidByteTimestamp1 = 1

	// uuidByteTimestamp2 is the byte index for the third timestamp byte in a UUID.
	uuidByteTimestamp2 = 2

	// uuidByteTimestamp3 is the byte index for the third timestamp byte in a UUID.
	uuidByteTimestamp3 = 3

	// uuidByteTimestamp4 is the byte index for the fifth timestamp byte in a UUID.
	uuidByteTimestamp4 = 4

	// uuidByteTimestamp5 is the byte index for the lowest 8 bits of the timestamp.
	uuidByteTimestamp5 = 5

	// uuidByteVersion is the byte index of the version nibble in a UUID.
	uuidByteVersion = 6

	// uuidByteRandA is the byte index for the random A field in a UUID.
	uuidByteRandA = 7

	// uuidByteVariant is the byte index of the variant field in a UUID.
	uuidByteVariant = 8

	// uuidByteRandB0 is the byte index for the first random B byte in a UUID.
	uuidByteRandB0 = 9

	// uuidByteRandB1 is the byte index for the second random B byte in a UUID.
	uuidByteRandB1 = 10

	// uuidByteRandB2 is the byte index for the third byte of the rand_b field.
	uuidByteRandB2 = 11

	// uuidByteRandB3 is the byte index for the third byte of random section B.
	uuidByteRandB3 = 12

	// uuidByteRandB4 is the byte index for the fifth byte of the random B field.
	uuidByteRandB4 = 13

	// uuidByteRandB5 is the byte index for the fifth random B byte in a UUID.
	uuidByteRandB5 = 14

	// uuidByteRandB6 is the byte index for the seventh random B byte in a UUID.
	uuidByteRandB6 = 15
)

// base58DecodeMap maps ASCII bytes to their Base58 digit values.
// Invalid characters map to -1.
var base58DecodeMap [256]int8

// UUIDToShorterString converts a UUID to a 21-character base58 string by
// removing the version (4 bits) and variant (2 bits) fields. These fields are
// fixed for any given UUID version, so they can be left out.
//
// This produces the shortest encoding for UUIDs when the version is known.
// When decoding, pass the version to ShorterStringToUUID.
//
// Takes id (uuid.UUID) which is the UUID to encode.
//
// Returns string which is the 21-character base58-encoded UUID.
func UUIDToShorterString(id uuid.UUID) string {
	timestamp := extractTimestamp(id)
	randA := extractRandA(id)
	randB := extractRandB(id)

	hi, lo := packToUint128(timestamp, randA, randB)
	return encode122BitsToBase58(hi, lo)
}

// ShorterStringToUUID decodes a 21-character shorter UUID string back to a
// UUID. It uses the given version to restore the version and variant bits.
//
// Takes s (string) which is the 21-character encoded string to decode.
// Takes version (int) which is the UUID version (1-15) to use when rebuilding.
//
// Returns uuid.UUID which is the decoded UUID with version and variant restored.
// Returns error when the string is not valid or version is out of range.
func ShorterStringToUUID(s string, version int) (uuid.UUID, error) {
	if version < uuidVersionMin || version > uuidVersionMax {
		return uuid.Nil, fmt.Errorf("invalid UUID version: %d (must be 1-15)", version)
	}

	hi, lo, err := decodeBase58To122Bits(s)
	if err != nil {
		return uuid.Nil, err
	}

	timestamp, randA, randB := unpackFromUint128(hi, lo)
	return reconstructUUID(timestamp, randA, randB, version), nil
}

// ShorterStringToUUIDv1 decodes a short UUID string to a version 1 UUID.
//
// Takes s (string) which is the 21-character encoded string to decode.
//
// Returns uuid.UUID which is the decoded UUID with version 1 bits set.
// Returns error when the string is not valid.
func ShorterStringToUUIDv1(s string) (uuid.UUID, error) {
	return ShorterStringToUUID(s, uuidVersion1)
}

// ShorterStringToUUIDv4 decodes a short UUID string and sets version 4 bits.
//
// Takes s (string) which is the 21-character encoded string to decode.
//
// Returns uuid.UUID which is the decoded UUID with version 4 bits set.
// Returns error when the string is not valid.
func ShorterStringToUUIDv4(s string) (uuid.UUID, error) {
	return ShorterStringToUUID(s, uuidVersion4)
}

// ShorterStringToUUIDv7 decodes a short string into a UUID version 7.
//
// Takes s (string) which is the 21-character encoded string to decode.
//
// Returns uuid.UUID which is the decoded UUID with version 7 bits set.
// Returns error when the string is not a valid encoded format.
func ShorterStringToUUIDv7(s string) (uuid.UUID, error) {
	return ShorterStringToUUID(s, uuidVersion7)
}

// MustShorterStringToUUID is like ShorterStringToUUID but panics on error.
// Use this only when you are certain the input is valid.
//
// Takes s (string) which is the 21-character encoded string to decode.
// Takes version (int) which is the UUID version to use when reconstructing.
//
// Returns uuid.UUID which is the decoded UUID.
//
// Panics if the string cannot be decoded.
func MustShorterStringToUUID(s string, version int) uuid.UUID {
	id, err := ShorterStringToUUID(s, version)
	if err != nil {
		panic(fmt.Sprintf("MustShorterStringToUUID: %v", err))
	}
	return id
}

// extractTimestamp extracts the 48-bit value from bytes 0-5 of a UUID.
//
// Takes id (uuid.UUID) which is the UUID to extract the value from.
//
// Returns uint64 which is the extracted 48-bit value.
func extractTimestamp(id uuid.UUID) uint64 {
	return uint64(id[uuidByteTimestamp0])<<timestampShift40 |
		uint64(id[uuidByteTimestamp1])<<timestampShift32 |
		uint64(id[uuidByteTimestamp2])<<timestampShift24 |
		uint64(id[uuidByteTimestamp3])<<timestampShift16 |
		uint64(id[uuidByteTimestamp4])<<timestampShift8 |
		uint64(id[uuidByteTimestamp5])
}

// extractRandA extracts the 12-bit rand_a field from a UUID v7.
//
// Takes id (uuid.UUID) which is the UUID to extract from.
//
// Returns uint64 which holds the 12-bit rand_a value. This combines the low
// nibble of byte 6 with byte 7.
func extractRandA(id uuid.UUID) uint64 {
	return uint64(id[uuidByteVersion]&0x0F)<<timestampShift8 | uint64(id[uuidByteRandA])
}

// extractRandB extracts the 62-bit rand_b field from a UUID v7.
//
// The rand_b field is made up of the low 6 bits of byte 8 plus bytes 9-15.
//
// Takes id (uuid.UUID) which is the UUID to extract the rand_b field from.
//
// Returns uint64 which contains the extracted 62-bit rand_b value.
func extractRandB(id uuid.UUID) uint64 {
	return uint64(id[uuidByteVariant]&randBMask)<<randBShift56 |
		uint64(id[uuidByteRandB0])<<randBShift48 |
		uint64(id[uuidByteRandB1])<<randBShift40 |
		uint64(id[uuidByteRandB2])<<randBShift32 |
		uint64(id[uuidByteRandB3])<<randBShift24 |
		uint64(id[uuidByteRandB4])<<randBShift16 |
		uint64(id[uuidByteRandB5])<<randBShift8 |
		uint64(id[uuidByteRandB6])
}

// packToUint128 combines the three UUID fields into a 122-bit value stored
// as two uint64s (hi, lo). Layout: ts(48) | randA(12) | randB(62).
//
// The 122 bits are right-aligned in the 128-bit space:
//   - hi contains bits 121-64: timestamp bits and top 10 bits of randA
//   - lo contains bits 63-0: bottom 2 bits of randA and all of randB
//
// Takes timestamp (uint64) which is the 48-bit time part of the UUID.
// Takes randA (uint64) which is the 12-bit first random part.
// Takes randB (uint64) which is the 62-bit second random part.
//
// Returns hi (uint64) which is the upper 64 bits of the 128-bit value.
// Returns lo (uint64) which is the lower 64 bits of the 128-bit value.
func packToUint128(timestamp, randA, randB uint64) (hi, lo uint64) {
	hi = (timestamp << uint128RandAShift) | (randA >> 2)
	lo = (randA << uint128RandALoShift) | randB
	return hi, lo
}

// unpackFromUint128 extracts the three UUID fields from a 122-bit value
// stored as two uint64s.
//
// Takes hi (uint64) which is the upper 64 bits of the 128-bit value.
// Takes lo (uint64) which is the lower 64 bits of the 128-bit value.
//
// Returns timestamp (uint64) which is the 48-bit timestamp field.
// Returns randA (uint64) which is the 12-bit random field.
// Returns randB (uint64) which is the 62-bit random field.
func unpackFromUint128(hi, lo uint64) (timestamp, randA, randB uint64) {
	timestamp = hi >> uint128RandAShift
	randA = ((hi & uint128RandAHiMask) << 2) | (lo >> uint128RandALoShift)
	randB = lo & uint128RandBMask
	return timestamp, randA, randB
}

// divMod58_128 divides a 128-bit number (hi, lo) by 58.
//
// Takes hi (uint64) which is the upper 64 bits.
// Takes lo (uint64) which is the lower 64 bits.
//
// Returns qHi (uint64) which is the upper 64 bits of the quotient.
// Returns qLo (uint64) which is the lower 64 bits of the quotient.
// Returns rem (uint64) which is the remainder (0-57).
func divMod58_128(hi, lo uint64) (qHi, qLo, rem uint64) {
	qHi = hi / base58Divisor
	r := hi % base58Divisor

	qLo, rem = bits.Div64(r, lo, base58Divisor)
	return qHi, qLo, rem
}

// mulAdd58_128 computes (hi, lo) * 58 + digit for 128-bit multiplication.
//
// Takes hi (uint64) which is the upper 64 bits.
// Takes lo (uint64) which is the lower 64 bits.
// Takes digit (uint64) which is the digit to add (0-57).
//
// Returns rHi (uint64) which is the upper 64 bits of the result.
// Returns rLo (uint64) which is the lower 64 bits of the result.
func mulAdd58_128(hi, lo, digit uint64) (rHi, rLo uint64) {
	loHi, loLo := bits.Mul64(lo, base58Divisor)

	hiProduct := hi*base58Divisor + loHi

	rLo, carry := bits.Add64(loLo, digit, 0)
	rHi = hiProduct + carry

	return rHi, rLo
}

// encode122BitsToBase58 encodes a 122-bit value (stored as hi, lo) to a
// 21-character Base58 string. Uses a fixed-size buffer to avoid allocations.
//
// Takes hi (uint64) which is the upper 64 bits of the 128-bit value.
// Takes lo (uint64) which is the lower 64 bits of the 128-bit value.
//
// Returns string which is the 21-character Base58-encoded result.
func encode122BitsToBase58(hi, lo uint64) string {
	var buffer [ShorterUUIDLength]byte

	for i := ShorterUUIDLength - 1; i >= 0; i-- {
		var rem uint64
		hi, lo, rem = divMod58_128(hi, lo)
		buffer[i] = base58Alphabet[rem]
	}

	return string(buffer[:])
}

// decodeBase58To122Bits decodes a Base58 string to a 122-bit value stored
// as two uint64s.
//
// Takes s (string) which is the Base58-encoded string to decode.
//
// Returns hi (uint64) which is the upper 64 bits of the decoded value.
// Returns lo (uint64) which is the lower 64 bits of the decoded value.
// Returns error when the string contains invalid Base58 characters.
func decodeBase58To122Bits(s string) (hi, lo uint64, err error) {
	for i := range len(s) {
		digit := base58DecodeMap[s[i]]
		if digit < 0 {
			return 0, 0, fmt.Errorf("invalid base58 character: %c", s[i])
		}
		hi, lo = mulAdd58_128(hi, lo, uint64(digit))
	}
	return hi, lo, nil
}

// reconstructUUID builds a UUID from three fields plus version and variant.
//
// Takes timestamp (uint64) which provides the 48-bit timestamp value.
// Takes randA (uint64) which provides the first random field.
// Takes randB (uint64) which provides the second random field.
// Takes version (int) which specifies the UUID version number.
//
// Returns uuid.UUID which is the rebuilt UUID with RFC 4122 variant set.
//
//nolint:gosec // intentional byte extraction
func reconstructUUID(timestamp, randA, randB uint64, version int) uuid.UUID {
	var id uuid.UUID

	id[uuidByteTimestamp0] = byte(timestamp >> timestampShift40)
	id[uuidByteTimestamp1] = byte(timestamp >> timestampShift32)
	id[uuidByteTimestamp2] = byte(timestamp >> timestampShift24)
	id[uuidByteTimestamp3] = byte(timestamp >> timestampShift16)
	id[uuidByteTimestamp4] = byte(timestamp >> timestampShift8)
	id[uuidByteTimestamp5] = byte(timestamp)

	id[uuidByteVersion] = byte(version<<versionShift) | byte(randA>>timestampShift8)
	id[uuidByteRandA] = byte(randA)

	id[uuidByteVariant] = variantRFC4122 | byte(randB>>randBShift56)
	id[uuidByteRandB0] = byte(randB >> randBShift48)
	id[uuidByteRandB1] = byte(randB >> randBShift40)
	id[uuidByteRandB2] = byte(randB >> randBShift32)
	id[uuidByteRandB3] = byte(randB >> randBShift24)
	id[uuidByteRandB4] = byte(randB >> randBShift16)
	id[uuidByteRandB5] = byte(randB >> randBShift8)
	id[uuidByteRandB6] = byte(randB)

	return id
}

func init() {
	for i := range base58DecodeMap {
		base58DecodeMap[i] = -1
	}
	for i, c := range base58Alphabet {
		base58DecodeMap[c] = safeconv.IntToInt8(i)
	}
}
