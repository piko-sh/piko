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
	"context"
	"sync"
	"time"
)

// continuationKey identifies the successor registration stored in an execution context.
type continuationKey struct{}

// settledContinuation holds one successor until the current task settles.
type settledContinuation struct {
	// next holds the registered successor until take consumes it.
	next *Task

	// mu guards next and closed.
	mu sync.Mutex

	// closed prevents registration after the continuation has been consumed.
	closed bool
}

// ScheduleSuccessor registers one recurring task to persist with successful settlement.
//
// Executors opt in by implementing SchedulesSuccessor() bool and returning true. Further
// registrations are ignored once a successor is set or the continuation is closed.
//
// Takes task (*Task) which becomes the scheduled successor if registration is still open.
// Takes executeAt (time.Time) which sets the successor's execution time in UTC.
//
// Returns bool which is true when the context supports successors, even if registration
// is ignored; false means callers should schedule normally.
//
// Safe for concurrent registration on the same continuation; its mutex guards
// registration and consumption. Callers must not access the task concurrently during
// registration.
func ScheduleSuccessor(ctx context.Context, task *Task, executeAt time.Time) bool {
	c, ok := ctx.Value(continuationKey{}).(*settledContinuation)
	if !ok {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed && c.next == nil {
		task.Status = StatusScheduled
		task.Background = true
		task.BuildTag = ""
		task.ExecuteAt = executeAt.UTC()
		task.ScheduledExecuteAt = task.ExecuteAt
		c.next = task
	}
	return true
}

// take closes registration and consumes the registered successor.
//
// Returns *Task which is the successor, or nil for an absent or already consumed
// continuation.
//
// Safe for concurrent use; holds the continuation mutex while consuming the task.
func (c *settledContinuation) take() *Task {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	next := c.next
	c.next = nil
	return next
}
