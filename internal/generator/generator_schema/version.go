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

package generator_schema

import (
	_ "embed"

	"piko.sh/piko/internal/fbs"
)

//go:embed manifest.fbs
var schemaContent []byte

// SchemaHash is the SHA-256 hash of manifest.fbs, computed at init time.
// This hash changes whenever the schema file is modified, so the
// cache invalidates automatically when the schema evolves.
var SchemaHash = fbs.ComputeSchemaHash(schemaContent)

// Pack wraps a serialised Manifest FlatBuffer with the schema version hash.
// The returned slice has format: [32-byte hash][payload].
func Pack(payload []byte) []byte {
	return fbs.PackAlloc(SchemaHash, payload)
}

// PackInto writes the schema hash and payload into dst.
// dst must have length >= fbs.PackedSize(len(payload)).
// Returns the number of bytes written.
//
// Use this zero-allocation variant in hot paths where the buffer is reused.
func PackInto(dst, payload []byte) int {
	return fbs.Pack(dst, SchemaHash, payload)
}

// Unpack validates the schema hash and returns the raw FlatBuffer payload.
// Returns fbs.ErrSchemaVersionMismatch if the stored hash doesn't match
// the current schema version (indicating stale cache data).
//
// The returned slice is a zero-copy view into the original data.
func Unpack(data []byte) ([]byte, error) {
	return fbs.Unpack(SchemaHash, data)
}

// Validate checks if data was serialised with the current schema version.
// This is a fast check that doesn't extract the payload.
func Validate(data []byte) bool {
	return fbs.ValidateHash(SchemaHash, data)
}
