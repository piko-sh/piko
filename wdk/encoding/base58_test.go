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

func TestBase58_Bytes(t *testing.T) {
	tests := [][]byte{
		{},
		{0},
		{0, 0, 1},
		[]byte("Hello, Base58!"),
		{255, 0, 1, 45, 128},
	}

	for _, data := range tests {
		encoded := encoding.EncodeBytesBase58(data)
		decoded, err := encoding.DecodeBytesBase58(encoded)
		if err != nil {
			t.Fatalf("DecodeBytesBase58 returned error for data %v: %v", data, err)
		}
		if !bytes.Equal(decoded, data) {
			t.Errorf("Round-trip mismatch.\nOriginal: %v\nDecoded:  %v", data, decoded)
		}
	}
}

func TestBase58_Int(t *testing.T) {
	tests := []uint64{
		0,
		1,
		57,
		58,
		999999999,
		18446744073709551615,
	}

	for _, value := range tests {
		enc := encoding.EncodeUint64Base58(value)
		dec, err := encoding.DecodeUint64Base58(enc)
		if err != nil {
			t.Fatalf("DecodeUint64Base58 error for encoded %q: %v", enc, err)
		}
		if dec != value {
			t.Errorf("Base58 mismatch. Original: %d, Decoded: %d", value, dec)
		}
	}
}

func TestBase58_Invalid(t *testing.T) {
	input := "012345"
	_, err := encoding.DecodeBytesBase58(input)
	if err == nil {
		t.Errorf("Expected error for invalid character '0' in base58 input, got none")
	}
}

func TestBase58_Overflow(t *testing.T) {
	input := "zzzzzzzzzzzzzz"
	_, err := encoding.DecodeUint64Base58(input)
	if err == nil {
		t.Errorf("Expected overflow error, got none")
	}
}
