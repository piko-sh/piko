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

package fbs

import (
	"crypto/sha256"
	"errors"
)

// hashSize is the size in bytes of the schema hash prefix (32 bytes for SHA-256).
const hashSize = 32

var (
	// ErrSchemaVersionMismatch is returned when stored data was saved with a
	// different schema version. The caller should treat this as a cache miss and
	// create the data again.
	ErrSchemaVersionMismatch = errors.New("flatbuffer schema version mismatch")

	// errDataTooShort indicates the data is shorter than the minimum required
	// length (hashSize bytes for the hash prefix).
	errDataTooShort = errors.New("flatbuffer data too short for version header")
)

// SchemaHash is a fixed-size array that holds a SHA-256 hash of a schema file.
type SchemaHash [hashSize]byte

// ComputeSchemaHash computes a SHA-256 hash of schema file content.
// Call this once during init with embedded schema bytes.
//
// Takes schemaContent ([]byte) which is the raw schema file bytes to hash.
//
// Returns SchemaHash which is the computed hash of the content.
func ComputeSchemaHash(schemaContent []byte) SchemaHash {
	return sha256.Sum256(schemaContent)
}

// PackedSize returns the total size needed for a versioned blob.
// Use this to pre-allocate a buffer of the correct size.
//
// Takes payloadLen (int) which is the size of the payload data.
//
// Returns int which is the total size including the hash prefix.
func PackedSize(payloadLen int) int {
	return hashSize + payloadLen
}

// Pack writes a schema hash and payload into the given buffer.
//
// The buffer dst must have length at least PackedSize(len(payload)).
// This is the zero-allocation version; the caller provides the buffer.
//
// Takes dst ([]byte) which is the destination buffer for the packed data.
// Takes hash (SchemaHash) which identifies the schema version.
// Takes payload ([]byte) which contains the data to pack after the hash.
//
// Returns int which is the number of bytes written.
func Pack(dst []byte, hash SchemaHash, payload []byte) int {
	copy(dst[:hashSize], hash[:])
	copy(dst[hashSize:], payload)
	return hashSize + len(payload)
}

// PackAlloc creates a new versioned blob by prepending the schema hash to
// payload. This allocates a new slice - use Pack with a pre-allocated buffer
// in hot paths.
//
// Takes hash (SchemaHash) which identifies the schema version.
// Takes payload ([]byte) which contains the data to be packed.
//
// Returns []byte which contains the hash followed by the payload data.
func PackAlloc(hash SchemaHash, payload []byte) []byte {
	result := make([]byte, hashSize+len(payload))
	Pack(result, hash, payload)
	return result
}

// Unpack checks the schema hash and returns the payload slice.
// The returned payload is a sub-slice of data (zero-copy, no allocation).
//
// Takes expectedHash (SchemaHash) which is the hash to check against.
// Takes data ([]byte) which holds the hash prefix followed by the payload.
//
// Returns []byte which is the payload part of data after the hash.
// Returns error when data is smaller than hashSize (errDataTooShort) or when
// the stored hash does not match expectedHash (ErrSchemaVersionMismatch).
//
// Safety: The returned slice shares the same underlying array as data.
// Do not change data while using the returned payload.
func Unpack(expectedHash SchemaHash, data []byte) ([]byte, error) {
	if len(data) < hashSize {
		return nil, errDataTooShort
	}

	for i := range hashSize {
		if data[i] != expectedHash[i] {
			return nil, ErrSchemaVersionMismatch
		}
	}

	return data[hashSize:], nil
}

// ValidateHash checks if data starts with the expected schema hash.
// This is a quick check that does not extract the payload.
//
// Takes expectedHash (SchemaHash) which is the hash to match against.
// Takes data ([]byte) which is the raw data to check.
//
// Returns bool which is true if data starts with the expected hash.
func ValidateHash(expectedHash SchemaHash, data []byte) bool {
	if len(data) < hashSize {
		return false
	}
	for i := range hashSize {
		if data[i] != expectedHash[i] {
			return false
		}
	}
	return true
}

// extractHash extracts the schema hash from versioned data without validation.
// Returns a zero hash if data is too short.
//
// Takes data ([]byte) which contains the versioned data to extract the hash
// from.
//
// Returns SchemaHash which is the extracted hash, or a zero hash if data is
// shorter than hashSize.
func extractHash(data []byte) SchemaHash {
	var hash SchemaHash
	if len(data) >= hashSize {
		copy(hash[:], data[:hashSize])
	}
	return hash
}
