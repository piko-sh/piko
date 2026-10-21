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

package sqldb_domain

import (
	"errors"
	"fmt"

	"piko.sh/piko/internal/sqldb/sqldb_dto"
)

var (
	// ErrLockNotAcquired is returned when a non-blocking run finds the migration lock
	// already held.
	ErrLockNotAcquired = errors.New("migration lock is already held")

	// ErrNoDownMigration is returned when a rollback needs a down migration that does not
	// exist.
	ErrNoDownMigration = errors.New("no down migration file")

	// ErrInvalidIdentifier is returned when a table or lock name is not a safe SQL
	// identifier.
	ErrInvalidIdentifier = errors.New("invalid SQL identifier")

	// ErrMigrationFileTooLarge is returned when a migration or seed file exceeds the size
	// limit.
	ErrMigrationFileTooLarge = errors.New("migration file exceeds configured size limit")

	// ErrMalformedSQLStatement is returned when a migration cannot be split into statements,
	// such as an unterminated string or comment.
	ErrMalformedSQLStatement = errors.New("malformed SQL statement")

	// ErrCountRewriteNoSelect is returned when a paginated query has no SELECT to rewrite
	// into a count query.
	ErrCountRewriteNoSelect = errors.New("count rewrite: input is not a SELECT statement")

	// ErrAsyncExecNotSupported is returned when asyncexec is used with an engine that has no
	// asynchronous mutation semantics.
	ErrAsyncExecNotSupported = errors.New("asyncexec is only supported on engines that surface asynchronous mutation semantics")

	// ErrNoMigrationBackend is returned when a migration or seed service is requested for a
	// dialect that has no migration backend.
	ErrNoMigrationBackend = errors.New("dialect has no migration backend")

	// ErrNoSQLToolchain is returned when a database is used for code generation or
	// migrations without an engine that supplies a toolchain.
	ErrNoSQLToolchain = errors.New("database has no SQL engine; set EngineConfig from a wdk/db/db_engine_* module")

	// ErrUnsupportedDriver is returned when the toolchain cannot generate code for the
	// requested driver API.
	ErrUnsupportedDriver = errors.New("generated driver is not supported by this toolchain")
)

// ChecksumMismatchError reports an applied migration whose file has changed since.
type ChecksumMismatchError struct {
	// AppliedChecksum is the checksum recorded when the migration was applied.
	AppliedChecksum string

	// FileChecksum is the checksum of the file now.
	FileChecksum string

	// Name is the migration's name.
	Name string

	// Version is the migration's version number.
	Version int64
}

// Error describes the mismatch.
//
// Returns string which names the migration and both checksums.
func (e *ChecksumMismatchError) Error() string {
	return fmt.Sprintf(
		"checksum mismatch for migration %d (%s): applied=%s file=%s",
		e.Version, e.Name, e.AppliedChecksum, e.FileChecksum,
	)
}

// DownChecksumMismatchError reports a down migration that changed after its up migration
// was applied.
type DownChecksumMismatchError struct {
	// RecordedChecksum is the down checksum recorded when the up migration was applied.
	RecordedChecksum string

	// FileChecksum is the checksum of the down file now.
	FileChecksum string

	// Name is the migration's name.
	Name string

	// Version is the migration's version number.
	Version int64
}

// Error describes the mismatch.
//
// Returns string which names the migration and both checksums.
func (e *DownChecksumMismatchError) Error() string {
	return fmt.Sprintf(
		"down checksum mismatch for migration %d (%s): recorded=%s file=%s",
		e.Version, e.Name, e.RecordedChecksum, e.FileChecksum,
	)
}

// MigrationExecutionError reports a migration or rollback that failed to run.
type MigrationExecutionError struct {
	// Cause is the database error.
	Cause error

	// Name is the migration's name.
	Name string

	// Version is the migration's version number.
	Version int64

	// Direction says whether the migration was being applied or reverted.
	Direction sqldb_dto.MigrationDirection
}

// Error describes the failure.
//
// Returns string which names the migration and the cause.
func (e *MigrationExecutionError) Error() string {
	label := "migration"
	if e.Direction == sqldb_dto.DirectionDown {
		label = "rollback"
	}
	return fmt.Sprintf("%s %d (%s): %v", label, e.Version, e.Name, e.Cause)
}

// Unwrap returns the database error.
//
// Returns error which is the cause.
func (e *MigrationExecutionError) Unwrap() error {
	return e.Cause
}

// LockAcquisitionError reports a failure to take the migration lock.
type LockAcquisitionError struct {
	// Cause is the underlying error.
	Cause error
}

// Error describes the failure.
//
// Returns string which includes the cause.
func (e *LockAcquisitionError) Error() string {
	return fmt.Sprintf("acquiring migration lock: %v", e.Cause)
}

// Unwrap returns the underlying error.
//
// Returns error which is the cause.
func (e *LockAcquisitionError) Unwrap() error {
	return e.Cause
}

// MissingMigrationFileError reports an applied migration whose file no longer exists.
type MissingMigrationFileError struct {
	// Name is the migration's name.
	Name string

	// Version is the migration's version number.
	Version int64
}

// Error describes the missing file.
//
// Returns string which names the migration.
func (e *MissingMigrationFileError) Error() string {
	return fmt.Sprintf("applied migration %d (%s) has no corresponding file on disk", e.Version, e.Name)
}

// NoDownMigrationError reports a rollback that needs a missing down migration.
type NoDownMigrationError struct {
	// Version is the migration's version number.
	Version int64
}

// Error describes the missing down migration.
//
// Returns string which names the version.
func (e *NoDownMigrationError) Error() string {
	return fmt.Sprintf("no down migration for version %d", e.Version)
}

// Is reports whether target is ErrNoDownMigration.
//
// Takes target (error) which is compared.
//
// Returns bool which is true for ErrNoDownMigration.
func (*NoDownMigrationError) Is(target error) bool {
	return target == ErrNoDownMigration
}

// DuplicateMigrationVersionError reports two migration files with the same version and
// direction.
type DuplicateMigrationVersionError struct {
	// FirstFilename is one of the clashing files.
	FirstFilename string

	// SecondFilename is the other clashing file.
	SecondFilename string

	// Version is the shared version number.
	Version int64

	// Direction is the shared direction.
	Direction sqldb_dto.MigrationDirection
}

// Error describes the clash.
//
// Returns string which names both files.
func (e *DuplicateMigrationVersionError) Error() string {
	direction := "up"
	if e.Direction == sqldb_dto.DirectionDown {
		direction = "down"
	}
	return fmt.Sprintf("duplicate %s migration for version %d: %s and %s", direction, e.Version, e.FirstFilename, e.SecondFilename)
}

// DirtyMigrationError reports a migration that stopped part-way through.
type DirtyMigrationError struct {
	// Version is the migration's version number.
	Version int64

	// LastStatement is the index of the last statement that completed.
	LastStatement int
}

// Error describes the dirty migration.
//
// Returns string which names the version and the last completed statement.
func (e *DirtyMigrationError) Error() string {
	return fmt.Sprintf(
		"migration %d is dirty (last completed statement: %d); resolve manually or retry",
		e.Version, e.LastStatement,
	)
}

// DuplicateSeedVersionError reports two seed files with the same version.
type DuplicateSeedVersionError struct {
	// FirstFilename is one of the clashing files.
	FirstFilename string

	// SecondFilename is the other clashing file.
	SecondFilename string

	// Version is the shared version number.
	Version int64
}

// Error describes the clash.
//
// Returns string which names both files.
func (e *DuplicateSeedVersionError) Error() string {
	return fmt.Sprintf("duplicate seed for version %d: %s and %s", e.Version, e.FirstFilename, e.SecondFilename)
}

// SeedExecutionError reports a seed that failed to run.
type SeedExecutionError struct {
	// Cause is the database error.
	Cause error

	// Name is the seed's name.
	Name string

	// Version is the seed's version number.
	Version int64
}

// Error describes the failure.
//
// Returns string which names the seed and the cause.
func (e *SeedExecutionError) Error() string {
	return fmt.Sprintf("seed %d (%s): %v", e.Version, e.Name, e.Cause)
}

// Unwrap returns the database error.
//
// Returns error which is the cause.
func (e *SeedExecutionError) Unwrap() error {
	return e.Cause
}

// TranslatedError carries an error from the toolchain together with Piko's equivalent, so
// errors.Is and errors.As match both vocabularies while the message stays the toolchain's
// own.
type TranslatedError struct {
	// Piko is Piko's equivalent sentinel or typed error.
	Piko error

	// Original is the error the toolchain returned.
	Original error
}

// Error returns the toolchain's message.
//
// Returns string which is the original error's message.
func (e *TranslatedError) Error() string {
	return e.Original.Error()
}

// Unwrap exposes both errors to errors.Is and errors.As.
//
// Returns []error which holds Piko's error first, then the original.
func (e *TranslatedError) Unwrap() []error {
	return []error{e.Piko, e.Original}
}
