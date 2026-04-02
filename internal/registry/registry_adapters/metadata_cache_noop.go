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

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

var _ registry_domain.MetadataCache = (*NoOpMetadataCache)(nil)

// NoOpMetadataCache implements MetadataCache as a no-op cache that always
// returns cache misses. Use it in tests when caching is not needed.
type NoOpMetadataCache struct{}

// NewNoOpMetadataCache creates a new no-op metadata cache for testing.
//
// Returns *NoOpMetadataCache which does nothing when its methods are called.
func NewNoOpMetadataCache() *NoOpMetadataCache {
	return &NoOpMetadataCache{}
}

// Get always returns nil, indicating a cache miss.
//
// Returns *registry_dto.ArtefactMeta which is always nil for this no-op
// implementation.
// Returns error which is always nil.
func (*NoOpMetadataCache) Get(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
}

// GetMultiple returns all IDs as misses since nothing is cached.
//
// Takes artefactIDs ([]string) which specifies the artefact IDs to look up.
//
// Returns []*registry_dto.ArtefactMeta which is always nil for this no-op
// implementation.
// Returns []string which contains all requested IDs as cache misses.
func (*NoOpMetadataCache) GetMultiple(_ context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, []string) {
	return nil, artefactIDs
}

// Set does nothing as this is a no-operation cache.
func (*NoOpMetadataCache) Set(_ context.Context, _ *registry_dto.ArtefactMeta) {
}

// SetMultiple does nothing as this is a no-op implementation.
func (*NoOpMetadataCache) SetMultiple(_ context.Context, _ []*registry_dto.ArtefactMeta) {
}

// Delete does nothing as this is a no-op implementation.
func (*NoOpMetadataCache) Delete(_ context.Context, _ string) {
}

// Close does nothing and returns nil.
//
// Returns error which is always nil.
func (*NoOpMetadataCache) Close(_ context.Context) error {
	return nil
}
