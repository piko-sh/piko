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

package security_domain

import "errors"

const (
	// csrfErrorCodeExpired indicates the CSRF cookie was rotated or the
	// safety-net timeout was exceeded. Frontend should refresh the partial
	// and retry the action.
	csrfErrorCodeExpired = "csrf_expired"

	// CSRFErrorCodeInvalid indicates the token format, signature, or binding
	// is not valid. This usually means the token was changed or there is a bug.
	CSRFErrorCodeInvalid = "csrf_invalid"

	// CSRFErrorCodeMissing indicates that CSRF tokens were expected but not
	// provided. This occurs when a browser request (identified by the
	// Sec-Fetch-Site header) omits CSRF tokens entirely.
	CSRFErrorCodeMissing = "csrf_missing"
)

var (
	// errMissingSecret indicates that the HMAC secret key is not configured for
	// CSRF.
	errMissingSecret = errors.New("security: HMAC secret key not configured for CSRF")

	// errEphemeralTokenGeneration indicates that the CSRF ephemeral token
	// generation failed.
	errEphemeralTokenGeneration = errors.New("security: failed to generate CSRF ephemeralToken ")

	// errInvalidCSRFTokenFormat is returned when a CSRF token has the wrong
	// format.
	errInvalidCSRFTokenFormat = errors.New("security: invalid CSRF token format")

	// errCSRFTokenSignature indicates that the CSRF token signature verification
	// failed.
	errCSRFTokenSignature = errors.New("security: CSRF token signature mismatch")

	// errCSRFTokenExpired indicates that the CSRF token has expired.
	errCSRFTokenExpired = errors.New("security: CSRF token expired")

	// errCSRFCookieMissing is returned when the CSRF cookie is missing from the
	// request.
	errCSRFCookieMissing = errors.New("security: CSRF cookie not found")

	// errCSRFCookieMismatch indicates the cookie value doesn't match the token.
	errCSRFCookieMismatch = errors.New("security: CSRF cookie value mismatch")

	// errCSRFEphemeralTokenMismatch indicates that the CSRF ephemeral token does
	// not match.
	errCSRFEphemeralTokenMismatch = errors.New("security: CSRF ephemeral token mismatch")

	// errCSRFBinderMismatch indicates that the CSRF token binder does not match.
	errCSRFBinderMismatch = errors.New("security: CSRF token binder mismatch")

	// errCSRFCookieSourceNil is returned when the cookie source adapter is nil.
	errCSRFCookieSourceNil = errors.New("security: CSRF cookie source adapter is nil")
)

// CSRFValidationError provides structured error information for CSRF failures.
// This enables the frontend to distinguish between different failure modes
// and take appropriate recovery actions (e.g., refresh partial on expiry).
type CSRFValidationError struct {
	// Err is the underlying error that caused this failure; nil if there is no
	// cause.
	Err error

	// Code is the error code for frontend consumption.
	// One of csrfErrorCodeExpired, CSRFErrorCodeInvalid, or CSRFErrorCodeMissing.
	Code string

	// Message is a description of the error that users can read.
	Message string
}

// Error implements the error interface.
//
// Returns string which contains the error message, including the wrapped
// error if present.
func (e *CSRFValidationError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error for errors.Is/As support.
//
// Returns error which is the wrapped error, or nil if none exists.
func (e *CSRFValidationError) Unwrap() error {
	return e.Err
}

// newCSRFValidationError creates a new CSRF validation error with the given
// details.
//
// Takes code (string) which identifies the type of validation failure.
// Takes message (string) which describes the error.
// Takes err (error) which is the cause of the failure.
//
// Returns *CSRFValidationError which wraps the error with CSRF context.
func newCSRFValidationError(code, message string, err error) *CSRFValidationError {
	return &CSRFValidationError{
		Err:     err,
		Code:    code,
		Message: message,
	}
}
