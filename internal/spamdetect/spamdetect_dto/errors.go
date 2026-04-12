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

package spamdetect_dto

import (
	"errors"
	"fmt"
)

var (
	// ErrSpamDetectDisabled indicates spam detection is not configured.
	ErrSpamDetectDisabled = errors.New("spam detection: not configured - use piko.WithSpamDetector()")

	// ErrSpamDetected indicates the submission was classified as spam. Returned by callers
	// that wrap Analyse and reject above-threshold submissions via a single sentinel rather
	// than inspecting AnalysisResult.IsSpam.
	ErrSpamDetected = errors.New("spam detection: submission detected as spam")

	// ErrAllDetectorsFailed indicates every matching detector returned an error during
	// analysis.
	ErrAllDetectorsFailed = errors.New("spam detection: all matching detectors failed")

	// ErrNoMatchingDetectors indicates no registered detectors handle the signals declared
	// in the schema.
	ErrNoMatchingDetectors = errors.New("spam detection: no registered detectors match the schema signals")

	// ErrDetectorUnavailable indicates a named detector could not be resolved from the
	// registry. Returned when an expected detector is missing or temporarily unavailable.
	ErrDetectorUnavailable = errors.New("spam detection: detector unavailable")

	// ErrSubmissionNil indicates Analyse was called with a nil submission.
	ErrSubmissionNil = errors.New("spam detection: submission is nil")

	// ErrSchemaNil indicates Analyse was called with a nil schema.
	ErrSchemaNil = errors.New("spam detection: schema is nil")

	// ErrDetectorNameEmpty indicates RegisterDetector was called with an empty name.
	ErrDetectorNameEmpty = errors.New("spam detection: detector name is empty")

	// ErrDetectorNil indicates RegisterDetector was called with a nil detector.
	ErrDetectorNil = errors.New("spam detection: detector is nil")

	// ErrTooManyDetectors indicates the detector limit was reached.
	ErrTooManyDetectors = errors.New("spam detection: maximum detector count reached")

	// ErrSchemaTooManyFields indicates a schema declared more fields than the maximum
	// allowed.
	ErrSchemaTooManyFields = errors.New("spam detection: schema exceeds maximum field count")

	// ErrSchemaDuplicateField indicates a schema declared the same field key more than once.
	ErrSchemaDuplicateField = errors.New("spam detection: schema declares a duplicate field key")

	// ErrSchemaInvalidThreshold indicates a schema threshold is outside the valid range of
	// [0.0, 1.0].
	ErrSchemaInvalidThreshold = errors.New("spam detection: schema threshold must be between 0 and 1")

	// ErrSchemaCapExceeded indicates a builder cap (languages, detector weights, detector
	// options, metadata, captured headers) was reached during schema construction.
	ErrSchemaCapExceeded = errors.New("spam detection: schema builder cap exceeded")

	// ErrBlocklistTooLarge indicates the blocklist pattern count exceeds the maximum
	// allowed.
	ErrBlocklistTooLarge = errors.New("spam detection: blocklist pattern count exceeds maximum")

	// ErrBlocklistPatternInvalid indicates a blocklist pattern failed to compile as a
	// regular expression.
	ErrBlocklistPatternInvalid = errors.New("spam detection: blocklist pattern is invalid")

	// ErrBlocklistPatternTooLong indicates a blocklist pattern exceeded the per-pattern
	// length cap.
	ErrBlocklistPatternTooLong = errors.New("spam detection: blocklist pattern exceeds maximum length")

	// ErrUnexpectedDetectorResponse indicates a detector returned a response type that the
	// service could not interpret.
	ErrUnexpectedDetectorResponse = errors.New("spam detection: detector returned unexpected response type")

	// ErrSubmissionIDGeneration indicates the submission identifier generator failed to
	// produce a random identifier.
	ErrSubmissionIDGeneration = errors.New("spam detection: failed to generate submission identifier")
)

// SpamDetectError wraps a spam detection error with additional context about which
// operation and detector produced the failure. Callers can use errors.As to extract
// operation and detector attribution.
type SpamDetectError struct {
	// Err is the underlying error that caused the failure.
	Err error

	// Operation is the operation that failed, such as "analyse".
	Operation string

	// Detector is the name of the detector that encountered the error.
	Detector string
}

// NewSpamDetectError creates a new SpamDetectError with the given details.
//
// Takes operation (string) which identifies the failed operation.
// Takes detector (string) which identifies the detector that failed.
// Takes err (error) which is the underlying error.
//
// Returns *SpamDetectError which wraps the error with context.
func NewSpamDetectError(operation, detector string, err error) *SpamDetectError {
	return &SpamDetectError{
		Operation: operation,
		Detector:  detector,
		Err:       err,
	}
}

// Error implements the error interface.
//
// Returns string which describes the error.
func (e *SpamDetectError) Error() string {
	return fmt.Sprintf("spam detection %s failed with detector %s: %v", e.Operation, e.Detector, e.Err)
}

// Unwrap returns the underlying error.
//
// Returns error which is the wrapped error.
func (e *SpamDetectError) Unwrap() error {
	return e.Err
}
