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

//nolint:dupl // parallel typed API per alphabet.
package encoding

import "fmt"

var (
	// base36Alphabet holds the character set for Base36 encoding (digits 0-9, uppercase A-Z).
	base36Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	// Base36Encoding is a ready-to-use encoding using the standard Base36 alphabet
	// (0-9, A-Z). It is safe for concurrent use.
	Base36Encoding *Encoding
)

// EncodeBytesBase36 encodes raw bytes into a Base36 string.
//
// Takes data ([]byte) which is the raw bytes to encode.
//
// Returns string which is the Base36-encoded form of the input.
func EncodeBytesBase36(data []byte) string {
	return Base36Encoding.EncodeBytes(data)
}

// DecodeBytesBase36 decodes a Base36-encoded string back into raw bytes.
//
// Takes input (string) which is the Base36-encoded string to decode.
//
// Returns []byte which contains the decoded bytes.
// Returns error when the input contains invalid Base36 characters.
func DecodeBytesBase36(input string) ([]byte, error) {
	return Base36Encoding.DecodeBytes(input)
}

// EncodeUint64Base36 encodes a uint64 value into a Base36 string,
// producing short, URL-safe identifiers from database IDs.
//
// Takes value (uint64) which is the number to encode.
//
// Returns string which is the Base36 encoded result.
func EncodeUint64Base36(value uint64) string {
	return Base36Encoding.EncodeUint64(value)
}

// DecodeUint64Base36 decodes a base36 string back into a uint64 value.
//
// Takes input (string) which is the base36 string to decode.
//
// Returns uint64 which is the decoded value.
// Returns error when the input is not valid base36 or overflows uint64.
func DecodeUint64Base36(input string) (uint64, error) {
	return Base36Encoding.DecodeUint64(input)
}

func init() {
	enc, err := NewEncoding(base36Alphabet)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialise Base36Encoding: %v", err))
	}
	Base36Encoding = enc
}
