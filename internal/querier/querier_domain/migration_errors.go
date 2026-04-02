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

package querier_domain

import (
	"errors"
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

var (
	// ErrLockNotAcquired is returned when a non-blocking lock attempt fails
	// because another process already holds the migration lock.
	ErrLockNotAcquired = errors.New("migration lock is already held")

	// ErrNoDownMigration is returned when a rollback is requested for a
	// migration version that has no corresponding .down.sql file.
	ErrNoDownMigration = errors.New("no down migration file")
)

// ChecksumMismatchError is returned when an applied migration's recorded
// checksum does not match the current file on disk.
type ChecksumMismatchError struct {
	// AppliedChecksum holds the checksum recorded when the migration was applied.
	AppliedChecksum string

	// FileChecksum holds the current checksum of the migration file on disk.
	FileChecksum string

	// Name holds the human-readable name of the migration.
	Name string

	// Version holds the numeric version of the migration.
	Version int64
}

// Error returns a human-readable message describing the checksum mismatch.
//
// Returns string which contains the version, name, applied checksum, and file
// checksum.
func (e *ChecksumMismatchError) Error() string {
	return fmt.Sprintf(
		"checksum mismatch for migration %d (%s): applied=%s file=%s",
		e.Version, e.Name, e.AppliedChecksum, e.FileChecksum,
	)
}

// DownChecksumMismatchError is returned when a down migration file's checksum
// does not match the checksum recorded when the up migration was applied.
type DownChecksumMismatchError struct {
	// RecordedChecksum holds the down checksum recorded when the up migration
	// was applied.
	RecordedChecksum string

	// FileChecksum holds the current checksum of the down migration file on
	// disk.
	FileChecksum string

	// Name holds the human-readable name of the migration.
	Name string

	// Version holds the numeric version of the migration.
	Version int64
}

// Error returns a human-readable message describing the down checksum
// mismatch.
//
// Returns string which contains the version, name, recorded checksum, and file
// checksum.
func (e *DownChecksumMismatchError) Error() string {
	return fmt.Sprintf(
		"down checksum mismatch for migration %d (%s): recorded=%s file=%s",
		e.Version, e.Name, e.RecordedChecksum, e.FileChecksum,
	)
}

// MigrationExecutionError wraps an error from executing a migration's SQL
// content, carrying the migration identity and direction.
type MigrationExecutionError struct {
	// Cause holds the underlying error from migration execution.
	Cause error

	// Name holds the human-readable name of the migration.
	Name string

	// Version holds the numeric version of the migration.
	Version int64

	// Direction holds whether the migration was an up or down operation.
	Direction querier_dto.MigrationDirection
}

// Error returns a human-readable message describing the execution failure.
//
// Returns string which contains the direction label, version, name, and
// underlying cause.
func (e *MigrationExecutionError) Error() string {
	label := "migration"
	if e.Direction == querier_dto.MigrationDirectionDown {
		label = "rollback"
	}
	return fmt.Sprintf("%s %d (%s): %v", label, e.Version, e.Name, e.Cause)
}

// Unwrap returns the underlying cause for errors.Is/errors.As.
//
// Returns error which is the wrapped cause of the execution failure.
func (e *MigrationExecutionError) Unwrap() error {
	return e.Cause
}

// LockAcquisitionError wraps a failure to acquire the migration advisory lock.
type LockAcquisitionError struct {
	// Cause holds the underlying error from the lock acquisition attempt.
	Cause error
}

// Error returns a human-readable message describing the lock failure.
//
// Returns string which contains the underlying cause of the lock failure.
func (e *LockAcquisitionError) Error() string {
	return fmt.Sprintf("acquiring migration lock: %v", e.Cause)
}

// Unwrap returns the underlying cause for errors.Is/errors.As.
//
// Returns error which is the wrapped cause of the lock acquisition failure.
func (e *LockAcquisitionError) Unwrap() error {
	return e.Cause
}

// MissingMigrationFileError is returned when the database records an applied
// migration but no corresponding file exists on disk.
type MissingMigrationFileError struct {
	// Name holds the human-readable name of the missing migration.
	Name string

	// Version holds the numeric version of the missing migration.
	Version int64
}

// Error returns a human-readable message describing the missing file.
//
// Returns string which contains the version and name of the missing migration.
func (e *MissingMigrationFileError) Error() string {
	return fmt.Sprintf(
		"applied migration %d (%s) has no corresponding file on disk",
		e.Version, e.Name,
	)
}

// NoDownMigrationError is returned when a rollback is requested for a specific
// version that has no .down.sql file.
type NoDownMigrationError struct {
	// Version holds the numeric version of the migration missing a down file.
	Version int64
}

// Error returns a human-readable message describing the missing down file.
//
// Returns string which contains the version number.
func (e *NoDownMigrationError) Error() string {
	return fmt.Sprintf("no down migration for version %d", e.Version)
}

// Is reports whether target matches ErrNoDownMigration.
//
// Takes target (error) which specifies the error to compare against.
//
// Returns bool which is true if target is ErrNoDownMigration.
func (*NoDownMigrationError) Is(target error) bool {
	return target == ErrNoDownMigration
}

// DirtyMigrationError is returned when a dirty (partially-applied) migration
// blocks further progress. If the dirty migration matches the next pending
// version it can be retried automatically; otherwise manual intervention is
// required.
type DirtyMigrationError struct {
	// Version holds the numeric version of the dirty migration.
	Version int64

	// LastStatement holds the 0-based index of the last successfully applied
	// statement, or -1 if no statements completed.
	LastStatement int
}

// Error returns a human-readable message describing the dirty migration.
//
// Returns string which contains the version and last completed statement.
func (e *DirtyMigrationError) Error() string {
	return fmt.Sprintf(
		"migration %d is dirty (last completed statement: %d); "+
			"resolve manually or retry",
		e.Version, e.LastStatement,
	)
}
