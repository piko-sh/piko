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

package safedisk

import (
	"errors"
	"os"
)

var (
	// errReadOnly is returned when attempting a write operation on a read-only
	// sandbox.
	errReadOnly = errors.New("safedisk: sandbox is read-only")

	// errClosed is returned when an operation is attempted on a closed sandbox.
	errClosed = errors.New("safedisk: sandbox is closed")

	// errPathNotAllowed is returned when attempting to create a sandbox for a path
	// that is not in the allowed paths list.
	errPathNotAllowed = errors.New("safedisk: path is not in allowed paths")

	// errEmptyPath is returned when an empty path is given.
	errEmptyPath = errors.New("safedisk: path cannot be empty")

	// errTempFileExhausted is returned when CreateTemp fails to find a unique
	// filename after the maximum number of attempts.
	errTempFileExhausted = errors.New("safedisk: failed to create temp file after maximum attempts")
)

// IsNotExist reports whether the error shows that a file or directory does
// not exist.
//
// This is a wrapper around os.IsNotExist to avoid needing to import the os
// package separately.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if err shows the file does not exist.
func IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

// isExist reports whether the error shows that a file or directory already
// exists.
//
// This is a wrapper around os.IsExist so callers do not need to import the os
// package.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if the error indicates the file already exists.
func isExist(err error) bool {
	return os.IsExist(err)
}

// isPermission reports whether the error shows a permission problem.
//
// This is a wrapper around os.IsPermission so callers do not need to import
// the os package.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if the error indicates a permission problem.
func isPermission(err error) bool {
	return os.IsPermission(err)
}
