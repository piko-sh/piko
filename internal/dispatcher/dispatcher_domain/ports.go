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

package dispatcher_domain

import (
	"context"
	"time"
)

// DispatcherInspector provides read-only access to dispatcher state and dead
// letter queues. It abstracts over the email and notification dispatchers to
// provide a unified monitoring interface.
type DispatcherInspector interface {
	// GetDispatcherSummaries returns statistics for all configured dispatchers.
	//
	// Returns []DispatcherSummary which contains one entry per dispatcher type.
	// Returns error when the statistics cannot be retrieved.
	GetDispatcherSummaries(ctx context.Context) ([]DispatcherSummary, error)

	// GetDLQEntries returns dead letter queue entries for a specific dispatcher.
	//
	// Takes dispatcherType (string) which identifies the dispatcher, either
	// "email" or "notification".
	// Takes limit (int) which caps the number of entries returned.
	//
	// Returns []DLQEntry which contains the dead letter entries.
	// Returns error when the entries cannot be retrieved or the type is unknown.
	GetDLQEntries(ctx context.Context, dispatcherType string, limit int) ([]DLQEntry, error)

	// GetDLQCount returns the number of entries in a dispatcher's dead letter queue.
	//
	// Takes dispatcherType (string) which identifies the dispatcher.
	//
	// Returns int which is the count of dead letter entries.
	// Returns error when the count cannot be retrieved or the type is unknown.
	GetDLQCount(ctx context.Context, dispatcherType string) (int, error)

	// ClearDLQ removes all entries from a dispatcher's dead letter queue.
	//
	// Takes dispatcherType (string) which identifies the dispatcher.
	//
	// Returns error when the queue cannot be cleared or the type is unknown.
	ClearDLQ(ctx context.Context, dispatcherType string) error
}

// DispatcherSummary holds statistics for a single dispatcher.
type DispatcherSummary struct {
	// Type identifies the dispatcher ("email" or "notification").
	Type string

	// QueuedItems is the number of items waiting in the main queue.
	QueuedItems int

	// RetryQueueSize is the number of items in the retry queue.
	RetryQueueSize int

	// DeadLetterCount is the number of items in the dead letter queue.
	DeadLetterCount int

	// TotalProcessed is the total number of items processed since start.
	TotalProcessed int64

	// TotalSuccessful is the total number of items processed without error.
	TotalSuccessful int64

	// TotalFailed is the total number of items that failed permanently.
	TotalFailed int64

	// TotalRetries is the total number of retry attempts made.
	TotalRetries int64

	// Uptime is how long the dispatcher has been running.
	Uptime time.Duration
}

// DLQEntry holds details about a single dead letter queue entry.
type DLQEntry struct {
	// FirstAttempt is when the first delivery attempt was made.
	FirstAttempt time.Time

	// LastAttempt is when the last retry was attempted.
	LastAttempt time.Time

	// AddedAt is when the entry was added to the dead letter queue.
	AddedAt time.Time

	// ID is the unique identifier for this entry.
	ID string

	// Type identifies the dispatcher ("email" or "notification").
	Type string

	// OriginalError is the error message from the failed delivery.
	OriginalError string

	// TotalAttempts is the number of delivery attempts made.
	TotalAttempts int
}
