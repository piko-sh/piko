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

package deadletter_adapters

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"piko.sh/piko/internal/deadletter/deadletter_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// wrappedEntry wraps user data with dead letter queue metadata.
type wrappedEntry[T any] struct {
	// Timestamp is when the log entry was recorded.
	Timestamp time.Time `json:"timestamp"`

	// Data holds the wrapped entry value.
	Data T `json:"data"`

	// ID is the unique identifier for this entry.
	ID string `json:"id"`
}

// MemoryDeadLetterQueue is a generic in-memory dead letter queue.
type MemoryDeadLetterQueue[T any] struct {
	// entries stores items that failed processing, keyed by unique identifier.
	entries map[string]wrappedEntry[T]

	// mu guards the queue's internal state for safe concurrent access.
	mu sync.RWMutex
}

var _ deadletter_domain.DeadLetterPort[any] = (*MemoryDeadLetterQueue[any])(nil)

// Add stores an item in the dead letter queue.
//
// Takes entry (T) which is the item to store.
//
// Returns error when the operation fails (currently always returns nil).
//
// Safe for concurrent use; protects the queue with a mutex.
func (m *MemoryDeadLetterQueue[T]) Add(ctx context.Context, entry T) error {
	ctx, l := logger_domain.From(ctx, log)

	m.mu.Lock()
	defer m.mu.Unlock()

	wrapped := wrappedEntry[T]{
		ID:        uuid.NewString(),
		Timestamp: time.Now(),
		Data:      entry,
	}

	m.entries[wrapped.ID] = wrapped

	l.Warn("Item added to in-memory dead letter queue",
		logger_domain.String("entry_id", wrapped.ID),
		logger_domain.Int("queue_size", len(m.entries)))

	return nil
}

// Get retrieves items from the dead letter queue.
//
// Takes limit (int) which caps the number of entries returned; zero or
// negative returns all entries.
//
// Returns []T which contains up to limit entries from the queue.
// Returns error which is always nil for the in-memory implementation.
//
// Safe for concurrent use; protected by a read lock.
func (m *MemoryDeadLetterQueue[T]) Get(_ context.Context, limit int) ([]T, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.entries) {
		limit = len(m.entries)
	}

	result := make([]T, 0, limit)
	count := 0
	for _, wrapped := range m.entries {
		if count >= limit {
			break
		}
		result = append(result, wrapped.Data)
		count++
	}

	return result, nil
}

// Remove deletes entries from the dead letter queue.
//
// Takes entries ([]T) which specifies the entries to remove.
//
// Returns error when removal fails, though current implementation always
// returns nil.
//
// Safe for concurrent use; protected by a mutex.
//
// Note: This requires entries to match by value, which may not work for all
// types. For better removal, services should track entry IDs. Current
// implementation logs a warning and does not perform actual removal.
func (m *MemoryDeadLetterQueue[T]) Remove(ctx context.Context, entries []T) error {
	ctx, l := logger_domain.From(ctx, log)

	m.mu.Lock()
	defer m.mu.Unlock()

	l.Warn("Memory DLQ Remove not fully implemented for generic types",
		logger_domain.Int("requested_to_remove", len(entries)))

	return nil
}

// Count returns the number of entries in the dead letter queue.
//
// Returns int which is the current entry count.
// Returns error which is always nil.
//
// Safe for concurrent use; protects access with a read lock.
func (m *MemoryDeadLetterQueue[T]) Count(_ context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.entries), nil
}

// Clear removes all entries from the dead letter queue.
//
// Returns error when the operation fails.
//
// Safe for concurrent use; holds the lock while clearing entries.
func (m *MemoryDeadLetterQueue[T]) Clear(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	m.mu.Lock()
	defer m.mu.Unlock()

	count := len(m.entries)
	m.entries = make(map[string]wrappedEntry[T])

	l.Internal("Cleared all entries from in-memory dead letter queue",
		logger_domain.Int("cleared_count", count))

	return nil
}

// GetOlderThan retrieves entries older than the specified duration.
//
// Takes duration (time.Duration) which sets the age threshold for
// entries to retrieve.
//
// Returns []T which contains entries older than the cutoff time.
// Returns error which is always nil for the in-memory implementation.
//
// Safe for concurrent use; protected by a read lock.
func (m *MemoryDeadLetterQueue[T]) GetOlderThan(_ context.Context, duration time.Duration) ([]T, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	var result []T

	for _, wrapped := range m.entries {
		if wrapped.Timestamp.Before(cutoff) {
			result = append(result, wrapped.Data)
		}
	}

	return result, nil
}

// NewMemoryDeadLetterQueue creates a new in-memory dead letter queue.
//
// Returns deadletter_domain.DeadLetterPort[T] which is ready to use.
func NewMemoryDeadLetterQueue[T any]() deadletter_domain.DeadLetterPort[T] {
	return &MemoryDeadLetterQueue[T]{
		entries: make(map[string]wrappedEntry[T]),
	}
}
