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

//go:build fuzz

package encoding_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"piko.sh/piko/wdk/encoding"
)

func FuzzUUIDShorterRoundTrip(f *testing.F) {

	f.Add(make([]byte, 16), 4)
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, 4)
	f.Add([]byte{0x39, 0x56, 0x89, 0xAB, 0xCD, 0xEF, 0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0, 0x12, 0x34}, 4)
	f.Add([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}, 7)
	f.Add([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, 1)

	f.Fuzz(func(t *testing.T, uuidBytes []byte, version int) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic on uuid %x, version %d: %v", uuidBytes, version, r)
			}
		}()

		if len(uuidBytes) != 16 {
			return
		}

		var uuidArray [16]byte
		copy(uuidArray[:], uuidBytes)

		if version < 1 || version > 15 {

			id := uuid.UUID(uuidArray)
			encoded := encoding.UUIDToShorterString(id)
			_, err := encoding.ShorterStringToUUID(encoded, version)
			if err == nil {
				t.Errorf("expected error for invalid version %d", version)
			}
			return
		}

		id := uuid.UUID(uuidArray)
		encoded := encoding.UUIDToShorterString(id)

		if len(encoded) != 21 {
			t.Errorf("expected encoded length 21, got %d for uuid %s", len(encoded), id)
		}

		decoded, err := encoding.ShorterStringToUUID(encoded, version)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}

		origTimestamp := extractTimestamp(id)
		decodedTimestamp := extractTimestamp(decoded)
		if origTimestamp != decodedTimestamp {
			t.Errorf("timestamp mismatch:\noriginal: %x\ndecoded:  %x", origTimestamp, decodedTimestamp)
		}

		origRandA := extractRandA(id)
		decodedRandA := extractRandA(decoded)
		if origRandA != decodedRandA {
			t.Errorf("randA mismatch:\noriginal: %x\ndecoded:  %x", origRandA, decodedRandA)
		}

		origRandB := extractRandB(id)
		decodedRandB := extractRandB(decoded)
		if origRandB != decodedRandB {
			t.Errorf("randB mismatch:\noriginal: %x\ndecoded:  %x", origRandB, decodedRandB)
		}

		decodedVersion := int(decoded[6] >> 4)
		if decodedVersion != version {
			t.Errorf("version mismatch: expected %d, got %d", version, decodedVersion)
		}

		if decoded[8]&0xC0 != 0x80 {
			t.Errorf("variant bits incorrect: expected 10xxxxxx, got %08b", decoded[8])
		}
	})
}

func extractTimestamp(id uuid.UUID) uint64 {
	return uint64(id[0])<<40 |
		uint64(id[1])<<32 |
		uint64(id[2])<<24 |
		uint64(id[3])<<16 |
		uint64(id[4])<<8 |
		uint64(id[5])
}

func extractRandA(id uuid.UUID) uint64 {
	return uint64(id[6]&0x0F)<<8 | uint64(id[7])
}

func extractRandB(id uuid.UUID) uint64 {
	return uint64(id[8]&0x3F)<<56 |
		uint64(id[9])<<48 |
		uint64(id[10])<<40 |
		uint64(id[11])<<32 |
		uint64(id[12])<<24 |
		uint64(id[13])<<16 |
		uint64(id[14])<<8 |
		uint64(id[15])
}

func FuzzUUIDDecodeInvalid(f *testing.F) {

	f.Add("", 4)
	f.Add("0invalid", 4)
	f.Add("abc", 4)
	f.Add("abcdefghijklmnopqrstuvwxyz", 4)
	f.Add("1111111111111111111111", 4)
	f.Add("11111111111111111111", 4)
	f.Add("validbase58string21c", 0)
	f.Add("validbase58string21c", 16)
	f.Add("validbase58string21c", -1)
	f.Add("OIl0", 4)
	f.Add("     ", 4)
	f.Add("\x00\x00\x00", 4)
	f.Add("🚀🚀🚀🚀🚀", 4)
	f.Add(strings.Repeat("1", 1000), 4)

	f.Fuzz(func(t *testing.T, input string, version int) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ShorterStringToUUID panicked on input %q, version %d: %v",
					input, version, r)
			}
		}()

		_, _ = encoding.ShorterStringToUUID(input, version)
	})
}

func FuzzUUIDVersionSpecificDecode(f *testing.F) {

	f.Add("")
	f.Add("111111111111111111111")
	f.Add("JPwcyDCgEuq21e1WW1111")
	f.Add("0OIl")
	f.Add("abc")
	f.Add(strings.Repeat("z", 21))

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("version-specific decode panicked on input %q: %v", input, r)
			}
		}()

		_, _ = encoding.ShorterStringToUUIDv1(input)
		_, _ = encoding.ShorterStringToUUIDv4(input)
		_, _ = encoding.ShorterStringToUUIDv7(input)
	})
}

func FuzzUUIDMustDecode(f *testing.F) {

	f.Add("111111111111111111111", 4, true)
	f.Add("111111111111111111111", 0, false)
	f.Add("0OIl", 4, false)
	f.Add("", 4, false)

	f.Fuzz(func(t *testing.T, input string, version int, expectSuccess bool) {
		defer func() {
			r := recover()
			if expectSuccess && r != nil {
				t.Errorf("unexpected panic for supposedly valid input %q, version %d: %v",
					input, version, r)
			}

		}()

		if expectSuccess {

			_, err := encoding.ShorterStringToUUID(input, version)
			if err != nil {

				return
			}
			_ = encoding.MustShorterStringToUUID(input, version)
		} else {

			_, err := encoding.ShorterStringToUUID(input, version)
			if err == nil {

				_ = encoding.MustShorterStringToUUID(input, version)
			}
		}
	})
}
