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

package coordinator_adapters

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// memoryFileHashCache is a thread-safe, in-memory implementation of
// FileHashCachePort that provides stat-then-read optimisation without
// persistent storage. It is best suited for development mode where cache
// lifetime matches process lifetime.
type memoryFileHashCache struct {
	// cache maps file paths to their cached entries. Paths are absolute and
	// cleaned using filepath.Clean.
	cache map[string]cacheEntry

	// mu guards access to the cache map for safe concurrent reads and writes.
	mu sync.RWMutex
}

var _ coordinator_domain.FileHashCachePort = (*memoryFileHashCache)(nil)

// Get retrieves the cached hash for a file if its modification time matches.
// This uses a "stat-then-read" pattern by letting the caller check the cache
// using only file metadata (ModTime from os.Stat).
//
// Takes path (string) which specifies the file path to look up.
// Takes modTime (time.Time) which is the current modification time to compare.
//
// Returns hash (string) which is the cached hash value if found and valid.
// Returns found (bool) which indicates whether a valid cache entry exists.
func (m *memoryFileHashCache) Get(ctx context.Context, path string, modTime time.Time) (hash string, found bool) {
	return getFromFileHashCache(ctx, &m.mu, m.cache, "MemoryFileHashCache.Get", path, modTime)
}

// Set stores or updates the cached hash for a file with its modification time.
// Call this after computing a new hash from the file's content.
//
// Takes path (string) which is the file path to cache.
// Takes modTime (time.Time) which is the file's modification time.
// Takes hash (string) which is the computed hash of the file content.
func (m *memoryFileHashCache) Set(ctx context.Context, path string, modTime time.Time, hash string) {
	setInFileHashCache(ctx, &m.mu, m.cache, "MemoryFileHashCache.Set", path, modTime, hash)
}

// Load performs no operation for the in-memory cache.
// It exists to satisfy the FileHashCachePort interface.
//
// Returns error when the operation fails, though this always returns nil.
func (*memoryFileHashCache) Load(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "MemoryFileHashCache.Load")
	defer span.End()

	l.Internal("Load called on in-memory cache (no-op).")
	span.SetStatus(codes.Ok, "No-op for memory cache")
	return nil
}

// Persist does nothing for the in-memory cache.
// It satisfies the FileHashCachePort interface.
//
// Returns error which is always nil for this implementation.
func (*memoryFileHashCache) Persist(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "MemoryFileHashCache.Persist")
	defer span.End()

	l.Internal("Persist called on in-memory cache (no-op).")
	span.SetStatus(codes.Ok, "No-op for memory cache")
	return nil
}

// NewMemoryFileHashCache creates a new in-memory file hash cache.
// The cache starts empty and is filled as hashes are calculated.
//
// Returns coordinator_domain.FileHashCachePort which provides the cache
// interface for file hash operations.
func NewMemoryFileHashCache() coordinator_domain.FileHashCachePort {
	return &memoryFileHashCache{
		cache: make(map[string]cacheEntry),
		mu:    sync.RWMutex{},
	}
}
