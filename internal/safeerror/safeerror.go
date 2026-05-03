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

package safeerror

import (
	"errors"
	"fmt"
)

// Error is an error that separates a user-safe message from internal details.
// In production, only SafeMessage reaches the user; in development, the full
// Error string (including wrapped causes) is shown instead.
//
// Any error type can implement Error by adding a single method.
// The standard error chain (errors.Is, errors.As, Unwrap) is unaffected.
type Error interface {
	error

	// SafeMessage returns the user-safe message suitable for
	// HTTP responses and error pages in production.
	//
	// Returns string which is the sanitised message.
	SafeMessage() string
}

// safeError is the concrete implementation returned by NewError
// and Errorf.
type safeError struct {
	// cause is the underlying error with internal details.
	cause error

	// safeMessage is the user-safe message suitable for HTTP
	// responses and error pages in production.
	safeMessage string
}

// Error returns the internal error message from the cause chain.
//
// Returns string which is the detail that gets logged but never
// shown to users in production.
func (e *safeError) Error() string { return e.cause.Error() }

// SafeMessage returns the user-safe message suitable for HTTP
// responses and error pages in production.
//
// Returns string which is the sanitised message.
func (e *safeError) SafeMessage() string { return e.safeMessage }

// Unwrap returns the underlying cause, preserving the error chain
// for errors.Is and errors.As.
//
// Returns error which is the wrapped cause.
func (e *safeError) Unwrap() error { return e.cause }

// NewError wraps cause with a user-safe message.
//
// Takes safeMessage (string) which is the message safe for end users.
// Takes cause (error) which is the underlying error with internal details.
//
// Returns error which implements Error with both safe and internal messages.
func NewError(safeMessage string, cause error) error {
	return &safeError{
		safeMessage: safeMessage,
		cause:       cause,
	}
}

// Errorf wraps a formatted internal error with a user-safe message. The
// internal cause is created via fmt.Errorf, so %%w wrapping is supported.
//
// Takes safeMessage (string) which is the message safe for end users.
// Takes format (string) which is the fmt.Errorf format string for the
// internal cause.
// Takes args (...any) which are the format arguments.
//
// Returns error which implements Error with both safe and internal messages.
func Errorf(safeMessage string, format string, args ...any) error {
	return &safeError{
		safeMessage: safeMessage,
		cause:       fmt.Errorf(format, args...),
	}
}

// ExtractSafeMessage returns the appropriate error message for the
// given mode.
//
// Takes err (error) which is the error to extract a message from.
// Takes developmentMode (bool) which controls whether internal details
// are returned.
//
// Returns string which is the message suitable for the current mode.
func ExtractSafeMessage(err error, developmentMode bool) string {
	if developmentMode {
		return err.Error()
	}

	if safeErr, ok := errors.AsType[Error](err); ok {
		return safeErr.SafeMessage()
	}

	return "An internal error occurred"
}
