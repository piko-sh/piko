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

package pml_domain

import (
	"fmt"

	"piko.sh/piko/internal/ast/ast_domain"
)

// Severity defines the level of importance for a diagnostic message.
type Severity string

const (
	// SeverityError indicates a fatal problem that should stop the build in
	// 'strict' mode.
	SeverityError Severity = "error"

	// SeverityWarning indicates a potential issue or bad practice that does not
	// stop the build.
	SeverityWarning Severity = "warning"
)

// Error represents a single diagnostic (error or warning) generated during
// PikoML validation or transformation. It implements the standard error
// interface.
type Error struct {
	// Message is the human-readable description of the error.
	Message string

	// TagName is the PikoML tag where the error occurred.
	TagName string

	// Severity indicates how important this error is (error or warning).
	Severity Severity

	// Location specifies the line and column where the error occurred.
	Location ast_domain.Location
}

// Error implements the standard Go error interface.
//
// Returns string which contains the formatted error with severity, tag name,
// line, column, and message.
func (e *Error) Error() string {
	return fmt.Sprintf("PikoML %s in <%s> at L%d:C%d: %s",
		e.Severity, e.TagName, e.Location.Line, e.Location.Column, e.Message)
}

// newError creates a new Error with the given details.
//
// Takes message (string) which provides the error description.
// Takes tagName (string) which identifies the tag that caused the error.
// Takes severity (Severity) which indicates how serious the error is.
// Takes location (ast_domain.Location) which specifies where the error
// occurred.
//
// Returns *Error which is the constructed error ready for use.
func newError(message, tagName string, severity Severity, location ast_domain.Location) *Error {
	return &Error{
		Location: location,
		Message:  message,
		TagName:  tagName,
		Severity: severity,
	}
}
