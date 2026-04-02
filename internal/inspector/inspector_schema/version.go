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

package inspector_schema

import (
	_ "embed" // Required for go:embed directive.

	"piko.sh/piko/internal/fbs"
)

var (
	// schemaContent holds the embedded FlatBuffers schema from type_data.fbs.
	schemaContent []byte

	// SchemaHash is the SHA-256 hash of type_data.fbs, computed at init time.
	// This hash changes whenever the schema file is modified, so the
	// cache invalidates automatically when the schema evolves.
	SchemaHash = fbs.ComputeSchemaHash(schemaContent)
)

// Pack wraps a serialised Inspector FlatBuffer with the schema version hash.
// The returned slice has format: [32-byte hash][payload].
//
// Takes payload ([]byte) which contains the serialised FlatBuffer data.
//
// Returns []byte which contains the hash-prefixed payload ready for storage.
func Pack(payload []byte) []byte {
	return fbs.PackAlloc(SchemaHash, payload)
}

// PackInto writes the schema hash and payload into dst.
//
// Takes dst ([]byte) which is the destination buffer that must have length
// greater than or equal to fbs.PackedSize(len(payload)).
// Takes payload ([]byte) which is the data to pack after the schema hash.
//
// Returns int which is the number of bytes written.
//
// Use this zero-allocation variant in hot paths where the buffer is reused.
func PackInto(dst, payload []byte) int {
	return fbs.Pack(dst, SchemaHash, payload)
}

// Unpack validates the schema hash and returns the raw FlatBuffer payload.
//
// The returned slice is a zero-copy view into the original data.
//
// Takes data ([]byte) which contains the packed FlatBuffer with schema hash.
//
// Returns []byte which is the raw FlatBuffer payload.
// Returns error when the stored hash does not match the current schema version,
// returning fbs.ErrSchemaVersionMismatch to indicate stale cache data.
func Unpack(data []byte) ([]byte, error) {
	return fbs.Unpack(SchemaHash, data)
}

// Validate checks if data was serialised with the current schema version.
// This is a fast check that does not extract the payload.
//
// Takes data ([]byte) which is the serialised data to check.
//
// Returns bool which is true if the data matches the current schema version.
func Validate(data []byte) bool {
	return fbs.ValidateHash(SchemaHash, data)
}
