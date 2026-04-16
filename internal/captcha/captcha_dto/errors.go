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

package captcha_dto

import (
	"errors"
	"fmt"
)

var (
	// ErrCaptchaDisabled indicates captcha is not configured.
	ErrCaptchaDisabled = errors.New("captcha: not configured - use piko.WithCaptchaProvider()")

	// ErrVerificationFailed indicates the captcha token failed verification.
	ErrVerificationFailed = errors.New("captcha verification failed")

	// ErrTokenMissing indicates no captcha token was provided in the request.
	ErrTokenMissing = errors.New("captcha token missing")

	// ErrTokenExpired indicates the captcha token has expired.
	ErrTokenExpired = errors.New("captcha token expired")

	// ErrProviderUnavailable indicates the captcha provider is not available.
	ErrProviderUnavailable = errors.New("captcha provider unavailable")

	// ErrTokenTooLong indicates the captcha token exceeds the maximum allowed
	// length.
	ErrTokenTooLong = errors.New("captcha token exceeds maximum length")

	// ErrScoreBelowThreshold indicates the captcha score is below the required
	// threshold for score-based providers like reCAPTCHA v3.
	ErrScoreBelowThreshold = errors.New("captcha score below threshold")

	// ErrRateLimited indicates the client has exceeded the captcha verification
	// rate limit.
	ErrRateLimited = errors.New("captcha rate limit exceeded")

	// ErrActionTooLong indicates the action name exceeds the maximum allowed
	// length.
	ErrActionTooLong = errors.New("captcha action name exceeds maximum length")
)

// CaptchaError wraps a captcha error with additional context.
type CaptchaError struct {
	// Err is the underlying error that caused the failure.
	Err error

	// Operation is the operation that failed, such as "Verify".
	Operation string

	// Provider is the name of the captcha provider that encountered the error.
	Provider string
}

// NewCaptchaError creates a new CaptchaError with the given details.
//
// Takes operation (string) which specifies the operation that failed.
// Takes provider (string) which identifies the captcha provider.
// Takes err (error) which is the underlying error that caused the failure.
//
// Returns *CaptchaError which wraps the error with captcha context.
func NewCaptchaError(operation, provider string, err error) *CaptchaError {
	return &CaptchaError{
		Operation: operation,
		Provider:  provider,
		Err:       err,
	}
}

// Error implements the error interface for CaptchaError.
//
// Returns string which describes the failed captcha operation, including the
// provider name.
func (e *CaptchaError) Error() string {
	return fmt.Sprintf("captcha %s failed with provider %s: %v", e.Operation, e.Provider, e.Err)
}

// Unwrap returns the underlying error.
//
// Returns error which is the wrapped error, or nil if none exists.
func (e *CaptchaError) Unwrap() error {
	return e.Err
}
