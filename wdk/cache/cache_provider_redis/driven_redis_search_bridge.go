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

package cache_provider_redis

import (
	"context"
	"fmt"

	"piko.sh/piko/wdk/cache"
)

// Search performs full-text search across indexed TEXT fields
// using RediSearch.
//
// When no SearchSchema is configured, returns
// ErrSearchNotSupported.
//
// Takes query (string) which is the search query to match
// against TEXT fields.
// Takes opts (*cache.SearchOptions) which contains filters,
// pagination, and sorting settings.
//
// Returns the result containing matching items and total count.
// Returns error if search is not supported or the search fails.
func (a *RedisAdapter[K, V]) Search(ctx context.Context, query string, opts *cache.SearchOptions) (cache.SearchResult[K, V], error) {
	if a.schema == nil {
		return cache.SearchResult[K, V]{}, fmt.Errorf(
			"%w: configure a SearchSchema via Searchable() to enable search",
			cache.ErrSearchNotSupported,
		)
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.searchTimeout, fmt.Errorf("redis Search exceeded %s timeout", a.searchTimeout))
	defer cancel()

	return a.searchWithRediSearch(timeoutCtx, query, opts)
}

// Query performs structured filtering, sorting, and pagination
// without full-text search.
//
// When no SearchSchema is configured, returns
// ErrSearchNotSupported.
//
// Takes opts (*cache.QueryOptions) which contains filters,
// pagination, and sorting settings.
//
// Returns the result containing matching items and total count.
// Returns error if search is not supported or the query fails.
func (a *RedisAdapter[K, V]) Query(ctx context.Context, opts *cache.QueryOptions) (cache.SearchResult[K, V], error) {
	if a.schema == nil {
		return cache.SearchResult[K, V]{}, fmt.Errorf(
			"%w: configure a SearchSchema via Searchable() to enable query",
			cache.ErrSearchNotSupported,
		)
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.searchTimeout, fmt.Errorf("redis Query exceeded %s timeout", a.searchTimeout))
	defer cancel()

	return a.queryWithRediSearch(timeoutCtx, opts)
}

// SupportsSearch returns true if a SearchSchema was configured for this cache.
//
// Returns bool indicating whether search operations are available.
func (a *RedisAdapter[K, V]) SupportsSearch() bool {
	return a.schema != nil
}

// GetSchema returns the search schema set for this cache.
//
// Returns *cache.SearchSchema which is the schema for this cache, or nil if
// the cache does not support search.
func (a *RedisAdapter[K, V]) GetSchema() *cache.SearchSchema {
	return a.schema
}
