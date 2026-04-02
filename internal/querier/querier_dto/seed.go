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

import "time"

// SeedFile represents a parsed seed file with version, name, and content.
type SeedFile struct {
	// Name holds the descriptive segment of the filename (e.g. "demo_authors").
	Name string

	// Filename holds the original base name for error reporting.
	Filename string

	// Checksum holds the SHA-256 hex digest of Content.
	Checksum string

	// Content holds the raw SQL bytes of the seed file.
	Content []byte

	// Version holds the numeric prefix parsed from the filename.
	Version int64
}

// AppliedSeed represents a seed that has been applied to the database.
type AppliedSeed struct {
	// AppliedAt holds the timestamp when the seed was applied.
	AppliedAt time.Time

	// Name holds the descriptive name of the seed.
	Name string

	// Checksum holds the SHA-256 hex digest recorded at application time.
	Checksum string

	// Version holds the numeric version of the seed.
	Version int64

	// DurationMs holds the execution duration in milliseconds.
	DurationMs int64
}

// SeedStatus combines a seed file with its applied state.
type SeedStatus struct {
	// AppliedAt holds the timestamp when the seed was applied, or the zero
	// value if not yet applied.
	AppliedAt time.Time

	// Name holds the descriptive name of the seed.
	Name string

	// Filename holds the original base name of the seed file.
	Filename string

	// Version holds the numeric version of the seed.
	Version int64

	// Applied indicates whether the seed has been applied.
	Applied bool

	// ChecksumMatch indicates whether the applied checksum matches the
	// current file on disk. Always true for unapplied seeds.
	ChecksumMatch bool
}

// SeedRecord holds the data needed to execute and record a single seed.
type SeedRecord struct {
	// Name holds the descriptive name of the seed.
	Name string

	// Checksum holds the SHA-256 hex digest of Content.
	Checksum string

	// Content holds the raw SQL bytes to execute.
	Content []byte

	// Version holds the numeric version of the seed.
	Version int64
}
