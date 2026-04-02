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

package deadletter_domain

import (
	"context"
	"sync/atomic"
	"time"
)

// MockDeadLetterPort is a test double for DeadLetterPort that uses overridable
// function fields. Nil fields return zero values, and call counts are tracked
// for assertions.
type MockDeadLetterPort[T any] struct {
	// AddFunc is the mock implementation for adding a dead letter entry.
	AddFunc func(ctx context.Context, entry T) error

	// GetFunc retrieves items up to the specified limit.
	GetFunc func(ctx context.Context, limit int) ([]T, error)

	// RemoveFunc is called to remove dead letter entries from storage.
	RemoveFunc func(ctx context.Context, entries []T) error

	// CountFunc is called when the Count method is invoked on the mock.
	CountFunc func(ctx context.Context) (int, error)

	// ClearFunc is the mock function for the Clear method.
	ClearFunc func(ctx context.Context) error

	// GetOlderThanFunc is the mock function for GetOlderThan.
	GetOlderThanFunc func(ctx context.Context, duration time.Duration) ([]T, error)

	// AddCallCount tracks the number of times Add was called. Use
	// atomic.LoadInt64 to read safely from concurrent goroutines.
	AddCallCount int64

	// GetCallCount tracks the number of times Get was called.
	GetCallCount int64

	// RemoveCallCount tracks the number of times Remove was called.
	RemoveCallCount int64

	// CountCallCount tracks the number of times Count was called.
	CountCallCount int64

	// ClearCallCount tracks the number of times Clear was called.
	ClearCallCount int64

	// GetOlderThanCallCount tracks the number of times GetOlderThan was called.
	GetOlderThanCallCount int64
}

// Add adds a failed item to the dead letter queue.
//
// Takes entry (T) which is the failed item to store.
//
// Returns error when AddFunc is set and returns an error. Returns nil if
// AddFunc is nil.
func (m *MockDeadLetterPort[T]) Add(ctx context.Context, entry T) error {
	atomic.AddInt64(&m.AddCallCount, 1)
	if m.AddFunc != nil {
		return m.AddFunc(ctx, entry)
	}
	return nil
}

// Get retrieves failed items from the dead letter queue.
//
// Takes limit (int) which specifies the maximum number of items to retrieve.
//
// Returns []T which contains the retrieved items, or nil if GetFunc is nil.
// Returns error when the underlying GetFunc returns an error.
func (m *MockDeadLetterPort[T]) Get(ctx context.Context, limit int) ([]T, error) {
	atomic.AddInt64(&m.GetCallCount, 1)
	if m.GetFunc != nil {
		return m.GetFunc(ctx, limit)
	}
	return nil, nil
}

// Remove removes entries from the dead letter queue.
//
// Takes entries ([]T) which contains the entries to remove.
//
// Returns error when RemoveFunc fails. Returns nil if RemoveFunc is nil.
func (m *MockDeadLetterPort[T]) Remove(ctx context.Context, entries []T) error {
	atomic.AddInt64(&m.RemoveCallCount, 1)
	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, entries)
	}
	return nil
}

// Count returns the number of entries in the dead letter queue.
//
// Returns int which is the count of entries, or zero if CountFunc is nil.
// Returns error when the count operation fails.
func (m *MockDeadLetterPort[T]) Count(ctx context.Context) (int, error) {
	atomic.AddInt64(&m.CountCallCount, 1)
	if m.CountFunc != nil {
		return m.CountFunc(ctx)
	}
	return 0, nil
}

// Clear removes all entries from the dead letter queue.
//
// Returns error when ClearFunc is set and returns an error. Returns nil if
// ClearFunc is nil.
func (m *MockDeadLetterPort[T]) Clear(ctx context.Context) error {
	atomic.AddInt64(&m.ClearCallCount, 1)
	if m.ClearFunc != nil {
		return m.ClearFunc(ctx)
	}
	return nil
}

// GetOlderThan retrieves entries older than the specified duration. If
// GetOlderThanFunc is nil, returns (nil, nil).
//
// Takes duration (time.Duration) which specifies the age threshold.
//
// Returns []T which contains entries older than the specified duration.
// Returns error when the retrieval fails.
func (m *MockDeadLetterPort[T]) GetOlderThan(ctx context.Context, duration time.Duration) ([]T, error) {
	atomic.AddInt64(&m.GetOlderThanCallCount, 1)
	if m.GetOlderThanFunc != nil {
		return m.GetOlderThanFunc(ctx, duration)
	}
	return nil, nil
}
