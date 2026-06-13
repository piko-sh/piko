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

// JobRecord is the type-erased carrier of one jobs row across the port boundary.
type JobRecord struct {
	// ClaimedAt is when the current lease was taken. Unset when unclaimed.
	ClaimedAt time.Time

	// ScheduledAt is when the job became eligible to run.
	ScheduledAt time.Time

	// EnqueueAt is the record creation time.
	EnqueueAt time.Time

	// ID is the id of the job.
	ID string

	// Status is the lifecycle position.
	Status string

	// Kind is the identity of the job being run.
	Kind string

	// Queue is the target named queue.
	Queue string

	// LastError is the most recent failure message.
	LastError string

	// ClaimedByWorkerID is he id of the worker node which is currently leasing the row.
	// Empty when unclaimed.
	ClaimedByWorkerID string

	// Payload is the raw JSON-encoded args, decoded by the registered handler.
	Payload []byte

	// Priority is the claim-ordering weight.
	Priority int64

	// Attempt is the number of attempts which have been done so far.
	Attempt int64

	// MaxAttempts is the retry cap.
	MaxAttempts int64

	// TimeoutSeconds is the per-attempt wall-clock budget.
	TimeoutSeconds int64
}

// Outcome is the resolved result of one finished attempt.
type Outcome struct {
	// NextScheduledAt is when a retryable job next becomes eligible to be run.
	NextScheduledAt time.Time

	// ErrorMessage is the failure message recorded. Empty on success.
	ErrorMessage string

	// Status is the terminal status.
	Status string

	// Result is the JSON-encoded recorded output.
	Result []byte
}
