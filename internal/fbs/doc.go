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

// Package fbs handles versioned FlatBuffer serialisation with
// automatic cache invalidation via schema-hash prefixes.
//
// Each serialised blob is prefixed with a 32-byte SHA-256 hash of the
// schema file. On deserialisation, the hash is validated; a mismatch
// means the data is stale and came from an older schema version.
//
// # Binary format
//
//	+----------------------+-----------------------------+
//	|  Schema Hash (32B)   |  FlatBuffer Payload (var)   |
//	+----------------------+-----------------------------+
//
// # Usage
//
// Compute a schema hash once at initialisation:
//
//	//go:embed schema.fbs
//	var schemaBytes []byte
//	var schemaHash = fbs.ComputeSchemaHash(schemaBytes)
//
// Pack data for storage:
//
//	payload := builder.FinishedBytes()
//	versioned := fbs.PackAlloc(schemaHash, payload)
//	cache.Set(key, versioned)
//
// Unpack and validate on retrieval:
//
//	data, _ := cache.Get(key)
//	payload, err := fbs.Unpack(schemaHash, data)
//	if errors.Is(err, fbs.ErrSchemaVersionMismatch) {
//	    // Treat as cache miss and regenerate the data
//	}
//
// # Integration
//
// Used across Piko's caching infrastructure to version FlatBuffer-serialised data
// including AST nodes, search indices, i18n bundles, and collection manifests.
// Each schema module (e.g., ast/schema, search/search_schema) maintains its own
// embedded schema hash.
//
// # Debugging versioned files
//
// The hash prefix means standard flatc inspection will not work
// directly. Strip the first 32 bytes before passing the file to
// flatc for inspection.
//
// # Thread safety
//
// All functions are stateless and safe for concurrent use.
package fbs
