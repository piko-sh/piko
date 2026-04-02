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
	"bytes"
	"testing"

	"piko.sh/piko/wdk/encoding"
)

func TestBase36_Bytes(t *testing.T) {
	tests := [][]byte{
		{},
		{0},
		{0, 0, 1},
		[]byte("Hello, World!"),
		{255, 128, 0, 1},
	}

	for _, data := range tests {
		encoded := encoding.EncodeBytesBase36(data)
		decoded, err := encoding.DecodeBytesBase36(encoded)
		if err != nil {
			t.Fatalf("DecodeBytesBase36 returned error for data %v: %v", data, err)
		}
		if !bytes.Equal(decoded, data) {
			t.Errorf("Round-trip mismatch. Original: %v, Decoded: %v", data, decoded)
		}
	}
}

func TestBase36_Int(t *testing.T) {
	tests := []uint64{
		0,
		1,
		35,
		36,
		999999999,
		18446744073709551615,
	}

	for _, value := range tests {
		enc := encoding.EncodeUint64Base36(value)
		dec, err := encoding.DecodeUint64Base36(enc)
		if err != nil {
			t.Fatalf("Failed to decode %q: %v", enc, err)
		}
		if dec != value {
			t.Errorf("Base36 mismatch. Original: %d, Decoded: %d", value, dec)
		}
	}
}

func TestBase36_Invalid(t *testing.T) {
	input := "HELLO!"
	_, err := encoding.DecodeBytesBase36(input)
	if err == nil {
		t.Errorf("Expected error for invalid base36 character, got none")
	}
}

func TestBase36_Overflow(t *testing.T) {
	_, err := encoding.DecodeUint64Base36("ZZZZZZZZZZZZZZ")
	if err == nil {
		t.Errorf("Expected overflow or invalid error, got none")
	}
}
