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

// RawEntry represents a single WAL entry with untyped key and value bytes.
// This is used by inspection tools that need to read WAL files without
// knowing the concrete key/value types at compile time.
type RawEntry struct {
	// Key is the raw key bytes from the WAL entry.
	Key []byte

	// Value is the raw value bytes from the WAL entry.
	// Only meaningful for OpSet; empty for OpDelete and OpClear.
	Value []byte

	// Tags are the cache tags associated with this entry.
	// Only meaningful for OpSet.
	Tags []string

	// ExpiresAt is the Unix nanosecond timestamp when this entry expires.
	// Zero means no expiration.
	ExpiresAt int64

	// Timestamp is the Unix nanosecond timestamp when this operation occurred.
	Timestamp int64

	// SizeBytes is the total on-disk size of this entry including the length
	// prefix, CRC, and payload.
	SizeBytes int

	// Operation is the type of operation (Set, Delete, Clear).
	Operation Operation

	// CRCValid indicates whether the stored CRC32 checksum matched the
	// computed checksum for this entry.
	CRCValid bool
}
