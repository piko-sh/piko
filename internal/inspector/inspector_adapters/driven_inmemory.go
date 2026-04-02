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

package inspector_adapters

import (
	"context"
	"fmt"
	"sync"

	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// InMemoryProvider is a TypeData cache that stores data in memory.
// It is primarily used for testing and development purposes.
type InMemoryProvider struct {
	// err is an optional error returned by all operations; used in tests to
	// simulate error conditions.
	err error

	// data holds cached TypeData entries, keyed by cache key string.
	data map[string]*inspector_dto.TypeData

	// mu guards concurrent access to the err and data fields.
	mu sync.RWMutex
}

var _ inspector_domain.TypeDataProvider = (*InMemoryProvider)(nil)

// NewInMemoryProvider creates a new InMemoryProvider with optional initial data.
// If initialData is nil, an empty map is created.
//
// Takes initialData (map[string]*inspector_dto.TypeData) which provides the
// initial type data to populate the provider.
//
// Returns *InMemoryProvider which is ready for use.
func NewInMemoryProvider(initialData map[string]*inspector_dto.TypeData) *InMemoryProvider {
	if initialData == nil {
		initialData = make(map[string]*inspector_dto.TypeData)
	}
	return &InMemoryProvider{
		err:  nil,
		data: initialData,
		mu:   sync.RWMutex{},
	}
}

// GetTypeData retrieves cached type data for the given cache key.
//
// Takes cacheKey (string) which identifies the cached type data to retrieve.
//
// Returns *inspector_dto.TypeData which contains the cached type information.
// Returns error when a test error is set or the key is not found.
//
// Safe for concurrent use; protected by a read lock.
func (p *InMemoryProvider) GetTypeData(_ context.Context, cacheKey string) (*inspector_dto.TypeData, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.err != nil {
		return nil, p.err
	}

	if d, ok := p.data[cacheKey]; ok {
		return d, nil
	}

	return nil, fmt.Errorf("in-memory cache miss for key: %s", cacheKey)
}

// SaveTypeData stores TypeData in the cache with the given key.
// Returns a simulated error if one is set.
//
// Takes cacheKey (string) which identifies the cache entry.
// Takes typeData (*inspector_dto.TypeData) which is the type data to store.
//
// Returns error when a simulated error has been set on the provider.
//
// Safe for concurrent use.
func (p *InMemoryProvider) SaveTypeData(_ context.Context, cacheKey string, typeData *inspector_dto.TypeData) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.err != nil {
		return p.err
	}

	p.data[cacheKey] = typeData
	return nil
}

// InvalidateCache removes the cached TypeData for the given key.
//
// Takes cacheKey (string) which identifies the cache entry to remove.
//
// Returns error when a simulated error is set on the provider.
//
// Safe for concurrent use.
func (p *InMemoryProvider) InvalidateCache(_ context.Context, cacheKey string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.err != nil {
		return p.err
	}

	delete(p.data, cacheKey)
	return nil
}

// ClearCache removes all cached data from the provider.
//
// Returns error when a test error has been set on the provider.
//
// Safe for concurrent use.
func (p *InMemoryProvider) ClearCache(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.err != nil {
		return p.err
	}

	p.data = make(map[string]*inspector_dto.TypeData)
	return nil
}
