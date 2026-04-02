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

package registry_adapters

import (
	"context"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/wdk/safeconv"
)

var _ registry_domain.MetadataCache = (*metadataCache)(nil)

// approxProfileConfigSize is the approximate memory size of a profile
// configuration in bytes.
const approxProfileConfigSize = 128

// metadataCache wraps the cache hexagon to provide artefact metadata caching
// for the Registry service. It implements registry_domain.MetadataCache.
//
// This adapter lets the Registry use the cache hexagon's features (multiple
// providers, transformations) while keeping the MetadataCache interface.
//
// The adapter uses Cache[string, *ArtefactMeta] for full compile-time type
// safety. The cache hexagon creates this typed instance using the factory
// blueprint pattern, which allows:
//
//  1. Full type safety with no runtime type assertions needed.
//  2. Resource sharing where one provider serves multiple subsystems via
//     namespaces.
//  3. No circular dependencies as cache_adapters imports registry_adapters.
//  4. Easy provider swapping for all subsystems at once.
//
// The factory blueprint is registered in cache_provider.go via init(). This
// lets the cache hexagon create Cache[string, *ArtefactMeta] without knowing
// about the registry domain at compile time.
type metadataCache struct {
	// cache stores artefact metadata indexed by artefact ID.
	cache cache_domain.Cache[string, *registry_dto.ArtefactMeta]
}

// Get retrieves an artefact from the cache by ID.
//
// Takes artefactID (string) which identifies the artefact to retrieve.
//
// Returns *registry_dto.ArtefactMeta which contains the cached artefact data.
// Returns error when the artefact is not found in the cache.
func (c *metadataCache) Get(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	artefact, ok, _ := c.cache.GetIfPresent(ctx, artefactID)
	if !ok {
		return nil, registry_domain.ErrCacheMiss
	}

	return artefact, nil
}

// GetMultiple fetches several artefacts from the cache at once.
//
// Takes artefactIDs ([]string) which lists the artefact IDs to look up.
//
// Returns hits ([]*registry_dto.ArtefactMeta) which contains the cached
// artefacts that were found.
// Returns misses ([]string) which contains the IDs not found in the cache.
func (c *metadataCache) GetMultiple(ctx context.Context, artefactIDs []string) (hits []*registry_dto.ArtefactMeta, misses []string) {
	hits = make([]*registry_dto.ArtefactMeta, 0, len(artefactIDs))
	misses = make([]string, 0, max(1, len(artefactIDs)/5))

	for _, id := range artefactIDs {
		artefact, ok, _ := c.cache.GetIfPresent(ctx, id)
		if ok {
			hits = append(hits, artefact)
		} else {
			misses = append(misses, id)
		}
	}

	return hits, misses
}

// Set stores an artefact in the cache.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the artefact to store.
func (c *metadataCache) Set(ctx context.Context, artefact *registry_dto.ArtefactMeta) {
	_ = c.cache.Set(ctx, artefact.ID, artefact)
}

// SetMultiple stores multiple artefacts in the cache using a bulk operation.
//
// Takes artefacts ([]*registry_dto.ArtefactMeta) which are the items to cache.
func (c *metadataCache) SetMultiple(ctx context.Context, artefacts []*registry_dto.ArtefactMeta) {
	ctx, l := logger_domain.From(ctx, log)
	items := make(map[string]*registry_dto.ArtefactMeta, len(artefacts))
	for _, art := range artefacts {
		items[art.ID] = art
	}

	if err := c.cache.BulkSet(ctx, items); err != nil {
		l.Warn("Failed to bulk set artefacts in cache",
			logger_domain.Int("artefact_count", len(artefacts)),
			logger_domain.Error(err))
		for _, art := range artefacts {
			c.Set(ctx, art)
		}
	}
}

// Delete removes an artefact from the cache.
//
// Takes artefactID (string) which identifies the artefact to remove.
func (c *metadataCache) Delete(ctx context.Context, artefactID string) {
	ctx, l := logger_domain.From(ctx, log)
	_ = c.cache.Invalidate(ctx, artefactID)
	l.Trace("Deleted artefact from cache", logger_domain.String("artefactID", artefactID))
}

// Close shuts down the cache and releases resources.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
//
// Returns error when the cache cannot be closed cleanly.
func (c *metadataCache) Close(ctx context.Context) error {
	err := c.cache.Close(ctx)
	_, l := logger_domain.From(ctx, log)
	l.Internal("Metadata cache closed.")
	return err
}

// NewMetadataCache creates a new metadata cache adapter that wraps the cache
// hexagon. The cache is provided via dependency injection, allowing full
// flexibility in configuration (provider choice, transformations, etc.).
//
// Accepts fully-typed Cache[string, *ArtefactMeta] for compile-time type
// safety. The typed cache is created using the factory blueprint pattern.
//
// Takes cache (Cache[string, *ArtefactMeta]) which provides the underlying
// typed cache storage.
//
// Returns registry_domain.MetadataCache which is the configured metadata
// cache ready for use.
func NewMetadataCache(
	cache cache_domain.Cache[string, *registry_dto.ArtefactMeta],
) registry_domain.MetadataCache {
	return &metadataCache{
		cache: cache,
	}
}

// ArtefactMetaWeigher calculates the approximate memory weight of an artefact
// metadata entry. This weigher is used for weight-based cache eviction to
// ensure the cache stays within its configured memory limits.
//
// Takes key (string) which is the cache key for the artefact entry.
// Takes art (*registry_dto.ArtefactMeta) which is the artefact metadata to
// weigh.
//
// Returns uint32 which is the estimated memory size in bytes.
func ArtefactMetaWeigher(key string, art *registry_dto.ArtefactMeta) uint32 {
	var weight uint32

	weight += safeStringWeight(key)
	weight += safeStringWeight(art.ID)
	weight += safeStringWeight(art.SourcePath)

	for i := range art.ActualVariants {
		v := &art.ActualVariants[i]
		weight += safeStringWeight(v.VariantID)
		weight += safeStringWeight(v.StorageBackendID)
		weight += safeStringWeight(v.StorageKey)
		weight += safeStringWeight(v.MimeType)
		for k, value := range v.MetadataTags.All() {
			weight += safeSumWeight(k, value)
		}
	}

	for i := range art.DesiredProfiles {
		weight += safeStringWeight(art.DesiredProfiles[i].Name)
		weight += safeStringWeight(art.DesiredProfiles[i].Profile.CapabilityName)
		weight += approxProfileConfigSize
	}

	return weight
}

// safeStringWeight returns the byte length of a string as uint32, capped at
// MaxUint32 to prevent overflow.
//
// Takes s (string) which is the string to measure.
//
// Returns uint32 which is the byte length, capped at MaxUint32.
func safeStringWeight(s string) uint32 {
	return safeconv.IntToUint32(len(s))
}

// safeSumWeight returns the total length of two strings as a uint32.
//
// Takes a (string) which is the first string to measure.
// Takes b (string) which is the second string to measure.
//
// Returns uint32 which is the combined length, capped at MaxUint32.
func safeSumWeight(a, b string) uint32 {
	sum := len(a) + len(b)
	return safeconv.IntToUint32(sum)
}
