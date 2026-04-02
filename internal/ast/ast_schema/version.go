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

package ast_schema

import (
	_ "embed" // Required for go:embed directive.

	"piko.sh/piko/internal/fbs"
)

var (
	// schemaContent holds the embedded FlatBuffers schema from template_ast.fbs.
	schemaContent []byte

	// SchemaHash is the SHA-256 hash of template_ast.fbs, computed at init time.
	// This hash changes whenever the schema file is modified, so the
	// cache invalidates automatically when the schema evolves.
	SchemaHash = fbs.ComputeSchemaHash(schemaContent)
)

// Pack wraps a serialised AST FlatBuffer with the schema version hash.
//
// Takes payload ([]byte) which is the serialised FlatBuffer data to wrap.
//
// Returns []byte which has the format [32-byte hash][payload].
func Pack(payload []byte) []byte {
	return fbs.PackAlloc(SchemaHash, payload)
}

// PackInto writes the schema hash and payload into dst.
//
// The dst buffer must have length >= fbs.PackedSize(len(payload)). Use this
// zero-allocation variant in hot paths where the buffer is reused.
//
// Takes destination ([]byte) which is the destination buffer to write into.
// Takes payload ([]byte) which is the data to pack after the schema hash.
//
// Returns int which is the number of bytes written.
func PackInto(destination, payload []byte) int {
	return fbs.Pack(destination, SchemaHash, payload)
}

// Unpack validates the schema hash and returns the raw FlatBuffer payload.
//
// Takes data ([]byte) which contains the packed FlatBuffer data with schema
// hash prefix.
//
// Returns []byte which is a zero-copy view into the original data.
// Returns error when the stored hash does not match the current schema
// version, returning fbs.ErrSchemaVersionMismatch to indicate stale cache
// data.
func Unpack(data []byte) ([]byte, error) {
	return fbs.Unpack(SchemaHash, data)
}

// Validate checks if data was serialised with the current schema version.
// This is a fast check that does not extract the payload.
//
// Takes data ([]byte) which is the serialised data to validate.
//
// Returns bool which is true if the data matches the current schema version.
func Validate(data []byte) bool {
	return fbs.ValidateHash(SchemaHash, data)
}
