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
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
)

const (
	// uuidByteLength is the number of bytes in a UUID (128 bits / 8).
	uuidByteLength = 16

	// ShortUUIDLength is the length of a base64url-encoded UUID without padding.
	// A UUID is 16 bytes, which encodes to 22 base64 characters.
	ShortUUIDLength = 22

	// ShortUUIDBase58Length is the typical length of a base58-encoded UUID.
	// Base58 output can vary in length, so values may be 21 or 22 characters.
	ShortUUIDBase58Length = 22
)

var (
	// Base64URLEncoding is a pre-initialised, concurrency-safe
	// encoding using the URL-safe Base64 alphabet (RFC 4648),
	// where '-' and '_' replace '+' and '/' for URL safety.
	Base64URLEncoding *Encoding
)

// UUIDToShortString converts a UUID to a short, URL-safe string.
//
// The result is a 22-character string using base64url encoding (RFC 4648).
// This is the shortest standard way to show a UUID as text.
//
// Takes id (uuid.UUID) which is the UUID to convert.
//
// Returns string which is the 22-character encoded UUID.
func UUIDToShortString(id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString(id[:])
}

// ShortStringToUUID decodes a 22-character base64url string back to a UUID.
//
// Takes s (string) which is the 22-character encoded string to decode.
//
// Returns uuid.UUID which is the decoded UUID.
// Returns error when the string is not a valid 22-character base64url encoding
// or does not decode to exactly 16 bytes.
func ShortStringToUUID(s string) (uuid.UUID, error) {
	if len(s) != ShortUUIDLength {
		return uuid.Nil, fmt.Errorf("invalid short UUID length: expected %d, got %d", ShortUUIDLength, len(s))
	}

	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid base64url encoding: %w", err)
	}

	if len(data) != uuidByteLength {
		return uuid.Nil, fmt.Errorf("decoded data length mismatch: expected %d bytes, got %d", uuidByteLength, len(data))
	}

	var id uuid.UUID
	copy(id[:], data)
	return id, nil
}

// UUIDToBase58String encodes a UUID to a base58 string that excludes visually
// ambiguous characters (0, O, I, l), producing a 21-22 character output ideal
// for user-facing identifiers.
//
// Takes id (uuid.UUID) which is the UUID to encode.
//
// Returns string which is the base58-encoded UUID.
func UUIDToBase58String(id uuid.UUID) string {
	return EncodeBytesBase58(id[:])
}

// Base58StringToUUID converts a base58-encoded string back to a UUID.
//
// Takes s (string) which is the base58-encoded string to convert.
//
// Returns uuid.UUID which is the decoded UUID.
// Returns error when the string is not valid base58 or does not decode to
// 16 bytes or fewer.
func Base58StringToUUID(s string) (uuid.UUID, error) {
	data, err := DecodeBytesBase58(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid base58 encoding: %w", err)
	}

	if len(data) > uuidByteLength {
		return uuid.Nil, fmt.Errorf("decoded data too long: expected <= %d bytes, got %d", uuidByteLength, len(data))
	}

	var id uuid.UUID
	offset := uuidByteLength - len(data)
	copy(id[offset:], data)
	return id, nil
}

// MustShortStringToUUID is like ShortStringToUUID but panics on error.
// Use this only when you are certain the input is valid.
//
// Takes s (string) which is the 22-character encoded string to decode.
//
// Returns uuid.UUID which is the decoded UUID.
//
// Panics if the string cannot be decoded.
func MustShortStringToUUID(s string) uuid.UUID {
	id, err := ShortStringToUUID(s)
	if err != nil {
		panic(fmt.Sprintf("MustShortStringToUUID: %v", err))
	}
	return id
}

// MustBase58StringToUUID is like Base58StringToUUID but panics on error.
// Use this only when you are certain the input is valid.
//
// Takes s (string) which is the base58-encoded string to decode.
//
// Returns uuid.UUID which is the decoded UUID.
//
// Panics if the string cannot be decoded.
func MustBase58StringToUUID(s string) uuid.UUID {
	id, err := Base58StringToUUID(s)
	if err != nil {
		panic(fmt.Sprintf("MustBase58StringToUUID: %v", err))
	}
	return id
}

func init() {
	enc, err := NewEncoding(URLBase64Alphabet)
	if err != nil {
		panic(fmt.Sprintf("failed to initialise Base64URLEncoding from compile-time alphabet: %v", err))
	}
	Base64URLEncoding = enc
}
