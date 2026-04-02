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
	base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	// Base58Encoding is a pre-initialised encoding using
	// the Bitcoin Base58 alphabet
	// that excludes visually ambiguous characters (0, O, I, l). It is safe for
	// concurrent use.
	Base58Encoding *Encoding
)

// EncodeBytesBase58 encodes raw bytes into a Base58 string representation.
// The Base58 alphabet excludes visually ambiguous characters for improved
// readability.
//
// Takes data ([]byte) which is the raw bytes to encode.
//
// Returns string which is the Base58-encoded representation of the input.
func EncodeBytesBase58(data []byte) string {
	return Base58Encoding.EncodeBytes(data)
}

// DecodeBytesBase58 decodes a Base58-encoded string into raw bytes.
//
// Takes input (string) which is the Base58-encoded string to decode.
//
// Returns []byte which contains the decoded bytes.
// Returns error when the input contains invalid Base58 characters.
func DecodeBytesBase58(input string) ([]byte, error) {
	return Base58Encoding.DecodeBytes(input)
}

// EncodeUint64Base58 encodes a uint64 value into a Base58 string,
// producing short, readable identifiers from database IDs.
//
// Takes value (uint64) which is the number to encode.
//
// Returns string which is the Base58 form of the value.
func EncodeUint64Base58(value uint64) string {
	return Base58Encoding.EncodeUint64(value)
}

// DecodeUint64Base58 decodes a Base58-encoded string into a uint64 value.
//
// Takes input (string) which is the Base58-encoded string to decode.
//
// Returns uint64 which is the decoded value.
// Returns error when the input is not valid Base58 or the value is too large.
func DecodeUint64Base58(input string) (uint64, error) {
	return Base58Encoding.DecodeUint64(input)
}

func init() {
	enc, err := NewEncoding(base58Alphabet)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialise Base58Encoding: %v", err))
	}
	Base58Encoding = enc
}
