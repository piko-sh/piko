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

package collection_dto

import "time"

const (
	// defaultRevalidationSeconds is the default time in seconds between cache
	// revalidations.
	defaultRevalidationSeconds = 60

	// ETagSourceModtimeHash uses file modification times to compute the ETag.
	// This is fast and works well for local file-based providers like markdown.
	ETagSourceModtimeHash = "modtime-hash"

	// ETagSourceContentHash computes the ETag from the file content. This is more
	// accurate but slower because it must read all files.
	ETagSourceContentHash = "content-hash"

	// ETagSourceProviderETag uses the ETag from the provider.
	// Suitable for HTTP-based providers that return ETag headers.
	ETagSourceProviderETag = "provider-etag"
)

// HybridConfig contains per-collection configuration for hybrid mode (ISR).
//
// Hybrid mode combines static generation with runtime revalidation:
//   - Build-time: Generate static snapshot with computed ETag
//   - Runtime: Serve snapshot immediately, validate ETag in background
//   - Revalidation: Only refetch when ETag changes
//   - Fallback: Serve stale content if revalidation fails
//
// This configuration controls the revalidation behaviour.
type HybridConfig struct {
	// ETagSource specifies how to compute the ETag for staleness detection.
	//
	// Options:
	//   - "modtime-hash": Hash of file modification times (default for markdown).
	//   - "content-hash": Hash of file contents (more accurate but slower).
	//   - "provider-etag": Use ETag from provider (e.g. HTTP ETag header from CMS).
	//
	// Default: "modtime-hash".
	ETagSource string

	// RevalidationTTL is how long to wait before checking if cached data is stale.
	//
	// After this time passes, the next request starts a background ETag check.
	// The request still gets the cached content straight away while the check
	// runs.
	//
	// Default: 60 seconds.
	// Zero value: check every request (not suggested for production).
	RevalidationTTL time.Duration

	// MaxStaleAge is the maximum age of content before forcing a refresh.
	//
	// If content is older than this, the system will attempt a synchronous
	// refresh instead of serving stale content (unless StaleIfError is true
	// and the refresh fails).
	//
	// Default: 0 (no maximum - content can be arbitrarily stale)
	MaxStaleAge time.Duration

	// StaleIfError controls whether to serve stale content when revalidation fails.
	//
	// When true (default), if the ETag check or content fetch fails, the system
	// continues serving the last known good content instead of returning an error.
	//
	// Default: true
	StaleIfError bool
}

// HybridRevalidationResult is returned by background revalidation operations.
type HybridRevalidationResult struct {
	// RevalidatedAt is when this revalidation finished.
	RevalidatedAt time.Time

	// Error holds any failure that occurred during revalidation.
	// When StaleIfError is true, stale content is served despite this error.
	Error error

	// NewETag is the ETag from the latest check; unchanged if still valid.
	NewETag string

	// NewItems holds the fresh content when ETagChanged is true.
	// Nil if the content has not changed or if an error occurred.
	NewItems []ContentItem

	// ETagChanged indicates whether the content has changed since the last check.
	ETagChanged bool
}

// IsValid returns true if the revalidation completed successfully.
//
// Returns bool which indicates whether no error occurred during revalidation.
func (r *HybridRevalidationResult) IsValid() bool {
	return r.Error == nil
}

// NeedsUpdate returns true if the cache should be updated with new content.
//
// Returns bool which is true when there is no error, the ETag has changed,
// and new items are available.
func (r *HybridRevalidationResult) NeedsUpdate() bool {
	return r.Error == nil && r.ETagChanged && len(r.NewItems) > 0
}

// DefaultHybridConfig returns the default settings for hybrid caching.
//
// Returns HybridConfig which provides useful defaults for hybrid caching.
func DefaultHybridConfig() HybridConfig {
	return HybridConfig{
		RevalidationTTL: defaultRevalidationSeconds * time.Second,
		StaleIfError:    true,
		MaxStaleAge:     0,
		ETagSource:      ETagSourceModtimeHash,
	}
}
