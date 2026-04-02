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

import "piko.sh/piko/internal/fbs"

// SchemaHash is a fixed-size array holding a SHA-256 hash of a schema file.
type SchemaHash = fbs.SchemaHash

var (
	// ErrSchemaVersionMismatch is returned when stored data was saved with a
	// different schema version.
	ErrSchemaVersionMismatch = fbs.ErrSchemaVersionMismatch

	// ComputeSchemaHash computes a SHA-256 hash of schema file content.
	ComputeSchemaHash = fbs.ComputeSchemaHash

	// Unpack checks the schema hash and returns the payload slice.
	Unpack = fbs.Unpack

	// ValidateHash checks if data starts with the expected schema hash.
	ValidateHash = fbs.ValidateHash
)
