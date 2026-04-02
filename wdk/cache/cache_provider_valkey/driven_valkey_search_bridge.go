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

package cache_provider_valkey

import (
	"context"
	"fmt"
	"strings"
	"time"

	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/safeconv"
)

// EstimatedSize returns the approximate number of keys in the Valkey database.
//
// Returns int which is the count of keys, or zero if the query fails.
func (a *ValkeyAdapter[K, V]) EstimatedSize() int {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("valkey EstimatedSize exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	if a.namespace == "" {
		size, err := a.client.Do(ctx, a.client.B().Dbsize().Build()).AsInt64()
		if err != nil {
			l.Warn("Failed to get DBSize from Valkey", logger.Error(err))
			return 0
		}
		return int(size)
	}

	var count int
	scanPattern := a.namespace + "*"
	var cursor uint64
	for {
		response, err := a.client.Do(ctx, a.client.B().Scan().Cursor(cursor).Match(scanPattern).Count(scanBatchSize).Build()).AsScanEntry()
		if err != nil {
			l.Warn("Failed to scan keys for EstimatedSize", logger.Error(err))
			break
		}
		count += len(response.Elements)
		cursor = response.Cursor
		if cursor == 0 {
			break
		}
	}
	return count
}

// Stats returns cache statistics from the Valkey server.
//
// Returns cache.Stats which contains hit and miss counts from the server's
// INFO command.
func (a *ValkeyAdapter[K, V]) Stats() cache.Stats {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("valkey Stats exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	info, err := a.client.Do(ctx, a.client.B().Info().Section("stats").Build()).ToString()
	if err != nil {
		l.Warn("Failed to get INFO from Valkey", logger.Error(err))
		return cache.Stats{}
	}

	var hits, misses int64
	for line := range strings.SplitSeq(info, "\r\n") {
		if strings.HasPrefix(line, "keyspace_hits:") {
			if _, err := fmt.Sscanf(line, "keyspace_hits:%d", &hits); err != nil {
				l.Trace("Failed to parse keyspace_hits", logger.String("line", line), logger.Error(err))
			}
		}
		if strings.HasPrefix(line, "keyspace_misses:") {
			if _, err := fmt.Sscanf(line, "keyspace_misses:%d", &misses); err != nil {
				l.Trace("Failed to parse keyspace_misses", logger.String("line", line), logger.Error(err))
			}
		}
	}

	var hitsUint, missesUint uint64
	if hits > 0 {
		hitsUint = safeconv.Int64ToUint64(hits)
	}
	if misses > 0 {
		missesUint = safeconv.Int64ToUint64(misses)
	}
	return cache.Stats{
		Hits:             hitsUint,
		Misses:           missesUint,
		Evictions:        0,
		LoadSuccessCount: 0,
		LoadFailureCount: 0,
		TotalLoadTime:    0,
	}
}

// Close releases the Valkey client connection.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when resources cannot be released cleanly.
func (a *ValkeyAdapter[K, V]) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.client.Close()
	return nil
}

// SetExpiresAfter updates the time-to-live for an existing key using the
// Valkey EXPIRE command.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies the new time-to-live.
//
// Returns error when the operation fails.
func (a *ValkeyAdapter[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey SetExpiresAfter exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf("failed to encode key: %w", err)
	}

	if err := a.client.Do(timeoutCtx, a.client.B().Pexpire().Key(keyString).Milliseconds(expiresAfter.Milliseconds()).Build()).Error(); err != nil {
		return fmt.Errorf("valkey pexpire failed for key %s: %w", keyString, err)
	}

	return nil
}

// GetMaximum returns the Valkey maxmemory configuration value.
//
// Returns uint64 which is the maximum memory in bytes, or 0 if the value
// cannot be retrieved or parsed.
func (a *ValkeyAdapter[K, V]) GetMaximum() uint64 {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("valkey GetMaximum exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	result, err := a.client.Do(ctx, a.client.B().ConfigGet().Parameter("maxmemory").Build()).AsStrMap()
	if err != nil {
		return 0
	}
	if maxString, ok := result["maxmemory"]; ok {
		var maxMemory uint64
		if _, err := fmt.Sscanf(maxString, "%d", &maxMemory); err != nil {
			l.Trace("Failed to parse maxmemory", logger.String("value", maxString), logger.Error(err))
			return 0
		}
		return maxMemory
	}
	return 0
}

// SetMaximum is not supported by the Valkey provider as it is a server-level
// configuration.
func (*ValkeyAdapter[K, V]) SetMaximum(_ uint64) {
	_, l := logger.From(context.Background(), log)
	l.Warn("SetMaximum is not supported by the Valkey provider and will have no effect.")
}

// WeightedSize returns the memory usage in bytes from the Valkey used_memory
// statistic.
//
// Returns uint64 which is the memory usage in bytes, or zero if the statistic
// cannot be read.
func (a *ValkeyAdapter[K, V]) WeightedSize() uint64 {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("valkey WeightedSize exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	info, err := a.client.Do(ctx, a.client.B().Info().Section("memory").Build()).ToString()
	if err != nil {
		return 0
	}
	for line := range strings.SplitSeq(info, "\r\n") {
		if strings.HasPrefix(line, "used_memory:") {
			var used uint64
			if _, err := fmt.Sscanf(line, "used_memory:%d", &used); err != nil {
				l.Trace("Failed to parse used_memory", logger.String("line", line), logger.Error(err))
				return 0
			}
			return used
		}
	}
	return 0
}

// SetRefreshableAfter is a no-op as Valkey does not natively support refresh
// scheduling.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error which is always nil for this provider.
func (*ValkeyAdapter[K, V]) SetRefreshableAfter(ctx context.Context, _ K, _ time.Duration) error {
	_, l := logger.From(ctx, log)

	l.Internal("SetRefreshableAfter is not natively supported by the Valkey provider.")
	return nil
}

// Search performs full-text search across indexed TEXT fields, returning an
// error for text queries because Valkey Search does not support TEXT fields
// and directing callers to use Query() instead.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes query (string) which is the search query to match against TEXT fields.
// Takes opts (*cache.SearchOptions) which contains filters, pagination, and
// sorting.
//
// Returns SearchResult containing matching items and total count.
// Returns error if search is not supported or the search fails.
func (a *ValkeyAdapter[K, V]) Search(ctx context.Context, query string, opts *cache.SearchOptions) (cache.SearchResult[K, V], error) {
	if a.schema == nil {
		return cache.SearchResult[K, V]{}, fmt.Errorf(
			"%w: configure a SearchSchema via Searchable() to enable search",
			cache.ErrSearchNotSupported,
		)
	}

	if query != "" {
		return cache.SearchResult[K, V]{}, fmt.Errorf(
			"%w: Valkey Search does not support full-text TEXT fields; use Query() with TAG/NUMERIC filters instead",
			cache.ErrSearchNotSupported,
		)
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.searchTimeout, fmt.Errorf("valkey Search exceeded %s timeout", a.searchTimeout))
	defer cancel()

	return a.searchWithValkeySearch(timeoutCtx, query, opts)
}

// Query performs structured filtering, sorting, and pagination without
// full-text search.
// Returns ErrSearchNotSupported if no SearchSchema was configured.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes opts (*cache.QueryOptions) which contains filters, pagination, and
// sorting.
//
// Returns SearchResult containing matching items and total count.
// Returns error if search is not supported or the query fails.
func (a *ValkeyAdapter[K, V]) Query(ctx context.Context, opts *cache.QueryOptions) (cache.SearchResult[K, V], error) {
	if a.schema == nil {
		return cache.SearchResult[K, V]{}, fmt.Errorf(
			"%w: configure a SearchSchema via Searchable() to enable query",
			cache.ErrSearchNotSupported,
		)
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.searchTimeout, fmt.Errorf("valkey Query exceeded %s timeout", a.searchTimeout))
	defer cancel()

	return a.queryWithValkeySearch(timeoutCtx, opts)
}

// SupportsSearch returns true if a SearchSchema was configured for this cache.
//
// Returns bool indicating whether search operations are available.
func (a *ValkeyAdapter[K, V]) SupportsSearch() bool {
	return a.schema != nil
}

// GetSchema returns the search schema set for this cache.
//
// Returns *cache.SearchSchema which is the schema for this cache, or nil if
// the cache does not support search.
func (a *ValkeyAdapter[K, V]) GetSchema() *cache.SearchSchema {
	return a.schema
}
