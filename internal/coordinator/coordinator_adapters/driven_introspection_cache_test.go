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
	"errors"
	"iter"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
)

var errInjectedCache = errors.New("injected cache failure")

type failingCache struct {
	getIfPresentValue *coordinator_domain.IntrospectionCacheEntry
	getIfPresentFound bool
	getIfPresentErr   error
	setErr            error
	invalidateErr     error
	invalidateAllErr  error
	closeErr          error
}

func (f *failingCache) GetIfPresent(_ context.Context, _ string) (*coordinator_domain.IntrospectionCacheEntry, bool, error) {
	return f.getIfPresentValue, f.getIfPresentFound, f.getIfPresentErr
}

func (*failingCache) Get(_ context.Context, _ string, _ cache_dto.Loader[string, *coordinator_domain.IntrospectionCacheEntry]) (*coordinator_domain.IntrospectionCacheEntry, error) {
	return nil, nil
}

func (f *failingCache) Set(_ context.Context, _ string, _ *coordinator_domain.IntrospectionCacheEntry, _ ...string) error {
	return f.setErr
}

func (*failingCache) SetWithTTL(_ context.Context, _ string, _ *coordinator_domain.IntrospectionCacheEntry, _ time.Duration, _ ...string) error {
	return nil
}

func (f *failingCache) Invalidate(_ context.Context, _ string) error {
	return f.invalidateErr
}

func (*failingCache) Compute(_ context.Context, _ string, _ func(*coordinator_domain.IntrospectionCacheEntry, bool) (*coordinator_domain.IntrospectionCacheEntry, cache_dto.ComputeAction)) (*coordinator_domain.IntrospectionCacheEntry, bool, error) {
	return nil, false, nil
}

func (*failingCache) ComputeIfAbsent(_ context.Context, _ string, _ func() *coordinator_domain.IntrospectionCacheEntry) (*coordinator_domain.IntrospectionCacheEntry, bool, error) {
	return nil, false, nil
}

func (*failingCache) ComputeIfPresent(_ context.Context, _ string, _ func(*coordinator_domain.IntrospectionCacheEntry) (*coordinator_domain.IntrospectionCacheEntry, cache_dto.ComputeAction)) (*coordinator_domain.IntrospectionCacheEntry, bool, error) {
	return nil, false, nil
}

func (*failingCache) ComputeWithTTL(_ context.Context, _ string, _ func(*coordinator_domain.IntrospectionCacheEntry, bool) cache_dto.ComputeResult[*coordinator_domain.IntrospectionCacheEntry]) (*coordinator_domain.IntrospectionCacheEntry, bool, error) {
	return nil, false, nil
}

func (*failingCache) BulkGet(_ context.Context, _ []string, _ cache_dto.BulkLoader[string, *coordinator_domain.IntrospectionCacheEntry]) (map[string]*coordinator_domain.IntrospectionCacheEntry, error) {
	return nil, nil
}

func (*failingCache) BulkSet(_ context.Context, _ map[string]*coordinator_domain.IntrospectionCacheEntry, _ ...string) error {
	return nil
}

func (*failingCache) InvalidateByTags(_ context.Context, _ ...string) (int, error) {
	return 0, nil
}

func (f *failingCache) InvalidateAll(_ context.Context) error {
	return f.invalidateAllErr
}

func (*failingCache) BulkRefresh(_ context.Context, _ []string, _ cache_dto.BulkLoader[string, *coordinator_domain.IntrospectionCacheEntry]) {
}

func (*failingCache) Refresh(_ context.Context, _ string, _ cache_dto.Loader[string, *coordinator_domain.IntrospectionCacheEntry]) <-chan cache_dto.LoadResult[*coordinator_domain.IntrospectionCacheEntry] {
	return nil
}

func (*failingCache) All() iter.Seq2[string, *coordinator_domain.IntrospectionCacheEntry] {
	return func(_ func(string, *coordinator_domain.IntrospectionCacheEntry) bool) {}
}

func (*failingCache) Keys() iter.Seq[string] {
	return func(_ func(string) bool) {}
}

func (*failingCache) Values() iter.Seq[*coordinator_domain.IntrospectionCacheEntry] {
	return func(_ func(*coordinator_domain.IntrospectionCacheEntry) bool) {}
}

func (*failingCache) GetEntry(_ context.Context, _ string) (cache_dto.Entry[string, *coordinator_domain.IntrospectionCacheEntry], bool, error) {
	return cache_dto.Entry[string, *coordinator_domain.IntrospectionCacheEntry]{}, false, nil
}

func (*failingCache) ProbeEntry(_ context.Context, _ string) (cache_dto.Entry[string, *coordinator_domain.IntrospectionCacheEntry], bool, error) {
	return cache_dto.Entry[string, *coordinator_domain.IntrospectionCacheEntry]{}, false, nil
}

func (*failingCache) EstimatedSize() int {
	return 0
}

func (*failingCache) Stats() cache_dto.Stats {
	return cache_dto.Stats{}
}

func (f *failingCache) Close(_ context.Context) error {
	return f.closeErr
}

func (*failingCache) GetMaximum() uint64 {
	return 0
}

func (*failingCache) SetMaximum(_ uint64) {}

func (*failingCache) WeightedSize() uint64 {
	return 0
}

func (*failingCache) SetExpiresAfter(_ context.Context, _ string, _ time.Duration) error {
	return nil
}

func (*failingCache) SetRefreshableAfter(_ context.Context, _ string, _ time.Duration) error {
	return nil
}

func (*failingCache) Search(_ context.Context, _ string, _ *cache_dto.SearchOptions) (cache_dto.SearchResult[string, *coordinator_domain.IntrospectionCacheEntry], error) {
	return cache_dto.SearchResult[string, *coordinator_domain.IntrospectionCacheEntry]{}, nil
}

func (*failingCache) Query(_ context.Context, _ *cache_dto.QueryOptions) (cache_dto.SearchResult[string, *coordinator_domain.IntrospectionCacheEntry], error) {
	return cache_dto.SearchResult[string, *coordinator_domain.IntrospectionCacheEntry]{}, nil
}

func (*failingCache) SupportsSearch() bool {
	return false
}

func (*failingCache) GetSchema() *cache_dto.SearchSchema {
	return nil
}

func validIntrospectionEntry() *coordinator_domain.IntrospectionCacheEntry {
	return &coordinator_domain.IntrospectionCacheEntry{
		VirtualModule:  &annotator_dto.VirtualModule{},
		TypeResolver:   &annotator_domain.TypeResolver{},
		ComponentGraph: &annotator_dto.ComponentGraph{},
		ScriptHashes:   map[string]string{"path": "hash"},
		Timestamp:      time.Now(),
		Version:        coordinator_domain.CurrentIntrospectionCacheVersion,
	}
}

func staleIntrospectionEntry() *coordinator_domain.IntrospectionCacheEntry {
	return &coordinator_domain.IntrospectionCacheEntry{
		VirtualModule:  &annotator_dto.VirtualModule{},
		TypeResolver:   &annotator_domain.TypeResolver{},
		ComponentGraph: &annotator_dto.ComponentGraph{},
		ScriptHashes:   map[string]string{},
		Timestamp:      time.Now(),
		Version:        coordinator_domain.CurrentIntrospectionCacheVersion - 1,
	}
}

func TestIntrospectionCache_SetSwallowsBackingError(t *testing.T) {
	t.Parallel()

	fake := &failingCache{setErr: errInjectedCache}
	cache := &introspectionCache{cache: fake}

	err := cache.Set(context.Background(), "key-1", validIntrospectionEntry())

	assert.NoError(t, err, "Set must not surface backing cache errors to callers")
}

func TestIntrospectionCache_SetSucceedsWhenBackingHealthy(t *testing.T) {
	t.Parallel()

	fake := &failingCache{}
	cache := &introspectionCache{cache: fake}

	err := cache.Set(context.Background(), "key-1", validIntrospectionEntry())

	require.NoError(t, err)
}

func TestIntrospectionCache_GetInvalidateErrorDoesNotPropagate(t *testing.T) {
	t.Parallel()

	fake := &failingCache{
		getIfPresentValue: staleIntrospectionEntry(),
		getIfPresentFound: true,
		invalidateErr:     errInjectedCache,
	}
	cache := &introspectionCache{cache: fake}

	entry, err := cache.Get(context.Background(), "key-1")

	assert.Nil(t, entry)
	assert.ErrorIs(t, err, coordinator_domain.ErrCacheMiss,
		"stale entries are reported as misses regardless of invalidate failure")
}

func TestIntrospectionCache_ClearSwallowsBackingError(t *testing.T) {
	t.Parallel()

	fake := &failingCache{invalidateAllErr: errInjectedCache}
	cache := &introspectionCache{cache: fake}

	err := cache.Clear(context.Background())

	assert.NoError(t, err, "Clear must not surface backing cache errors to callers")
}

func TestIntrospectionCache_CloseSwallowsBackingError(t *testing.T) {
	t.Parallel()

	fake := &failingCache{closeErr: errInjectedCache}
	cache := &introspectionCache{cache: fake}

	assert.NotPanics(t, func() { cache.Close() },
		"Close must absorb backing cache errors without panicking")
}

var _ cache_domain.Cache[string, *coordinator_domain.IntrospectionCacheEntry] = (*failingCache)(nil)
