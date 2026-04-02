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

package wal_domain

import "errors"

var (
	// ErrCorrupted indicates the WAL contains invalid data that failed CRC validation.
	// During recovery, the WAL is truncated at the corruption point, preserving
	// valid entries before it.
	ErrCorrupted = errors.New("wal: corrupted entry detected")

	// ErrWALClosed indicates an operation was attempted on a closed WAL.
	ErrWALClosed = errors.New("wal: already closed")

	// ErrInvalidVersion indicates an unsupported WAL format version.
	// This typically occurs when trying to read a WAL created by a newer version
	// of the software.
	ErrInvalidVersion = errors.New("wal: unsupported format version")

	// ErrSnapshotNotFound indicates no snapshot file exists.
	// This is not necessarily an error during recovery; it may simply mean
	// this is a fresh start.
	ErrSnapshotNotFound = errors.New("wal: snapshot not found")

	// ErrInvalidConfig indicates the WAL configuration is invalid.
	ErrInvalidConfig = errors.New("wal: invalid configuration")

	// ErrInvalidEntry indicates an entry could not be encoded or decoded
	// due to invalid data (separate from CRC corruption).
	ErrInvalidEntry = errors.New("wal: invalid entry")

	// ErrCodecRequired indicates a codec must be provided for the WAL.
	ErrCodecRequired = errors.New("wal: codec is required")

	// ErrWriterInBadState indicates the WAL encountered a fatal I/O error and
	// rejects all subsequent writes. The WAL file may be in an inconsistent
	// state and should not be written to further.
	ErrWriterInBadState = errors.New("wal: writer in bad state after I/O failure")
)
