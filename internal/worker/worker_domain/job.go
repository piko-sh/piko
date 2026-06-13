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

import (
	"time"
)

// Kinded is the contract for a stable identity for a job, such as: "<namespace>:<name>".
type Kinded interface {
	// Kind reports the stable, hand-written identity persisted in jobs.kind.
	Kind() string
}

// Job is the per-execution envelope handed to Worker[T].Work. All fields are read-only
// from the worker's perspective; the framework owns the row.
type Job[T any] struct {
	// EnqueueAt is when the job was first enqueued.
	EnqueueAt time.Time

	// Args is the typed payload decoded from the job's stored arguments.
	Args T

	// ID is the job row id.
	ID string

	// Attempt is the 1-based current attempt number.
	Attempt int64

	// MaxAttempt is the retry cap before the row is sent to its terminal state.
	MaxAttempt int64
}

// IsFinalAttempt reports whether this is the last attempt the framework will make.
//
// Returns bool which is true when the current attempt has reached MaxAttempt.
func (j Job[T]) IsFinalAttempt() bool {
	return j.Attempt >= j.MaxAttempt
}
