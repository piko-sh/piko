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

package retry

import (
	"context"
	"errors"
	"net"
	"slices"
	"strings"
	"syscall"
)

var (
	// retryableSyscallErrors contains syscall error codes that indicate transient
	// network failures worth retrying.
	retryableSyscallErrors = []syscall.Errno{
		syscall.ECONNREFUSED,
		syscall.ECONNRESET,
		syscall.ETIMEDOUT,
		syscall.EHOSTUNREACH,
		syscall.ENETUNREACH,
		syscall.ECONNABORTED,
	}

	// defaultRetryablePatterns contains error message substrings that indicate
	// transient failures worth retrying.
	defaultRetryablePatterns = []string{
		"connection refused", "connection reset", "connection timeout", "timeout",
		"temporary failure", "too many requests", "rate limit", "throttle",
		"500", "502", "503", "504",
	}

	// defaultPermanentErrors contains errors that should never be retried.
	defaultPermanentErrors = []error{
		context.Canceled,
		context.DeadlineExceeded,
	}
)

// ErrorClassifier determines whether errors are retryable or permanent.
// It checks permanent errors, network timeouts, retryable syscall errors,
// and error message patterns.
//
// The zero value is not usable; create instances with [NewErrorClassifier].
type ErrorClassifier struct {
	// permanentErrors lists errors that should never be retried.
	permanentErrors []error

	// retryablePatterns lists case-insensitive substrings in error messages
	// that indicate transient failures.
	retryablePatterns []string
}

// ClassifierOption configures an [ErrorClassifier].
type ClassifierOption func(*ErrorClassifier)

// NewErrorClassifier creates an ErrorClassifier with the default permanent
// errors and retryable patterns, plus any domain-specific additions from
// the provided options.
//
// Takes opts (...ClassifierOption) which configure additional permanent
// errors or retryable patterns.
//
// Returns *ErrorClassifier which is ready for use.
func NewErrorClassifier(opts ...ClassifierOption) *ErrorClassifier {
	c := &ErrorClassifier{
		permanentErrors:   append([]error{}, defaultPermanentErrors...),
		retryablePatterns: append([]string{}, defaultRetryablePatterns...),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// IsRetryable reports whether an error is temporary and worth retrying.
// It returns false for nil errors and permanent errors, and true for network
// timeouts, retryable syscall errors, and errors whose message matches a
// retryable pattern.
//
// Takes err (error) which is the error to classify.
//
// Returns bool which is true if the error can be retried.
func (c *ErrorClassifier) IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if c.isPermanentError(err) {
		return false
	}
	if IsNetworkTimeout(err) || IsSyscallRetryable(err) || c.isRetryableByMessage(err.Error()) {
		return true
	}
	return false
}

// isPermanentError checks if an error matches any configured permanent error.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if the error matches any permanent error.
func (c *ErrorClassifier) isPermanentError(err error) bool {
	for _, pe := range c.permanentErrors {
		if errors.Is(err, pe) {
			return true
		}
	}
	return false
}

// isRetryableByMessage checks if an error message contains any retryable
// pattern.
//
// Takes errMessage (string) which is the error message to check.
//
// Returns bool which is true if the message matches a retryable pattern.
func (c *ErrorClassifier) isRetryableByMessage(errMessage string) bool {
	lowerErrMessage := strings.ToLower(errMessage)
	for _, pattern := range c.retryablePatterns {
		if strings.Contains(lowerErrMessage, pattern) {
			return true
		}
	}
	return false
}

// WithPermanentErrors adds domain-specific errors that should never be
// retried. These are appended to the default permanent errors
// (context.Canceled, context.DeadlineExceeded).
//
// Takes errs (...error) which are errors to treat as permanent.
//
// Returns ClassifierOption which configures the classifier with the
// permanent errors.
func WithPermanentErrors(errs ...error) ClassifierOption {
	return func(c *ErrorClassifier) {
		c.permanentErrors = append(c.permanentErrors, errs...)
	}
}

// WithRetryablePatterns adds domain-specific error message patterns that
// indicate retryable failures. These are appended to the default patterns.
//
// Takes patterns (...string) which are case-insensitive substrings to match
// against error messages.
//
// Returns ClassifierOption which adds the patterns to the classifier.
func WithRetryablePatterns(patterns ...string) ClassifierOption {
	return func(c *ErrorClassifier) {
		c.retryablePatterns = append(c.retryablePatterns, patterns...)
	}
}

// IsNetworkTimeout reports whether an error is a network timeout.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if the error is a network timeout.
func IsNetworkTimeout(err error) bool {
	if netErr, ok := errors.AsType[net.Error](err); ok {
		return netErr.Timeout()
	}
	return false
}

// IsSyscallRetryable reports whether an error wraps a retryable syscall
// error code (ECONNREFUSED, ECONNRESET, ETIMEDOUT, EHOSTUNREACH,
// ENETUNREACH, ECONNABORTED).
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if the error wraps a retryable syscall.
func IsSyscallRetryable(err error) bool {
	if errno, ok := errors.AsType[syscall.Errno](err); ok {
		return slices.Contains(retryableSyscallErrors, errno)
	}
	return false
}
