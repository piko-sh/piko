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

package storage_domain

import (
	"context"
	"sync"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/wdk/clock"
)

// DefaultRIDCleanupInterval is the default interval for purging expired random
// identifiers.
const DefaultRIDCleanupInterval = 1 * time.Minute

// ridEntry stores a random identifier with its expiry time.
type ridEntry struct {
	// expiresAt is when this entry should be removed from the cache.
	expiresAt time.Time
}

// PresignRIDCacheOption configures a PresignRIDCache.
type PresignRIDCacheOption func(*PresignRIDCache)

// PresignRIDCache provides thread-safe random identifier tracking for replay
// protection. It stores random identifiers with their expiry times and
// periodically purges expired entries.
type PresignRIDCache struct {
	// rids maps random identifier strings to their expiry details.
	rids map[string]ridEntry

	// clock provides time operations for expiry checks.
	clock clock.Clock

	// stopCh signals the cleanup goroutine to stop.
	stopCh chan struct{}

	// mu guards concurrent access to the rids map.
	mu sync.RWMutex

	// stopped indicates whether Stop has been called.
	stopped bool
}

// NewPresignRIDCache creates a new random identifier cache with background
// cleanup.
//
// Takes ctx (context.Context) which is threaded to the background cleanup
// goroutine for panic recovery.
// Takes cleanupInterval (time.Duration) which specifies how often to purge
// expired identifiers. Use DefaultRIDCleanupInterval for sensible defaults.
//
// Returns *PresignRIDCache which is ready for use.
//
// Spawns a background goroutine for periodic cleanup. Call Stop to terminate
// the cleanup goroutine when the cache is no longer needed.
func NewPresignRIDCache(ctx context.Context, cleanupInterval time.Duration, opts ...PresignRIDCacheOption) *PresignRIDCache {
	if cleanupInterval <= 0 {
		cleanupInterval = DefaultRIDCleanupInterval
	}

	c := &PresignRIDCache{
		rids:    make(map[string]ridEntry),
		mu:      sync.RWMutex{},
		stopCh:  make(chan struct{}),
		clock:   clock.RealClock(),
		stopped: false,
	}
	for _, opt := range opts {
		opt(c)
	}

	go c.cleanupLoop(ctx, cleanupInterval)

	return c
}

// Add registers a random identifier with the given expiry time. Returns true
// if the identifier was added (not seen before), or false if it already exists
// (indicating a potential replay attack).
//
// Takes rid (string) which is the random identifier value to register.
// Takes expiresAt (time.Time) which is when the identifier should expire.
//
// Returns bool which is true if the identifier was successfully added, false if
// it was already present.
//
// Safe for concurrent use. Protected by a mutex.
func (c *PresignRIDCache) Add(rid string, expiresAt time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.rids[rid]; exists {
		return false
	}

	c.rids[rid] = ridEntry{
		expiresAt: expiresAt,
	}
	return true
}

// Has checks whether a random identifier has already been used.
//
// Takes rid (string) which is the random identifier value to check.
//
// Returns bool which is true if the identifier is known, false otherwise.
//
// Safe for concurrent use.
func (c *PresignRIDCache) Has(rid string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.rids[rid]
	return exists
}

// Count returns the number of random identifiers currently in the cache.
//
// Returns int which is the current cache size.
//
// Safe for concurrent use.
func (c *PresignRIDCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.rids)
}

// Clear removes all random identifiers from the cache.
//
// Safe for concurrent use.
func (c *PresignRIDCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rids = make(map[string]ridEntry)
}

// Stop terminates the background cleanup goroutine.
// This should be called during graceful shutdown.
func (c *PresignRIDCache) Stop() {
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return
	}
	c.stopped = true
	c.mu.Unlock()

	close(c.stopCh)
}

// cleanupLoop removes expired random identifiers at regular intervals.
//
// Takes ctx (context.Context) which provides context for panic recovery.
// Takes interval (time.Duration) which sets how often cleanup runs.
func (c *PresignRIDCache) cleanupLoop(ctx context.Context, interval time.Duration) {
	ticker := c.clock.NewTicker(interval)
	defer ticker.Stop()
	defer goroutine.RecoverPanic(context.WithoutCancel(ctx), "storage.presignCleanupLoop")

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C():
			c.purgeExpired()
		}
	}
}

// purgeExpired removes all random identifiers that have passed their expiry
// time.
//
// Safe for concurrent use; acquires the cache mutex during iteration.
func (c *PresignRIDCache) purgeExpired() {
	now := c.clock.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	for rid, entry := range c.rids {
		if now.After(entry.expiresAt) {
			delete(c.rids, rid)
		}
	}
}

// WithPresignRIDCacheClock sets the clock used for expiry checks. If not
// provided, the real system clock is used.
//
// Takes c (clock.Clock) which provides time operations.
//
// Returns PresignRIDCacheOption which configures the cache's clock.
func WithPresignRIDCacheClock(c clock.Clock) PresignRIDCacheOption {
	return func(cache *PresignRIDCache) {
		if c != nil {
			cache.clock = c
		}
	}
}
