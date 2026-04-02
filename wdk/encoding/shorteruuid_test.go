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

package encoding_test

import (
	"testing"

	"github.com/google/uuid"
	"piko.sh/piko/wdk/encoding"
)

func TestUUIDToShorterString_RoundTrip(t *testing.T) {
	testCases := []struct {
		name    string
		id      uuid.UUID
		version int
	}{
		{
			name:    "UUID v7",
			id:      uuid.MustParse("018e89a0-c1a0-7000-8000-000000000000"),
			version: 7,
		},
		{
			name:    "UUID v4 random",
			id:      uuid.MustParse("3956899f-f09d-4f30-928a-49e4bfa0ff3b"),
			version: 4,
		},
		{
			name:    "UUID v4 with minimal payload",
			id:      uuid.MustParse("00000000-0000-4000-8000-000000000000"),
			version: 4,
		},
		{
			name:    "UUID v4 near max",
			id:      uuid.MustParse("ffffffff-ffff-4fff-bfff-ffffffffffff"),
			version: 4,
		},
		{
			name:    "UUID v1",
			id:      uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			version: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shorter := encoding.UUIDToShorterString(tc.id)

			if len(shorter) != encoding.ShorterUUIDLength {
				t.Errorf("expected length %d, got %d for shorter UUID %q",
					encoding.ShorterUUIDLength, len(shorter), shorter)
			}

			decoded, err := encoding.ShorterStringToUUID(shorter, tc.version)
			if err != nil {
				t.Fatalf("ShorterStringToUUID returned error: %v", err)
			}

			if decoded != tc.id {
				t.Errorf("round-trip mismatch.\nOriginal: %s\nDecoded:  %s", tc.id, decoded)
			}
		})
	}
}

func TestShorterUUID_LengthComparison(t *testing.T) {
	id := uuid.MustParse("3956899f-f09d-4f30-928a-49e4bfa0ff3b")

	original := id.String()
	base58Short := encoding.UUIDToBase58String(id)
	shorter := encoding.UUIDToShorterString(id)

	t.Logf("Original UUID:  %s (%d chars)", original, len(original))
	t.Logf("Base58 short:   %s (%d chars)", base58Short, len(base58Short))
	t.Logf("Shorter:        %s (%d chars)", shorter, len(shorter))

	if len(shorter) >= len(base58Short) {
		t.Errorf("shorter UUID should be shorter than base58: %d >= %d",
			len(shorter), len(base58Short))
	}

	if len(shorter) != 21 {
		t.Errorf("shorter UUID should be 21 chars, got %d", len(shorter))
	}

	if len(base58Short) != 22 {
		t.Errorf("base58 short UUID should be 22 chars, got %d", len(base58Short))
	}
}

func TestShorterStringToUUID_InvalidVersion(t *testing.T) {
	shorter := encoding.UUIDToShorterString(uuid.New())

	testCases := []struct {
		name    string
		version int
	}{
		{name: "version 0", version: 0},
		{name: "version 16", version: 16},
		{name: "negative version", version: -1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := encoding.ShorterStringToUUID(shorter, tc.version)
			if err == nil {
				t.Errorf("expected error for version %d", tc.version)
			}
		})
	}
}

func TestShorterStringToUUID_InvalidEncoding(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{name: "contains 0", input: "0abcdefghijklmnopqrst"},
		{name: "contains O", input: "Oabcdefghijklmnopqrst"},
		{name: "contains I", input: "Iabcdefghijklmnopqrst"},
		{name: "contains l", input: "labcdefghijklmnopqrst"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := encoding.ShorterStringToUUID(tc.input, 4)
			if err == nil {
				t.Errorf("expected error for input %q", tc.input)
			}
		})
	}
}

func TestShorterStringToUUIDv4(t *testing.T) {
	id := uuid.MustParse("3956899f-f09d-4f30-928a-49e4bfa0ff3b")
	shorter := encoding.UUIDToShorterString(id)

	decoded, err := encoding.ShorterStringToUUIDv4(shorter)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if decoded != id {
		t.Errorf("mismatch: expected %s, got %s", id, decoded)
	}
}

func TestShorterStringToUUIDv7(t *testing.T) {
	id := uuid.MustParse("018e89a0-c1a0-7000-8000-000000000000")
	shorter := encoding.UUIDToShorterString(id)

	decoded, err := encoding.ShorterStringToUUIDv7(shorter)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if decoded != id {
		t.Errorf("mismatch: expected %s, got %s", id, decoded)
	}
}

func TestShorterStringToUUIDv1(t *testing.T) {
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	shorter := encoding.UUIDToShorterString(id)

	decoded, err := encoding.ShorterStringToUUIDv1(shorter)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if decoded != id {
		t.Errorf("mismatch: expected %s, got %s", id, decoded)
	}
}

func TestMustShorterStringToUUID_Success(t *testing.T) {
	id := uuid.MustParse("3956899f-f09d-4f30-928a-49e4bfa0ff3b")
	shorter := encoding.UUIDToShorterString(id)

	decoded := encoding.MustShorterStringToUUID(shorter, 4)
	if decoded != id {
		t.Errorf("round-trip mismatch.\nOriginal: %s\nDecoded:  %s", id, decoded)
	}
}

func TestMustShorterStringToUUID_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got none")
		}
	}()

	encoding.MustShorterStringToUUID("0invalid", 4)
}

func TestShorterUUID_VersionMismatch(t *testing.T) {
	v4 := uuid.MustParse("3956899f-f09d-4f30-928a-49e4bfa0ff3b")
	shorter := encoding.UUIDToShorterString(v4)

	decoded, err := encoding.ShorterStringToUUID(shorter, 7)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if decoded == v4 {
		t.Error("expected mismatch when using wrong version")
	}

	if decoded.Version() != 7 {
		t.Errorf("expected version 7, got %d", decoded.Version())
	}

	t.Logf("Original v4: %s", v4)
	t.Logf("Decoded v7:  %s (version mismatch expected)", decoded)
}

func TestShorterUUID_Deterministic(t *testing.T) {
	id := uuid.MustParse("3956899f-f09d-4f30-928a-49e4bfa0ff3b")

	shorter1 := encoding.UUIDToShorterString(id)
	shorter2 := encoding.UUIDToShorterString(id)

	if shorter1 != shorter2 {
		t.Errorf("encoding not deterministic: %q != %q", shorter1, shorter2)
	}
}

func TestShorterUUID_AllZeros(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000000")
	shorter := encoding.UUIDToShorterString(id)

	t.Logf("Minimal v4 UUID: %s", id)
	t.Logf("Shorter encoded: %s (%d chars)", shorter, len(shorter))

	if len(shorter) != 21 {
		t.Errorf("expected 21 chars, got %d", len(shorter))
	}

	decoded, err := encoding.ShorterStringToUUIDv4(shorter)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if decoded != id {
		t.Errorf("round-trip mismatch: expected %s, got %s", id, decoded)
	}
}

func TestShorterUUID_ManyRandomUUIDs(t *testing.T) {
	for range 100 {
		id := uuid.New()

		shorter := encoding.UUIDToShorterString(id)
		decoded, err := encoding.ShorterStringToUUIDv4(shorter)
		if err != nil {
			t.Fatalf("error decoding %q: %v", shorter, err)
		}

		if decoded != id {
			t.Errorf("round-trip failed:\nOriginal: %s\nDecoded:  %s", id, decoded)
		}
	}
}
