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

package capabilities_domain

import (
	"errors"
	"fmt"
)

var (
	// errCapabilityNotFound is returned when a requested capability does not exist.
	errCapabilityNotFound = errors.New("capability: not found")

	// errCapabilityExists is returned when attempting to register a capability
	// that has already been registered.
	errCapabilityExists = errors.New("capability: already exists")

	// ErrFatal signals that a capability failed permanently and retrying will
	// not help. Capabilities wrap errors with this to indicate deterministic
	// failures such as parse errors or invalid input.
	ErrFatal = errors.New("capability: fatal error")
)

// NewFatalError wraps an error to mark it as a fatal capability failure that
// should not be retried.
//
// Takes err (error) which is the underlying error to mark as fatal.
//
// Returns error which wraps both the original error and the ErrFatal sentinel.
func NewFatalError(err error) error {
	return fmt.Errorf("%w: %w", err, ErrFatal)
}

// IsFatalError reports whether err or any error in its chain is a fatal
// capability error.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true when the error chain contains ErrFatal.
func IsFatalError(err error) bool {
	return errors.Is(err, ErrFatal)
}
