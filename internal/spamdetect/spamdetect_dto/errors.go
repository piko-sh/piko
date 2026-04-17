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

	// ErrSpamDetected indicates the submission was classified as spam.
	ErrSpamDetected = errors.New("submission detected as spam")

	// ErrAllDetectorsFailed indicates every matching detector returned an
	// error during analysis.
	ErrAllDetectorsFailed = errors.New("all spam detection detectors failed")

	// ErrNoMatchingDetectors indicates no registered detectors handle the
	// signals declared in the schema.
	ErrNoMatchingDetectors = errors.New("no registered detectors match the schema signals")

	// ErrDetectorUnavailable indicates no detectors are registered.
	ErrDetectorUnavailable = errors.New("spam detection detector unavailable")
)

// SpamDetectError wraps a spam detection error with additional context
// about which operation and detector produced the failure.
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
