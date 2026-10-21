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

package db_provider_seagull

import (
	"errors"

	"piko.sh/piko/internal/sqldb/sqldb_domain"

	"seagull.sh/seagull"
	"seagull.sh/seagull/sdk/migrate"
)

var (
	// sentinelPairs maps Seagull errors to Piko equivalents, most specific first.
	//
	// Errors without an equivalent pass through unchanged.
	sentinelPairs = []struct {
		seagull error
		piko    error
	}{
		{seagull: migrate.ErrLockNotAcquired, piko: sqldb_domain.ErrLockNotAcquired},
		{seagull: migrate.ErrInvalidIdentifier, piko: sqldb_domain.ErrInvalidIdentifier},
		{seagull: seagull.ErrFileTooLarge, piko: sqldb_domain.ErrMigrationFileTooLarge},
		{seagull: seagull.ErrMalformedSQL, piko: sqldb_domain.ErrMalformedSQLStatement},
	}
)

// translateError attaches Piko's equivalent to a Seagull error, so callers can match it
// with Piko's sentinels and typed errors while the message stays Seagull's.
//
// Takes err (error) which may be nil.
//
// Returns error which is nil for nil, the original when Piko has no equivalent, and a
// *sqldb_domain.TranslatedError otherwise.
func translateError(err error) error {
	if err == nil {
		return nil
	}
	if piko := translateTypedError(err); piko != nil {
		return &sqldb_domain.TranslatedError{Piko: piko, Original: err}
	}
	for _, pair := range sentinelPairs {
		if errors.Is(err, pair.seagull) {
			return &sqldb_domain.TranslatedError{Piko: pair.piko, Original: err}
		}
	}
	return err
}

// translateTypedError copies a Seagull typed error into Piko's equivalent.
//
// Takes err (error) which is searched for a Seagull typed error.
//
// Returns error which is Piko's typed error, or nil when there is none.
func translateTypedError(err error) error {
	if piko := translateMigrationTypedError(err); piko != nil {
		return piko
	}
	if piko := translateDirtyMigrationError(err); piko != nil {
		return piko
	}
	return translateSeedTypedError(err)
}

// translateMigrationTypedError copies a Seagull migration typed error.
//
// Takes err (error) which is searched for a Seagull migration typed error.
//
// Returns error which is Piko's typed error, or nil when there is none.
func translateMigrationTypedError(err error) error {
	if typed, ok := errors.AsType[*migrate.ChecksumMismatchError](err); ok {
		return &sqldb_domain.ChecksumMismatchError{AppliedChecksum: typed.AppliedChecksum, FileChecksum: typed.FileChecksum, Name: typed.Name, Version: typed.Version}
	}
	if typed, ok := errors.AsType[*migrate.DownChecksumMismatchError](err); ok {
		return &sqldb_domain.DownChecksumMismatchError{RecordedChecksum: typed.RecordedChecksum, FileChecksum: typed.FileChecksum, Name: typed.Name, Version: typed.Version}
	}
	if typed, ok := errors.AsType[*migrate.MigrationExecutionError](err); ok {
		return &sqldb_domain.MigrationExecutionError{Cause: typed.Cause, Name: typed.Name, Version: typed.Version, Direction: fromSeagullDirection(typed.Direction)}
	}
	if typed, ok := errors.AsType[*migrate.LockAcquisitionError](err); ok {
		return &sqldb_domain.LockAcquisitionError{Cause: typed.Cause}
	}
	if typed, ok := errors.AsType[*migrate.MissingMigrationFileError](err); ok {
		return &sqldb_domain.MissingMigrationFileError{Name: typed.Name, Version: typed.Version}
	}
	if typed, ok := errors.AsType[*migrate.NoDownMigrationError](err); ok {
		return &sqldb_domain.NoDownMigrationError{Version: typed.Version}
	}
	if typed, ok := errors.AsType[*migrate.DuplicateMigrationVersionError](err); ok {
		return &sqldb_domain.DuplicateMigrationVersionError{
			FirstFilename:  typed.FirstFilename,
			SecondFilename: typed.SecondFilename,
			Version:        typed.Version,
			Direction:      fromSeagullDirection(typed.Direction),
		}
	}
	return nil
}

// translateDirtyMigrationError converts interrupted and non-resumable Seagull migrations
// to Piko's DirtyMigrationError.
//
// Takes err (error) which is searched for a Seagull dirty-migration error.
//
// Returns error which is Piko's DirtyMigrationError, or nil when there is none.
func translateDirtyMigrationError(err error) error {
	if typed, ok := errors.AsType[*migrate.DirtyMigrationError](err); ok {
		return &sqldb_domain.DirtyMigrationError{Version: typed.Version, LastStatement: typed.LastStatement}
	}
	if typed, ok := errors.AsType[*migrate.ResumeMismatchError](err); ok {
		return &sqldb_domain.DirtyMigrationError{Version: typed.Version, LastStatement: typed.LastStatement}
	}
	return nil
}

// translateSeedTypedError copies a Seagull seed typed error.
//
// Takes err (error) which is searched for a Seagull seed typed error.
//
// Returns error which is Piko's typed error, or nil when there is none.
func translateSeedTypedError(err error) error {
	if typed, ok := errors.AsType[*migrate.DuplicateSeedVersionError](err); ok {
		return &sqldb_domain.DuplicateSeedVersionError{FirstFilename: typed.FirstFilename, SecondFilename: typed.SecondFilename, Version: typed.Version}
	}
	if typed, ok := errors.AsType[*migrate.SeedExecutionError](err); ok {
		return &sqldb_domain.SeedExecutionError{Cause: typed.Cause, Name: typed.Name, Version: typed.Version}
	}
	return nil
}
