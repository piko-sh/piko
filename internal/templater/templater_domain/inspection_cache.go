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

package templater_domain

import (
	"sync"
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

// InspectionResult holds cached inspection data for a PK component.
// It stores the parsed component, its dependencies, and the script hash used
// as the cache key.
type InspectionResult struct {
	// Component is the virtual component from the annotator.
	Component *annotator_dto.VirtualComponent

	// Timestamp is when this result was cached.
	Timestamp time.Time

	// ScriptHash is the hash of the script block this result belongs to.
	ScriptHash string
}

// InspectionCache provides a thread-safe cache for component inspection
// results. Results are keyed by file path and script hash, which allows reuse
// when scripts have not changed.
type InspectionCache interface {
	// Get retrieves a cached inspection result if it exists and matches the
	// script hash.
	//
	// Takes path (string) which identifies the cached item.
	// Takes scriptHash (string) which must match the stored hash for a valid hit.
	//
	// Returns *InspectionResult which contains the cached data.
	// Returns bool which indicates whether a valid cache entry was found.
	Get(path, scriptHash string) (*InspectionResult, bool)

	// Store caches an inspection result for a given path and script hash.
	//
	// Takes path (string) which identifies the file being inspected.
	// Takes scriptHash (string) which uniquely identifies the script version.
	// Takes result (*InspectionResult) which contains the inspection findings.
	Store(path, scriptHash string, result *InspectionResult)

	// Remove deletes all cached results for a specific file path.
	//
	// Takes path (string) which identifies the file whose cached results should be
	// removed.
	Remove(path string)

	// Clear removes all cached inspection results.
	Clear()

	// Stats returns cache statistics for monitoring.
	//
	// Returns InspectionCacheStats which contains the current cache metrics.
	Stats() InspectionCacheStats
}

// InspectionCacheStats provides metrics about cache performance.
type InspectionCacheStats struct {
	// TotalEntries is the number of entries stored in the cache.
	TotalEntries int

	// Hits is the number of cache lookups that found an existing entry.
	Hits int64

	// Misses is the number of cache lookups that did not find an entry.
	Misses int64

	// Evictions is the number of items removed from the cache to make space.
	Evictions int64
}

// inMemoryInspectionCache implements InspectionCache using an in-memory store.
// It uses a two-level map: file path to script hash to result.
type inMemoryInspectionCache struct {
	// cache is a two-level map: path to script hash to result.
	cache map[string]map[string]*InspectionResult

	// hits is the number of successful cache lookups.
	hits atomic.Int64

	// misses counts cache lookups where no matching entry was found.
	misses atomic.Int64

	// evictions counts cache entries removed via Store overwrites, Remove, or Clear.
	evictions atomic.Int64

	// mu guards concurrent access to the cache map.
	mu sync.RWMutex
}

// Get retrieves a cached inspection result.
//
// Takes path (string) which specifies the file path to look up.
// Takes scriptHash (string) which identifies the specific version of the file.
//
// Returns *InspectionResult which is the cached result, or nil if not found.
// Returns bool which indicates whether the result was found in the cache.
//
// Safe for concurrent use. Protected by a read lock.
func (c *inMemoryInspectionCache) Get(path, scriptHash string) (*InspectionResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	pathCache, pathExists := c.cache[path]
	if !pathExists {
		c.misses.Add(1)
		return nil, false
	}

	result, hashExists := pathCache[scriptHash]
	if !hashExists {
		c.misses.Add(1)
		return nil, false
	}

	c.hits.Add(1)
	return result, true
}

// Store caches an inspection result.
//
// Takes path (string) which identifies the file path for the cached result.
// Takes scriptHash (string) which is the hash of the script version.
// Takes result (*InspectionResult) which is the inspection result to cache.
//
// Safe for concurrent use.
func (c *inMemoryInspectionCache) Store(path, scriptHash string, result *InspectionResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.cache[path]; !exists {
		c.cache[path] = make(map[string]*InspectionResult)
	}

	if _, exists := c.cache[path][scriptHash]; exists {
		c.evictions.Add(1)
	}

	c.cache[path][scriptHash] = result
}

// Remove deletes all cached inspection results for a specific file.
//
// Takes path (string) which specifies the file path to remove from the cache.
//
// Safe for concurrent use.
func (c *inMemoryInspectionCache) Remove(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if pathCache, exists := c.cache[path]; exists {
		c.evictions.Add(int64(len(pathCache)))
		delete(c.cache, path)
	}
}

// Clear removes all cached inspection results.
//
// Safe for concurrent use.
func (c *inMemoryInspectionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	totalEntries := 0
	for _, pathCache := range c.cache {
		totalEntries += len(pathCache)
	}
	c.evictions.Add(int64(totalEntries))

	c.cache = make(map[string]map[string]*InspectionResult)
}

// Stats returns current cache statistics.
//
// Returns InspectionCacheStats which contains entry count and hit/miss data.
//
// Safe for concurrent use.
func (c *inMemoryInspectionCache) Stats() InspectionCacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalEntries := 0
	for _, pathCache := range c.cache {
		totalEntries += len(pathCache)
	}

	return InspectionCacheStats{
		TotalEntries: totalEntries,
		Hits:         c.hits.Load(),
		Misses:       c.misses.Load(),
		Evictions:    c.evictions.Load(),
	}
}

// NewInMemoryInspectionCache creates a new in-memory inspection cache.
//
// Returns InspectionCache which provides safe caching of inspection results
// for use by multiple goroutines at the same time.
func NewInMemoryInspectionCache() InspectionCache {
	return &inMemoryInspectionCache{
		mu:    sync.RWMutex{},
		cache: make(map[string]map[string]*InspectionResult),
	}
}
