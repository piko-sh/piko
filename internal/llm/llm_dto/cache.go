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

package llm_dto

import (
	"time"

	"piko.sh/piko/wdk/maths"
)

// CacheConfig holds settings for caching LLM completion requests.
type CacheConfig struct {
	// Key is a custom cache key. If empty, a key is generated from the request.
	Key string

	// TTL is how long cached responses are kept before they expire.
	// If zero, the cache manager's default TTL is used.
	TTL time.Duration

	// Enabled controls whether caching is used for this request.
	Enabled bool

	// SkipWrite prevents writing to the cache (but still reads from it).
	SkipWrite bool

	// SkipRead prevents reading from the cache (but still writes to it).
	SkipRead bool

	// UseProviderCache enables provider-specific caching, such as Anthropic
	// prompt caching.
	UseProviderCache bool
}

// CacheEntry represents a cached LLM response.
type CacheEntry struct {
	// Response is the cached completion response.
	Response *CompletionResponse

	// CreatedAt is when this cache entry was created.
	CreatedAt time.Time

	// ExpiresAt is when this cache entry becomes invalid.
	ExpiresAt time.Time

	// RequestHash is a hash of the request that created this response.
	RequestHash string

	// Provider is the name of the provider that generated the response.
	Provider string

	// Model is the model that generated the response.
	Model string

	// HitCount tracks how many times this entry has been read from the cache.
	HitCount int64
}

// IsExpiredAt checks if the cache entry has expired relative to the given time,
// enabling deterministic testing with mock clocks.
//
// Takes now (time.Time) which is the current time to compare against.
//
// Returns bool which is true if the entry has expired.
func (e *CacheEntry) IsExpiredAt(now time.Time) bool {
	return now.After(e.ExpiresAt)
}

// CacheStats holds data about cache usage, including hits, misses, and savings.
type CacheStats struct {
	// EstimatedCostSaved is the estimated cost saved by cache hits.
	EstimatedCostSaved maths.Money

	// Hits is the number of successful cache lookups.
	Hits int64

	// Misses is the number of times a cache lookup did not find the item.
	Misses int64

	// Size is the current number of entries in the cache.
	Size int64
}

// HitRate calculates the cache hit rate as a fraction.
//
// Returns float64 which is the hit rate (0.0 to 1.0).
func (s *CacheStats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

// DefaultCacheConfig returns a CacheConfig with sensible default values.
// Caching is enabled with a one-hour TTL.
//
// Returns *CacheConfig with caching enabled.
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		Enabled: true,
		TTL:     time.Hour,
	}
}
