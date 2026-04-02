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
	"path/filepath"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/logger/logger_domain"
)

// getFromFileHashCache retrieves a cached file hash if it exists and is fresh.
//
// Takes mu (*sync.RWMutex) which protects concurrent access to the cache.
// Takes cache (map[string]cacheEntry) which stores the file hash entries.
// Takes spanName (string) which identifies the tracing span.
// Takes path (string) which is the file path to look up.
// Takes modTime (time.Time) which is the current modification time of the file.
//
// Returns string which is the cached hash value, or empty if not found.
// Returns bool which indicates whether a valid cache entry was found.
//
// Safe for concurrent use. Uses a read lock to access the cache.
func getFromFileHashCache(
	ctx context.Context,
	mu *sync.RWMutex,
	cache map[string]cacheEntry,
	spanName string,
	path string,
	modTime time.Time,
) (string, bool) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, spanName,
		logger_domain.String(logKeyPath, path),
		logger_domain.String("mod_time", modTime.Format(time.RFC3339Nano)),
	)
	defer span.End()

	cleanPath := filepath.Clean(path)

	mu.RLock()
	entry, exists := cache[cleanPath]
	mu.RUnlock()

	if !exists {
		l.Trace("Cache MISS: file not in cache.", logger_domain.String(logKeyPath, cleanPath))
		span.SetAttributes(attribute.String("cache.status", "MISS_NOT_FOUND"))
		span.SetStatus(codes.Ok, "Cache miss - file not found")
		return "", false
	}

	if !entry.ModTime.Equal(modTime) {
		l.Trace("Cache MISS: modification time changed.",
			logger_domain.String(logKeyPath, cleanPath),
			logger_domain.String("cached_mod_time", entry.ModTime.Format(time.RFC3339Nano)),
			logger_domain.String("actual_mod_time", modTime.Format(time.RFC3339Nano)),
		)
		span.SetAttributes(attribute.String("cache.status", "MISS_STALE"))
		span.SetStatus(codes.Ok, "Cache miss - file modified")
		return "", false
	}

	l.Trace("Cache HIT.",
		logger_domain.String(logKeyPath, cleanPath),
		logger_domain.String("hash", entry.Hash),
	)
	span.SetAttributes(attribute.String("cache.status", "HIT"))
	span.SetStatus(codes.Ok, "Cache hit")

	return entry.Hash, true
}

// setInFileHashCache stores a file hash entry in the provided cache.
//
// Takes mu (*sync.RWMutex) which guards access to the cache map.
// Takes cache (map[string]cacheEntry) which stores the hash entries.
// Takes spanName (string) which identifies the tracing span.
// Takes path (string) which is the file path to use as the cache key.
// Takes modTime (time.Time) which is the file modification time.
// Takes hash (string) which is the computed hash value to store.
//
// Safe for concurrent use. Uses the provided mutex to serialise cache writes.
func setInFileHashCache(
	ctx context.Context,
	mu *sync.RWMutex,
	cache map[string]cacheEntry,
	spanName string,
	path string,
	modTime time.Time,
	hash string,
) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, spanName,
		logger_domain.String(logKeyPath, path),
		logger_domain.String("hash", hash),
	)
	defer span.End()

	cleanPath := filepath.Clean(path)

	mu.Lock()
	cache[cleanPath] = cacheEntry{
		ModTime: modTime,
		Hash:    hash,
	}
	mu.Unlock()

	l.Trace("Cache entry updated.",
		logger_domain.String(logKeyPath, cleanPath),
		logger_domain.String("mod_time", modTime.Format(time.RFC3339Nano)),
	)
	span.SetStatus(codes.Ok, "Cache set successfully")
}
