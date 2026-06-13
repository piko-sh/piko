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

// EnqueueSpec dictates the specification of the work to be processed.
type EnqueueSpec struct {
	// ID is the pre-allocated job id.
	ID string

	// ScheduledAt is when the job becomes eligible to be run.
	ScheduledAt time.Time

	// Kind is the identity of the job being run.
	Kind string

	// CorrelationID is the caller-supplied trace token carried onto the job.
	CorrelationID string

	// UniqueKey is the idempotency token used to deduplicate the enqueue; empty means no
	// deduplication.
	UniqueKey string

	// Queue is the target named queue.
	Queue string

	// Payload is the JSON-encoded args.
	Payload []byte

	// Priority is the claim-ordering weight.
	Priority int64

	// MaxAttempts is the retry cap.
	MaxAttempts int64

	// TimeoutSeconds is the per-attempt budget.
	TimeoutSeconds int64
}

// EnqueueRequest is the caller-facing enqueue intent the facade builds from a worker's
// typed args and options.
type EnqueueRequest struct {
	// Kind is the identity of the job being run.
	Kind string

	// Queue is the target named queue; empty resolves to the default queue.
	Queue string

	// Payload is the JSON-encoded args.
	Payload []byte

	// MaxAttempts caps the total number of runs, retries included; below 1 resolves to the
	// default.
	MaxAttempts int

	// TimeoutSeconds is the per-attempt budget in whole seconds; 0 uses the pool default.
	TimeoutSeconds int

	// UniqueKey is the idempotency token used to deduplicate the enqueue; empty means no
	// deduplication.
	UniqueKey string

	// CorrelationID is the caller-supplied trace token carried onto the job.
	CorrelationID string

	// RunAt is the absolute first-run time; when set it takes priority over Delay.
	RunAt time.Time

	// Delay is a relative deferral of the first run, applied only when RunAt is unset.
	Delay time.Duration

	// Priority is the claim-ordering weight.
	Priority int64
}
