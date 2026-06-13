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

// JobVersion is one immutable row from the append-only job_versions log.
//
// The log is the source of truth: the ordered sequence of a job's versions is its full
// history, and the latest version is the current state that job_index projects.
type JobVersion struct {
	// ScheduledAt is when the job was next due at this point.
	ScheduledAt time.Time

	// ClaimedAt is when the lease was taken at this point. Zero when the job held no lease.
	ClaimedAt time.Time

	// DeletedAt is the recoverable soft-delete marker at this point. Zero when not deleted.
	DeletedAt time.Time

	// RecordedAt is when this version row was written.
	RecordedAt time.Time

	// JobID is the job this version belongs to.
	JobID string

	// Event is the transition that produced this version, such as "enqueued" or "claimed".
	Event string

	// Status is the job's status at this point.
	Status string

	// LastError is the most recent failure message at this point. Empty when there was none.
	LastError string

	// ClaimedByWorkerID is the worker holding the job at this point. Empty when unclaimed.
	ClaimedByWorkerID string

	// Result is the JSON-encoded output recorded at this point. Nil until the job completes.
	Result []byte

	// Sequence is the monotonic version sequence; higher is newer, ordering the history.
	Sequence int64

	// Priority is the job's priority at this point.
	Priority int64

	// Attempt is the attempt count at this point.
	Attempt int64
}
