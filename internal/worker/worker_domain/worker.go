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
)

// Worker is the body of a fire-and-forget job. Implementations MUST be safe for
// concurrent use and idempotent, since execution is at-least-once.
type Worker[T any] interface {
	// Work runs one job and reports the outcome through its returned error.
	Work(ctx context.Context, job Job[T]) error
}

// WorkerFunc adapts an ordinary function to Worker[T] (the http.HandlerFunc idiom).
type WorkerFunc[T any] func(ctx context.Context, job Job[T]) error

// Work calls the underlying function, satisfying Worker[T].
//
// Takes job (Job[T]) which is the typed envelope for this execution.
//
// Returns error which is whatever the underlying function reports.
func (f WorkerFunc[T]) Work(ctx context.Context, job Job[T]) error {
	return f(ctx, job)
}
