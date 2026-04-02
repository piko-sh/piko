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

package inspector_domain

import (
	"context"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

// TypeDataProvider defines the contract for caching the querier's encoded type
// data. Implementations can use in-memory, file system, or Redis storage.
type TypeDataProvider interface {
	// GetTypeData retrieves serialised type information from the cache.
	//
	// Takes cacheKey (string) which identifies the cached type data.
	//
	// Returns *inspector_dto.TypeData which contains the type information.
	// Returns error when the data for the given key is not found (a cache miss)
	// or when there is an error reading from the underlying storage.
	GetTypeData(ctx context.Context, cacheKey string) (*inspector_dto.TypeData, error)

	// SaveTypeData saves serialised type information to the cache.
	// Implementations should handle creation or overwriting of data for the given
	// cache key and should aim for atomic writes to prevent cache corruption.
	//
	// Takes cacheKey (string) which identifies the cache entry.
	// Takes data (*inspector_dto.TypeData) which contains the type information.
	//
	// Returns error when the save operation fails.
	SaveTypeData(ctx context.Context, cacheKey string, data *inspector_dto.TypeData) error

	// InvalidateCache removes a specific entry from the cache.
	// It should not return an error if the key does not exist.
	//
	// Takes cacheKey (string) which identifies the entry to remove.
	//
	// Returns error when the cache operation fails.
	InvalidateCache(ctx context.Context, cacheKey string) error

	// ClearCache removes all entries from the cache.
	//
	// Returns error when the cache cannot be cleared.
	ClearCache(ctx context.Context) error
}
