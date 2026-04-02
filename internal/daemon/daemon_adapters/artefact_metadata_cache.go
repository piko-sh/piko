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

package daemon_adapters

import (
	"context"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

// artefactMetadataCache stores artefact metadata to reduce registry database
// lookups for assets that are accessed often. Backed by the cache hexagon with
// automatic TTL expiry and size-limited eviction.
type artefactMetadataCache struct {
	// cache stores artefact metadata entries, keyed by artefact ID.
	cache cache_domain.Cache[string, *registry_dto.ArtefactMeta]

	// registryService fetches artefact metadata from the registry.
	registryService registry_domain.RegistryService
}

// Get retrieves artefact metadata from the cache.
//
// Takes artefactID (string) which identifies the artefact to look up.
//
// Returns *registry_dto.ArtefactMeta which contains the cached metadata, or
// nil if not found.
// Returns bool which is true when the artefact was found in the cache.
func (c *artefactMetadataCache) Get(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, bool) {
	if c.cache == nil {
		return nil, false
	}
	_, l := logger_domain.From(ctx, log)
	artefact, ok, _ := c.cache.GetIfPresent(ctx, artefactID)
	if ok {
		l.Trace("Artefact metadata cache HIT",
			logger_domain.String("artefactID", artefactID))
	} else {
		l.Trace("Artefact metadata cache MISS",
			logger_domain.String("artefactID", artefactID))
	}
	return artefact, ok
}

// GetOrLoad retrieves artefact metadata from cache or loads it from registry.
//
// Takes artefactID (string) which identifies the artefact to retrieve.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
// Returns error when the artefact cannot be loaded from the registry.
func (c *artefactMetadataCache) GetOrLoad(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	if c.cache == nil {
		return c.registryService.GetArtefact(ctx, artefactID)
	}
	loader := cache_dto.LoaderFunc[string, *registry_dto.ArtefactMeta](
		func(ctx context.Context, key string) (*registry_dto.ArtefactMeta, error) {
			return c.registryService.GetArtefact(ctx, key)
		},
	)
	return c.cache.Get(ctx, artefactID, loader)
}

// Invalidate removes artefact metadata from cache.
// Called after variant generation to force refresh on next access.
//
// Takes artefactID (string) which identifies the artefact to remove.
func (c *artefactMetadataCache) Invalidate(ctx context.Context, artefactID string) {
	if c.cache != nil {
		_ = c.cache.Invalidate(ctx, artefactID)
	}
}

// Close releases all resources held by the cache.
func (c *artefactMetadataCache) Close(ctx context.Context) {
	if c.cache != nil {
		_ = c.cache.Close(ctx)
	}
}

// Stats returns cache statistics for monitoring.
//
// Returns cache_dto.Stats which contains the current cache metrics.
func (c *artefactMetadataCache) Stats() cache_dto.Stats {
	return c.cache.Stats()
}

// newArtefactMetadataCache creates a new metadata cache.
//
// Takes cache (cache_domain.Cache) which provides the underlying storage.
// Takes registryService (RegistryService) which provides access to the artefact
// registry for cache-miss loading.
//
// Returns *artefactMetadataCache which is ready for use.
func newArtefactMetadataCache(
	cache cache_domain.Cache[string, *registry_dto.ArtefactMeta],
	registryService registry_domain.RegistryService,
) *artefactMetadataCache {
	return &artefactMetadataCache{
		cache:           cache,
		registryService: registryService,
	}
}
