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
	"time"
)

// DeadLetterPort defines the interface for dead letter queue adapters.
// It uses Go generics to support any entry type.
type DeadLetterPort[T any] interface {
	// Add adds a failed item to the dead letter queue.
	//
	// Takes entry (T) which contains the failed item data.
	//
	// Returns error when the entry cannot be added to the queue.
	Add(ctx context.Context, entry T) error

	// Get retrieves failed items from the dead letter queue.
	//
	// Takes limit (int) which specifies the maximum number of entries to return.
	//
	// Returns []T which contains the failed item entries.
	// Returns error when retrieval fails.
	Get(ctx context.Context, limit int) ([]T, error)

	// Remove removes entries from the dead letter queue.
	//
	// Takes entries ([]T) which specifies the entries to remove.
	//
	// Returns error when the removal operation fails.
	Remove(ctx context.Context, entries []T) error

	// Count returns the number of entries in the dead letter queue.
	//
	// Returns int which is the count of entries in the queue.
	// Returns error when the count operation fails.
	Count(ctx context.Context) (int, error)

	// Clear removes all entries from the dead letter queue.
	//
	// Returns error when the queue cannot be cleared.
	Clear(ctx context.Context) error

	// GetOlderThan retrieves entries older than the specified duration.
	//
	// Takes duration (time.Duration) which specifies the age threshold.
	//
	// Returns []T which contains the matching entries.
	// Returns error when the retrieval fails.
	GetOlderThan(ctx context.Context, duration time.Duration) ([]T, error)
}

// Entry defines the methods that dead letter entry types must provide. The
// generic dead letter queue uses this interface to work with any entry type.
type Entry interface {
	// GetID returns the unique identifier for this entry.
	GetID() string

	// SetID sets the unique identifier for this entry.
	//
	// Takes id (string) which is the identifier to assign.
	SetID(id string)

	// GetTimestamp returns when this entry was added to the dead letter queue.
	GetTimestamp() time.Time

	// SetTimestamp sets the time when this entry was added to the dead letter queue.
	//
	// Takes t (time.Time) which is the timestamp to record.
	SetTimestamp(t time.Time)
}
