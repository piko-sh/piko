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

package worker_dto

import (
	"time"
)

// JobState is the snapshot of a job's persisted state.
type JobState struct {
	// ScheduledAt is when the job became eligible to run.
	ScheduledAt time.Time

	// UpdatedAt is when the row was last mutated.
	UpdatedAt time.Time

	// CreatedAt is when the job row was first written.
	CreatedAt time.Time

	// ID is the job row id.
	ID string

	// Kind is the identity of the job being run.
	Kind string

	// Queue is the named queue the job was routed to.
	Queue string

	// LastError is the most recent failure message. Empty when the job has not failed.
	LastError string

	// Status is the lowercase lifecycle position.
	Status string

	// Priority is the claim-ordering weight.
	Priority int64

	// Attempt is the number of attempts consumed so far.
	Attempt int64

	// MaxAttempts is the configured retry cap.
	MaxAttempts int64
}

// IsTerminal reports whether a job is in a state which never leaves.
//
// Returns bool which is true for completed, failed, timeout, cancelled and discarded.
func (s JobState) IsTerminal() bool {
	switch s.Status {
	case "failed", "completed", "timeout", "cancelled", "discarded":
		return true
	default:
		return false
	}
}

// IsActive reports whether a job is in a state which implies change.
//
// Returns bool which is true for pending, scheduled, running and retryable.
func (s JobState) IsActive() bool {
	switch s.Status {
	case "pending", "running", "scheduled", "retryable":
		return true
	default:
		return false
	}
}

// IsCompleted reports whether a job has been successfully completed.
//
// Returns bool which is true only for the completed state.
func (s JobState) IsCompleted() bool {
	return s.Status == "completed"
}
