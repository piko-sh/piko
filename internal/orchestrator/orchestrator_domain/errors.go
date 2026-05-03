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

package orchestrator_domain

import (
	"errors"
	"fmt"
)

var (
	// ErrExecutorNotFound is returned when a task references an executor that
	// has not been registered.
	ErrExecutorNotFound = errors.New("orchestrator: executor not found")

	// ErrTaskFailedMaxRetries is returned when a task has exhausted all retry
	// attempts.
	ErrTaskFailedMaxRetries = errors.New("orchestrator: task failed after max retries")

	// ErrServiceClosed is returned when an operation is tried on a stopped
	// service.
	ErrServiceClosed = errors.New("orchestrator: service is closed")

	// ErrDuplicateTask is returned when attempting to create a task that
	// duplicates an existing active task with the same deduplication key. Active
	// tasks are those in SCHEDULED, PENDING, PROCESSING, or RETRYING status.
	ErrDuplicateTask = errors.New("orchestrator: duplicate active task exists")

	// ErrFatal signals that a task failed with a non-retryable error. When the
	// orchestrator detects this sentinel in an error chain, it skips retry
	// backoff and marks the task as failed immediately.
	ErrFatal = errors.New("orchestrator: fatal error")

	// ErrOrchestratorShuttingDown is returned by Dispatch and Schedule when the
	// orchestrator service is shutting down and the task insertion channel has
	// been closed. Callers should treat this as a terminal condition rather
	// than a retryable backpressure signal.
	ErrOrchestratorShuttingDown = errors.New("orchestrator: orchestrator is shutting down")
)

// NewFatalError wraps an error to mark it as a fatal orchestrator failure that
// should not be retried.
//
// Takes err (error) which is the underlying error to mark as fatal.
//
// Returns error which wraps both the original error and the ErrFatal sentinel.
func NewFatalError(err error) error {
	return fmt.Errorf("%w: %w", err, ErrFatal)
}

// IsFatalError reports whether err or any error in its chain is a fatal
// orchestrator error that should not be retried.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true when the error chain contains ErrFatal.
func IsFatalError(err error) bool {
	return errors.Is(err, ErrFatal)
}
