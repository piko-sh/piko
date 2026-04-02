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

package querier_dto

// Canonical type names used by the domain layer for type promotion, mapping,
// and default type construction. Engine adapters normalise dialect-specific
// type names (e.g. SQLite's "integer", "real") via NormaliseTypeName; the
// canonical names here are used in promoteWithinCategory rank maps and
// defaultMappings lookups.
//
// These are PostgreSQL-aligned because PostgreSQL has the richest built-in
// type system. Adapters for simpler engines (SQLite, MySQL) normalise to a
// subset of these names.
const (
	// CanonicalInt2 represents a 16-bit integer (smallint).
	CanonicalInt2 = "int2"

	// CanonicalInt4 represents a 32-bit integer (integer).
	CanonicalInt4 = "int4"

	// CanonicalInt8 represents a 64-bit integer (bigint).
	CanonicalInt8 = "int8"

	// CanonicalFloat4 represents a 32-bit float (real).
	CanonicalFloat4 = "float4"

	// CanonicalFloat8 represents a 64-bit float (double precision).
	CanonicalFloat8 = "float8"

	// CanonicalNumeric represents an arbitrary-precision decimal.
	CanonicalNumeric = "numeric"

	// CanonicalBoolean represents a boolean type.
	CanonicalBoolean = "boolean"

	// CanonicalText represents an unbounded text type.
	CanonicalText = "text"

	// CanonicalVarchar represents a variable-length character type.
	CanonicalVarchar = "varchar"

	// CanonicalChar represents a fixed-length character type.
	CanonicalChar = "char"

	// CanonicalDate represents a calendar date (no time component).
	CanonicalDate = "date"

	// CanonicalTime represents a time of day (no date component).
	CanonicalTime = "time"

	// CanonicalTimestamp represents a date and time without time zone.
	CanonicalTimestamp = "timestamp"

	// CanonicalTimestampTZ represents a date and time with time zone.
	CanonicalTimestampTZ = "timestamptz"

	// CanonicalInterval represents a time duration.
	CanonicalInterval = "interval"
)
