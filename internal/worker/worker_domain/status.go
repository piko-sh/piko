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

package worker_domain

// Status is the representation of the current lifecycle position of a job.
type Status string

const (
	// StatusUnknown is the zero value; a job is never written in this state.
	StatusUnknown Status = "unknown"

	// StatusPending means the job is waiting to be claimed.
	StatusPending Status = "pending"

	// StatusScheduled means the job is deferred until its scheduled time.
	StatusScheduled Status = "scheduled"

	// StatusRunning means a worker has claimed the job and is executing it.
	StatusRunning Status = "running"

	// StatusCompleted means the job finished successfully.
	StatusCompleted Status = "completed"

	// StatusFailed means the job exhausted its attempts or failed fatally.
	StatusFailed Status = "failed"

	// StatusTimeout means the job exceeded its per-attempt budget.
	StatusTimeout Status = "timeout"

	// StatusCancelled means the job was cancelled before it completed.
	StatusCancelled Status = "cancelled"

	// StatusRetryable means the job failed and is eligible for another attempt.
	StatusRetryable Status = "retryable"

	// StatusDiscarded means the job was dropped without running to completion.
	StatusDiscarded Status = "discarded"
)

// IsTerminal reports whether a state is in a state which never leaves.
//
// Returns bool which is true for completed, failed, timeout, cancelled and discarded.
func (s Status) IsTerminal() bool {
	switch s {
	case "completed", "failed", "timeout", "cancelled", "discarded":
		return true
	default:
		return false
	}
}

// IsActive reports whether a state is in a state which implies change.
//
// Returns bool which is true for pending, scheduled, running and retryable.
func (s Status) IsActive() bool {
	switch s {
	case "pending", "scheduled", "running", "retryable":
		return true
	default:
		return false
	}
}

// IsCompleted reports whether a state has been successfully completed.
//
// Returns bool which is true only for the completed state.
func (s Status) IsCompleted() bool {
	return s == "completed"
}
