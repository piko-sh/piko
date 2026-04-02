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

package wal_domain

// Operation represents the type of WAL entry.
type Operation uint8

const (
	// OpSet represents a set or update operation.
	OpSet Operation = iota + 1

	// OpDelete represents a delete operation for a single key.
	OpDelete

	// OpClear represents a clear-all operation.
	OpClear
)

// String returns the string representation of the operation.
//
// Returns string which is the operation name (SET, DELETE, CLEAR, or UNKNOWN).
func (o Operation) String() string {
	switch o {
	case OpSet:
		return "SET"
	case OpDelete:
		return "DELETE"
	case OpClear:
		return "CLEAR"
	default:
		return "UNKNOWN"
	}
}

// IsValid returns true if the operation is a known valid operation.
//
// Returns bool which is true when the operation is within the valid range.
func (o Operation) IsValid() bool {
	return o >= OpSet && o <= OpClear
}

// Entry represents a single WAL entry for recovery.
// Generic over key (K) and value (V) types.
type Entry[K comparable, V any] struct {
	// Key is the cache key for this entry.
	// For OpClear, this is the zero value.
	Key K

	// Value is the cache value for this entry.
	// Only meaningful for OpSet; zero value for OpDelete and OpClear.
	Value V

	// Tags are the cache tags associated with this entry.
	// Only meaningful for OpSet.
	Tags []string

	// ExpiresAt is the Unix nanosecond timestamp when this entry expires.
	// Zero means no expiration.
	ExpiresAt int64

	// Timestamp is the Unix nanosecond timestamp when this operation occurred.
	Timestamp int64

	// Operation is the type of operation (Set, Delete, Clear).
	Operation Operation
}

// IsExpired returns true if the entry has expired relative to the given
// Unix nanosecond timestamp.
//
// Takes nowNano (int64) which is the current time as a Unix nanosecond
// timestamp.
//
// Returns bool which is true when the entry has an expiration time set and
// that time has passed.
func (e Entry[K, V]) IsExpired(nowNano int64) bool {
	return e.ExpiresAt > 0 && e.ExpiresAt < nowNano
}

// HasTags returns true if the entry has any associated tags.
//
// Returns bool which is true when the entry has one or more tags.
func (e Entry[K, V]) HasTags() bool {
	return len(e.Tags) > 0
}
