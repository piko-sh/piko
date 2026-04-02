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

package binder

import (
	"fmt"
	"strings"
)

// MultiError holds multiple binding errors from a single form.
// It implements the error interface.
type MultiError map[string]error

// Error implements the error interface for MultiError.
//
// Returns string which contains a formatted list of all validation errors
// with their field names, or "(0 errors)" when the map is empty.
func (m MultiError) Error() string {
	if len(m) == 0 {
		return "(0 errors)"
	}
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%d validation errors:\n", len(m))
	for key, err := range m {
		_, _ = fmt.Fprintf(&b, "  - field '%s': %v\n", key, err)
	}
	return b.String()
}

// SafeMessage returns a user-safe error message that does not expose
// field names or internal types.
//
// Returns string which is a generic validation failure message.
func (MultiError) SafeMessage() string { return "validation failed" }

// errInvalidTarget is returned when the destination for binding is not a
// pointer to a struct.
type errInvalidTarget struct {
	// targetType is the name of the type that was given instead of a struct
	// pointer.
	targetType string
}

// Error implements the error interface for errInvalidTarget.
//
// Returns string which describes the invalid target type that was given.
func (e errInvalidTarget) Error() string {
	return fmt.Sprintf("binder: destination must be a pointer to a struct, but got %s", e.targetType)
}

// SafeMessage returns a user-safe error message that does not expose
// internal type information.
//
// Returns string which is a generic binding failure message.
func (errInvalidTarget) SafeMessage() string { return "invalid form data" }

// errInvalidPath is returned when a key in the source map is not a valid Piko
// expression path.
type errInvalidPath struct {
	// err is the underlying error that explains why the path is invalid.
	err error

	// path is the file path that failed validation.
	path string
}

// Error implements the error interface for errInvalidPath.
//
// Returns string which contains the invalid path and the underlying error.
func (e errInvalidPath) Error() string {
	return fmt.Sprintf("binder: invalid path '%s': %v", e.path, e.err)
}

// SafeMessage returns a user-safe error message that does not expose
// internal path details.
//
// Returns string which is a generic binding failure message.
func (errInvalidPath) SafeMessage() string { return "invalid form data" }

// errSetField is returned when a value cannot be converted or set on a struct
// field.
type errSetField struct {
	// err is the underlying error from the failed field assignment.
	err error

	// path is the binder path where setting the field failed.
	path string

	// field is the name of the struct field that could not be set.
	field string

	// fieldType is the expected type name for the field.
	fieldType string
}

// Error implements the error interface for errSetField.
//
// Returns string which describes the field binding failure with context.
func (e errSetField) Error() string {
	return fmt.Sprintf("binder: could not set field '%s' for path '%s' (expected type %s): %v", e.field, e.path, e.fieldType, e.err)
}

// SafeMessage returns a user-safe error message that does not expose
// field names or internal type information.
//
// Returns string which is a generic binding failure message.
func (errSetField) SafeMessage() string { return "invalid form data" }
