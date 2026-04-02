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

package cache_domain

import "time"

const (
	// DefaultTTL is the default time-to-live for cache entries.
	DefaultTTL = 1 * time.Hour

	// DefaultOperationTimeout is the default timeout for standard cache
	// operations such as Get and Set.
	DefaultOperationTimeout = 2 * time.Second

	// DefaultAtomicOperationTimeout is the default timeout for atomic
	// operations such as Compute.
	DefaultAtomicOperationTimeout = 5 * time.Second

	// DefaultBulkOperationTimeout is the default timeout for bulk operations
	// such as BulkGet and BulkSet.
	DefaultBulkOperationTimeout = 10 * time.Second

	// DefaultFlushTimeout is the default timeout for flush operations.
	DefaultFlushTimeout = 30 * time.Second

	// DefaultMaxComputeRetries is the default number of retry attempts for
	// optimistic-lock compute operations.
	DefaultMaxComputeRetries = 10

	// DefaultConnectionTimeout is the default timeout for initial connection
	// checks during provider creation.
	DefaultConnectionTimeout = 5 * time.Second

	// DefaultSearchTimeout is the default timeout for search operations.
	DefaultSearchTimeout = 5 * time.Second

	// DefaultIndexPrefix is the default prefix for search index names.
	DefaultIndexPrefix = "index:"
)

// ProviderDefaultsParams groups pointer fields that ApplyProviderDefaults
// fills with sensible defaults when their current value is zero. Nil pointers
// are safely skipped.
type ProviderDefaultsParams struct {
	// DefaultTTL points to the TTL field; set to DefaultTTL when zero.
	DefaultTTL *time.Duration

	// OperationTimeout points to the operation timeout field.
	OperationTimeout *time.Duration

	// AtomicOperationTimeout points to the atomic operation timeout field.
	AtomicOperationTimeout *time.Duration

	// BulkOperationTimeout points to the bulk operation timeout field.
	BulkOperationTimeout *time.Duration

	// FlushTimeout points to the flush timeout field.
	FlushTimeout *time.Duration

	// MaxComputeRetries points to the retry count field.
	MaxComputeRetries *int

	// SearchTimeout points to the search timeout field.
	SearchTimeout *time.Duration

	// IndexPrefix points to the index prefix field; set to DefaultIndexPrefix
	// when empty.
	IndexPrefix *string
}

// ApplyProviderDefaults fills zero-valued provider configuration fields with
// sensible defaults. Nil pointers are safely skipped.
//
// Takes p (ProviderDefaultsParams) which groups all configurable pointer
// fields.
func ApplyProviderDefaults(p ProviderDefaultsParams) {
	setDurationDefault(p.DefaultTTL, DefaultTTL)
	setDurationDefault(p.OperationTimeout, DefaultOperationTimeout)
	setDurationDefault(p.AtomicOperationTimeout, DefaultAtomicOperationTimeout)
	setDurationDefault(p.BulkOperationTimeout, DefaultBulkOperationTimeout)
	setDurationDefault(p.FlushTimeout, DefaultFlushTimeout)
	setDurationDefault(p.SearchTimeout, DefaultSearchTimeout)
	if p.MaxComputeRetries != nil && *p.MaxComputeRetries == 0 {
		*p.MaxComputeRetries = DefaultMaxComputeRetries
	}
	if p.IndexPrefix != nil && *p.IndexPrefix == "" {
		*p.IndexPrefix = DefaultIndexPrefix
	}
}

// setDurationDefault sets the value pointed to by ptr to defaultVal when ptr is
// non-nil and the current value is zero.
//
// Takes ptr (*time.Duration) which points to the duration field to fill.
// Takes defaultVal (time.Duration) which is the default value to apply.
func setDurationDefault(ptr *time.Duration, defaultVal time.Duration) {
	if ptr != nil && *ptr == 0 {
		*ptr = defaultVal
	}
}
