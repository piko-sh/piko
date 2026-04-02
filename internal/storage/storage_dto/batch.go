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

package storage_dto

import (
	"fmt"
	"time"
)

// BatchResult holds the result of a batch operation, with details for each
// item processed.
type BatchResult struct {
	// SuccessfulKeys contains the keys that were processed without errors.
	SuccessfulKeys []string

	// FailedKeys contains each key that failed along with its specific error message.
	FailedKeys []BatchFailure

	// TotalRequested is the total number of items in the batch.
	TotalRequested int

	// TotalSuccessful is the number of operations that finished without error.
	TotalSuccessful int

	// TotalFailed is the number of operations that did not complete successfully.
	TotalFailed int

	// ProcessingTime is the total time taken to complete the batch operation.
	ProcessingTime time.Duration
}

// BatchFailure holds details about a single failed operation in a batch.
type BatchFailure struct {
	// Key is the object key that failed.
	Key string

	// Error is the message describing why the operation failed.
	Error string

	// ErrorCode is the provider-specific error code; empty if not available.
	ErrorCode string

	// Retryable indicates whether this failure can be retried.
	Retryable bool
}

// HasErrors returns true if any operations failed.
//
// Returns bool which is true when TotalFailed is greater than zero.
func (r *BatchResult) HasErrors() bool {
	return r.TotalFailed > 0
}

// IsPartialSuccess returns true if some operations succeeded and some failed.
//
// Returns bool which is true when both successful and failed counts are
// greater than zero.
func (r *BatchResult) IsPartialSuccess() bool {
	return r.TotalSuccessful > 0 && r.TotalFailed > 0
}

// formatSummary returns a short text summary of the batch result.
//
// Returns string which shows the success count, failure count, and how long
// processing took.
func (r *BatchResult) formatSummary() string {
	if !r.HasErrors() {
		return fmt.Sprintf("All %d operations succeeded in %v", r.TotalRequested, r.ProcessingTime)
	}
	if r.IsPartialSuccess() {
		return fmt.Sprintf("Partial success: %d/%d succeeded, %d failed in %v",
			r.TotalSuccessful, r.TotalRequested, r.TotalFailed, r.ProcessingTime)
	}
	return fmt.Sprintf("All %d operations failed in %v", r.TotalRequested, r.ProcessingTime)
}
