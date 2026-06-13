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
	"context"
	"fmt"

	"piko.sh/piko/internal/worker/worker_dto"
)

// Handle is a caller-side reference to an enqueued job used to await its outcome.
type Handle struct {
	// waiter resolves the job's terminal state; nil once the owning service is closed.
	waiter JobStateWaiter

	// ID is the job row's id.
	ID string
}

// NewHandle builds a Handle bound to a job id and the waiter that resolves its outcome.
//
// Takes id (string) which is the job row id this handle refers to.
// Takes waiter (JobStateWaiter) which resolves the job's terminal state.
//
// Returns *Handle which is the ready handle.
func NewHandle(id string, waiter JobStateWaiter) *Handle {
	return &Handle{
		ID:     id,
		waiter: waiter,
	}
}

// Wait blocks until the job reaches a terminal state.
//
// Returns worker_dto.JobState which is the final state of the job.
// Returns error when the owning service is closed or the wait is interrupted.
func (h *Handle) Wait(ctx context.Context) (worker_dto.JobState, error) {
	if h.waiter == nil {
		return worker_dto.JobState{}, fmt.Errorf("waiting on job %q: %w", h.ID, ErrServiceClosed)
	}

	return h.waiter.WaitForTerminal(ctx, h.ID)
}
