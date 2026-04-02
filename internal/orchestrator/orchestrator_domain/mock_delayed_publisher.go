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
	"sync/atomic"
)

// MockDelayedPublisher is a test double for DelayedPublisher where nil
// function fields return zero values and call counts are tracked
// atomically.
type MockDelayedPublisher struct {
	// ScheduleFunc is the function called by Schedule.
	ScheduleFunc func(ctx context.Context, task *Task) error

	// StartFunc is the function called by Start.
	StartFunc func(ctx context.Context)

	// StopFunc is the function called by Stop.
	StopFunc func()

	// PendingCountFunc is the function called by
	// PendingCount.
	PendingCountFunc func() int

	// ScheduleCallCount tracks how many times Schedule
	// was called.
	ScheduleCallCount int64

	// StartCallCount tracks how many times Start was
	// called.
	StartCallCount int64

	// StopCallCount tracks how many times Stop was
	// called.
	StopCallCount int64

	// PendingCountCallCount tracks how many times
	// PendingCount was called.
	PendingCountCallCount int64
}

// Schedule enqueues a task for delayed execution.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes task (*Task) which is the task to schedule.
//
// Returns error, or nil if ScheduleFunc is nil.
func (m *MockDelayedPublisher) Schedule(ctx context.Context, task *Task) error {
	atomic.AddInt64(&m.ScheduleCallCount, 1)
	if m.ScheduleFunc != nil {
		return m.ScheduleFunc(ctx, task)
	}
	return nil
}

// Start begins processing delayed tasks.
func (m *MockDelayedPublisher) Start(ctx context.Context) {
	atomic.AddInt64(&m.StartCallCount, 1)
	if m.StartFunc != nil {
		m.StartFunc(ctx)
	}
}

// Stop halts processing of delayed tasks.
func (m *MockDelayedPublisher) Stop() {
	atomic.AddInt64(&m.StopCallCount, 1)
	if m.StopFunc != nil {
		m.StopFunc()
	}
}

// PendingCount returns the number of tasks awaiting execution.
//
// Returns int, or 0 if PendingCountFunc is nil.
func (m *MockDelayedPublisher) PendingCount() int {
	atomic.AddInt64(&m.PendingCountCallCount, 1)
	if m.PendingCountFunc != nil {
		return m.PendingCountFunc()
	}
	return 0
}
