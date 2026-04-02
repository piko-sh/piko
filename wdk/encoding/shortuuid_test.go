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

func TestUUIDToShortString_RoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		id   uuid.UUID
	}{
		{
			name: "random UUID",
			id:   uuid.MustParse("3956899f-f09d-4f30-928a-49e4bfa0ff3b"),
		},
		{
			name: "nil UUID",
			id:   uuid.Nil,
		},
		{
			name: "max UUID",
			id:   uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		},
		{
			name: "UUID with leading zeros",
			id:   uuid.MustParse("00000000-0000-0001-0000-000000000001"),
		},
		{
			name: "UUID v7 example",
			id:   uuid.MustParse("018e89a0-c1a0-7000-8000-000000000000"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			short := encoding.UUIDToShortString(tc.id)

			if len(short) != encoding.ShortUUIDLength {
				t.Errorf("expected length %d, got %d for short UUID %q",
					encoding.ShortUUIDLength, len(short), short)
			}

			decoded, err := encoding.ShortStringToUUID(short)
			if err != nil {
				t.Fatalf("ShortStringToUUID returned error: %v", err)
			}

			if decoded != tc.id {
				t.Errorf("round-trip mismatch.\nOriginal: %s\nDecoded:  %s", tc.id, decoded)
			}
		})
	}
}

func TestShortStringToUUID_InvalidLength(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{name: "too short", input: "abc"},
		{name: "too long", input: "abcdefghijklmnopqrstuvwxyz"},
		{name: "empty", input: ""},
		{name: "21 chars", input: "abcdefghijklmnopqrstu"},
		{name: "23 chars", input: "abcdefghijklmnopqrstuvw"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := encoding.ShortStringToUUID(tc.input)
			if err == nil {
				t.Errorf("expected error for input %q, got none", tc.input)
			}
		})
	}
}

func TestShortStringToUUID_InvalidEncoding(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{name: "contains +", input: "abcdefghijklmnopqrst+u"},
		{name: "contains /", input: "abcdefghijklmnopqrst/u"},
		{name: "contains =", input: "abcdefghijklmnopqrst=u"},
		{name: "contains space", input: "abcdefghijklmnopqrst u"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := encoding.ShortStringToUUID(tc.input)
			if err == nil {
				t.Errorf("expected error for input %q, got none", tc.input)
			}
		})
	}
}

func TestUUIDToBase58String_RoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		id   uuid.UUID
	}{
		{
			name: "random UUID",
			id:   uuid.MustParse("3956899f-f09d-4f30-928a-49e4bfa0ff3b"),
		},
		{
			name: "nil UUID",
			id:   uuid.Nil,
		},
		{
			name: "max UUID",
			id:   uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		},
		{
			name: "UUID with leading zeros",
			id:   uuid.MustParse("00000000-0000-0001-0000-000000000001"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			short := encoding.UUIDToBase58String(tc.id)

			if len(short) < 1 || len(short) > 24 {
				t.Errorf("unexpected base58 length %d for short UUID %q", len(short), short)
			}

			decoded, err := encoding.Base58StringToUUID(short)
			if err != nil {
				t.Fatalf("Base58StringToUUID returned error: %v", err)
			}

			if decoded != tc.id {
				t.Errorf("round-trip mismatch.\nOriginal: %s\nDecoded:  %s", tc.id, decoded)
			}
		})
	}
}

func TestBase58StringToUUID_Invalid(t *testing.T) {
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
			_, err := encoding.Base58StringToUUID(tc.input)
			if err == nil {
				t.Errorf("expected error for input %q, got none", tc.input)
			}
		})
	}
}

func TestMustShortStringToUUID_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got none")
		}
	}()

	encoding.MustShortStringToUUID("invalid")
}

func TestMustBase58StringToUUID_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got none")
		}
	}()

	encoding.MustBase58StringToUUID("0invalid")
}

func TestShortUUID_URLSafety(t *testing.T) {
	for range 100 {
		id := uuid.New()
		short := encoding.UUIDToShortString(id)

		for _, c := range short {
			isValid := (c >= 'A' && c <= 'Z') ||
				(c >= 'a' && c <= 'z') ||
				(c >= '0' && c <= '9') ||
				c == '-' || c == '_'
			if !isValid {
				t.Errorf("found non-URL-safe character %q in short UUID %q", c, short)
			}
		}
	}
}

func TestMustShortStringToUUID_Success(t *testing.T) {
	id := uuid.MustParse("3956899f-f09d-4f30-928a-49e4bfa0ff3b")
	short := encoding.UUIDToShortString(id)

	decoded := encoding.MustShortStringToUUID(short)
	if decoded != id {
		t.Errorf("round-trip mismatch.\nOriginal: %s\nDecoded:  %s", id, decoded)
	}
}

func TestMustBase58StringToUUID_Success(t *testing.T) {
	id := uuid.MustParse("3956899f-f09d-4f30-928a-49e4bfa0ff3b")
	base58 := encoding.UUIDToBase58String(id)

	decoded := encoding.MustBase58StringToUUID(base58)
	if decoded != id {
		t.Errorf("round-trip mismatch.\nOriginal: %s\nDecoded:  %s", id, decoded)
	}
}

func TestBase58StringToUUID_TooLong(t *testing.T) {
	longData := make([]byte, 17)
	for i := range longData {
		longData[i] = 0xFF
	}
	encoded := encoding.EncodeBytesBase58(longData)

	_, err := encoding.Base58StringToUUID(encoded)
	if err == nil {
		t.Error("expected error for data longer than 16 bytes, got nil")
	}
}

func TestShortUUID_Deterministic(t *testing.T) {
	id := uuid.MustParse("3956899f-f09d-4f30-928a-49e4bfa0ff3b")

	short1 := encoding.UUIDToShortString(id)
	short2 := encoding.UUIDToShortString(id)

	if short1 != short2 {
		t.Errorf("encoding not deterministic: %q != %q", short1, short2)
	}

	base58_1 := encoding.UUIDToBase58String(id)
	base58_2 := encoding.UUIDToBase58String(id)

	if base58_1 != base58_2 {
		t.Errorf("base58 encoding not deterministic: %q != %q", base58_1, base58_2)
	}
}
