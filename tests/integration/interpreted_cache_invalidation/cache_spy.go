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

package cache_invalidation_test

import (
	"context"
	"sync"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
)

// CacheSpy wraps a cache and tracks all Get operations to detect hits/misses.
// This is used in tests to verify that the correct cache tier is being used.
type CacheSpy struct {
	// underlying is the wrapped cache that handles actual storage operations.
	underlying coordinator_domain.BuildResultCachePort

	// lastKey is the most recent key passed to Get; empty after reset.
	lastKey string

	// getCount tracks the total number of Get calls.
	getCount int

	// hitCount is the number of cache hits.
	hitCount int

	// missCount is the number of cache misses recorded.
	missCount int

	// mu guards access to the spy statistics fields.
	mu sync.Mutex
}

// NewCacheSpy creates a new cache spy that wraps an underlying cache.
//
// Takes underlying (BuildResultCachePort) which is the cache to wrap and spy
// on.
//
// Returns *CacheSpy which wraps the underlying cache to record interactions.
func NewCacheSpy(underlying coordinator_domain.BuildResultCachePort) *CacheSpy {
	return &CacheSpy{
		underlying: underlying,
	}
}

// Get retrieves a cache entry by key, delegating to the underlying cache.
//
// Takes ctx (context.Context) for cancellation.
// Takes key (string) which identifies the cache entry to retrieve.
//
// Returns *annotator_dto.ProjectAnnotationResult which is the cached result if
// found.
// Returns error when the key is not found or the underlying cache fails.
//
// Safe for concurrent use. Access to spy counters is protected by a mutex.
func (s *CacheSpy) Get(ctx context.Context, key string) (*annotator_dto.ProjectAnnotationResult, error) {
	s.mu.Lock()
	s.getCount++
	s.lastKey = key
	s.mu.Unlock()

	result, err := s.underlying.Get(ctx, key)

	s.mu.Lock()
	if err == nil {
		s.hitCount++
	} else {
		s.missCount++
	}
	s.mu.Unlock()

	return result, err
}

// Set stores a cache entry under the given key.
//
// Takes ctx (context.Context) for cancellation.
// Takes key (string) which identifies the cache entry.
// Takes result (*annotator_dto.ProjectAnnotationResult) which is the value to
// store.
//
// Returns error when the underlying cache fails to store the entry.
func (s *CacheSpy) Set(ctx context.Context, key string, result *annotator_dto.ProjectAnnotationResult) error {
	return s.underlying.Set(ctx, key, result)
}

// Clear removes all entries from the underlying cache.
//
// Returns error when the underlying cache fails to clear.
func (s *CacheSpy) Clear(ctx context.Context) error {
	return s.underlying.Clear(ctx)
}

// GetStats returns the current spy statistics.
//
// Returns CacheStats which contains the accumulated spy metrics.
//
// Safe for concurrent use.
func (s *CacheSpy) GetStats() CacheStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return CacheStats{
		GetCount:  s.getCount,
		HitCount:  s.hitCount,
		MissCount: s.missCount,
		LastKey:   s.lastKey,
	}
}

// ResetStats clears all tracking statistics.
//
// Safe for concurrent use.
func (s *CacheSpy) ResetStats() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCount = 0
	s.hitCount = 0
	s.missCount = 0
	s.lastKey = ""
}

// CacheStats holds cache usage numbers collected by a spy during testing.
type CacheStats struct {
	// LastKey is the most recently accessed cache key.
	LastKey string

	// GetCount is the number of cache retrieval operations performed.
	GetCount int

	// HitCount is the number of cache hits.
	HitCount int

	// MissCount is the number of cache misses.
	MissCount int
}

// IntrospectionCacheSpy wraps an introspection cache and tracks operations.
type IntrospectionCacheSpy struct {
	// underlying is the wrapped cache that performs the actual operations.
	underlying coordinator_domain.IntrospectionCachePort

	// lastKey is the most recent cache key accessed via Get.
	lastKey string

	// getCount is the number of times Get has been called.
	getCount int

	// hitCount is the number of cache hits.
	hitCount int

	// missCount is the number of cache lookups that did not find an entry.
	missCount int

	// mu guards access to the spy statistics fields.
	mu sync.Mutex
}

// NewIntrospectionCacheSpy creates a new introspection cache spy.
//
// Takes underlying (IntrospectionCachePort) which provides the cache to wrap.
//
// Returns *IntrospectionCacheSpy which wraps the underlying cache for testing.
func NewIntrospectionCacheSpy(underlying coordinator_domain.IntrospectionCachePort) *IntrospectionCacheSpy {
	return &IntrospectionCacheSpy{
		underlying: underlying,
	}
}

// Get retrieves a cache entry by key, delegating to the underlying cache.
//
// Takes key (string) which identifies the cache entry to retrieve.
//
// Returns *coordinator_domain.IntrospectionCacheEntry which is the cached
// entry if found.
// Returns error when the key is not found or the underlying cache fails.
//
// Safe for concurrent use. Access to spy counters is protected by a mutex.
func (s *IntrospectionCacheSpy) Get(ctx context.Context, key string) (*coordinator_domain.IntrospectionCacheEntry, error) {
	s.mu.Lock()
	s.getCount++
	s.lastKey = key
	s.mu.Unlock()

	result, err := s.underlying.Get(ctx, key)

	s.mu.Lock()
	if err == nil {
		s.hitCount++
	} else {
		s.missCount++
	}
	s.mu.Unlock()

	return result, err
}

// Set stores an introspection cache entry under the given key.
//
// Takes key (string) which identifies the cache entry.
// Takes entry (*coordinator_domain.IntrospectionCacheEntry) which is the value
// to store.
//
// Returns error when the underlying cache fails to store the entry.
func (s *IntrospectionCacheSpy) Set(ctx context.Context, key string, entry *coordinator_domain.IntrospectionCacheEntry) error {
	return s.underlying.Set(ctx, key, entry)
}

// Clear removes all entries from the underlying cache.
//
// Returns error when the cache cannot be cleared.
func (s *IntrospectionCacheSpy) Clear(ctx context.Context) error {
	return s.underlying.Clear(ctx)
}

// GetStats returns the current spy statistics.
//
// Returns CacheStats which contains the accumulated counters and last key.
//
// Safe for concurrent use.
func (s *IntrospectionCacheSpy) GetStats() CacheStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return CacheStats{
		GetCount:  s.getCount,
		HitCount:  s.hitCount,
		MissCount: s.missCount,
		LastKey:   s.lastKey,
	}
}

// ResetStats clears all tracking statistics.
//
// Safe for concurrent use.
func (s *IntrospectionCacheSpy) ResetStats() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCount = 0
	s.hitCount = 0
	s.missCount = 0
	s.lastKey = ""
}
