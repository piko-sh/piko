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
	"errors"
	"math"
	"testing"

	"piko.sh/piko/wdk/encoding"
)

func TestNewEncoding(t *testing.T) {
	_, err := encoding.NewEncoding("")
	if err == nil {
		t.Errorf("Expected error when creating encoding with empty alphabet, got nil")
	}

	_, err = encoding.NewEncoding("ABCDA")
	if err == nil {
		t.Errorf("Expected error for repeated character in alphabet, got nil")
	}

	_, err = encoding.NewEncoding("ABCDEF")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestEncodeDecodeBytes(t *testing.T) {
	alphabet := "0123456789ABCDEF"
	enc, err := encoding.NewEncoding(alphabet)
	if err != nil {
		t.Fatalf("Unexpected error creating encoding: %v", err)
	}

	tests := []struct {
		name     string
		expected string
		input    []byte
	}{
		{
			name:     "Empty",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "ZeroByte",
			input:    []byte{0},
			expected: "0",
		},
		{
			name:     "Hello",
			input:    []byte("Hello"),
			expected: "",
		},
		{
			name:     "LeadingZeros",
			input:    []byte{0, 0, 5, 128},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := enc.EncodeBytes(tt.input)

			decoded, err := enc.DecodeBytes(encoded)
			if err != nil {
				t.Fatalf("DecodeBytes failed: %v", err)
			}
			if !bytes.Equal(decoded, tt.input) {
				t.Errorf("Round-trip mismatch.\nOriginal: %v\nDecoded:  %v", tt.input, decoded)
			}
		})
	}
}

func TestBaseX_Decode_RejectsOversizedInput(t *testing.T) {
	encoding.SetMaxBaseXInputBytes(64)
	t.Cleanup(func() { encoding.SetMaxBaseXInputBytes(0) })

	alphabet := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	enc, err := encoding.NewEncoding(alphabet)
	if err != nil {
		t.Fatalf("creating encoding: %v", err)
	}

	tooLarge := make([]byte, 65)
	for i := range tooLarge {
		tooLarge[i] = '0'
	}

	_, decodeErr := enc.DecodeBytes(string(tooLarge))
	if decodeErr == nil {
		t.Fatalf("expected error for oversize decode input")
	}
	if !errors.Is(decodeErr, encoding.ErrBaseXInputTooLarge) {
		t.Fatalf("expected ErrBaseXInputTooLarge, got %v", decodeErr)
	}

	oversizeBytes := make([]byte, 65)
	if _, err := enc.TryEncodeBytes(oversizeBytes); !errors.Is(err, encoding.ErrBaseXInputTooLarge) {
		t.Fatalf("expected encode to surface ErrBaseXInputTooLarge, got %v", err)
	}
	if got := enc.EncodeBytes(oversizeBytes); got != "" {
		t.Fatalf("expected EncodeBytes to return empty string for oversize input, got %q", got)
	}
}

func TestInvalidDecodeBytes(t *testing.T) {
	alphabet := "123456789ABCDEF"
	enc, err := encoding.NewEncoding(alphabet)
	if err != nil {
		t.Fatalf("Error creating encoding: %v", err)
	}

	input := "Z111"
	_, decErr := enc.DecodeBytes(input)
	if decErr == nil {
		t.Errorf("Expected error decoding invalid character, got nil")
	}
}

func TestEncodeDecodeUint64(t *testing.T) {
	alphabet := "0123456789ABCDEF"
	enc, err := encoding.NewEncoding(alphabet)
	if err != nil {
		t.Fatalf("Error creating encoding: %v", err)
	}

	tests := []struct {
		name  string
		value uint64
	}{
		{name: "Zero", value: 0},
		{name: "One", value: 1},
		{name: "SmallNumber", value: 12345},
		{name: "BigNumber", value: 0xFFFFFFFFFFFFFFFE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := enc.EncodeUint64(tt.value)
			decoded, err := enc.DecodeUint64(encoded)
			if err != nil {
				t.Fatalf("DecodeUint64 failed: %v", err)
			}
			if decoded != tt.value {
				t.Errorf("Round-trip mismatch. Original: %d, Decoded: %d", tt.value, decoded)
			}
		})
	}
}

func TestDecodeUint64Overflow(t *testing.T) {
	enc, err := encoding.NewEncoding("0123456789")
	if err != nil {
		t.Fatalf("Error creating encoding: %v", err)
	}

	_, decErr := enc.DecodeUint64("999999999999999999999")
	if decErr == nil {
		t.Errorf("Expected overflow error, got nil")
	}
}

func TestEncodeDecode8Bytes(t *testing.T) {
	testCases := []struct {
		name  string
		input []byte
	}{
		{name: "all zeros", input: []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{name: "all ones", input: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}},
		{name: "sequential", input: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
		{name: "random values", input: []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE}},
		{name: "leading zeros", input: []byte{0, 0, 0, 0, 0xDE, 0xAD, 0xBE, 0xEF}},
		{name: "single byte value", input: []byte{0, 0, 0, 0, 0, 0, 0, 1}},
	}

	for _, tc := range testCases {
		t.Run("Base58_"+tc.name, func(t *testing.T) {
			encoded := encoding.EncodeBytesBase58(tc.input)
			decoded, err := encoding.DecodeBytesBase58(encoded)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if !bytes.Equal(decoded, tc.input) {
				t.Errorf("round-trip mismatch.\nInput:   %x\nEncoded: %s\nDecoded: %x",
					tc.input, encoded, decoded)
			}
		})

		t.Run("Base36_"+tc.name, func(t *testing.T) {
			encoded := encoding.EncodeBytesBase36(tc.input)
			decoded, err := encoding.DecodeBytesBase36(encoded)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if !bytes.Equal(decoded, tc.input) {
				t.Errorf("round-trip mismatch.\nInput:   %x\nEncoded: %s\nDecoded: %x",
					tc.input, encoded, decoded)
			}
		})
	}
}

func TestEncodeDecode16Bytes(t *testing.T) {
	testCases := []struct {
		name  string
		input []byte
	}{
		{
			name:  "all zeros",
			input: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:  "all ones",
			input: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name:  "sequential",
			input: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
		{
			name:  "uuid-like",
			input: []byte{0x39, 0x56, 0x89, 0x9f, 0xf0, 0x9d, 0x4f, 0x30, 0x92, 0x8a, 0x49, 0xe4, 0xbf, 0xa0, 0xff, 0x3b},
		},
		{
			name:  "leading zeros",
			input: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE},
		},
		{
			name:  "many leading zeros",
			input: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
	}

	for _, tc := range testCases {
		t.Run("Base58_"+tc.name, func(t *testing.T) {
			encoded := encoding.EncodeBytesBase58(tc.input)
			decoded, err := encoding.DecodeBytesBase58(encoded)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if !bytes.Equal(decoded, tc.input) {
				t.Errorf("round-trip mismatch.\nInput:   %x\nEncoded: %s\nDecoded: %x",
					tc.input, encoded, decoded)
			}
		})

		t.Run("Base36_"+tc.name, func(t *testing.T) {
			encoded := encoding.EncodeBytesBase36(tc.input)
			decoded, err := encoding.DecodeBytesBase36(encoded)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if !bytes.Equal(decoded, tc.input) {
				t.Errorf("round-trip mismatch.\nInput:   %x\nEncoded: %s\nDecoded: %x",
					tc.input, encoded, decoded)
			}
		})
	}
}

func TestNonASCIIEncoding_Uint64RoundTrip(t *testing.T) {
	enc, err := encoding.NewEncoding("αβγδεζηθικ")
	if err != nil {
		t.Fatalf("Failed to create non-ASCII encoding: %v", err)
	}

	testCases := []struct {
		name  string
		value uint64
	}{
		{name: "zero", value: 0},
		{name: "one", value: 1},
		{name: "small", value: 12345},
		{name: "large", value: math.MaxUint64 - 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded := enc.EncodeUint64(tc.value)
			decoded, err := enc.DecodeUint64(encoded)
			if err != nil {
				t.Fatalf("DecodeUint64 failed: %v", err)
			}
			if decoded != tc.value {
				t.Errorf("round-trip mismatch. Original: %d, Decoded: %d", tc.value, decoded)
			}
		})
	}
}

func TestNonASCIIEncoding_Uint64DecodeErrors(t *testing.T) {
	enc, err := encoding.NewEncoding("αβγδεζηθικ")
	if err != nil {
		t.Fatalf("Failed to create non-ASCII encoding: %v", err)
	}

	t.Run("empty string", func(t *testing.T) {
		_, err := enc.DecodeUint64("")
		if err == nil {
			t.Error("expected error for empty string, got nil")
		}
	})

	t.Run("invalid rune", func(t *testing.T) {
		_, err := enc.DecodeUint64("αβγX")
		if err == nil {
			t.Error("expected error for invalid rune, got nil")
		}
	})

	t.Run("overflow", func(t *testing.T) {
		_, err := enc.DecodeUint64("αβγδεζηθικαβγδεζηθικαβγδ")
		if err == nil {
			t.Error("expected overflow error, got nil")
		}
	})
}

func TestNonASCIIEncoding_BytesRoundTrip(t *testing.T) {
	enc, err := encoding.NewEncoding("αβγδεζηθικ")
	if err != nil {
		t.Fatalf("Failed to create non-ASCII encoding: %v", err)
	}

	testCases := []struct {
		name  string
		input []byte
	}{
		{name: "empty", input: []byte{}},
		{name: "single zero", input: []byte{0}},
		{name: "hello", input: []byte("Hello")},
		{name: "leading zeros", input: []byte{0, 0, 5, 128}},
		{name: "all zeros", input: []byte{0, 0, 0}},
		{name: "high bytes", input: []byte{0xFF, 0xFE, 0xFD}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded := enc.EncodeBytes(tc.input)
			decoded, err := enc.DecodeBytes(encoded)
			if err != nil {
				t.Fatalf("DecodeBytes failed: %v", err)
			}
			if !bytes.Equal(decoded, tc.input) {
				t.Errorf("round-trip mismatch.\nOriginal: %v\nDecoded:  %v", tc.input, decoded)
			}
		})
	}
}

func TestNonASCIIEncoding_DecodeBytesInvalidRune(t *testing.T) {
	enc, err := encoding.NewEncoding("αβγδεζηθικ")
	if err != nil {
		t.Fatalf("Failed to create non-ASCII encoding: %v", err)
	}

	_, err = enc.DecodeBytes("αβγX")
	if err == nil {
		t.Error("expected error for invalid rune in DecodeBytes, got nil")
	}
}

func TestStandardAlphabetFastPaths(t *testing.T) {
	testCases := []struct {
		name     string
		alphabet string
	}{
		{name: "StdBase64", alphabet: encoding.StdBase64Alphabet},
		{name: "URLBase64", alphabet: encoding.URLBase64Alphabet},
		{name: "HexLower", alphabet: encoding.StdHexAlphabetLower},
		{name: "HexUpper", alphabet: encoding.StdHexAlphabetUpper},
		{name: "StdBase32", alphabet: encoding.StdBase32Alphabet},
		{name: "HexBase32", alphabet: encoding.HexBase32Alphabet},
	}

	data := []byte("Hello, World!")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enc, err := encoding.NewEncoding(tc.alphabet)
			if err != nil {
				t.Fatalf("NewEncoding failed: %v", err)
			}

			encoded := enc.EncodeBytes(data)
			decoded, err := enc.DecodeBytes(encoded)
			if err != nil {
				t.Fatalf("DecodeBytes failed: %v", err)
			}
			if !bytes.Equal(decoded, data) {
				t.Errorf("round-trip mismatch.\nOriginal: %v\nDecoded:  %v", data, decoded)
			}
		})
	}
}

func TestSmallBaseEncoding_BytesRoundTrip(t *testing.T) {
	enc, err := encoding.NewEncoding("0123456789")
	if err != nil {
		t.Fatalf("Failed to create base-10 encoding: %v", err)
	}

	testCases := []struct {
		name  string
		input []byte
	}{
		{name: "8 bytes", input: []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE}},
		{name: "16 bytes", input: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}},
		{name: "8 bytes leading zeros", input: []byte{0, 0, 0, 0, 0, 0, 0, 1}},
		{name: "16 bytes leading zeros", input: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded := enc.EncodeBytes(tc.input)
			decoded, err := enc.DecodeBytes(encoded)
			if err != nil {
				t.Fatalf("DecodeBytes failed: %v", err)
			}
			if !bytes.Equal(decoded, tc.input) {
				t.Errorf("round-trip mismatch.\nOriginal: %x\nDecoded:  %x", tc.input, decoded)
			}
		})
	}
}

func TestFastPathConsistency(t *testing.T) {

	sizes := []int{7, 8, 9, 15, 16, 17}

	for _, size := range sizes {
		t.Run("Base58_size_"+string(rune('0'+size/10))+string(rune('0'+size%10)), func(t *testing.T) {
			input := make([]byte, size)
			for i := range input {
				input[i] = byte(i + 1)
			}

			encoded := encoding.EncodeBytesBase58(input)
			decoded, err := encoding.DecodeBytesBase58(encoded)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if !bytes.Equal(decoded, input) {
				t.Errorf("round-trip mismatch for size %d.\nInput:   %x\nEncoded: %s\nDecoded: %x",
					size, input, encoded, decoded)
			}
		})
	}
}
