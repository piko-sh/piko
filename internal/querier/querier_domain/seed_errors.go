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
	"fmt"
)

// DuplicateSeedVersionError is returned when two seed files share the same version (for
// example two 001_*.sql files). The downstream version-keyed map would otherwise silently
// drop one of them; this names both files so the collision is resolved before any seed
// runs.
type DuplicateSeedVersionError struct {
	// FirstFilename holds the filename first seen for the version.
	FirstFilename string

	// SecondFilename holds the conflicting filename for the same version.
	SecondFilename string

	// Version holds the shared numeric version.
	Version int64
}

// Error returns a human-readable message naming both conflicting files.
//
// Returns string which contains the version and both filenames.
func (e *DuplicateSeedVersionError) Error() string {
	return fmt.Sprintf(
		"duplicate seed for version %d: %s and %s",
		e.Version, e.FirstFilename, e.SecondFilename,
	)
}

// SeedExecutionError wraps an error from executing a seed's SQL content, carrying the
// seed identity for diagnostics.
type SeedExecutionError struct {
	// Cause holds the underlying error from seed execution.
	Cause error

	// Name holds the descriptive name of the seed.
	Name string

	// Version holds the numeric version of the seed.
	Version int64
}

// Error returns a human-readable message describing the execution failure.
//
// Returns string which contains the version, name, and underlying cause.
func (e *SeedExecutionError) Error() string {
	return fmt.Sprintf("seed %d (%s): %v", e.Version, e.Name, e.Cause)
}

// Unwrap returns the underlying cause for errors.Is/errors.As.
//
// Returns error which is the wrapped cause of the execution failure.
func (e *SeedExecutionError) Unwrap() error {
	return e.Cause
}
