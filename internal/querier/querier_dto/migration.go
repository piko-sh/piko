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

// MigrationDirection indicates whether a migration file is a forward (up) or
// rollback (down) migration.
type MigrationDirection uint8

const (
	// MigrationDirectionUp is a forward migration that applies schema changes.
	MigrationDirectionUp MigrationDirection = iota

	// MigrationDirectionDown is a rollback migration that reverts schema changes.
	MigrationDirectionDown
)

// MigrationFile represents a parsed migration file with version, direction,
// and content. Files follow the {version}_{name}.{up|down}.sql naming
// convention.
type MigrationFile struct {
	// Name is the descriptive segment extracted from the filename (e.g.
	// "create_users" from "0001_create_users.up.sql").
	Name string

	// Filename is the original filename for error reporting.
	Filename string

	// Checksum is the SHA-256 hex digest of Content, computed at read time.
	Checksum string

	// Content is the raw SQL content of the migration file.
	Content []byte

	// Version is the numeric prefix extracted from the filename. Stored as
	// int64 to support both sequential (0001) and timestamp-based
	// (20260324120000) versioning.
	Version int64

	// Direction indicates whether this is an up or down migration.
	Direction MigrationDirection
}

// AppliedMigration represents a migration that has been applied to the
// database, as recorded in the migration history table.
type AppliedMigration struct {
	// AppliedAt is when the migration was applied.
	AppliedAt time.Time

	// LastStatement is the 0-based index of the last successfully executed
	// statement within this migration. Nil means the migration completed
	// as a single unit without statement-level tracking (legacy behaviour).
	LastStatement *int

	// Name is the migration name.
	Name string

	// Checksum is the SHA-256 hex digest recorded when the migration was
	// applied.
	Checksum string

	// DownChecksum is the SHA-256 hex digest of the corresponding .down.sql
	// file at the time the up migration was applied. Empty if no down file
	// existed or the migration was applied before this feature.
	DownChecksum string

	// Version is the migration version number.
	Version int64

	// DurationMs is how long the migration took to execute in milliseconds.
	DurationMs int64

	// Dirty indicates the migration failed partway through and is in a
	// partially-applied state. A dirty migration must be resolved (retried
	// or manually fixed) before further migrations can be applied.
	Dirty bool
}

// MigrationRecord holds the data needed to execute and record a single
// migration.
type MigrationRecord struct {
	// Name is the migration name.
	Name string

	// Checksum is the SHA-256 hex digest of Content.
	Checksum string

	// DownChecksum is the SHA-256 hex digest of the corresponding .down.sql
	// file, if one exists. Empty when no down file is available.
	DownChecksum string

	// Content is the SQL to execute.
	Content []byte

	// Version is the migration version number.
	Version int64

	// SkipUpTo is the 0-based index of statements to skip when retrying a
	// dirty migration (-1 means execute all from the beginning).
	SkipUpTo int
}

// MigrationStatus combines a migration file with its applied state, used by
// the Status operation to present a unified view.
type MigrationStatus struct {
	// AppliedAt is when the migration was applied (zero value if not applied).
	AppliedAt time.Time

	// LastStatement is the 0-based index of the last successfully applied
	// statement. Only meaningful when Dirty is true.
	LastStatement *int

	// Name is the migration name.
	Name string

	// Filename is the source filename.
	Filename string

	// Version is the migration version number.
	Version int64

	// Applied indicates whether this migration has been applied.
	Applied bool

	// ChecksumMatch indicates whether the file checksum matches the recorded
	// checksum. Only meaningful when Applied is true.
	ChecksumMatch bool

	// HasDownMigration indicates whether a .down.sql file exists for this
	// version.
	HasDownMigration bool

	// Dirty indicates the migration is in a partially-applied state.
	// Only meaningful when Applied is true.
	Dirty bool
}
